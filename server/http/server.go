package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/KamdynS/go-agents/agent/core"
)

// Server wraps an agent with HTTP endpoints
type Server struct {
	agent  core.Agent
	config Config
	server *http.Server
}

// Config holds HTTP server configuration
type Config struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	EnableCORS   bool
}

// NewServer creates a new HTTP server for an agent
func NewServer(agent core.Agent, config Config) *Server {
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}

	s := &Server{
		agent:  agent,
		config: config,
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return s
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/chat", s.chatHandler)
	mux.HandleFunc("/chat/stream", s.streamHandler)
}

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Message   string            `json:"message"`
	SessionID string            `json:"session_id,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message   string            `json:"message"`
	SessionID string            `json:"session_id,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// healthHandler provides a health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// chatHandler handles chat requests
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		s.writeError(w, "Message is required", http.StatusBadRequest)
		return
	}

	input := core.Message{
		Role:    "user",
		Content: req.Message,
		Meta:    req.Meta,
	}

	response, err := s.agent.Run(r.Context(), input)
	if err != nil {
		log.Printf("Agent error: %v", err)
		s.writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	chatResp := ChatResponse{
		Message:   response.Content,
		SessionID: req.SessionID,
		Meta:      response.Meta,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

// streamHandler handles streaming chat requests
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	input := core.Message{
		Role:    "user",
		Content: req.Message,
		Meta:    req.Meta,
	}

	// Create a channel for streaming responses
	output := make(chan core.Message)

	go func() {
		if err := s.agent.RunStream(r.Context(), input, output); err != nil {
			log.Printf("Streaming error: %v", err)
		}
	}()

	// Stream responses as SSE events
	for {
		select {
		case message, ok := <-output:
			if !ok {
				// Channel closed, streaming complete
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return
			}

			resp := ChatResponse{
				Message:   message.Content,
				SessionID: req.SessionID,
				Meta:      message.Meta,
			}

			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ChatResponse{Error: message})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(ctx context.Context) error {
	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("HTTP server starting on port %d", s.config.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}