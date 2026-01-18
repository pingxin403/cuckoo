# Quick Reference Card

## Creating Apps

```bash
# Interactive creation
make create

# Direct creation
./scripts/create-app.sh java user-service --port 9092 --description "User service"
./scripts/create-app.sh go payment-service --port 9093
./scripts/create-app.sh node admin-dashboard
```

## Managing Apps

```bash
# List all apps
make list-apps

# Build (supports short names: hello, todo, web)
make build APP=hello            # Specific app (short name)
make build APP=hello-service    # Specific app (full name)
make build                      # Changed apps

# Test (supports short names)
make test APP=hello             # Specific app (short name)
make test APP=hello-service     # Specific app (full name)
make test                       # Changed apps

# Lint (supports short names)
make lint APP=hello             # Specific app (short name)
make lint                       # Changed apps

# Auto-fix lint errors (supports short names)
make lint-fix APP=hello         # Specific app (short name)
make lint-fix                   # Changed apps

# Format (supports short names)
make format APP=hello           # Specific app (short name)
make format                     # Changed apps

# Docker (supports short names)
make docker-build APP=hello     # Specific app (short name)
make docker-build               # Changed apps

# Run (supports short names)
make run APP=hello              # Specific app only

# Clean (supports short names)
make clean APP=hello            # Specific app (short name)
make clean                      # Changed apps
```

**Short name mappings:**
- `hello` → `hello-service`
- `todo` → `todo-service`
- `web` → `web`
```

## Development Workflow

```bash
# 1. Create new app
make create

# 2. Define API
vim api/v1/your-app.proto

# 3. Generate code
make gen-proto

# 4. Implement service
vim apps/your-app/...

# 5. Write tests
vim apps/your-app/...test...

# 6. Build and test
make build APP=your-app
make test APP=your-app

# 7. Check coverage
make test-coverage-your-app

# 8. Run locally
make run APP=your-app
# or
./scripts/dev.sh

# 9. Build Docker image
make docker-build APP=your-app

# 10. Deploy
kubectl apply -k k8s/overlays/production
```

## Testing

```bash
# Run all tests
make test

# Run specific app tests
make test APP=hello-service

# Test with coverage
make test-coverage-hello
make test-coverage-todo

# Verify coverage thresholds (CI)
make verify-coverage
```

## Protobuf

```bash
# Generate all
make gen-proto

# Generate specific language
make gen-proto-go
make gen-proto-java
make gen-proto-ts

# Verify generated code is up to date
make verify-proto
```

## Docker

```bash
# Build all images
make docker-build

# Build specific image
make docker-build APP=hello-service

# Run with docker-compose
docker-compose up
```

## Kubernetes

```bash
# Deploy all services
kubectl apply -k k8s/overlays/production

# Check status
kubectl get pods
kubectl get services
kubectl get ingress

# View logs
kubectl logs -f deployment/hello-service
kubectl logs -f deployment/todo-service
```

## Development

```bash
# Initialize environment
make init

# Check environment
make check-env

# Start all services
./scripts/dev.sh

# Format code
make format

# Lint code
make lint
```

## Troubleshooting

```bash
# Check if ports are in use
lsof -i :9090
lsof -i :9091

# View service logs
cd apps/hello-service && ./mvnw spring-boot:run
cd apps/todo-service && go run .

# Clean and rebuild
make clean APP=hello-service
make build APP=hello-service

# Regenerate protobuf
make gen-proto
```

## File Locations

```
.
├── api/v1/                     # Protobuf definitions
├── apps/                       # Applications
│   ├── hello-service/          # Java service
│   ├── todo-service/           # Go service
│   └── web/                    # React app
├── templates/                  # App templates
│   ├── java-service/
│   └── go-service/
├── scripts/                    # Build scripts
│   ├── create-app.sh           # Create new app
│   ├── app-manager.sh          # Manage apps
│   ├── detect-changed-apps.sh  # Detect changes
│   └── dev.sh                  # Dev environment
├── k8s/                        # Kubernetes configs
│   ├── base/
│   └── overlays/
├── docs/                       # Documentation
└── Makefile                    # Build commands
```

## Port Assignments

- **9090** - Hello Service (Java)
- **9091** - TODO Service (Go)
- **9092+** - New services (auto-assigned)
- **5173** - Web Application (dev)
- **8080** - Envoy Proxy

## Common Issues

### "App not found"
```bash
# Check app is registered
make list-apps

# Re-register app
./scripts/create-app.sh <type> <name>
```

### "Port already in use"
```bash
# Find process using port
lsof -i :<port>

# Kill process
kill -9 <pid>
```

### "Build failed"
```bash
# Clean and rebuild
make clean APP=<app>
make build APP=<app>

# Check for errors
cd apps/<app> && ./gradlew build --stacktrace  # Java
cd apps/<app> && go build -v .                 # Go
```

### "Tests failed"
```bash
# Run tests with verbose output
cd apps/<app> && ./gradlew test --info  # Java
cd apps/<app> && go test -v ./...       # Go

# Check coverage
make test-coverage-<app>
```

## Links

- [App Management Guide](APP_MANAGEMENT.md)
- [Create App Guide](CREATE_APP_GUIDE.md)
- [Testing Guide](TESTING_GUIDE.md)
- [Getting Started](GETTING_STARTED.md)
- [Architecture](ARCHITECTURE.md)
