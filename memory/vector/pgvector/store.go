//go:build adapters_pgvector

package pgvector

import (
	"context"
	"errors"
	"fmt"

	"github.com/KamdynS/go-agents/memory"
	"github.com/jackc/pgx/v5"
)

type Store struct {
	pool  *pgx.Conn
	table string
}

// Expect table schema similar to:
// CREATE EXTENSION IF NOT EXISTS vector;
// CREATE TABLE IF NOT EXISTS documents (
//   id text PRIMARY KEY,
//   content text NOT NULL,
//   embedding vector(1536),
//   meta jsonb
// );

func New(conn *pgx.Conn, table string) *Store {
	if table == "" {
		table = "documents"
	}
	return &Store{pool: conn, table: table}
}

func (s *Store) AddDocument(ctx context.Context, id string, content string, embedding []float64) error {
	if len(embedding) == 0 {
		return errors.New("empty embedding")
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id, content, embedding) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET content=excluded.content, embedding=excluded.embedding", s.table), id, content, embedding)
	return err
}

func (s *Store) QuerySimilar(ctx context.Context, queryEmbedding []float64, limit int) ([]memory.Document, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf("SELECT id, content, embedding <#> $1 AS score FROM %s ORDER BY embedding <#> $1 ASC LIMIT $2", s.table), queryEmbedding, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]memory.Document, 0, limit)
	for rows.Next() {
		var id string
		var content string
		var score float64
		if err := rows.Scan(&id, &content, &score); err != nil {
			return nil, err
		}
		out = append(out, memory.Document{ID: id, Content: content, Score: score})
	}
	return out, rows.Err()
}

func (s *Store) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE id=$1", s.table), id)
	return err
}

func (s *Store) GetDocument(ctx context.Context, id string) (*memory.Document, error) {
	row := s.pool.QueryRow(ctx, fmt.Sprintf("SELECT id, content FROM %s WHERE id=$1", s.table), id)
	var doc memory.Document
	if err := row.Scan(&doc.ID, &doc.Content); err != nil {
		return nil, err
	}
	return &doc, nil
}

var _ memory.VectorStore = (*Store)(nil)
