package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/KamdynS/go-agents/memory"
)

// Store implements an in-memory storage for agent memory
type Store struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	return &Store{
		data: make(map[string]interface{}),
	}
}

// Store implements memory.Store interface
func (s *Store) Store(ctx context.Context, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data[key] = value
	return nil
}

// Retrieve implements memory.Store interface
func (s *Store) Retrieve(ctx context.Context, key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	value, exists := s.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}
	
	return value, nil
}

// Delete implements memory.Store interface
func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.data, key)
	return nil
}

// List implements memory.Store interface
func (s *Store) List(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	
	return keys, nil
}

// Clear implements memory.Store interface
func (s *Store) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.data = make(map[string]interface{})
	return nil
}

// ConversationStore implements memory.ConversationStore interface
type ConversationStore struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewConversationStore creates a new in-memory conversation store
func NewConversationStore() *ConversationStore {
	return &ConversationStore{
		data: make(map[string]interface{}),
	}
}

// Store implements memory.Store interface
func (cs *ConversationStore) Store(ctx context.Context, key string, value interface{}) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.data[key] = value
	return nil
}

// Retrieve implements memory.Store interface
func (cs *ConversationStore) Retrieve(ctx context.Context, key string) (interface{}, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	value, exists := cs.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}
	
	return value, nil
}

// Delete implements memory.Store interface
func (cs *ConversationStore) Delete(ctx context.Context, key string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	delete(cs.data, key)
	return nil
}

// List implements memory.Store interface
func (cs *ConversationStore) List(ctx context.Context) ([]string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	keys := make([]string, 0, len(cs.data))
	for k := range cs.data {
		keys = append(keys, k)
	}
	
	return keys, nil
}

// Clear implements memory.Store interface
func (cs *ConversationStore) Clear(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.data = make(map[string]interface{})
	return nil
}

// AppendMessage implements memory.ConversationStore interface
func (cs *ConversationStore) AppendMessage(ctx context.Context, sessionID string, role, content string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	key := fmt.Sprintf("conversation:%s", sessionID)
	
	var messages []memory.Message
	if existing, exists := cs.data[key]; exists {
		if msgs, ok := existing.([]memory.Message); ok {
			messages = msgs
		}
	}
	
	message := memory.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
	
	messages = append(messages, message)
	cs.data[key] = messages
	
	return nil
}

// GetMessages implements memory.ConversationStore interface
func (cs *ConversationStore) GetMessages(ctx context.Context, sessionID string) ([]memory.Message, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	key := fmt.Sprintf("conversation:%s", sessionID)
	
	value, exists := cs.data[key]
	if !exists {
		return []memory.Message{}, nil
	}
	
	if messages, ok := value.([]memory.Message); ok {
		return messages, nil
	}
	
	return nil, fmt.Errorf("invalid message format for session %s", sessionID)
}

// ClearSession implements memory.ConversationStore interface
func (cs *ConversationStore) ClearSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("conversation:%s", sessionID)
	return cs.Delete(ctx, key)
}

// Ensure implementations satisfy interfaces
var _ memory.Store = (*Store)(nil)
var _ memory.ConversationStore = (*ConversationStore)(nil)