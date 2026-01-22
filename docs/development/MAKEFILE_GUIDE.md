# Makefile Guide

## Overview
This guide covers the Makefile usage in the monorepo, including proto generation, app management, testing, and deployment.

## Quick Reference

```bash
# Development
make init                    # Initialize development environment
make proto                   # Generate proto code for all languages
make build APP=hello         # Build specific app
make test APP=shortener      # Test specific app
make lint-fix                # Auto-fix linting issues

# Local Deployment
make dev-up                  # Start all services
make infra-up                # Start infrastructure only
make services-up             # Start application services only

# Kubernetes Deployment
make k8s-deploy-dev          # Deploy to development
make k8s-deploy-prod         # Deploy to production
make k8s-infra-deploy        # Deploy infrastructure with Helm
```

## Proto Generation

### Philosophy: Single Source of Truth

Proto files (`.proto`) are the single source of truth. Generated code is **NOT committed to git**.

**Why?**
1. **Consistency**: All languages follow the same pattern
2. **Atomic Changes**: Proto changes and implementation changes happen in one commit
3. **No Sync Issues**: Generated code is always in sync with proto definitions
4. **Clean Repository**: Git history focuses on source code, not generated artifacts

### Commands

```bash
# Generate all languages
make proto

# Generate specific language
make proto-go
make proto-java
make proto-ts

# Verify generated code is up-to-date (CI)
make verify-proto
```

### Configuration

Each app's `metadata.yaml` specifies which proto files it needs:

```yaml
spec:
  name: shortener-service
  short_name: shortener
  type: go
  proto_files:
    - shortener_service.proto
```

### How It Works

1. **Auto-detection**: Script scans `apps/` for services
2. **Read config**: Loads `metadata.yaml` for each app
3. **Filter by language**: Only generates for matching app types
4. **Generate code**: Runs protoc with appropriate plugins
5. **Report results**: Shows success/failure for each app

### Generated Code Locations

| Language | Location | Gitignored |
|----------|----------|------------|
| Go | `apps/*/gen/{proto}pb/` | ✅ Yes |
| Java | `apps/*/src/main/java-gen/` | ✅ Yes |
| TypeScript | `apps/*/src/gen/` | ✅ Yes |

### Adding New Proto Files

1. Create proto file in `api/v1/`
2. Add to `proto_files` in `metadata.yaml`
3. Run `make proto`
4. No Makefile changes needed!

### Docker Builds

**Strategy varies by language:**

**Go Services** (self-contained):
```dockerfile
# Generate proto inside Docker
FROM golang:1.25-alpine AS build
RUN apk add protoc protoc-gen-go protoc-gen-go-grpc
COPY api/v1 /api/v1
RUN protoc --go_out=. /api/v1/*.proto
```

**Java Services** (pre-generated):
```bash
# Generate in CI before Docker build
./gradlew generateProto
docker build .
```

**TypeScript Services** (npm script):
```bash
# Generate via npm
npm run gen-proto
```

## App Management

### List Apps

```bash
make list-apps
```

### Build Apps

```bash
# Build all changed apps
make build

# Build specific app (by short name)
make build APP=hello
make build APP=shortener
make build APP=todo
```

### Test Apps

```bash
# Test all changed apps
make test

# Test specific app
make test APP=hello

# Test with coverage
make test-coverage APP=shortener

# Verify coverage thresholds (CI)
make verify-coverage
```

### Lint Apps

```bash
# Lint all changed apps
make lint

# Lint specific app
make lint APP=hello

# Auto-fix lint errors
make lint-fix APP=shortener
```

### Docker Build

```bash
# Build Docker images for all changed apps
make docker-build

# Build Docker image for specific app
make docker-build APP=shortener
```

### Run Apps

```bash
# Run specific app locally
make run APP=hello
```

### Clean Apps

```bash
# Clean all apps
make clean

# Clean specific app
make clean APP=shortener
```

## App Auto-Detection

The Makefile automatically detects which apps have changed:

```bash
# Detects changed apps based on git diff
./scripts/detect-changed-apps.sh

# Uses metadata.yaml for app configuration
./scripts/app-manager.sh
```

### Short Names

Apps can be referenced by short names (defined in `metadata.yaml`):

| Full Name | Short Name |
|-----------|------------|
| hello-service | hello |
| todo-service | todo |
| shortener-service | shortener |
| web | web |

## Local Development

### Start Everything

```bash
# Start infrastructure + services
make dev-up

# Check status
make infra-status

# View logs
docker compose logs -f
```

### Start Infrastructure Only

```bash
# Start MySQL, Redis, etcd, Kafka
make infra-up

# Check health
make infra-status

# View logs
make infra-logs
```

### Start Services Only

```bash
# Start application services
make services-up

# Restart services (keep infrastructure running)
make dev-restart
```

### Stop Services

```bash
# Stop everything
make dev-down

# Stop infrastructure
make infra-down

# Stop services
make services-down
```

### Clean Data

```bash
# Clean all infrastructure data (WARNING: Deletes all data!)
make infra-clean
```

## Kubernetes Deployment

### Validate Manifests

```bash
make k8s-validate
```

### Deploy Infrastructure

```bash
# Deploy all infrastructure (Helm charts)
make k8s-infra-deploy
```

### Deploy Services

```bash
# Deploy to development
make k8s-deploy-dev

# Deploy to production
make k8s-deploy-prod
```

## Quality Checks

### Pre-commit Checks

```bash
# Run all quality checks
make pre-commit
```

This runs:
- Linting
- Testing
- Security checks
- Proto verification

### Coverage

```bash
# Run tests with coverage
make test-coverage

# Verify coverage thresholds
make verify-coverage
```

## Tool Versions

All tools use versions defined in `.tool-versions`:

```bash
# Check tool versions
make check-versions

# Verify all required tools are installed
make check-env
```

## Troubleshooting

### "Cannot find proto generated code"

**Solution**: Run `make proto`

### "App not detected by auto-detection"

**Solution**: Ensure `metadata.yaml` exists and has correct format

### "Docker build fails"

**Solution**: Check that proto files are copied in Dockerfile

### "Coverage threshold not met"

**Solution**: Add more tests or adjust threshold in `metadata.yaml`

## Best Practices

1. **Always run `make proto`** after modifying proto files
2. **Use short names** for convenience (e.g., `make test APP=hello`)
3. **Run `make pre-commit`** before committing
4. **Use `make dev-restart`** to quickly restart services during development
5. **Check `make infra-status`** if services fail to start

## Related Documentation

- [Proto Generation Strategy](../archive/PROTO_GENERATION_STRATEGY.md)
- [App Management](./APP_MANAGEMENT.md)
- [Testing Guide](./TESTING_GUIDE.md)
- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md)
