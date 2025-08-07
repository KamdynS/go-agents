# Tools

- Status: Registry scaffolded; initial HTTP tool present

### Interface
- `tools.Tool`: `Name()`, `Description()`, `Execute(ctx,input)`, `Schema()`
- `tools.DefaultRegistry` for registration and execution

### Observability
- Span per execution: `tool.execute` with label `genai.tool.name`
- Latency metric and `tool_error` on failures

### Next
- Define input schema typing and validation
- Add built-ins (calculator, web search)
- Safety/allowlist config
