# Regression Tests

- Live script: `regression-test-backend/run.sh`
  - Loads `.env`, builds server, runs integration tests, shuts server down
- Endpoints covered:
  - /health, /test/llm, /test/structured, /test/models

### Package testing strategy
- Unit tests for pure logic (parsers, error mapping, SSE framing)
- Integration tests for HTTP handlers, observability in-memory collectors
- Contract tests for public interfaces:
  - `memory.Store`, `ConversationStore`, `VectorStore`
  - `llm.Client` via a fake client and `llm.NewInstrumentedClient`
- Example tests that compile and run as documentation
- Optional smoke tests with external infra (behind build tags)

### Running tests
- Fast path (default):
  - `go test ./... -race`
- With adapters compiled in (no external infra required to compile):
  - `go test ./... -race -tags adapters_redis,adapters_pgvector`
- With external services for smoke (opt-in):
  - Start Redis/Postgres (docker/docker-compose) and set `DATABASE_URL`
  - `go test ./... -race -tags adapters_redis,adapters_pgvector,smoke`

### CI recommendations
- Job 1: default `go test ./... -race`
- Job 2: adapters compile check `-tags adapters_redis,adapters_pgvector`
- Job 3 (nightly): smoke `-tags adapters_redis,adapters_pgvector,smoke`
