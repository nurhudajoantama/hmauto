# Observability

Four pillars: structured logs, traces, metrics, error tracking.

## 1. Structured logging (zerolog)

Already in use. Target configuration:

- **Production**: `log.type: json`, `log.level: info`
- **Development**: `log.type: console`, `log.level: debug`
- Every handler gets logger from ctx: `zerolog.Ctx(ctx)`
- Standard fields to always include:
  - `request_id` — injected by `hlog.RequestIDHandler`
  - `ip` — injected by `hlog.RemoteAddrHandler`
  - Domain fields: `hmstt_type`, `hmstt_key`, `alert_level`, etc.

Never use `log.Print` or `fmt.Printf` in app code — always use `zerolog`.

## 2. OpenTelemetry traces

Skeleton already exists in `internal/instrumentation/otel.go`.

Target:
- Export to OTEL collector via OTLP/gRPC (`OTEL_EXPORTER_OTLP_ENDPOINT` env var)
- Wrap HTTP server handler with `otelhttp.NewHandler` (already done)
- Add spans for Redis operations in store layer:
  ```go
  ctx, span := otel.Tracer("hmstt").Start(ctx, "store.GetState")
  defer span.End()
  ```
- Propagate `ctx` through entire call chain (handlers → service → store)
- Config: OTEL endpoint via env var `OTEL_EXPORTER_OTLP_ENDPOINT` or config field

## 3. Prometheus metrics

New. Add `/metrics` endpoint.

Recommended library: `github.com/prometheus/client_golang/prometheus`

Metrics to expose:

```
# HTTP
http_requests_total{method, path, status}  counter
http_request_duration_seconds{method, path} histogram

# Business
hmstt_state_changes_total{type}            counter
hmalert_published_total{level}             counter
hmalert_discord_send_total{level, status}  counter

# Infrastructure
hmmon_internet_check_total{result}         counter  (ok/fail)
hmmon_modem_restart_total                  counter
```

Middleware approach:
```go
// internal/middleware/prometheus.go
func PrometheusMiddleware(next http.Handler) http.Handler {
    return promhttp.InstrumentHandlerDuration(
        httpDurationHistogram,
        promhttp.InstrumentHandlerCounter(httpRequestsTotal, next),
    )
}
```

Route: `GET /metrics` → `promhttp.Handler()` — public (no auth, scraper accesses it).

## 4. Sentry

Error and panic tracking.

Library: `github.com/getsentry/sentry-go`

Integration points:
- Init in `main.go` with DSN from config: `config.Sentry.DSN`
- Wrap HTTP handler with Sentry middleware: `sentryhttp.New(...).Handle(router)`
- Capture unexpected errors in service layer:
  ```go
  sentry.CaptureException(err)
  ```
- On panic: Sentry middleware captures automatically

Config addition in `types.go`:
```go
type Sentry struct {
    DSN         string  `yaml:"dsn"`
    Environment string  `yaml:"environment"` // production, staging
    SampleRate  float64 `yaml:"sampleRate"`  // 0.0-1.0 for traces
}
```

## Config additions for observability

```yaml
# conf.example.yaml additions

otel:
  endpoint: "localhost:4317"  # OTLP gRPC endpoint
  serviceName: "hmauto"
  enabled: true

sentry:
  dsn: "https://xxx@sentry.io/yyy"
  environment: "production"
  sampleRate: 0.1
```

## Log correlation

To correlate logs ↔ traces, inject trace ID into zerolog context:
```go
span := trace.SpanFromContext(ctx)
traceID := span.SpanContext().TraceID().String()
logger := zerolog.Ctx(ctx).With().Str("trace_id", traceID).Logger()
ctx = logger.WithContext(ctx)
```
