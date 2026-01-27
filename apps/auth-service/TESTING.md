# Testing Guide for auth-service

This document provides comprehensive testing guidelines for the auth-service.

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
   - Test complete authentication workflows
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

func TestAuthService_Authenticate(t *testing.T) {
    // Arrange
    service := NewAuthService()
    ctx := context.Background()
    
    req := &authpb.AuthRequest{
        Username: "testuser",
        Password: "testpass",
    }
    
    // Act
    resp, err := service.Authenticate(ctx, req)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Token == "" {
        t.Error("expected non-empty token")
    }
}
```

### Table-Driven Tests

```go
func TestAuthService_ValidateToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {
            name:    "valid token",
            token:   "valid.jwt.token",
            wantErr: false,
        },
        {
            name:    "expired token",
            token:   "expired.jwt.token",
            wantErr: true,
        },
        {
            name:    "malformed token",
            token:   "invalid-token",
            wantErr: true,
        },
        {
            name:    "empty token",
            token:   "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := NewAuthService()
            err := service.ValidateToken(context.Background(), tt.token)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing JWT Token Generation

```go
func TestAuthService_GenerateToken(t *testing.T) {
    service := NewAuthService()
    ctx := context.Background()
    
    userID := "user123"
    token, err := service.GenerateToken(ctx, userID)
    
    if err != nil {
        t.Fatalf("GenerateToken failed: %v", err)
    }
    
    if token == "" {
        t.Error("expected non-empty token")
    }
    
    // Verify token can be validated
    claims, err := service.ParseToken(ctx, token)
    if err != nil {
        t.Fatalf("ParseToken failed: %v", err)
    }
    
    if claims.UserID != userID {
        t.Errorf("got userID %v, want %v", claims.UserID, userID)
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
func TestProperty_TokenRoundTrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random user ID
        userID := rapid.String().Draw(t, "userID")
        
        service := NewAuthService()
        ctx := context.Background()
        
        // Generate token
        token, err := service.GenerateToken(ctx, userID)
        if err != nil {
            t.Fatalf("GenerateToken failed: %v", err)
        }
        
        // Parse token
        claims, err := service.ParseToken(ctx, token)
        if err != nil {
            t.Fatalf("ParseToken failed: %v", err)
        }
        
        // Verify round-trip
        if claims.UserID != userID {
            t.Fatalf("round-trip failed: got %v, want %v", claims.UserID, userID)
        }
    })
}
```

### Custom Generators

```go
// Generate valid usernames
func genUsername() *rapid.Generator[string] {
    return rapid.StringMatching("[a-zA-Z0-9_]{3,20}")
}

// Generate valid passwords
func genPassword() *rapid.Generator[string] {
    return rapid.StringMatching("[a-zA-Z0-9!@#$%^&*]{8,32}")
}

func TestProperty_AuthenticationWithValidCredentials(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        username := genUsername().Draw(t, "username")
        password := genPassword().Draw(t, "password")
        
        service := NewAuthService()
        ctx := context.Background()
        
        // Register user
        err := service.Register(ctx, username, password)
        if err != nil {
            t.Fatalf("Register failed: %v", err)
        }
        
        // Authenticate
        token, err := service.Authenticate(ctx, username, password)
        if err != nil {
            t.Fatalf("Authenticate failed: %v", err)
        }
        
        if token == "" {
            t.Fatal("expected non-empty token")
        }
    })
}
```

### Property Test Examples

**Idempotence:**
```go
func TestProperty_ValidateTokenIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        userID := rapid.String().Draw(t, "userID")
        
        service := NewAuthService()
        ctx := context.Background()
        
        token, _ := service.GenerateToken(ctx, userID)
        
        // Validate token twice
        err1 := service.ValidateToken(ctx, token)
        err2 := service.ValidateToken(ctx, token)
        
        // Results should be identical
        if (err1 == nil) != (err2 == nil) {
            t.Fatalf("not idempotent: %v vs %v", err1, err2)
        }
    })
}
```

**Invariants:**
```go
func TestProperty_TokenAlwaysHasExpiration(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        userID := rapid.String().Draw(t, "userID")
        
        service := NewAuthService()
        ctx := context.Background()
        
        token, err := service.GenerateToken(ctx, userID)
        if err != nil {
            return // Skip if generation fails
        }
        
        claims, err := service.ParseToken(ctx, token)
        if err != nil {
            t.Fatalf("ParseToken failed: %v", err)
        }
        
        // Token must have expiration
        if claims.ExpiresAt == 0 {
            t.Fatal("token must have expiration time")
        }
    })
}
```

**Security Properties:**
```go
func TestProperty_TokensAreUnique(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        userID := rapid.String().Draw(t, "userID")
        
        service := NewAuthService()
        ctx := context.Background()
        
        // Generate two tokens for same user
        token1, err1 := service.GenerateToken(ctx, userID)
        token2, err2 := service.GenerateToken(ctx, userID)
        
        if err1 != nil || err2 != nil {
            return // Skip if generation fails
        }
        
        // Tokens should be different (due to timestamp/nonce)
        if token1 == token2 {
            t.Fatal("tokens should be unique")
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
   - Token expiration handling
   - Invalid token formats
   - Refresh token logic
   - Error cases (invalid credentials, expired tokens)

3. **Use table-driven tests** for multiple scenarios

4. **Add property tests** for security properties

## Best Practices

### DO

✅ Write tests before or alongside code (TDD)
✅ Use descriptive test names (`TestAuthService_Authenticate_InvalidPassword_ReturnsError`)
✅ Test one thing per test function
✅ Use table-driven tests for multiple scenarios
✅ Test error cases and edge cases
✅ Use property-based tests for security properties
✅ Keep tests fast (< 1 second for unit tests)
✅ Use build tags for slow tests
✅ Test token expiration logic
✅ Clean up resources in tests (use `defer`)

### DON'T

❌ Don't test implementation details
❌ Don't use real authentication services in unit tests
❌ Don't write flaky tests (time-dependent, order-dependent)
❌ Don't skip error checking in tests
❌ Don't use `time.Sleep()` for synchronization
❌ Don't commit commented-out tests
❌ Don't test third-party JWT library code
❌ Don't write tests that depend on each other
❌ Don't hardcode secrets in tests

## Testing Security

### Password Hashing

```go
func TestAuthService_PasswordHashing(t *testing.T) {
    service := NewAuthService()
    
    password := "mySecurePassword123"
    
    // Hash password
    hash1, err := service.HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    // Hash same password again
    hash2, err := service.HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    // Hashes should be different (due to salt)
    if hash1 == hash2 {
        t.Error("password hashes should be unique")
    }
    
    // Both hashes should verify
    if !service.VerifyPassword(password, hash1) {
        t.Error("hash1 verification failed")
    }
    if !service.VerifyPassword(password, hash2) {
        t.Error("hash2 verification failed")
    }
}
```

### Token Expiration

```go
func TestAuthService_TokenExpiration(t *testing.T) {
    service := NewAuthService()
    ctx := context.Background()
    
    // Generate token with short expiration
    token, err := service.GenerateTokenWithExpiry(ctx, "user123", 1*time.Second)
    if err != nil {
        t.Fatalf("GenerateToken failed: %v", err)
    }
    
    // Token should be valid immediately
    err = service.ValidateToken(ctx, token)
    if err != nil {
        t.Errorf("token should be valid: %v", err)
    }
    
    // Wait for expiration
    time.Sleep(2 * time.Second)
    
    // Token should be expired
    err = service.ValidateToken(ctx, token)
    if err == nil {
        t.Error("token should be expired")
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
make test APP=auth

# Full test suite (nightly)
cd apps/auth-service && go test ./... -tags=property

# Coverage verification
cd apps/auth-service && ./scripts/test-coverage.sh
```

## Troubleshooting

### Tests Are Slow

- Check if property tests are running (should use build tags)
- Profile tests: `go test -cpuprofile=cpu.prof ./...`
- Look for `time.Sleep()` calls
- Check for unnecessary cryptographic operations

### Flaky Tests

- Avoid time-dependent tests (use mock time)
- Use deterministic random seeds
- Avoid parallel test conflicts
- Check for race conditions: `go test -race ./...`

### Low Coverage

- Run `go tool cover -html=coverage.out` to see uncovered code
- Add tests for error paths (invalid tokens, expired tokens)
- Add tests for edge cases (empty passwords, very long tokens)
- Use property tests for security properties

### Property Tests Failing

- Check the failing example in the output
- Simplify the generator
- Add constraints to the generator (e.g., valid username format)
- Verify the security property is correct

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [rapid Documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
