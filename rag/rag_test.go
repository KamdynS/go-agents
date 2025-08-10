package rag

import (
	"context"
	"testing"

	"github.com/KamdynS/go-agents/memory"
)

type fakeEmb struct {
	vec []float64
	err error
}

func (f fakeEmb) EmbedText(ctx context.Context, input string) ([]float64, error) { return f.vec, f.err }

type fakeVS struct{ docs []memory.Document }

func (f *fakeVS) AddDocument(ctx context.Context, id, content string, vector []float64) error {
	f.docs = append(f.docs, memory.Document{ID: id, Content: content, Embedding: vector})
	return nil
}
func (f *fakeVS) QuerySimilar(ctx context.Context, vector []float64, topK int) ([]memory.Document, error) {
	if len(f.docs) == 0 {
		return nil, nil
	}
	if topK <= 0 || topK > len(f.docs) {
		topK = len(f.docs)
	}
	return f.docs[:topK], nil
}
func (f *fakeVS) DeleteDocument(ctx context.Context, id string) error { return nil }
func (f *fakeVS) GetDocument(ctx context.Context, id string) (*memory.Document, error) {
	return nil, nil
}

func TestChunkBasic(t *testing.T) {
	chunks := Chunk("a\n\nbbb\n\ncccc", 3)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks")
	}
}

func TestIndexQueryBuildContext(t *testing.T) {
	vs := &fakeVS{}
	emb := fakeEmb{vec: []float64{0.1, 0.2}}
	docs := map[string]string{"doc1": "para1\n\npara2"}
	if err := IndexDocuments(context.Background(), vs, emb, docs); err != nil {
		t.Fatalf("index: %v", err)
	}
	got, err := Query(context.Background(), vs, emb, "q", 1)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	ctx := BuildContext(got)
	if ctx == "" {
		t.Fatalf("empty context")
	}
}
