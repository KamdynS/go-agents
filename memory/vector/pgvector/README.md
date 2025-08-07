### pgvector VectorStore Adapter

Implements `memory.VectorStore` on Postgres + pgvector.

Schema example:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS documents (
  id text PRIMARY KEY,
  content text NOT NULL,
  embedding vector(1536),
  meta jsonb
);
```

Usage example:
```go
import (
  "context"
  "github.com/jackc/pgx/v5"
  pgv "github.com/KamdynS/go-agents/memory/vector/pgvector"
)

conn, _ := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
store := pgv.New(conn, "documents")
```

