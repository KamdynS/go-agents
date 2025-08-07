package memory

import (
	"testing"
)

func TestMessage(t *testing.T) {
	msg := Message{
		Role:      "user",
		Content:   "Hello, world!",
		Timestamp: 1234567890,
		Meta: map[string]string{
			"source": "test",
		},
	}

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", msg.Content)
	}

	if msg.Timestamp != 1234567890 {
		t.Errorf("Expected timestamp 1234567890, got %d", msg.Timestamp)
	}

	if msg.Meta["source"] != "test" {
		t.Errorf("Expected meta source 'test', got %s", msg.Meta["source"])
	}
}

func TestDocument(t *testing.T) {
	embedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	doc := Document{
		ID:        "doc1",
		Content:   "This is a test document",
		Embedding: embedding,
		Meta: map[string]string{
			"category": "test",
		},
		Score: 0.95,
	}

	if doc.ID != "doc1" {
		t.Errorf("Expected ID 'doc1', got %s", doc.ID)
	}

	if doc.Content != "This is a test document" {
		t.Errorf("Expected content 'This is a test document', got %s", doc.Content)
	}

	if len(doc.Embedding) != len(embedding) {
		t.Errorf("Expected embedding length %d, got %d", len(embedding), len(doc.Embedding))
	}

	for i, v := range embedding {
		if doc.Embedding[i] != v {
			t.Errorf("Expected embedding[%d] = %f, got %f", i, v, doc.Embedding[i])
		}
	}

	if doc.Meta["category"] != "test" {
		t.Errorf("Expected meta category 'test', got %s", doc.Meta["category"])
	}

	if doc.Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", doc.Score)
	}
}