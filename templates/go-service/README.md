# Go gRPC Service Template

This template provides a standardized structure for creating new Go gRPC services in the monorepo.

## Features

- Go 1.21+ with gRPC support
- Protobuf code generation
- In-memory storage with interface for easy extension
- Graceful shutdown handling
- Kubernetes deployment configurations
- Backstage service catalog integration
- Docker multi-stage build
- Health checks and monitoring

## Quick Start

### 1. Copy Template

```bash
# From the monorepo root
cp -r templates/go-service apps/your-service-name
cd apps/your-service-name
```

### 2. Customize Configuration

Replace the following placeholders throughout the project:

- `{{SERVICE_NAME}}` → Your service name (e.g., `user-service`)
- `{{SERVICE_DESCRIPTION}}` → Brief description of your service
- `{{GRPC_PORT}}` → gRPC port number (e.g., `9092`)
- `{{MODULE_PATH}}` → Go module path (e.g., `github.com/myorg/myrepo/apps/user-service`)
- `{{PROTO_FILE}}` → Protobuf file name (e.g., `user.proto`)
- `{{PROTO_PACKAGE}}` → Protobuf package name (e.g., `userpb`)
- `{{TEAM_NAME}}` → Owning team name (e.g., `backend-team`)

### 3. Update Files

#### go.mod
```go
module {{MODULE_PATH}}

go 1.21

require (
    github.com/google/uuid v1.6.0
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
)
```

#### main.go
Update the import paths and port:
```go
import (
    "{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
    "{{MODULE_PATH}}/service"
    "{{MODULE_PATH}}/storage"
)

// Update port
port := os.Getenv("PORT")
if port == "" {
    port = "{{GRPC_PORT}}"
}
```

### 4. Define Protobuf API

Create your service's Protobuf definition in `api/v1/{{PROTO_FILE}}`:

```protobuf
syntax = "proto3";

package api.v1;

option go_package = "github.com/myorg/myrepo/apps/{{SERVICE_NAME}}/gen/{{PROTO_PACKAGE}}";

service {{ServiceName}}Service {
  rpc YourMethod(YourRequest) returns (YourResponse);
}

message YourRequest {
  string field = 1;
}

message YourResponse {
  string result = 1;
}
```

### 5. Generate Protobuf Code

```bash
# From monorepo root
make gen-proto-go

# Or manually
protoc --go_out=apps/{{SERVICE_NAME}}/gen \
       --go_opt=paths=source_relative \
       --go-grpc_out=apps/{{SERVICE_NAME}}/gen \
       --go-grpc_opt=paths=source_relative \
       -I api/v1 \
       api/v1/{{PROTO_FILE}}
```

### 6. Implement Service Logic

Update `service/{{SERVICE_NAME}}_service.go`:

```go
package service

import (
    "context"
    "{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
    "{{MODULE_PATH}}/storage"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type {{ServiceName}}ServiceServer struct {
    {{PROTO_PACKAGE}}.Unimplemented{{ServiceName}}ServiceServer
    store storage.YourStore
}

func New{{ServiceName}}ServiceServer(store storage.YourStore) *{{ServiceName}}ServiceServer {
    return &{{ServiceName}}ServiceServer{
        store: store,
    }
}

func (s *{{ServiceName}}ServiceServer) YourMethod(ctx context.Context, req *{{PROTO_PACKAGE}}.YourRequest) (*{{PROTO_PACKAGE}}.YourResponse, error) {
    // Implement your logic here
    
    return &{{PROTO_PACKAGE}}.YourResponse{
        Result: "Your result",
    }, nil
}
```

### 7. Implement Storage Layer

Update `storage/memory_store.go` or create your own storage implementation:

```go
package storage

import (
    "sync"
    "{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
)

type YourStore interface {
    Create(item *{{PROTO_PACKAGE}}.YourItem) error
    Get(id string) (*{{PROTO_PACKAGE}}.YourItem, error)
    List() ([]*{{PROTO_PACKAGE}}.YourItem, error)
    Update(item *{{PROTO_PACKAGE}}.YourItem) error
    Delete(id string) error
}

type MemoryStore struct {
    mu    sync.RWMutex
    items map[string]*{{PROTO_PACKAGE}}.YourItem
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        items: make(map[string]*{{PROTO_PACKAGE}}.YourItem),
    }
}

// Implement interface methods...
```

### 8. Update Kubernetes Resources

Update the following files in `k8s/`:

- `deployment.yaml`: Update service name, port, and resource limits
- `service.yaml`: Update service name and port

### 9. Update Backstage Catalog

Edit `catalog-info.yaml`:

```yaml
metadata:
  name: {{SERVICE_NAME}}
  description: {{SERVICE_DESCRIPTION}}
  tags:
    - go
    - grpc
spec:
  owner: {{TEAM_NAME}}
  providesApis:
    - {{SERVICE_NAME}}-api
```

### 10. Build and Test

```bash
# Download dependencies
go mod download

# Build the service
go build -o bin/{{SERVICE_NAME}} .

# Run tests
go test ./...

# Run locally
PORT={{GRPC_PORT}} go run .

# Build Docker image
docker build -t {{SERVICE_NAME}}:latest .
```

### 11. Add to Monorepo Build

Update the root `Makefile`:

```makefile
build-{{SERVICE_NAME}}:
	@echo "Building {{SERVICE_NAME}}..."
	cd apps/{{SERVICE_NAME}} && go build -o bin/{{SERVICE_NAME}} .

test-{{SERVICE_NAME}}:
	@echo "Testing {{SERVICE_NAME}}..."
	cd apps/{{SERVICE_NAME}} && go test ./...
```

## Project Structure

```
your-service-name/
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── main.go                   # Application entry point
├── Dockerfile                # Multi-stage Docker build
├── catalog-info.yaml         # Backstage service catalog
├── README.md                 # Service documentation
├── gen/                      # Generated Protobuf code
│   └── {{PROTO_PACKAGE}}/
├── service/                  # Service implementation
│   └── {{SERVICE_NAME}}_service.go
├── storage/                  # Storage layer
│   └── memory_store.go
└── k8s/                      # Kubernetes resources
    ├── deployment.yaml
    └── service.yaml
```

## Dependencies

The template includes:

- **gRPC 1.60.0**: gRPC framework
- **Protobuf 1.32.0**: Protocol Buffers runtime
- **UUID 1.6.0**: UUID generation

## Configuration

### Environment Variables

Supported environment variables:

- `PORT`: gRPC server port (default: `{{GRPC_PORT}}`)
- `LOG_LEVEL`: Logging level (default: `info`)

### Service Configuration

The service uses environment variables for configuration. No configuration files are needed.

## Testing

### Unit Tests

Run unit tests with coverage:

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run tests with coverage verification (80% overall, 90% service/storage)
./scripts/test-coverage.sh
```

Coverage reports are generated at:
- HTML: `coverage.html`
- Text: `coverage.out`

### Coverage Requirements

The template enforces test coverage thresholds:
- **Overall coverage**: 80% minimum
- **Service/storage packages**: 90% minimum

These thresholds are verified in CI and will fail the build if not met.

### Writing Tests

Example unit test structure:

```go
func TestYourService_YourMethod(t *testing.T) {
    // Arrange
    store := storage.NewMemoryStore()
    service := NewYourService(store)
    
    req := &yourpb.YourRequest{
        Field: "test-value",
    }
    
    // Act
    resp, err := service.YourMethod(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, "expected-value", resp.Result)
}
```

Use table-driven tests for multiple scenarios:

```go
func TestYourService_YourMethod_Scenarios(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
        {"special chars", "test@#$", "result", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

See `service/template_service_test.go` and `storage/memory_store_test.go` for complete examples.

### Concurrent Testing

Test thread safety with goroutines:

```go
func TestYourStore_ConcurrentAccess(t *testing.T) {
    store := NewMemoryStore()
    const numGoroutines = 100
    var wg sync.WaitGroup
    
    wg.Add(numGoroutines)
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer wg.Done()
            // Concurrent operations
        }(i)
    }
    wg.Wait()
    
    // Verify results
}
```

### Property-Based Tests

For property-based testing in Go, use libraries like:
- [gopter](https://github.com/leanovate/gopter)
- [rapid](https://github.com/flyingmutant/rapid)

Example with rapid:

```go
import "pgregory.net/rapid"

func TestYourProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        // Test your property
        result := processInput(input)
        assert.NotNil(t, result)
    })
}
```

### Integration Tests

Integration tests verify the service running in a real environment with Docker:

```bash
# Run integration tests (uses root docker-compose.yml)
./scripts/run-integration-tests.sh
```

The integration test script:
1. Builds the service Docker image
2. Starts required dependencies (databases, caches, etc.)
3. Starts the service container
4. Waits for all services to be healthy
5. Runs integration tests against the running service
6. Cleans up containers automatically

Example integration test:

```go
package integration_test

import (
    "context"
    "os"
    "testing"
    "time"

    yourpb "{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

var grpcAddr = getEnv("GRPC_ADDR", "localhost:{{GRPC_PORT}}")

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func setupClient(t *testing.T) (yourpb.YourServiceClient, *grpc.ClientConn) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, err := grpc.DialContext(ctx, grpcAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }

    return yourpb.NewYourServiceClient(conn), conn
}

func TestEndToEndFlow(t *testing.T) {
    client, conn := setupClient(t)
    defer func() {
        if err := conn.Close(); err != nil {
            t.Logf("Failed to close connection: %v", err)
        }
    }()

    ctx := context.Background()

    // Test your service end-to-end
    resp, err := client.YourMethod(ctx, &yourpb.YourRequest{
        Field: "test-value",
    })

    if err != nil {
        t.Fatalf("YourMethod failed: %v", err)
    }

    if resp.Result != "expected-result" {
        t.Errorf("Expected 'expected-result', got '%s'", resp.Result)
    }
}
```

Create the test runner script at `scripts/run-integration-tests.sh`:

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

cd "$PROJECT_DIR"

# Build and start service
docker compose build {{SERVICE_NAME}}
docker compose up -d {{SERVICE_NAME}}

# Wait for service to be healthy
echo "Waiting for service to be ready..."
sleep 5

# Run tests
cd "apps/{{SERVICE_NAME}}"
GRPC_ADDR="localhost:{{GRPC_PORT}}" go test -v ./integration_test/... -count=1 -timeout 5m

# Cleanup
cd "$PROJECT_DIR"
docker compose stop {{SERVICE_NAME}}
```

Make the script executable:
```bash
chmod +x scripts/run-integration-tests.sh
```

### Integration Tests

```bash
go test -tags=integration ./...
```

For more details, see the [Testing Guide](../../docs/TESTING_GUIDE.md).

## Docker

### Build Image

```bash
docker build -t {{SERVICE_NAME}}:latest .
```

### Run Container

```bash
docker run -p {{GRPC_PORT}}:{{GRPC_PORT}} {{SERVICE_NAME}}:latest
```

## Kubernetes Deployment

### Deploy to Cluster

```bash
kubectl apply -f k8s/
```

### Check Status

```bash
kubectl get pods -l app={{SERVICE_NAME}}
kubectl logs -f deployment/{{SERVICE_NAME}}
```

## Best Practices

1. **Keep Services Small**: Focus on a single domain or capability
2. **Use Protobuf**: Define all APIs in Protobuf for type safety
3. **Add Tests**: Write both unit tests and property-based tests
4. **Document APIs**: Add clear comments to Protobuf definitions
5. **Handle Errors**: Use appropriate gRPC status codes
6. **Log Appropriately**: Use structured logging with context
7. **Graceful Shutdown**: Always implement graceful shutdown
8. **Version APIs**: Use semantic versioning for breaking changes

## Storage Options

The template includes an in-memory storage implementation. For production:

### PostgreSQL

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    return &PostgresStore{db: db}, nil
}
```

### Redis

```go
import "github.com/go-redis/redis/v8"

type RedisStore struct {
    client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
    return &RedisStore{
        client: redis.NewClient(&redis.Options{
            Addr: addr,
        }),
    }
}
```

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
PORT=9093 go run .
```

### Build Fails

```bash
# Clean and rebuild
go clean
go mod tidy
go build .
```

## Additional Resources

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Go Protobuf Guide](https://protobuf.dev/getting-started/gotutorial/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Backstage Service Catalog](https://backstage.io/docs/features/software-catalog/)

## Support

For questions or issues:
- Check the monorepo root README
- Contact the platform team
- Review existing services for examples
