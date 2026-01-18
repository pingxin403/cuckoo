# Proto Generation: Hybrid Strategy (Pragmatic Approach)

## TL;DR

After attempting a pure "generate-in-Docker" strategy, we adopted a **pragmatic hybrid approach** that balances idealism with tooling constraints.

## Core Principle (Unchanged)

**Proto files (`.proto`) are the Single Source of Truth. Generated code is NOT committed to git.**

## Implementation (Hybrid)

### Local Development
- ✅ Generate proto code with `make proto`
- ✅ Generated code is gitignored
- ✅ All languages follow same pattern

### CI/CD
- ✅ Generate proto code in each build job
- ✅ Use generated code for testing
- ✅ For Docker builds: strategy varies by language

### Docker Builds (Language-Specific)

| Language | Strategy | Reason |
|----------|----------|--------|
| **Go** | Generate in Docker | protoc works well in Docker |
| **TypeScript** | Generate in CI | Used for testing, not Docker |
| **Java** | Generate in CI, copy to Docker | Gradle protobuf plugin has Docker path issues |

## Why Hybrid?

### Initial Goal: Pure "Generate-in-Docker"

We wanted all Docker builds to be self-contained:
```dockerfile
# Ideal: Generate proto inside Docker
COPY api/v1 /api/v1
RUN protoc ...
RUN gradle build
```

### Reality: Gradle Protobuf Plugin Issues

Java's Gradle protobuf plugin has path mapping issues in Docker:
```
Error: Cannot remap path '/opt' which does not have '/' as a prefix
```

This is a known limitation of the Gradle protobuf plugin when:
1. Building from repository root
2. Proto files are outside the service directory
3. Running in Docker with different path structures

### Pragmatic Solution

**Generate proto in CI, then copy to Docker:**

```yaml
# CI workflow
- name: Generate Protobuf code
  run: ./gradlew generateProto

- name: Build Docker image
  run: docker build -f apps/hello-service/Dockerfile .
```

```dockerfile
# Dockerfile
COPY apps/hello-service/build/generated ./build/generated
RUN ./gradlew build -x generateProto
```

## Benefits of Hybrid Approach

### Still Achieves Core Goals
1. ✅ Proto files are Single Source of Truth
2. ✅ Generated code NOT committed to git
3. ✅ Clean repository
4. ✅ Consistent local development

### Pragmatic Advantages
1. ✅ Works around Gradle tooling limitations
2. ✅ Faster Docker builds (no protoc installation for Java)
3. ✅ Simpler Dockerfiles for Java services
4. ✅ Go services still self-contained (best of both worlds)

### Trade-offs
1. ⚠️ Java Docker builds depend on CI proto generation
2. ⚠️ Not perfectly "self-contained" for Java
3. ⚠️ Different strategies for different languages

## Comparison with Pure Approaches

### Pure "Commit Generated Code" (Old)
- ❌ Repository bloat
- ❌ Merge conflicts in generated code
- ❌ Sync issues
- ✅ Simple Docker builds

### Pure "Generate in Docker" (Ideal)
- ✅ Self-contained Docker builds
- ✅ Clean repository
- ❌ Doesn't work with Gradle protobuf plugin
- ❌ Complex Dockerfiles

### Hybrid "Generate in CI" (Pragmatic)
- ✅ Clean repository
- ✅ Works with all tooling
- ✅ Go services still self-contained
- ⚠️ Java services depend on CI

## When to Use Each Strategy

### Generate in Docker (Go, Rust, C++)
Use when:
- protoc works well in Docker
- No complex build tool path issues
- Want truly self-contained builds

### Generate in CI (Java with Gradle)
Use when:
- Build tool has Docker path issues
- Proto files are outside service directory
- Tooling constraints prevent Docker generation

## Future Improvements

### Option 1: Fix Gradle Plugin
- Contribute to gradle-protobuf-plugin
- Fix path mapping issues
- Move to pure "generate in Docker"

### Option 2: Restructure Repository
- Move proto files inside each service
- Eliminates cross-directory references
- Allows pure "generate in Docker"

### Option 3: Use Bazel
- Bazel handles proto generation elegantly
- Works well in Docker
- Requires significant migration effort

## Conclusion

The hybrid strategy is a **pragmatic compromise** that:
1. Achieves the core goal (clean repository, Single Source of Truth)
2. Works around tooling limitations
3. Maintains simplicity where possible (Go services)
4. Accepts complexity where necessary (Java services)

**Perfect is the enemy of good.** This approach works reliably across all languages while maintaining the benefits of not committing generated code.

## References

- [Proto Generation Strategy](./PROTO_GENERATION_STRATEGY.md)
- [Gradle Protobuf Plugin Issues](https://github.com/google/protobuf-gradle-plugin/issues)
- [Bazel Proto Rules](https://bazel.build/reference/be/protocol-buffer)
