package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmbedSuccessAndErrors(t *testing.T) {
	// Success server
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1,2,3]}]}`))
	}))
	defer good.Close()

	c, _ := NewClient(Config{APIKey: "k", Timeout: time.Second, BaseURL: good.URL})
	vec, err := c.Embed(context.Background(), "hi", "")
	if err != nil || len(vec) != 3 {
		t.Fatalf("embed good: %v %v", err, vec)
	}

	// Error responses
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"bad"}}`))
	}))
	defer bad.Close()
	c, _ = NewClient(Config{APIKey: "k", Timeout: time.Second, BaseURL: bad.URL})
	if _, err := c.Embed(context.Background(), "hi", ""); err == nil {
		t.Fatalf("expected error")
	}

	// Malformed JSON
	malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer malformed.Close()
	c, _ = NewClient(Config{APIKey: "k", Timeout: time.Second, BaseURL: malformed.URL})
	if _, err := c.Embed(context.Background(), "hi", ""); err == nil {
		t.Fatalf("expected decode error")
	}
}
