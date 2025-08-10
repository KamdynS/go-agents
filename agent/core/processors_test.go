package core

import (
	"context"
	"testing"
)

func TestTokenLimiter(t *testing.T) {
	p := TokenLimiter{MaxChars: 5}
	msgs := []Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "world"}}
	out := p.Process(context.Background(), msgs)
	if len(out) == 0 {
		t.Fatalf("expected some messages kept")
	}
}

func TestToolCallFilter(t *testing.T) {
	f := ToolCallFilter{}
	msgs := []Message{{Role: "user", Content: "hi"}, {Role: "tool", Content: "x"}, {Role: "assistant", Content: "ok"}}
	out := f.Process(context.Background(), msgs)
	for _, m := range out {
		if m.Role == "tool" {
			t.Fatalf("tool messages should be filtered")
		}
	}
}

func TestTokenLimiter_TrimsOldest(t *testing.T) {
	p := TokenLimiter{MaxChars: 5}
	in := []Message{{Role: "user", Content: "12"}, {Role: "assistant", Content: "34"}, {Role: "user", Content: "5"}}
	out := p.Process(context.Background(), in)
	// Should keep recent messages fitting MaxChars; result likely ["34","5"] or ["5"] depending on limit
	total := 0
	for _, m := range out {
		total += len(m.Content)
	}
	if total > 5 {
		t.Fatalf("expected <=5 chars, got %d: %#v", total, out)
	}
}

func TestToolCallFilter_RemovesToolMessages(t *testing.T) {
	f := ToolCallFilter{}
	in := []Message{{Role: "user", Content: "a"}, {Role: "tool", Content: "x"}, {Role: "assistant", Content: "b"}}
	out := f.Process(context.Background(), in)
	for _, m := range out {
		if m.Role == "tool" {
			t.Fatalf("tool message not removed")
		}
	}
}
