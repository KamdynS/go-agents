### Redis Memory Adapters

Implements `memory.Store` and `memory.ConversationStore` on Redis.

Usage example:
```go
import (
  rds "github.com/redis/go-redis/v9"
  memredis "github.com/KamdynS/go-agents/memory/redis"
)

client := rds.NewClient(&rds.Options{Addr: "localhost:6379"})
store := memredis.NewStore(client, 24*time.Hour, "agents")
conv  := memredis.NewConversationStore(client, "agents", 24*time.Hour)
```

