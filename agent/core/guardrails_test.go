package core

import (
	"context"
	"testing"

	"github.com/KamdynS/go-agents/llm"
)

func TestSimpleGuardrails(t *testing.T) {
	g := &SimpleGuardrails{MaxInputChars: 5, DenySubstrings: []string{"bad"}}
	// Not denied (no bad), but will be trimmed when over limit
	req := &llm.ChatRequest{Messages: []llm.Message{{Role: "user", Content: "hello"}}}
	if err := g.BeforeLLMCall(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req = &llm.ChatRequest{Messages: []llm.Message{{Role: "user", Content: "toolong"}}}
	_ = g.BeforeLLMCall(context.Background(), req)
	if got := req.Messages[0].Content; got != "toolo" {
		t.Fatalf("expected trimmed, got %q", got)
	}
	// Allowlist blocks when configured and no allowed substring present
	g.AllowSubstrings = []string{"ok"}
	req = &llm.ChatRequest{Messages: []llm.Message{{Role: "user", Content: "fine"}}}
	if err := g.BeforeLLMCall(context.Background(), req); err == nil {
		t.Fatalf("expected allowlist block")
	}
	// But allows when allowed substring appears
	req = &llm.ChatRequest{Messages: []llm.Message{{Role: "user", Content: "ok content"}}}
	if err := g.BeforeLLMCall(context.Background(), req); err != nil {
		t.Fatalf("expected allowlist pass, got %v", err)
	}
	// Deny substring blocks
	req = &llm.ChatRequest{Messages: []llm.Message{{Role: "user", Content: "very bad thing"}}}
	if err := g.BeforeLLMCall(context.Background(), req); err == nil {
		t.Fatalf("expected deny error")
	}
}
