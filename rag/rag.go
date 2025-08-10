package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/KamdynS/go-agents/llm/openai"
	"github.com/KamdynS/go-agents/memory"
)

// Chunk splits text into roughly fixed-size chunks by rune count with simple paragraph awareness.
func Chunk(text string, approxChunkSize int) []string {
	if approxChunkSize <= 0 {
		approxChunkSize = 1200
	}
	paras := strings.Split(text, "\n\n")
	var chunks []string
	var cur strings.Builder
	for _, p := range paras {
		if cur.Len()+len(p) > approxChunkSize && cur.Len() > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
		}
		if len(p) > approxChunkSize {
			// Hard split long paragraph
			for i := 0; i < len(p); i += approxChunkSize {
				end := i + approxChunkSize
				if end > len(p) {
					end = len(p)
				}
				if cur.Len() > 0 {
					chunks = append(chunks, cur.String())
					cur.Reset()
				}
				chunks = append(chunks, p[i:end])
			}
			continue
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(p)
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}

// Embedder provides text embeddings.
type Embedder interface {
	EmbedText(ctx context.Context, input string) ([]float64, error)
}

// OpenAIEmbedder implements Embedder using OpenAI embeddings API.
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
}

// NewOpenAIEmbedder creates a new embedder.
func NewOpenAIEmbedder(cfg openai.Config, model string) *OpenAIEmbedder {
	c, _ := openai.NewClient(cfg)
	return &OpenAIEmbedder{client: c, model: model}
}

// EmbedText returns a single vector for the input text.
func (e *OpenAIEmbedder) EmbedText(ctx context.Context, input string) ([]float64, error) {
	model := e.model
	if model == "" || strings.Contains(model, "gpt") {
		model = "text-embedding-3-small"
	}
	return e.client.Embed(ctx, input, model)
}

// IndexDocuments chunks, embeds and upserts content into a VectorStore.
func IndexDocuments(ctx context.Context, store memory.VectorStore, emb Embedder, docs map[string]string) error {
	for id, content := range docs {
		for i, ch := range Chunk(content, 1200) {
			cid := fmt.Sprintf("%s#%d", id, i)
			vec, err := emb.EmbedText(ctx, ch)
			if err != nil {
				return fmt.Errorf("embed %s: %w", cid, err)
			}
			if err := store.AddDocument(ctx, cid, ch, vec); err != nil {
				return fmt.Errorf("upsert %s: %w", cid, err)
			}
		}
	}
	return nil
}

// Query retrieves topK documents by embedding similarity for the question.
func Query(ctx context.Context, store memory.VectorStore, emb Embedder, question string, topK int) ([]memory.Document, error) {
	if topK <= 0 {
		topK = 5
	}
	qvec, err := emb.EmbedText(ctx, question)
	if err != nil {
		return nil, err
	}
	return store.QuerySimilar(ctx, qvec, topK)
}

// BuildContext formats retrieved docs into a context string for prompts.
func BuildContext(docs []memory.Document) string {
	var b strings.Builder
	for i, d := range docs {
		fmt.Fprintf(&b, "[D%d]\n%s\n\n", i+1, strings.TrimSpace(d.Content))
	}
	return b.String()
}
