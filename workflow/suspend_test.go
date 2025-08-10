package workflow

import (
	"context"
	"testing"
)

func TestMemorySuspender(t *testing.T) {
	ms := NewMemorySuspender()
	s := &SuspendState{WorkflowID: "id1", Cursor: "c", Data: 123}
	if err := ms.Save(context.Background(), s); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := ms.Load(context.Background(), "id1")
	if err != nil || got.WorkflowID != "id1" {
		t.Fatalf("load: %v %#v", err, got)
	}
	if _, err := ms.Load(context.Background(), "missing"); err == nil {
		t.Fatalf("expected not found")
	}
}

func TestRequestSuspend(t *testing.T) {
	err := RequestSuspend("w", "cur", map[string]any{"k": "v"})
	if err == nil {
		t.Fatalf("expected error type")
	}
	if _, ok := err.(*suspendError); !ok {
		t.Fatalf("wrong type: %T", err)
	}
}
