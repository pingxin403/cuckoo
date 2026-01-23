# TODO Service

TODO management service built with Go and gRPC.

## Overview

This service is part of the monorepo and provides TODO task management capabilities with inter-service communication.

- **Port**: 9091
- **Protocol**: gRPC
- **Language**: Go 1.21+
- **Team**: backend-go-team

## Features

- Create TODO items
- List all TODO items
- Update TODO items
- Delete TODO items
- Inter-service communication (calls Hello Service)
- Property-based testing with rapid
- 70% test coverage requirement

## Technology Stack

- Go 1.21+
- gRPC
- Protocol Buffers
- In-memory storage (extensible to persistent storage)

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)

### Local Development

```bash
# From project root
cd apps/todo-service

# Install dependencies
go mod download

# Run the service
go run .
```

The service will listen on port 9091.

### Environment Variables

- `PORT`: gRPC server port (default: 9091)
- `HELLO_SERVICE_ADDR`: Hello service address (default: localhost:9090)

### Build

```bash
# From monorepo root
make build APP=todo

# Or directly
go build -o bin/todo-service .

# Run
./bin/todo-service
```

### Docker

```bash
# Build image
docker build -t todo-service:latest .

# Run container
docker run -p 9091:9091 \
  -e HELLO_SERVICE_ADDR=host.docker.internal:9090 \
  todo-service:latest
```

## API

The service implements the following gRPC methods:

- `CreateTodo`: Create a new TODO item
- `ListTodos`: Get all TODO items
- `UpdateTodo`: Update an existing TODO item
- `DeleteTodo`: Delete a TODO item

For detailed API definitions, refer to `api/v1/todo.proto`.

## Testing

### Run All Tests

```bash
# Run unit tests only (fast)
go test ./...

# Run all tests including property-based tests
go test ./... -tags=property

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run coverage verification script
./scripts/test-coverage.sh
```

### Coverage Requirements

- **Overall coverage**: 70% minimum
- **Service/storage packages**: 80% minimum

### Property-Based Tests

The service uses property-based testing with `pgregory.net/rapid`. Property tests are separated using build tags:

```bash
# Run property tests
go test ./... -tags=property
```

For more details, see [TESTING.md](./TESTING.md) and [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md).

## Project Structure

```
apps/todo-service/
├── main.go              # Main entry point
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── Dockerfile           # Docker image build
├── README.md            # This file
├── TESTING.md           # Testing guide
├── metadata.yaml        # Service metadata
├── catalog-info.yaml    # Backstage catalog
├── .apptype             # App type marker
├── .golangci.yml        # Linter configuration
├── service/             # gRPC service implementation
│   ├── todo_service.go
│   ├── todo_service_test.go
│   └── todo_service_property_test.go
├── storage/             # Storage layer
│   ├── memory_store.go
│   └── memory_store_test.go
├── client/              # Hello service client
│   └── hello_client.go
├── gen/                 # Generated Protobuf code
│   ├── hellopb/
│   └── todopb/
└── scripts/
    ├── test-coverage.sh
    └── run-integration-tests.sh
```

## Deployment

### Kubernetes

```bash
# Apply Kubernetes resources
kubectl apply -k deploy/k8s/overlays/development

# Check deployment status
kubectl get pods -l app=todo-service
kubectl get svc todo-service
```

## Development

### Adding New Features

1. Update `api/v1/todo.proto` to define new messages or methods
2. Run `make gen-proto-go` to regenerate code
3. Implement new methods in `service/todo_service.go`
4. Add corresponding tests

### Code Standards

- Follow Go standard code style
- Use `gofmt` to format code
- Use `golangci-lint` for code checking

## Troubleshooting

### Service Won't Start

- Check if port 9091 is already in use
- Confirm Hello Service is running on port 9090 (if inter-service communication is needed)

### Failed to Connect to Hello Service

- Check `HELLO_SERVICE_ADDR` environment variable setting
- Confirm Hello Service is running and accessible

## Resources

- [Monorepo Documentation](../../docs/README.md)
- [API Documentation](../../api/v1/README.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Deployment Guide](../../docs/deployment/DEPLOYMENT_GUIDE.md)

## Support

For questions or issues:
- Check the monorepo documentation
- Contact backend-go-team
- Review existing services for examples
