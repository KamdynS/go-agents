# Agent Tools: How to build your own

This guide shows how to add custom tools that agents can call during the reasoning loop.

## Interface

Implement `tools.Tool`:

```go
package yourpkg

import (
  "context"
  "github.com/KamdynS/go-agents/tools"
)

type EchoTool struct{}

func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Description() string { return "Echoes the input string" }
func (e *EchoTool) Execute(ctx context.Context, input string) (string, error) {
  return "ECHO:" + input, nil
}
// Parameters follow JSON Schema; for v0 we expect a single string field named "input".
func (e *EchoTool) Schema() map[string]interface{} {
  return map[string]interface{}{
    "type":       "object",
    "properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
    "required":   []string{"input"},
  }
}

var _ tools.Tool = (*EchoTool)(nil)
```

## Register your tool

Tools are registered in a `tools.Registry` and passed into the agent config:

```go
reg := tools.NewRegistry()
_ = reg.Register(&yourpkg.EchoTool{})

agent := core.NewChatAgent(core.ChatConfig{
  Model:  model,    // any llm.Client
  Tools:  reg,      // your registry with tools
  Mem:    memStore, // optional
  Config: core.AgentConfig{ SystemPrompt: "You are a helpful assistant", MaxIterations: 2 },
})
```

## How tools are invoked (v0 behavior)
- The agent advertises each registered tool to the LLM as a function with your `Schema()` as JSON schema parameters.
- When the LLM returns a tool call, the agent:
  - Parses `arguments` as JSON and extracts the `input` field if present; otherwise uses the raw argument string.
  - Executes your tool with `Execute(ctx, input)` via the registry.
  - Appends a `tool` role message with your tool result and repeats the loop until no more tool calls or `MaxIterations` reached.
- Errors from `Execute` are surfaced back to the model as a tool message in the form `"error: <message>"`.

## Input schema guidelines
- For v0, prefer a single-parameter schema with `{ "input": string }`.
- You may return a richer JSON Schema, but only `input` is extracted by the agent today.

## Observability
- Each tool run is wrapped in a span named `tool.execute` with `genai.tool.name` attribute.
- Latency and `tool_error` metrics are recorded by the default metrics implementation.

## Testing
- Unit-test your tool by calling `Execute` directly.
- For agent integration, script an LLM response with a tool call and verify final output. See `agent/core/agent_test.go`.

## Notes
- Tool names must be unique within a registry; duplicate registration returns an error.
- Respect context cancellations/timeouts inside `Execute`.
- Keep tools deterministic and side-effect-aware; prefer idempotent behavior and explicit inputs.
