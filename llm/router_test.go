package llm

import (
	"context"
	"errors"
	"testing"
)

type dummyClient struct{ id string }

func (d dummyClient) Chat(ctx context.Context, req *ChatRequest) (*Response, error) {
	return &Response{Content: d.id, Model: req.Model}, nil
}
func (d dummyClient) Completion(ctx context.Context, prompt string) (*Response, error) {
	return &Response{Content: d.id}, nil
}
func (d dummyClient) Stream(ctx context.Context, req *ChatRequest, output chan<- *Response) error {
	close(output)
	return nil
}
func (d dummyClient) Model() string      { return d.id }
func (d dummyClient) Provider() Provider { return Provider("x") }
func (d dummyClient) Validate() error    { return nil }

func TestStaticPolicyAndRouter(t *testing.T) {
	def := dummyClient{id: "def"}
	p := StaticPolicy{Default: def, ByModel: map[string]Client{"m": dummyClient{id: "m"}}}
	r := NewRouterClient(p)
	if err := r.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Model override path
	out, err := r.Chat(context.Background(), &ChatRequest{Model: "m"})
	if err != nil || out.Model != "m" || out.Content != "m" {
		t.Fatalf("chat override: %v %#v", err, out)
	}

	// Default path
	out, err = r.Chat(context.Background(), &ChatRequest{})
	if err != nil || out.Content != "def" {
		t.Fatalf("chat default: %v %#v", err, out)
	}

	// Completion uses default
	if _, err := r.Completion(context.Background(), "p"); err != nil {
		t.Fatalf("completion: %v", err)
	}
}

type errPolicy struct{}

func (errPolicy) Select(req *ChatRequest) (Client, string, error) { return nil, "", errors.New("no") }

func TestRouterPolicyError(t *testing.T) {
	r := NewRouterClient(errPolicy{})
	if _, err := r.Chat(context.Background(), &ChatRequest{}); err == nil {
		t.Fatalf("expected error")
	}
}
