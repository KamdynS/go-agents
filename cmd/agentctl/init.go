package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func initProject(name, projectType string) error {
	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	switch projectType {
	case "basic":
		return initBasicProject(name)
	case "rag":
		return initRAGProject(name)
	case "multi-agent":
		return initMultiAgentProject(name)
	default:
		return fmt.Errorf("unknown project type: %s", projectType)
	}
}

func initBasicProject(name string) error {
	files := map[string]string{
		"main.go": basicMainGo,
		"go.mod":  fmt.Sprintf(goModTemplate, name),
		"README.md": fmt.Sprintf(basicReadme, name),
		"Dockerfile": dockerfileTemplate,
		".gitignore": gitignoreTemplate,
	}

	return writeFiles(name, files)
}

func initRAGProject(name string) error {
	files := map[string]string{
		"main.go": ragMainGo,
		"go.mod":  fmt.Sprintf(goModTemplate, name),
		"README.md": fmt.Sprintf(ragReadme, name),
		"Dockerfile": dockerfileTemplate,
		".gitignore": gitignoreTemplate,
	}

	return writeFiles(name, files)
}

func initMultiAgentProject(name string) error {
	files := map[string]string{
		"main.go": multiAgentMainGo,
		"go.mod":  fmt.Sprintf(goModTemplate, name),
		"README.md": fmt.Sprintf(multiAgentReadme, name),
		"Dockerfile": dockerfileTemplate,
		".gitignore": gitignoreTemplate,
	}

	return writeFiles(name, files)
}

func writeFiles(projectDir string, files map[string]string) error {
	for filename, content := range files {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}
	return nil
}

// Template files
const goModTemplate = `module %s

go 1.21

require (
    github.com/KamdynS/go-agents v0.1.0
    github.com/KamdynS/go-agents/llm/openai v0.1.0
)
`

const basicMainGo = `package main

import (
	"context"
	"log"
	"os"

	"github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/server/http"
	"github.com/KamdynS/go-agents/tools"
)

func main() {
	// Initialize LLM client
	llmClient := openai.NewClient(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-3.5-turbo",
	})

	// Initialize memory store
	memory := inmemory.NewStore()

	// Initialize tools registry
	toolRegistry := tools.NewRegistry()

	// Create agent
	agent := core.NewChatAgent(core.ChatConfig{
		Model: llmClient,
		Mem:   memory,
		Tools: toolRegistry,
		Config: core.AgentConfig{
			SystemPrompt:  "You are a helpful AI assistant.",
			MaxIterations: 5,
			Timeout:       "30s",
		},
	})

	// Create HTTP server
	server := http.NewServer(agent, http.Config{
		Port:       8080,
		EnableCORS: true,
	})

	log.Println("Starting agent server on port 8080...")
	if err := server.ListenAndServe(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
`

const ragMainGo = `package main

import (
	"context"
	"log"
	"os"

	"github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/server/http"
	"github.com/KamdynS/go-agents/tools"
	httpTool "github.com/KamdynS/go-agents/tools/http"
)

func main() {
	// Initialize LLM client
	llmClient := openai.NewClient(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-3.5-turbo",
	})

	// Initialize memory store (conversation + vector store for RAG)
	memory := inmemory.NewStore()

	// Initialize tools registry with HTTP tool for external data access
	toolRegistry := tools.NewRegistry()
	httpRequestTool := httpTool.NewRequestTool(0) // Use default timeout
	toolRegistry.Register(httpRequestTool)

	// Create RAG-enabled agent
	agent := core.NewChatAgent(core.ChatConfig{
		Model: llmClient,
		Mem:   memory,
		Tools: toolRegistry,
		Config: core.AgentConfig{
			SystemPrompt: "You are a RAG-enabled AI assistant. You can retrieve information from external sources using the http_request tool to provide more accurate and up-to-date responses.",
			MaxIterations: 10,
			Timeout:       "60s",
		},
	})

	// Create HTTP server
	server := http.NewServer(agent, http.Config{
		Port:       8080,
		EnableCORS: true,
	})

	log.Println("Starting RAG agent server on port 8080...")
	if err := server.ListenAndServe(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
`

const multiAgentMainGo = `package main

import (
	"context"
	"log"
	"os"

	"github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/server/http"
	"github.com/KamdynS/go-agents/tools"
)

func main() {
	// Initialize LLM client
	llmClient := openai.NewClient(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-3.5-turbo",
	})

	// Initialize shared memory store
	memory := inmemory.NewStore()

	// Initialize tools registry
	toolRegistry := tools.NewRegistry()

	// Create coordinator agent
	coordinator := core.NewChatAgent(core.ChatConfig{
		Model: llmClient,
		Mem:   memory,
		Tools: toolRegistry,
		Config: core.AgentConfig{
			SystemPrompt: "You are a coordinator agent that manages multiple specialized agents. Break down complex tasks and delegate to appropriate specialists.",
			MaxIterations: 5,
			Timeout:       "45s",
		},
	})

	// TODO: Implement multi-agent orchestration
	// For now, using single coordinator agent

	// Create HTTP server
	server := http.NewServer(coordinator, http.Config{
		Port:       8080,
		EnableCORS: true,
	})

	log.Println("Starting multi-agent coordinator server on port 8080...")
	if err := server.ListenAndServe(context.Background()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
`

const basicReadme = `# %s

A basic AI agent built with the Go AI Agent framework.

## Setup

1. Set your OpenAI API key:
   ` + "```" + `bash
   export OPENAI_API_KEY=your_api_key_here
   ` + "```" + `

2. Install dependencies:
   ` + "```" + `bash
   go mod tidy
   ` + "```" + `

3. Run the agent:
   ` + "```" + `bash
   go run main.go
   ` + "```" + `

## Usage

The agent exposes an HTTP API on port 8080:

### Chat endpoint
` + "```" + `bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you?"}'
` + "```" + `

### Health check
` + "```" + `bash
curl http://localhost:8080/health
` + "```" + `

## Customization

- Modify the system prompt in ` + "`" + `main.go` + "`" + `
- Add custom tools by implementing the ` + "`" + `tools.Tool` + "`" + ` interface
- Integrate different LLM providers or memory stores
`

const ragReadme = `# %s

A RAG-enabled AI agent built with the Go AI Agent framework.

## Features

- Retrieval-Augmented Generation (RAG) capabilities
- HTTP tool for external data retrieval
- Conversation memory management

## Setup

1. Set your OpenAI API key:
   ` + "```" + `bash
   export OPENAI_API_KEY=your_api_key_here
   ` + "```" + `

2. Install dependencies:
   ` + "```" + `bash
   go mod tidy
   ` + "```" + `

3. Run the agent:
   ` + "```" + `bash
   go run main.go
   ` + "```" + `

## Usage

This agent can retrieve information from external sources to provide more accurate responses.

### Example query
` + "```" + `bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the current weather in New York?"}'
` + "```" + `

The agent will use the HTTP tool to fetch real-time data when needed.
`

const multiAgentReadme = `# %s

A multi-agent system built with the Go AI Agent framework.

## Features

- Multiple specialized agents
- Coordinator agent for task delegation
- Shared memory and communication

## Setup

1. Set your OpenAI API key:
   ` + "```" + `bash
   export OPENAI_API_KEY=your_api_key_here
   ` + "```" + `

2. Install dependencies:
   ` + "```" + `bash
   go mod tidy
   ` + "```" + `

3. Run the system:
   ` + "```" + `bash
   go run main.go
   ` + "```" + `

## Architecture

- **Coordinator Agent**: Breaks down complex tasks and delegates to specialists
- **Specialist Agents**: Handle specific domains (coming soon)

This is a basic template - extend it by adding your own specialist agents.
`

const dockerfileTemplate = `FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
`

const gitignoreTemplate = `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
main

# Test binary
*.test

# Output of go coverage tool
*.out

# Go workspace file
go.work

# Environment variables
.env

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
`