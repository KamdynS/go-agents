package llm

import (
	"context"
	"time"
)

// Message represents a message in a conversation with an LLM
type Message struct {
	Role    string            `json:"role"`    // "system", "user", "assistant", "tool"
	Content string            `json:"content"` // Message content
	Name    string            `json:"name,omitempty"` // Optional name for the message
	ToolCallID string         `json:"tool_call_id,omitempty"` // For tool response messages
}

// Response represents the response from an LLM
type Response struct {
	Content      string            `json:"content"`
	Role         string            `json:"role,omitempty"`
	Model        string            `json:"model"`
	Provider     Provider          `json:"provider"`
	Usage        *Usage            `json:"usage,omitempty"`
	FinishReason string            `json:"finish_reason,omitempty"` // "stop", "length", "tool_calls", etc.
	ToolCalls    []ToolCall        `json:"tool_calls,omitempty"`
	Meta         map[string]string `json:"meta,omitempty"`
	Latency      time.Duration     `json:"latency,omitempty"`
	Timestamp    time.Time         `json:"timestamp,omitempty"`
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function Function `json:"function"`
}

// Function represents a function call
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Client defines the interface for interacting with Large Language Models
type Client interface {
	// Chat sends a conversation to the LLM and returns a response
	Chat(ctx context.Context, req *ChatRequest) (*Response, error)
	
	// Completion sends a single prompt to the LLM and returns a response
	Completion(ctx context.Context, prompt string) (*Response, error)
	
	// Stream enables streaming responses (if supported by the provider)
	Stream(ctx context.Context, req *ChatRequest, output chan<- *Response) error
	
	// Model returns the model identifier
	Model() string
	
	// Provider returns the provider name
	Provider() Provider
	
	// Validate checks if the client configuration is valid
	Validate() error
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages         []Message              `json:"messages"`
	Model            string                 `json:"model,omitempty"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Seed             *int                   `json:"seed,omitempty"` // For reproducible outputs
	User             string                 `json:"user,omitempty"` // User identifier for abuse monitoring
	Meta             map[string]interface{} `json:"meta,omitempty"` // Provider-specific options
}

// Tool represents a tool/function that the LLM can call
type Tool struct {
	Type     string   `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a function definition
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ResponseFormat specifies the format of the response
type ResponseFormat struct {
	Type       string                 `json:"type"` // "text" or "json_object"
	JSONSchema map[string]interface{} `json:"json_schema,omitempty"`
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"rate_limit_exceeded",
			"server_error",
			"timeout",
			"connection_error",
		},
	}
}

// Config holds common configuration options for LLM clients
type Config struct {
	APIKey       string        `json:"api_key"`
	Model        string        `json:"model"`
	BaseURL      string        `json:"base_url,omitempty"`
	Temperature  float64       `json:"temperature,omitempty"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	RetryConfig  RetryConfig   `json:"retry_config,omitempty"`
	Debug        bool          `json:"debug,omitempty"`
	UserAgent    string        `json:"user_agent,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}