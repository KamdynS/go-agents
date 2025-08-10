package tools

import (
	"context"
	"testing"

	wf "github.com/KamdynS/go-agents/workflow"
)

func TestWorkflowTool(t *testing.T) {
	// simple workflow: uppercases input string
	b := wf.New().Step("s1", func(ctx context.Context, in any) (any, error) { return map[string]any{"out": "ok"}, nil })
	w := b.Build()
	wt := &WorkflowTool{NameStr: "wf", Desc: "d", WF: w}
	if wt.Name() != "wf" || wt.Description() != "d" {
		t.Fatalf("bad meta")
	}
	out, err := wt.Execute(context.Background(), `{"a":1}`)
	if err != nil || out == "" {
		t.Fatalf("exec: %v %q", err, out)
	}
	if _, err := wt.Execute(context.Background(), ``); err != nil {
		t.Fatalf("empty input should still work: %v", err)
	}
	wt.WF = nil
	if _, err := wt.Execute(context.Background(), `{"a":1}`); err == nil {
		t.Fatalf("expected nil workflow error")
	}
}
