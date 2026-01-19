# Go gRPC Service UshortenerUservice

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

### 1. Copy UshortenerUservice

```bash
# From the monorepo root
cp -r templates/go-service apps/your-service-name
cd apps/your-service-name
```

### 2. Customize Configuration

Replace the following placeholders throughout the project:

- `shortener-service` → Your service name (e.g., `user-service`)
- `High-performance URL shortening service` → Brief description of your service
- `9092` → gRPC port number (e.g., `9092`)
- `github.com/pingxin403/cuckoo/apps/shortener-service` → Go module path (e.g., `github.com/myorg/myrepo/apps/user-service`)
- `shortener_service` → Protobuf file name (e.g., `user.proto`)
- `shortener_servicepb` → Protobuf package name (e.g., `userpb`)
- `backend-go-team` → Owning team name (e.g., `backend-team`)

### 3. Update Files

#### go.mod
```go
module github.com/pingxin403/cuckoo/apps/shortener-service

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
    "github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
    "github.com/pingxin403/cuckoo/apps/shortener-service/service"
    "github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// Update port
port := os.Getenv("PORT")
if port == "" {
    port = "9092"
}
```

### 4. Define Protobuf API

Create your service's Protobuf definition in `api/v1/shortener_service`:

```protobuf
syntax = "proto3";

package api.v1;

option go_package = "github.com/myorg/myrepo/apps/shortener-service/gen/shortener_servicepb";

service UshortenerUserviceService {
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
protoc --go_out=apps/shortener-service/gen \
       --go_opt=paths=source_relative \
       --go-grpc_out=apps/shortener-service/gen \
       --go-grpc_opt=paths=source_relative \
       -I api/v1 \
       api/v1/shortener_service
```

### 6. Implement Service Logic

Update `service/shortener-service_service.go`:

```go
package service

import (
    "context"
    "github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
    "github.com/pingxin403/cuckoo/apps/shortener-service/storage"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type UshortenerUserviceServiceServer struct {
    shortener_servicepb.UnimplementedUshortenerUserviceServiceServer
    store storage.YourStore
}

func NewUshortenerUserviceServiceServer(store storage.YourStore) *UshortenerUserviceServiceServer {
    return &UshortenerUserviceServiceServer{
        store: store,
    }
}

func (s *UshortenerUserviceServiceServer) YourMethod(ctx context.Context, req *shortener_servicepb.YourRequest) (*shortener_servicepb.YourResponse, error) {
    // Implement your logic here
    
    return &shortener_servicepb.YourResponse{
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
    "github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
)

type YourStore interface {
    Create(item *shortener_servicepb.YourItem) error
    Get(id string) (*shortener_servicepb.YourItem, error)
    List() ([]*shortener_servicepb.YourItem, error)
    Update(item *shortener_servicepb.YourItem) error
    Delete(id string) error
}

type MemoryStore struct {
    mu    sync.RWMutex
    items map[string]*shortener_servicepb.YourItem
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        items: make(map[string]*shortener_servicepb.YourItem),
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
  name: shortener-service
  description: High-performance URL shortening service
  tags:
    - go
    - grpc
spec:
  owner: backend-go-team
  providesApis:
    - shortener-service-api
```

### 10. Build and Test

```bash
# Download dependencies
go mod download

# Build the service
go build -o bin/shortener-service .

# Run tests
go test ./...

# Run locally
PORT=9092 go run .

# Build Docker image
docker build -t shortener-service:latest .
```

### 11. Add to Monorepo Build

Update the root `Makefile`:

```makefile
build-shortener-service:
	@echo "Building shortener-service..."
	cd apps/shortener-service && go build -o bin/shortener-service .

test-shortener-service:
	@echo "Testing shortener-service..."
	cd apps/shortener-service && go test ./...
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
│   └── shortener_servicepb/
├── service/                  # Service implementation
│   └── shortener-service_service.go
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

- `PORT`: gRPC server port (default: `9092`)
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

```bash
go test -tags=integration ./...
```

For more details, see the [Testing Guide](../../docs/TESTING_GUIDE.md).

## Docker

### Build Image

```bash
docker build -t shortener-service:latest .
```

### Run Container

```bash
docker run -p 9092:9092 shortener-service:latest
```

## Kubernetes Deployment

### Deploy to Cluster

```bash
kubectl apply -f k8s/
```

### Check Status

```bash
kubectl get pods -l app=shortener-service
kubectl logs -f deployment/shortener-service
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
