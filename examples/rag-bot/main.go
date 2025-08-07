package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/server/http"
	"github.com/KamdynS/go-agents/tools"
	httpTool "github.com/KamdynS/go-agents/tools/http"
)

// DocumentSearchTool demonstrates a simple document search capability
type DocumentSearchTool struct {
	documents map[string]string // Simple in-memory document store
}

func NewDocumentSearchTool() *DocumentSearchTool {
	// Sample documents for demonstration
	docs := map[string]string{
		"go-basics": `Go is a programming language developed by Google. Key features include:
- Compiled language with garbage collection
- Strong static typing
- Built-in concurrency with goroutines and channels
- Simple syntax and fast compilation
- Excellent standard library
- Cross-platform support`,

		"go-concurrency": `Go's concurrency model is based on:
- Goroutines: lightweight threads managed by the Go runtime
- Channels: typed conduits for communication between goroutines
- Select statement: for non-blocking channel operations
- Mutex: for protecting shared data
- WaitGroup: for waiting for a collection of goroutines

The motto is: "Don't communicate by sharing memory, share memory by communicating"`,

		"go-web-development": `Go is excellent for web development with:
- net/http package for HTTP servers and clients
- Template packages for HTML generation
- JSON encoding/decoding built-in
- Popular frameworks: Gin, Echo, Fiber
- Excellent performance and low memory usage
- Easy deployment as single binaries`,

		"ai-agents": `AI agents are autonomous software systems that:
- Perceive their environment through sensors
- Process information and make decisions
- Act upon the environment through actuators
- Learn from experience to improve performance
- Can be reactive, proactive, or goal-oriented
- Often use machine learning and natural language processing`,
	}

	return &DocumentSearchTool{
		documents: docs,
	}
}

func (t *DocumentSearchTool) Name() string {
	return "document_search"
}

func (t *DocumentSearchTool) Description() string {
	return "Search through available documents for relevant information. Provide keywords or topics to search for."
}

func (t *DocumentSearchTool) Execute(ctx context.Context, input string) (string, error) {
	query := strings.ToLower(strings.TrimSpace(input))
	if query == "" {
		return "Please provide search keywords or topics", nil
	}

	var results []string
	var matchingDocs []string

	// Simple keyword matching
	for docID, content := range t.documents {
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, query) || strings.Contains(docID, query) {
			results = append(results, fmt.Sprintf("Document: %s\n%s", docID, content))
			matchingDocs = append(matchingDocs, docID)
		}
	}

	if len(results) == 0 {
		available := make([]string, 0, len(t.documents))
		for docID := range t.documents {
			available = append(available, docID)
		}
		return fmt.Sprintf("No documents found matching '%s'. Available documents: %s", 
			query, strings.Join(available, ", ")), nil
	}

	response := fmt.Sprintf("Found %d document(s) matching '%s':\n\n", len(results), query)
	response += strings.Join(results, "\n\n---\n\n")

	return response, nil
}

func (t *DocumentSearchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "Keywords or topics to search for in documents",
				"example":     "go concurrency",
			},
		},
		"required": []string{"input"},
	}
}

func main() {
	// Check for required environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Initialize LLM client
	llmClient := openai.NewClient(openai.Config{
		APIKey:      apiKey,
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3, // Lower temperature for more focused responses
		MaxTokens:   1500, // Higher limit for detailed responses
	})

	// Initialize memory store
	memory := inmemory.NewStore()

	// Initialize tools registry with RAG capabilities
	toolRegistry := tools.NewRegistry()

	// Add document search tool
	docSearchTool := NewDocumentSearchTool()
	if err := toolRegistry.Register(docSearchTool); err != nil {
		log.Fatalf("Failed to register document search tool: %v", err)
	}

	// Add HTTP request tool for external data
	httpRequestTool := httpTool.NewRequestTool(30 * time.Second)
	if err := toolRegistry.Register(httpRequestTool); err != nil {
		log.Fatalf("Failed to register HTTP request tool: %v", err)
	}

	// Create RAG-enabled agent
	agent := core.NewChatAgent(core.ChatConfig{
		Model: llmClient,
		Mem:   memory,
		Tools: toolRegistry,
		Config: core.AgentConfig{
			SystemPrompt: `You are a knowledgeable assistant with access to a document search system and web requests.

When answering questions:
1. First, search the available documents using the document_search tool if the question relates to Go programming or AI agents
2. If you need additional or more current information, you can use the http_request tool to fetch data from APIs
3. Always cite your sources when providing information from documents
4. Provide comprehensive, accurate answers based on the retrieved information
5. If you can't find relevant information, say so clearly

Available document topics include: Go programming basics, concurrency, web development, and AI agents.`,
			MaxIterations: 8, // Allow multiple tool calls
			Timeout:       "60s",
		},
	})

	// Create HTTP server
	server := http.NewServer(agent, http.Config{
		Port:       8080,
		EnableCORS: true,
	})

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	log.Println("🤖 Starting RAG-enabled agent server...")
	log.Println("📡 Server running on http://localhost:8080")
	log.Println("🔍 Health check: http://localhost:8080/health")
	log.Println("💬 Chat endpoint: http://localhost:8080/chat")
	log.Println("📚 Available documents: go-basics, go-concurrency, go-web-development, ai-agents")
	log.Println("🌐 External data access via HTTP requests enabled")
	log.Println("Press Ctrl+C to stop")

	if err := server.ListenAndServe(ctx); err != nil {
		log.Printf("Server error: %v", err)
	}

	log.Println("Server stopped")
}