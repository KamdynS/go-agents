package memory

import "context"

// Store defines the interface for agent memory/state management
type Store interface {
	// Store saves data with the given key
	Store(ctx context.Context, key string, value interface{}) error
	
	// Retrieve gets data by key
	Retrieve(ctx context.Context, key string) (interface{}, error)
	
	// Delete removes data by key
	Delete(ctx context.Context, key string) error
	
	// List returns all keys
	List(ctx context.Context) ([]string, error)
	
	// Clear removes all stored data
	Clear(ctx context.Context) error
}

// ConversationStore is a specialized interface for managing conversation history
type ConversationStore interface {
	Store
	
	// AppendMessage adds a message to the conversation
	AppendMessage(ctx context.Context, sessionID string, role, content string) error
	
	// GetMessages retrieves conversation history
	GetMessages(ctx context.Context, sessionID string) ([]Message, error)
	
	// ClearSession removes all messages for a session
	ClearSession(ctx context.Context, sessionID string) error
}

// Message represents a conversation message
type Message struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	Timestamp int64             `json:"timestamp"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// VectorStore defines the interface for vector-based retrieval (RAG)
type VectorStore interface {
	// AddDocument adds a document with its vector embedding
	AddDocument(ctx context.Context, id string, content string, embedding []float64) error
	
	// QuerySimilar finds similar documents based on query embedding
	QuerySimilar(ctx context.Context, queryEmbedding []float64, limit int) ([]Document, error)
	
	// DeleteDocument removes a document by ID
	DeleteDocument(ctx context.Context, id string) error
	
	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, id string) (*Document, error)
}

// Document represents a stored document with its metadata
type Document struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Embedding []float64         `json:"embedding"`
	Meta      map[string]string `json:"meta,omitempty"`
	Score     float64           `json:"score,omitempty"` // Similarity score for query results
}