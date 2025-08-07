# Server

- Status: HTTP server implemented with tests

### Endpoints
- `GET /health`
- `POST /chat` (JSON)
- `POST /chat/stream` (SSE)

SSE: headers `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`. Flush after each event. Final `event: done` sent on completion or cancel.

### Middleware stack
- Recovery → Request ID → Timeout → Observability → CORS

### Config
- `Port` (8080)
- `ReadTimeout` 10s, `WriteTimeout` 10s, `RequestTimeout` 60s
- `EnableCORS` (false), `AllowedOrigins` ("*")

### Observability hooks
- New span per request with route/method/status and request id injected as `X-Request-ID`.
- Request metrics (rate, latency, status).

### Next
- gRPC service
- Auth middleware (API key / bearer)
