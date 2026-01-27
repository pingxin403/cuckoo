# {{SERVICE_NAME}}

{{SERVICE_DESCRIPTION}}

## Overview

This service is part of the monorepo and follows the standard Go gRPC service structure.

- **Port**: {{GRPC_PORT}}
- **Protocol**: gRPC
- **Language**: Go 1.21+
- **Team**: {{TEAM_NAME}}

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run the service
PORT={{GRPC_PORT}} go run .

# Run tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
```

### Build

```bash
# From monorepo root
make build APP={{SERVICE_NAME}}

# Or directly
go build -o bin/{{SERVICE_NAME}} .
```

### Docker

```bash
# Build image
docker build -t {{SERVICE_NAME}}:latest .

# Run container
docker run -p {{GRPC_PORT}}:{{GRPC_PORT}} {{SERVICE_NAME}}:latest
```

## Project Structure

```
{{SERVICE_NAME}}/
├── main.go                   # Application entry point
├── go.mod                    # Go module definition
├── Dockerfile                # Multi-stage Docker build
├── metadata.yaml             # Service metadata
├── catalog-info.yaml         # Backstage catalog
├── gen/                      # Generated Protobuf code
│   └── {{PROTO_PACKAGE}}/
├── service/                  # Service implementation
│   ├── {{SERVICE_NAME_SNAKE}}_service.go
│   └── {{SERVICE_NAME_SNAKE}}_service_test.go
└── storage/                  # Storage layer (if needed)
    ├── memory_store.go
    └── memory_store_test.go
```

**Note**: Kubernetes resources are created separately in `deploy/k8s/services/{{SERVICE_NAME}}/` using templates from `templates/k8s/`.

## API Definition

The service API is defined in `api/v1/{{PROTO_FILE}}.proto`.

To regenerate protobuf code:

```bash
# From monorepo root
make gen-proto-go
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Requirements

- Overall: 80% minimum
- Service/storage packages: 90% minimum

### Integration Tests

```bash
# From monorepo root
./scripts/run-integration-tests.sh
```

## Deployment

### Kubernetes

Kubernetes resources are located in `deploy/k8s/services/{{SERVICE_NAME}}/`:

- `{{SERVICE_NAME}}-deployment.yaml` - Deployment configuration
- `{{SERVICE_NAME}}-service.yaml` - Service configuration
- `kustomization.yaml` - Kustomize configuration

To deploy:

```bash
# Development
kubectl apply -k deploy/k8s/overlays/development

# Production
kubectl apply -k deploy/k8s/overlays/production
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | gRPC server port | {{GRPC_PORT}} |
| `SERVICE_VERSION` | Service version for metrics | 1.0.0 |
| `DEPLOYMENT_ENVIRONMENT` | Deployment environment | development |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |
| `ENABLE_OTEL_METRICS` | Enable OpenTelemetry metrics | false |
| `ENABLE_OTEL_LOGS` | Enable OpenTelemetry logs | false |
| `ENABLE_OTEL_TRACING` | Enable distributed tracing | false |
| `ENABLE_PROMETHEUS` | Enable Prometheus metrics export | true |
| `ENABLE_PPROF` | Enable pprof profiling endpoints | false |
| `OTLP_ENDPOINT` | OTLP collector endpoint | - |
| `METRICS_PORT` | Metrics HTTP server port | 9090 |

## Observability

This service uses the unified observability library (`libs/observability`) for metrics, logging, and tracing.

### Metrics

Metrics are exposed on port 9090 by default:

```bash
# View metrics
curl http://localhost:9090/metrics
```

**Available Metrics:**
- `{{SERVICE_NAME_SNAKE}}_grpc_requests_total{method, status}` - Total gRPC requests
- `{{SERVICE_NAME_SNAKE}}_grpc_request_duration_seconds{method}` - gRPC request latency
- `{{SERVICE_NAME_SNAKE}}_operations_total{operation, status}` - Service operations

### Logging

Structured JSON logging with trace correlation:

```json
{
  "level": "info",
  "ts": "2026-01-25T10:00:00Z",
  "msg": "Service started",
  "service": "{{SERVICE_NAME}}",
  "port": "{{GRPC_PORT}}"
}
```

### Tracing

Enable distributed tracing by setting:

```bash
ENABLE_OTEL_TRACING=true
OTLP_ENDPOINT=localhost:4317
```

### Profiling

Enable pprof for performance analysis:

```bash
ENABLE_PPROF=true

# CPU profile
go tool pprof http://localhost:9090/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:9090/debug/pprof/heap
```

## Development

### Adding New RPC Methods

1. Update `api/v1/{{PROTO_FILE}}.proto`
2. Regenerate code: `make gen-proto-go`
3. Implement method in `service/{{SERVICE_NAME_SNAKE}}_service.go`
4. Add tests in `service/{{SERVICE_NAME_SNAKE}}_service_test.go`

### Storage Layer

The template includes an in-memory storage implementation. For production:

- Replace with PostgreSQL, MySQL, or other database
- Implement the storage interface in `storage/`
- Update tests accordingly

## Monitoring

The service exposes gRPC health checks for Kubernetes probes:

- Liveness probe: gRPC health check on port {{GRPC_PORT}}
- Readiness probe: gRPC health check on port {{GRPC_PORT}}

## Resources

- [Monorepo Documentation](../../docs/README.md)
- [API Documentation](../../api/v1/README.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [Deployment Guide](../../docs/deployment/DEPLOYMENT_GUIDE.md)

## Support

For questions or issues:
- Check the monorepo documentation
- Contact {{TEAM_NAME}}
- Review existing services for examples
