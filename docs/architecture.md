# Architecture

## Current state (as-is)

```
HTTP (gorilla/mux, :8080)
  Middleware chain (global):
    SecurityHeaders → MaxBytesReader(1MB) → zerolog/hlog → RateLimit
  Middleware chain (protected subrouters):
    APIKeyAuth (static map from config)

Routes:
  Public:
    GET  /healthz              → 200 OK
    GET  /hello                → 200 OK
    GET  /health               → JSON dependency status
    GET  /ready, /live         → probe endpoints

  Protected (API key required):
    POST /hmalert/publish      → enqueue single alert to RabbitMQ
    POST /hmalert/publishbatch → enqueue batch alerts
    GET  /hmstt/               → HTMX HTML page (TO REMOVE)
    GET  /hmstt/statehtml/{type}/{key}    → HTMX fragment (TO REMOVE)
    POST /hmstt/setstatehtml/{type}/{key} → HTMX fragment (TO REMOVE)
    GET  /hmstt/getstatevalue/{type}/{key} → plain text value

Workers (errgroup):
  hmmon.internetWorker    → tick, ping internet, restart modem switch
  hmalert.consumerWorker  → consume RabbitMQ, call Discord webhook

Storage:
  PostgreSQL (GORM) → hmstt_states table
    PK: type.key (e.g. "switch.modem_switch")
    Fields: Key, Type, K, Title, Value, UpdatedAt

External:
  RabbitMQ  → hmstt_channel, hmalert queue
  Discord   → 3 webhook URLs (info/warning/error)
  PostgreSQL
```

## Target state (production)

```
HTTP (gorilla/mux, :8080)
  Middleware chain (global):
    SecurityHeaders → MaxBytesReader → zerolog/hlog → RequestID
    → RateLimit → Prometheus metrics middleware
  Middleware chain (protected):
    APIKeyAuth (Redis lookup)
  Middleware chain (admin):
    AdminKeyAuth (config lookup only)

Routes:
  Public:
    GET  /healthz, /ready, /live
    GET  /health          → JSON: redis + rabbitmq status
    GET  /metrics         → Prometheus scrape endpoint

  Protected (API key from Redis):
    POST /hmalert/publish
    POST /hmalert/publishbatch
    GET  /hmstt/state/{type}/{key}     → JSON {type, key, value, updated_at}
    POST /hmstt/state/{type}/{key}     → JSON body {value}
    GET  /hmstt/states/{type}          → JSON list of states for type
    GET  /hmstt/states                 → JSON all states

  Admin (master key from config):
    GET    /admin/apikeys              → list all keys (metadata only)
    POST   /admin/apikeys              → create key, returns generated key
    DELETE /admin/apikeys/{key}        → revoke key

Workers:
  hmmon.internetWorker  (unchanged)
  hmalert.consumerWorker (unchanged)

Storage:
  Redis (persistent, AOF enabled, no in-memory only mode)
    hmstt state:   HSET hmstt:{type}  {k}  {json: value+updated_at}
    api keys:      SET  apikey:{key}  {json: label, created_at, last_used}
                   SADD apikeys:index {key}    ← for listing
    (no TTL on state or apikeys — permanent until deleted)

External:
  RabbitMQ  (unchanged)
  Discord   (unchanged)
  Redis     (replaces Postgres)
  OTEL collector (traces via OTLP/gRPC)
  Prometheus scraper
  Sentry    (error/panic DSN)
```

## Middleware execution order (target)

```
[Every request]
  1. SecurityHeaders       - set CSP, X-Frame-Options, etc.
  2. MaxBytesReader        - cap body at MaxRequestSize
  3. hlog.NewHandler       - attach zerolog to ctx
  4. hlog.AccessHandler    - log method/url/status/duration
  5. hlog.RequestIDHandler - X-Request-ID header + log field
  6. RateLimit             - per-IP token bucket
  7. PrometheusMiddleware  - record http_requests_total, duration histogram

[Protected subrouter]
  8. APIKeyAuth            - Bearer token → Redis GET apikey:{key}

[Admin subrouter]
  8. AdminKeyAuth          - Bearer token == config.Security.AdminKey
```

## Module dependency graph

```
main.go
 ├─ config.InitializeConfig
 ├─ instrumentation.InitializeLogger (zerolog)
 ├─ instrumentation.SetupOTelSDK (OTEL)
 ├─ redis.NewClient              ← NEW (replaces postgres)
 ├─ rabbitmq.NewRabbitMQConn
 ├─ discord.NewDiscordWebhook x3
 ├─ server.NewWithConfig
 │    └─ middleware chain wired here
 ├─ hmalert.NewEvent + NewService + RegisterHandler + RegisterWorkers
 ├─ hmstt.NewStore(redis) + NewEvent + NewService + RegisterHandlers
 └─ hmmon.RegisterWorkers
```
