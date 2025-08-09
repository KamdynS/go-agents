package core

import (
	"context"
	"testing"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/tools"
)

// LLM that triggers a tool call then returns a final
type toolCallLLM struct{ called bool }

func (m *toolCallLLM) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.Response, error) {
	if !m.called {
		m.called = true
		return &llm.Response{
			Content:   "call tool",
			Model:     "mock",
			Provider:  llm.ProviderOpenAI,
			ToolCalls: []llm.ToolCall{{ID: "1", Type: "function", Function: llm.Function{Name: "echo", Arguments: `{"input":"ok"}`}}},
		}, nil
	}
	return &llm.Response{Content: "done", Model: "mock", Provider: llm.ProviderOpenAI}, nil
}
func (m *toolCallLLM) Completion(ctx context.Context, prompt string) (*llm.Response, error) {
	return &llm.Response{Content: "c"}, nil
}
func (m *toolCallLLM) Stream(ctx context.Context, req *llm.ChatRequest, out chan<- *llm.Response) error {
	return nil
}
func (m *toolCallLLM) Model() string          { return "mock" }
func (m *toolCallLLM) Provider() llm.Provider { return llm.ProviderOpenAI }
func (m *toolCallLLM) Validate() error        { return nil }

type echoTool struct{}

func (echoTool) Name() string                                              { return "echo" }
func (echoTool) Description() string                                       { return "echo" }
func (echoTool) Execute(ctx context.Context, input string) (string, error) { return "E:" + input, nil }
func (echoTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}}, "required": []string{"input"}}
}

func TestToolInvocation_Path(t *testing.T) {
	mock := &toolCallLLM{}
	reg := tools.NewRegistry()
	_ = reg.Register(echoTool{})
	agent := NewChatAgent(ChatConfig{Model: mock, Tools: reg, Config: AgentConfig{SystemPrompt: "sys", MaxIterations: 2}})
	out, err := agent.Run(context.Background(), Message{Role: "user", Content: "hi"})
	if err != nil {
		t.Fatalf("run err: %v", err)
	}
	if out.Content != "done" {
		t.Fatalf("unexpected final: %q", out.Content)
	}
}
