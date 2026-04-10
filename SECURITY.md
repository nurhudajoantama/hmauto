# Security

## Features

### HTTP Bearer Token Authentication

All `/v1/*` endpoints require a Bearer token validated against `config.Security.BearerToken`.

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/states
```

### MCP Query Token Authentication

`/mcp` requires a query token validated against `config.Security.MCPToken`.

```bash
curl -X POST "http://localhost:8081/mcp?token=<mcp-token>"
```

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

- [ ] Strong `security.bearerToken` (`openssl rand -hex 32`)
- [ ] Strong `security.mcpToken` (`openssl rand -hex 32`)
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
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/states

# Should return 401
curl -X POST "http://localhost:8081/mcp"

# Should succeed
curl -X POST "http://localhost:8081/mcp?token=<mcp-token>"
```

## Support

Report security issues privately to the maintainers.
