# Quality Practices

**Status**: Implemented  
**Owner**: Platform Team  
**Last Updated**: 2026-01-18

## Overview

Shift-left quality practices ensuring code quality, consistency, and security before code reaches production.

## Pre-Commit Checks

**Script**: `scripts/pre-commit-checks.sh`

**Command**: `make pre-commit`

**Six Categories of Checks**:

### 1. Tool Version Consistency
- Verifies all tools match `.tool-versions`
- Checks: protoc, protoc-gen-go, protoc-gen-go-grpc, go, java, node
- Prevents version drift between developers

### 2. Protobuf Code Synchronization
- Verifies generated code is up-to-date
- Runs `make proto` and checks for diffs
- Ensures API contracts are synchronized

### 3. Linting
- **Java**: Checkstyle
- **Go**: golangci-lint
- **TypeScript**: ESLint + Prettier
- Enforces code style consistency

### 4. Unit Tests
- Runs all unit tests
- Verifies test coverage thresholds
- Fast feedback on code changes

### 5. Common Issues
- Detects `console.log` in production code
- Finds TODO/FIXME comments
- Checks for large files (>1MB)
- Prevents common mistakes

### 6. Security Scanning
- Scans for potential secrets
- Checks for hardcoded credentials
- Detects API keys and tokens

## Test Coverage

### Coverage Philosophy

**Core Principle**: "使用集成测试提高测试覆盖率并不合理"

Test coverage should reflect **unit test** coverage of core business logic, not be inflated by integration tests. Coverage metrics should accurately represent code quality, not be gamed by running tests against external dependencies.

### Coverage Requirements

**Go Services**:
- **Core packages**: 70% minimum
  - Includes: business logic packages (service, cache, errors, idgen, analytics, etc.)
  - Excludes: packages requiring external dependencies
- **Overall coverage**: Reference only, not a hard requirement
- **Excludes**: 
  - Generated code (`gen/`, `*_pb.go`)
  - Infrastructure code (`main.go`, `logger/`)
  - External dependency wrappers (`storage/`, `cache/l2_cache.go`)

**Java Services**:
- Overall: 30%
- Service layer: 50%
- Excludes: generated code, configuration classes

### Package Classification

**Core Business Logic Packages** (must meet coverage threshold):
- Pure business logic that can be unit tested without external dependencies
- Examples: validation, ID generation, error handling, service layer logic
- Use mocks for external dependencies in tests

**Infrastructure Packages** (excluded from threshold):
- Require real external services (databases, caches, message queues)
- Tested in integration tests with Docker Compose
- Examples: database operations, Redis operations, Kafka producers
- Low unit test coverage is expected and acceptable

### Coverage Scripts

**Go** (`apps/shortener-service/scripts/test-coverage.sh`):
```bash
#!/bin/bash
set -e

echo "Running tests with coverage..."
# Exclude integration tests (they require external dependencies)
go test -v -race -coverprofile=coverage.out \
  $(go list ./... | grep -v '/integration_test')

echo "Checking coverage thresholds..."
OVERALL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Overall coverage: ${OVERALL_COVERAGE}%"

# Check core business logic packages only
CORE_LINES=$(go tool cover -func=coverage.out | \
  grep -E 'github.com/pingxin403/cuckoo/apps/shortener-service/(analytics|cache|errors|idgen|service)/' | \
  grep -v 'l2_cache.go' || true)

if [ -n "$CORE_LINES" ]; then
    CORE_COVERAGE=$(echo "$CORE_LINES" | \
      awk '{sum+=$3; count++} END {if (count > 0) print sum/count; else print 0}' | \
      sed 's/%//')
    echo "Core packages coverage: ${CORE_COVERAGE}%"
    
    if (( $(echo "$CORE_COVERAGE < 70" | bc -l) )); then
        echo "❌ FAIL: Core packages coverage ${CORE_COVERAGE}% is below 70% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Core packages coverage meets 70% threshold"
fi
```

**Java** (Gradle configuration):
```groovy
jacocoTestCoverageVerification {
    violationRules {
        rule {
            limit {
                minimum = 0.30
            }
        }
        rule {
            element = 'CLASS'
            includes = ['*.service.*']
            limit {
                minimum = 0.50
            }
        }
    }
}
```

### Coverage Strategy by Service

**shortener-service** (Go):
- Core packages: 88.0% ✅ (target: 70%)
- Overall: 50.7% (reference only)
- See: `docs/COVERAGE_STRATEGY_SUMMARY.md`

**todo-service** (Go):
- Overall: 70%
- Service layer: 75%

**hello-service** (Java):
- Overall: 30%
- Service layer: 50%

### Excluded from Coverage

**Generated Code**:
- `gen/` directories
- `*_pb.go` files
- `*Grpc.java` files

**Infrastructure Code**:
- `main.go` / `Application.java` - Application bootstrap
- `logger/` - Logging initialization
- Configuration classes
- DTOs and models

**External Dependency Wrappers**:
- `storage/` - Database operations (tested in integration tests)
- `cache/l2_cache.go` - Redis operations (tested in integration tests)
- Message queue producers/consumers (tested in integration tests)

### Testing Strategy

**Unit Tests**:
- Test core business logic in isolation
- Use mocks for external dependencies
- Fast execution (seconds)
- Run in CI on every commit
- **Purpose**: Verify business logic correctness

**Integration Tests**:
- Test service interactions with real dependencies
- Use Docker Compose for dependencies
- Slower execution (minutes)
- Run locally or in dedicated test environments
- **Purpose**: Verify integration with external services
- **Not used for coverage metrics**

### Best Practices

1. **Write unit tests for core logic** - Focus on business logic that doesn't require external services
2. **Use mocks appropriately** - Mock external dependencies in unit tests
3. **Don't game coverage** - Don't use integration tests to inflate coverage numbers
4. **Separate concerns** - Keep business logic separate from infrastructure code
5. **Property-based testing** - Use property tests for universal correctness properties
6. **Integration tests complement** - Use integration tests to verify real-world behavior, not coverage

## Linting

### Java Linting

**Tool**: Checkstyle

**Configuration**: `checkstyle.xml`

**Rules**:
- Google Java Style Guide
- 4-space indentation
- Line length: 120 characters
- No wildcard imports

**Commands**:
```bash
make lint APP=hello
make lint-fix APP=hello  # Auto-fix where possible
```

### Go Linting

**Tool**: golangci-lint

**Configuration**: `.golangci.yml`

**Enabled Linters**:
- gofmt
- goimports
- govet
- errcheck
- staticcheck
- unused

**Commands**:
```bash
make lint APP=todo
make lint-fix APP=todo
```

### TypeScript Linting

**Tools**: ESLint + Prettier

**Configuration**: `.eslintrc.js`, `.prettierrc`

**Rules**:
- 2-space indentation
- Single quotes
- Trailing commas
- No semicolons

**Commands**:
```bash
make lint APP=web
make lint-fix APP=web
```

## Testing Strategy

### Test Levels

**1. Unit Tests**:
- Test individual components in isolation
- Mock external dependencies
- Fast execution (<1s per test)

**2. Property-Based Tests**:
- Verify correctness properties
- Generate random test inputs
- Tools: jqwik (Java), rapid (Go), fast-check (TypeScript)

**3. Integration Tests**:
- Test service interactions with real dependencies
- Use actual databases, caches, and services
- Run in Docker containers
- See [Integration Testing Strategy](./integration-testing.md)

**4. E2E Tests**:
- Test complete user flows
- Browser-based testing
- Tools: Playwright, Cypress

### Property-Based Testing

**Correctness Properties**:

1. **Hello Service Name Inclusion**:
   - For any non-empty name, response contains the name
   - File: `HelloServicePropertyTest.java`

2. **TODO ID Uniqueness**:
   - All created TODO IDs are unique
   - File: `todo_service_property_test.go`

3. **TODO CRUD Consistency**:
   - Create → List: appears in list
   - Update → Get: changes persisted
   - Delete → List: removed from list
   - File: `todo_service_property_test.go`

### Test Organization

```
apps/hello-service/
└── src/test/java/
    ├── unit/
    │   └── HelloServiceTest.java
    └── property/
        └── HelloServicePropertyTest.java

apps/todo-service/
└── service/
    ├── todo_service_test.go
    └── todo_service_property_test.go

apps/web/
└── src/
    ├── components/
    │   ├── HelloForm.test.tsx
    │   └── TodoList.test.tsx
    └── __tests__/
        └── properties/
```

## CI/CD Quality Gates

### GitHub Actions Workflow

**Quality Checks**:
1. Tool version verification
2. Protobuf code generation
3. Linting (all languages)
4. Unit tests
5. Test coverage verification
6. Security scanning
7. Docker image building

**Failure Conditions**:
- Any test fails
- Coverage below threshold
- Linting errors
- Security vulnerabilities found
- Docker build fails

### Security Scanning

**Tools**:
- Trivy for Docker images
- Dependabot for dependencies
- Custom secret scanning

**Scanned For**:
- Known vulnerabilities (CVEs)
- Outdated dependencies
- Hardcoded secrets
- Insecure configurations

## Code Formatting

### Auto-Formatting

**Java**:
```bash
./gradlew spotlessApply
```

**Go**:
```bash
gofmt -w .
goimports -w .
```

**TypeScript**:
```bash
npm run format
```

### Format on Save

**VS Code** (`.vscode/settings.json`):
```json
{
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  }
}
```

## Git Hooks

**Location**: `.githooks/pre-commit`

**Installation**:
```bash
git config core.hooksPath .githooks
```

**What It Does**:
1. Runs `make pre-commit`
2. Blocks commit if checks fail
3. Provides clear error messages
4. Suggests fixes

**Bypass** (use sparingly):
```bash
git commit --no-verify
```

## Best Practices

### Before Committing

1. Run `make pre-commit`
2. Fix any issues found
3. Verify tests pass
4. Check coverage reports

### Before Creating PR

1. Rebase on latest main
2. Run full test suite
3. Update documentation
4. Add tests for new features

### Code Review Checklist

- [ ] All CI checks pass
- [ ] Code follows style guide
- [ ] Tests added for new features
- [ ] Documentation updated
- [ ] No security issues
- [ ] Coverage thresholds met

## Monitoring Quality

### Metrics Tracked

**Build Health**:
- CI build time
- Test execution time
- Build success rate

**Code Quality**:
- Test coverage percentage
- Linting violations
- Code duplication

**Security**:
- Vulnerability count
- Dependency age
- Secret scan results

### Quality Dashboard

**Future Enhancement**: Backstage integration
- Real-time quality metrics
- Trend analysis
- Team scorecards

## References

- [Shift-Left Documentation](../../docs/SHIFT_LEFT.md)
- [Testing Guide](../../docs/TESTING_GUIDE.md)
- [Integration Testing Strategy](./integration-testing.md)
- [Coverage Strategy Summary](../../docs/COVERAGE_STRATEGY_SUMMARY.md)
- [Linting Guide](../../docs/LINTING_GUIDE.md)
- [Lint Fix Guide](../../docs/LINT_FIX_GUIDE.md)
