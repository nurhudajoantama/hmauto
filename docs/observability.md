# Observability

## Logging (zerolog)

- Structured JSON logs in production (`log.type: json`)
- Console format for development (`log.type: console`)
- Logger attached to every request context via `hlog.NewHandler`
- Access log: method, url, status, size, duration — via `hlog.AccessHandler`
- Fields always present: `request_id`, `ip`, `user_agent`
- Domain fields added per handler: `hmstt_type`, `hmstt_key`, `alert_level`, etc.
- Always use `zerolog.Ctx(ctx)` — never `log.Print` in app code

Config:
```yaml
log:
  level: "info"   # debug, info, warn, error
  type: "json"    # json (prod) | console (dev)
```

## Tracing (OpenTelemetry)

- OTLP/gRPC export when `otel.enabled: true` and `otel.endpoint` is set
- Falls back to stdout (pretty-print) when endpoint is not configured
- HTTP handler wrapped with `otelhttp.NewHandler` (spans created per request)
- Store-level spans in `app/hmstt/store.go`: `store.GetState`, `store.SetState`, etc.
- Trace ID injected into zerolog context via `TraceIDMiddleware` (field: `trace_id`)
- Propagation: W3C TraceContext + Baggage headers

Config:
```yaml
otel:
  endpoint: "localhost:4317"  # OTLP gRPC, e.g. Grafana Agent or OTel Collector
  serviceName: "hmauto"
  enabled: true
```

## Metrics (Prometheus)

Scraped at `GET /metrics` — public, no auth required.

### HTTP metrics (middleware/prometheus.go)

```
http_requests_total{method, path, status}       counter
http_request_duration_seconds{method, path}     histogram (default buckets)
```

### Business metrics

```
hmstt_state_changes_total{type}                 counter  (app/hmstt/service.go)
hmalert_published_total{level}                  counter  (app/hmalert/service.go)
```

### Recommended Grafana dashboard queries

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# p99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# State changes per type
rate(hmstt_state_changes_total[5m])
```

## Error tracking (Sentry)

- Panics and unhandled errors captured automatically via `sentryhttp` middleware
- Sentry wraps the full HTTP handler chain (outermost layer in `server.Start`)
- Manual capture available via `sentry.CaptureException(err)` for unexpected errors
- Init skipped if `sentry.dsn` is empty (safe for local dev)

Config:
```yaml
sentry:
  dsn: "https://xxx@sentry.io/yyy"
  environment: "production"
  sampleRate: 0.1   # 10% of transactions sampled for performance monitoring
```
