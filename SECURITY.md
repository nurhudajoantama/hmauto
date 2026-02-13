# Security Documentation

## Overview

This document describes the security features implemented in the application and how to configure them for production use.

## Security Features

### 1. API Key Authentication

All API endpoints (except `/healthz` and `/hello`) can be protected with API key authentication.

**Configuration:**
```yaml
security:
  enableAuth: true
  apiKeys:
    - "your-secure-api-key-here"
```

**Usage:**
```bash
# Make authenticated request
curl -H "Authorization: Bearer your-secure-api-key-here" \
  http://localhost:8080/hmalert/publish \
  -X POST \
  -d '{"level":"info","message":"test","tipe":"test"}'
```

**Best Practices:**
- Generate strong API keys: `openssl rand -hex 32`
- Rotate keys regularly
- Use different keys for different clients
- Store keys securely (environment variables, secrets manager)

### 2. Input Validation

All user inputs are validated and sanitized:

- **Alert levels**: Only `info`, `warning`, `error` are allowed
- **Message length**: Maximum 1000 characters
- **Type length**: Maximum 100 characters
- **Null bytes**: Automatically removed
- **Whitespace**: Trimmed from inputs

### 3. Rate Limiting

Prevents abuse by limiting requests per IP address.

**Configuration:**
```yaml
security:
  rateLimitPerMin: 60    # 60 requests per minute
  rateLimitBurst: 10     # Allow burst of 10 requests
```

### 4. Request Size Limits

Prevents DoS attacks by limiting request body size.

**Configuration:**
```yaml
security:
  maxRequestSize: 1048576  # 1MB in bytes
```

### 5. Security Headers

The following security headers are automatically added to all responses:

- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-Content-Type-Options: nosniff` - Prevents MIME sniffing
- `X-XSS-Protection: 1; mode=block` - Enables XSS protection
- `Content-Security-Policy` - Restricts resource loading
- `Referrer-Policy` - Controls referrer information
- `Permissions-Policy` - Disables unnecessary browser features

### 6. SSL/TLS for Database

Database connections support SSL/TLS encryption.

**Configuration:**
```yaml
db:
  sslMode: "require"  # Options: disable, require, verify-ca, verify-full
```

**Recommended for production:** `verify-full` with proper certificates

### 7. Fixed Race Conditions

The batch alert endpoint has been fixed to properly handle concurrent requests without race conditions:

- Uses WaitGroup to synchronize goroutines
- Validates all inputs before processing
- Limits batch size to 100 items
- Uses background context to prevent premature cancellation
- Returns proper error status if any alert fails

### 8. Error Handling

Improved error handling to prevent information disclosure:

- Generic error messages sent to clients
- Detailed errors logged for debugging
- Template errors don't expose internal paths

## Jenkins Deployment Security

The Jenkinsfile has been updated to remove `StrictHostKeyChecking=no`:

**Before (Insecure):**
```bash
ssh -o StrictHostKeyChecking=no user@host
```

**After (Secure):**
```bash
ssh user@host  # Uses known_hosts file
```

**Setup:**
1. Add server to known_hosts: `ssh-keyscan -H your-server.com >> ~/.ssh/known_hosts`
2. Ensure Jenkins has proper SSH credentials configured
3. Test connection before deployment

## Production Checklist

- [ ] Enable authentication: `security.enableAuth: true`
- [ ] Generate strong API keys
- [ ] Set database `sslMode: require` or higher
- [ ] Configure appropriate rate limits
- [ ] Set reasonable request size limits
- [ ] Review and test security headers
- [ ] Set up SSH known_hosts for deployments
- [ ] Use HTTPS/TLS for external connections
- [ ] Implement log monitoring and alerting
- [ ] Regular security audits
- [ ] Keep dependencies updated

## Testing Security Features

### Test Authentication

```bash
# Should fail without API key
curl http://localhost:8080/hmalert/publish

# Should succeed with valid API key
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/hmalert/publish \
  -X POST \
  -d '{"level":"info","message":"test","tipe":"test"}'
```

### Test Rate Limiting

```bash
# Send rapid requests to trigger rate limit
for i in {1..100}; do
  curl -H "Authorization: Bearer your-api-key" \
    http://localhost:8080/hmalert/publish \
    -X POST \
    -d '{"level":"info","message":"test '$i'","tipe":"test"}'
done
```

### Test Input Validation

```bash
# Should fail - invalid level
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/hmalert/publish \
  -X POST \
  -d '{"level":"invalid","message":"test","tipe":"test"}'

# Should fail - message too long
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/hmalert/publish \
  -X POST \
  -d '{"level":"info","message":"'$(python3 -c 'print("x"*1001)')' ","tipe":"test"}'
```

## Monitoring

Monitor these metrics for security:

- Failed authentication attempts
- Rate limit violations
- Request size rejections
- Input validation failures
- SSL/TLS connection errors

Configure alerts for suspicious patterns.

## Support

For security issues, please report them privately to the maintainers.
