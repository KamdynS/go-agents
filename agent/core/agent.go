package core

import (
	"context"
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
}

// ToolCall represents a requested tool execution parsed from an LLM response
type ToolCall struct {
	Name      string
	Arguments string // JSON string per llm.Function.Arguments
}
