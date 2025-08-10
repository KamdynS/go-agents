package core

import (
	"context"
	"testing"

	"github.com/KamdynS/go-agents/llm"
)

// Streaming mock that sends partials then closes
type streamMock struct{ chunks []string }

func (m *streamMock) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.Response, error) {
	return &llm.Response{Content: "final", Model: "mock", Provider: llm.ProviderOpenAI}, nil
}
func (m *streamMock) Completion(ctx context.Context, prompt string) (*llm.Response, error) {
	return &llm.Response{Content: "c"}, nil
}
func (m *streamMock) Stream(ctx context.Context, req *llm.ChatRequest, out chan<- *llm.Response) error {
	for _, c := range m.chunks {
		out <- &llm.Response{Content: c, Model: "mock", Provider: llm.ProviderOpenAI}
	}
	close(out)
	return nil
}
func (m *streamMock) Model() string          { return "mock" }
func (m *streamMock) Provider() llm.Provider { return llm.ProviderOpenAI }
func (m *streamMock) Validate() error        { return nil }

func TestRunStream_EmitsChunksAndFinal(t *testing.T) {
	mock := &streamMock{chunks: []string{"a", "b", "c"}}
	agent := NewChatAgent(ChatConfig{Model: mock, Config: AgentConfig{SystemPrompt: "sys"}})
	out := make(chan Message, 8)
	if err := agent.RunStream(context.Background(), Message{Role: "user", Content: "x"}, out); err != nil {
		t.Fatalf("RunStream err: %v", err)
	}
	got := []string{}
	for m := range out {
		got = append(got, m.Content)
	}
	if len(got) == 0 || got[0] != "a" {
		t.Fatalf("expected streaming chunks, got %v", got)
	}
	if got[len(got)-1] != "abc" {
		t.Fatalf("expected final aggregated output 'abc', got %q", got[len(got)-1])
	}
}
