### Observability (defaults: no-ops; in-memory dev collectors)

This project exposes swappable observability interfaces with safe defaults.

- **Tracing interfaces**: `observability.Tracer`, `observability.Span`
- **Metrics interface**: `observability.Metrics`
- **Globals**: `observability.TracerImpl`, `observability.MetricsImpl` (no-op by default)

#### Spans emitted
- **http.request**: labels `http.method`, `http.route`, `http.status_code`, `request.id`
- **llm.chat / llm.completion / llm.stream**: labels `genai.model`, `genai.provider`, `genai.finish_reason`, tokens when available
- **tool.execute**: label `genai.tool.name`

#### Metrics emitted
- **Requests**: increments per request with labels `route`, `method`, `status_code`
- **Latency**: per route/method/status
- **Errors**: `llm_error`, `tool_error`
- **Tokens used**: input + output tokens
- **Active agents gauge**: `SetActiveAgents` hook (stub)

#### HTTP context propagation
- Request IDs via `X-Request-ID` header
- Helpers: `ExtractHTTPContext`, `InjectHTTPHeaders`, `GenerateRequestID`

#### Enable OTel/Prom
Replace globals at startup in your app wiring:

```go
// Prometheus-like endpoint without external deps (observability/prom)
promExp := prom.New()
observability.SetMetrics(promExp)
http.Handle("/metrics", prom.Handler(promExp))

// OpenTelemetry tracer shim (requires -tags adapters_otel and app-level OTel setup)
observability.SetTracer(otel.NewTracer("my-service", nil))
```

Expose Prom scrape endpoint and OTel exporter in your app wiring; defaults keep CI green without external infra.
# Observability

- Status: Placeholders for metrics and tracing
- Next:
  - Prometheus metrics for request/latency/tokens
  - OpenTelemetry spans around LLM calls and tools
  - Cost tracking helpers
