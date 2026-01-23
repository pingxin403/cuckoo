# Testing Guide for {{SERVICE_NAME}}

This document provides comprehensive testing guidelines for the {{SERVICE_NAME}} service.

## Test Organization

### Test Types

1. **Unit Tests** (`*_test.go`)
   - Test individual functions and methods
   - Fast execution (< 1 second)
   - No external dependencies
   - Run by default with `go test ./...`

2. **Property-Based Tests** (`*_property_test.go`)
   - Test universal properties across many inputs
   - Use `pgregory.net/rapid` framework
   - Separated with build tags for performance
   - Run with `go test ./... -tags=property`

3. **Integration Tests** (`integration_test/`)
   - Test complete workflows with real dependencies
   - Use Docker Compose for infrastructure
   - Run with `./scripts/run-integration-tests.sh`

## Running Tests

### Quick Test (Unit Tests Only)

```bash
# Fast unit tests only (default)
go test ./...

# With verbose output
go test -v ./...

# Specific package
go test ./service/...
```

### Full Test Suite (Including Property Tests)

```bash
# All tests including property-based tests
go test ./... -tags=property

# With coverage
go test -v -race -coverprofile=coverage.out ./... -tags=property

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Verification

```bash
# Run coverage verification script
./scripts/test-coverage.sh

# This checks:
# - Overall coverage >= 80%
# - Service package coverage >= 90%
```

### Integration Tests

```bash
# Run integration tests with Docker
./scripts/run-integration-tests.sh
```

## Writing Unit Tests

### Basic Test Structure

```go
package service

import (
    "context"
    "testing"
)

func TestServiceMethod(t *testing.T) {
    // Arrange
    service := NewService()
    ctx := context.Background()
    
    // Act
    result, err := service.Method(ctx, input)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Table-Driven Tests

```go
func TestServiceMethod_Scenarios(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "result",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := NewService()
            got, err := service.Method(context.Background(), tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Method() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Method() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing with Mocks

```go
// Use interfaces for dependencies
type Store interface {
    Get(id string) (*Item, error)
    Save(item *Item) error
}

// Mock implementation for testing
type mockStore struct {
    items map[string]*Item
}

func (m *mockStore) Get(id string) (*Item, error) {
    item, ok := m.items[id]
    if !ok {
        return nil, errors.New("not found")
    }
    return item, nil
}

func TestWithMock(t *testing.T) {
    store := &mockStore{
        items: map[string]*Item{
            "1": {ID: "1", Name: "test"},
        },
    }
    
    service := NewService(store)
    // Test service with mock store
}
```

## Writing Property-Based Tests

### Build Tags

All property-based tests MUST include build tags:

```go
//go:build property
// +build property

package service

import (
    "testing"
    "pgregory.net/rapid"
)
```

### Basic Property Test

```go
func TestProperty_AlwaysReturnsNonEmpty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random input
        input := rapid.String().Draw(t, "input")
        
        // Execute function
        result := ProcessInput(input)
        
        // Verify property
        if result == "" {
            t.Fatalf("result should never be empty, got empty for input: %q", input)
        }
    })
}
```

### Custom Generators

```go
// Generate valid user IDs
func genUserID() *rapid.Generator[string] {
    return rapid.Custom(func(t *rapid.T) string {
        prefix := rapid.SampledFrom([]string{"user", "admin", "guest"}).Draw(t, "prefix")
        number := rapid.IntRange(1, 9999).Draw(t, "number")
        return fmt.Sprintf("%s_%d", prefix, number)
    })
}

func TestProperty_WithCustomGenerator(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        userID := genUserID().Draw(t, "userID")
        
        // Test with generated user ID
        result := ValidateUserID(userID)
        if !result {
            t.Fatalf("valid user ID rejected: %s", userID)
        }
    })
}
```

### Property Test Examples

**Idempotence:**
```go
func TestProperty_Idempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        
        result1 := Process(input)
        result2 := Process(input)
        
        if result1 != result2 {
            t.Fatalf("not idempotent: %v != %v", result1, result2)
        }
    })
}
```

**Round-trip:**
```go
func TestProperty_RoundTrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        original := rapid.String().Draw(t, "original")
        
        encoded := Encode(original)
        decoded := Decode(encoded)
        
        if decoded != original {
            t.Fatalf("round-trip failed: %q != %q", decoded, original)
        }
    })
}
```

**Invariants:**
```go
func TestProperty_LengthInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        items := rapid.SliceOf(rapid.String()).Draw(t, "items")
        
        result := ProcessList(items)
        
        if len(result) != len(items) {
            t.Fatalf("length changed: %d != %d", len(result), len(items))
        }
    })
}
```

## Coverage Requirements

### Targets

- **Overall coverage**: 80% minimum
- **Service package**: 90% minimum

### Checking Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by package
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

### Improving Coverage

1. **Identify uncovered code:**
   ```bash
   go tool cover -html=coverage.out
   ```

2. **Add tests for uncovered paths:**
   - Error handling paths
   - Edge cases
   - Boundary conditions

3. **Use table-driven tests** for multiple scenarios

4. **Add property tests** for universal properties

## Best Practices

### DO

✅ Write tests before or alongside code (TDD)
✅ Use descriptive test names (`TestServiceMethod_EmptyInput_ReturnsError`)
✅ Test one thing per test function
✅ Use table-driven tests for multiple scenarios
✅ Test error cases and edge cases
✅ Use property-based tests for universal properties
✅ Keep tests fast (< 1 second for unit tests)
✅ Use build tags for slow tests
✅ Mock external dependencies
✅ Clean up resources in tests (use `defer`)

### DON'T

❌ Don't test implementation details
❌ Don't use real databases/services in unit tests
❌ Don't write flaky tests (time-dependent, order-dependent)
❌ Don't skip error checking in tests
❌ Don't use `time.Sleep()` for synchronization
❌ Don't commit commented-out tests
❌ Don't test third-party library code
❌ Don't write tests that depend on each other

## Continuous Integration

Tests run automatically in CI on:
- Every pull request
- Every commit to main branch
- Nightly builds (with property tests)

### CI Test Commands

```bash
# Fast unit tests (PR checks)
make test APP={{SHORT_NAME}}

# Full test suite (nightly)
cd apps/{{SERVICE_NAME}} && go test ./... -tags=property

# Coverage verification
cd apps/{{SERVICE_NAME}} && ./scripts/test-coverage.sh
```

## Troubleshooting

### Tests Are Slow

- Check if property tests are running (should use build tags)
- Profile tests: `go test -cpuprofile=cpu.prof ./...`
- Look for `time.Sleep()` calls
- Check for unnecessary setup/teardown

### Flaky Tests

- Avoid time-dependent tests
- Use deterministic random seeds
- Avoid parallel test conflicts
- Check for race conditions: `go test -race ./...`

### Low Coverage

- Run `go tool cover -html=coverage.out` to see uncovered code
- Add tests for error paths
- Add tests for edge cases
- Use property tests for complex logic

### Property Tests Failing

- Check the failing example in the output
- Simplify the generator
- Add constraints to the generator
- Verify the property is correct

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [rapid Documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
