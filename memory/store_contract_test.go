package memory_test

import (
	"context"
	"testing"

	mem "github.com/KamdynS/go-agents/memory"
	inm "github.com/KamdynS/go-agents/memory/inmemory"
)

type storeFactory func(t *testing.T) mem.Store
type convFactory func(t *testing.T) mem.ConversationStore

func runStoreContract(t *testing.T, makeStore storeFactory) {
	t.Helper()
	ctx := context.Background()
	s := makeStore(t)

	// Store/Retrieve
	if err := s.Store(ctx, "k1", "v1"); err != nil {
		t.Fatalf("store: %v", err)
	}
	v, err := s.Retrieve(ctx, "k1")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if v.(string) != "v1" {
		t.Fatalf("want v1 got %v", v)
	}

	// List
	keys, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("list: empty")
	}

	// Delete + not found
	if err := s.Delete(ctx, "k1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Retrieve(ctx, "k1"); err == nil {
		t.Fatalf("expected error for missing key")
	}

	// Clear
	_ = s.Store(ctx, "k2", 123)
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	keys, _ = s.List(ctx)
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys after clear, got %d", len(keys))
	}
}

func runConversationContract(t *testing.T, makeConv convFactory) {
	t.Helper()
	ctx := context.Background()
	cs := makeConv(t)

	session := "s1"
	if err := cs.AppendMessage(ctx, session, "user", "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := cs.AppendMessage(ctx, session, "assistant", "hi"); err != nil {
		t.Fatalf("append: %v", err)
	}

	msgs, err := cs.GetMessages(ctx, session)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("unexpected roles: %+v", msgs)
	}

	if err := cs.ClearSession(ctx, session); err != nil {
		t.Fatalf("clear session: %v", err)
	}
	msgs, err = cs.GetMessages(ctx, session)
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 after clear, got %d", len(msgs))
	}
}

func TestStoreContract_InMemory(t *testing.T) {
	runStoreContract(t, func(t *testing.T) mem.Store { return inm.NewStore() })
}

func TestConversationContract_InMemory(t *testing.T) {
	runConversationContract(t, func(t *testing.T) mem.ConversationStore { return inm.NewConversationStore() })
}
