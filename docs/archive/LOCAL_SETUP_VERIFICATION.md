# Local Setup Verification Guide

This document provides step-by-step instructions for verifying your local development environment for the Monorepo Hello/TODO Services project.

## Prerequisites

Before running the verification, ensure you have the following installed:

- **Java 17+** (for Hello Service)
- **Go 1.21+** (for TODO Service)
- **Node.js 18+** (for Frontend)
- **Protocol Buffers Compiler** (`protoc`)
- **Envoy Proxy** (optional, for API gateway testing)
- **grpcurl** (optional, for gRPC testing)

## Quick Start

### 1. Start All Services

```bash
# From the project root
./scripts/dev.sh
```

This script will:
- Start Hello Service on port 9090
- Start TODO Service on port 9091
- Start Envoy Proxy on port 8080 (if installed)
- Start Frontend on port 5173

Wait for all services to start (you'll see "All services are running" message).

### 2. Run Automated Tests

In a new terminal:

```bash
./scripts/test-services.sh
```

This will run automated tests for:
- Service availability
- Hello Service functionality
- TODO Service CRUD operations
- Service-to-service communication
- Frontend accessibility

## Manual Testing

### Test Hello Service

#### Using grpcurl (Recommended)

```bash
# Test with a name
grpcurl -plaintext -d '{"name":"Alice"}' localhost:9090 api.v1.HelloService/SayHello

# Expected output:
# {
#   "message": "Hello, Alice!"
# }

# Test with empty name
grpcurl -plaintext -d '{"name":""}' localhost:9090 api.v1.HelloService/SayHello

# Expected output:
# {
#   "message": "Hello, World!"
# }
```

#### Using the Frontend

1. Open http://localhost:5173 in your browser
2. Enter a name in the Hello form
3. Click "Say Hello"
4. Verify the greeting message appears

### Test TODO Service

#### Using grpcurl

```bash
# Create a TODO
grpcurl -plaintext -d '{"title":"Buy groceries","description":"Milk, eggs, bread"}' \
  localhost:9091 api.v1.TodoService/CreateTodo

# Expected output:
# {
#   "todo": {
#     "id": "...",
#     "title": "Buy groceries",
#     "description": "Milk, eggs, bread",
#     "completed": false,
#     "createdAt": "...",
#     "updatedAt": "..."
#   }
# }

# List all TODOs
grpcurl -plaintext -d '{}' localhost:9091 api.v1.TodoService/ListTodos

# Update a TODO (replace <id> with actual ID)
grpcurl -plaintext -d '{"id":"<id>","title":"Buy groceries","description":"Updated","completed":true}' \
  localhost:9091 api.v1.TodoService/UpdateTodo

# Delete a TODO (replace <id> with actual ID)
grpcurl -plaintext -d '{"id":"<id>"}' localhost:9091 api.v1.TodoService/DeleteTodo
```

#### Using the Frontend

1. Open http://localhost:5173 in your browser
2. Navigate to the TODO section
3. **Create**: Enter a title and description, click "Add TODO"
4. **List**: Verify the TODO appears in the list
5. **Update**: Click "Edit" on a TODO, modify it, and save
6. **Delete**: Click "Delete" on a TODO and verify it's removed

### Test Service-to-Service Communication

The TODO Service calls the Hello Service when creating TODOs (if implemented).

1. Check the TODO Service logs:
   ```bash
   tail -f logs/todo-service.log
   ```

2. Create a TODO via the frontend or grpcurl

3. Look for log entries showing Hello Service calls

### Test API Gateway (Envoy)

If Envoy is running:

1. **Check Envoy Admin Interface**:
   ```bash
   curl http://localhost:9901
   ```

2. **Test Routing**:
   ```bash
   # Hello Service through Envoy
   curl -X POST http://localhost:8080/api/hello/api.v1.HelloService/SayHello \
     -H "Content-Type: application/grpc-web+proto" \
     -d '{"name":"Alice"}'
   
   # TODO Service through Envoy
   curl -X POST http://localhost:8080/api/todo/api.v1.TodoService/ListTodos \
     -H "Content-Type: application/grpc-web+proto" \
     -d '{}'
   ```

## Verification Checklist

Use this checklist to verify your local setup:

### Services Running

- [ ] Hello Service is running on port 9090
- [ ] TODO Service is running on port 9091
- [ ] Frontend is running on port 5173
- [ ] Envoy Proxy is running on port 8080 (optional)

### Hello Service

- [ ] Returns greeting with provided name
- [ ] Returns default greeting for empty name
- [ ] Responds to gRPC calls

### TODO Service

- [ ] Creates TODO items successfully
- [ ] Lists all TODO items
- [ ] Updates TODO items
- [ ] Deletes TODO items
- [ ] Validates input (rejects empty titles)

### Frontend

- [ ] Loads successfully in browser
- [ ] Hello form works correctly
- [ ] TODO list displays items
- [ ] Can create new TODOs
- [ ] Can update existing TODOs
- [ ] Can delete TODOs
- [ ] Displays error messages appropriately

### Service-to-Service Communication

- [ ] TODO Service can reach Hello Service
- [ ] Services use correct addresses (localhost:9090, localhost:9091)
- [ ] No connection errors in logs

### API Gateway (if using Envoy)

- [ ] Envoy routes requests to correct services
- [ ] CORS headers are set correctly
- [ ] gRPC-Web protocol conversion works

## Troubleshooting

### Port Already in Use

If you see "Port already in use" errors:

```bash
# Find and kill processes using the ports
lsof -ti:9090 | xargs kill -9  # Hello Service
lsof -ti:9091 | xargs kill -9  # TODO Service
lsof -ti:5173 | xargs kill -9  # Frontend
lsof -ti:8080 | xargs kill -9  # Envoy
```

### Service Won't Start

1. **Check logs**:
   ```bash
   tail -f logs/hello-service.log
   tail -f logs/todo-service.log
   tail -f logs/web.log
   ```

2. **Verify dependencies**:
   ```bash
   make check-env
   ```

3. **Rebuild services**:
   ```bash
   make clean
   make build
   ```

### gRPC Connection Errors

1. **Verify service is listening**:
   ```bash
   lsof -i :9090  # Hello Service
   lsof -i :9091  # TODO Service
   ```

2. **Check firewall settings** (macOS):
   ```bash
   # Allow incoming connections for Java and Go
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/bin/java
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/go/bin/go
   ```

### Frontend Can't Connect to Backend

1. **Check Vite proxy configuration** (`apps/web/vite.config.ts`):
   ```typescript
   server: {
     proxy: {
       '/api': {
         target: 'http://localhost:8080',
         changeOrigin: true,
       }
     }
   }
   ```

2. **Verify Envoy is running**:
   ```bash
   curl http://localhost:8080
   ```

3. **Check browser console** for CORS or network errors

### Envoy Not Starting

1. **Install Envoy**:
   ```bash
   # macOS
   brew install envoy
   
   # Linux
   # See https://www.envoyproxy.io/docs/envoy/latest/start/install
   ```

2. **Verify configuration**:
   ```bash
   envoy --mode validate -c deploy/docker/envoy-local-config.yaml
   ```

### grpcurl Not Found

Install grpcurl for gRPC testing:

```bash
# macOS
brew install grpcurl

# Go install
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Verify installation
grpcurl --version
```

## Performance Verification

### Response Times

Expected response times for local development:

- **Hello Service**: < 10ms
- **TODO Service**: < 20ms
- **Frontend**: < 100ms (initial load), < 50ms (subsequent requests)

### Memory Usage

Expected memory usage:

- **Hello Service**: ~200-300 MB
- **TODO Service**: ~20-50 MB
- **Frontend (dev server)**: ~100-200 MB
- **Envoy**: ~50-100 MB

Monitor with:

```bash
# macOS
top -pid $(lsof -ti:9090) -pid $(lsof -ti:9091)

# Linux
ps aux | grep -E 'java|go|node'
```

## Next Steps

Once local verification is complete:

1. **Run unit tests**:
   ```bash
   make test
   ```

2. **Run linters**:
   ```bash
   make lint
   ```

3. **Build Docker images**:
   ```bash
   make docker-build
   ```

4. **Deploy to Kubernetes** (see [DEPLOYMENT.md](./DEPLOYMENT.md))

## Additional Resources

- [Getting Started Guide](./GETTING_STARTED.md)
- [Architecture Documentation](./ARCHITECTURE.md)
- [API Documentation](../api/v1/README.md)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)

