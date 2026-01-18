# Shift-Left Quality Practices

This document describes the shift-left quality practices implemented in this monorepo to catch issues early in the development cycle.

## Overview

"Shift-left" means moving quality checks earlier in the development process. Instead of finding bugs in CI or production, we catch them during development and before commit.

## Pre-Commit Quality Checks

### Automated Checks

Every commit automatically runs these checks via git hooks:

1. **Tool Version Verification** - Ensures you're using the correct tool versions
2. **Protobuf Code Generation** - Verifies generated code is up to date
3. **Linting** - Checks code style and potential issues
4. **Testing** - Runs unit tests for changed code
5. **Common Issues** - Detects console.log, TODOs, large files
6. **Security** - Scans for potential secrets in code

### Manual Pre-Commit Check

Run all checks manually before committing:

```bash
make pre-commit
```

This is useful when:
- You want to verify changes before staging
- The git hook is disabled
- You're troubleshooting issues

## Test Coverage Scope

### What's Included in Coverage

Coverage metrics only include **business logic code**:

- Service layer (business logic)
- Storage layer (data access)
- Client code (API clients)
- Utility functions

### What's Excluded from Coverage

The following are excluded from coverage requirements:

- **Generated code** (`gen/`, `generated/`, `proto/`)
- **Main entry points** (`main.go`, `*Application.java`)
- **Configuration classes** (`*Config.java`, `*Configuration.java`)
- **Build artifacts**

### Coverage Thresholds

#### Go Services (TODO Service)

```bash
# Overall: 80% (excluding generated code and main.go)
# Service package: 90%
# Storage package: 90%
```

#### Java Services (Hello Service)

```gradle
// Overall: 30% (TODO: increase to 80%)
// Service classes: 50% (TODO: increase to 90%)
```

### Why Exclude Generated Code?

Generated code (protobuf, etc.) should not be tested because:
- It's automatically generated and maintained by tools
- Testing it provides no value (we trust the code generator)
- It inflates coverage metrics without improving quality
- It wastes CI time and developer effort

## Linting Strategy

### Unified Lint Command

```bash
# Lint all changed apps
make lint

# Lint specific app
make lint APP=hello
make lint APP=todo
make lint APP=web

# Auto-fix linting issues
make lint-fix
make lint-fix APP=hello
```

### Language-Specific Linters

#### Java (Hello Service)
- **Spotless** - Code formatting (Google Java Format)
- **SpotBugs** - Bug detection
- ~~Checkstyle~~ - Disabled (too strict, use Spotless instead)

#### Go (TODO Service)
- **gofmt** - Standard Go formatting
- **golangci-lint** - Comprehensive linting (if installed)

#### TypeScript (Web)
- **ESLint** - Code quality and style
- **Prettier** - Code formatting

### Linting in CI

All linting runs automatically in CI:
- Fails the build if issues are found
- Provides detailed error messages
- Suggests fix commands

## Additional Shift-Left Practices

### 1. Local Development Checks

Before starting work:
```bash
# Verify your environment
make check-versions

# Ensure dependencies are up to date
make init
```

### 2. Continuous Feedback

While developing:
```bash
# Run tests in watch mode (if supported)
cd apps/todo-service && go test ./... -watch

# Run linter on save (configure your IDE)
```

### 3. Pre-Push Validation

Before pushing to remote:
```bash
# Run full test suite
make test

# Verify coverage
make test-coverage

# Check for issues
make pre-commit
```

### 4. IDE Integration

Configure your IDE to:
- Run formatters on save
- Show linting errors inline
- Run tests automatically
- Highlight coverage gaps

### 5. Code Review Checklist

Before requesting review:
- [ ] All tests pass locally
- [ ] Coverage meets thresholds
- [ ] No linting errors
- [ ] No console.log or debug statements
- [ ] No TODOs without tracking issues
- [ ] No secrets or sensitive data
- [ ] Generated code is up to date

## Bypassing Checks (Not Recommended)

### Skip Git Hook

If you absolutely must skip the pre-commit hook:

```bash
git commit --no-verify -m "message"
```

**Warning**: This bypasses all quality checks. Use only in emergencies.

### Skip Coverage in CI

Coverage checks run during `gradle check` or `go test`. To skip:

```bash
# Java - run build without check
./gradlew build -x check

# Go - run tests without coverage
go test ./... -short
```

**Warning**: CI will still enforce coverage requirements.

## Troubleshooting

### Pre-commit checks are slow

The checks only run for changed files. If they're still slow:

1. Run specific checks:
   ```bash
   make lint APP=hello  # Only lint one app
   make test APP=todo   # Only test one app
   ```

2. Disable coverage verification locally:
   ```bash
   # Go
   go test ./... -short
   
   # Java
   ./gradlew test -x jacocoTestCoverageVerification
   ```

3. Use `--no-verify` for emergency commits (not recommended)

### Coverage is failing for generated code

Check that exclusions are configured:

**Go** (`apps/todo-service/scripts/test-coverage.sh`):
```bash
grep -v "/gen/" "$COVERAGE_FILE" | grep -v "main.go"
```

**Java** (`apps/hello-service/build.gradle`):
```gradle
exclude: [
    '**/gen/**',
    '**/generated/**',
    '**/proto/**',
    '**/*Application.class',
    '**/*Config.class'
]
```

### Linting is failing

1. Try auto-fix:
   ```bash
   make lint-fix
   ```

2. Check specific errors:
   ```bash
   make lint APP=hello
   ```

3. Review linting rules in:
   - Java: `apps/hello-service/build.gradle` (Spotless config)
   - Go: `.golangci.yml` (if exists)
   - TypeScript: `apps/web/.eslintrc.cjs`

## Best Practices

### 1. Commit Often

Small, frequent commits are easier to review and test:
- Each commit should be a logical unit
- Pre-commit checks run faster on small changes
- Easier to bisect if issues arise

### 2. Fix Issues Immediately

Don't accumulate technical debt:
- Fix linting errors as they appear
- Address test failures immediately
- Don't commit with TODOs unless tracked

### 3. Keep Coverage High

Maintain high test coverage:
- Write tests as you write code
- Test edge cases and error conditions
- Use property-based testing for complex logic

### 4. Review Generated Code

Even though generated code is excluded from coverage:
- Review proto changes carefully
- Regenerate code after proto changes
- Commit generated code to git

### 5. Use the Tools

Take advantage of automation:
- Let formatters handle style
- Let linters catch bugs
- Let tests verify behavior
- Let git hooks enforce quality

## See Also

- [Testing Guide](TESTING_GUIDE.md) - Comprehensive testing documentation
- [Linting Guide](LINTING_GUIDE.md) - Linting configuration and usage
- [Code Quality](CODE_QUALITY.md) - Overall code quality standards
- [Quick Reference](QUICK_REFERENCE.md) - Common commands
