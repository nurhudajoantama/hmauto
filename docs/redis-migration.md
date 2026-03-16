# Redis

Redis is the sole persistent storage. No SQL/ORM dependency.

## Production configuration

Redis must run with persistence enabled — not ephemeral:

```conf
appendonly yes
appendfsync everysec
```

RDB snapshots are also acceptable for lower durability requirements, but AOF is recommended for this use case (state + API keys must survive restarts).

## Config

```yaml
redis:
  host: "localhost"
  port: "6379"
  password: ""
  db: 0
```

Client: `github.com/redis/go-redis/v9` via `internal/redis.NewClient(cfg)`.

## Key schema

### State (hmstt)

```
Redis type : Hash
Key        : hmstt:{type}
Field      : {k}
Value      : JSON  {"value":"on","updated_at":"2024-01-01T00:00:00Z"}
```

Operations:
- `HSET hmstt:{type} {k} {json}` — set/update a state value
- `HGET hmstt:{type} {k}` — read single state
- `HGETALL hmstt:{type}` — all states for a type
- `KEYS hmstt:*` + HGETALL per key — all states (used in `GetAll`)

Implementation: `app/hmstt/store.go` → `HmsttStore`

Current allowed types and values:
- `switch` → `on` | `off`

Adding a new type: add it to `canTypeChangedWithKey` in `app/hmstt/util.go`.

### API keys

```
Redis type : String
Key        : apikey:{key_value}
Value      : JSON  {"label":"iot-1","created_at":"...","last_used":"..."}

Redis type : Set
Key        : apikeys:index
Members    : all active key_values
```

Operations:
- `SET apikey:{key} {json}` + `SADD apikeys:index {key}` — create (pipelined)
- `GET apikey:{key}` — validate (non-nil = valid)
- `DEL apikey:{key}` + `SREM apikeys:index {key}` — revoke
- `SMEMBERS apikeys:index` → `GET` each — list

Implementation: `internal/apikey/store.go` → `RedisStore`

Notes:
- `last_used` is updated asynchronously on validate (background goroutine, non-blocking)
- `ListKeys` returns `key_hint` only (first4...last4) — never the full key value
- `RevokeKey` returns `ErrKeyNotFound` if the key doesn't exist (`DEL` returns 0)
