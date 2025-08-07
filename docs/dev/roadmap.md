# Roadmap & Status

- Phase 1 – MVP: Core runtime, HTTP server, LLM (OpenAI/Anthropic), in-memory memory
  - Status: Complete (tests green; regression tests passing)
- Phase 2 – Enhanced SDK & Beta
  - Tools interface + a few built-ins
  - Vector memory (RAG) integration(s)
  - Observability hooks
  - Docs + quickstarts
  - Target: publish 0.1.0-beta next week
- Phase 3 – Multi-Agent & Production Hardening
  - Multi-agent orchestration (optional)
  - gRPC + streaming
  - Prometheus + OpenTelemetry
  - Load tests
  - Target: v0.9 (~2 months after beta)
- Phase 4 – 1.0
  - API stabilization
  - Performance tuning
  - Additional providers
  - Deployment tooling polish
  - Target: v1.0 (~4 months after beta)

Release cadence: tags on root and provider submodules.
