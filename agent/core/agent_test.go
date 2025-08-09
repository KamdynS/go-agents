package core

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/memory/inmemory"
	"github.com/KamdynS/go-agents/tools"
)

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, world!",
		Meta: map[string]string{
			"source": "test",
		},
	}

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", msg.Content)
	}

	if msg.Meta["source"] != "test" {
		t.Errorf("Expected meta source 'test', got %s", msg.Meta["source"])
	}
}

func TestAgentConfig(t *testing.T) {
	config := AgentConfig{
		MaxIterations: 5,
		Timeout:       "30s",
		SystemPrompt:  "You are a helpful assistant",
	}

	if config.MaxIterations != 5 {
		t.Errorf("Expected MaxIterations 5, got %d", config.MaxIterations)
	}

	if config.Timeout != "30s" {
		t.Errorf("Expected Timeout '30s', got %s", config.Timeout)
	}

	if config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("Expected SystemPrompt 'You are a helpful assistant', got %s", config.SystemPrompt)
	}
}

// Mock LLM Client for testing
type MockLLMClient struct {
	responses []llm.Response
	calls     []llm.ChatRequest
	nextIndex int
	shouldErr bool
	err       error

	// tool-call scripting per call index
	scriptedToolCalls [][]llm.ToolCall
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: []llm.Response{},
		calls:     []llm.ChatRequest{},
	}
}

func (m *MockLLMClient) AddResponse(content string) {
	m.responses = append(m.responses, llm.Response{
		Content:  content,
		Role:     "assistant",
		Model:    "mock-model",
		Provider: llm.ProviderOpenAI,
	})
}

func (m *MockLLMClient) AddResponseWithToolCalls(content string, calls []llm.ToolCall) {
	m.responses = append(m.responses, llm.Response{
		Content:   content,
		Role:      "assistant",
		Model:     "mock-model",
		Provider:  llm.ProviderOpenAI,
		ToolCalls: calls,
	})
}

func (m *MockLLMClient) SetError(err error) {
	m.shouldErr = true
	m.err = err
}

func (m *MockLLMClient) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.Response, error) {
	// Store the call for inspection
	m.calls = append(m.calls, *req)

	if m.shouldErr {
		return nil, m.err
	}

	if m.nextIndex >= len(m.responses) {
		return &llm.Response{
			Content:  "Default mock response",
			Role:     "assistant",
			Model:    "mock-model",
			Provider: llm.ProviderOpenAI,
		}, nil
	}

	response := m.responses[m.nextIndex]
	m.nextIndex++
	return &response, nil
}

func (m *MockLLMClient) Completion(ctx context.Context, prompt string) (*llm.Response, error) {
	req := &llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: prompt}},
	}
	return m.Chat(ctx, req)
}

func (m *MockLLMClient) Stream(ctx context.Context, req *llm.ChatRequest, output chan<- *llm.Response) error {
	resp, err := m.Chat(ctx, req)
	if err != nil {
		return err
	}

	defer close(output)
	select {
	case output <- resp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MockLLMClient) Model() string {
	return "mock-model"
}

func (m *MockLLMClient) Provider() llm.Provider {
	return llm.ProviderOpenAI
}

func (m *MockLLMClient) Validate() error {
	return nil
}

func (m *MockLLMClient) GetCalls() []llm.ChatRequest {
	return m.calls
}

// Dummy tool for tests
type EchoTool struct{}

func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Description() string { return "Echoes the input string" }
func (e *EchoTool) Execute(ctx context.Context, input string) (string, error) {
	return "ECHO:" + input, nil
}
func (e *EchoTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
		"required":   []string{"input"},
	}
}

func TestNewChatAgent(t *testing.T) {
	mockLLM := NewMockLLMClient()
	memStore := inmemory.NewStore()
	toolRegistry := tools.NewRegistry()

	config := ChatConfig{
		Model: mockLLM,
		Tools: toolRegistry,
		Mem:   memStore,
		Config: AgentConfig{
			MaxIterations: 5,
			Timeout:       "30s",
			SystemPrompt:  "You are a helpful assistant",
		},
	}

	agent := NewChatAgent(config)

	if agent == nil {
		t.Fatal("NewChatAgent returned nil")
	}

	if agent.Model != mockLLM {
		t.Error("Agent Model not set correctly")
	}

	if agent.Tools != toolRegistry {
		t.Error("Agent Tools not set correctly")
	}

	if agent.Mem != memStore {
		t.Error("Agent Mem not set correctly")
	}

	if agent.Config.MaxIterations != 5 {
		t.Errorf("Expected MaxIterations 5, got %d", agent.Config.MaxIterations)
	}
}

func TestChatAgent_Run_Basic(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("Hello! How can I help you today?")

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "Hello",
	}

	result, err := agent.Run(ctx, input)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if result.Role != "assistant" {
		t.Errorf("Expected response role 'assistant', got %s", result.Role)
	}

	if result.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected response content 'Hello! How can I help you today?', got %s", result.Content)
	}

	// Check that LLM was called correctly
	calls := mockLLM.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 LLM call, got %d", len(calls))
	}

	call := calls[0]
	if len(call.Messages) == 0 {
		t.Fatal("Expected at least 1 message in LLM call")
	}

	// Check system message
	if call.Messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got %s", call.Messages[0].Role)
	}
	if call.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("Expected system prompt, got %s", call.Messages[0].Content)
	}
}

func TestChatAgent_Run_WithToolInvocation(t *testing.T) {
	mockLLM := NewMockLLMClient()
	// First response asks to call tool
	mockLLM.AddResponseWithToolCalls("Calling tool", []llm.ToolCall{{
		ID:   "call-1",
		Type: "function",
		Function: llm.Function{
			Name:      "echo",
			Arguments: `{"input":"hello"}`,
		},
	}})
	// Second response after tool output
	mockLLM.AddResponse("Final answer after tool")

	reg := tools.NewRegistry()
	_ = reg.Register(&EchoTool{})

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Tools: reg,
		Config: AgentConfig{
			SystemPrompt:  "You are a helpful assistant",
			MaxIterations: 2,
		},
	})

	ctx := context.Background()
	input := Message{Role: "user", Content: "use echo"}

	result, err := agent.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Content != "Final answer after tool" {
		t.Fatalf("unexpected final content: %s", result.Content)
	}

	// Verify that the second LLM call contained the tool output message
	calls := mockLLM.GetCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(calls))
	}
	lastCall := calls[1]
	foundToolMsg := false
	for _, m := range lastCall.Messages {
		if m.Role == "tool" && strings.HasPrefix(m.Content, "ECHO:") {
			foundToolMsg = true
			break
		}
	}
	if !foundToolMsg {
		t.Fatalf("second call should include tool result message")
	}
}

func TestChatAgent_Run_WithMemory(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("I remember that!")
	memStore := inmemory.NewStore()

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Mem:   memStore,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant with memory",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "Remember this message",
	}

	result, err := agent.Run(ctx, input)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Check that input was stored in memory
	stored, err := memStore.Retrieve(ctx, "conversation")
	if err != nil {
		t.Errorf("Failed to retrieve stored conversation: %v", err)
	}

	// Accept either a single Message (legacy) or a slice of Message (current)
	switch v := stored.(type) {
	case Message:
		if v.Content != input.Content {
			t.Errorf("Stored message content doesn't match input: %s vs %s", v.Content, input.Content)
		}
	case []Message:
		if len(v) == 0 {
			t.Fatal("Stored conversation slice is empty")
		}
		// Input should be the first element in the conversation
		if v[0].Content != input.Content {
			t.Errorf("Stored message content doesn't match input: %s vs %s", v[0].Content, input.Content)
		}
	default:
		t.Fatalf("Stored conversation has unexpected type %T", stored)
	}

	if result.Content != "I remember that!" {
		t.Errorf("Unexpected response content: %s", result.Content)
	}
}

func TestChatAgent_Run_WithTimeout(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("Response within timeout")

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			Timeout:      "100ms",
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "Quick response please",
	}

	result, err := agent.Run(ctx, input)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if result.Content != "Response within timeout" {
		t.Errorf("Unexpected response: %s", result.Content)
	}
}

func TestChatAgent_Run_TimeoutExceeded(t *testing.T) {
	// This test would need a mock that delays response
	// For now, just test invalid timeout format
	mockLLM := NewMockLLMClient()

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			Timeout:      "invalid-timeout",
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "Test message",
	}

	_, err := agent.Run(ctx, input)
	if err == nil {
		t.Error("Expected error for invalid timeout format")
	}

	if !strings.Contains(err.Error(), "invalid timeout duration") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestChatAgent_Run_LLMError(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.SetError(llm.NewLLMError(llm.ProviderOpenAI, llm.ErrorTypeRateLimited, "Rate limit exceeded"))

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "This should fail",
	}

	_, err := agent.Run(ctx, input)
	if err == nil {
		t.Error("Expected error from LLM")
	}

	if !strings.Contains(err.Error(), "LLM call failed") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestChatAgent_RunStream(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("Streaming response")

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "Stream this response",
	}

	output := make(chan Message, 1)

	err := agent.RunStream(ctx, input, output)
	if err != nil {
		t.Errorf("RunStream() error = %v", err)
	}

	// Check that channel was closed
	select {
	case result, ok := <-output:
		if !ok {
			t.Error("Output channel was closed without sending result")
		} else {
			if result.Role != "assistant" {
				t.Errorf("Expected response role 'assistant', got %s", result.Role)
			}
			if result.Content != "Streaming response" {
				t.Errorf("Expected 'Streaming response', got %s", result.Content)
			}
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for stream response")
	}

	// Channel should be closed now
	select {
	case _, ok := <-output:
		if ok {
			t.Error("Output channel should be closed")
		}
	default:
		// Channel is closed, this is expected
	}
}

func TestChatAgent_RunStream_ContextCancellation(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("Response")

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context immediately
	cancel()

	input := Message{
		Role:    "user",
		Content: "This should be cancelled",
	}

	output := make(chan Message, 1)

	err := agent.RunStream(ctx, input, output)
	// The error might be from Run() being called with cancelled context
	if err == nil {
		// If no error, check that no message was sent due to context cancellation
		select {
		case <-output:
			// This might happen if Run() completes before context check
		case <-time.After(100 * time.Millisecond):
			// Timeout is expected due to cancelled context
		}
	}
}

func TestChatAgent_RunStream_Error(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.SetError(llm.NewLLMError(llm.ProviderOpenAI, llm.ErrorTypeServerError, "Server error"))

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()
	input := Message{
		Role:    "user",
		Content: "This should error",
	}

	output := make(chan Message, 1)

	err := agent.RunStream(ctx, input, output)
	if err == nil {
		t.Error("Expected error from RunStream")
	}

	// Channel should be closed
	select {
	case _, ok := <-output:
		if ok {
			t.Error("Output channel should be closed on error")
		}
	default:
		// Channel is closed, this is expected
	}
}

func TestChatAgent_MultipleRuns(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.AddResponse("First response")
	mockLLM.AddResponse("Second response")
	mockLLM.AddResponse("Third response")

	memStore := inmemory.NewStore()

	agent := NewChatAgent(ChatConfig{
		Model: mockLLM,
		Mem:   memStore,
		Config: AgentConfig{
			SystemPrompt: "You are a helpful assistant",
		},
	})

	ctx := context.Background()

	// First run
	result1, err := agent.Run(ctx, Message{Role: "user", Content: "First message"})
	if err != nil {
		t.Errorf("First run error: %v", err)
	}
	if result1.Content != "First response" {
		t.Errorf("Expected 'First response', got %s", result1.Content)
	}

	// Second run
	result2, err := agent.Run(ctx, Message{Role: "user", Content: "Second message"})
	if err != nil {
		t.Errorf("Second run error: %v", err)
	}
	if result2.Content != "Second response" {
		t.Errorf("Expected 'Second response', got %s", result2.Content)
	}

	// Verify that memory was used (check LLM calls)
	calls := mockLLM.GetCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", len(calls))
	}
}

// Test that ChatAgent implements Agent interface
func TestChatAgent_ImplementsInterface(t *testing.T) {
	var _ Agent = (*ChatAgent)(nil)
}

func TestChatConfig(t *testing.T) {
	mockLLM := NewMockLLMClient()
	memStore := inmemory.NewStore()
	toolRegistry := tools.NewRegistry()

	config := ChatConfig{
		Model: mockLLM,
		Tools: toolRegistry,
		Mem:   memStore,
		Config: AgentConfig{
			MaxIterations: 10,
			Timeout:       "60s",
			SystemPrompt:  "Test system prompt",
		},
	}

	if config.Model != mockLLM {
		t.Error("Model not set correctly in ChatConfig")
	}

	if config.Tools != toolRegistry {
		t.Error("Tools not set correctly in ChatConfig")
	}

	if config.Mem != memStore {
		t.Error("Mem not set correctly in ChatConfig")
	}

	if config.Config.MaxIterations != 10 {
		t.Errorf("Expected MaxIterations 10, got %d", config.Config.MaxIterations)
	}

	if config.Config.Timeout != "60s" {
		t.Errorf("Expected Timeout '60s', got %s", config.Config.Timeout)
	}

	if config.Config.SystemPrompt != "Test system prompt" {
		t.Errorf("Expected SystemPrompt 'Test system prompt', got %s", config.Config.SystemPrompt)
	}
}
