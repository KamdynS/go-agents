# Memory

- Status: In-memory store + conversation store; tests green

### Approaches
1) In-memory (current)
- Pros: zero deps, fastest
- Cons: non-persistent, single-instance only
- Use: unit tests, local dev

2) Redis as ephemeral vector + kv
- Store short-term context windows, tool results, session state; TTL based eviction
- Optional Redis Streams for event logs
- Pros: simple ops, horizontal, good latency; supports semantic caches if paired with external embedder
- Cons: not durable; embeddings still need a store or recompute

3) Vector DB (Chroma, PGVector, Qdrant)
- Store embeddings for RAG; session-docs mapping in a side kv
- Pros: scalable similarity search, filters
- Cons: extra infra; migration/versioning

4) Hybrid: Redis short-term + Vector DB long-term
- Short-lived dialogue/cache in Redis; long-term knowledge in vector store
- Pros: fast hot path; durable knowledge
- Cons: two systems

5) File-backed local (bolt/badger) for dev
- Pros: single binary, persistent between restarts
- Cons: not for multi-instance

Recommended path
- Start with in-memory (default)
- Add Redis adapter for session state + ephemeral recall (no SQL)
- Gate vector store behind interface; later allow PGVector or Qdrant

Interfaces to define
- `MemoryStore` for session state and messages
- `VectorStore` for embeddings (optional)

Operational guidance
- Keep PII minimal; set TTLs; metricize token growth per session
- Avoid noisy logs; rely on metrics/traces

### Adapters and submodules
- Core package exposes only interfaces + in-memory defaults.
- Official adapters as submodules (own go.mod):
  - `memory/redis`: `Store` and `ConversationStore` using Redis (TTL-based, list ops)
  - `memory/vector/pgvector`: `VectorStore` on Postgres with pgvector
- Community-contributed adapters welcome (Qdrant, Chroma, Weaviate, Milvus). See `docs/dev/memory-adapters.md`.
