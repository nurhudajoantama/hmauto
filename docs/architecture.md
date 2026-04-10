# Architecture

## Stack

| Layer | Technology |
|---|---|
| HTTP server | gorilla/mux |
| Storage | Redis (persistent, AOF) — `go-redis/v9` |
| Message broker | RabbitMQ (AMQP) — `amqp091-go` |
| Logging | zerolog (structured JSON) |
| Tracing | OpenTelemetry (OTLP/gRPC) |
| Metrics | Prometheus (`/metrics`) |
| Error tracking | Sentry |

## HTTP middleware chain

```
[Every request]
  1. SecurityHeaders       — CSP, X-Frame-Options, etc.
  2. MaxBytesReader        — cap body at MaxRequestSize (default 1MB)
  3. hlog.NewHandler       — attach zerolog logger to ctx
  4. hlog.AccessHandler    — log method/url/status/duration
  5. hlog.RemoteAddrHandler — log client IP
  6. hlog.UserAgentHandler
  7. hlog.RefererHandler
  8. hlog.RequestIDHandler — X-Request-ID header + log field
  9. RateLimit             — per-IP token bucket
 10. PrometheusMiddleware  — http_requests_total, http_request_duration_seconds
 11. TraceIDMiddleware     — inject OTEL trace_id into zerolog ctx

[otelhttp.NewHandler wraps router — spans created here]
[sentryhttp wraps otelhttp — panics captured here]

[/v1/states/* subrouter]
  + BearerTokenAuth        — Bearer token == config.Security.BearerToken
```

## Routes

```
Public (no auth):
  GET  /healthz            → 200 "OK"
  GET  /health             → JSON {status, dependencies: {redis, rabbitmq}}
  GET  /ready              → 200/503 readiness probe
  GET  /live               → 200 liveness probe
  GET  /metrics            → Prometheus scrape endpoint

Protected (config bearer token):
  GET  /v1/states                → all states (all types)
  POST /v1/states                → create state
  GET  /v1/states/{type}         → all states for one type
  GET  /v1/states/{type}/{key}   → single state entry
  PUT  /v1/states/{type}/{key}   → set state value
  PATCH /v1/states/{type}/{key}  → patch state value/description
  POST /mcp                      → MCP streamable HTTP endpoint
```

## Redis key schema

```
State storage:
  Key type : Hash
  Key      : hmstt:{type}          e.g. hmstt:switch
  Field    : {k}                   e.g. modem_switch
  Value    : JSON {"value":"on","updated_at":"..."}
```

## RabbitMQ events

State changes are published to the `amq.topic` exchange with routing key `hmstt_channel.{full_key}` (e.g. `hmstt_channel.hmstt.switch.modem`). Payload is plain text (the new value). External subscribers can bind queues to this exchange.

## Module wiring (main.go)

```
config.InitializeConfig
sentry.Init
instrumentation.InitializeLogger (zerolog)
instrumentation.SetupOTelSDK (OTEL)
  ↓
redis.NewClient        ← state storage
rabbitmq.NewRabbitMQConn
  ↓
server.NewWithConfig   ← middleware chain assembled here
health.NewHealthChecker(rdb, mq) → /health, /ready, /live
  ↓
hmstt:   NewStore(rdb) + NewEvent + NewService + RegisterHandlers
  ↓
errgrp.Wait → graceful shutdown (5s timeout)
Close: http server, rabbitmq, redis, otel, logger
```
