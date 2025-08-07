//go:build adapters_redis

package redis

import (
	"context"
	"testing"
	"time"

	mem "github.com/KamdynS/go-agents/memory"
	rds "github.com/redis/go-redis/v9"
)

func makeRedisStore(t *testing.T) mem.Store {
	t.Helper()
	client := rds.NewClient(&rds.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })
	return NewStore(client, time.Minute, "test")
}

func makeRedisConv(t *testing.T) mem.ConversationStore {
	t.Helper()
	client := rds.NewClient(&rds.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })
	return NewConversationStore(client, "test", time.Minute)
}

func TestStoreContract_Redis(t *testing.T) {
	// Reuse core contract via simple inline copy to avoid import cycles
	ctx := context.Background()
	s := makeRedisStore(t)
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
}

func TestConversationContract_Redis(t *testing.T) {
	ctx := context.Background()
	cs := makeRedisConv(t)
	if err := cs.AppendMessage(ctx, "s1", "user", "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}
	msgs, err := cs.GetMessages(ctx, "s1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected messages")
	}
}
