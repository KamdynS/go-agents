package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/memory"
	obs "github.com/KamdynS/go-agents/observability"
	"github.com/KamdynS/go-agents/tools"
)

// ChatAgent is the default implementation of the Agent interface
type ChatAgent struct {
	Model  llm.Client
	Tools  tools.Registry
	Mem    memory.Store
	Config AgentConfig
}

// NewChatAgent creates a new ChatAgent with the given configuration
func NewChatAgent(config ChatConfig) *ChatAgent {
	return &ChatAgent{
		Model:  config.Model,
		Tools:  config.Tools,
		Mem:    config.Mem,
		Config: config.Config,
	}
}

// ChatConfig holds configuration for ChatAgent
type ChatConfig struct {
	Model  llm.Client
	Tools  tools.Registry
	Mem    memory.Store
	Config AgentConfig
}

// Run implements the Agent interface
func (a *ChatAgent) Run(ctx context.Context, input Message) (Message, error) {
	// Agent-level span
	span, ctx := obs.TracerImpl.StartSpan(ctx, "agent.run")
	defer span.End()

	if a.Config.Timeout != "" {
		timeout, err := time.ParseDuration(a.Config.Timeout)
		if err != nil {
			span.SetStatus(obs.StatusCodeError, err.Error())
			return Message{}, fmt.Errorf("invalid timeout duration: %w", err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Store input message in memory
	if a.Mem != nil {
		if existing, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := existing.([]Message); ok {
				msgs = append(msgs, input)
				if err := a.Mem.Store(ctx, "conversation", msgs); err != nil {
					return Message{}, fmt.Errorf("failed to store message: %w", err)
				}
			} else {
				// initialize as slice with prior value if it was a single message
				_ = a.Mem.Store(ctx, "conversation", []Message{input})
			}
		} else {
			if err := a.Mem.Store(ctx, "conversation", []Message{input}); err != nil {
				return Message{}, fmt.Errorf("failed to store message: %w", err)
			}
		}
	}

	// Get conversation history
	var history []Message
	if a.Mem != nil {
		if h, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := h.([]Message); ok {
				history = msgs
			} else if msg, ok := h.(Message); ok { // legacy single message
				history = []Message{msg}
			}
		}
	}

	// Prepare messages for LLM
	messages := []llm.Message{{
		Role:    "system",
		Content: a.Config.SystemPrompt,
	}}

	for _, msg := range history {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Build tool definitions from registry (if any)
	var toolDefs []llm.Tool
	if a.Tools != nil {
		for _, name := range a.Tools.List() {
			if t, ok := a.Tools.Get(name); ok {
				toolDefs = append(toolDefs, llm.Tool{
					Type: "function",
					Function: llm.ToolFunction{
						Name:        t.Name(),
						Description: t.Description(),
						Parameters:  t.Schema(),
					},
				})
			}
		}
	}

	// ReAct-lite loop
	maxIterations := a.Config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 1
	}

	var finalResp *llm.Response
	for iter := 0; iter < maxIterations; iter++ {
		req := &llm.ChatRequest{
			Messages:     messages,
			Tools:        toolDefs,
			ToolChoice:   nil, // allow provider to auto-select
			SystemPrompt: "",  // already injected as first message
		}

		response, err := a.Model.Chat(ctx, req)
		if err != nil {
			span.SetStatus(obs.StatusCodeError, err.Error())
			return Message{}, fmt.Errorf("LLM call failed: %w", err)
		}
		finalResp = response

		// If tool calls are requested, execute them and continue loop
		if len(response.ToolCalls) > 0 && a.Tools != nil {
			// Append assistant message that triggered tool call to conversation
			messages = append(messages, llm.Message{Role: "assistant", Content: response.Content})

			for _, tc := range response.ToolCalls {
				// Resolve tool
				toolName := tc.Function.Name
				tool, ok := a.Tools.Get(toolName)
				if !ok {
					span.AddEvent("tool.not_found", map[string]interface{}{"tool": toolName})
					continue
				}

				// Parse arguments; support {"input":"..."} or raw string
				inputStr := tc.Function.Arguments
				var argObj map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &argObj); err == nil {
					if v, ok := argObj["input"].(string); ok {
						inputStr = v
					}
				}

				// Execute tool via registry (already instrumented)
				result, err := a.Tools.Execute(ctx, tool.Name(), inputStr)
				if err != nil {
					// Provide error back to model as tool content
					result = fmt.Sprintf("error: %v", err)
				}

				// Append tool result message
				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			// Continue to next iteration for model to observe tool outputs
			continue
		}

		// No tool calls, take this as final answer
		break
	}

	// Fallback if finalResp is nil (should not happen)
	if finalResp == nil {
		span.SetStatus(obs.StatusCodeError, "no response")
		return Message{}, fmt.Errorf("no response from model")
	}

	result := Message{
		Role:    "assistant",
		Content: finalResp.Content,
	}

	// Store response in memory
	if a.Mem != nil {
		if existing, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := existing.([]Message); ok {
				msgs = append(msgs, result)
				if err := a.Mem.Store(ctx, "conversation", msgs); err != nil {
					span.SetStatus(obs.StatusCodeError, err.Error())
					return Message{}, fmt.Errorf("failed to store response: %w", err)
				}
			} else {
				if err := a.Mem.Store(ctx, "conversation", []Message{result}); err != nil {
					span.SetStatus(obs.StatusCodeError, err.Error())
					return Message{}, fmt.Errorf("failed to store response: %w", err)
				}
			}
		} else {
			if err := a.Mem.Store(ctx, "conversation", []Message{result}); err != nil {
				span.SetStatus(obs.StatusCodeError, err.Error())
				return Message{}, fmt.Errorf("failed to store response: %w", err)
			}
		}
	}

	span.SetStatus(obs.StatusCodeOk, "")
	return result, nil
}

// RunStream implements the Agent interface for streaming responses
func (a *ChatAgent) RunStream(ctx context.Context, input Message, output chan<- Message) error {
	defer close(output)

	// For now, just run normally and send the result
	// TODO: Implement actual streaming when LLM clients support it
	result, err := a.Run(ctx, input)
	if err != nil {
		return err
	}

	select {
	case output <- result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
