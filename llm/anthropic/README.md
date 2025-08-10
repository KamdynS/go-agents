# Anthropic LLM Provider

This package provides an Anthropic Claude client implementation for the Go AI Agents framework.

## Installation

```bash
go get github.com/KamdynS/go-agents/llm/anthropic
```

## Configuration

### Basic Setup

```go
import "github.com/KamdynS/go-agents/llm/anthropic"

client := anthropic.NewClient(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-3-sonnet-20240229",
})
```

### Configuration Options

```go
type Config struct {
    APIKey      string  // Required: Anthropic API key
    Model       string  // Model to use (default: "claude-3-sonnet-20240229")
    Temperature float64 // Randomness 0.0-1.0 (default: 0.7)
    MaxTokens   int     // Max response tokens (default: 1000)
}
```

## Supported Models

### Claude 3 Family
- `claude-3-opus-20240229` - Most capable model, best for complex tasks
- `claude-3-sonnet-20240229` - Balanced performance and speed
- `claude-3-haiku-20240307` - Fastest model for simple tasks

### Claude 2 Family (Legacy)
- `claude-2.1` - Previous generation model
- `claude-2.0` - Earlier version with smaller context window

### Model Comparison

| Model | Context Window | Use Case | Relative Cost |
|-------|----------------|----------|---------------|
| Opus | 200K tokens | Complex reasoning, analysis | High |
| Sonnet | 200K tokens | General purpose, balanced | Medium |
| Haiku | 200K tokens | Simple tasks, fast responses | Low |

## Usage Examples

### Basic Chat

```go
client := anthropic.NewClient(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-3-sonnet-20240229",
})

messages := []llm.Message{
    {Role: "user", Content: "What are the key differences between Go and Python?"},
}

response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Long Context Processing

```go
// Claude 3 models support up to 200K token context windows
longDocument := readLargeFile("document.txt") // Up to ~150K tokens

messages := []llm.Message{
    {Role: "user", Content: fmt.Sprintf("Summarize this document:\n\n%s", longDocument)},
}

response, err := client.Chat(ctx, messages)
```

### Tool Use

Tool use support is currently limited; general chat/streaming are implemented. Tool result handling will be expanded as the SDK stabilizes.

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ANTHROPIC_API_KEY` | Your Anthropic API key | Yes |

## Error Handling

The client handles common Anthropic API errors:

- `401 Unauthorized` - Invalid API key
- `429 Too Many Requests` - Rate limiting
- `400 Bad Request` - Invalid request format
- `500 Internal Server Error` - Anthropic service issues

## Rate Limiting

Anthropic enforces rate limits based on your usage tier:

- **Free tier**: Limited requests per minute
- **Paid tier**: Higher limits based on payment history

Rate limits are typically measured in:
- Requests per minute (RPM)
- Tokens per minute (TPM)
- Tokens per day (TPD)

## Cost Optimization

### Token Usage Monitoring

```go
response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

log.Printf("Tokens used: %d", response.TokenUsed)
log.Printf("Model: %s", response.Model)
```

### Model Selection Strategy

- **Haiku**: Use for simple Q&A, basic summarization, quick responses
- **Sonnet**: Use for general-purpose tasks, moderate complexity
- **Opus**: Use for complex analysis, code generation, detailed reasoning

## Claude-Specific Features

### Safety and Harmlessness

Claude models are trained with Constitutional AI principles:
- Built-in safety guardrails
- Helpful, harmless, and honest responses
- Transparent about limitations

### Large Context Windows

All Claude 3 models support 200K token context windows:
- Process entire documents
- Maintain long conversations
- Analyze large codebases

## Implementation Status

- [x] Basic chat completion
- [x] Simple completion
- [x] Model identification
- [x] Error handling
- [x] Streaming responses
- [ ] Tool use (function calling)
- [ ] Vision capabilities (Claude 3 with images)
- [ ] System prompts optimization
- [ ] Conversation management

## Best Practices

### Prompt Engineering

Claude responds well to:
- Clear, specific instructions
- Step-by-step reasoning requests
- Examples of desired output format
- Polite, conversational tone

### Context Management

- Utilize the large context window effectively
- Structure long inputs with clear sections
- Use markdown formatting for better parsing

### Safety Considerations

- Claude may refuse harmful requests
- Be specific about intended use cases
- Respect content policies and guidelines

## Testing

```go
// TODO: Add testing snippets with mock responses
```

## Contributing

To contribute to this provider:

1. Implement streaming responses
2. Add tool use support
3. Implement vision capabilities
4. Add comprehensive tests
5. Follow Anthropic API best practices

## References

- [Anthropic API Documentation](https://docs.anthropic.com/claude/reference/)
- [Claude Model Card](https://www.anthropic.com/claude)
- [Constitutional AI Paper](https://www.anthropic.com/constitutional-ai-harmlessness-from-ai-feedback)
- [Pricing Information](https://www.anthropic.com/pricing)