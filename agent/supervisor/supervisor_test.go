package supervisor

import (
	"context"
	"errors"
	"testing"

	core "github.com/KamdynS/go-agents/agent/core"
)

type fakeAgent struct {
	reply string
	err   error
}

func (f fakeAgent) Run(ctx context.Context, input core.Message) (core.Message, error) {
	if f.err != nil {
		return core.Message{}, f.err
	}
	return core.Message{Role: "assistant", Content: f.reply + ":" + input.Content}, nil
}
func (f fakeAgent) RunStream(ctx context.Context, input core.Message, output chan<- core.Message) error {
	defer close(output)
	if f.err != nil {
		return f.err
	}
	output <- core.Message{Role: "assistant", Content: f.reply}
	return nil
}

func TestAgentTool(t *testing.T) {
	// Name/Description/Schema basics
	at := &AgentTool{NameStr: "delegate", Desc: "wraps an agent", Agent: fakeAgent{reply: "ok"}}
	if at.Name() != "delegate" || at.Description() != "wraps an agent" {
		t.Fatalf("unexpected name/desc")
	}
	if _, ok := at.Schema()["type"]; !ok {
		t.Fatalf("schema should contain type")
	}
	// Execute success
	out, err := at.Execute(context.Background(), "hello")
	if err != nil || out == "" {
		t.Fatalf("execute failed: %v %q", err, out)
	}
	// Execute error when nil agent
	at.Agent = nil
	if _, err := at.Execute(context.Background(), "x"); err == nil {
		t.Fatalf("expected error on nil agent")
	}
}

func TestSequentialPolicy(t *testing.T) {
	p := SequentialPolicy{}
	a1 := fakeAgent{reply: "A1"}
	a2 := fakeAgent{reply: "A2"}
	out, err := p.Execute(context.Background(), "seed", []core.Agent{a1, a2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A1 echoes with input, A2 receives previous output content
	if out == "" || out[:2] != "A2" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFanOutFirst(t *testing.T) {
	p := FanOutFirst{}
	// one failing, one succeeding fast
	slowErr := fakeAgent{err: errors.New("boom")}
	fastOk := fakeAgent{reply: "OK"}
	out, err := p.Execute(context.Background(), "q", []core.Agent{slowErr, fastOk})
	if err != nil || out == "" {
		t.Fatalf("expected first success, got %v %q", err, out)
	}

	// all failing -> last error returned
	_, err = p.Execute(context.Background(), "q", []core.Agent{fakeAgent{err: errors.New("e1")}, fakeAgent{err: errors.New("e2")}})
	if err == nil {
		t.Fatalf("expected error when all fail")
	}
}
