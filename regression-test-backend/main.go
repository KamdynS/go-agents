// Regression Test Server
// This is a simple HTTP server for testing the LLM functionality live
// DO NOT COMMIT - For local testing only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/llm/anthropic"
	"github.com/KamdynS/go-agents/llm/openai"
)

type TestServer struct {
	openaiClient    llm.Client
	anthropicClient llm.Client
}

type TestRequest struct {
	Message  string `json:"message"`
	Provider string `json:"provider"` // "openai" or "anthropic"
	Model    string `json:"model,omitempty"`
}

type TestResponse struct {
	Success   bool                   `json:"success"`
	Response  string                 `json:"response,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Usage     *llm.Usage             `json:"usage,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  string                 `json:"duration"`
}

func main() {
	// Get port from environment
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "18080"
	}

	server, err := NewTestServer()
	if err != nil {
		log.Fatal("Failed to create test server:", err)
	}

	log.Printf("Starting regression test server on port %s...", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/test/llm", server.llmTestHandler)
	mux.HandleFunc("/test/structured", server.structuredTestHandler)
	mux.HandleFunc("/test/models", server.modelsHandler)

	// Start server
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func NewTestServer() (*TestServer, error) {
	// Create OpenAI client
	openaiClient, err := openai.NewClient(openai.Config{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       llm.ModelGPT4oMini, // Use cheaper model for testing
		Temperature: 0.7,
		MaxTokens:   500,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// Create Anthropic client
	anthropicClient, err := anthropic.NewClient(anthropic.Config{
		APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		Model:       llm.ModelClaude35Haiku, // Use cheaper model for testing
		Temperature: 0.7,
		MaxTokens:   500,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	return &TestServer{
		openaiClient:    openaiClient,
		anthropicClient: anthropicClient,
	}, nil
}

func (s *TestServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "regression-test",
		"endpoints": []string{"/health", "/test/llm", "/test/structured", "/test/models"},
	})
}

func (s *TestServer) llmTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	w.Header().Set("Content-Type", "application/json")

	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorResponse(w, "Invalid JSON", start)
		return
	}

	if req.Message == "" {
		s.sendErrorResponse(w, "Message is required", start)
		return
	}

	if req.Provider == "" {
		req.Provider = "openai" // Default to OpenAI
	}

	// Select client
	var client llm.Client
	switch req.Provider {
	case "openai":
		client = s.openaiClient
	case "anthropic":
		client = s.anthropicClient
	default:
		s.sendErrorResponse(w, "Invalid provider. Use 'openai' or 'anthropic'", start)
		return
	}

	// Create request
	chatReq := &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: req.Message},
		},
	}

	if req.Model != "" {
		chatReq.Model = req.Model
	}

	// Make API call
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, chatReq)
	if err != nil {
		s.sendErrorResponse(w, fmt.Sprintf("LLM call failed: %v", err), start)
		return
	}

	// Send successful response
	response := TestResponse{
		Success:   true,
		Response:  resp.Content,
		Usage:     resp.Usage,
		Timestamp: start,
		Duration:  time.Since(start).String(),
		Meta: map[string]interface{}{
			"provider":      req.Provider,
			"model":         resp.Model,
			"finish_reason": resp.FinishReason,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *TestServer) structuredTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	w.Header().Set("Content-Type", "application/json")

	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorResponse(w, "Invalid JSON", start)
		return
	}

	if req.Message == "" {
		s.sendErrorResponse(w, "Message is required", start)
		return
	}

	if req.Provider == "" {
		req.Provider = "openai"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test structured output with sentiment analysis
	var structuredResp interface{}
	var err error
	var usage *llm.Usage

	switch req.Provider {
	case "openai":
		resp, e := openai.StructuredCompletion(
			s.openaiClient.(*openai.Client),
			ctx,
			req.Message,
			llm.Sentiment{},
		)
		if e != nil {
			err = e
		} else {
			structuredResp = resp.Data
			usage = resp.Usage
		}
	case "anthropic":
		resp, e := anthropic.StructuredCompletion(
			s.anthropicClient.(*anthropic.Client),
			ctx,
			req.Message,
			llm.Sentiment{},
		)
		if e != nil {
			err = e
		} else {
			structuredResp = resp.Data
			usage = resp.Usage
		}
	default:
		s.sendErrorResponse(w, "Invalid provider", start)
		return
	}

	if err != nil {
		s.sendErrorResponse(w, fmt.Sprintf("Structured call failed: %v", err), start)
		return
	}

	// Send successful response
	response := TestResponse{
		Success:   true,
		Usage:     usage,
		Timestamp: start,
		Duration:  time.Since(start).String(),
		Meta: map[string]interface{}{
			"provider":        req.Provider,
			"structured_type": "sentiment",
			"structured_data": structuredResp,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *TestServer) modelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	models := map[string]interface{}{
		"openai_models": []string{
			llm.ModelGPT4o,
			llm.ModelGPT4oMini,
			llm.ModelGPT4Turbo,
			llm.ModelGPT35Turbo,
		},
		"anthropic_models": []string{
			llm.ModelClaude35Sonnet,
			llm.ModelClaude35Haiku,
			llm.ModelClaudeOpus,
			llm.ModelClaudeSonnet,
		},
		"model_info": make(map[string]interface{}),
	}

	// Add model metadata
	modelInfo := models["model_info"].(map[string]interface{})
	for _, modelName := range []string{
		llm.ModelGPT4o, llm.ModelGPT4oMini, llm.ModelGPT4Turbo, llm.ModelGPT35Turbo,
		llm.ModelClaude35Sonnet, llm.ModelClaude35Haiku, llm.ModelClaudeOpus, llm.ModelClaudeSonnet,
	} {
		if model, err := llm.GetModel(modelName); err == nil {
			// For compatibility with tests expecting costs per 1k, convert 1M -> 1k
			modelInfo[modelName] = map[string]interface{}{
				"provider":        model.Provider,
				"context_size":    model.ContextSize,
				"input_cost_1k":   model.InputCost / 1000.0,
				"output_cost_1k":  model.OutputCost / 1000.0,
				"supports_tools":  model.Capabilities.FunctionCalling || model.Capabilities.ToolUse,
				"supports_vision": model.Capabilities.Vision,
			}
		}
	}

	json.NewEncoder(w).Encode(models)
}

func (s *TestServer) sendErrorResponse(w http.ResponseWriter, errMsg string, start time.Time) {
	w.WriteHeader(http.StatusBadRequest)
	response := TestResponse{
		Success:   false,
		Error:     errMsg,
		Timestamp: start,
		Duration:  time.Since(start).String(),
	}
	json.NewEncoder(w).Encode(response)
}
