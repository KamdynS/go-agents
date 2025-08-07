package vector_test

import (
	"context"
	"testing"

	mem "github.com/KamdynS/go-agents/memory"
)

type vectorFactory func(t *testing.T) mem.VectorStore

func runVectorContract(t *testing.T, makeStore vectorFactory) {
	t.Helper()
	ctx := context.Background()
	s := makeStore(t)

	// Basic add and get
	if err := s.AddDocument(ctx, "d1", "content one", []float64{0.1, 0.2, 0.3}); err != nil {
		t.Fatalf("add: %v", err)
	}
	doc, err := s.GetDocument(ctx, "d1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if doc.ID != "d1" {
		t.Fatalf("want d1 got %s", doc.ID)
	}

	// Similarity query should not error (scores may vary by backend)
	_, err = s.QuerySimilar(ctx, []float64{0.1, 0.2, 0.3}, 3)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	// Delete
	if err := s.DeleteDocument(ctx, "d1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// Note: no default in-memory VectorStore impl, so this contract will be exercised
// by adapter-specific test files with build tags.
