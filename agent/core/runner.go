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
	// Optional processing and middleware hooks
	processors []MemoryProcessor
	mw         []Middleware
	resolver   ConfigResolver
}

// NewChatAgent creates a new ChatAgent with the given configuration
func NewChatAgent(config ChatConfig) *ChatAgent {
	return &ChatAgent{
		Model:      config.Model,
		Tools:      config.Tools,
		Mem:        config.Mem,
		Config:     config.Config,
		processors: config.Processors,
		mw:         config.Middleware,
		resolver:   config.Resolver,
	}
}

// ChatConfig holds configuration for ChatAgent
type ChatConfig struct {
	Model  llm.Client
	Tools  tools.Registry
	Mem    memory.Store
	Config AgentConfig
	// Optional
	Processors []MemoryProcessor
	Middleware []Middleware
	Resolver   ConfigResolver
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
	messages := []llm.Message{{Role: "system", Content: a.Config.SystemPrompt}}
	for _, msg := range history {
		messages = append(messages, llm.Message{Role: msg.Role, Content: msg.Content})
	}
	// Always include current input
	messages = append(messages, llm.Message{Role: input.Role, Content: input.Content})
	// Apply optional memory processors
	if len(a.processors) > 0 {
		pruned := a.applyProcessors(ctx, history)
		messages = []llm.Message{{Role: "system", Content: a.Config.SystemPrompt}}
		for _, m := range pruned {
			messages = append(messages, llm.Message{Role: m.Role, Content: m.Content})
		}
		messages = append(messages, llm.Message{Role: input.Role, Content: input.Content})
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

		// Middleware: before LLM
		for _, m := range a.mw {
			if err := m.BeforeLLMCall(ctx, req); err != nil {
				span.SetStatus(obs.StatusCodeError, err.Error())
				return Message{}, err
			}
		}

		response, err := a.Model.Chat(ctx, req)
		if err != nil {
			span.SetStatus(obs.StatusCodeError, err.Error())
			return Message{}, fmt.Errorf("LLM call failed: %w", err)
		}
		finalResp = response

		// Middleware: after LLM
		for _, m := range a.mw {
			if err := m.AfterLLMResponse(ctx, response); err != nil {
				span.SetStatus(obs.StatusCodeError, err.Error())
				return Message{}, err
			}
		}

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

				// Middleware: before tool
				for _, m := range a.mw {
					if err := m.BeforeToolExecute(ctx, toolName, inputStr); err != nil {
						span.SetStatus(obs.StatusCodeError, err.Error())
						return Message{}, err
					}
				}

				// Execute tool via registry (already instrumented)
				result, err := a.Tools.Execute(ctx, tool.Name(), inputStr)
				if err != nil {
					// Provide error back to model as tool content
					result = fmt.Sprintf("error: %v", err)
				}

				// Middleware: after tool
				for _, m := range a.mw {
					_ = m.AfterToolExecute(ctx, toolName, result, err)
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

	// Middleware: after run
	for _, m := range a.mw {
		_ = m.AfterRun(ctx, result)
	}

	span.SetStatus(obs.StatusCodeOk, "")
	return result, nil
}

// RunStream implements the Agent interface for streaming responses
func (a *ChatAgent) RunStream(ctx context.Context, input Message, output chan<- Message) error {
	defer close(output)

	span, ctx := obs.TracerImpl.StartSpan(ctx, "agent.run_stream")
	defer span.End()

	// Store incoming message
	if a.Mem != nil {
		if existing, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := existing.([]Message); ok {
				msgs = append(msgs, input)
				_ = a.Mem.Store(ctx, "conversation", msgs)
			}
		} else {
			_ = a.Mem.Store(ctx, "conversation", []Message{input})
		}
	}

	// Build history
	var history []Message
	if a.Mem != nil {
		if h, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := h.([]Message); ok {
				history = msgs
			} else if msg, ok := h.(Message); ok {
				history = []Message{msg}
			}
		}
	}

	// Prepare LLM request
	messages := []llm.Message{{Role: "system", Content: a.Config.SystemPrompt}}
	for _, msg := range history {
		messages = append(messages, llm.Message{Role: msg.Role, Content: msg.Content})
	}
	messages = append(messages, llm.Message{Role: input.Role, Content: input.Content})
	if len(a.processors) > 0 {
		pruned := a.applyProcessors(ctx, history)
		messages = []llm.Message{{Role: "system", Content: a.Config.SystemPrompt}}
		for _, m := range pruned {
			messages = append(messages, llm.Message{Role: m.Role, Content: m.Content})
		}
		messages = append(messages, llm.Message{Role: input.Role, Content: input.Content})
	}

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

	req := &llm.ChatRequest{Messages: messages, Tools: toolDefs}
	for _, m := range a.mw {
		if err := m.BeforeLLMCall(ctx, req); err != nil {
			span.SetStatus(obs.StatusCodeError, err.Error())
			return err
		}
	}

	// Stream from LLM and forward chunks
	inner := make(chan *llm.Response)
	errCh := make(chan error, 1)
	go func() {
		// Do not close the inner channel here. Some providers close it; others may not on error.
		errCh <- a.Model.Stream(ctx, req, inner)
	}()

	var buffer string
	for {
		select {
		case resp, ok := <-inner:
			if !ok {
				// Streaming done, emit final message and persist
				if buffer != "" {
					final := Message{Role: "assistant", Content: buffer}
					if a.Mem != nil {
						if existing, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
							if msgs, ok := existing.([]Message); ok {
								msgs = append(msgs, final)
								_ = a.Mem.Store(ctx, "conversation", msgs)
							}
						} else {
							_ = a.Mem.Store(ctx, "conversation", []Message{final})
						}
					}
					select {
					case output <- final:
					default:
					}
				}
				span.SetStatus(obs.StatusCodeOk, "")
				return nil
			}
			if resp == nil {
				continue
			}
			// Forward incremental content
			if resp.Content != "" {
				buffer += resp.Content
				select {
				case output <- Message{Role: "assistant", Content: resp.Content, Meta: map[string]string{"streaming": "true"}}:
				default:
				}
			}
			for _, m := range a.mw {
				_ = m.AfterLLMResponse(ctx, resp)
			}
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// applyProcessors applies configured memory processors to history
func (a *ChatAgent) applyProcessors(ctx context.Context, history []Message) []Message {
	out := history
	for _, p := range a.processors {
		out = p.Process(ctx, out)
	}
	return out
}
