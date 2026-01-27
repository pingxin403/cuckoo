# {{SERVICE_NAME}}

{{SERVICE_DESCRIPTION}}

## Overview

This service is part of the monorepo and follows the standard Go gRPC service structure.

- **Port**: {{GRPC_PORT}}
- **Protocol**: gRPC
- **Language**: Go 1.21+
- **Team**: {{TEAM_NAME}}

## Features

- gRPC API with Protocol Buffers
- Configuration management via `libs/config`
- Full observability support (metrics, logging, tracing)
- Property-based testing with rapid
- 80% test coverage requirement

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run the service
go run .

# Run tests (unit tests only, fast)
go test ./...

# Run tests with property-based tests
go test ./... -tags=property

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
├── config/
│   ├── config.go           # Configuration loading logic
│   └── local/
│       └── config.yaml     # Local development configuration
├── service/
│   ├── template_service.go           # Service implementation
│   ├── template_service_test.go      # Unit tests
│   └── template_service_property_test.go  # Property-based tests
├── storage/
│   ├── memory_store.go     # In-memory storage implementation
│   └── memory_store_test.go
├── scripts/
│   └── test-coverage.sh    # Test coverage script
├── .apptype                # Application type identifier
├── .gitignore
├── .golangci.yml           # Linter configuration
├── catalog-info.yaml       # Backstage catalog information
├── Dockerfile              # Docker build file
├── go.mod                  # Go module definition
├── main.go                 # Application entry point
├── metadata.yaml           # Service metadata
├── README.md               # Documentation
└── TESTING.md              # Testing documentation
```

### Directory Descriptions

| Directory/File | Description |
|----------------|-------------|
| `config/` | Configuration-related code including config struct definitions and loading logic |
| `config/local/` | Local development configuration files |
| `service/` | gRPC service implementation and business logic |
| `storage/` | Data storage layer with interfaces and implementations |
| `scripts/` | Utility scripts for testing and development |

**Note**: Kubernetes resources are created separately in `deploy/k8s/services/{{SERVICE_NAME}}/` using templates from `templates/k8s/`.

## Template Placeholders

When creating a new service from this template, the following placeholders will be replaced:

| Placeholder | Description | Example Value |
|-------------|-------------|---------------|
| `{{SERVICE_NAME}}` | Service name (kebab-case) | `auth-service` |
| `{{ServiceName}}` | Service name (PascalCase) | `AuthService` |
| `{{SHORT_NAME}}` | Short name for CLI convenience | `auth` |
| `{{MODULE_PATH}}` | Go module path | `github.com/pingxin403/cuckoo/apps/auth-service` |
| `{{PROTO_PACKAGE}}` | Proto package name | `authpb` |
| `{{PROTO_FILE}}` | Proto file name (without extension) | `auth_service` |
| `{{GRPC_PORT}}` | gRPC service port | `9095` |
| `{{SERVICE_NAME_SNAKE}}` | Service name (snake_case) | `auth_service` |
| `{{SERVICE_DESCRIPTION}}` | Service description | `JWT authentication service` |
| `{{TEAM_NAME}}` | Team name | `platform-team` |

## Configuration

The service uses the unified configuration library (`libs/config`) for configuration management.

### Configuration Files

Configuration is loaded from the following locations (in order of precedence):
1. Environment variables
2. `./config.yaml`
3. `./config/config.yaml`
4. `./config/local/config.yaml` (for local development)

### Local Development Configuration

Edit `config/local/config.yaml` for local development:

```yaml
# {{SERVICE_NAME}} - Local Development Configuration

server:
  port: {{GRPC_PORT}}
  host: "0.0.0.0"

observability:
  service_name: "{{SERVICE_NAME}}"
  service_version: "1.0.0"
  environment: "local"
  enable_metrics: true
  metrics_port: 9090
  log_level: "debug"
  log_format: "text"

# Add service-specific configuration here
```

### Environment Variables

Supported environment variables:

#### Service Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | gRPC server port | {{GRPC_PORT}} |
| `HOST` | Server host address | 0.0.0.0 |

#### Observability Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `SERVICE_NAME` | Service name for observability | {{SERVICE_NAME}} |
| `SERVICE_VERSION` | Service version | 1.0.0 |
| `DEPLOYMENT_ENVIRONMENT` | Deployment environment | development |
| `ENABLE_METRICS` | Enable metrics collection | true |
| `METRICS_PORT` | Metrics HTTP server port | 9090 |
| `ENABLE_OTEL_METRICS` | Enable OpenTelemetry metrics export | false |
| `ENABLE_OTEL_LOGS` | Enable OpenTelemetry logs export | false |
| `ENABLE_OTEL_TRACING` | Enable OpenTelemetry tracing | false |
| `ENABLE_PROMETHEUS` | Enable Prometheus metrics export | true |
| `ENABLE_PPROF` | Enable pprof profiling endpoints | false |
| `OTLP_ENDPOINT` | OTLP collector endpoint | localhost:4317 |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |
| `LOG_FORMAT` | Log format (json, text) | json |

## API Definition

The service API is defined in `api/v1/{{PROTO_FILE}}.proto`.

To regenerate protobuf code:

```bash
# From monorepo root
make gen-proto-go
```

## Testing

### Unit Tests

Run unit tests with coverage:

```bash
# Run unit tests only (fast, no property tests)
go test ./...

# Run all tests including property-based tests
go test ./... -tags=property

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run tests with coverage verification (80% overall, 90% service)
./scripts/test-coverage.sh
```

### Coverage Requirements

The service enforces test coverage thresholds:
- **Overall coverage**: 80% minimum
- **Service package**: 90% minimum

These thresholds are verified in CI and will fail the build if not met.

### Property-Based Tests

The service uses property-based testing with `pgregory.net/rapid`:

```go
//go:build property
// +build property

package service

import (
    "testing"
    "pgregory.net/rapid"
)

func TestServiceProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random inputs
        input := rapid.String().Draw(t, "input")
        
        // Test properties
        // ...
    })
}
```

Property tests are separated using build tags to keep regular test runs fast. Run them with:

```bash
go test ./... -tags=property
```

For more details on property-based testing, see [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md).

### Integration Tests

Integration tests verify the service running in a real environment:

```bash
# Run integration tests (uses Docker Compose)
./scripts/run-integration-tests.sh
```

For more details, see the [Testing Guide](../../docs/development/TESTING_GUIDE.md).

## Observability

The service integrates with the unified observability library (`libs/observability`), providing metrics, structured logging, distributed tracing, and profiling capabilities.

### Metrics

The service exposes Prometheus metrics on port 9090 (configurable via `METRICS_PORT`):

```bash
curl http://localhost:9090/metrics
```

#### Available Metrics

**gRPC Request Metrics:**
- `{{SERVICE_NAME_SNAKE}}_grpc_requests_total{method, status}` - Counter of gRPC requests by method and status
- `{{SERVICE_NAME_SNAKE}}_grpc_request_duration_seconds{method}` - Histogram of gRPC request duration

**Service Operation Metrics:**
- `{{SERVICE_NAME_SNAKE}}_operations_total{operation, status}` - Counter of service operations

All metrics include standard labels:
- `service_name`: {{SERVICE_NAME}}
- `service_version`: Service version
- `environment`: Deployment environment

### Structured Logging

The service uses structured logging with JSON format by default:

```json
{
  "timestamp": "2024-01-25T10:30:00Z",
  "level": "info",
  "service": "{{SERVICE_NAME}}",
  "message": "Starting {{SERVICE_NAME}}",
  "version": "1.0.0"
}
```

Log levels can be configured via `LOG_LEVEL` environment variable:
- `debug`: Detailed debugging information
- `info`: General informational messages (default)
- `warn`: Warning messages
- `error`: Error messages

### OpenTelemetry Integration

The service supports OpenTelemetry for metrics, logs, and traces export to an OTLP collector:

```bash
# Enable OpenTelemetry metrics export
ENABLE_OTEL_METRICS=true OTLP_ENDPOINT=localhost:4317 go run .

# Enable OpenTelemetry logs export
ENABLE_OTEL_LOGS=true OTLP_ENDPOINT=localhost:4317 go run .

# Enable OpenTelemetry tracing
ENABLE_OTEL_TRACING=true OTLP_ENDPOINT=localhost:4317 go run .
```

The service will continue to function normally if the OTLP collector is unavailable.

### Profiling (pprof)

When enabled, the service exposes pprof endpoints for performance profiling:

```bash
# Enable pprof endpoints
ENABLE_PPROF=true go run .

# Access pprof endpoints
curl http://localhost:9090/debug/pprof/
curl http://localhost:9090/debug/pprof/heap
curl http://localhost:9090/debug/pprof/goroutine
curl http://localhost:9090/debug/pprof/profile?seconds=30
```

**Security Note**: pprof endpoints should only be enabled in development or controlled environments.

### Observability Stack

For local development with full observability stack (Prometheus, Jaeger, Grafana):

```bash
# Start observability stack
make observability-up

# Start service with OpenTelemetry enabled
ENABLE_OTEL_METRICS=true \
ENABLE_OTEL_LOGS=true \
ENABLE_OTEL_TRACING=true \
OTLP_ENDPOINT=localhost:4317 \
go run .

# Access observability UIs
# Prometheus: http://localhost:9091
# Jaeger: http://localhost:16686
# Grafana: http://localhost:3000
```

For more details, see:
- [Observability Library Documentation](../../libs/observability/README.md)
- [OpenTelemetry Guide](../../libs/observability/OPENTELEMETRY_GUIDE.md)
- [Observability Deployment](../../deploy/docker/OBSERVABILITY.md)

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

### Check Status

```bash
kubectl get pods -l app={{SERVICE_NAME}}
kubectl logs -f deployment/{{SERVICE_NAME}}
```

## Development

### Adding New RPC Methods

1. Update `api/v1/{{PROTO_FILE}}.proto`
2. Regenerate code: `make gen-proto-go`
3. Implement method in `service/template_service.go`
4. Add tests in `service/template_service_test.go`
5. Add property tests in `service/template_service_property_test.go`

### Storage Layer

The template includes an in-memory storage implementation. For production:

- Replace with PostgreSQL, MySQL, or other database
- Implement the storage interface in `storage/`
- Update tests accordingly

## Monitoring

The service exposes gRPC health checks for Kubernetes probes:

- Liveness probe: gRPC health check on port {{GRPC_PORT}}
- Readiness probe: gRPC health check on port {{GRPC_PORT}}

## Troubleshooting

### Protobuf Generation Fails

```bash
# Install protoc-gen-go and protoc-gen-go-grpc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Regenerate
make gen-proto-go
```

### Port Already in Use

Change the port using environment variable:

```bash
PORT=9096 go run .
```

### Build Fails

```bash
# Clean and rebuild
go clean
go mod tidy
go build .
```

### Configuration Not Loading

1. Check that `config/local/config.yaml` exists
2. Verify YAML syntax is correct
3. Check environment variable overrides

## Resources

- [Monorepo Documentation](../../docs/README.md)
- [API Documentation](../../api/v1/README.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Deployment Guide](../../docs/deployment/DEPLOYMENT_GUIDE.md)
- [Observability Library](../../libs/observability/README.md)
- [Configuration Library](../../libs/config/README.md)

## Support

For questions or issues:
- Check the monorepo documentation
- Contact {{TEAM_NAME}}
- Review existing services for examples
