# Testing Guide

This document explains the testing strategy for the monorepo.

## Test Types

### 1. Unit Tests (Fast)
- **Location**: `*_test.go` files (excluding `*_property_test.go`)
- **Runtime**: < 5 seconds per service
- **Purpose**: Test individual functions and components
- **Run by default**: Yes

### 2. Property-Based Tests (Slow)
- **Location**: `*_property_test.go` files
- **Runtime**: 5-10 minutes per service (100+ iterations per property)
- **Purpose**: Verify correctness properties across many random inputs
- **Run by default**: No (requires `-tags=property`)

## Running Tests

### Quick Test (Unit Tests Only)
```bash
# Test all apps (fast, ~10 seconds)
make test

# Test specific app (fast, ~2 seconds)
make test APP=im

# Test specific Go service directly
cd apps/im-service
go test ./...
```

### Full Test (Including Property Tests)
```bash
# Test specific app with property tests
cd apps/im-service
go test ./... -tags=property -timeout=30m

# Test specific package with property tests
cd apps/im-service
go test ./registry/... -tags=property -timeout=10m
```

### Test with Coverage
```bash
# Unit test coverage (fast)
make test-coverage APP=im

# Full coverage including property tests
cd apps/im-service
go test ./... -tags=property -coverprofile=coverage.out -timeout=30m
go tool cover -html=coverage.out
```

## Why Separate Property Tests?

Property-based tests are **extremely valuable** but **very slow**:

1. **TTL Tests**: Wait for actual timeouts (1-2 seconds per iteration)
2. **Iterations**: Run 100+ times per property
3. **Total Time**: 5-10 minutes per service

By using build tags, we:
- ✅ Keep CI/CD fast (unit tests only)
- ✅ Run property tests on-demand or in nightly builds
- ✅ Maintain high confidence with both test types

## Build Tags Explained

Go build tags allow conditional compilation:

```go
//go:build property
// +build property

package mypackage
// This file only compiles when -tags=property is specified
```

**Without tag** (default):
```bash
go test ./...  # Runs only unit tests (fast)
```

**With tag**:
```bash
go test ./... -tags=property  # Runs ALL tests (slow)
```

## CI/CD Strategy

### Pull Request Checks (Fast)
```bash
make test  # Unit tests only, ~30 seconds
make lint
```

### Nightly Builds (Comprehensive)
```bash
go test ./... -tags=property -timeout=1h  # All tests
```

### Pre-Release (Full Validation)
```bash
go test ./... -tags=property -timeout=2h
make test-coverage
```

## Test Naming Conventions

- `Test*`: Regular unit tests
- `TestProperty_*`: Property-based tests (require `-tags=property`)
- `Test*_Integration`: Integration tests (may require external services)

## Adding New Tests

### Unit Test
```go
// my_test.go
func TestMyFunction(t *testing.T) {
    // Fast, focused test
}
```

### Property Test
```go
//go:build property
// +build property

// my_property_test.go
func TestProperty_MyInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Test property across many inputs
    })
}
```

## Troubleshooting

### Tests Hang or Take Too Long
- You're probably running property tests unintentionally
- Solution: Make sure property test files have build tags
- Run: `./scripts/add-property-tags.sh` to add tags automatically

### Property Tests Not Running
- Make sure you include `-tags=property`
- Check file has correct build tags at the top

### Coverage Seems Low
- Unit test coverage excludes property tests by default
- Run with `-tags=property` for full coverage

## Service-Specific Test Commands

### IM Service
```bash
cd apps/im-service

# Fast: Unit tests only (~2s)
go test ./...

# Slow: All tests (~10m)
go test ./... -tags=property -timeout=30m

# Custom script with options
./scripts/test-coverage.sh                    # Unit tests only
./scripts/test-coverage.sh --with-property-tests  # All tests
```

### Auth Service
```bash
cd apps/auth-service

# Fast: Unit tests only
go test ./...

# Slow: All tests
go test ./... -tags=property -timeout=10m
```

### User Service
```bash
cd apps/user-service

# Fast: Unit tests only
go test ./...

# Slow: All tests
go test ./... -tags=property -timeout=10m
```

## Best Practices

1. **Write unit tests first** - Fast feedback during development
2. **Add property tests for critical logic** - Verify correctness properties
3. **Run unit tests frequently** - Every commit
4. **Run property tests before merging** - Catch edge cases
5. **Use appropriate timeouts** - Property tests need longer timeouts

## References

- [Go Build Tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Property-Based Testing with rapid](https://pkg.go.dev/pgregory.net/rapid)
- [Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
