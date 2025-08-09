package workflow

import (
	"context"
	"errors"
	"time"
)

// StepFunc is the function executed by a step. It receives the previous output and returns the next output.
type StepFunc func(ctx context.Context, input any) (any, error)

// ConditionFunc decides whether the step/edge should execute.
type ConditionFunc func(ctx context.Context, input any, previousOutput any) bool

// MergeFunc combines outputs from multiple branches.
type MergeFunc func(ctx context.Context, inputs []any) (any, error)

// Event represents a single execution event for observability/streaming.
type Event struct {
	Type      string    `json:"type"` // "start_step", "end_step", "error"
	Step      string    `json:"step"`
	Status    string    `json:"status"` // "ok" or "error"
	Timestamp time.Time `json:"timestamp"`
	Output    any       `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Option configures workflow runs.
type Option func(*runConfig)

type runConfig struct {
	events chan<- Event
}

// WithEvents streams events to the provided channel during Run.
func WithEvents(events chan<- Event) Option { return func(rc *runConfig) { rc.events = events } }

// step represents a node in a workflow.
type step struct {
	name     string
	fn       StepFunc
	precond  ConditionFunc
	next     *step
	branches []*step
	brConds  []ConditionFunc
	merge    *mergeStep
}

type mergeStep struct {
	name string
	fn   MergeFunc
	next *step
}

// Builder constructs a workflow graph using a fluent API.
type Builder struct {
	root              *step
	current           *step
	lastBranchParent  *step
	lastEdgeIsBranch  bool
	lastBranchEdgeIdx int
}

// New creates a workflow builder.
func New() *Builder { return &Builder{} }

// Branch creates a new branch builder with a single root step.
func Branch(name string, fn StepFunc) *Builder {
	b := &Builder{}
	b.Step(name, fn)
	return b
}

// Step adds a step. If this is the first, it becomes the root; otherwise it chains after the current step.
func (b *Builder) Step(name string, fn StepFunc) *Builder {
	s := &step{name: name, fn: fn}
	if b.root == nil {
		b.root = s
		b.current = s
		b.lastBranchParent = nil
		b.lastEdgeIsBranch = false
		return b
	}
	// Chain after current
	b.current.next = s
	b.current = s
	b.lastBranchParent = nil
	b.lastEdgeIsBranch = false
	return b
}

// Then is an alias for Step.
func (b *Builder) Then(name string, fn StepFunc) *Builder { return b.Step(name, fn) }

// When applies a condition to the most recently added edge or step.
// If called after Branch(), it applies to the last attached branch edge.
// If called after Step/Then, it applies to the precondition of the current step.
func (b *Builder) When(cond ConditionFunc) *Builder {
	if cond == nil {
		return b
	}
	if b.lastEdgeIsBranch && b.lastBranchParent != nil && b.lastBranchEdgeIdx >= 0 {
		// Condition on the branch edge
		parent := b.lastBranchParent
		if b.lastBranchEdgeIdx < len(parent.branches) {
			if len(parent.brConds) == 0 {
				parent.brConds = make([]ConditionFunc, len(parent.branches))
			}
			parent.brConds[b.lastBranchEdgeIdx] = cond
		}
		return b
	}
	// Precondition for current step
	if b.current != nil {
		b.current.precond = cond
	}
	return b
}

// Branch attaches the provided branches to the current step. Use Merge() after this to combine outputs.
func (b *Builder) Branch(branches ...*Builder) *Builder {
	if b.current == nil {
		return b
	}
	parent := b.current
	for i, childB := range branches {
		if childB == nil || childB.root == nil {
			continue
		}
		parent.branches = append(parent.branches, childB.root)
		b.lastEdgeIsBranch = true
		b.lastBranchParent = parent
		b.lastBranchEdgeIdx = len(parent.branches) - 1
		// Ensure each branch's tail continues nowhere for now; chaining within branch is handled inside child builder
		_ = i // index kept for When() calls between Branch() invocations
	}
	return b
}

// Merge attaches a merge step after the current step's branches. It will receive all executed branch outputs.
func (b *Builder) Merge(name string, fn MergeFunc) *Builder {
	if b.current == nil {
		return b
	}
	if b.current.merge != nil {
		return b
	}
	b.current.merge = &mergeStep{name: name, fn: fn}
	// Move current to merge to allow continued Then()
	b.current = &step{name: name + "#internal-merge"}
	b.current.fn = func(ctx context.Context, in any) (any, error) { return in, nil }
	b.current.precond = nil
	b.current.branches = nil
	b.current.brConds = nil
	b.current.merge = nil
	b.current.next = nil
	b.current.name = name // external name retained
	// Link merge -> current passthrough node for further chaining
	b.lastBranchParent.merge.next = b.current
	// Reset branching state
	b.lastBranchParent = nil
	b.lastEdgeIsBranch = false
	return b
}

// Build finalizes the workflow and returns a runnable Workflow.
func (b *Builder) Build() *Workflow { return &Workflow{root: b.root} }

// Workflow executes a built graph.
type Workflow struct {
	root *step
}

// Run executes the workflow.
func (w *Workflow) Run(ctx context.Context, input any, opts ...Option) (any, error) {
	rc := &runConfig{}
	for _, o := range opts {
		o(rc)
	}
	if w == nil || w.root == nil {
		return input, nil
	}
	return w.execStep(ctx, w.root, input, rc)
}

func (w *Workflow) execStep(ctx context.Context, s *step, in any, rc *runConfig) (any, error) {
	cur := s
	prevOut := in
	for cur != nil {
		// Precondition check (for Then/Step)
		if cur.precond != nil {
			if !cur.precond(ctx, in, prevOut) {
				// Skip execution; carry prevOut forward
				if cur.next == nil && len(cur.branches) == 0 {
					return prevOut, nil
				}
				in = prevOut
				cur = cur.next
				continue
			}
		}
		emit(rc, Event{Type: "start_step", Step: cur.name, Status: "ok", Timestamp: time.Now()})
		out, err := cur.fn(ctx, prevOut)
		if err != nil {
			// Suspended?
			if se, ok := err.(*suspendError); ok {
				emit(rc, Event{Type: "error", Step: cur.name, Status: "error", Timestamp: time.Now(), Error: se.Error()})
				return nil, err
			}
			emit(rc, Event{Type: "error", Step: cur.name, Status: "error", Timestamp: time.Now(), Error: err.Error()})
			return nil, err
		}
		emit(rc, Event{Type: "end_step", Step: cur.name, Status: "ok", Timestamp: time.Now(), Output: out})

		// Branching
		if len(cur.branches) > 0 {
			results := make([]any, 0, len(cur.branches))
			for i, child := range cur.branches {
				// Condition per branch
				if len(cur.brConds) > i && cur.brConds[i] != nil {
					if !cur.brConds[i](ctx, out, out) {
						continue
					}
				}
				childOut, err := w.execStep(ctx, child, out, rc)
				if err != nil {
					if _, ok := err.(*suspendError); ok {
						return nil, err
					}
					return nil, err
				}
				results = append(results, childOut)
			}
			if cur.merge != nil {
				emit(rc, Event{Type: "start_step", Step: cur.merge.name, Status: "ok", Timestamp: time.Now()})
				merged, err := cur.merge.fn(ctx, results)
				if err != nil {
					if _, ok := err.(*suspendError); ok {
						return nil, err
					}
					emit(rc, Event{Type: "error", Step: cur.merge.name, Status: "error", Timestamp: time.Now(), Error: err.Error()})
					return nil, err
				}
				emit(rc, Event{Type: "end_step", Step: cur.merge.name, Status: "ok", Timestamp: time.Now(), Output: merged})
				prevOut = merged
				cur = cur.merge.next
				continue
			}
			// No merge; return last result if any
			if len(results) > 0 {
				return results[len(results)-1], nil
			}
			return out, nil
		}

		prevOut = out
		cur = cur.next
	}
	return prevOut, nil
}

func emit(rc *runConfig, e Event) {
	if rc != nil && rc.events != nil {
		select {
		case rc.events <- e:
		default:
		}
	}
}

// Errors
var (
	ErrNoRoot = errors.New("workflow has no root step")
)
