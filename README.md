# Go AI Agents

The first Go-native framework for building and deploying production-ready AI agent microservices.

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%2300ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

Go AI Agents is a comprehensive framework designed to bridge the gap between AI agent prototyping and production deployment. While Python dominates AI experimentation, Go excels at building reliable, high-performance microservices. This framework combines both worlds.

### Key Features

- 🚀 **Production-Ready**: Built-in HTTP server, observability, and deployment tools
- ⚡ **High Performance**: Leverages Go's concurrency for parallel LLM calls and tool execution
- 🔧 **Modular Architecture**: Pluggable LLM providers, memory stores, and tools
- 📦 **Single Binary Deployment**: No runtime dependencies, containerization-friendly
- 🛡️ **Type Safety**: Compile-time guarantees for agent workflows
- 🔍 **Observability**: Built-in metrics, tracing, and logging
- 🌐 **Multi-Provider**: Support for OpenAI, Anthropic, and other LLM providers

## Quick Start

### Prerequisites

- Go 1.21 or later
- OpenAI API key (or other LLM provider)

### Installation

```bash
# Install the CLI tool
go install github.com/KamdynS/go-agents/cmd/agentctl@latest

# Create a new agent project
agentctl init my-agent
cd my-agent

# Set up environment
export OPENAI_API_KEY=your_api_key_here

# Run the agent
go run main.go
```

Your agent is now running on `http://localhost:8080`!

### Test the Agent

```bash
# Health check
curl http://localhost:8080/health

# Chat endpoint
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you?"}'

# Streaming chat
curl -X POST http://localhost:8080/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me a story"}' \
  --no-buffer
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Agent Core    │────│   LLM Client    │────│   Providers     │
│  (Orchestrator) │    │   (Interface)   │    │ (OpenAI, etc.)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ├─────────────────────────────────────────────────────────────┐
         │                              │                              │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Tools       │    │     Memory      │    │  Observability  │
│   (Registry)    │    │     (Store)     │    │ (Metrics/Traces) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                              │                              │
         ├──────────────────────────────┼──────────────────────────────┘
         │                              │
┌─────────────────┐    ┌─────────────────┐
│  HTTP Server    │    │   Deployment    │
│   Server        │    │     Tools       │
└─────────────────┘    └─────────────────┘
```

## Core Concepts

### Agents

Agents are the core orchestrators that manage the reasoning-action loop:

```go
type Agent interface {
    Run(ctx context.Context, input Message) (Message, error)
    RunStream(ctx context.Context, input Message, output chan<- Message) error
}
```

### LLM Clients

Our LLM abstraction provides a unified interface across providers with production-ready features:

#### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/KamdynS/go-agents/llm"
    "github.com/KamdynS/go-agents/llm/openai"
)

func main() {
    // Create client with typed model constants (no magic strings!)
    client, err := openai.NewClient(openai.Config{
        APIKey: "your-api-key",
        Model:  llm.ModelGPT4o,  // Typed constant
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Make request
    req := &llm.ChatRequest{
        Messages: []llm.Message{
            {Role: "user", Content: "Hello!"},
        },
    }
    
    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response: %s", resp.Content)
    log.Printf("Cost: $%.4f", resp.Usage.Cost)
    log.Printf("Tokens: %d", resp.Usage.TotalTokens)
}
```

#### Structured Output (Python Instructor-like)

```go
// Define your structure
type SentimentAnalysis struct {
    llm.BaseStructured
    Sentiment  string  `json:"sentiment" description:"positive, negative, or neutral"`
    Score      float64 `json:"score" description:"confidence score between -1 and 1"`
    Reasoning  string  `json:"reasoning" description:"brief explanation"`
}

func (s SentimentAnalysis) Validate() error {
    validSentiments := []string{"positive", "negative", "neutral"}
    for _, valid := range validSentiments {
        if s.Sentiment == valid {
            return nil
        }
    }
    return fmt.Errorf("invalid sentiment: %s", s.Sentiment)
}

// Use structured output
func main() {
    client, _ := openai.NewClient(openai.Config{...})
    
    resp, err := openai.StructuredCompletion(
        client,
        context.Background(),
        "I love this product! It's amazing.",
        SentimentAnalysis{},
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // resp.Data is fully typed and validated
    fmt.Printf("Sentiment: %s (%.2f)\n", resp.Data.Sentiment, resp.Data.Score)
    fmt.Printf("Reasoning: %s\n", resp.Data.Reasoning)
}
```

#### Available Models (Typed Constants)

All models are defined as typed constants with metadata:

```go
// OpenAI Models
llm.ModelGPT4o              // Most capable, latest
llm.ModelGPT4oMini          // Fast and cheap  
llm.ModelGPT4Turbo          // Previous generation
llm.ModelGPT35Turbo         // Legacy, cheapest

// Anthropic Models
llm.ModelClaude35Sonnet     // Most capable
llm.ModelClaude35Haiku      // Fast and cheap
llm.ModelClaude3Opus        // Previous generation
llm.ModelClaude3Sonnet      // Balanced

// Get model info
model, _ := llm.GetModel(llm.ModelGPT4o)
fmt.Printf("Max tokens: %d\n", model.ContextSize)
fmt.Printf("Cost per 1K tokens: $%.4f\n", model.InputCostPer1K)
```

#### Error Handling & Retries

Built-in comprehensive error handling with exponential backoff:

```go
config := openai.Config{
    APIKey: "your-key",
    Model:  llm.ModelGPT4o,
    RetryConfig: llm.RetryConfig{
        MaxRetries:    3,
        InitialDelay:  time.Second,
        MaxDelay:      60 * time.Second,
        BackoffFactor: 2.0,
        RetryableErrors: []string{"rate_limit", "server_error", "timeout"},
    },
}

client, _ := openai.NewClient(config)

// Automatically retries on transient failures
resp, err := client.Chat(ctx, req)
if llm.IsRateLimitError(err) {
    // Handle rate limit specifically
} else if llm.IsRetryableError(err) {
    // Was retried but still failed
}
```

#### Client Interface

```go
type Client interface {
    Chat(ctx context.Context, req *ChatRequest) (*Response, error)
    Completion(ctx context.Context, prompt string) (*Response, error)
    Stream(ctx context.Context, req *ChatRequest, output chan<- *Response) error
    Model() string
    Provider() Provider
    Validate() error
}
```

### Tools

Extensible tools that agents can use to interact with external systems:

```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, input string) (string, error)
    Schema() map[string]interface{}
}
```

### Memory

Pluggable storage for conversation history and long-term memory:

```go
type Store interface {
    Store(ctx context.Context, key string, value interface{}) error
    Retrieve(ctx context.Context, key string) (interface{}, error)
    // ... other methods
}
```

## Project Types

### Basic Agent

A simple conversational agent:

```bash
agentctl init --type=basic my-chatbot
```

### RAG Agent

Retrieval-Augmented Generation with external data sources:

```bash
agentctl init --type=rag my-rag-bot
```

### Multi-Agent System

Coordinated agents for complex workflows:

```bash
agentctl init --type=multi-agent my-team
```

## Deployment

### Docker

```bash
# Build image
docker build -f deploy/Dockerfile -t my-agent .

# Run container
docker run -p 8080:8080 -e OPENAI_API_KEY=your_key my-agent
```

### Docker Compose

```bash
cd deploy
docker-compose up -d
```

Note: services under `deploy/` are for demos. For production, use your application repo's deployment.

### Kubernetes

This repository is a library. Use your application repo's own Kubernetes manifests. Previously referenced `deploy/k8s/` is intentionally not included here.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key | required |
| `ANTHROPIC_API_KEY` | Anthropic API key | optional |
| `AGENT_PORT` | HTTP server port | 8080 |
| `AGENT_LOG_LEVEL` | Log level | info |

### Security
- Libraries accept API keys only via code (config structs). Packages do not read environment variables and never log API keys.
- Applications and examples may load keys from environment variables for deployability. If you use the regression tests, create a `.env` at repo root (see `regression-test-backend/run.sh`).
- Core HTTP server intentionally omits CORS/auth; add those in your application or reverse proxy.
- Avoid logging sensitive configuration values.

### Build Tags
- Optional adapters are behind build tags:
  - `adapters_redis`
  - `adapters_pgvector`
- Examples:
  - Compile and test: `go test ./... -race -tags adapters_redis,adapters_pgvector`
  - Smoke with external services: start Redis/Postgres, set `DATABASE_URL`, then run with those tags

### Programmatic Configuration

```go
agent := core.NewChatAgent(core.ChatConfig{
    Model: openai.NewClient(openai.Config{
        APIKey: "your-key",
        Model:  "gpt-4",
    }),
    Config: core.AgentConfig{
        SystemPrompt:  "You are a helpful assistant",
        MaxIterations: 5,
        Timeout:       "30s",
    },
})
```

## Integration

Use the library inside your own server or framework. A minimal reference HTTP/SSE server is provided in `server/http`, but CORS/auth/policy should be implemented in your app or reverse proxy.

Basic example using the reference server:

```go
agent := core.NewChatAgent(core.ChatConfig{ /* ... */ })
srv := httpserver.NewServer(agent, httpserver.Config{ Port: 8080 })
ctx := context.Background()
_ = srv.ListenAndServe(ctx)
```

For custom servers, call your `core.Agent` directly and shape HTTP/SSE responses as needed.

## Development

### Prerequisites

- Go 1.21+
- Docker (for testing)

### Building from Source

```bash
git clone https://github.com/KamdynS/go-agents.git
cd go-agents

# Install dependencies
go mod tidy

# Build CLI tool
go build -o bin/agentctl ./cmd/agentctl

# Run tests
go test ./...
```

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Roadmap

- [x] Core agent framework
- [x] OpenAI and Anthropic providers
- [x] HTTP server and CLI tools
  
- [ ] Vector database integrations
- [ ] Multi-agent orchestration
- [ ] Streaming responses
- [ ] Plugin ecosystem
- [ ] Performance benchmarks

## Comparison with Other Frameworks

| Feature | Go AI Agents | LangChain | AutoGen | Semantic Kernel |
|---------|--------------|-----------|---------|-----------------|
| Language | Go | Python | Python | C#/Python |
| Production Ready | ✅ | ⚠️ | ⚠️ | ✅ |
| Type Safety | ✅ | ❌ | ❌ | ✅ |
| Single Binary | ✅ | ❌ | ❌ | ❌ |
| Concurrency | ✅ (Goroutines) | ⚠️ (AsyncIO) | ⚠️ (AsyncIO) | ✅ (Tasks) |
| Deployment | ✅ | ⚠️ | ⚠️ | ✅ |

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](./docs/)
- 💬 [Discussions](https://github.com/KamdynS/go-agents/discussions)
- 🐛 [Issues](https://github.com/KamdynS/go-agents/issues)

---

Built with ❤️ for the Go and AI communities.