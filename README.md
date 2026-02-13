# hmauto

A production-ready Go application for home automation with comprehensive security features.

## Features

- **Alert Management**: Publish and manage alerts with different severity levels (info, warning, error)
- **State Management**: Track and update state for various home automation components
- **Health Monitoring**: Comprehensive health checks for all dependencies
- **Security**: API key authentication, rate limiting, input validation, and security headers
- **Observability**: Structured logging with zerolog and OpenTelemetry integration

## Quick Start

### Prerequisites

- Go 1.25.0 or higher
- PostgreSQL database
- RabbitMQ message broker

### Configuration

1. Copy the example configuration:
```bash
cp conf.example.yaml conf/conf.yaml
```

2. Edit `conf/conf.yaml` with your settings:
   - Database credentials and SSL mode
   - RabbitMQ connection details
   - Discord webhook URLs
   - Security settings (API keys, rate limits)

### Running the Application

```bash
# Build
go build -o hmauto .

# Run
./hmauto
```

Or use a custom config path:
```bash
CONFIG_PATH=/path/to/config.yaml ./hmauto
```

## Security Features

This application implements production-grade security features:

- **API Key Authentication**: Protect endpoints with Bearer token authentication
- **Rate Limiting**: Prevent abuse with configurable per-IP rate limits
- **Input Validation**: All inputs are validated and sanitized
- **Request Size Limits**: Prevent DoS attacks with configurable size limits
- **Security Headers**: Comprehensive HTTP security headers (CSP, X-Frame-Options, etc.)
- **SSL/TLS Support**: Encrypted database and message broker connections

See [SECURITY.md](SECURITY.md) for detailed security documentation.

## API Endpoints

### Public Endpoints

- `GET /healthz` - Basic health check
- `GET /health` - Detailed health check with dependency status
- `GET /ready` - Readiness probe
- `GET /live` - Liveness probe
- `GET /hello` - Simple hello endpoint

### Protected Endpoints (require API key)

- `POST /hmalert/publish` - Publish a single alert
- `POST /hmalert/publishbatch` - Publish multiple alerts
- `GET /hmstt/` - View state management UI
- `GET /hmstt/getstatevalue/{type}/{key}` - Get state value
- `POST /hmstt/setstatehtml/{type}/{key}` - Set state value

### Authentication

Protected endpoints require an API key in the Authorization header:

```bash
curl -H "Authorization: Bearer your-api-key-here" \
  http://localhost:8080/hmalert/publish \
  -X POST \
  -d '{"level":"info","message":"Test alert","tipe":"test"}'
```

## Development

### Building

```bash
go build -o hmauto .
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Deployment

### Jenkins CI/CD

A Jenkinsfile is included for automated builds and deployments. Configure the following environment variables in Jenkins:

- `REMOTE_HOST`: Deployment target hostname
- `REMOTE_USER`: SSH user for deployment
- `REMOTE_PATH`: Installation path on remote host
- `SSH_CREDENTIALS_ID`: Jenkins SSH credentials ID
- `SERVICE_NAME`: Systemd service name

### Manual Deployment

1. Build for Linux:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o hmstt-linux .
```

2. Copy to server and set up systemd service
3. Configure and start the service

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please ensure:

- Code follows Go best practices
- Security considerations are addressed
- Tests are included for new features
- Documentation is updated

For security issues, please report them privately to the maintainers.