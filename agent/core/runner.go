package core

import (
	"context"
	"fmt"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/memory"
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
	if a.Config.Timeout != "" {
		timeout, err := time.ParseDuration(a.Config.Timeout)
		if err != nil {
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

	// Call LLM
	req := &llm.ChatRequest{
		Messages: messages,
	}
	response, err := a.Model.Chat(ctx, req)
	if err != nil {
		return Message{}, fmt.Errorf("LLM call failed: %w", err)
	}

	result := Message{
		Role:    "assistant",
		Content: response.Content,
	}

	// Store response in memory
	if a.Mem != nil {
		if existing, err := a.Mem.Retrieve(ctx, "conversation"); err == nil {
			if msgs, ok := existing.([]Message); ok {
				msgs = append(msgs, result)
				if err := a.Mem.Store(ctx, "conversation", msgs); err != nil {
					return Message{}, fmt.Errorf("failed to store response: %w", err)
				}
			} else {
				if err := a.Mem.Store(ctx, "conversation", []Message{result}); err != nil {
					return Message{}, fmt.Errorf("failed to store response: %w", err)
				}
			}
		} else {
			if err := a.Mem.Store(ctx, "conversation", []Message{result}); err != nil {
				return Message{}, fmt.Errorf("failed to store response: %w", err)
			}
		}
	}

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
