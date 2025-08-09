package workflow

import (
	"context"
	"errors"
)

// SuspendState stores serialized progress to resume later.
type SuspendState struct {
	WorkflowID string      `json:"workflow_id"`
	Cursor     string      `json:"cursor"` // implementation-defined pointer to step
	Data       interface{} `json:"data"`   // input/output so far
}

// Suspender persists and loads suspended workflow states.
type Suspender interface {
	Save(ctx context.Context, state *SuspendState) error
	Load(ctx context.Context, id string) (*SuspendState, error)
}

// MemorySuspender is a dev in-memory storage (not for production).
type MemorySuspender struct{ store map[string]*SuspendState }

func NewMemorySuspender() *MemorySuspender {
	return &MemorySuspender{store: map[string]*SuspendState{}}
}
func (m *MemorySuspender) Save(ctx context.Context, state *SuspendState) error {
	if state == nil || state.WorkflowID == "" {
		return errors.New("invalid state")
	}
	m.store[state.WorkflowID] = state
	return nil
}
func (m *MemorySuspender) Load(ctx context.Context, id string) (*SuspendState, error) {
	if s, ok := m.store[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

// Internal suspension signal
type suspendError struct{ state SuspendState }

func (e *suspendError) Error() string { return "workflow suspended" }

// RequestSuspend can be returned by a step to suspend workflow execution.
func RequestSuspend(id string, cursor string, data any) error {
	return &suspendError{state: SuspendState{WorkflowID: id, Cursor: cursor, Data: data}}
}
