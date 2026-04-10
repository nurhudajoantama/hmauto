# hmauto — Claude Code Context

Home automation backend in Go. IoT state management via Redis + RabbitMQ event bus.

## Package map

| Package | Role |
|---|---|
| `app/hmstt` | State store (type+key → value). Redis Hash per type. Publishes state changes to RabbitMQ. |
| `app/server` | gorilla/mux HTTP server, middleware wiring |
| `internal/redis` | Redis client init and close |
| `internal/middleware` | Auth (config bearer token), rate limit, security headers, Prometheus, trace ID |
| `internal/instrumentation` | zerolog + OpenTelemetry (OTLP) |
| `internal/rabbitmq` | RabbitMQ AMQP connection |
| `internal/health` | Health check handlers (Redis + RabbitMQ) |
| `internal/response` | JSON response envelope helpers |

## Conventions

- Errors wrapped with context: `fmt.Errorf("...: %w", err)`.
- No ORM — Redis client calls directly in store layer.
- HTTP handlers return JSON always via `internal/response`.
- Zerolog context logger via `zerolog.Ctx(ctx)` — propagate ctx everywhere.
- All endpoints require auth unless explicitly public (health, metrics, healthz, hello).
- `/v1/*` routes validate `Authorization: Bearer {token}` against config `Security.BearerToken`.
- `/mcp` validates `?token={token}` against config `Security.MCPToken`.
- State key schema: `HSET hmstt:{type} {k} {json}`.
- No `log.Fatal` outside `main.go`.

## Key files

- `main.go` — wiring, DI, graceful shutdown
- `internal/config/types.go` — all config structs
- `conf.example.yaml` — update whenever config structs change
- `app/hmstt/store.go` — Redis state store
- `app/hmstt/event.go` — RabbitMQ topic publisher for state changes
- `app/hmstt/util.go` — type/value validation (`canTypeChangedWithKey`)
- `internal/middleware/auth.go` — `BearerTokenAuth` for `/v1/*` and `QueryTokenAuth` for `/mcp`

## Dependency overview

```
Redis (state)             RabbitMQ (amq.topic event bus)
      ↕                         ↕
   hmstt ──────state change event──────→ (external subscribers)
      ↕
   HTTP server (gorilla/mux + middleware chain)
```

## What NOT to do

- Do not add GORM or any SQL dependency.
- Do not return HTML from any endpoint.
- Do not store or validate auth tokens in Redis.
- Do not skip zerolog context propagation in new handlers/services.
- Do not use `log.Fatal` outside of `main.go`.
