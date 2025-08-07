//go:build adapters_redis

package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/KamdynS/go-agents/memory"
	rds "github.com/redis/go-redis/v9"
)

type Store struct {
	client *rds.Client
	ttl    time.Duration
	prefix string
}

func NewStore(client *rds.Client, ttl time.Duration, prefix string) *Store {
	return &Store{client: client, ttl: ttl, prefix: prefix}
}

func (s *Store) key(k string) string {
	if s.prefix == "" {
		return k
	}
	return s.prefix + ":" + k
}

func (s *Store) Store(ctx context.Context, key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(key), b, s.ttl).Err()
}

func (s *Store) Retrieve(ctx context.Context, key string) (interface{}, error) {
	val, err := s.client.Get(ctx, s.key(key)).Bytes()
	if err != nil {
		if errors.Is(err, rds.Nil) {
			return nil, fmt.Errorf("key %s not found", key)
		}
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(val, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.key(key)).Err()
}

func (s *Store) List(ctx context.Context) ([]string, error) {
	// For simplicity, scan all keys with prefix
	var cursor uint64
	keys := []string{}
	pattern := s.prefix + ":*"
	if s.prefix == "" {
		pattern = "*"
	}
	for {
		ks, cur, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range ks {
			keys = append(keys, k)
		}
		if cur == 0 {
			break
		}
		cursor = cur
	}
	return keys, nil
}

func (s *Store) Clear(ctx context.Context) error {
	keys, err := s.List(ctx)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.client.Del(ctx, keys...).Err()
}

var _ memory.Store = (*Store)(nil)

type ConversationStore struct {
	client *rds.Client
	prefix string
	ttl    time.Duration
}

func NewConversationStore(client *rds.Client, prefix string, ttl time.Duration) *ConversationStore {
	return &ConversationStore{client: client, prefix: prefix, ttl: ttl}
}

func (cs *ConversationStore) convKey(sessionID string) string {
	p := cs.prefix
	if p != "" {
		p += ":"
	}
	return fmt.Sprintf("%sconversation:%s", p, sessionID)
}

func (cs *ConversationStore) Store(ctx context.Context, key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cs.client.Set(ctx, cs.convKey(key), b, cs.ttl).Err()
}

func (cs *ConversationStore) Retrieve(ctx context.Context, key string) (interface{}, error) {
	val, err := cs.client.Get(ctx, cs.convKey(key)).Bytes()
	if err != nil {
		if errors.Is(err, rds.Nil) {
			return nil, fmt.Errorf("key %s not found", key)
		}
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(val, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (cs *ConversationStore) Delete(ctx context.Context, key string) error {
	return cs.client.Del(ctx, cs.convKey(key)).Err()
}

func (cs *ConversationStore) List(ctx context.Context) ([]string, error) {
	pattern := cs.convKey("*")
	var cursor uint64
	keys := []string{}
	for {
		ks, cur, err := cs.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, ks...)
		if cur == 0 {
			break
		}
		cursor = cur
	}
	return keys, nil
}

func (cs *ConversationStore) Clear(ctx context.Context) error {
	keys, err := cs.List(ctx)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return cs.client.Del(ctx, keys...).Err()
}

func (cs *ConversationStore) AppendMessage(ctx context.Context, sessionID string, role, content string) error {
	key := cs.convKey(sessionID)
	// append to list for efficient streaming
	msg := memory.Message{Role: role, Content: content, Timestamp: time.Now().Unix()}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := cs.client.RPush(ctx, key, b).Err(); err != nil {
		return err
	}
	if cs.ttl > 0 {
		_ = cs.client.Expire(ctx, key, cs.ttl).Err()
	}
	return nil
}

func (cs *ConversationStore) GetMessages(ctx context.Context, sessionID string) ([]memory.Message, error) {
	key := cs.convKey(sessionID)
	vals, err := cs.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if errors.Is(err, rds.Nil) {
			return []memory.Message{}, nil
		}
		return nil, err
	}
	msgs := make([]memory.Message, 0, len(vals))
	for _, v := range vals {
		var m memory.Message
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (cs *ConversationStore) ClearSession(ctx context.Context, sessionID string) error {
	return cs.client.Del(ctx, cs.convKey(sessionID)).Err()
}

var _ memory.ConversationStore = (*ConversationStore)(nil)
