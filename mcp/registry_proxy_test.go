package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/KamdynS/go-agents/tools"
)

type fakeClient struct {
	tools   []ToolInfo
	execErr error
}

func (f *fakeClient) ListTools(ctx context.Context) ([]ToolInfo, error) { return f.tools, nil }
func (f *fakeClient) ExecuteTool(ctx context.Context, name string, input string) (string, error) {
	if f.execErr != nil {
		return "", f.execErr
	}
	return name + ":" + input, nil
}

func TestRegisterAllToolsAndExecute(t *testing.T) {
	reg := tools.NewRegistry()
	fc := &fakeClient{tools: []ToolInfo{{Name: "echo", Description: "d", Schema: map[string]any{"type": "object"}}}}
	if err := RegisterAllTools(context.Background(), reg, fc); err != nil {
		t.Fatalf("register: %v", err)
	}
	out, err := reg.Execute(context.Background(), "echo", "hi")
	if err != nil || out != "echo:hi" {
		t.Fatalf("exec via proxy: %v %q", err, out)
	}
}

func TestRegisterAllToolsNil(t *testing.T) {
	if err := RegisterAllTools(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error for nil args")
	}
}

func TestProxyExecuteError(t *testing.T) {
	reg := tools.NewRegistry()
	fc := &fakeClient{tools: []ToolInfo{{Name: "bad", Description: "", Schema: nil}}, execErr: errors.New("boom")}
	_ = RegisterAllTools(context.Background(), reg, fc)
	if _, err := reg.Execute(context.Background(), "bad", "x"); err == nil {
		t.Fatalf("expected error from client")
	}
}
