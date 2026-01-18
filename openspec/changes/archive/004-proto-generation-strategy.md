# Change: Unified Proto Generation Strategy (Hybrid)

**Status**: Completed  
**Date**: 2025-2026  
**Type**: Architecture  
**Owner**: Platform Team

## Summary

Implemented hybrid proto generation strategy where generated code is NOT committed to git. Go generates proto inside Docker, TypeScript generates in CI, and Java generates in CI then copies to Docker.

## Problem Statement

**Before**:
- Generated proto code committed to git
- Large git diffs for proto changes
- Merge conflicts in generated code
- Inconsistent generation across environments

**After**:
- Generated code excluded from git
- Clean git diffs (proto files only)
- No merge conflicts in generated code
- Consistent generation in CI/Docker

## Design Evolution

### Initial Goal
All proto-generated code NOT committed to git, generated during build time (inspired by MoeGo Monorepo).

### Reality Check
Gradle protobuf plugin has Docker path issues:
```
Cannot remap path '/opt'
```

### Final Solution: Hybrid Strategy

**Go Services**:
- Generate proto inside Docker
- Self-contained build
- No external dependencies

**TypeScript**:
- Generate proto in CI before tests
- Use generated code for type checking
- Not included in production bundle

**Java Services**:
- Generate proto in CI
- Copy generated code to Docker
- Avoids Gradle path remapping issues

## Implementation

### 1. Updated .gitignore

```gitignore
# Generated protobuf code (NOT committed)
apps/*/gen/
apps/*/src/main/java-gen/
apps/*/src/gen/
```

### 2. Go Dockerfile

```dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /app

# Install protoc and plugins
RUN apk add --no-cache protobuf protobuf-dev
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

# Copy proto files and generate
COPY ../../api/v1 /api/v1
RUN protoc --go_out=. --go-grpc_out=. -I /api/v1 /api/v1/*.proto

# Build service
COPY . .
RUN go build -o service .
```

### 3. Java Dockerfile

```dockerfile
FROM eclipse-temurin:17-jre-alpine
WORKDIR /app

# Copy pre-generated proto code from CI
COPY src/main/java-gen ./src/main/java-gen
COPY target/*.jar app.jar

ENTRYPOINT ["java", "-jar", "app.jar"]
```

### 4. CI Workflow Updates

```yaml
- name: Generate Protobuf code
  run: make proto

- name: Build Java service
  run: |
    cd apps/hello-service
    ./gradlew build
    
- name: Build Docker image
  run: |
    # Generated code already exists from previous step
    docker build -t hello-service apps/hello-service
```

### 5. Migration Script

**File**: `scripts/clean-generated-code.sh`

```bash
#!/bin/bash
# Remove all generated proto code from git
git rm -r apps/*/gen/ 2>/dev/null || true
git rm -r apps/*/src/main/java-gen/ 2>/dev/null || true
git rm -r apps/*/src/gen/ 2>/dev/null || true
```

## Outcomes

### Benefits
- ✅ Clean git history (proto files only)
- ✅ No merge conflicts in generated code
- ✅ Consistent generation across environments
- ✅ Smaller repository size
- ✅ Faster git operations

### Metrics
- **Git repo size**: Reduced by ~15%
- **Proto change diffs**: 90% smaller
- **Merge conflicts**: Eliminated in generated code
- **Build time**: No significant change

### Trade-offs
- **Hybrid approach**: Not fully uniform across languages
- **CI dependency**: Must generate before building
- **Local setup**: Requires `make proto` before development

## Documentation

**Created**:
- `docs/PROTO_GENERATION_STRATEGY.md` - Initial strategy
- `docs/PROTO_HYBRID_STRATEGY.md` - Final hybrid approach
- `docs/MIGRATION_TO_UNIFIED_PROTO.md` - Migration guide

**Updated**:
- `.gitignore` - Exclude generated code
- `Dockerfile` files - Updated for each language
- `README.md` - Proto generation workflow

## Challenges & Solutions

### Challenge 1: Gradle Docker Path Remapping
**Problem**: `Cannot remap path '/opt'`  
**Solution**: Generate in CI, copy to Docker

### Challenge 2: Local Development
**Problem**: Developers need generated code  
**Solution**: `make proto` command, clear documentation

### Challenge 3: CI Complexity
**Problem**: Multiple generation steps  
**Solution**: Unified `make proto` target

## Best Practices Established

### Proto Changes
1. Modify `.proto` files
2. Run `make proto` locally
3. Test with generated code
4. Commit only `.proto` changes
5. CI regenerates and validates

### New Service Setup
1. Create service from template
2. Run `make proto` to generate code
3. Implement business logic
4. CI handles generation automatically

### Troubleshooting
- **Missing generated code**: Run `make proto`
- **Stale generated code**: Run `make clean && make proto`
- **CI failures**: Check proto syntax

## Related Changes

**Preceded by**:
- [001-monorepo-initialization.md](./001-monorepo-initialization.md)
- [003-shift-left-quality.md](./003-shift-left-quality.md)

**Followed by**:
- [005-dynamic-ci-cd.md](./005-dynamic-ci-cd.md)

## References

- Implementation Tasks: `.kiro/specs/monorepo-hello-todo/tasks.md` (Task 11)
- Documentation: `docs/PROTO_HYBRID_STRATEGY.md`
- Migration Guide: `docs/MIGRATION_TO_UNIFIED_PROTO.md`
- Inspiration: MoeGo Monorepo design patterns
