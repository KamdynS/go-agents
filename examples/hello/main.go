package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/server/http"
	"github.com/KamdynS/go-agents/tools"
)

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
		Temperature: 0.7,
		MaxTokens:   1000,
	})

	// Initialize memory store
	memory := inmemory.NewStore()

	// Initialize tools registry (empty for basic example)
	toolRegistry := tools.NewRegistry()

	// Create agent with helpful system prompt
	agent := core.NewChatAgent(core.ChatConfig{
		Model: llmClient,
		Mem:   memory,
		Tools: toolRegistry,
		Config: core.AgentConfig{
			SystemPrompt: `You are a helpful AI assistant built with the Go AI Agents framework. 
You can have conversations and answer questions on a wide variety of topics. 
Be friendly, informative, and concise in your responses.`,
			MaxIterations: 5,
			Timeout:       "30s",
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

	log.Println("🚀 Starting Hello World agent server...")
	log.Println("📡 Server running on http://localhost:8080")
	log.Println("🔍 Health check: http://localhost:8080/health")
	log.Println("💬 Chat endpoint: http://localhost:8080/chat")
	log.Println("Press Ctrl+C to stop")

	if err := server.ListenAndServe(ctx); err != nil {
		log.Printf("Server error: %v", err)
	}

	log.Println("Server stopped")
}