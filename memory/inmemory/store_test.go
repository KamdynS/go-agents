package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/KamdynS/go-agents/memory"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	if store.data == nil {
		t.Error("Store data map should be initialized")
	}
}

func TestStore_Store(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string value", "test-key", "test-value"},
		{"int value", "int-key", 42},
		{"map value", "map-key", map[string]string{"nested": "value"}},
		{"struct value", "struct-key", struct{ Name string }{"test"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := store.Store(ctx, test.key, test.value)
			if err != nil {
				t.Errorf("Store() error = %v", err)
			}

			// Verify the value was stored
			if stored, exists := store.data[test.key]; !exists {
				t.Error("Value was not stored")
			} else {
				// Use deep comparison for complex types like maps
				switch v := test.value.(type) {
				case map[string]string:
					if storedMap, ok := stored.(map[string]string); ok {
						if len(storedMap) != len(v) {
							t.Errorf("Stored map length %d != expected %d", len(storedMap), len(v))
						} else {
							for key, val := range v {
								if storedMap[key] != val {
									t.Errorf("Stored map[%s] = %s != expected %s", key, storedMap[key], val)
								}
							}
						}
					} else {
						t.Error("Stored value is not a map[string]string")
					}
				default:
					if stored != test.value {
						t.Errorf("Stored value %v != expected %v", stored, test.value)
					}
				}
			}
		})
	}
}

func TestStore_Retrieve(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Store a test value
	testKey := "test-key"
	testValue := "test-value"
	store.data[testKey] = testValue

	// Test retrieving existing value
	retrieved, err := store.Retrieve(ctx, testKey)
	if err != nil {
		t.Errorf("Retrieve() error = %v", err)
	}
	if retrieved != testValue {
		t.Errorf("Retrieved value %v != expected %v", retrieved, testValue)
	}

	// Test retrieving non-existent value
	_, err = store.Retrieve(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Store a test value
	testKey := "test-key"
	testValue := "test-value"
	store.data[testKey] = testValue

	// Delete the value
	err := store.Delete(ctx, testKey)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify the value was deleted
	if _, exists := store.data[testKey]; exists {
		t.Error("Value was not deleted")
	}

	// Delete non-existent key should not error
	err = store.Delete(ctx, "non-existent")
	if err != nil {
		t.Errorf("Delete() error for non-existent key = %v", err)
	}
}

func TestStore_List(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Test empty store
	keys, err := store.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys in empty store, got %d", len(keys))
	}

	// Add some test data
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		store.data[k] = v
	}

	// Test listing keys
	keys, err = store.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(keys) != len(testData) {
		t.Errorf("Expected %d keys, got %d", len(testData), len(keys))
	}

	// Check that all expected keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for expectedKey := range testData {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %s not found in list", expectedKey)
		}
	}
}

func TestStore_Clear(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Add some test data
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		store.data[k] = v
	}

	// Clear the store
	err := store.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// Verify the store is empty
	if len(store.data) != 0 {
		t.Errorf("Expected empty store after Clear(), got %d items", len(store.data))
	}

	// Test that List returns empty result
	keys, err := store.List(ctx)
	if err != nil {
		t.Errorf("List() error after Clear() = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after Clear(), got %d", len(keys))
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	
	// Test concurrent reads and writes
	done := make(chan bool)
	errors := make(chan error, 10)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := "key" + string(rune(i))
			value := "value" + string(rune(i))
			if err := store.Store(ctx, key, value); err != nil {
				errors <- err
				return
			}
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			if _, err := store.List(ctx); err != nil {
				errors <- err
				return
			}
			time.Sleep(time.Microsecond) // Small delay to create race conditions
		}
		done <- true
	}()

	// Wait for both goroutines
	completedCount := 0
	for completedCount < 2 {
		select {
		case err := <-errors:
			t.Errorf("Concurrent access error: %v", err)
		case <-done:
			completedCount++
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}
}

// Test that Store implements memory.Store interface
func TestStore_ImplementsInterface(t *testing.T) {
	var _ memory.Store = (*Store)(nil)
}

func TestNewConversationStore(t *testing.T) {
	store := NewConversationStore()
	if store == nil {
		t.Fatal("NewConversationStore() returned nil")
	}

	if store.data == nil {
		t.Error("ConversationStore data map should be initialized")
	}
}

func TestConversationStore_AppendMessage(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	sessionID := "test-session"

	// Test appending first message
	err := store.AppendMessage(ctx, sessionID, "user", "Hello")
	if err != nil {
		t.Errorf("AppendMessage() error = %v", err)
	}

	// Test appending second message
	err = store.AppendMessage(ctx, sessionID, "assistant", "Hi there!")
	if err != nil {
		t.Errorf("AppendMessage() error = %v", err)
	}

	// Verify messages were stored
	key := "conversation:" + sessionID
	value, exists := store.data[key]
	if !exists {
		t.Fatal("Messages were not stored")
	}

	messages, ok := value.([]memory.Message)
	if !ok {
		t.Fatal("Stored value is not []memory.Message")
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Check first message
	if messages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got %s", messages[0].Role)
	}
	if messages[0].Content != "Hello" {
		t.Errorf("Expected first message content 'Hello', got %s", messages[0].Content)
	}

	// Check second message
	if messages[1].Role != "assistant" {
		t.Errorf("Expected second message role 'assistant', got %s", messages[1].Role)
	}
	if messages[1].Content != "Hi there!" {
		t.Errorf("Expected second message content 'Hi there!', got %s", messages[1].Content)
	}

	// Check timestamps
	if messages[0].Timestamp <= 0 {
		t.Error("First message should have valid timestamp")
	}
	if messages[1].Timestamp <= 0 {
		t.Error("Second message should have valid timestamp")
	}
	if messages[1].Timestamp < messages[0].Timestamp {
		t.Error("Second message timestamp should be after first message")
	}
}

func TestConversationStore_GetMessages(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	sessionID := "test-session"

	// Test getting messages from empty session
	messages, err := store.GetMessages(ctx, sessionID)
	if err != nil {
		t.Errorf("GetMessages() error for empty session = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for empty session, got %d", len(messages))
	}

	// Add some messages
	err = store.AppendMessage(ctx, sessionID, "user", "Hello")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	err = store.AppendMessage(ctx, sessionID, "assistant", "Hi!")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	// Test getting messages
	messages, err = store.GetMessages(ctx, sessionID)
	if err != nil {
		t.Errorf("GetMessages() error = %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("First message incorrect: role=%s, content=%s", messages[0].Role, messages[0].Content)
	}

	if messages[1].Role != "assistant" || messages[1].Content != "Hi!" {
		t.Errorf("Second message incorrect: role=%s, content=%s", messages[1].Role, messages[1].Content)
	}
}

func TestConversationStore_ClearSession(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	sessionID := "test-session"

	// Add some messages
	err := store.AppendMessage(ctx, sessionID, "user", "Hello")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	// Verify messages exist
	messages, err := store.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message before clear, got %d", len(messages))
	}

	// Clear the session
	err = store.ClearSession(ctx, sessionID)
	if err != nil {
		t.Errorf("ClearSession() error = %v", err)
	}

	// Verify messages were cleared
	messages, err = store.GetMessages(ctx, sessionID)
	if err != nil {
		t.Errorf("GetMessages() error after clear = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

func TestConversationStore_MultipleSessions(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	
	session1 := "session-1"
	session2 := "session-2"

	// Add messages to different sessions
	err := store.AppendMessage(ctx, session1, "user", "Hello from session 1")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	err = store.AppendMessage(ctx, session2, "user", "Hello from session 2")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	err = store.AppendMessage(ctx, session1, "assistant", "Hi from session 1")
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	// Verify session 1 has 2 messages
	messages1, err := store.GetMessages(ctx, session1)
	if err != nil {
		t.Errorf("GetMessages() error for session 1 = %v", err)
	}
	if len(messages1) != 2 {
		t.Errorf("Expected 2 messages in session 1, got %d", len(messages1))
	}

	// Verify session 2 has 1 message
	messages2, err := store.GetMessages(ctx, session2)
	if err != nil {
		t.Errorf("GetMessages() error for session 2 = %v", err)
	}
	if len(messages2) != 1 {
		t.Errorf("Expected 1 message in session 2, got %d", len(messages2))
	}

	// Verify correct content
	if messages1[0].Content != "Hello from session 1" {
		t.Errorf("Incorrect content in session 1: %s", messages1[0].Content)
	}
	if messages2[0].Content != "Hello from session 2" {
		t.Errorf("Incorrect content in session 2: %s", messages2[0].Content)
	}

	// Clear session 1
	err = store.ClearSession(ctx, session1)
	if err != nil {
		t.Errorf("ClearSession() error = %v", err)
	}

	// Verify session 1 is empty but session 2 is not affected
	messages1, err = store.GetMessages(ctx, session1)
	if err != nil {
		t.Errorf("GetMessages() error after clear = %v", err)
	}
	if len(messages1) != 0 {
		t.Errorf("Expected 0 messages in session 1 after clear, got %d", len(messages1))
	}

	messages2, err = store.GetMessages(ctx, session2)
	if err != nil {
		t.Errorf("GetMessages() error for session 2 after clear = %v", err)
	}
	if len(messages2) != 1 {
		t.Errorf("Expected 1 message in session 2 after clear, got %d", len(messages2))
	}
}

func TestConversationStore_InvalidData(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	sessionID := "test-session"

	// Store invalid data manually
	key := "conversation:" + sessionID
	store.data[key] = "invalid-data-not-messages"

	// Test GetMessages with invalid data
	_, err := store.GetMessages(ctx, sessionID)
	if err == nil {
		t.Error("Expected error for invalid message data")
	}
}

// Test that ConversationStore implements memory.ConversationStore interface
func TestConversationStore_ImplementsInterface(t *testing.T) {
	var _ memory.ConversationStore = (*ConversationStore)(nil)
	var _ memory.Store = (*ConversationStore)(nil)
}

func TestConversationStore_ConcurrentAccess(t *testing.T) {
	store := NewConversationStore()
	ctx := context.Background()
	sessionID := "test-session"
	
	// Test concurrent message appending
	done := make(chan bool)
	errors := make(chan error, 10)

	// Multiple writers
	for i := 0; i < 3; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				role := "user"
				if j%2 == 1 {
					role = "assistant"
				}
				content := "Message from goroutine " + string(rune(id)) + " iteration " + string(rune(j))
				if err := store.AppendMessage(ctx, sessionID, role, content); err != nil {
					errors <- err
					return
				}
			}
			done <- true
		}(i)
	}

	// Reader
	go func() {
		for i := 0; i < 30; i++ {
			if _, err := store.GetMessages(ctx, sessionID); err != nil {
				errors <- err
				return
			}
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	completedCount := 0
	for completedCount < 4 {
		select {
		case err := <-errors:
			t.Errorf("Concurrent access error: %v", err)
		case <-done:
			completedCount++
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify final message count (should be 30 messages from 3 goroutines * 10 iterations each)
	messages, err := store.GetMessages(ctx, sessionID)
	if err != nil {
		t.Errorf("Final GetMessages() error = %v", err)
	}
	if len(messages) != 30 {
		t.Errorf("Expected 30 messages after concurrent writes, got %d", len(messages))
	}
}