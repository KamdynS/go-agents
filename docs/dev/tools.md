# Tools

- Status: Registry scaffolded; initial HTTP tool present

### Interface
- `tools.Tool`: `Name()`, `Description()`, `Execute(ctx,input)`, `Schema()`
- `tools.DefaultRegistry` for registration and execution

### Observability
- Span per execution: `tool.execute` with label `genai.tool.name`
- Latency metric and `tool_error` on failures

### Next
- Define input schema typing and validation (agent extracts `input` today; richer schemas supported by user tools)
- Encourage example repos to host built-in tools (calculator, web search) rather than core library
- Safety/allowlist guidance in examples; core remains unopinionated
