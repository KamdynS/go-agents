# Developer Docs

- Overview: `docs/dev/product-plan.md`
- Roadmap & Status: `docs/dev/roadmap.md`
- LLM Package: `docs/dev/llm.md`
- Agent Runtime: `docs/dev/agent-core.md`
- Memory: `docs/dev/memory.md`
- Server: `docs/dev/server.md`
- Tools: `docs/dev/tools.md`
- Observability: `docs/dev/observability.md`
- Regression Testing: `docs/dev/regression-tests.md`

### Running tests
- Default: `go test ./... -race`
- Adapters compiled: `go test ./... -race -tags adapters_redis,adapters_pgvector`
- Full smoke (external services): `go test ./... -race -tags adapters_redis,adapters_pgvector,smoke`


