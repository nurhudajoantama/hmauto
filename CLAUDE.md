# hmauto — Claude Code Context

Home automation backend in Go. IoT state management + alerting + internet monitoring.

## Package map

| Package | Role |
|---|---|
| `app/hmstt` | State store (type+key → value). Redis Hash per type. |
| `app/hmalert` | Alert pipeline: publish → RabbitMQ → consume → Discord webhook |
| `app/hmapikey` | Admin API for API key create/revoke/list |
| `app/hmmon` | Background worker: internet ping, modem restart via switch state |
| `app/server` | gorilla/mux HTTP server, middleware wiring |
| `app/worker` | errgroup wrapper for background goroutines |
| `internal/apikey` | Redis-backed API key store (create/validate/revoke/list) |
| `internal/redis` | Redis client init and close |
| `internal/middleware` | Auth (API key + admin), rate limit, security headers, Prometheus, trace ID |
| `internal/instrumentation` | zerolog + OpenTelemetry (OTLP) |
| `internal/rabbitmq` | RabbitMQ AMQP connection |
| `internal/discord` | Discord webhook client for alert delivery |
| `internal/health` | Health check handlers (Redis + RabbitMQ) |
| `internal/response` | JSON response envelope helpers |

## Conventions

- Errors wrapped with context: `fmt.Errorf("...: %w", err)`.
- No ORM — Redis client calls directly in store layer.
- HTTP handlers return JSON always via `internal/response`.
- Zerolog context logger via `zerolog.Ctx(ctx)` — propagate ctx everywhere.
- All endpoints require auth unless explicitly public (health, metrics, healthz, hello).
- Admin routes (`/admin/*`) use config `AdminKey` only — never Redis.
- Regular API keys validated from Redis: `GET apikey:{key}`.
- State key schema: `HSET hmstt:{type} {k} {json}`.
- No `log.Fatal` outside `main.go`.

## Key files

- `main.go` — wiring, DI, graceful shutdown
- `internal/config/types.go` — all config structs
- `conf.example.yaml` — update whenever config structs change
- `app/hmstt/store.go` — Redis state store
- `app/hmstt/util.go` — type/value validation (`canTypeChangedWithKey`)
- `internal/apikey/store.go` — API key CRUD + `ErrKeyNotFound` sentinel
- `internal/middleware/auth.go` — `APIKeyAuth` (Redis) + `AdminKeyAuth` (config)
- `docs/` — architecture, API design, Redis schema, observability

## Dependency overview

```
Redis (state + apikeys)   RabbitMQ (event bus)   Discord (alerts)
      ↕                         ↕
   hmstt ←──event──→ hmalert ←── hmmon
      ↕                ↕
   HTTP server (gorilla/mux + middleware chain)
         ↕
   /admin/* (hmapikey)
```

## What NOT to do

- Do not add GORM or any SQL dependency.
- Do not return HTML from any endpoint.
- Do not store or validate admin key in Redis — config only.
- Do not skip zerolog context propagation in new handlers/services.
- Do not use `log.Fatal` outside of `main.go`.
- Do not use `math/rand` for key generation — always `crypto/rand`.
- Do not return the full API key value except in `POST /admin/apikeys` response.
