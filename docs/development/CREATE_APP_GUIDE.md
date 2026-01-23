# Creating New Apps Guide

This guide explains how to create new applications in the monorepo using the automated app creation system.

## Quick Start

The fastest way to create a new app:

```bash
make create
```

This will interactively guide you through the process.

## Command Reference

### Interactive Mode

```bash
make create
```

You'll be prompted for:
1. **App type**: java, go, or node
2. **App name**: kebab-case name (e.g., user-service)
3. **Port**: gRPC port number (leave empty for auto-assign)
4. **Description**: Brief description of the service
5. **Package** (Java only): Java package name
6. **Module** (Go only): Go module path
7. **Team**: Team name for ownership

### Direct Script Mode

For automation or when you know all parameters:

```bash
./scripts/create-app.sh <type> <name> [options]
```

**Options:**
- `--port <port>` - gRPC port (default: auto-assigned)
- `--description <desc>` - Service description
- `--package <package>` - Java package name (for Java apps)
- `--module <module>` - Go module path (for Go apps)
- `--proto <proto-file>` - Protobuf file name without .proto extension
- `--team <team-name>` - Team name (default: platform-team)

## Examples

### Create a Java Service

```bash
./scripts/create-app.sh java user-service \
  --port 9092 \
  --description "User management and authentication service" \
  --package com.pingxin403.cuckoo.user \
  --team backend-team
```

This creates:
- `apps/user-service/` - Service directory with Spring Boot setup
- `api/v1/user.proto` - Protobuf definition
- Gradle build configuration with JaCoCo coverage
- Kubernetes deployment files
- Test templates (unit + property-based)

### Create a Go Service

```bash
./scripts/create-app.sh go payment-service \
  --port 9093 \
  --description "Payment processing and billing service" \
  --module github.com/pingxin403/cuckoo/apps/payment-service \
  --team payment-team
```

This creates:
- `apps/payment-service/` - Service directory with Go modules
- `api/v1/payment.proto` - Protobuf definition
- Go module configuration
- Test coverage scripts
- Kubernetes deployment files
- Test templates (unit + property-based)

### Create a Node.js Application

```bash
./scripts/create-app.sh node admin-dashboard \
  --description "Admin dashboard for system management" \
  --team frontend-team
```

This creates:
- `apps/admin-dashboard/` - React/TypeScript application
- Vite configuration
- Package.json with dependencies
- Test setup with Vitest
- Kubernetes deployment files

## What Gets Created

When you run the create command, the following happens automatically:

### 1. Template Copy

The appropriate template is copied to `apps/<your-app>/`:
- `templates/java-service/` → Java/Spring Boot apps
- `templates/go-service/` → Go apps
- `templates/node-service/` → Node.js/React apps (if exists)

### 2. App Type Metadata

Two metadata files are automatically created:

**`.apptype` file:**
```
java  # or go, or node
```

This file enables automatic app type detection in CI/CD and build scripts.

**`metadata.yaml` file:**
```yaml
spec:
  name: your-app-name
  description: Your app description
  type: java  # or go, or node
  cd: true
  codeowners:
    - "@your-team"
test:
  coverage: 30  # or 80 for Go
```

This file provides structured metadata for:
- Service catalog integration
- Ownership tracking
- Test coverage requirements
- CI/CD configuration

### 3. Placeholder Replacement

All template placeholders are replaced with your values:
- `{{SERVICE_NAME}}` → your-app-name
- `{{SERVICE_DESCRIPTION}}` → Your description
- `{{GRPC_PORT}}` → Port number
- `{{PACKAGE_NAME}}` → Java package (Java only)
- `{{MODULE_PATH}}` → Go module (Go only)
- `{{TEAM_NAME}}` → Team name

### 4. Protobuf File Creation

A basic protobuf file is created at `api/v1/<your-app>.proto`:

```protobuf
syntax = "proto3";

package <your-app>pb;

service YourAppService {
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message HealthCheckRequest {}
message HealthCheckResponse {
  string status = 1;
}
```

### 5. App Registration

Your app is automatically registered in `scripts/app-manager.sh`:
- Added to `get_app_type()` function
- Added to `get_app_path()` function
- Added to `get_all_apps()` function

**Note:** With the new `.apptype` and `metadata.yaml` files, the app-manager.sh can also auto-detect your app without explicit registration.

### 6. Port Auto-Assignment

If you don't specify a port, the script:
1. Scans existing apps for used ports
2. Finds the highest port number
3. Assigns the next available port

## Immediate Integration

Your new app is immediately integrated with all monorepo features:

### ✅ App Management System

```bash
make test APP=your-app
make build APP=your-app
make lint APP=your-app
make format APP=your-app
make docker-build APP=your-app
make run APP=your-app
```

### ✅ Auto-Detection

When you make changes to your app, it's automatically detected:

```bash
make test  # Tests only changed apps, including yours
make build # Builds only changed apps
```

### ✅ CI/CD Pipeline

Your app is automatically included in:
- GitHub Actions workflows
- Test coverage verification (80% overall, 90% for service classes)
- Docker image building
- Kubernetes deployment

### ✅ Testing Framework

Your app includes:
- Unit test templates
- Property-based test templates
- Coverage reporting (JaCoCo for Java, go test -cover for Go)
- Test scripts

### ✅ Docker Support

```bash
make docker-build APP=your-app
```

Dockerfile is included with multi-stage builds.

### ✅ Kubernetes Deployment

K8s resources are automatically created in `deploy/k8s/services/your-app/`:
- `your-app-deployment.yaml` - Deployment configuration
- `your-app-service.yaml` - Service definition
- `kustomization.yaml` - Kustomize configuration

**Automatic Integration:**
Your service is automatically added to:
- `deploy/k8s/overlays/development/kustomization.yaml` (1 replica for dev)
- `deploy/k8s/overlays/production/kustomization.yaml` (3 replicas for prod)

### ✅ Docker Compose Integration

Your service is automatically added to `deploy/docker/docker-compose.services.yml` with:
- Build configuration
- Port mappings
- Environment variables
- Health checks
- Network configuration

## Next Steps After Creation

### 1. Define Your API

Edit `api/v1/<your-app>.proto` to define your service interface:

```protobuf
service YourAppService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

### 2. Generate Protobuf Code

```bash
make gen-proto
```

This generates code for all languages (Java, Go, TypeScript).

### 3. Implement Your Service

**For Java:**
- Edit `apps/your-app/src/main/java/.../service/YourAppServiceImpl.java`
- Implement the gRPC service methods

**For Go:**
- Edit `apps/your-app/service/your_app_service.go`
- Implement the gRPC service methods
- Edit `apps/your-app/storage/memory_store.go` for data storage

**For Node.js:**
- Edit `apps/your-app/src/` files
- Implement your React components

### 4. Write Tests

**Unit Tests:**
- Test specific functionality
- Test edge cases
- Test error handling

**Property-Based Tests:**
- Test universal properties
- Use random input generation
- Verify correctness across many inputs

### 5. Build and Test

```bash
# Build your app
make build APP=your-app

# Run tests
make test APP=your-app

# Check test coverage
make test-coverage-your-app  # For specific coverage report
make verify-coverage         # For CI verification
```

### 6. Run Locally

```bash
# Run your app
make run APP=your-app

# Or use the dev script to run all services
./scripts/dev.sh
```

### 7. Deploy

```bash
# Build Docker image
make docker-build APP=your-app

# Deploy to Kubernetes
kubectl apply -k k8s/overlays/production
```

## Customization

After creation, you can customize:

### Build Configuration

**Java (build.gradle):**
- Add dependencies
- Configure plugins
- Adjust JaCoCo coverage thresholds

**Go (go.mod):**
- Add dependencies with `go get`
- Update module path if needed

**Node.js (package.json):**
- Add npm dependencies
- Configure build scripts

### Kubernetes Resources

Edit files in `deploy/k8s/services/your-app/`:
- Adjust resource limits in `your-app-deployment.yaml`
- Add environment variables
- Configure health checks
- Add volumes or secrets

Remember to update the overlays:
- `deploy/k8s/overlays/development/kustomization.yaml`
- `deploy/k8s/overlays/production/kustomization.yaml`

### Testing

Customize test configuration:
- Adjust coverage thresholds
- Add more test cases
- Configure test frameworks

## Troubleshooting

### App Not Detected

If `make list-apps` doesn't show your app:

1. Check `scripts/app-manager.sh` was updated correctly
2. Verify the app directory exists in `apps/`
3. Try running `./scripts/create-app.sh` again

### Build Fails

If build fails after creation:

1. Run `make gen-proto` to generate protobuf code
2. Check for syntax errors in generated files
3. Verify dependencies are installed
4. Check build tool (gradlew/mvnw/go/npm) is available

### Port Conflict

If the assigned port conflicts:

1. Check what ports are in use: `lsof -i :<port>`
2. Edit the port in:
   - `deploy/k8s/services/your-app/your-app-deployment.yaml`
   - `deploy/k8s/services/your-app/your-app-service.yaml`
   - Application configuration files

### Tests Fail

If tests fail after creation:

1. Implement the service methods (templates have placeholders)
2. Update test expectations to match your implementation
3. Run `make test APP=your-app` to see specific errors

## Best Practices

### Naming Conventions

- **App names**: Use kebab-case (e.g., `user-service`, `payment-gateway`)
- **Proto files**: Use snake_case (e.g., `user_service.proto`)
- **Java packages**: Use dot notation (e.g., `com.pingxin403.cuckoo.user`)
- **Go modules**: Use full path (e.g., `github.com/pingxin403/cuckoo/apps/user-service`)

### Port Assignment

- **9090-9099**: Backend services
- **5000-5999**: Frontend applications
- **8000-8999**: API gateways and proxies

### Team Ownership

Specify the team that owns the service:
- `backend-team` - Backend services
- `frontend-team` - Frontend applications
- `platform-team` - Infrastructure and shared services
- `data-team` - Data processing services

### Documentation

After creating your app:
1. Update the app's README.md
2. Document API endpoints in the proto file
3. Add usage examples
4. Document configuration options

## Related Documentation

- [App Management System](APP_MANAGEMENT.md) - Managing apps
- [Testing Guide](TESTING_GUIDE.md) - Writing tests
- [Getting Started](GETTING_STARTED.md) - Development setup
- [Architecture](ARCHITECTURE.md) - System architecture
