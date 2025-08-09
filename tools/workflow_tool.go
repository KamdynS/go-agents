package tools

import (
	"context"
	"encoding/json"
	"fmt"

	wf "github.com/KamdynS/go-agents/workflow"
)

// WorkflowTool runs a prebuilt workflow from a tool call. Input JSON can carry a payload.
type WorkflowTool struct {
	NameStr string
	Desc    string
	WF      *wf.Workflow
}

func (w *WorkflowTool) Name() string        { return w.NameStr }
func (w *WorkflowTool) Description() string { return w.Desc }
func (w *WorkflowTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{"type": "string", "description": "JSON payload for workflow input"},
		},
		"required": []string{"input"},
	}
}
func (w *WorkflowTool) Execute(ctx context.Context, input string) (string, error) {
	if w.WF == nil {
		return "", fmt.Errorf("nil workflow")
	}
	var payload interface{}
	if input != "" {
		var tmp any
		if err := json.Unmarshal([]byte(input), &tmp); err == nil {
			payload = tmp
		} else {
			payload = input
		}
	}
	out, err := w.WF.Run(ctx, payload)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

var _ Tool = (*WorkflowTool)(nil)
