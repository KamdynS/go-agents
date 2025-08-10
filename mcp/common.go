package mcp

import "context"

// ClientLike abstracts over different MCP client transports
type ClientLike interface {
	ListTools(ctx context.Context) ([]ToolInfo, error)
	ExecuteTool(ctx context.Context, name string, input string) (string, error)
}
