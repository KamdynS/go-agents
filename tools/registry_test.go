package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

type dummyTool struct {
	name, desc string
	out        string
	err        error
}

func (d dummyTool) Name() string                   { return d.name }
func (d dummyTool) Description() string            { return d.desc }
func (d dummyTool) Schema() map[string]interface{} { return map[string]interface{}{"type": "object"} }
func (d dummyTool) Execute(ctx context.Context, input string) (string, error) {
	if d.err != nil {
		return "", d.err
	}
	return d.out + ":" + input, nil
}

func TestRegistryRegisterGetListExecute(t *testing.T) {
	r := NewRegistry()
	a := dummyTool{name: "a", desc: "A", out: "OA"}
	b := dummyTool{name: "b", desc: "B", out: "OB"}

	if err := r.Register(a); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := r.Register(b); err != nil {
		t.Fatalf("register b: %v", err)
	}
	if err := r.Register(a); err == nil {
		t.Fatalf("expected duplicate register error")
	}

	if _, ok := r.Get("a"); !ok {
		t.Fatalf("expected to get a")
	}
	names := r.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := r.Execute(ctx, "a", "in")
	if err != nil || out != "OA:in" {
		t.Fatalf("execute unexpected: %v %q", err, out)
	}
}

func TestRegistryExecuteErrors(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Execute(context.Background(), "none", "x"); err == nil {
		t.Fatalf("expected not found error")
	}
	_ = r.Register(dummyTool{name: "e", err: errors.New("boom")})
	if _, err := r.Execute(context.Background(), "e", "x"); err == nil {
		t.Fatalf("expected execution error")
	}
}
