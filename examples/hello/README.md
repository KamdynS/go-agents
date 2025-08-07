# Hello World Agent Example

This is the simplest possible AI agent built with the Go AI Agents framework.

## What it does

- Creates a basic conversational AI agent
- Exposes HTTP endpoints for chat interaction
- Uses OpenAI's GPT-3.5-turbo model
- Includes graceful shutdown handling

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Setup

1. **Get an OpenAI API key**
   - Sign up at [OpenAI Platform](https://platform.openai.com/)
   - Generate an API key from your account dashboard

2. **Set environment variable**
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

3. **Run the example**
   ```bash
   cd examples/hello
   go run main.go
   ```

## Testing

Once the server is running, you can interact with it:

### Health Check
```bash
curl http://localhost:8080/health
```

### Chat with the Agent
```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello! What can you help me with?"}'
```

### Example Conversations

```bash
# Ask a question
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the capital of Japan?"}'

# Get help with coding
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "How do I reverse a string in Go?"}'

# Have a casual conversation
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me a joke about programming"}'
```

## Code Structure

```go
// 1. Initialize LLM client
llmClient := openai.NewClient(openai.Config{
    APIKey: apiKey,
    Model:  "gpt-3.5-turbo",
})

// 2. Create memory and tools
memory := inmemory.NewStore()
toolRegistry := tools.NewRegistry()

// 3. Create the agent
agent := core.NewChatAgent(core.ChatConfig{
    Model: llmClient,
    Mem:   memory,
    Tools: toolRegistry,
    Config: core.AgentConfig{
        SystemPrompt: "You are a helpful AI assistant...",
    },
})

// 4. Serve via HTTP
server := http.NewServer(agent, http.Config{Port: 8080})
server.ListenAndServe(ctx)
```

## Customization

### Change the Model
```go
llmClient := openai.NewClient(openai.Config{
    APIKey: apiKey,
    Model:  "gpt-4", // More capable but slower/expensive
})
```

### Customize System Prompt
```go
Config: core.AgentConfig{
    SystemPrompt: "You are a specialized coding assistant that only helps with Go programming questions.",
}
```

### Change the Port
```go
server := http.NewServer(agent, http.Config{
    Port: 3000, // Run on port 3000 instead
})
```

## Next Steps

- Try the [RAG Bot Example](../rag-bot/) for document-based Q&A
- Add custom tools to extend agent capabilities
- Implement streaming responses
- Add authentication and rate limiting

## Troubleshooting

**Error: "OPENAI_API_KEY environment variable is required"**
- Make sure you've set the environment variable with your OpenAI API key

**Error: "connection refused"**
- Check that the server started successfully on port 8080
- Make sure no other service is using port 8080

**Error: "invalid API key"**
- Verify your OpenAI API key is correct
- Check that you have credits available in your OpenAI account