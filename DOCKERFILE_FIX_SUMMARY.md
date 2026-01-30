# Dockerfile Fix Summary

## Problem

All Go service Dockerfiles had the same issue: the WORKDIR didn't match the go.mod replace directives.

### Root Cause

Go services use replace directives in go.mod:
```go
replace github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
replace github.com/pingxin403/cuckoo/libs/observability => ../../libs/observability
```

When WORKDIR is `/app`, the relative path `../../api/gen/go` resolves to `/api/gen/go` (going up two levels from `/app`).

But the Dockerfile copies files to `/app/api/gen/`, so Go can't find them at `/api/gen/go`.

### Solution

Change WORKDIR to `/app/apps/<service-name>` so that:
- `../../api/gen/go` resolves to `/app/api/gen/go` ✅
- `../../libs/observability` resolves to `/app/libs/observability` ✅

## Fixed Services

### ✅ Completed
1. **todo-service** - Fixed and tested successfully
2. **shortener-service** - Fixed (network timeout during test, but Dockerfile is correct)
3. **auth-service** - Fixed
4. **user-service** - Fixed (also removed unnecessary proto generation)

### 🔄 Need Fixing
5. **im-gateway-service** - Needs same fix + remove proto generation
6. **im-service** - Needs same fix + remove proto generation

## Standard Dockerfile Pattern

All Go services should follow this pattern:

```dockerfile
# Multi-stage build for Go service
FROM golang:1.25-alpine AS build

# IMPORTANT: WORKDIR must be /app/apps/<service-name> to match go.mod replace directives
WORKDIR /app/apps/<service-name>

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files first (for better caching)
COPY apps/<service-name>/go.mod apps/<service-name>/go.sum ./

# Copy shared libraries and proto code to match go.mod replace directives
# From /app/apps/<service-name>, ../../api/gen/go resolves to /app/api/gen/go
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/

# Download dependencies
RUN go mod download

# Copy source code
COPY apps/<service-name>/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o <service-name> .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/apps/<service-name>/<service-name> .

# Expose ports
EXPOSE <port>

# Run the service
ENTRYPOINT ["./<service-name>"]
```

## Key Changes

### 1. WORKDIR Location
**Before:**
```dockerfile
WORKDIR /app
```

**After:**
```dockerfile
WORKDIR /app/apps/<service-name>
```

### 2. Copy Order (for better caching)
**Before:**
```dockerfile
COPY api/gen/go /app/api/gen/go
COPY apps/<service-name>/go.mod apps/<service-name>/go.sum ./
RUN go mod download
```

**After:**
```dockerfile
COPY apps/<service-name>/go.mod apps/<service-name>/go.sum ./
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/
RUN go mod download
```

### 3. Binary Copy Path
**Before:**
```dockerfile
COPY --from=build /app/<service-name> .
```

**After:**
```dockerfile
COPY --from=build /app/apps/<service-name>/<service-name> .
```

### 4. Remove Proto Generation
Services like user-service and im-gateway-service had proto generation inside the Dockerfile. This is unnecessary because:
- Proto code is pre-generated in `api/gen/`
- `make build-image` validates proto code before building
- Generating inside Docker is slower and duplicates work

**Removed:**
```dockerfile
# Install protoc and required tools
ARG PROTOC_VERSION=28.3
...
RUN mkdir -p gen/hellopb gen/todopb && \
    protoc --proto_path=/api/v1 ...
```

## Testing

### Test Individual Service
```bash
make build-image APP=<service-name>
```

### Test All Services
```bash
make build-image
```

### Verify Image
```bash
docker images | grep <service-name>
docker run --rm <service-name>:latest --version
```

## Remaining Work

1. Fix im-gateway-service Dockerfile
2. Fix im-service Dockerfile
3. Test all Go services
4. Update CI to use `make build-image` instead of `docker-build`
5. Document the pattern for future services

## Benefits

1. **Consistency** - All Go services use the same pattern
2. **Simplicity** - No proto generation in Dockerfiles
3. **Speed** - Better layer caching, faster builds
4. **Reliability** - Proto validation before build prevents errors
5. **Maintainability** - Single source of truth for proto code

## Related Files

- **Spec**: `.kiro/specs/docker-image-builder/`
- **Build Script**: `scripts/build-image.sh`
- **Makefile**: `Makefile` (build-image target)
- **Fixed Dockerfiles**:
  - `apps/todo-service/Dockerfile`
  - `apps/shortener-service/Dockerfile`
  - `apps/auth-service/Dockerfile`
  - `apps/user-service/Dockerfile`
- **Need Fixing**:
  - `apps/im-gateway-service/Dockerfile`
  - `apps/im-service/Dockerfile`
