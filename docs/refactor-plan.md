# Refactor Plan

Ordered steps. Complete each before starting the next — they have dependencies.

## Step 1 — Config: add Redis, AdminKey, remove SQL

**Files**: `internal/config/types.go`, `conf.example.yaml`

- Add `Redis` struct (host, port, password, db)
- Add `Security.AdminKey string` field
- Remove `SQL` struct from `Config`
- Update `conf.example.yaml`

No functional change yet — just config shape.

---

## Step 2 — New `internal/redis` package

**Files**: `internal/redis/client.go`

- `NewClient(cfg config.Redis) *redis.Client`
- `Close(ctx, client)`
- Add `github.com/redis/go-redis/v9` to go.mod

---

## Step 3 — Replace hmstt store (Postgres → Redis)

**Files**: `app/hmstt/store.go`, `app/hmstt/model.go`

- Delete GORM model (`model.go`)
- Rewrite `store.go` with Redis Hash operations:
  - `GetState(ctx, tipe, k)` → `HGET hmstt:{tipe} {k}` → unmarshal JSON
  - `SetState(ctx, tipe, k, value)` → marshal JSON → `HSET hmstt:{tipe} {k} {json}`
  - `GetAllByType(ctx, tipe)` → `HGETALL hmstt:{tipe}`
  - `GetAll(ctx)` → `KEYS hmstt:*` + HGETALL per key
- Implement `StateStore` interface
- Update `NewStore` to accept `*redis.Client` instead of `*gorm.DB`
- Update service to remove transaction logic (not needed for single-key sets)

---

## Step 4 — Remove HTMX: replace handlers with JSON API

**Files**: `app/hmstt/handlers.go`, `app/hmstt/constant.go`

- Delete HTML template loading, HTMX handler methods
- New handlers: `getState`, `setState`, `listStatesByType`, `listAllStates`
- All return JSON via `internal/response`
- Update `RegisterHandlers` to use new routes (see `docs/api-design.md`)
- Remove `HTML_TEMPLATE_*` constants and `TYPE_TEMPLATES` map from `constant.go`
- Delete `views/` directory

---

## Step 5 — API key store in Redis

**Files**: new `internal/apikey/store.go`

- `CreateKey(ctx, label) (key string, err error)`
  - `crypto/rand` 32 bytes → hex
  - `SET apikey:{key} {json}` + `SADD apikeys:index {key}`
- `ValidateKey(ctx, key) (bool, error)`
  - `GET apikey:{key}` → exists and non-empty = valid
  - Update `last_used` async (goroutine, non-blocking)
- `RevokeKey(ctx, key) error`
  - `DEL apikey:{key}` + `SREM apikeys:index {key}`
- `ListKeys(ctx) ([]KeyMetadata, error)`
  - `SMEMBERS apikeys:index` → for each member `GET apikey:{m}`
  - Return `key_hint` = first 4 + "..." + last 4 chars — never full key

---

## Step 6 — Auth middleware reads from Redis

**Files**: `internal/middleware/auth.go`

- Replace `validAPIKeys map[string]bool` param with `apikey.Store` interface
- `ValidateKey(ctx, key)` call instead of map lookup
- Add `AdminKeyAuth(adminKey string)` for admin routes (config-only check, no Redis)

Update `server.go`:
- `ApplyAuthMiddleware(subrouter)` — passes Redis-backed store
- `ApplyAdminMiddleware(subrouter)` — passes config admin key

---

## Step 7 — Admin API for key management

**Files**: new `app/hmapikey/handler.go`, `app/hmapikey/service.go`

- `GET  /admin/apikeys` → list (hints only)
- `POST /admin/apikeys` → create, returns full key once
- `DELETE /admin/apikeys/{key}` → revoke

Register in `main.go` on admin subrouter.

---

## Step 8 — Remove Postgres entirely

**Files**: `internal/postgres/`, `main.go`, `internal/health/health.go`

- Delete `internal/postgres/` package
- Remove Postgres wiring in `main.go`
- Update `HealthChecker` to check Redis instead of Postgres
- `go mod tidy` to remove GORM and pgx deps

---

## Step 9 — Prometheus metrics

**Files**: `internal/middleware/prometheus.go`, `main.go`, `app/server/server.go`

- Add `github.com/prometheus/client_golang` to go.mod
- Define metrics registry
- Add `PrometheusMiddleware` to global middleware chain
- Mount `GET /metrics` → `promhttp.Handler()` as public route
- Add business counters in service layer (hmstt, hmalert)

---

## Step 10 — Sentry integration

**Files**: `main.go`, `internal/config/types.go`, `conf.example.yaml`

- Add `github.com/getsentry/sentry-go` to go.mod
- Init Sentry in `main.go` before any other setup
- Wrap HTTP handler with `sentryhttp` middleware
- Add `CaptureException` calls for unexpected store/service errors

---

## Step 11 — OTEL trace completion

**Files**: `internal/instrumentation/otel.go`, store/service layers

- Ensure OTLP endpoint is configurable via config (not just env var)
- Add spans in `hmstt` store Redis calls
- Add spans in `hmalert` publish/consume
- Inject trace ID into zerolog context in middleware

---

## Cleanup

After all steps:
- `go mod tidy`
- Delete `views/` directory (if not already done in Step 4)
- Update `conf.example.yaml` with all new fields
- Update `README.md` API section to reflect new endpoints
- Update `internal/health/health.go` imports (Postgres → Redis)
