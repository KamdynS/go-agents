package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KamdynS/go-agents/agent/core"
	obs "github.com/KamdynS/go-agents/observability"
)

// Mock Agent for testing
type MockAgent struct {
	responses    []core.Message
	calls        []core.Message
	nextIndex    int
	shouldErr    bool
	err          error
	streamDelay  time.Duration
	streamChunks []string
}

func NewMockAgent() *MockAgent {
	return &MockAgent{
		responses: []core.Message{},
		calls:     []core.Message{},
	}
}

func (m *MockAgent) AddResponse(content string) {
	m.responses = append(m.responses, core.Message{
		Role:    "assistant",
		Content: content,
	})
}

func (m *MockAgent) SetError(err error) {
	m.shouldErr = true
	m.err = err
}

func (m *MockAgent) SetStreamChunks(chunks []string, delay time.Duration) {
	m.streamChunks = chunks
	m.streamDelay = delay
}

func (m *MockAgent) Run(ctx context.Context, input core.Message) (core.Message, error) {
	// Store the call for inspection
	m.calls = append(m.calls, input)

	if m.shouldErr {
		return core.Message{}, m.err
	}

	if m.nextIndex >= len(m.responses) {
		return core.Message{
			Role:    "assistant",
			Content: "Default mock response",
		}, nil
	}

	response := m.responses[m.nextIndex]
	m.nextIndex++
	return response, nil
}

func (m *MockAgent) RunStream(ctx context.Context, input core.Message, output chan<- core.Message) error {
	defer close(output)

	// Store the call for inspection
	m.calls = append(m.calls, input)

	if m.shouldErr {
		return m.err
	}

	// Send chunks with delay if configured
	if len(m.streamChunks) > 0 {
		for _, chunk := range m.streamChunks {
			select {
			case output <- core.Message{Role: "assistant", Content: chunk}:
				if m.streamDelay > 0 {
					time.Sleep(m.streamDelay)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	// Default streaming behavior - just use Run result
	result, err := m.Run(ctx, input)
	if err != nil {
		return err
	}

	select {
	case output <- result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MockAgent) GetCalls() []core.Message {
	return m.calls
}

func TestNewServer(t *testing.T) {
	agent := NewMockAgent()
	config := Config{
		Port:         9090,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	server := NewServer(agent, config)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.agent != agent {
		t.Error("Agent not set correctly")
	}

	if server.config.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", server.config.Port)
	}

	if server.config.ReadTimeout != 5*time.Second {
		t.Errorf("Expected ReadTimeout 5s, got %v", server.config.ReadTimeout)
	}

	if server.config.WriteTimeout != 5*time.Second {
		t.Errorf("Expected WriteTimeout 5s, got %v", server.config.WriteTimeout)
	}

	// CORS is not part of core server anymore

	if server.server == nil {
		t.Error("HTTP server not initialized")
	}

	expectedAddr := ":9090"
	if server.server.Addr != expectedAddr {
		t.Errorf("Expected server addr %s, got %s", expectedAddr, server.server.Addr)
	}
}

func TestNewServer_DefaultConfig(t *testing.T) {
	agent := NewMockAgent()
	config := Config{} // Empty config to test defaults

	server := NewServer(agent, config)

	if server.config.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", server.config.Port)
	}

	if server.config.ReadTimeout != 10*time.Second {
		t.Errorf("Expected default ReadTimeout 10s, got %v", server.config.ReadTimeout)
	}

	if server.config.WriteTimeout != 10*time.Second {
		t.Errorf("Expected default WriteTimeout 10s, got %v", server.config.WriteTimeout)
	}
}

func TestServer_HealthHandler(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", response["status"])
	}

	if response["time"] == "" {
		t.Error("Expected time field to be set")
	}

	// Verify time format
	_, err = time.Parse(time.RFC3339, response["time"])
	if err != nil {
		t.Errorf("Invalid time format: %v", err)
	}
}

func TestServer_ChatHandler_Success(t *testing.T) {
	agent := NewMockAgent()
	agent.AddResponse("Hello! How can I help you today?")

	server := NewServer(agent, Config{})

	requestBody := ChatRequest{
		Message:   "Hello",
		SessionID: "test-session",
		Meta:      map[string]string{"source": "test"},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.chatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response ChatResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response.Message != "Hello! How can I help you today?" {
		t.Errorf("Expected response message 'Hello! How can I help you today?', got %s", response.Message)
	}

	if response.SessionID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got %s", response.SessionID)
	}

	if response.Error != "" {
		t.Errorf("Expected no error, got %s", response.Error)
	}

	// Verify agent was called correctly
	calls := agent.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 agent call, got %d", len(calls))
	}

	call := calls[0]
	if call.Role != "user" {
		t.Errorf("Expected call role 'user', got %s", call.Role)
	}

	if call.Content != "Hello" {
		t.Errorf("Expected call content 'Hello', got %s", call.Content)
	}

	if call.Meta["source"] != "test" {
		t.Errorf("Expected meta source 'test', got %s", call.Meta["source"])
	}
}

func TestServer_ChatHandler_MethodNotAllowed(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	req := httptest.NewRequest("GET", "/chat", nil)
	w := httptest.NewRecorder()

	server.chatHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code 405, got %d", w.Code)
	}
}

func TestServer_ChatHandler_InvalidJSON(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	req := httptest.NewRequest("POST", "/chat", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.chatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}

	var response ChatResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "Invalid JSON" {
		t.Errorf("Expected error 'Invalid JSON', got %s", response.Error)
	}
}

func TestServer_ChatHandler_EmptyMessage(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	requestBody := ChatRequest{Message: ""}
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.chatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}

	var response ChatResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "Message is required" {
		t.Errorf("Expected error 'Message is required', got %s", response.Error)
	}
}

func TestServer_ChatHandler_AgentError(t *testing.T) {
	agent := NewMockAgent()
	agent.SetError(fmt.Errorf("agent processing error"))

	server := NewServer(agent, Config{})

	requestBody := ChatRequest{Message: "This will fail"}
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.chatHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", w.Code)
	}

	var response ChatResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "Internal server error" {
		t.Errorf("Expected error 'Internal server error', got %s", response.Error)
	}
}

func TestServer_StreamHandler_Success(t *testing.T) {
	agent := NewMockAgent()
	agent.SetStreamChunks([]string{"Hello", " world", "!"}, 10*time.Millisecond)

	server := NewServer(agent, Config{})

	requestBody := ChatRequest{
		Message:   "Stream this response",
		SessionID: "stream-session",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.streamHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control no-cache, got %s", cacheControl)
	}

	connection := w.Header().Get("Connection")
	if connection != "keep-alive" {
		t.Errorf("Expected Connection keep-alive, got %s", connection)
	}

	// Check response body contains SSE events
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "event: message") {
		t.Error("Response should contain SSE message events")
	}

	if !strings.Contains(responseBody, "event: done") {
		t.Error("Response should contain SSE done event")
	}

	// Verify agent was called
	calls := agent.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 agent call, got %d", len(calls))
	}
}

func TestServer_StreamHandler_MethodNotAllowed(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	req := httptest.NewRequest("GET", "/chat/stream", nil)
	w := httptest.NewRecorder()

	server.streamHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code 405, got %d", w.Code)
	}
}

func TestServer_StreamHandler_CancelSendsDone(t *testing.T) {
	agent := NewMockAgent()
	agent.SetStreamChunks([]string{"chunk1", "chunk2"}, 50*time.Millisecond)

	server := NewServer(agent, Config{})

	reqBody := ChatRequest{Message: "cancel soon"}
	body, _ := json.Marshal(reqBody)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "/chat/stream", bytes.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Cancel shortly after starting
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	server.streamHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "event: done") {
		t.Error("expected final done event on cancel")
	}
}

func TestObservability_HealthSpanAndMetrics(t *testing.T) {
	// Swap in-memory implementations
	tracer := obs.NewDefaultTracer()
	metrics := obs.NewDefaultMetrics()
	obs.SetTracer(tracer)
	obs.SetMetrics(metrics)

	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	// Exercise through full middleware chain
	ts := httptest.NewServer(server.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID header")
	}

	// Verify metrics changed
	stats := metrics.GetStats()
	if stats["requests"].(int64) == 0 {
		t.Error("expected requests counter to increment")
	}

	// Verify a span recorded
	spans := tracer.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span recorded")
	}
}

func TestServer_StreamHandler_InvalidJSON(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	req := httptest.NewRequest("POST", "/chat/stream", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.streamHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

func TestServer_WriteError(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	w := httptest.NewRecorder()
	server.writeError(w, "Test error message", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response ChatResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response.Error != "Test error message" {
		t.Errorf("Expected error 'Test error message', got %s", response.Error)
	}
}

func TestServer_CorsMiddleware(t *testing.T) {
	t.Skip("CORS middleware removed from core; test skipped")
}

func TestServer_CorsMiddleware_Options(t *testing.T) {
	t.Skip("CORS middleware removed from core; test skipped")
}

func TestChatRequest(t *testing.T) {
	req := ChatRequest{
		Message:   "Test message",
		SessionID: "session123",
		Meta: map[string]string{
			"client": "test",
		},
	}

	if req.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", req.Message)
	}

	if req.SessionID != "session123" {
		t.Errorf("Expected session ID 'session123', got %s", req.SessionID)
	}

	if req.Meta["client"] != "test" {
		t.Errorf("Expected meta client 'test', got %s", req.Meta["client"])
	}
}

func TestChatResponse(t *testing.T) {
	resp := ChatResponse{
		Message:   "Response message",
		SessionID: "session456",
		Meta: map[string]string{
			"model": "test-model",
		},
		Error: "",
	}

	if resp.Message != "Response message" {
		t.Errorf("Expected message 'Response message', got %s", resp.Message)
	}

	if resp.SessionID != "session456" {
		t.Errorf("Expected session ID 'session456', got %s", resp.SessionID)
	}

	if resp.Meta["model"] != "test-model" {
		t.Errorf("Expected meta model 'test-model', got %s", resp.Meta["model"])
	}

	if resp.Error != "" {
		t.Errorf("Expected no error, got %s", resp.Error)
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Port:         9999,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	if config.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", config.Port)
	}

	if config.ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout 15s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 20*time.Second {
		t.Errorf("Expected WriteTimeout 20s, got %v", config.WriteTimeout)
	}

	// CORS no longer part of Config
}

// Integration test with httptest server
func TestServer_Integration(t *testing.T) {
	agent := NewMockAgent()
	agent.AddResponse("Integration test response")

	server := NewServer(agent, Config{})

	// Create test server
	testServer := httptest.NewServer(server.server.Handler)
	defer testServer.Close()

	// Test health endpoint
	resp, err := http.Get(testServer.URL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected health status 200, got %d", resp.StatusCode)
	}

	// Test chat endpoint
	chatReq := ChatRequest{
		Message: "Integration test message",
	}
	body, _ := json.Marshal(chatReq)

	resp, err = http.Post(testServer.URL+"/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Chat request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected chat status 200, got %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &chatResp)
	if err != nil {
		t.Errorf("Failed to parse chat response: %v", err)
	}

	if chatResp.Message != "Integration test response" {
		t.Errorf("Expected response 'Integration test response', got %s", chatResp.Message)
	}

	// Verify agent was called
	calls := agent.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 agent call, got %d", len(calls))
	}

	if calls[0].Content != "Integration test message" {
		t.Errorf("Expected agent call content 'Integration test message', got %s", calls[0].Content)
	}
}

func TestServer_Shutdown(t *testing.T) {
	agent := NewMockAgent()
	server := NewServer(agent, Config{})

	// Test shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown error: %v", err)
	}
}
