# HTTP Server (Optional Reference)

- Status: HTTP server implemented with tests

### Endpoints
- `GET /health`
- `POST /chat` (JSON)
- `POST /chat/stream` (SSE)

SSE: headers `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`. Flush after each event. Final `event: done` sent on completion or cancel.

### Middleware stack
- Recovery → Request ID → Timeout → Observability

### Config
- `Port` (8080)
- `ReadTimeout` 10s, `WriteTimeout` 10s, `RequestTimeout` 60s

### Observability hooks
- New span per request with route/method/status and request id injected as `X-Request-ID`.
- Request metrics (rate, latency, status).

### Security & Limits
- Strict JSON decoding (`DisallowUnknownFields`) and request size limit (default 1 MiB, configurable via `MaxRequestBodyBytes`).

Note: CORS is intentionally not handled in the core server. Add it in your application layer or reverse proxy.

### Integrate into your own server
- Preferred: import `agent/core` and call your agent from your handlers; use our SSE format as a reference.
- Or: wrap the reference server’s handlers in your routing/middleware stack.

### Next
- Auth middleware (API key / bearer)
