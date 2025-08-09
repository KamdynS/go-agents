package core

import (
	"context"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/tools"
)

// Message represents a conversation message with role and content
type Message struct {
	Role    string            `json:"role"`
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Agent defines the core interface for AI agents
type Agent interface {
	// Run executes one reasoning-action loop with the given input and returns output
	Run(ctx context.Context, input Message) (Message, error)

	// RunStream executes the agent loop and streams responses via the provided channel
	RunStream(ctx context.Context, input Message, output chan<- Message) error
}

// AgentConfig holds configuration for creating agents
type AgentConfig struct {
	MaxIterations int
	Timeout       string
	SystemPrompt  string
	// Optional: model override for a request (used with router clients)
	ModelOverride string
}

// Middleware allows hooks around key lifecycle events
type Middleware interface {
	BeforeLLMCall(ctx context.Context, req *llm.ChatRequest) error
	AfterLLMResponse(ctx context.Context, resp *llm.Response) error
	BeforeToolExecute(ctx context.Context, toolName string, input string) error
	AfterToolExecute(ctx context.Context, toolName string, result string, execErr error) error
	AfterRun(ctx context.Context, final Message) error
}

// MemoryProcessor can transform/prune conversation history before sending to LLM
type MemoryProcessor interface {
	Process(ctx context.Context, history []Message) []Message
}

// ConfigResolver can adjust configuration and tools at runtime based on input
type ConfigResolver interface {
	Resolve(ctx context.Context, input Message, base AgentConfig) (AgentConfig, tools.Registry)
}

// Built-in processors
// TokenLimiter removes oldest messages until roughly under token budget (simple char-based heuristic)
type TokenLimiter struct{ MaxChars int }

func (t TokenLimiter) Process(ctx context.Context, history []Message) []Message {
	if t.MaxChars <= 0 {
		return history
	}
	total := 0
	for i := len(history) - 1; i >= 0; i-- { // accumulate from newest backwards
		total += len(history[i].Content)
	}
	if total <= t.MaxChars {
		return history
	}
	// Trim from oldest
	out := make([]Message, 0, len(history))
	cur := 0
	for i := len(history) - 1; i >= 0; i-- { // iterate newest to oldest building list
		if cur+len(history[i].Content) > t.MaxChars {
			continue
		}
		out = append(out, history[i])
		cur += len(history[i].Content)
	}
	// reverse back to chronological
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ToolCallFilter removes any prior tool messages from history to save tokens
type ToolCallFilter struct{}

func (ToolCallFilter) Process(ctx context.Context, history []Message) []Message {
	out := make([]Message, 0, len(history))
	for _, m := range history {
		if m.Role == "tool" {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ToolCall represents a requested tool execution parsed from an LLM response
type ToolCall struct {
	Name      string
	Arguments string // JSON string per llm.Function.Arguments
}
