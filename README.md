# hmauto

Home automation backend in Go. IoT state management with Redis storage and RabbitMQ event publishing.

## Features

- **State Management**: Track and update state for home automation components (switches, etc.)
- **Event Publishing**: State changes published to RabbitMQ `amq.topic` for external subscribers
- **Bearer Token Auth**: Single config-based bearer token for protected HTTP and MCP access
- **Health Monitoring**: Health checks for Redis and RabbitMQ dependencies
- **Observability**: Structured zerolog, OpenTelemetry tracing, Prometheus metrics, Sentry error tracking

## Quick Start

### Prerequisites

- Go 1.21+
- Redis
- RabbitMQ

### Configuration

```bash
cp conf.example.yaml conf/conf.yaml
# Edit conf/conf.yaml with your Redis, RabbitMQ, and security settings
```

### Running

```bash
go build -o hmauto .
./hmauto
# or with custom config path:
CONFIG_PATH=/path/to/config.yaml ./hmauto
```

## API Endpoints

### Public

- `GET /healthz` - Liveness
- `GET /health` - Dependency health (Redis + RabbitMQ)
- `GET /ready` - Readiness probe
- `GET /live` - Liveness probe
- `GET /metrics` - Prometheus scrape endpoint

### Protected (bearer token required)

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/states
```

- `GET /v1/states` - All states
- `GET /v1/states/{type}` - States by type
- `GET /v1/states/{type}/{key}` - Single state
- `PUT /v1/states/{type}/{key}` - Set state value
- `PATCH /v1/states/{type}/{key}` - Partially update state value/description
- `POST /mcp` - MCP endpoint using the same bearer token

See [docs/api-design.md](docs/api-design.md) for full API reference.

## Deployment

Build for Linux:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o hmauto-linux .
```

## License

MIT
