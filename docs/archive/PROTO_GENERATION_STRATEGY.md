# Protobuf Code Generation Strategy

## Philosophy: Single Source of Truth

Following the principles of modern monorepo design (inspired by Google's Bazel and similar systems), we adopt a **unified proto generation strategy** across all languages.

## Core Principle

**Proto files (`.proto`) are the Single Source of Truth. Generated code is NOT committed to git.**

### Why This Approach?

1. **Consistency**: All languages follow the same pattern
2. **Atomic Changes**: Proto changes and implementation changes happen in one commit
3. **No Sync Issues**: Generated code is always in sync with proto definitions
4. **Clean Repository**: Git history focuses on source code, not generated artifacts
5. **Build Reproducibility**: Same proto files always generate the same code

## Implementation

### Local Development

Generated code is created during local builds and **NOT committed to git**:

```bash
# Generate proto code for all services
make proto

# Generated code locations (gitignored):
# - Go: apps/*/gen/
# - TypeScript: apps/*/src/gen/
# - Java: apps/*/build/generated/
```

### CI/CD Pipeline

Proto code is generated in CI for testing and Docker builds:

1. **verify-proto job**: Generates proto code and verifies it's up-to-date
2. **build-* jobs**: Generate proto code before testing
3. **Docker builds**: 
   - **Go**: Generate proto inside Docker (self-contained)
   - **TypeScript**: Generate proto in CI before tests
   - **Java**: Generate proto in CI, then copy to Docker (avoids Gradle path issues)

### Docker Builds

**Strategy varies by language due to tooling constraints:**

#### Go Services (Self-Contained)
Proto code is generated **inside the Docker build**:

```dockerfile
# Stage 1: Install protoc and generate code
FROM golang:1.25-alpine AS build
RUN apk add protoc
COPY api/v1 /api/v1
RUN protoc --go_out=. /api/v1/*.proto
COPY . .
RUN go build -o app .
```

#### Java Services (Pre-Generated)
Proto code is generated **in CI before Docker build**:

```bash
# In CI
./gradlew generateProto
docker build -f apps/hello-service/Dockerfile .
```

```dockerfile
# Dockerfile copies pre-generated code
COPY apps/hello-service/build/generated ./build/generated
RUN ./gradlew build -x generateProto
```

**Why different strategies?**
- Go: protoc works well in Docker
- Java: Gradle protobuf plugin has path mapping issues in Docker

## Migration from Old Approach

### Old Approach (Inconsistent)
- ✅ Go: Generated code committed to git
- ✅ TypeScript: Generated code committed to git
- ❌ Java: Generated code NOT committed to git

### New Approach (Unified)
- ✅ All languages: Generated code NOT committed to git
- ✅ All languages: Generate during build time
- ✅ All languages: Docker builds are self-contained

## Benefits

### For Developers
- No need to remember which language commits generated code
- Proto changes automatically trigger regeneration
- Cleaner git diffs (only proto changes visible)

### For CI/CD
- Faster builds (Docker layer caching for proto generation)
- Self-contained Docker images (no external dependencies)
- Consistent build process across all services

### For Repository
- Smaller repository size
- Cleaner git history
- No merge conflicts in generated code

## Tool Versions

All proto generation tools use versions defined in `.tool-versions`:

```bash
# Protobuf compiler
PROTOC_VERSION=28.3

# Go plugins
PROTOC_GEN_GO_VERSION=v1.36.6
PROTOC_GEN_GO_GRPC_VERSION=v1.5.1

# Java plugins (via Gradle)
# See apps/*/build.gradle

# TypeScript plugins (via npm)
# See apps/web/package.json
```

## Verification

The `verify-proto` CI job ensures generated code stays in sync:

```bash
make proto
git diff --exit-code || exit 1
```

If this fails, it means:
1. Proto files were changed
2. Generated code was not regenerated
3. Developer needs to run `make proto` and commit

## Best Practices

### When Modifying Proto Files

1. Edit `.proto` files in `api/v1/`
2. Run `make proto` to regenerate code
3. Update implementations to match new proto
4. Commit everything in one atomic commit

### When Creating New Services

1. Use templates that follow this strategy
2. Configure build.gradle/go.mod to reference proto files
3. Ensure Dockerfile generates proto code in build stage

### When Reviewing PRs

1. Check that proto changes are accompanied by implementation changes
2. Verify `verify-proto` CI job passes
3. Ensure no generated code is committed

## Troubleshooting

### "Cannot find proto generated code"

**Cause**: Generated code not created yet

**Solution**: Run `make proto`

### "Docker build fails: proto files not found"

**Cause**: Dockerfile not copying proto files correctly

**Solution**: Ensure `COPY api/v1 /api/v1` is in Dockerfile

### "CI verify-proto job fails"

**Cause**: Generated code out of sync with proto files

**Solution**: Run `make proto` locally and commit changes

## References

- [Bazel Build System](https://bazel.build/)
- [Google's Monorepo Philosophy](https://research.google/pubs/pub45424/)
- [Protocol Buffers Best Practices](https://protobuf.dev/programming-guides/dos-donts/)
