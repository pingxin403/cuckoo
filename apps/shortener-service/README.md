# URL Shortener Service

A high-performance URL shortening service built with Go and gRPC. This service provides URL shortening capabilities with custom short codes, expiration management, and multi-tier caching for optimal performance.

## Features

- **URL Shortening**: Generate short codes for long URLs (7-character Base62 codes)
- **Custom Short Codes**: Support for user-defined short codes (4-20 characters)
- **Expiration Management**: Optional expiration times for short links
- **Multi-Tier Caching**: L1 (Ristretto) + L2 (Redis) + MySQL for high performance
- **Request Coalescing**: Singleflight pattern to prevent cache stampede
- **HTTP Redirects**: Fast HTTP 302 redirects with proper status codes
- **Security**: URL validation, malicious pattern detection, security headers
- **Observability**: Prometheus metrics, structured logging, health checks
- **Graceful Degradation**: Continues operation even if Redis is unavailable

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ├─────────────────────────────────────┐
       │                                     │
       ▼                                     ▼
┌─────────────┐                      ┌─────────────┐
│ gRPC API    │                      │ HTTP API    │
│ (Port 9092) │                      │ (Port 8080) │
└──────┬──────┘                      └──────┬──────┘
       │                                     │
       │         ┌───────────────────────────┘
       │         │
       ▼         ▼
┌──────────────────────┐
│   Service Layer      │
│  - URL Validator     │
│  - ID Generator      │
│  - Error Handling    │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│   Cache Manager      │
│  - Singleflight      │
│  - L1: Ristretto     │
│  - L2: Redis         │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│   Storage Layer      │
│  - MySQL             │
└──────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- MySQL 8.0+
- Redis 6.0+ (optional, for caching)
- Protocol Buffers compiler (protoc)

### Installation

```bash
# Clone the repository
cd apps/shortener-service

# Install dependencies
go mod download

# Build the service
make build APP=shortener
```

### Configuration

Set the following environment variables:

```bash
# MySQL Configuration (Required)
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
export MYSQL_DATABASE=shortener
export MYSQL_USER=root
export MYSQL_PASSWORD=your_password

# Redis Configuration (Optional)
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

# Service Configuration
export GRPC_PORT=9092
export HTTP_PORT=8080
export METRICS_PORT=9090
export BASE_URL=http://localhost:8080

# Logging
export LOG_LEVEL=info  # debug, info, warn, error
export LOG_ENV=development  # development or production
```

### Database Setup

Run the migration script to create the database schema:

```bash
# Set MySQL environment variables first
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
export MYSQL_DATABASE=shortener
export MYSQL_USER=root
export MYSQL_PASSWORD=your_password

# Run migrations
./scripts/run-migrations.sh
```

### Running the Service

```bash
# Run locally
go run .

# Or use the monorepo command
make run APP=shortener

# Or run with Docker Compose (includes MySQL and Redis)
docker-compose -f docker-compose.test.yml up
```

## API Documentation

### gRPC API

The service exposes a gRPC API on port 9092. See the [Protocol Buffer definition](../../api/v1/shortener.proto) for details.

#### CreateShortLink

Creates a new short link.

```protobuf
rpc CreateShortLink(CreateShortLinkRequest) returns (CreateShortLinkResponse);

message CreateShortLinkRequest {
  string long_url = 1;           // Required: URL to shorten
  string custom_code = 2;        // Optional: Custom short code (4-20 chars)
  int64 expires_at = 3;          // Optional: Unix timestamp for expiration
}

message CreateShortLinkResponse {
  string short_code = 1;         // Generated or custom short code
  string short_url = 2;          // Full short URL
  int64 created_at = 3;          // Creation timestamp
  int64 expires_at = 4;          // Expiration timestamp (0 if no expiration)
}
```

**Example (grpcurl):**

```bash
# Create a short link
grpcurl -plaintext -d '{
  "long_url": "https://example.com/very/long/url"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink

# Create with custom code
grpcurl -plaintext -d '{
  "long_url": "https://example.com/page",
  "custom_code": "my-link"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink

# Create with expiration (7 days from now)
grpcurl -plaintext -d '{
  "long_url": "https://example.com/temp",
  "expires_at": 1737446400
}' localhost:9092 api.v1.ShortenerService/CreateShortLink
```

#### GetLinkInfo

Retrieves information about a short link.

```protobuf
rpc GetLinkInfo(GetLinkInfoRequest) returns (GetLinkInfoResponse);

message GetLinkInfoRequest {
  string short_code = 1;         // Required: Short code to look up
}

message GetLinkInfoResponse {
  string short_code = 1;
  string long_url = 2;
  int64 created_at = 3;
  int64 expires_at = 4;
  int32 click_count = 5;
  bool is_deleted = 6;
}
```

**Example:**

```bash
grpcurl -plaintext -d '{
  "short_code": "abc1234"
}' localhost:9092 api.v1.ShortenerService/GetLinkInfo
```

#### DeleteShortLink

Soft deletes a short link.

```protobuf
rpc DeleteShortLink(DeleteShortLinkRequest) returns (DeleteShortLinkResponse);

message DeleteShortLinkRequest {
  string short_code = 1;         // Required: Short code to delete
}

message DeleteShortLinkResponse {
  bool success = 1;
}
```

**Example:**

```bash
grpcurl -plaintext -d '{
  "short_code": "abc1234"
}' localhost:9092 api.v1.ShortenerService/DeleteShortLink
```

### HTTP API

The service exposes an HTTP redirect endpoint on port 8080.

#### Redirect Endpoint

```
GET /:code
```

Redirects to the long URL associated with the short code.

**Response Codes:**
- `302 Found`: Successful redirect to long URL
- `404 Not Found`: Short code does not exist
- `410 Gone`: Short link has expired or been deleted

**Example:**

```bash
# Redirect to long URL
curl -L http://localhost:8080/abc1234

# Get redirect without following (see Location header)
curl -I http://localhost:8080/abc1234
```

#### Health Check Endpoints

```
GET /health    # Liveness probe (always returns 200)
GET /ready     # Readiness probe (checks MySQL and Redis)
```

**Example:**

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Metrics Endpoint

Prometheus metrics are exposed on port 9090:

```
GET /metrics
```

**Available Metrics:**
- `shortener_requests_total`: Total number of requests by method and status
- `shortener_request_duration_seconds`: Request duration histogram
- `shortener_cache_hits_total`: Cache hits by layer (L1/L2)
- `shortener_cache_misses_total`: Cache misses by layer
- `shortener_errors_total`: Total errors by type
- `shortener_redirects_total`: Total redirects by status code
- `shortener_singleflight_waits_total`: Singleflight coalesced requests

**Example:**

```bash
curl http://localhost:9090/metrics
```

## Observability

The shortener-service uses the unified observability library (`libs/observability`) for structured logging, metrics, and tracing.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | `shortener-service` | Service name for observability |
| `SERVICE_VERSION` | `1.0.0` | Service version |
| `DEPLOYMENT_ENVIRONMENT` | `development` | Deployment environment |
| `ENABLE_METRICS` | `true` | Enable Prometheus metrics |
| `METRICS_PORT` | `9090` | Port for metrics endpoint |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | - | OTLP endpoint for telemetry export |

### Metrics

The service exposes the following Prometheus metrics on port 9090:

**Request Metrics:**
- `shortener_requests_total{method, status}` - Total requests by method and status
- `shortener_request_duration_seconds{method}` - Request duration histogram

**URL Operation Metrics:**
- `shortener_url_operations_total{operation, status}` - URL operations (create, resolve, delete)
- `shortener_operation_duration_seconds{operation}` - Operation duration histogram
- `shortener_links_created_total` - Total short links created
- `shortener_links_deleted_total` - Total short links deleted
- `shortener_redirects_total` - Total successful redirects

**Cache Metrics:**
- `shortener_cache_hits_total{layer}` - Cache hits by layer (L1, L2)
- `shortener_cache_misses_total{layer}` - Cache misses by layer
- `shortener_cache_operations_total{operation, layer}` - Cache operations (hit, miss)
- `shortener_singleflight_waits_total` - Singleflight coalesced requests

**Error Metrics:**
- `shortener_errors_total{type}` - Errors by type

**Analytics Metrics:**
- `shortener_click_events_logged_total` - Click events logged to Kafka

### Structured Logging

All logs are output in structured JSON format with the following fields:
- `timestamp` - ISO 8601 timestamp
- `level` - Log level (info, warn, error)
- `message` - Log message
- `service` - Service name
- `trace_id` - Trace ID (when tracing is enabled)
- Additional context fields

Example log output:
```json
{
  "timestamp": "2025-01-25T10:30:00Z",
  "level": "info",
  "message": "Short link created",
  "service": "shortener-service",
  "short_code": "abc1234",
  "long_url": "https://example.com/page",
  "creator_ip": "192.168.1.1"
}
```

### Health Checks

- `GET /health` - Liveness probe (always returns 200)
- `GET /ready` - Readiness probe (checks MySQL connectivity)

### Graceful Shutdown

The service implements graceful shutdown with a 5-second timeout for observability:
1. Receives SIGTERM/SIGINT signal
2. Stops accepting new requests
3. Completes in-flight requests
4. Flushes metrics and logs
5. Shuts down observability components

## Error Codes

The service uses standard gRPC status codes:

| Error Code | gRPC Status | Description |
|------------|-------------|-------------|
| INVALID_URL | InvalidArgument | URL is invalid or malicious |
| URL_TOO_LONG | InvalidArgument | URL exceeds 2048 characters |
| INVALID_CUSTOM_CODE | InvalidArgument | Custom code format is invalid |
| CUSTOM_CODE_RESERVED | InvalidArgument | Custom code is reserved |
| SHORT_CODE_NOT_FOUND | NotFound | Short code does not exist |
| SHORT_CODE_EXISTS | AlreadyExists | Custom code already in use |
| SHORT_CODE_EXPIRED | FailedPrecondition | Short link has expired |
| SHORT_CODE_DELETED | FailedPrecondition | Short link has been deleted |
| STORAGE_ERROR | Internal | Database operation failed |
| GENERATION_FAILED | Internal | Failed to generate short code |

## Testing

### Run All Tests

```bash
# Run all tests
make test APP=shortener

# Run tests with coverage
cd apps/shortener-service && ./scripts/test-coverage.sh

# Run specific package tests
go test ./service/...
go test ./storage/...
go test ./cache/...
```

### Test Coverage Requirements

- **Overall coverage**: 70% minimum
- **Service layer**: 75% minimum

### Property-Based Tests

The service includes property-based tests using the `rapid` framework:

```bash
# Run property tests (included in make test)
go test -v ./idgen/...
go test -v ./service/...
```

### Integration Tests

Integration tests verify the complete service functionality with real MySQL and Redis instances running in Docker containers.

```bash
# Run integration tests (automated script)
./scripts/run-integration-tests.sh

# Or manually:
# 1. Start test environment
docker compose -f docker-compose.test.yml up -d

# 2. Wait for services to be healthy (check with docker compose ps)
docker compose -f docker-compose.test.yml ps

# 3. Run integration tests
GRPC_ADDR="localhost:9092" BASE_URL="http://localhost:8081" \
  go test -v -tags=integration ./integration_test/... -timeout 5m

# 4. Stop test environment
docker compose -f docker-compose.test.yml down -v
```

**Integration Test Coverage:**
- ✅ End-to-end flow: Create → Retrieve → Redirect → Delete
- ✅ Custom short code functionality
- ✅ Link expiration handling (410 Gone)
- ✅ Cache warming and performance
- ✅ URL validation and security
- ✅ Concurrent creation
- ✅ Health check endpoints

**Test Environment:**
- MySQL 8.0 on port 3307
- Redis 7.2 on port 6380
- Shortener Service:
  - gRPC on port 9092
  - HTTP redirect on port 8081
  - Metrics on port 9091

## Development

### Monorepo Commands

```bash
# List all services
make list-apps

# Run tests
make test APP=shortener

# Build service
make build APP=shortener

# Run linter
make lint APP=shortener

# Auto-fix linting issues
make lint-fix APP=shortener

# Format code
make format APP=shortener

# Build Docker image
make docker-build APP=shortener

# Generate protobuf code
make proto

# Verify proto code is up to date
make verify-proto

# Run all pre-commit checks
make pre-commit
```

### Local Development with Docker Compose

```bash
# Start all dependencies (MySQL, Redis)
docker-compose -f docker-compose.test.yml up -d

# Run the service locally
go run .

# Stop dependencies
docker-compose -f docker-compose.test.yml down
```

### Code Quality

```bash
# Run linter
make lint APP=shortener

# Auto-fix linting issues
make lint-fix APP=shortener

# Format code
make format APP=shortener

# Check test coverage
cd apps/shortener-service && ./scripts/test-coverage.sh
```

## Deployment

### Kubernetes

Deploy to Kubernetes using the provided manifests:

```bash
# Create ConfigMap and Secret
kubectl apply -f k8s/configmap.yaml
kubectl create secret generic shortener-db-secret \
  --from-literal=mysql-password=your_password \
  --from-literal=redis-password=your_redis_password

# Deploy service
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Check status
kubectl get pods -l app=shortener-service
kubectl logs -f deployment/shortener-service

# Port forward for local testing
kubectl port-forward svc/shortener-service 9092:9092
kubectl port-forward svc/shortener-service 8080:8080
```

### Docker

```bash
# Build image
make docker-build APP=shortener

# Run container
docker run -p 9092:9092 -p 8080:8080 -p 9090:9090 \
  -e MYSQL_HOST=host.docker.internal \
  -e MYSQL_PORT=3306 \
  -e MYSQL_DATABASE=shortener \
  -e MYSQL_USER=root \
  -e MYSQL_PASSWORD=password \
  -e REDIS_ADDR=host.docker.internal:6379 \
  shortener-service:latest
```

## Performance

### Expected Performance

- **Redirect Latency**: P99 < 10ms (with warm cache)
- **Creation Latency**: P99 < 50ms
- **Throughput**: 500K+ QPS for redirects (with caching)
- **Cache Hit Rate**: >95% for L1+L2 combined

### Optimization Tips

1. **Enable Redis**: Significantly improves performance
2. **Tune Cache Sizes**: Adjust L1 cache size based on memory
3. **Connection Pooling**: Configure MySQL connection pool
4. **Horizontal Scaling**: Deploy multiple replicas
5. **Database Indexing**: Ensure indexes on `short_code` and `expires_at`

## Troubleshooting

### Service Won't Start

```bash
# Check MySQL connection
mysql -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USER -p$MYSQL_PASSWORD

# Check Redis connection (if using)
redis-cli -h localhost -p 6379 ping

# Check logs
kubectl logs -f deployment/shortener-service
```

### High Latency

1. Check cache hit rates in metrics
2. Verify Redis is running and accessible
3. Check MySQL query performance
4. Review connection pool settings

### Database Errors

```bash
# Run migrations
./scripts/run-migrations.sh

# Check database schema
mysql -h $MYSQL_HOST -u $MYSQL_USER -p$MYSQL_PASSWORD $MYSQL_DATABASE \
  -e "SHOW TABLES; DESCRIBE url_mappings;"
```

### Cache Issues

```bash
# Check Redis connectivity
redis-cli -h $REDIS_ADDR ping

# Monitor Redis
redis-cli -h $REDIS_ADDR monitor

# Clear Redis cache
redis-cli -h $REDIS_ADDR FLUSHDB
```

## Project Structure

```
shortener-service/
├── main.go                      # Application entry point
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── Dockerfile                   # Multi-stage Docker build
├── README.md                    # This file
├── metadata.yaml                # Service metadata
├── .apptype                     # App type marker for CI/CD
├── cache/                       # Caching layer
│   ├── cache_manager.go         # Multi-tier cache manager
│   ├── l1_cache.go              # L1 (Ristretto) cache
│   └── l2_cache.go              # L2 (Redis) cache
├── errors/                      # Error definitions
│   ├── errors.go                # Error types and codes
│   └── errors_test.go           # Error tests
├── idgen/                       # ID generation
│   ├── id_generator.go          # Short code generator
│   └── id_generator_test.go     # Generator tests
├── logger/                      # Structured logging
│   └── logger.go                # Zap logger wrapper
├── metrics/                     # Prometheus metrics
│   └── metrics.go               # Metrics definitions
├── migrations/                  # Database migrations
│   └── 001_create_url_mappings.sql
├── service/                     # Service layer
│   ├── shortener_service_service.go  # gRPC service implementation
│   ├── redirect_handler.go      # HTTP redirect handler
│   ├── url_validator.go         # URL validation
│   └── *_test.go                # Service tests
├── storage/                     # Storage layer
│   ├── mysql_store.go           # MySQL implementation
│   └── mysql_store_test.go      # Storage tests
├── scripts/                     # Utility scripts
│   ├── run-migrations.sh        # Database migration runner
│   └── test-coverage.sh         # Coverage verification
└── k8s/                         # Kubernetes manifests
    ├── deployment.yaml          # Deployment configuration
    ├── service.yaml             # Service configuration
    ├── configmap.yaml           # ConfigMap
    └── secret.yaml.template     # Secret template
```

## Additional Resources

- [Design Document](../../.kiro/specs/url-shortener-service/design.md)
- [Requirements Document](../../.kiro/specs/url-shortener-service/requirements.md)
- [Implementation Tasks](../../.kiro/specs/url-shortener-service/tasks.md)
- [API Documentation](./docs/API.md) - Complete API reference
- [Quick Start Guide](./QUICK_START.md) - Get started in 5 minutes
- [Integration Test Summary](./INTEGRATION_TEST_SUMMARY.md) - Test results and coverage
- [MVP Completion Summary](./MVP_COMPLETION_SUMMARY.md) - Feature checklist
- [Gateway Verification](./GATEWAY_VERIFICATION.md) - Envoy routing validation
- [Gateway Setup Summary](./GATEWAY_SETUP_SUMMARY.md) - Quick gateway setup guide
- [Monorepo Documentation](../../docs/)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers Guide](https://protobuf.dev/)

## Support

For questions or issues:
- Check the monorepo root README
- Review the design and requirements documents
- Contact the backend-go-team

## License

Copyright © 2025 Cuckoo Project
