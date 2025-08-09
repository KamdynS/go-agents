package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/KamdynS/go-agents/tools"
)

// RegisterAllTools fetches tools from MCP server and registers proxy tools into the local registry.
func RegisterAllTools(ctx context.Context, reg tools.Registry, client ClientLike) error {
	if reg == nil || client == nil {
		return fmt.Errorf("nil registry or client")
	}
	toolsList, err := client.ListTools(ctx)
	if err != nil {
		return err
	}
	for _, t := range toolsList {
		proxy := &mcpToolProxy{client: client, name: t.Name, desc: t.Description, schema: t.Schema}
		if err := reg.Register(proxy); err != nil {
			return err
		}
	}
	return nil
}

type mcpToolProxy struct {
	client ClientLike
	name   string
	desc   string
	schema map[string]interface{}
}

func (m *mcpToolProxy) Name() string                   { return m.name }
func (m *mcpToolProxy) Description() string            { return m.desc }
func (m *mcpToolProxy) Schema() map[string]interface{} { return m.schema }
func (m *mcpToolProxy) Execute(ctx context.Context, input string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return m.client.ExecuteTool(c, m.name, input)
}

var _ tools.Tool = (*mcpToolProxy)(nil)
