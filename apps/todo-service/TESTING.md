# Testing Guide for todo-service

This document provides comprehensive testing guidelines for the todo-service.

## Test Organization

### Test Types

1. **Unit Tests** (`*_test.go`)
   - Test individual functions and methods
   - Fast execution (< 1 second)
   - Use in-memory storage for testing
   - Run by default with `go test ./...`

2. **Property-Based Tests** (`*_property_test.go`)
   - Test universal properties across many inputs
   - Use `pgregory.net/rapid` framework
   - Separated with build tags for performance
   - Run with `go test ./... -tags=property`

3. **Integration Tests** (`integration_test/`)
   - Test complete workflows including Hello Service communication
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
go test ./storage/...
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
# - Overall coverage >= 70%
# - Service/storage packages >= 80%
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

func TestTodoService_CreateTodo(t *testing.T) {
    // Arrange
    store := storage.NewMemoryStore()
    service := NewTodoService(store, nil)
    ctx := context.Background()
    
    req := &todopb.CreateTodoRequest{
        Title:       "Test TODO",
        Description: "Test description",
    }
    
    // Act
    resp, err := service.CreateTodo(ctx, req)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Todo.Title != "Test TODO" {
        t.Errorf("got title %v, want Test TODO", resp.Todo.Title)
    }
}
```

### Table-Driven Tests

```go
func TestTodoService_ValidateTitle(t *testing.T) {
    tests := []struct {
        name    string
        title   string
        wantErr bool
    }{
        {
            name:    "valid title",
            title:   "Buy groceries",
            wantErr: false,
        },
        {
            name:    "empty title",
            title:   "",
            wantErr: true,
        },
        {
            name:    "very long title",
            title:   string(make([]byte, 300)),
            wantErr: true,
        },
        {
            name:    "title with special characters",
            title:   "TODO: Fix bug #123",
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateTitle(tt.title)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateTitle() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing with In-Memory Store

```go
func TestTodoService_UpdateTodo(t *testing.T) {
    // Setup
    store := storage.NewMemoryStore()
    service := NewTodoService(store, nil)
    ctx := context.Background()
    
    // Create a TODO first
    createResp, err := service.CreateTodo(ctx, &todopb.CreateTodoRequest{
        Title:       "Original Title",
        Description: "Original Description",
    })
    if err != nil {
        t.Fatalf("CreateTodo failed: %v", err)
    }
    
    // Update the TODO
    updateResp, err := service.UpdateTodo(ctx, &todopb.UpdateTodoRequest{
        Id:          createResp.Todo.Id,
        Title:       "Updated Title",
        Description: "Updated Description",
        Completed:   true,
    })
    
    if err != nil {
        t.Fatalf("UpdateTodo failed: %v", err)
    }
    
    // Verify update
    if updateResp.Todo.Title != "Updated Title" {
        t.Errorf("got title %v, want Updated Title", updateResp.Todo.Title)
    }
    if !updateResp.Todo.Completed {
        t.Error("expected completed to be true")
    }
}
```

### Testing Inter-Service Communication

```go
func TestTodoService_WithHelloService(t *testing.T) {
    // Mock Hello Service client
    mockHelloClient := &mockHelloServiceClient{
        response: &hellopb.HelloResponse{
            Message: "Hello, Test!",
        },
    }
    
    store := storage.NewMemoryStore()
    service := NewTodoService(store, mockHelloClient)
    ctx := context.Background()
    
    // Test service with mock Hello Service
    resp, err := service.CreateTodoWithGreeting(ctx, &todopb.CreateTodoRequest{
        Title: "Test TODO",
    })
    
    if err != nil {
        t.Fatalf("CreateTodoWithGreeting failed: %v", err)
    }
    
    if resp.Greeting != "Hello, Test!" {
        t.Errorf("got greeting %v, want Hello, Test!", resp.Greeting)
    }
}

// Mock Hello Service client
type mockHelloServiceClient struct {
    response *hellopb.HelloResponse
    err      error
}

func (m *mockHelloServiceClient) SayHello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
    return m.response, m.err
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
func TestProperty_TodoIDUniqueness(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        store := storage.NewMemoryStore()
        service := NewTodoService(store, nil)
        ctx := context.Background()
        
        // Create multiple TODOs
        ids := make(map[string]bool)
        count := rapid.IntRange(1, 100).Draw(t, "count")
        
        for i := 0; i < count; i++ {
            title := rapid.String().Draw(t, "title")
            
            resp, err := service.CreateTodo(ctx, &todopb.CreateTodoRequest{
                Title: title,
            })
            
            if err != nil {
                continue // Skip invalid inputs
            }
            
            // Check ID uniqueness
            if ids[resp.Todo.Id] {
                t.Fatalf("duplicate ID found: %s", resp.Todo.Id)
            }
            ids[resp.Todo.Id] = true
        }
    })
}
```

### Custom Generators

```go
// Generate valid TODO titles
func genTodoTitle() *rapid.Generator[string] {
    return rapid.StringMatching("[a-zA-Z0-9 ]{1,100}")
}

// Generate valid TODO descriptions
func genTodoDescription() *rapid.Generator[string] {
    return rapid.StringMatching("[a-zA-Z0-9 .,!?]{0,500}")
}

func TestProperty_CreateTodoWithValidInputs(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        title := genTodoTitle().Draw(t, "title")
        description := genTodoDescription().Draw(t, "description")
        
        store := storage.NewMemoryStore()
        service := NewTodoService(store, nil)
        ctx := context.Background()
        
        resp, err := service.CreateTodo(ctx, &todopb.CreateTodoRequest{
            Title:       title,
            Description: description,
        })
        
        if err != nil {
            t.Fatalf("CreateTodo failed with valid inputs: %v", err)
        }
        
        if resp.Todo.Title != title {
            t.Fatalf("title mismatch: got %v, want %v", resp.Todo.Title, title)
        }
    })
}
```

### Property Test Examples

**Idempotence:**
```go
func TestProperty_GetTodoIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        store := storage.NewMemoryStore()
        service := NewTodoService(store, nil)
        ctx := context.Background()
        
        // Create a TODO
        createResp, err := service.CreateTodo(ctx, &todopb.CreateTodoRequest{
            Title: "Test TODO",
        })
        if err != nil {
            return
        }
        
        todoID := createResp.Todo.Id
        
        // Get TODO twice
        resp1, err1 := service.GetTodo(ctx, &todopb.GetTodoRequest{Id: todoID})
        resp2, err2 := service.GetTodo(ctx, &todopb.GetTodoRequest{Id: todoID})
        
        // Results should be identical
        if (err1 == nil) != (err2 == nil) {
            t.Fatalf("inconsistent errors: %v vs %v", err1, err2)
        }
        
        if err1 == nil && resp1.Todo.Id != resp2.Todo.Id {
            t.Fatalf("not idempotent: %v != %v", resp1.Todo.Id, resp2.Todo.Id)
        }
    })
}
```

**Invariants:**
```go
func TestProperty_CompletedTodosStayCompleted(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        store := storage.NewMemoryStore()
        service := NewTodoService(store, nil)
        ctx := context.Background()
        
        // Create and complete a TODO
        createResp, _ := service.CreateTodo(ctx, &todopb.CreateTodoRequest{
            Title: "Test TODO",
        })
        
        _, err := service.UpdateTodo(ctx, &todopb.UpdateTodoRequest{
            Id:        createResp.Todo.Id,
            Completed: true,
        })
        if err != nil {
            return
        }
        
        // Get TODO and verify it's still completed
        getResp, err := service.GetTodo(ctx, &todopb.GetTodoRequest{
            Id: createResp.Todo.Id,
        })
        
        if err == nil && !getResp.Todo.Completed {
            t.Fatal("completed TODO became uncompleted")
        }
    })
}
```

**List Consistency:**
```go
func TestProperty_ListTodosConsistency(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        store := storage.NewMemoryStore()
        service := NewTodoService(store, nil)
        ctx := context.Background()
        
        // Create multiple TODOs
        count := rapid.IntRange(0, 50).Draw(t, "count")
        for i := 0; i < count; i++ {
            title := rapid.String().Draw(t, "title")
            _, _ = service.CreateTodo(ctx, &todopb.CreateTodoRequest{
                Title: title,
            })
        }
        
        // List TODOs twice
        resp1, err1 := service.ListTodos(ctx, &todopb.ListTodosRequest{})
        resp2, err2 := service.ListTodos(ctx, &todopb.ListTodosRequest{})
        
        if err1 != nil || err2 != nil {
            return
        }
        
        // List should return same count
        if len(resp1.Todos) != len(resp2.Todos) {
            t.Fatalf("inconsistent list: %d vs %d", len(resp1.Todos), len(resp2.Todos))
        }
    })
}
```

## Coverage Requirements

### Targets

- **Overall coverage**: 70% minimum
- **Service/storage packages**: 80% minimum

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
   - Error handling (invalid inputs, not found errors)
   - Edge cases (empty lists, very long titles)
   - Boundary conditions (max TODO count)

3. **Use table-driven tests** for multiple scenarios

4. **Add property tests** for universal properties

## Best Practices

### DO

✅ Write tests before or alongside code (TDD)
✅ Use descriptive test names (`TestTodoService_CreateTodo_EmptyTitle_ReturnsError`)
✅ Test one thing per test function
✅ Use table-driven tests for multiple scenarios
✅ Test error cases and edge cases
✅ Use property-based tests for data consistency
✅ Keep tests fast (< 1 second for unit tests)
✅ Use build tags for slow tests
✅ Use in-memory storage for unit tests
✅ Clean up resources in tests (use `defer`)

### DON'T

❌ Don't test implementation details
❌ Don't use real databases in unit tests
❌ Don't write flaky tests (time-dependent, order-dependent)
❌ Don't skip error checking in tests
❌ Don't use `time.Sleep()` for synchronization
❌ Don't commit commented-out tests
❌ Don't test third-party library code
❌ Don't write tests that depend on each other

## Testing Storage Layer

### Memory Store Tests

```go
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
    store := storage.NewMemoryStore()
    const numGoroutines = 100
    var wg sync.WaitGroup
    
    wg.Add(numGoroutines)
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer wg.Done()
            
            // Concurrent create
            todo := &storage.Todo{
                ID:    fmt.Sprintf("todo-%d", id),
                Title: fmt.Sprintf("TODO %d", id),
            }
            _ = store.Create(todo)
            
            // Concurrent read
            _, _ = store.Get(todo.ID)
        }(i)
    }
    
    wg.Wait()
    
    // Verify all TODOs were created
    todos, err := store.List()
    if err != nil {
        t.Fatalf("List failed: %v", err)
    }
    
    if len(todos) != numGoroutines {
        t.Errorf("expected %d TODOs, got %d", numGoroutines, len(todos))
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
make test APP=todo

# Full test suite (nightly)
cd apps/todo-service && go test ./... -tags=property

# Coverage verification
cd apps/todo-service && ./scripts/test-coverage.sh
```

## Troubleshooting

### Tests Are Slow

- Check if property tests are running (should use build tags)
- Profile tests: `go test -cpuprofile=cpu.prof ./...`
- Look for `time.Sleep()` calls
- Check for unnecessary operations

### Flaky Tests

- Avoid time-dependent tests
- Use deterministic random seeds
- Avoid parallel test conflicts
- Check for race conditions: `go test -race ./...`

### Low Coverage

- Run `go tool cover -html=coverage.out` to see uncovered code
- Add tests for error paths (not found, invalid input)
- Add tests for edge cases (empty lists, max length)
- Use property tests for data consistency

### Property Tests Failing

- Check the failing example in the output
- Simplify the generator
- Add constraints to the generator (e.g., valid title format)
- Verify the property is correct

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [rapid Documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Property Testing Guide](../../docs/development/PROPERTY_TESTING.md)
- [Testing Guide](../../docs/development/TESTING_GUIDE.md)
