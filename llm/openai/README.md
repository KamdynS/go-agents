# OpenAI LLM Provider

This package provides an OpenAI client implementation for the Go AI Agents framework.

## Installation

```bash
go get github.com/KamdynS/go-agents/llm/openai
```

## Configuration

### Basic Setup

```go
import "github.com/KamdynS/go-agents/llm/openai"

client := openai.NewClient(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})
```

### Configuration Options

```go
type Config struct {
    APIKey      string  // Required: OpenAI API key
    Model       string  // Model to use (default: "gpt-3.5-turbo")
    Temperature float64 // Randomness 0.0-2.0 (default: 0.7)
    MaxTokens   int     // Max response tokens (default: 1000)
}
```

## Supported Models

### Chat Models
- `gpt-4` - Most capable model
- `gpt-4-turbo-preview` - Latest GPT-4 variant
- `gpt-3.5-turbo` - Fast and cost-effective
- `gpt-3.5-turbo-16k` - Extended context window

### Legacy Models
- `gpt-3.5-turbo-instruct` - Instruction-following model

## Usage Examples

### Basic Chat

```go
client := openai.NewClient(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})

messages := []llm.Message{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user", Content: "Explain quantum computing briefly."},
}

response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Function Calling

OpenAI tool/function calling is supported via `ChatRequest.Tools`.
Register tools (functions) by passing their JSON schema, and the client will surface tool calls in `Response.ToolCalls` for your agent loop to execute.

### Streaming Responses

```go
output := make(chan *llm.Response)

go client.Stream(ctx, messages, output)

for response := range output {
    fmt.Print(response.Content)
}
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `OPENAI_API_KEY` | Your OpenAI API key | Yes |
| `OPENAI_ORG_ID` | Organization ID (optional) | No |

## Error Handling

The client handles common OpenAI API errors:

- `401 Unauthorized` - Invalid API key
- `429 Too Many Requests` - Rate limiting
- `500 Internal Server Error` - OpenAI service issues
- Network timeouts and connection errors

## Rate Limiting

OpenAI enforces rate limits based on your plan:

- **Free tier**: 3 requests/minute, 40,000 tokens/minute
- **Pay-as-you-go**: 60 requests/minute, 60,000 tokens/minute
- **Pay-as-you-go (after $5)**: 3,500 requests/minute, 90,000 tokens/minute

The client will return rate limit errors that you should handle with retry logic.

## Cost Optimization

### Token Usage

Monitor token usage to control costs:

```go
response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

log.Printf("Tokens used: %d", response.TokenUsed)
```

### Model Selection

Choose models based on your needs:

- **gpt-3.5-turbo**: $0.002/1K tokens - Good for simple tasks
- **gpt-4**: $0.03/1K tokens - Best for complex reasoning
- **gpt-4-turbo**: $0.01/1K tokens - Balance of capability and cost

## Implementation Status

- [x] Basic chat completion
- [x] Simple completion
- [x] Model identification
- [x] Error handling
- [x] Streaming responses
- [x] Function/tool calling
- [ ] Fine-tuned model support
- [ ] Embeddings API
- [ ] Image generation (DALL-E)
- [ ] Moderation API

## Testing

```go
// TODO: Add testing snippets
```

## Contributing

To contribute to this provider:

1. Implement missing features
2. Add comprehensive tests
3. Update documentation
4. Follow OpenAI API best practices

## References

- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference)
- [OpenAI Go SDK](https://github.com/sashabaranov/go-openai)
- [Rate Limits](https://platform.openai.com/docs/guides/rate-limits)
- [Model Pricing](https://openai.com/pricing)