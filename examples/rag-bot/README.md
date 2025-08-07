# RAG Bot Example

A Retrieval-Augmented Generation (RAG) agent that can search through documents and external APIs to provide informed responses.

## What it does

- **Document Search**: Searches through a local knowledge base
- **External API Access**: Can fetch real-time data from web APIs
- **Intelligent Routing**: Decides when to use each tool based on the query
- **Source Citation**: References where information came from

## Features Demonstrated

- Custom tool implementation (DocumentSearchTool)
- Multiple tool registration and orchestration  
- RAG-style information retrieval
- External API integration
- Tool-assisted conversation flow

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Setup

1. **Set environment variable**
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

2. **Run the RAG bot**
   ```bash
   cd examples/rag-bot
   go run main.go
   ```

## Available Knowledge Base

The bot has access to documents covering:

- **go-basics**: Introduction to Go programming language
- **go-concurrency**: Go's goroutines, channels, and concurrency patterns  
- **go-web-development**: Web development with Go
- **ai-agents**: Overview of AI agent systems

## Example Interactions

### Document-based Questions

```bash
# Ask about Go basics
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What are the key features of Go?"}'

# Ask about concurrency
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "How does Go handle concurrency?"}'

# Ask about AI agents
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What are AI agents and how do they work?"}'
```

### External API Requests

```bash
# Get current weather (if API available)
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the current weather in San Francisco?"}'

# Fetch real-time data
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Get the latest news about artificial intelligence"}'
```

### Mixed Queries

```bash
# Combines document search with potential external lookup
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Compare Go web frameworks with the latest performance benchmarks"}'
```

## Code Architecture

### Custom Document Search Tool

```go
type DocumentSearchTool struct {
    documents map[string]string
}

func (t *DocumentSearchTool) Execute(ctx context.Context, input string) (string, error) {
    // Simple keyword matching against document content
    // Returns relevant document sections
}
```

### Multi-Tool Agent Setup

```go
// Register multiple tools
toolRegistry.Register(docSearchTool)      // Local documents
toolRegistry.Register(httpRequestTool)    // External APIs

// Agent can choose appropriate tool based on context
agent := core.NewChatAgent(core.ChatConfig{
    Tools: toolRegistry,
    Config: core.AgentConfig{
        MaxIterations: 8, // Allow multiple tool calls
        SystemPrompt: "You have access to documents and web APIs...",
    },
})
```

## Agent Decision Making

The agent intelligently chooses tools based on the query:

1. **Document Search First**: For topics covered in the knowledge base
2. **External APIs**: For real-time data or information not in documents
3. **Combined Approach**: Uses both sources when needed
4. **Source Attribution**: Always cites where information came from

## Extending the Knowledge Base

### Add More Documents

```go
docs := map[string]string{
    "new-topic": "Content about new topic...",
    "another-doc": "More information here...",
}
```

### Implement Vector Search

Replace simple keyword matching with:
- Vector embeddings
- Semantic similarity search
- Integration with vector databases (Chroma, Pinecone, etc.)

### Add More Tools

```go
// Custom tool for specific APIs
type WeatherTool struct{}

func (t *WeatherTool) Execute(ctx context.Context, input string) (string, error) {
    // Call weather API
}

toolRegistry.Register(weatherTool)
```

## Advanced Features

### Conversation Memory

The agent maintains conversation context:
- Previous queries and responses
- Tool usage history
- Progressive information building

### Smart Tool Selection

The system prompt guides the agent to:
- Search documents first for known topics
- Use external APIs for real-time data
- Combine sources when appropriate
- Always cite information sources

## Comparison with Python RAG

| Feature | Go RAG Agent | Python (LangChain) |
|---------|--------------|-------------------|
| Performance | High (compiled) | Lower (interpreted) |
| Memory Usage | Low | Higher |
| Deployment | Single binary | Complex dependencies |
| Concurrency | Native goroutines | AsyncIO limitations |
| Type Safety | Compile-time | Runtime |

## Troubleshooting

**"No documents found"**
- Check your search keywords
- Available documents: go-basics, go-concurrency, go-web-development, ai-agents

**External API errors**
- Verify internet connectivity
- Check if the target API requires authentication
- Some APIs may have rate limits

**Tool selection issues**
- The agent should automatically choose appropriate tools
- Check the system prompt if behavior is unexpected

## Next Steps

- Implement vector embeddings for semantic search
- Connect to a real vector database
- Add authentication for external APIs
- Implement document uploading/updating
- Add more specialized tools (database queries, file operations, etc.)