# Auth Service

JWT authentication and token management service built with Go and gRPC.

## Overview

This service is part of the monorepo and provides JWT authentication and token management capabilities.

- **Port**: 9095
- **Protocol**: gRPC
- **Language**: Go 1.21+
- **Team**: platform-team

## Features

- JWT token generation and validation
- User authentication
- Token refresh mechanism
- gRPC API
- Property-based testing with rapid
- 80% test coverage requirement

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run the service
PORT=9095 go run .

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
make build APP=auth

# Or directly
go build -o bin/auth-service .
```

### Docker

```bash
# Build image
docker build -t auth-service:latest .

# Run container
docker run -p 9095:9095 auth-service:latest
```

## Project Structure

```
auth-service/
├── main.go                   # Application entry point
├── go.mod                    # Go module definition
├── Dockerfile                # Multi-stage Docker build
├── metadata.yaml             # Service metadata
├── catalog-info.yaml         # Backstage catalog
├── .apptype                  # App type marker
├── .golangci.yml             # Linter configuration
├── gen/                      # Generated Protobuf code
│   ├── auth_servicepb/
│   └── authpb/
├── service/                  # Service implementation
│   ├── auth_service.go
│   ├── auth_service_test.go
│   └── auth_service_property_test.go
└── scripts/
    └── test-coverage.sh      # Coverage verification
```

## API Definition

The service API is defined in `api/v1/auth.proto`.

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

Coverage reports are generated at:
- HTML: `coverage.html`
- Text: `coverage.out`

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

func TestAuthProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random inputs
        username := rapid.String().Draw(t, "username")
        
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

### Writing Tests

Example unit test structure:

```go
func TestAuthService_Authenticate(t *testing.T) {
    // Arrange
    service := NewAuthService()
    
    req := &authpb.AuthRequest{
        Username: "test-user",
        Password: "test-pass",
    }
    
    // Act
    resp, err := service.Authenticate(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.NotEmpty(t, resp.Token)
}
```

Use table-driven tests for multiple scenarios:

```go
func TestAuthService_Authenticate_Scenarios(t *testing.T) {
    tests := []struct {
        name     string
        username string
        password string
        wantErr  bool
    }{
        {"valid credentials", "user", "pass", false},
        {"empty username", "", "pass", true},
        {"empty password", "user", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

See `service/auth_service_test.go` for complete examples.

### Integration Tests

Integration tests verify the service running in a real environment:

```bash
# Run integration tests (uses Docker Compose)
./scripts/run-integration-tests.sh
```

For more details, see the [Testing Guide](../../docs/development/TESTING_GUIDE.md).

## Docker

### Build Image

```bash
docker build -t auth-service:latest .
```

### Run Container

```bash
docker run -p 9095:9095 auth-service:latest
```

## Kubernetes Deployment

### Deploy to Cluster

```bash
# Development
kubectl apply -k deploy/k8s/overlays/development

# Production
kubectl apply -k deploy/k8s/overlays/production
```

### Check Status

```bash
kubectl get pods -l app=auth-service
kubectl logs -f deployment/auth-service
```

## Configuration

### Environment Variables

Supported environment variables:

#### Service Configuration
- `PORT`: gRPC server port (default: 9095)
- `JWT_SECRET`: JWT signing secret (required)

#### Observability Configuration
- `SERVICE_NAME`: Service name for observability (default: auth-service)
- `SERVICE_VERSION`: Service version (default: 1.0.0)
- `DEPLOYMENT_ENVIRONMENT`: Deployment environment (default: development)
- `ENABLE_METRICS`: Enable metrics collection (default: true)
- `METRICS_PORT`: Metrics HTTP server port (default: 9090)
- `ENABLE_OTEL_METRICS`: Enable OpenTelemetry metrics export (default: false)
- `ENABLE_OTEL_LOGS`: Enable OpenTelemetry logs export (default: false)
- `ENABLE_OTEL_TRACING`: Enable OpenTelemetry tracing (default: false)
- `ENABLE_PROMETHEUS`: Enable Prometheus metrics export (default: true)
- `ENABLE_PPROF`: Enable pprof profiling endpoints (default: false)
- `OTLP_ENDPOINT`: OTLP collector endpoint (default: localhost:4317)
- `LOG_LEVEL`: Logging level - debug, info, warn, error (default: info)
- `LOG_FORMAT`: Log format - json or text (default: json)

## Observability

The auth-service integrates with the unified observability library, providing metrics, structured logging, distributed tracing, and profiling capabilities.

### Metrics

The service exposes Prometheus metrics on port 9090 (configurable via `METRICS_PORT`):

```bash
curl http://localhost:9090/metrics
```

#### Available Metrics

**Token Validation Metrics:**
- `auth_token_validations_total{status="success|failure", reason="..."}` - Counter of token validation attempts
  - Success: Token validated successfully
  - Failure reasons: `empty_token`, `expired`, `malformed`, `invalid_signature`, `invalid`, `invalid_claims`, `missing_user_id`, `missing_device_id`

**Token Generation Metrics:**
- `auth_token_generation_total{status="success|failure", reason="..."}` - Counter of token generation attempts
  - Success: Tokens generated successfully
  - Failure reasons: `empty_refresh_token`, `expired`, `malformed`, `invalid_signature`, `invalid`, `invalid_claims`, `signing_error`

**gRPC Request Metrics:**
- `auth_grpc_requests_total{method="ValidateToken|RefreshToken"}` - Counter of gRPC requests by method
- `auth_grpc_request_duration_seconds{method="ValidateToken|RefreshToken"}` - Histogram of gRPC request duration

All metrics include standard labels:
- `service_name`: auth-service
- `service_version`: Service version
- `environment`: Deployment environment

### Structured Logging

The service uses structured logging with JSON format by default:

```json
{
  "timestamp": "2024-01-25T10:30:00Z",
  "level": "info",
  "service": "auth-service",
  "message": "Starting auth-service",
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

# Start auth-service with OpenTelemetry enabled
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

## Development

### Adding New RPC Methods

1. Update `api/v1/auth.proto`
2. Regenerate code: `make gen-proto-go`
3. Implement method in `service/auth_service.go`
4. Add tests in `service/auth_service_test.go`
5. Add property tests in `service/auth_service_property_test.go`

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

## Resources

- [Monorepo Documentation](../../docs/README.md)
- [API Documentation](../../api/v1/README.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Deployment Guide](../../docs/deployment/DEPLOYMENT_GUIDE.md)

## Support

For questions or issues:
- Check the monorepo documentation
- Contact platform-team
- Review existing services for examples
