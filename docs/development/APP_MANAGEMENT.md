# App Management System

This document explains the app management system in the monorepo, which provides a unified interface for managing applications.

## Overview

The app management system allows you to:
- Operate on specific apps by name
- Auto-detect changed apps based on git diff
- Run common operations (test, build, lint, etc.) across multiple apps
- Maintain consistent tooling across different app types (Java, Go, Node.js)

## Quick Start

### Create a New App

Create a new app from template using the interactive command:

```bash
make create
```

Or create directly with parameters:

```bash
# Create a Java service
./scripts/create-app.sh java user-service --port 9092 --description "User management service"

# Create a Go service
./scripts/create-app.sh go payment-service --port 9093 --module github.com/myorg/cuckoo/apps/payment-service

# Create a Node.js app
./scripts/create-app.sh node admin-dashboard --description "Admin dashboard"
```

The create command will:
- Copy the appropriate template
- Replace all placeholders with your app details
- Create a protobuf file for your service
- Automatically register the app in app-manager.sh
- Integrate with all monorepo features (CI/CD, testing, Docker, etc.)

### List Available Apps

```bash
make list-apps
```

This shows all available apps and their types:
```
Available apps:
  - hello-service (java)
  - todo-service (go)
  - web (node)
```

### Operate on Specific App

Use the `APP` parameter to target a specific app. You can use either the full app name or a convenient short name:

```bash
# Using full names
make test APP=hello-service
make build APP=todo-service
make run APP=web

# Using short names (more convenient!)
make test APP=hello
make build APP=todo
make run APP=web
```

**Supported short names:**
- `hello` → `hello-service`
- `todo` → `todo-service`
- `web` → `web` (no change)

**All available commands:**
```bash
# Test
make test APP=hello

# Build
make build APP=hello

# Run locally
make run APP=hello

# Lint
make lint APP=hello

# Auto-fix lint errors
make lint-fix APP=hello

# Format code
make format APP=hello

# Build Docker image
make docker-build APP=hello

# Clean build artifacts
make clean APP=web
```

### Auto-Detect Changed Apps

When you don't specify an `APP` parameter, the system automatically detects which apps have changed based on git diff:

```bash
# Test only changed apps
make test

# Build only changed apps
make build

# Lint only changed apps
make lint

# Build Docker images for changed apps
make docker-build
```

## How Auto-Detection Works

The `scripts/detect-changed-apps.sh` script analyzes git changes to determine which apps are affected:

1. **App-specific changes**: If files in `apps/hello-service/` changed, `hello-service` is affected
2. **API changes**: If files in `api/` changed, all backend services (`hello-service`, `todo-service`) are affected
3. **Shared library changes**: If files in `libs/` changed, all apps are affected
4. **No changes detected**: All apps are included (for safety)

### Examples

```bash
# You modified apps/hello-service/src/main/java/...
make test
# → Tests only hello-service

# You modified api/v1/hello.proto
make build
# → Builds hello-service and todo-service (both use the API)

# You modified libs/common/...
make test
# → Tests all apps (hello-service, todo-service, web)
```

## App Types and Commands

The system supports three app types, each with type-specific implementations:

### Java Apps (hello-service)

- **Build**: Uses Gradle or Maven (`./gradlew build` or `./mvnw package`)
- **Test**: `./gradlew test` or `./mvnw test`
- **Run**: `./gradlew bootRun` or `./mvnw spring-boot:run`
- **Lint**: `./gradlew checkstyleMain checkstyleTest`
- **Format**: Enforced by Checkstyle (runs lint check)

### Go Apps (todo-service)

- **Build**: `go build -o bin/<app-name> .`
- **Test**: `go test ./...`
- **Run**: `go run .`
- **Lint**: `golangci-lint run ./...` (falls back to `go vet` if not installed)
- **Format**: `gofmt -w . && goimports -w .`

### Node.js Apps (web)

- **Build**: `npm run build`
- **Test**: `npm test -- --run`
- **Run**: `npm run dev`
- **Lint**: `npm run lint`
- **Format**: `npm run format`

## Direct Script Usage

You can also use the app-manager script directly:

```bash
# List apps
./scripts/app-manager.sh list

# Test specific app
./scripts/app-manager.sh test hello-service

# Test changed apps
./scripts/app-manager.sh test

# Build all changed apps
./scripts/app-manager.sh build

# Run specific app
./scripts/app-manager.sh run web
```

## CI/CD Integration

The app management system is integrated into the CI/CD pipeline:

```yaml
# .github/workflows/ci.yml
- name: Test changed apps
  run: make test

- name: Build changed apps
  run: make build

- name: Build Docker images for changed apps
  run: make docker-build
```

This ensures that only affected apps are tested and built, speeding up CI runs.

## Adding New Apps

### Using the Create Command (Recommended)

The easiest way to add a new app is using the `make create` command:

```bash
make create
```

This will interactively prompt you for:
- App type (java, go, or node)
- App name (e.g., user-service)
- Port number (auto-assigned if not specified)
- Description
- Package name (for Java apps)
- Module path (for Go apps)
- Team name

### Direct Script Usage

You can also use the script directly with all parameters:

```bash
./scripts/create-app.sh <type> <name> [options]

Options:
  --port <port>           - gRPC port (default: auto-assign)
  --description <desc>    - Service description
  --package <package>     - Java package name (for Java apps)
  --module <module>       - Go module path (for Go apps)
  --proto <proto-file>    - Protobuf file name (without .proto)
  --team <team-name>      - Team name for ownership
```

### Examples

```bash
# Create a Java service
./scripts/create-app.sh java user-service \
  --port 9092 \
  --description "User management service" \
  --package com.pingxin403.cuckoo.user \
  --team backend-team

# Create a Go service
./scripts/create-app.sh go payment-service \
  --port 9093 \
  --description "Payment processing service" \
  --module github.com/pingxin403/cuckoo/apps/payment-service \
  --team payment-team

# Create a Node.js app
./scripts/create-app.sh node admin-dashboard \
  --description "Admin dashboard application" \
  --team frontend-team
```

### What Gets Created

When you create a new app, the script automatically:

1. **Copies the template** - Uses the appropriate template (java-service, go-service, or node-service)
2. **Replaces placeholders** - Updates all template variables with your app details
3. **Creates protobuf file** - Generates `api/v1/<your-app>.proto` with basic structure
4. **Registers in app-manager** - Adds your app to `scripts/app-manager.sh`
5. **Auto-assigns port** - Finds the next available port if not specified

Your new app is immediately integrated with:
- ✅ App management system (`make test/build/lint/etc`)
- ✅ Auto-detection for changed apps
- ✅ CI/CD pipeline
- ✅ Testing framework with coverage requirements
- ✅ Docker build support
- ✅ Kubernetes deployment templates
- ✅ Code quality tools (linting, formatting)

### Next Steps After Creation

After creating your app:

1. **Define your API** - Edit `api/v1/<your-app>.proto` to define your service interface
2. **Generate code** - Run `make gen-proto` to generate protobuf code
3. **Implement logic** - Write your service implementation in `apps/<your-app>/`
4. **Add tests** - Write unit tests and property-based tests
5. **Build and test** - Run `make build APP=<your-app>` and `make test APP=<your-app>`

## Adding New Apps Manually

If you prefer to add apps manually without using the create script:

1. **Create the app directory**: `apps/my-new-service/`

2. **Update `scripts/app-manager.sh`**:
   ```bash
   # Add to APP_TYPES
   APP_TYPES=(
       ["hello-service"]="java"
       ["todo-service"]="go"
       ["web"]="node"
       ["my-new-service"]="go"  # Add your app
   )
   
   # Add to APP_PATHS
   APP_PATHS=(
       ["hello-service"]="apps/hello-service"
       ["todo-service"]="apps/todo-service"
       ["web"]="apps/web"
       ["my-new-service"]="apps/my-new-service"  # Add your app
   )
   ```

3. **Test the integration**:
   ```bash
   make list-apps  # Should show your new app
   make test APP=my-new-service
   make build APP=my-new-service
   ```

## Troubleshooting

### App not detected

If your app isn't being detected as changed:

```bash
# Check what the detection script returns
./scripts/detect-changed-apps.sh

# Check git diff
git diff --name-only main...HEAD
```

### Build tool not found

If you see "No build tool found" errors:

- **Java apps**: Ensure `gradlew` or `mvnw` exists in the app directory
- **Go apps**: Ensure Go is installed (`go version`)
- **Node.js apps**: Ensure npm is installed (`npm --version`)

### Command fails for specific app

Run the command directly to see detailed errors:

```bash
# Instead of: make test APP=hello-service
./scripts/app-manager.sh test hello-service

# Or run the underlying command:
cd apps/hello-service && ./mvnw test
```

## Migration from Legacy Targets

The old app-specific targets are deprecated but still work:

```bash
# New unified way
make test APP=hello-service
make build APP=todo-service
make docker-build APP=hello-service
```

The legacy targets will show a deprecation warning and redirect to the new system.

## Best Practices

1. **Use auto-detection in CI**: Let the system detect changed apps to optimize build times
2. **Use specific apps locally**: When working on a specific app, use `APP=<name>` for faster feedback
3. **Test before commit**: Run `make test` to test all changed apps before committing
4. **Keep app definitions updated**: When adding new apps, update both `APP_TYPES` and `APP_PATHS`
5. **Use consistent naming**: App names should match directory names in `apps/`

## Related Documentation

- [Testing Guide](TESTING_GUIDE.md) - How to write and run tests
- [Getting Started](GETTING_STARTED.md) - Initial setup and development workflow
- [Architecture](ARCHITECTURE.md) - Overall system architecture
