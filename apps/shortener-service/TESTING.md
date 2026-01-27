# Testing Guide - URL Shortener Service

This document provides comprehensive testing guidance for the URL Shortener Service.

## Test Structure

```
shortener-service/
├── service/
│   ├── shortener_service_service_test.go      # Unit tests
│   ├── shortener_service_property_test.go     # Property-based tests
│   ├── url_validator_test.go                  # URL validation tests
│   ├── url_validator_property_test.go         # URL validation properties
│   ├── rate_limiter_test.go                   # Rate limiter tests
│   ├── rate_limiter_property_test.go          # Rate limiter properties
│   └── redirect_handler_test.go               # HTTP redirect tests
├── storage/
│   ├── mysql_store_test.go                    # Storage unit tests
│   └── mysql_store_property_test.go           # Storage properties
├── cache/
│   ├── l1_cache_test.go                       # L1 cache tests
│   ├── l1_cache_property_test.go              # L1 cache properties
│   ├── l2_cache_test.go                       # L2 cache tests
│   ├── l2_cache_property_test.go              # L2 cache properties
│   ├── cache_manager_test.go                  # Cache manager tests
│   └── cache_manager_property_test.go         # Cache manager properties
├── idgen/
│   ├── id_generator_test.go                   # ID generator tests
│   └── id_generator_property_test.go          # ID generator properties
├── analytics/
│   ├── analytics_writer_test.go               # Analytics tests
│   └── analytics_writer_property_test.go      # Analytics properties
└── integration_test/
    └── integration_test.go                    # End-to-end tests
```

## Running Tests

### Quick Test Commands

```bash
# Run all unit tests (fast, ~2 seconds)
go test ./...

# Run all tests including property-based tests (~10 minutes)
go test ./... -tags=property

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage including property tests
go test -v -race -coverprofile=coverage.out -tags=property ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run coverage verification script (80% overall, 90% service)
./scripts/test-coverage.sh

# Run integration tests
./scripts/run-integration-tests.sh
```

### Makefile Commands (from monorepo root)

```bash
# Run tests for shortener service
make test APP=shortener

# Run linter
make lint APP=shortener

# Run all pre-commit checks
make pre-commit
```

## Coverage Requirements

The service enforces test coverage thresholds:
- **Overall coverage**: 80% minimum
- **Service package**: 90% minimum

These thresholds are verified in CI and will fail the build if not met.

### Current Coverage

```bash
# Check current coverage
./scripts/test-coverage.sh

# Example output:
# ✓ Overall coverage: 85.2% (threshold: 80%)
# ✓ Service package coverage: 92.1% (threshold: 90%)
```

## Unit Testing

### Test Framework

- **Go testing package**: Standard library testing
- **testify/assert**: Fluent assertions
- **testify/require**: Assertions that stop test on failure
- **testify/mock**: Mocking framework

### Example Unit Test

```go
package service

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestShortenerService_CreateShortLink(t *testing.T) {
    // Arrange
    service := NewShortenerService(mockStorage, mockCache)
    ctx := context.Background()
    req := &pb.CreateShortLinkRequest{
        LongUrl: "https://example.com/very/long/url",
    }
    
    // Act
    resp, err := service.CreateShortLink(ctx, req)
    
    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, resp.ShortCode)
    assert.Equal(t, 7, len(resp.ShortCode))
    assert.Contains(t, resp.ShortUrl, resp.ShortCode)
}
```

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestURLValidator_Validate(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr bool
        errCode ErrorCode
    }{
        {
            name:    "valid https url",
            url:     "https://example.com",
            wantErr: false,
        },
        {
            name:    "valid http url",
            url:     "http://example.com",
            wantErr: false,
        },
        {
            name:    "invalid scheme",
            url:     "ftp://example.com",
            wantErr: true,
            errCode: ErrInvalidURL,
        },
        {
            name:    "url too long",
            url:     "https://example.com/" + strings.Repeat("a", 3000),
            wantErr: true,
            errCode: ErrURLTooLong,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            validator := NewURLValidator()
            err := validator.Validate(tt.url)
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Equal(t, tt.errCode, err.Code)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

## Property-Based Testing

### Framework

The service uses **pgregory.net/rapid** for property-based testing, which generates random test data to verify properties hold across many inputs.

### Build Tags

Property-based tests use build tags to separate them from fast unit tests:

```go
//go:build property
// +build property

package service

import (
    "testing"
    "pgregory.net/rapid"
)
```

This allows:
- Fast unit tests: `go test ./...` (~2 seconds)
- Full test suite: `go test ./... -tags=property` (~10 minutes)

### Example Property Test

```go
//go:build property
// +build property

package idgen

import (
    "testing"
    "pgregory.net/rapid"
)

// Property: Generated IDs are always 7 characters
func TestIDGenerator_LengthProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random counter value
        counter := rapid.Uint64().Draw(t, "counter")
        
        // Generate ID
        gen := NewIDGenerator()
        id := gen.Generate(counter)
        
        // Property: ID length is always 7
        if len(id) != 7 {
            t.Fatalf("expected length 7, got %d for counter %d", len(id), counter)
        }
    })
}

// Property: Generated IDs are unique for different counters
func TestIDGenerator_UniquenessProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        gen := NewIDGenerator()
        seen := make(map[string]uint64)
        
        // Generate multiple IDs
        for i := 0; i < 100; i++ {
            counter := rapid.Uint64().Draw(t, "counter")
            id := gen.Generate(counter)
            
            // Property: Each ID maps to unique counter
            if prevCounter, exists := seen[id]; exists {
                if prevCounter != counter {
                    t.Fatalf("collision: ID %s generated for both %d and %d", 
                        id, prevCounter, counter)
                }
            }
            seen[id] = counter
        }
    })
}
```

### Property Test Patterns

1. **Invariants**: Properties that always hold
   ```go
   rapid.Check(t, func(t *rapid.T) {
       input := rapid.String().Draw(t, "input")
       result := service.Process(input)
       // Property: result is never nil
       if result == nil {
           t.Fatal("result should never be nil")
       }
   })
   ```

2. **Round-trip**: Encode then decode returns original
   ```go
   rapid.Check(t, func(t *rapid.T) {
       original := rapid.Uint64().Draw(t, "original")
       encoded := gen.Encode(original)
       decoded := gen.Decode(encoded)
       // Property: decode(encode(x)) == x
       if decoded != original {
           t.Fatalf("round-trip failed: %d -> %s -> %d", 
               original, encoded, decoded)
       }
   })
   ```

3. **Idempotence**: Applying operation twice gives same result
   ```go
   rapid.Check(t, func(t *rapid.T) {
       input := rapid.String().Draw(t, "input")
       first := service.Normalize(input)
       second := service.Normalize(first)
       // Property: normalize(normalize(x)) == normalize(x)
       if first != second {
           t.Fatalf("not idempotent: %s -> %s -> %s", 
               input, first, second)
       }
   })
   ```

## Integration Testing

Integration tests verify the complete service functionality with real MySQL and Redis instances.

### Running Integration Tests

```bash
# Automated script (recommended)
./scripts/run-integration-tests.sh

# Manual steps:
# 1. Start test environment
docker compose -f docker-compose.test.yml up -d

# 2. Wait for services to be healthy
docker compose -f docker-compose.test.yml ps

# 3. Run integration tests
GRPC_ADDR="localhost:9092" BASE_URL="http://localhost:8081" \
  go test -v -tags=integration ./integration_test/... -timeout 5m

# 4. Stop test environment
docker compose -f docker-compose.test.yml down -v
```

### Integration Test Coverage

- ✅ End-to-end flow: Create → Retrieve → Redirect → Delete
- ✅ Custom short code functionality
- ✅ Link expiration handling (410 Gone)
- ✅ Cache warming and performance
- ✅ URL validation and security
- ✅ Concurrent creation
- ✅ Health check endpoints

### Test Environment

- **MySQL 8.0**: Port 3307
- **Redis 7.2**: Port 6380
- **Shortener Service**:
  - gRPC: Port 9092
  - HTTP redirect: Port 8081
  - Metrics: Port 9091

## Mocking

### Mock Storage

```go
type MockStorage struct {
    mock.Mock
}

func (m *MockStorage) Create(ctx context.Context, mapping *URLMapping) error {
    args := m.Called(ctx, mapping)
    return args.Error(0)
}

func (m *MockStorage) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
    args := m.Called(ctx, shortCode)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*URLMapping), args.Error(1)
}

// Usage in tests
func TestWithMock(t *testing.T) {
    mockStorage := new(MockStorage)
    mockStorage.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    service := NewShortenerService(mockStorage, nil)
    // Test service...
    
    mockStorage.AssertExpectations(t)
}
```

## Best Practices

### Test Organization

1. **Arrange-Act-Assert (AAA)**: Structure tests clearly
2. **One assertion per test**: Keep tests focused (or related assertions)
3. **Descriptive names**: Use `TestComponent_Method_Scenario` format
4. **Test edge cases**: Empty strings, nulls, boundaries, errors
5. **Use subtests**: Group related tests with `t.Run()`

### Coverage Guidelines

- **Focus on business logic**: Service layer should have 90%+ coverage
- **Test error paths**: Verify all error conditions
- **Test boundaries**: Min/max values, empty inputs, large inputs
- **Property tests for algorithms**: Use rapid for ID generation, encoding, etc.
- **Integration tests for flows**: Verify end-to-end scenarios

### Performance

- **Keep unit tests fast**: Should run in milliseconds
- **Use build tags for slow tests**: Separate property tests
- **Parallel execution**: Use `t.Parallel()` for independent tests
- **Mock external dependencies**: Don't call real databases in unit tests

## Continuous Integration

Tests run automatically in CI on:
- Pull requests
- Commits to main branch
- Scheduled nightly builds

CI enforces:
- All tests must pass
- Coverage thresholds must be met (80% overall, 90% service)
- No race conditions (`-race` flag)
- Linting passes (`golangci-lint`)

## Troubleshooting

### Tests Fail Locally But Pass in CI

```bash
# Clean test cache
go clean -testcache

# Run with verbose output
go test -v ./...

# Check for race conditions
go test -race ./...
```

### Coverage Below Threshold

```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check specific package coverage
go test -coverprofile=coverage.out ./service/...
go tool cover -func=coverage.out
```

### Property Tests Fail Intermittently

```bash
# Run with specific seed for reproducibility
go test -tags=property ./... -rapid.seed=12345

# Reduce iterations for debugging
go test -tags=property ./... -rapid.checks=10

# Run specific property test
go test -tags=property -run TestIDGenerator_UniquenessProperty ./idgen/...
```

### Integration Tests Fail

```bash
# Check Docker containers are running
docker compose -f docker-compose.test.yml ps

# Check MySQL is ready
docker compose -f docker-compose.test.yml exec mysql mysql -uroot -ppassword -e "SELECT 1"

# Check Redis is ready
docker compose -f docker-compose.test.yml exec redis redis-cli ping

# View service logs
docker compose -f docker-compose.test.yml logs shortener-service

# Clean up and restart
docker compose -f docker-compose.test.yml down -v
./scripts/run-integration-tests.sh
```

### Slow Test Execution

```bash
# Run tests in parallel
go test -parallel 4 ./...

# Profile test execution
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Identify slow tests
go test -v ./... 2>&1 | grep -E "PASS|FAIL" | grep -E "[0-9]+\.[0-9]+s"
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Rapid (Property Testing)](https://pkg.go.dev/pgregory.net/rapid)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Monorepo Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)

## Support

For questions about testing:
- Review existing test examples in the codebase
- Check the monorepo testing documentation
- Contact backend-go-team
