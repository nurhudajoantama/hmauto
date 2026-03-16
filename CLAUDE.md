# hmauto — Claude Code Context

Home automation backend in Go. IoT state management + alerting + internet monitoring.

## Quick orientation

| Package | Role |
|---|---|
| `app/hmstt` | State store (type+key → value). Currently Postgres/GORM → **migrating to Redis** |
| `app/hmalert` | Alert pipeline: publish → RabbitMQ → consume → Discord webhook |
| `app/hmmon` | Background worker: internet ping, modem restart via switch state |
| `app/server` | gorilla/mux HTTP server, middleware wiring |
| `app/worker` | errgroup wrapper for background goroutines |
| `internal/middleware` | Auth (API key), rate limit, security headers, request size |
| `internal/instrumentation` | zerolog + OpenTelemetry (OTLP) |
| `internal/rabbitmq` | RabbitMQ AMQP connection (kept — IoT/MQTT bus) |
| `internal/postgres` | GORM/Postgres — **being removed** |
| `internal/discord` | Discord webhook client for alert delivery |
| `internal/health` | Health check handlers |

## Active refactor goals

See `docs/refactor-plan.md` for ordered steps.

1. **Replace Postgres with Redis** (persistent, AOF). Redis Hash per type: `HSET hmstt:{type} {k} {value}`. Remove GORM entirely.
2. **Remove HTMX/HTML views** — pure JSON REST API. Drop `views/` and `html/template` usage.
3. **API key management via Redis** — runtime create/revoke via admin API. Master admin key stays in config.
4. **Auth middleware reads from Redis** (not static map from config).
5. **Full observability** — Prometheus `/metrics`, OTEL traces (OTLP), structured zerolog JSON, Sentry.

## Conventions

- Errors wrapped with context, not swallowed. Use `fmt.Errorf("...: %w", err)`.
- No ORM in new code — Redis client calls directly in store layer.
- HTTP handlers return JSON always. Use `internal/response` package for envelope.
- Zerolog context logger via `zerolog.Ctx(ctx)` — propagate ctx everywhere.
- All new endpoints require auth unless explicitly public (health, metrics).
- State key format: `{type}.{k}` (e.g. `switch.modem_switch`). Type determines allowed values.
- RabbitMQ channels: one producer, one consumer per app module (see `hmalert/event.go`).

## Key files

- `main.go` — wiring, DI, graceful shutdown
- `internal/config/types.go` — all config structs (add `Redis`, `AdminKey` here)
- `conf.example.yaml` — update whenever config structs change
- `app/hmstt/store.go` — **replace this** with Redis implementation
- `app/hmstt/handlers.go` — **replace HTMX handlers** with JSON handlers
- `internal/middleware/auth.go` — **update** to look up keys from Redis
- `docs/` — detailed specs for each refactor area

## Dependency overview

```
Redis (state + apikeys)   RabbitMQ (event bus)   Discord (alerts)
      ↕                         ↕
   hmstt ←──event──→ hmalert ←── hmmon
      ↕                ↕
   HTTP server (gorilla/mux + middleware chain)
```

## What NOT to do

- Do not add GORM back or introduce a new SQL dependency.
- Do not return HTML from any endpoint (HTMX is gone).
- Do not store API keys only in config — they must be manageable at runtime via Redis.
- Do not skip zerolog context propagation in new handlers/services.
- Do not use `log.Fatal` outside of `main.go`.
