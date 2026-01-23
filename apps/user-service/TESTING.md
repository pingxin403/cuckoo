# Testing Guide for user-service

This document provides comprehensive testing guidelines for the user-service.

## Test Organization

### Test Types

1. **Unit Tests** (`*_test.go`)
   - Test individual functions and methods
   - Fast execution (< 1 second)
   - Use mocks for external dependencies (MySQL)
   - Run by default with `go test ./...`

2. **Property-Based Tests** (`*_property_test.go`)
   - Test universal properties across many inputs
   - Use `pgregory.net/rapid` framework
   - Separated with build tags for performance
   - Run with `go test ./... -tags=property`

3. **Integration Tests** (`integration_test/`)
   - Test complete workflows with real MySQL database
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

func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    service := NewUserService(mockStore)
    ctx := context.Background()
    
    req := &userpb.CreateUserRequest{
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    // Act
    resp, err := service.CreateUser(ctx, req)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.User.Username != "testuser" {
        t.Errorf("got username %v, want testuser", resp.User.Username)
    }
}
```

### Table-Driven Tests

```go
func TestUserService_ValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   "user@example.com",
            wantErr: false,
        },
        {
            name:    "invalid email - no @",
            email:   "userexample.com",
            wantErr: true,
        },
        {
            name:    "invalid email - no domain",
            email:   "user@",
            wantErr: true,
        },
        {
            name:    "empty email",
            email:   "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing with Mocks

```go
// Mock MySQL store for testing
type mockUserStore struct {
    users map[string]*User
    mu    sync.RWMutex
}

func newMockUserStore() *mockUserStore {
    return &mockUserStore{
        users: make(map[string]*User),
    }
}

func (m *mockUserStore) CreateUser(ctx context.Context, user *User) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if _, exists := m.users[user.ID]; exists {
        return errors.New("user already exists")
    }
    
    m.users[user.ID] = user
    return nil
}

func TestWithMock(t *testing.T) {
    store := newMockUserStore()
    service := NewUserService(store)
    
    // Test service with mock store
    ctx := context.Background()
    req := &userpb.CreateUserRequest{
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    resp, err := service.CreateUser(ctx, req)
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }
    
    // Verify user was created in mock store
    if len(store.users) != 1 {
        t.Errorf("expected 1 user in store, got %d", len(store.users))
    }
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
func TestProperty_UsernameValidation(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random username
        username := rapid.StringMatching("[a-zA-Z0-9_]{3,20}").Draw(t, "username")
        
        // Validate username
        err := ValidateUsername(username)
        
        // Valid usernames should never return error
        if err != nil {
            t.Fatalf("valid username rejected: %s, error: %v", username, err)
        }
    })
}
```

### Custom Generators

```go
// Generate valid email addresses
func genEmail() *rapid.Generator[string] {
    return rapid.Custom(func(t *rapid.T) string {
        localPart := rapid.StringMatching("[a-z]{3,10}").Draw(t, "local")
        domain := rapid.SampledFrom([]string{"example.com", "test.org", "demo.net"}).Draw(t, "domain")
        return localPart + "@" + domain
    })
}

func TestProperty_EmailValidation(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        email := genEmail().Draw(t, "email")
        
        err := ValidateEmail(email)
        if err != nil {
            t.Fatalf("valid email rejected: %s, error: %v", email, err)
        }
    })
}
```

### Property Test Examples

**Idempotence:**
```go
func TestProperty_GetUserIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        userID := rapid.String().Draw(t, "userID")
        
        store := newMockUserStore()
        service := NewUserService(store)
        ctx := context.Background()
        
        // Call GetUser twice
        resp1, err1 := service.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
        resp2, err2 := service.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
        
        // Results should be identical
        if (err1 == nil) != (err2 == nil) {
            t.Fatalf("inconsistent errors: %v vs %v", err1, err2)
        }
        
        if err1 == nil && resp1.User.UserId != resp2.User.UserId {
            t.Fatalf("not idempotent: %v != %v", resp1.User.UserId, resp2.User.UserId)
        }
    })
}
```

**Invariants:**
```go
func TestProperty_UserIDNeverEmpty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        username := rapid.String().Draw(t, "username")
        email := genEmail().Draw(t, "email")
        
        store := newMockUserStore()
        service := NewUserService(store)
        ctx := context.Background()
        
        resp, err := service.CreateUser(ctx, &userpb.CreateUserRequest{
            Username: username,
            Email:    email,
        })
        
        if err == nil && resp.User.UserId == "" {
            t.Fatalf("user ID should never be empty")
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
   - Error handling paths (database errors, validation errors)
   - Edge cases (empty inputs, very long inputs)
   - Boundary conditions (max length, special characters)

3. **Use table-driven tests** for multiple scenarios

4. **Add property tests** for universal properties

## Best Practices

### DO

✅ Write tests before or alongside code (TDD)
✅ Use descriptive test names (`TestUserService_CreateUser_DuplicateUsername_ReturnsError`)
✅ Test one thing per test function
✅ Use table-driven tests for multiple scenarios
✅ Test error cases and edge cases
✅ Use property-based tests for validation logic
✅ Keep tests fast (< 1 second for unit tests)
✅ Use build tags for slow tests
✅ Mock external dependencies (MySQL)
✅ Clean up resources in tests (use `defer`)

### DON'T

❌ Don't test implementation details
❌ Don't use real MySQL database in unit tests
❌ Don't write flaky tests (time-dependent, order-dependent)
❌ Don't skip error checking in tests
❌ Don't use `time.Sleep()` for synchronization
❌ Don't commit commented-out tests
❌ Don't test third-party library code
❌ Don't write tests that depend on each other

## Testing MySQL Store

### Unit Tests with Mocks

```go
func TestMySQLStore_CreateUser(t *testing.T) {
    // Use mock store for unit tests
    store := newMockUserStore()
    
    user := &User{
        ID:       "user123",
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    err := store.CreateUser(context.Background(), user)
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }
}
```

### Integration Tests with Real MySQL

```go
// integration_test/mysql_test.go
//go:build integration
// +build integration

package integration_test

import (
    "context"
    "testing"
    "database/sql"
)

func setupTestDB(t *testing.T) *sql.DB {
    // Connect to test MySQL instance
    db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/user_service_test")
    if err != nil {
        t.Fatalf("failed to connect to test DB: %v", err)
    }
    
    // Clean up tables
    _, err = db.Exec("TRUNCATE TABLE users")
    if err != nil {
        t.Fatalf("failed to clean test DB: %v", err)
    }
    
    return db
}

func TestIntegration_CreateUser(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    store := NewMySQLStore(db)
    
    // Test with real database
    user := &User{
        ID:       "user123",
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    err := store.CreateUser(context.Background(), user)
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }
    
    // Verify user was created
    retrieved, err := store.GetUser(context.Background(), "user123")
    if err != nil {
        t.Fatalf("GetUser failed: %v", err)
    }
    
    if retrieved.Username != "testuser" {
        t.Errorf("got username %v, want testuser", retrieved.Username)
    }
}
```

## Continuous Integration

Tests run automatically in CI on:
- Every pull request
- Every commit to main branch
- Nightly builds (with property tests)

### CI Test Commands

```bash
# Fast unit tests (PR checks)
make test APP=user

# Full test suite (nightly)
cd apps/user-service && go test ./... -tags=property

# Coverage verification
cd apps/user-service && ./scripts/test-coverage.sh
```

## Troubleshooting

### Tests Are Slow

- Check if property tests are running (should use build tags)
- Profile tests: `go test -cpuprofile=cpu.prof ./...`
- Look for `time.Sleep()` calls
- Check for unnecessary database operations

### Flaky Tests

- Avoid time-dependent tests
- Use deterministic random seeds
- Avoid parallel test conflicts
- Check for race conditions: `go test -race ./...`

### Low Coverage

- Run `go tool cover -html=coverage.out` to see uncovered code
- Add tests for error paths (database errors, validation errors)
- Add tests for edge cases (empty inputs, max length)
- Use property tests for validation logic

### Property Tests Failing

- Check the failing example in the output
- Simplify the generator
- Add constraints to the generator (e.g., valid email format)
- Verify the property is correct

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [rapid Documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [MySQL Testing Best Practices](https://github.com/go-sql-driver/mysql/wiki/Testing)
