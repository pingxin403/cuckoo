# IM Service Testing Guide

## Quick Start

### Fast Tests (Unit Tests Only)
Run only unit tests, skipping slow property-based tests:

```bash
# From im-service directory
go test ./... -run "^Test[^P]" -v

# Or with coverage
go test ./... -run "^Test[^P]" -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Time**: ~1 second

### Full Test Suite (Including Property Tests)
Run all tests including property-based tests:

```bash
# From im-service directory
go test ./... -timeout=10m -v

# Or use make from root
make test APP=im
```

**Time**: ~7-8 minutes (property tests with TTL waits)

## Test Categories

### Unit Tests
- **Pattern**: `Test*` (excluding `TestProperty*`)
- **Speed**: Fast (< 1 second total)
- **Coverage**: Basic functionality, edge cases, error handling
- **Run**: `go test ./... -run "^Test[^P]"`

### Property-Based Tests
- **Pattern**: `TestProperty*`
- **Speed**: Slow (6-7 minutes total)
- **Coverage**: Correctness properties across many random inputs
- **Iterations**: 100 per property
- **Run**: `go test ./... -run "TestProperty"`

## Why Are Property Tests Slow?

Property-based tests validate system correctness across many scenarios:

1. **Registry TTL Tests**: Wait for lease expiration (1.5s × 100 iterations = 150s)
2. **Lease Renewal Tests**: Wait for TTL periods (2.5s × 100 iterations = 250s)
3. **Total**: ~400 seconds (6-7 minutes) for all property tests

These tests are **critical for correctness** but not needed for every development cycle.

## Recommended Workflow

### During Development
```bash
# Quick feedback loop
go test ./... -run "^Test[^P]" -v
```

### Before Commit
```bash
# Run linter
golangci-lint run ./...

# Run unit tests with coverage
go test ./... -run "^Test[^P]" -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

### Before Push / In CI
```bash
# Full test suite
go test ./... -timeout=10m -v
```

## Test Organization

```
apps/im-service/
├── registry/
│   ├── registry_client.go
│   ├── registry_client_test.go          # 22 unit tests (fast)
│   └── registry_client_property_test.go # 8 property tests (slow)
└── sequence/
    ├── sequence_generator.go
    ├── sequence_generator_test.go       # 19 unit tests (fast)
    ├── sequence_generator_property_test.go # 6 property tests (slow)
    ├── sequence_backup.go
    └── sequence_backup_test.go          # 7 unit tests (fast)
```

## Coverage Targets

- **Unit Tests**: 80%+ statement coverage
- **Property Tests**: Validate correctness properties
- **Combined**: 90%+ for service packages

## CI/CD Integration

### Fast CI (Pull Request)
```yaml
- name: Unit Tests
  run: go test ./... -run "^Test[^P]" -timeout=5m
```

### Full CI (Main Branch)
```yaml
- name: Full Test Suite
  run: go test ./... -timeout=15m
```

## Troubleshooting

### Tests Hang
- **Cause**: Property tests with TTL waits
- **Solution**: Use `-run "^Test[^P]"` to skip property tests

### Tests Timeout
- **Cause**: Default 10m timeout too short
- **Solution**: Increase timeout with `-timeout=15m`

### Want Faster Property Tests
- **Option 1**: Reduce iterations (edit test files, change rapid.Check iterations)
- **Option 2**: Run in parallel: `go test ./... -run TestProperty -parallel=4`
- **Option 3**: Skip in development, run in CI only

## Custom Test Script

Use the provided test script for convenience:

```bash
# Fast unit tests only
./scripts/test-coverage.sh

# Include property tests
./scripts/test-coverage.sh --with-property-tests

# Custom timeout
./scripts/test-coverage.sh --with-property-tests --timeout 20m
```
