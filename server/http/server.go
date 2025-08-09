package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/KamdynS/go-agents/agent/core"
	obs "github.com/KamdynS/go-agents/observability"
)

// Server wraps an agent with HTTP endpoints
type Server struct {
	agent  core.Agent
	config Config
	server *http.Server
}

// Config holds HTTP server configuration
type Config struct {
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	RequestTimeout time.Duration
	// MaxRequestBodyBytes limits the size of inbound JSON payloads. Defaults to 1 MiB.
	MaxRequestBodyBytes int64
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
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 60 * time.Second
	}
	if config.MaxRequestBodyBytes == 0 {
		config.MaxRequestBodyBytes = 1 << 20 // 1 MiB
	}

	s := &Server{
		agent:  agent,
		config: config,
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Wrap with middleware: recovery -> requestID -> timeout -> metrics/tracing
	var handler http.Handler = mux
	handler = s.recoveryMiddleware(handler)
	handler = s.requestIDMiddleware(handler)
	handler = s.timeoutMiddleware(handler)
	handler = s.observabilityMiddleware(handler)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      handler,
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
	obs.InjectHTTPHeaders(w, r.Context())
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

	// Enforce body size limit and strict JSON decoding
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req ChatRequest
	if err := decoder.Decode(&req); err != nil {
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

	obs.InjectHTTPHeaders(w, r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

// streamHandler handles streaming chat requests
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce body size limit and strict JSON decoding
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req ChatRequest
	if err := decoder.Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	obs.InjectHTTPHeaders(w, r.Context())

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
			// try to send final done event on cancel
			fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
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

// (CORS intentionally omitted in core; add in your application layer)

// timeoutMiddleware enforces a per-request timeout
func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, s.config.RequestTimeout, "request timeout")
}

// requestIDMiddleware extracts or generates a request id and attaches to context
func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := obs.ExtractHTTPContext(r.Context(), r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoveryMiddleware ensures panics become 500 with JSON error
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.writeError(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// observabilityMiddleware records spans and request metrics
func (s *Server) observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		route := r.URL.Path
		method := r.Method

		span, ctx := obs.TracerImpl.StartSpan(r.Context(), "http.request")
		span.SetAttribute(obs.AttrHTTPRoute, route)
		span.SetAttribute(obs.AttrHTTPMethod, method)
		if id, ok := obs.RequestIDFromContext(ctx); ok {
			span.SetAttribute(obs.AttrRequestID, id)
		}

		// Capture status code by wrapping ResponseWriter
		rw := &statusCapturingWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r.WithContext(ctx))

		latency := time.Since(start)
		span.SetAttribute(obs.AttrHTTPStatus, rw.status)
		if rw.status >= 500 {
			span.SetStatus(obs.StatusCodeError, http.StatusText(rw.status))
		} else {
			span.SetStatus(obs.StatusCodeOk, "")
		}
		span.End()

		// Emit metrics
		obs.MetricsImpl.IncrementRequests(map[string]string{
			"route":       route,
			"method":      method,
			"status_code": fmt.Sprintf("%d", rw.status),
		})
		obs.MetricsImpl.RecordLatency(latency, map[string]string{
			"route":       route,
			"method":      method,
			"status_code": fmt.Sprintf("%d", rw.status),
		})
	})
}

type statusCapturingWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
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
