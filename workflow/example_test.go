package workflow_test

import (
	"context"
	"fmt"
	"testing"

	wf "github.com/KamdynS/go-agents/workflow"
)

func TestWorkflow_StepChainBranchMerge(t *testing.T) {
	// Build: step A -> branch {B1, B2 (cond)} -> merge M -> step C
	w := wf.New().
		Step("A", func(ctx context.Context, in any) (any, error) { return 2, nil }).
		Branch(
			wf.Branch("B1", func(ctx context.Context, in any) (any, error) { return in.(int) + 3, nil }),
			wf.Branch("B2", func(ctx context.Context, in any) (any, error) { return in.(int) * 5, nil }).When(func(ctx context.Context, in any, prev any) bool { return in.(int)%2 == 0 }),
		).
		Merge("M", func(ctx context.Context, inputs []any) (any, error) {
			sum := 0
			for _, v := range inputs {
				sum += v.(int)
			}
			return sum, nil
		}).
		Then("C", func(ctx context.Context, in any) (any, error) { return in.(int) + 1, nil }).
		Build()

	out, err := w.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if out.(int) != 2+3+10+1 { // A=2; B1=5; B2=10; M=sum=15; C=16
		t.Fatalf("unexpected output: %v", out)
	}
}

func Example() {
	events := make(chan wf.Event, 16)
	done := make(chan struct{})
	go func() {
		for e := range events {
			fmt.Printf("%s %s\n", e.Type, e.Step)
		}
		close(done)
	}()
	w := wf.New().
		Step("parse_input", func(ctx context.Context, in any) (any, error) { return "q", nil }).
		Then("fetch", func(ctx context.Context, in any) (any, error) { return []string{"doc1", "doc2"}, nil }).
		Branch(
			wf.Branch("analyze_doc1", func(ctx context.Context, in any) (any, error) { return "ok1", nil }),
			wf.Branch("analyze_doc2", func(ctx context.Context, in any) (any, error) { return "ok2", nil }),
		).
		Merge("combine", func(ctx context.Context, inputs []any) (any, error) { return inputs, nil }).
		Build()
	_, _ = w.Run(context.Background(), nil, wf.WithEvents(events))
	close(events)
	<-done
}
