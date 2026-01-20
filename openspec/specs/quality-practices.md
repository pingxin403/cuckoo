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

### Coverage Requirements

**Go Services**:
- Overall: 70%
- Service layer: 75%
- Excludes: generated code, main.go

**Java Services**:
- Overall: 30%
- Service layer: 50%
- Excludes: generated code, configuration classes

### Coverage Scripts

**Go** (`apps/todo-service/scripts/test-coverage.sh`):
```bash
#!/bin/bash
go test -coverprofile=coverage.out \
  -coverpkg=./service,./storage \
  ./...

go tool cover -func=coverage.out
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

### Excluded from Coverage

**Generated Code**:
- `gen/` directories
- `*_pb.go` files
- `*Grpc.java` files

**Non-Business Logic**:
- `main.go` / `Application.java`
- Configuration classes
- DTOs and models

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
- [Linting Guide](../../docs/LINTING_GUIDE.md)
- [Lint Fix Guide](../../docs/LINT_FIX_GUIDE.md)
