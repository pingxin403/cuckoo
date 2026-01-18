# Migration to Unified Proto Generation Strategy

## Overview

This document describes the migration from an inconsistent proto generation approach to a unified strategy across all languages.

## Problem Statement

### Before Migration

The project had **inconsistent proto code generation**:

| Language | Generated Code Location | Committed to Git? | Docker Build Strategy |
|----------|------------------------|-------------------|----------------------|
| Go | `apps/*/gen/` | ✅ Yes | Copy from git |
| TypeScript | `apps/*/src/gen/` | ✅ Yes | Copy from git |
| Java | `apps/*/build/generated/` | ❌ No | Generate in CI, then copy |

This caused several issues:

1. **Confusion**: Developers had to remember which language commits generated code
2. **Docker Build Complexity**: Java service required special CI steps to generate proto before Docker build
3. **Sync Issues**: Risk of generated code being out of sync with proto files
4. **Repository Bloat**: Generated code increased repository size
5. **Merge Conflicts**: Generated code could cause merge conflicts

## Solution: Unified Strategy

### New Approach

**All languages follow the same pattern**: Generated code is **NOT committed to git** and is **generated during build time**.

| Language | Generated Code Location | Committed to Git? | Docker Build Strategy |
|----------|------------------------|-------------------|----------------------|
| Go | `apps/*/gen/` | ❌ No | Generate in Docker |
| TypeScript | `apps/*/src/gen/` | ❌ No | Generate locally |
| Java | `apps/*/build/generated/` | ❌ No | Generate in Docker |

### Benefits

1. **Consistency**: All languages follow the same pattern
2. **Simplicity**: Docker builds are self-contained (no CI dependencies)
3. **Clean Repository**: Smaller git history, no generated code
4. **No Sync Issues**: Generated code always matches proto files
5. **Atomic Changes**: Proto changes and implementation changes in one commit

## Implementation Changes

### 1. Updated `.gitignore`

```diff
# Generated code - DO NOT commit proto-generated code
+gen/
+**/gen/
+**/build/generated/
+**/src/gen/
```

### 2. Updated Dockerfiles

All Dockerfiles now:
1. Install protoc and plugins in build stage
2. Copy proto files from `api/v1/`
3. Generate proto code during Docker build
4. Build application with generated code

**Example (Go service)**:

```dockerfile
# Stage 1: Proto generation and build
FROM golang:1.25-alpine AS build

# Install protoc and plugins
RUN apk add --no-cache curl unzip git && \
    curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protoc-28.3-linux-x86_64.zip && \
    unzip -o protoc-28.3-linux-x86_64.zip -d /usr/local && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

# Copy proto files (Single Source of Truth)
COPY api/v1 /api/v1

# Generate proto code
RUN protoc --proto_path=/api/v1 --go_out=./gen ...

# Build application
RUN go build -o app .
```

### 3. Updated CI Workflow

Removed the "Generate Protobuf code for Docker build" step from CI:

```diff
- - name: Generate Protobuf code for Docker build
-   if: github.event_name == 'push'
-   working-directory: apps/hello-service
-   run: ./gradlew generateProto --no-daemon

  - name: Build Docker image
    if: github.event_name == 'push'
    run: docker build -f apps/hello-service/Dockerfile -t hello-service:${{ github.sha }} .
```

Docker builds are now self-contained and don't depend on CI-generated artifacts.

### 4. Updated Templates

Both `templates/java-service/Dockerfile` and `templates/go-service/Dockerfile` now follow the unified strategy.

## Migration Steps

### For Developers

1. **Pull latest changes**:
   ```bash
   git pull origin main
   ```

2. **Clean generated code from git** (one-time):
   ```bash
   ./scripts/clean-generated-code.sh
   git commit -m "chore: remove generated proto code from git"
   ```

3. **Regenerate code locally**:
   ```bash
   make proto
   ```

4. **Verify builds work**:
   ```bash
   make build
   make test
   ```

5. **Push changes**:
   ```bash
   git push origin main
   ```

### For CI/CD

No manual steps required. The CI workflow has been updated to:
1. Generate proto code in `verify-proto` job (for verification)
2. Build Docker images (proto generation happens inside Docker)

### For New Services

Use the updated templates:
- `templates/java-service/` for Java services
- `templates/go-service/` for Go services

Both templates follow the unified proto generation strategy.

## Verification

### Local Development

```bash
# Clean and regenerate
rm -rf apps/*/gen apps/*/src/gen apps/*/build/generated
make proto

# Verify builds
make build
```

### Docker Builds

```bash
# Build Docker image (should work without pre-generated code)
docker build -f apps/hello-service/Dockerfile -t hello-service:test .
docker build -f apps/todo-service/Dockerfile -t todo-service:test .
```

### CI/CD

Push to GitHub and verify:
1. ✅ `verify-proto` job passes
2. ✅ `build-hello-service` job passes (Docker build succeeds)
3. ✅ `build-todo-service` job passes (Docker build succeeds)

## Troubleshooting

### Issue: "Cannot find generated proto code"

**Cause**: Generated code not created yet

**Solution**: Run `make proto`

### Issue: "Docker build fails: protoc not found"

**Cause**: Dockerfile not installing protoc correctly

**Solution**: Check Dockerfile has protoc installation step

### Issue: "CI verify-proto job fails"

**Cause**: Generated code out of sync with proto files

**Solution**: Run `make proto` locally and commit changes

### Issue: "Import paths not found in generated code"

**Cause**: Proto generation command incorrect

**Solution**: Check protoc command in Dockerfile matches Makefile

## References

- [Proto Generation Strategy](./PROTO_GENERATION_STRATEGY.md)
- [Protobuf Tool Versions](./PROTO_TOOLS_VERSION.md)
- [Getting Started Guide](./GETTING_STARTED.md)

## Inspiration

This unified approach is inspired by:
- **Google's Bazel**: Build-time code generation
- **MoeGo Monorepo**: Single Source of Truth for proto files
- **Modern Monorepo Practices**: Atomic commits, consistent tooling
