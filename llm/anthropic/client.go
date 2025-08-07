package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/liushuangls/go-anthropic/v2"
)

// Client implements the llm.Client interface for Anthropic Claude
type Client struct {
	client  *anthropic.Client
	config  Config
	retrier *llm.Retrier
}

// Config holds Anthropic-specific configuration
type Config struct {
	APIKey      string          `json:"api_key"`
	Model       string          `json:"model"` // e.g., "claude-3-5-sonnet-20241022"
	BaseURL     string          `json:"base_url,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Timeout     time.Duration   `json:"timeout,omitempty"`
	RetryConfig llm.RetryConfig `json:"retry_config,omitempty"`
	Debug       bool            `json:"debug,omitempty"`
}

// NewClient creates a new Anthropic client
func NewClient(config Config) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set defaults
	if config.Model == "" {
		config.Model = llm.ModelClaude35Haiku // Default to fastest model
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 1000
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryConfig.MaxRetries == 0 {
		config.RetryConfig = llm.DefaultRetryConfig()
	}

	// Create Anthropic client
	opts := []anthropic.ClientOption{}

	if config.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(config.BaseURL))
	}

	if config.Timeout > 0 {
		opts = append(opts, anthropic.WithHTTPClient(&http.Client{
			Timeout: config.Timeout,
		}))
	}

	client := &Client{
		client:  anthropic.NewClient(config.APIKey, opts...),
		config:  config,
		retrier: llm.NewRetrier(config.RetryConfig),
	}

	return client, nil
}

// validateConfig validates the Anthropic configuration
func validateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.Model != "" {
		if err := llm.ValidateModel(config.Model); err != nil {
			return fmt.Errorf("invalid model: %w", err)
		}

		// Verify it's an Anthropic model
		model, _ := llm.GetModel(config.Model)
		if model.Provider != llm.ProviderAnthropic {
			return fmt.Errorf("model %s is not an Anthropic model", config.Model)
		}
	}

	if config.Temperature < 0 || config.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}

	if config.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative")
	}

	return nil
}

// Chat implements llm.Client interface
func (c *Client) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.Response, error) {
	start := time.Now()

	// Execute with retry logic
	result, err := llm.Execute(c.retrier, ctx, func(ctx context.Context, attempt int) (*llm.Response, error) {
		return c.chat(ctx, req, attempt)
	})

	if err != nil {
		return nil, err
	}

	// Set latency
	result.Latency = time.Since(start)
	result.Timestamp = start

	return result, nil
}

// chat performs the actual chat completion request
func (c *Client) chat(ctx context.Context, req *llm.ChatRequest, attempt int) (*llm.Response, error) {
	// Convert messages
	messages := make([]anthropic.Message, 0, len(req.Messages))

	var systemPrompt string
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
	}

	// Process messages - Anthropic separates system from messages
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			// Add to system prompt
			if systemPrompt != "" {
				systemPrompt += "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
		case "user":
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		case "assistant":
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleAssistant,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		case "tool":
			// For now, treat tool messages as user messages
			// TODO: Implement proper tool result handling
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		default:
			// Default to user
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		}
	}

	// Build request
	model := c.config.Model
	if req.Model != "" {
		model = req.Model
	}

	anthReq := anthropic.MessagesRequest{
		Model:     anthropic.Model(model),
		Messages:  messages,
		MaxTokens: c.config.MaxTokens,
	}

	// Set system prompt if provided
	if systemPrompt != "" {
		anthReq.System = systemPrompt
	}

	// Set optional parameters
	if req.Temperature != nil {
		t := float32(*req.Temperature)
		anthReq.Temperature = &t
	} else {
		temp := float32(c.config.Temperature)
		anthReq.Temperature = &temp
	}

	if req.MaxTokens != nil {
		anthReq.MaxTokens = *req.MaxTokens
	}

	if req.TopP != nil {
		p := float32(*req.TopP)
		anthReq.TopP = &p
	}

	if len(req.Stop) > 0 {
		anthReq.StopSequences = req.Stop
	}

	// Tools are not supported in this version - skip for now
	// TODO: Add tool support when available

	// Make the API call
	resp, err := c.client.CreateMessages(ctx, anthReq)
	if err != nil {
		return nil, c.convertError(err, attempt)
	}

	if len(resp.Content) == 0 {
		return nil, llm.NewLLMError(llm.ProviderAnthropic, llm.ErrorTypeUnknown, "no content returned")
	}

	// Extract content (Anthropic returns array of content blocks)
	var content strings.Builder
	var toolCalls []llm.ToolCall

	for _, block := range resp.Content {
		if block.Type == "text" && block.Text != nil {
			content.WriteString(*block.Text)
		}
		// Note: tool_use handling disabled for now due to API compatibility
		// TODO: Re-enable when anthropic client supports it
	}

	// Build usage info
	var usage *llm.Usage
	if resp.Usage.OutputTokens > 0 {
		modelInfo, _ := llm.GetModel(model)
		usage = &llm.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
			Cost:         modelInfo.EstimateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens),
		}
	}

	return &llm.Response{
		Content:      content.String(),
		Role:         "assistant",
		Model:        model,
		Provider:     llm.ProviderAnthropic,
		Usage:        usage,
		FinishReason: string(resp.StopReason),
		ToolCalls:    toolCalls,
		Meta: map[string]string{
			"id":   resp.ID,
			"type": string(resp.Type),
			"role": string(resp.Role),
		},
	}, nil
}

// Completion implements llm.Client interface
func (c *Client) Completion(ctx context.Context, prompt string) (*llm.Response, error) {
	req := &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	}
	return c.Chat(ctx, req)
}

// Stream implements llm.Client interface
func (c *Client) Stream(ctx context.Context, req *llm.ChatRequest, output chan<- *llm.Response) error {
	defer close(output)

	// Execute with retry logic
	_, err := llm.Execute(c.retrier, ctx, func(ctx context.Context, attempt int) (struct{}, error) {
		return struct{}{}, c.stream(ctx, req, output, attempt)
	})

	return err
}

// stream performs the actual streaming request
func (c *Client) stream(ctx context.Context, req *llm.ChatRequest, output chan<- *llm.Response, attempt int) error {
	// Convert messages (similar to chat method)
	messages := make([]anthropic.Message, 0, len(req.Messages))

	var systemPrompt string
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
	}

	// Process messages
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if systemPrompt != "" {
				systemPrompt += "\n\n" + msg.Content
			} else {
				systemPrompt = msg.Content
			}
		case "user":
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		case "assistant":
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleAssistant,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		default:
			messages = append(messages, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{{Type: "text", Text: &msg.Content}},
			})
		}
	}

	// Build streaming request with callbacks
	model := c.config.Model
	if req.Model != "" {
		model = req.Model
	}

	anthReq := anthropic.MessagesStreamRequest{
		MessagesRequest: anthropic.MessagesRequest{
			Model:     anthropic.Model(model),
			Messages:  messages,
			MaxTokens: c.config.MaxTokens,
		},
		OnContentBlockDelta: func(data anthropic.MessagesEventContentBlockDeltaData) {
			var text string
			if data.Delta.Text != nil && *data.Delta.Text != "" {
				text = *data.Delta.Text
			}
			if text == "" {
				return
			}
			start := time.Now()
			llmResp := &llm.Response{
				Content:   text,
				Role:      "assistant",
				Model:     model,
				Provider:  llm.ProviderAnthropic,
				Latency:   time.Since(start),
				Timestamp: start,
				Meta: map[string]string{
					"streaming": "true",
					"event":     "content_block_delta",
				},
			}
			select {
			case output <- llmResp:
			case <-ctx.Done():
			}
		},
	}

	// Set system prompt
	if systemPrompt != "" {
		anthReq.System = systemPrompt
	}

	// Set parameters
	if req.Temperature != nil {
		t := float32(*req.Temperature)
		anthReq.Temperature = &t
	} else {
		temp := float32(c.config.Temperature)
		anthReq.Temperature = &temp
	}

	if req.MaxTokens != nil {
		anthReq.MaxTokens = *req.MaxTokens
	}

	if _, err := c.client.CreateMessagesStream(ctx, anthReq); err != nil {
		return c.convertError(err, attempt)
	}
	return nil
}

// convertError converts Anthropic SDK errors to LLM errors
func (c *Client) convertError(err error, attempt int) error {
	if err == nil {
		return nil
	}

	// Try to extract Anthropic API error
	if apiErr, ok := err.(*anthropic.APIError); ok {
		// The SDK doesn't expose HTTP status consistently; map to provider error with message
		llmErr := llm.NewLLMErrorWithCause(llm.ProviderAnthropic, llm.ErrorTypeUnknown, apiErr.Message, err)
		llmErr.Code = string(apiErr.Type)
		return llmErr
	}

	// Handle context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return llm.NewLLMErrorWithCause(llm.ProviderAnthropic, llm.ErrorTypeTimeout, "request timeout", err)
	}
	if errors.Is(err, context.Canceled) {
		return llm.NewLLMErrorWithCause(llm.ProviderAnthropic, llm.ErrorTypeUnknown, "context error", err)
	}

	// Handle network errors
	if strings.Contains(strings.ToLower(err.Error()), "connection") ||
		strings.Contains(strings.ToLower(err.Error()), "network") {
		return llm.NewLLMErrorWithCause(llm.ProviderAnthropic, llm.ErrorTypeConnectionError, "connection error", err)
	}

	// Default to unknown error
	return llm.NewLLMErrorWithCause(llm.ProviderAnthropic, llm.ErrorTypeUnknown, err.Error(), err)
}

// Model implements llm.Client interface
func (c *Client) Model() string {
	return c.config.Model
}

// Provider implements llm.Client interface
func (c *Client) Provider() llm.Provider {
	return llm.ProviderAnthropic
}

// Validate implements llm.Client interface
func (c *Client) Validate() error {
	return validateConfig(c.config)
}

// StructuredChat performs chat completion with structured output
func StructuredChat[T llm.Structured](c *Client, ctx context.Context, req llm.StructuredRequest[T]) (*llm.StructuredResponse[T], error) {
	// Add JSON instruction to system prompt
	systemPrompt := req.SystemPrompt
	if systemPrompt != "" {
		systemPrompt += "\n\n"
	}
	systemPrompt += "You must respond ONLY with a JSON object matching the specified schema. Do not include any other text outside the JSON."

	// Build chat request
	chatReq := &llm.ChatRequest{
		Messages:     req.Messages,
		SystemPrompt: systemPrompt,
		Model:        req.Model,
		Temperature:  &req.Temperature,
		MaxTokens:    &req.MaxTokens,
	}

	// Add JSON schema instruction to the last user message
	if len(chatReq.Messages) > 0 {
		lastMsg := &chatReq.Messages[len(chatReq.Messages)-1]
		if lastMsg.Role == "user" {
			if schemaBytes, err := json.MarshalIndent(req.Schema, "", "  "); err == nil {
				lastMsg.Content += fmt.Sprintf("\n\nRespond with valid JSON matching this schema:\n```json\n%s\n```", string(schemaBytes))
			} else {
				lastMsg.Content += "\n\nRespond with a valid JSON object that includes all required fields."
			}
		}
	}

	// Execute chat completion
	resp, err := c.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	// Parse structured output
	structuredResp, err := llm.ParseStructured(resp.Content, req.OutputType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse structured output: %w", err)
	}

	// Set raw response and usage
	structuredResp.RawResponse = resp
	structuredResp.Usage = resp.Usage

	return structuredResp, nil
}

// StructuredCompletion performs completion with structured output
func StructuredCompletion[T llm.Structured](c *Client, ctx context.Context, prompt string, outputType T) (*llm.StructuredResponse[T], error) {
	req := llm.StructuredRequest[T]{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       c.config.Model,
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
		OutputType:  outputType,
		Schema:      outputType.JSONSchema(),
	}

	return StructuredChat(c, ctx, req)
}
