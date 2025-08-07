//go:build adapters_pgvector

package pgvector

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestVectorContract_PgVector(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("connect: %v", err)
	}
	defer conn.Close(ctx)

	s := New(conn, "documents")
	if err := s.AddDocument(ctx, "d1", "hello", []float64{0.1, 0.2}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := s.GetDocument(ctx, "d1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.QuerySimilar(ctx, []float64{0.1, 0.2}, 3); err != nil {
		t.Fatalf("query: %v", err)
	}
}
