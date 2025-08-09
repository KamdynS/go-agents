//go:build mcp_sdk

package mcp

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/KamdynS/go-agents/tools"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SDKConfig defines how to launch/connect to an MCP server via the official SDK
type SDKConfig struct {
	// Command to launch the MCP server (e.g. path to binary)
	Command string
	// Args to pass to the command
	Args []string
	// Optional implementation info
	ClientName    string
	ClientVersion string
}

// RegisterAllToolsWithSDK connects to an MCP server using the official SDK and registers its tools.
func RegisterAllToolsWithSDK(ctx context.Context, reg tools.Registry, cfg SDKConfig) error {
	if reg == nil {
		return fmt.Errorf("nil registry")
	}
	if cfg.Command == "" {
		return fmt.Errorf("empty SDK command")
	}

	impl := &sdkmcp.Implementation{Name: cfg.ClientName}
	if impl.Name == "" {
		impl.Name = "go-agents-mcp-client"
	}
	if cfg.ClientVersion != "" {
		impl.Version = cfg.ClientVersion
	}

	client := sdkmcp.NewClient(impl, nil)
	transport := sdkmcp.NewCommandTransport(exec.Command(cfg.Command, cfg.Args...))
	session, err := client.Connect(ctx, transport)
	if err != nil {
		return err
	}
	// The session will live for the duration of the process; tool execution proxies hold it.

	// List tools via SDK
	tl, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		return err
	}

	for _, t := range tl.Tools {
		// Convert JSON schema (sdkmcp.Tool.Schema) to generic map[string]interface{} if needed
		schema := map[string]interface{}{}
		if t.InputSchema != nil {
			// Best-effort marshal/unmarshal to map form
			m := t.InputSchema.JSONSchema()
			if m != nil {
				schema = m
			}
		}
		proxy := &sdkToolProxy{session: session, name: t.Name, desc: t.Description, schema: schema}
		if err := reg.Register(proxy); err != nil {
			return err
		}
	}
	return nil
}

type sdkToolProxy struct {
	session *sdkmcp.ClientSession
	name    string
	desc    string
	schema  map[string]interface{}
}

func (p *sdkToolProxy) Name() string                   { return p.name }
func (p *sdkToolProxy) Description() string            { return p.desc }
func (p *sdkToolProxy) Schema() map[string]interface{} { return p.schema }

func (p *sdkToolProxy) Execute(ctx context.Context, input string) (string, error) {
	if p.session == nil {
		return "", fmt.Errorf("nil session")
	}
	// Call tool with a single string under key "input" by convention; richer args can be added later
	params := &sdkmcp.CallToolParams{
		Name:      p.name,
		Arguments: map[string]any{"input": input},
	}
	res, err := p.session.CallTool(ctx, params)
	if err != nil {
		return "", err
	}
	if res.IsError {
		return "", fmt.Errorf("mcp tool error")
	}
	// Collect text content parts
	out := ""
	for _, c := range res.Content {
		if txt, ok := c.(*sdkmcp.TextContent); ok {
			out += txt.Text
		}
	}
	return out, nil
}

var _ tools.Tool = (*sdkToolProxy)(nil)
