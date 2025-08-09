package supervisor

import (
	"context"
	"fmt"

	core "github.com/KamdynS/go-agents/agent/core"
	"github.com/KamdynS/go-agents/tools"
)

// AgentTool wraps an Agent as a tools.Tool so it can be delegated to.
type AgentTool struct {
	NameStr, Desc string
	Agent         core.Agent
}

func (a *AgentTool) Name() string        { return a.NameStr }
func (a *AgentTool) Description() string { return a.Desc }
func (a *AgentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
		"required":   []string{"input"},
	}
}
func (a *AgentTool) Execute(ctx context.Context, input string) (string, error) {
	if a.Agent == nil {
		return "", fmt.Errorf("nil agent")
	}
	msg := core.Message{Role: "user", Content: input}
	out, err := a.Agent.Run(ctx, msg)
	if err != nil {
		return "", err
	}
	return out.Content, nil
}

var _ tools.Tool = (*AgentTool)(nil)
