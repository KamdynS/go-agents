package core

import (
	"context"
	"errors"
	"testing"

	"github.com/KamdynS/go-agents/llm"
)

type countingMW struct {
	beforeLLM, afterLLM, beforeTool, afterTool, afterRun int
}

func (m *countingMW) BeforeLLMCall(ctx context.Context, req *llm.ChatRequest) error {
	m.beforeLLM++
	return nil
}
func (m *countingMW) AfterLLMResponse(ctx context.Context, resp *llm.Response) error {
	m.afterLLM++
	return nil
}
func (m *countingMW) BeforeToolExecute(ctx context.Context, toolName string, input string) error {
	m.beforeTool++
	return nil
}
func (m *countingMW) AfterToolExecute(ctx context.Context, toolName string, result string, execErr error) error {
	m.afterTool++
	return nil
}
func (m *countingMW) AfterRun(ctx context.Context, final Message) error { m.afterRun++; return nil }

func TestGuardrails_BlocksInput(t *testing.T) {
	gr := &SimpleGuardrails{DenySubstrings: []string{"blocked"}}
	mock := NewMockLLMClient()
	agent := NewChatAgent(ChatConfig{
		Model:      mock,
		Config:     AgentConfig{SystemPrompt: "sys"},
		Middleware: []Middleware{gr},
	})
	_, err := agent.Run(context.Background(), Message{Role: "user", Content: "this is blocked content"})
	if err == nil {
		t.Fatalf("expected guardrails to block request")
	}
}

func TestMiddleware_Hooks_Invoked(t *testing.T) {
	mw := &countingMW{}
	mock := NewMockLLMClient()
	// ask for a simple run
	mock.AddResponse("ok")
	agent := NewChatAgent(ChatConfig{
		Model:      mock,
		Config:     AgentConfig{SystemPrompt: "sys"},
		Middleware: []Middleware{mw},
	})
	_, err := agent.Run(context.Background(), Message{Role: "user", Content: "hi"})
	if err != nil {
		t.Fatalf("run err: %v", err)
	}
	if mw.beforeLLM == 0 || mw.afterLLM == 0 || mw.afterRun == 0 {
		t.Fatalf("expected middleware hooks to be called: %+v", *mw)
	}
}

// Erroring MW to verify propagation
type errorMW struct{}

func (errorMW) BeforeLLMCall(ctx context.Context, req *llm.ChatRequest) error {
	return errors.New("nope")
}
func (errorMW) AfterLLMResponse(ctx context.Context, resp *llm.Response) error { return nil }
func (errorMW) BeforeToolExecute(ctx context.Context, toolName string, input string) error {
	return nil
}
func (errorMW) AfterToolExecute(ctx context.Context, toolName string, result string, execErr error) error {
	return nil
}
func (errorMW) AfterRun(ctx context.Context, final Message) error { return nil }

func TestMiddleware_Error_Propagates(t *testing.T) {
	mock := NewMockLLMClient()
	agent := NewChatAgent(ChatConfig{
		Model:      mock,
		Config:     AgentConfig{SystemPrompt: "sys"},
		Middleware: []Middleware{errorMW{}},
	})
	_, err := agent.Run(context.Background(), Message{Role: "user", Content: "hi"})
	if err == nil {
		t.Fatalf("expected error from middleware")
	}
}
