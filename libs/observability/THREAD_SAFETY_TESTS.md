# Thread Safety Tests

## Overview

This document describes the thread safety testing strategy for the observability library. We have two types of tests:

1. **Unit Tests** (`thread_safety_unit_test.go`) - Use no-op implementations, no external dependencies
2. **Integration Tests** (`thread_safety_integration_test.go`) - Use real OTLP connections, require running collector

## Unit Tests (Recommended for CI/CD)

**File**: `thread_safety_unit_test.go`

**Purpose**: Validate thread safety without external dependencies using no-op implementations.

**Tests**:
- `TestThreadSafety_NoOp` - Main test covering all operations with no-op implementations
- `TestThreadSafety_Metrics_NoOp` - Metrics-focused test with no-op collector
- `TestThreadSafety_Logging_NoOp` - Logging-focused test with no-op logger
- `TestThreadSafety_Tracing_NoOp` - Tracing-focused test with no-op tracer
- `TestThreadSafety_MixedOperations_NoOp` - Mixed operations test with no-op implementations
- `TestThreadSafety_SharedState_NoOp` - Shared state synchronization test with no-op collector

**Running Unit Tests**:
```bash
# Run with race detector
go test -v -race -run TestThreadSafety_NoOp ./...

# Run all unit tests
go test -v -race ./...
```

**Validates**: Requirements 11.1-11.7 (thread safety)

**Key Features**:
- No external dependencies (no OTLP collector required)
- Fast execution
- Suitable for CI/CD pipelines
- Uses race detector to catch data races

## Integration Tests (Requires OTLP Collector)

**File**: `thread_safety_integration_test.go`

**Purpose**: Validate thread safety with real OpenTelemetry implementations and OTLP export.

**Tests**:
- `TestProperty_ThreadSafety` - Property-based test with real OTLP connections
- `TestProperty_ThreadSafety_Metrics` - Metrics-focused property test
- `TestProperty_ThreadSafety_Logging` - Logging-focused property test
- `TestProperty_ThreadSafety_Tracing` - Tracing-focused property test
- `TestProperty_ThreadSafety_MixedOperations` - Mixed operations property test
- `TestProperty_ThreadSafety_SharedState` - Shared state property test

**Running Integration Tests**:
```bash
# Start OTLP collector first (e.g., using Docker)
docker run -p 4317:4317 otel/opentelemetry-collector

# Run integration tests
go test -v -race -run TestProperty_ThreadSafety ./...
```

**Validates**: Requirements 11.1-11.7 (thread safety) with real OTLP export

**Key Features**:
- Tests with real OpenTelemetry SDK implementations
- Validates OTLP export under concurrent load
- Uses property-based testing with `rapid` library
- Requires running OTLP collector on `localhost:4317`

## Test Classification

### Unit Tests
- **Location**: `*_test.go` files (except `*_integration_test.go`)
- **Dependencies**: None (uses no-op/mock implementations)
- **Purpose**: Fast feedback, CI/CD friendly
- **Examples**:
  - `thread_safety_unit_test.go`
  - `config_test.go`
  - `observability_test.go`
  - `tracing/otel_unit_test.go`

### Integration Tests
- **Location**: `*_integration_test.go` files
- **Dependencies**: External services (OTLP collector, databases, etc.)
- **Purpose**: End-to-end validation
- **Examples**:
  - `thread_safety_integration_test.go`
  - `integration_test.go`
  - `metrics/otel_integration_test.go`
  - `metrics/internal_metrics_integration_test.go`
  - `logging/otel_integration_test.go`
  - `logging/internal_metrics_integration_test.go`

## Testing Principles

1. **Unit tests MUST NOT depend on external services**
   - Use no-op implementations
   - Use mock objects
   - No database connections
   - No Redis connections
   - No OTLP collectors
   - No etcd connections

2. **Integration tests MAY depend on external services**
   - Clearly document required services
   - Provide setup instructions
   - Use `*_integration_test.go` naming convention

3. **All tests MUST be thread-safe**
   - Run with `-race` flag
   - No data races allowed
   - Proper synchronization primitives

## CI/CD Recommendations

### Fast Feedback (Unit Tests Only)
```bash
# Run only unit tests (exclude integration tests)
go test -v -race -short $(go list ./... | grep -v integration)
```

### Full Validation (All Tests)
```bash
# Start required services (OTLP collector, etc.)
docker-compose up -d

# Run all tests including integration tests
go test -v -race ./...
```

## Property-Based Testing

Integration tests use property-based testing with the `rapid` library to generate random test cases:

- **Minimum iterations**: 100 per property
- **Concurrent goroutines**: 5-50 (randomly generated)
- **Operations per goroutine**: 10-200 (randomly generated)
- **Test duration**: ~2-5 seconds per property

Property-based tests validate universal correctness properties across all valid inputs, providing stronger guarantees than example-based tests.

## Troubleshooting

### Race Detector Warnings
If you see race detector warnings:
1. Check the stack trace to identify the rac location
2. Verify proper use of synchronization primitives (mutexes, atomic operations)
3. Ensure shared state is properly protected

### Integration Test Failures
If integration tests fail:
1. Verify OTLP collector is running on `localhost:4317`
2. Check collector logs for errors
3. Ensure network connectivity
4. Try running unit tests first to isolate the issue

### Slow Tests
If tests are slow:
1. Use `-short` flag to skip property-based tests
2. Run unit tests only (exclude integration tests)
3. Reduce property test iterations (for development only)

## References

- [Requirements Document](../../.kiro/specs/observability-otel-enhancement/requirements.md) - Requirements 11.1-11.7
- [Design Document](../../.kiro/specs/observability-otel-enhancement/design.md) - Property 31: Thread Safety
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
