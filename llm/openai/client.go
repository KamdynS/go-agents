package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/sashabaranov/go-openai"
)

// Client implements the llm.Client interface for OpenAI
type Client struct {
	client  *openai.Client
	config  Config
	retrier *llm.Retrier
}

// Config holds OpenAI-specific configuration
type Config struct {
	APIKey       string          `json:"api_key"`
	Model        string          `json:"model"` // e.g., "gpt-4o", "gpt-3.5-turbo"
	BaseURL      string          `json:"base_url,omitempty"`
	Temperature  float64         `json:"temperature,omitempty"`
	MaxTokens    int             `json:"max_tokens,omitempty"`
	Timeout      time.Duration   `json:"timeout,omitempty"`
	RetryConfig  llm.RetryConfig `json:"retry_config,omitempty"`
	Debug        bool            `json:"debug,omitempty"`
	Organization string          `json:"organization,omitempty"`
}

// NewClient creates a new OpenAI client
func NewClient(config Config) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set defaults
	if config.Model == "" {
		config.Model = llm.ModelGPT4oMini // Default to cheapest GPT-4 model
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

	// Create OpenAI client configuration
	openaiConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	}
	if config.Organization != "" {
		openaiConfig.OrgID = config.Organization
	}

	// Configure HTTP client with timeout
	openaiConfig.HTTPClient = &http.Client{
		Timeout: config.Timeout,
	}

	client := &Client{
		client:  openai.NewClientWithConfig(openaiConfig),
		config:  config,
		retrier: llm.NewRetrier(config.RetryConfig),
	}

	return client, nil
}

// validateConfig validates the OpenAI configuration
func validateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.Model != "" {
		if err := llm.ValidateModel(config.Model); err != nil {
			return fmt.Errorf("invalid model: %w", err)
		}

		// Verify it's an OpenAI model
		model, _ := llm.GetModel(config.Model)
		if model.Provider != llm.ProviderOpenAI {
			return fmt.Errorf("model %s is not an OpenAI model", config.Model)
		}
	}

	if config.Temperature < 0 || config.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
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
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Convert request messages
	for _, msg := range req.Messages {
		oaiMsg := openai.ChatCompletionMessage{
			Content: msg.Content,
		}

		switch msg.Role {
		case "system":
			oaiMsg.Role = openai.ChatMessageRoleSystem
		case "user":
			oaiMsg.Role = openai.ChatMessageRoleUser
		case "assistant":
			oaiMsg.Role = openai.ChatMessageRoleAssistant
		case "tool":
			oaiMsg.Role = openai.ChatMessageRoleTool
			if msg.ToolCallID != "" {
				oaiMsg.ToolCallID = msg.ToolCallID
			}
		default:
			oaiMsg.Role = openai.ChatMessageRoleUser
		}

		if msg.Name != "" {
			oaiMsg.Name = msg.Name
		}

		messages = append(messages, oaiMsg)
	}

	// Build request
	model := c.config.Model
	if req.Model != "" {
		model = req.Model
	}

	oaiReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	// Set optional parameters
	if req.Temperature != nil {
		oaiReq.Temperature = float32(*req.Temperature)
	} else {
		oaiReq.Temperature = float32(c.config.Temperature)
	}

	if req.MaxTokens != nil {
		oaiReq.MaxTokens = *req.MaxTokens
	} else if c.config.MaxTokens > 0 {
		oaiReq.MaxTokens = c.config.MaxTokens
	}

	if req.TopP != nil {
		oaiReq.TopP = float32(*req.TopP)
	}

	if req.FrequencyPenalty != nil {
		oaiReq.FrequencyPenalty = float32(*req.FrequencyPenalty)
	}

	if req.PresencePenalty != nil {
		oaiReq.PresencePenalty = float32(*req.PresencePenalty)
	}

	if len(req.Stop) > 0 {
		oaiReq.Stop = req.Stop
	}

	if req.Seed != nil {
		oaiReq.Seed = req.Seed
	}

	if req.User != "" {
		oaiReq.User = req.User
	}

	// Handle tools/functions
	if len(req.Tools) > 0 {
		tools := make([]openai.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
		oaiReq.Tools = tools

		if req.ToolChoice != nil {
			oaiReq.ToolChoice = req.ToolChoice
		}
	}

	// Handle response format
	if req.ResponseFormat != nil {
		if req.ResponseFormat.Type == "json_object" {
			oaiReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			}
		}
	}

	// Make the API call
	resp, err := c.client.CreateChatCompletion(ctx, oaiReq)
	if err != nil {
		return nil, c.convertError(err, attempt)
	}

	if len(resp.Choices) == 0 {
		return nil, llm.NewLLMError(llm.ProviderOpenAI, llm.ErrorTypeUnknown, "no choices returned")
	}

	choice := resp.Choices[0]

	// Convert tool calls if present
	var toolCalls []llm.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls = make([]llm.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			toolCalls[i] = llm.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: llm.Function{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	// Build usage info
	var usage *llm.Usage
	if resp.Usage.TotalTokens > 0 {
		modelInfo, _ := llm.GetModel(model)
		usage = &llm.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
			Cost:         modelInfo.EstimateCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens),
		}
	}

	return &llm.Response{
		Content:      choice.Message.Content,
		Role:         "assistant",
		Model:        model,
		Provider:     llm.ProviderOpenAI,
		Usage:        usage,
		FinishReason: string(choice.FinishReason),
		ToolCalls:    toolCalls,
		Meta: map[string]string{
			"id":      resp.ID,
			"object":  resp.Object,
			"created": fmt.Sprintf("%d", resp.Created),
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
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Convert request messages
	for _, msg := range req.Messages {
		oaiMsg := openai.ChatCompletionMessage{
			Content: msg.Content,
		}

		switch msg.Role {
		case "system":
			oaiMsg.Role = openai.ChatMessageRoleSystem
		case "user":
			oaiMsg.Role = openai.ChatMessageRoleUser
		case "assistant":
			oaiMsg.Role = openai.ChatMessageRoleAssistant
		case "tool":
			oaiMsg.Role = openai.ChatMessageRoleTool
		default:
			oaiMsg.Role = openai.ChatMessageRoleUser
		}

		messages = append(messages, oaiMsg)
	}

	// Build streaming request
	model := c.config.Model
	if req.Model != "" {
		model = req.Model
	}

	oaiReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	// Set parameters (similar to chat method)
	if req.Temperature != nil {
		oaiReq.Temperature = float32(*req.Temperature)
	} else {
		oaiReq.Temperature = float32(c.config.Temperature)
	}

	if req.MaxTokens != nil {
		oaiReq.MaxTokens = *req.MaxTokens
	} else if c.config.MaxTokens > 0 {
		oaiReq.MaxTokens = c.config.MaxTokens
	}

	// Create streaming request
	stream, err := c.client.CreateChatCompletionStream(ctx, oaiReq)
	if err != nil {
		return c.convertError(err, attempt)
	}
	defer stream.Close()

	// Stream responses
	start := time.Now()
	for {
		response, err := stream.Recv()
		if err != nil {
			if strings.Contains(err.Error(), "stream finished") {
				break
			}
			return c.convertError(err, attempt)
		}

		if len(response.Choices) > 0 {
			choice := response.Choices[0]

			llmResp := &llm.Response{
				Content:      choice.Delta.Content,
				Role:         "assistant",
				Model:        model,
				Provider:     llm.ProviderOpenAI,
				FinishReason: string(choice.FinishReason),
				Latency:      time.Since(start),
				Timestamp:    start,
				Meta: map[string]string{
					"id":        response.ID,
					"created":   fmt.Sprintf("%d", response.Created),
					"streaming": "true",
				},
			}

			select {
			case output <- llmResp:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// convertError converts OpenAI SDK errors to LLM errors
func (c *Client) convertError(err error, attempt int) error {
	if err == nil {
		return nil
	}

	// Try to extract OpenAI API error
	if apiErr, ok := err.(*openai.APIError); ok {
		llmErr := llm.ParseHTTPError(llm.ProviderOpenAI, apiErr.HTTPStatusCode, apiErr.Message)
		if code, ok := apiErr.Code.(string); ok {
			llmErr.Code = code
		}

		// Add retry-after from headers if available
		if apiErr.HTTPStatusCode == 429 && len(apiErr.Message) > 0 {
			// Try to parse retry-after from error message
			if strings.Contains(strings.ToLower(apiErr.Message), "try again in") {
				// OpenAI sometimes includes retry time in error message
				llmErr.RetryAfter = 60 // Default to 60 seconds
			}
		}

		return llmErr
	}

	// Handle context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		if err == context.DeadlineExceeded {
			return llm.NewLLMErrorWithCause(llm.ProviderOpenAI, llm.ErrorTypeTimeout, "request timeout", err)
		}
		return llm.NewLLMErrorWithCause(llm.ProviderOpenAI, llm.ErrorTypeUnknown, "context error", err)
	}

	// Handle network errors
	if strings.Contains(strings.ToLower(err.Error()), "connection") ||
		strings.Contains(strings.ToLower(err.Error()), "network") {
		return llm.NewLLMErrorWithCause(llm.ProviderOpenAI, llm.ErrorTypeConnectionError, "connection error", err)
	}

	// Default to unknown error
	return llm.NewLLMErrorWithCause(llm.ProviderOpenAI, llm.ErrorTypeUnknown, err.Error(), err)
}

// Model implements llm.Client interface
func (c *Client) Model() string {
	return c.config.Model
}

// Provider implements llm.Client interface
func (c *Client) Provider() llm.Provider {
	return llm.ProviderOpenAI
}

// Validate implements llm.Client interface
func (c *Client) Validate() error {
	return validateConfig(c.config)
}

// StructuredChat performs chat completion with structured output
func StructuredChat[T llm.Structured](c *Client, ctx context.Context, req llm.StructuredRequest[T]) (*llm.StructuredResponse[T], error) {
	// Build chat request with JSON schema and strong instruction
	chatReq := &llm.ChatRequest{
		Messages:     req.Messages,
		SystemPrompt: req.SystemPrompt + "\n\nYou must respond ONLY with a JSON object matching the provided schema. Do not add explanations.",
		Model:        req.Model,
		Temperature:  &req.Temperature,
		MaxTokens:    &req.MaxTokens,
		ResponseFormat: &llm.ResponseFormat{
			Type:       "json_object",
			JSONSchema: req.Schema,
		},
	}

	// Add instruction to respond in JSON format
	if len(chatReq.Messages) > 0 {
		lastMsg := &chatReq.Messages[len(chatReq.Messages)-1]
		if lastMsg.Role == "user" {
			if schemaBytes, err := json.MarshalIndent(req.Schema, "", "  "); err == nil {
				lastMsg.Content += fmt.Sprintf("\n\nPlease respond with a valid JSON object matching this schema:\n```json\n%s\n```", string(schemaBytes))
			} else {
				lastMsg.Content += "\n\nPlease respond with a valid JSON object that includes all required fields."
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
