# Redis Migration

Replacing PostgreSQL/GORM with Redis for all persistent storage.

## Redis requirements

- **Persistent** — must run with `appendonly yes` (AOF) or RDB snapshots. Not ephemeral.
- Recommended: `appendonly yes` + `appendfsync everysec` in `redis.conf`.
- No TTL on state keys or API keys unless explicitly set.
- Single Redis DB (db 0) is fine; use key namespacing instead of multiple DBs.

## Config struct changes

In `internal/config/types.go`, replace `SQL` with `Redis`:

```go
type Redis struct {
    Host     string `yaml:"host"`
    Port     string `yaml:"port"`
    Password string `yaml:"password"`
    DB       int    `yaml:"db"`
}

func (r Redis) Addr() string {
    return fmt.Sprintf("%s:%s", r.Host, r.Port)
}
```

In `Config` struct:
- Remove `DB SQL`
- Add `Redis Redis`
- Add `Security.AdminKey string`

## Key schema

### hmstt state

```
Key pattern : hmstt:{type}
Redis type  : Hash
Field       : {k}   (the sub-key within the type)
Value       : JSON  {"value":"on","updated_at":"2024-01-01T00:00:00Z"}
```

Examples:
```
HSET hmstt:switch  modem_switch  '{"value":"on","updated_at":"..."}'
HSET hmstt:switch  router        '{"value":"off","updated_at":"..."}'
HGET hmstt:switch  modem_switch  → '{"value":"on","updated_at":"..."}'
HGETALL hmstt:switch             → all switch states
```

State type allowlist (from `constant.go`):
- `switch` → allowed values: `on`, `off`

New states for new types are added by simply writing to Redis — no migration needed.

### API keys

```
Key pattern : apikey:{key_value}
Redis type  : String (JSON)
Value       : {"label":"iot-device-1","created_at":"...","last_used":"..."}

Index set   : apikeys:index   (Redis Set, members = all active key values)
```

Operations:
- **Validate**: `EXISTS apikey:{key}` or `GET apikey:{key}` (non-null = valid)
- **Create**: `SET apikey:{key} {json}` + `SADD apikeys:index {key}`
- **Revoke**: `DEL apikey:{key}` + `SREM apikeys:index {key}`
- **List**: `SMEMBERS apikeys:index` → for each, `GET apikey:{member}`

## Store interface (hmstt)

Replace `HmsttStore` (GORM) with a Redis-backed store implementing:

```go
type StateStore interface {
    GetState(ctx context.Context, tipe, k string) (StateEntry, error)
    SetState(ctx context.Context, tipe, k, value string) error
    GetAllByType(ctx context.Context, tipe string) ([]StateEntry, error)
    GetAll(ctx context.Context) ([]StateEntry, error)
}

type StateEntry struct {
    Type      string
    K         string
    Value     string
    UpdatedAt time.Time
}
```

Notes:
- `SetState` uses `HSET hmstt:{tipe} {k} {json}` — atomic, no transaction needed for single writes.
- `GetAll` requires `KEYS hmstt:*` + `HGETALL` per key — acceptable for small datasets (home automation scale).
- The Postgres transaction pattern (BEGIN/COMMIT/ROLLBACK) maps to Redis WATCH+MULTI/EXEC only if needed for CAS; for simple state set it's not needed.

## New internal package: `internal/redis`

Create `internal/redis/client.go`:

```go
package redis

import (
    "context"
    "github.com/redis/go-redis/v9"
    "github.com/nurhudajoantama/hmauto/internal/config"
)

func NewClient(cfg config.Redis) *redis.Client {
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr(),
        Password: cfg.Password,
        DB:       cfg.DB,
    })
    return rdb
}

func Close(ctx context.Context, rdb *redis.Client) {
    if err := rdb.Close(); err != nil {
        // log
    }
}
```

Recommended client: `github.com/redis/go-redis/v9`

## Health check update

Replace Postgres health check with Redis ping:

```go
func (h *HealthChecker) checkRedis(ctx context.Context) error {
    return h.redis.Ping(ctx).Err()
}
```

## Files to delete after migration

- `internal/postgres/` (entire package)
- `app/hmstt/model.go` (GORM model)
- `views/` (HTMX templates)
- Remove GORM deps from `go.mod`
