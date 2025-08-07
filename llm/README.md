# LLM Package

The `llm` package provides provider-agnostic interfaces for interacting with Large Language Models.

## Overview

This package defines the core `Client` interface that all LLM providers implement, enabling seamless switching between different models and providers without changing your agent code.

## Interfaces

### Client Interface

```go
type Client interface {
    Chat(ctx context.Context, messages []Message) (*Response, error)
    Completion(ctx context.Context, prompt string) (*Response, error)
    Stream(ctx context.Context, messages []Message, output chan<- *Response) error
    Model() string
}
```

### Message and Response Types

```go
type Message struct {
    Role    string `json:"role"`    // "system", "user", "assistant"
    Content string `json:"content"` // Message content
}

type Response struct {
    Content   string            `json:"content"`
    TokenUsed int               `json:"tokens_used,omitempty"`
    Model     string            `json:"model,omitempty"`
    Meta      map[string]string `json:"meta,omitempty"`
}
```

## Supported Providers

### OpenAI

```go
import "github.com/KamdynS/go-agents/llm/openai"

client := openai.NewClient(openai.Config{
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    Model:       "gpt-4",
    Temperature: 0.7,
    MaxTokens:   1000,
})
```

### Anthropic

```go
import "github.com/KamdynS/go-agents/llm/anthropic"

client := anthropic.NewClient(anthropic.Config{
    APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
    Model:       "claude-3-sonnet-20240229",
    Temperature: 0.7,
    MaxTokens:   1000,
})
```

## Usage Examples

### Basic Chat

```go
messages := []llm.Message{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user", Content: "What is the capital of France?"},
}

response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Simple Completion

```go
response, err := client.Completion(ctx, "Translate 'hello' to French:")
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Streaming Responses

```go
output := make(chan *llm.Response)

go func() {
    err := client.Stream(ctx, messages, output)
    if err != nil {
        log.Printf("Streaming error: %v", err)
    }
}()

for response := range output {
    fmt.Print(response.Content)
}
```

## Adding New Providers

To add a new LLM provider:

1. Create a new submodule: `llm/provider-name/`
2. Initialize Go module: `go mod init github.com/KamdynS/go-agents/llm/provider-name`
3. Implement the `llm.Client` interface
4. Add to the workspace in `go.work`

Example structure:
```
llm/
├── client.go              # Core interfaces
├── provider-name/
│   ├── go.mod
│   ├── client.go          # Provider implementation
│   └── README.md          # Provider-specific docs
```

## Configuration

### Common Config Fields

- `APIKey`: Provider API key
- `Model`: Model identifier 
- `Temperature`: Randomness (0.0-2.0)
- `MaxTokens`: Maximum response length

### Provider-Specific Options

Each provider may support additional configuration options. Check the provider's README for details.

## Error Handling

All methods return errors for:
- Network failures
- API authentication issues
- Rate limiting
- Invalid requests
- Model-specific errors

Always check and handle errors appropriately in your application.

## Best Practices

1. **Use context**: Always pass context for cancellation and timeouts
2. **Handle errors**: Check all error returns and implement retry logic where appropriate
3. **Configure timeouts**: Set appropriate timeouts in your HTTP clients
4. **Monitor usage**: Track token usage and API costs
5. **Cache responses**: Consider caching for repeated queries

## Testing

Use the mock implementations for testing:

```go
// TODO: Implement mock client for testing
```

## Contributing

When adding new providers:

1. Follow the existing patterns
2. Add comprehensive tests
3. Update documentation
4. Ensure thread safety
5. Handle all error cases