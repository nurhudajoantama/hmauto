# Security

## Features

### API Key Authentication

All `/v1/*` endpoints require a Bearer token validated against Redis.

```bash
curl -H "Authorization: Bearer <api-key>" http://localhost:8080/v1/states
```

Admin endpoints (`/admin/*`) use a separate master key from config — never stored in Redis.

**Key generation:** `openssl rand -hex 32`

### Rate Limiting

Per-IP token bucket. Configurable in `conf.yaml`:

```yaml
security:
  rateLimitPerMin: 60
  rateLimitBurst: 10
```

### Request Size Limits

```yaml
security:
  maxRequestSize: 1048576  # 1MB
```

### Security Headers

Applied to all responses: `X-Frame-Options`, `X-Content-Type-Options`, `X-XSS-Protection`, `Content-Security-Policy`, `Referrer-Policy`, `Permissions-Policy`.

## Production Checklist

- [ ] `security.enableAuth: true`
- [ ] Strong admin key (`openssl rand -hex 32`)
- [ ] Appropriate rate limits
- [ ] Sentry DSN configured for error tracking
- [ ] OTEL endpoint configured for tracing
- [ ] Redis password set
- [ ] RabbitMQ credentials set

## Testing Auth

```bash
# Should return 401
curl http://localhost:8080/v1/states

# Should succeed
curl -H "Authorization: Bearer <key>" http://localhost:8080/v1/states
```

## Support

Report security issues privately to the maintainers.
