### Memory Adapter Guidelines

Implementations should satisfy the interfaces in `memory`:
- `Store`, `ConversationStore`, `VectorStore`

Requirements
- Context-aware: respect `ctx.Done()` and timeouts
- Thread-safe: concurrent calls allowed
- Deterministic errors: `not found` vs other errors
- Observability: optional; emit metrics/traces using `observability.*` if imported

Suggested labels
- `backend`: e.g., `redis`, `pgvector`, `qdrant`
- `op`: e.g., `store`, `retrieve`, `append_message`, `add_document`, `query`
- `status`: `ok` or `error`

Example skeleton

```go
type Store struct { /* client/pool */ }

func New(...) *Store { /* ... */ }

func (s *Store) Store(ctx context.Context, key string, value interface{}) error {
    // do op; respect context; return typed errors
}

// implement Retrieve, Delete, List, Clear
```

Versioning
- Place adapters in submodules (own `go.mod`) to avoid pulling deps into core
- Examples: `memory/redis`, `memory/vector/pgvector`, `memory/vector/qdrant`

