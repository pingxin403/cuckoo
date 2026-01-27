# Testing Guide

This guide explains how to write, run, and verify tests in the monorepo, including coverage requirements and best practices.

## Overview

The monorepo enforces test coverage requirements to ensure code quality:
- **Overall coverage**: 80% minimum
- **Service/Logic classes**: 85-90% minimum

For detailed coverage standards and exclusion rules, see [Unit Test Coverage Standard](./UNIT_TEST_COVERAGE_STANDARD.md).

## Running Tests

### All Services

```bash
# Run all tests
make test

# Run tests with coverage reports
make test-coverage

# Verify coverage thresholds (fails if below requirements)
make verify-coverage
```

### Hello Service (Java)

```bash
# Run tests
make test APP=hello-service

# Run tests with coverage report
make test-coverage-hello

# Verify coverage thresholds
make verify-coverage-hello

# Or directly with Gradle
cd apps/hello-service
./gradlew test                           # Run tests
./gradlew test jacocoTestReport          # Generate coverage report
./gradlew test jacocoTestCoverageVerification  # Verify thresholds
```

Coverage reports are generated at:
- HTML: `apps/hello-service/build/reports/jacoco/test/html/index.html`
- XML: `apps/hello-service/build/reports/jacoco/test/jacocoTestReport.xml`

### TODO Service (Go)

```bash
# Run tests
make test APP=todo-service

# Run tests with coverage verification
make test-coverage-todo

# Or directly
cd apps/todo-service
go test ./...                            # Run tests
./scripts/test-coverage.sh               # Run with coverage verification
```

Coverage reports are generated at:
- HTML: `apps/todo-service/coverage.html`
- Text: `apps/todo-service/coverage.out`

### Frontend (React)

```bash
# Run tests
make test APP=web

# Or directly
cd apps/web
npm test                                 # Run in watch mode
npm test -- --run                        # Run once (for CI)
npm test -- --coverage                   # With coverage
```

## Writing Tests

### Java Service Tests (JUnit 5)

Example test structure for a service class:

```java
@ExtendWith(MockitoExtension.class)
class HelloServiceImplTest {
    
    @InjectMocks
    private HelloServiceImpl helloService;
    
    @Test
    @DisplayName("Should return greeting with name when name is provided")
    void testSayHello_WithName() {
        // Arrange
        HelloRequest request = HelloRequest.newBuilder()
            .setName("Alice")
            .build();
        StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);
        
        // Act
        helloService.sayHello(request, responseObserver);
        
        // Assert
        ArgumentCaptor<HelloResponse> captor = ArgumentCaptor.forClass(HelloResponse.class);
        verify(responseObserver).onNext(captor.capture());
        verify(responseObserver).onCompleted();
        
        HelloResponse response = captor.getValue();
        assertThat(response.getMessage()).contains("Alice");
    }
}
```

Key points:
- Use `@DisplayName` for readable test names
- Follow Arrange-Act-Assert pattern
- Test both success and error cases
- Mock external dependencies
- Verify gRPC response observer calls

### Go Service Tests

Example test structure for a service:

```go
func TestTodoService_CreateTodo(t *testing.T) {
    // Arrange
    store := storage.NewMemoryStore()
    service := NewTodoService(store, nil)
    
    req := &todopb.CreateTodoRequest{
        Title:       "Test Todo",
        Description: "Test Description",
    }
    
    // Act
    resp, err := service.CreateTodo(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.Todo.Id)
    assert.Equal(t, "Test Todo", resp.Todo.Title)
}
```

Key points:
- Use table-driven tests for multiple scenarios
- Test concurrent access with goroutines
- Use `testify/assert` for assertions
- Test error cases with invalid inputs
- Verify storage state after operations

### Frontend Tests (Vitest + React Testing Library)

Example component test:

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { HelloForm } from './HelloForm';

describe('HelloForm', () => {
  it('should display greeting when form is submitted', async () => {
    // Arrange
    const mockSayHello = vi.fn().mockResolvedValue({
      message: 'Hello, Alice!'
    });
    
    render(<HelloForm sayHello={mockSayHello} />);
    
    // Act
    const input = screen.getByPlaceholderText('Enter your name');
    fireEvent.change(input, { target: { value: 'Alice' } });
    fireEvent.click(screen.getByText('Say Hello'));
    
    // Assert
    await waitFor(() => {
      expect(screen.getByText('Hello, Alice!')).toBeInTheDocument();
    });
    expect(mockSayHello).toHaveBeenCalledWith({ name: 'Alice' });
  });
});
```

## Coverage Configuration

For detailed coverage calculation rules and exclusion patterns, see [Unit Test Coverage Standard](./UNIT_TEST_COVERAGE_STANDARD.md).

### Java (JaCoCo)

Coverage is configured in `build.gradle`:

```gradle
jacoco {
    toolVersion = "0.8.11"
}

jacocoTestCoverageVerification {
    violationRules {
        rule {
            limit {
                minimum = 0.80  // 80% overall
            }
        }
        rule {
            element = 'CLASS'
            includes = ['com.pingxin403.cuckoo.*.service.*']
            limit {
                minimum = 0.90  // 90% for service classes
            }
        }
    }
}
```

### Go

Coverage is verified by `scripts/test-coverage.sh`. See [Unit Test Coverage Standard](./UNIT_TEST_COVERAGE_STANDARD.md) for the complete script template and exclusion rules.

Key points:
- Excludes generated code (`/gen/`), `main.go`, `/config/`, and `/storage/`
- Verifies 80% overall coverage (65% for services with external dependencies)
- Verifies 85% service package coverage (55% for services with external dependencies)

## CI/CD Integration

Tests and coverage verification run automatically in CI:

1. **On Pull Request**: All tests run, coverage verified
2. **On Push to main/develop**: Tests + coverage + Docker build
3. **Coverage Reports**: Uploaded as artifacts, viewable in GitHub Actions

The CI workflow fails if:
- Any test fails
- Coverage is below thresholds
- Generated Protobuf code is out of date

## Pre-commit Hooks

Install Git hooks to verify tests before commit:

```bash
./scripts/install-hooks.sh
```

The pre-commit hook:
- Verifies Protobuf code is up to date
- Runs tests for changed services
- Checks code formatting

## Best Practices

### Test Organization

- **Unit tests**: Test individual functions/methods in isolation
- **Integration tests**: Test service interactions (e.g., TODO calling Hello)
- **End-to-end tests**: Test full user workflows (see `scripts/test-services.sh`)

### Test Naming

- Java: Use `@DisplayName` with descriptive sentences
- Go: Use `TestServiceName_MethodName_Scenario` format
- TypeScript: Use `describe` and `it` with clear descriptions

### What to Test

For a complete guide on what should be covered by unit tests vs integration tests, see [Unit Test Coverage Standard](./UNIT_TEST_COVERAGE_STANDARD.md#单元测试-vs-集成测试).

**DO test:**
- Business logic and validation
- Error handling and edge cases
- Service method implementations
- State changes and side effects

**DON'T test:**
- Generated Protobuf code
- Third-party library internals
- Simple getters/setters
- Configuration classes

### Coverage Tips

- Focus on meaningful tests, not just coverage numbers
- Test behavior, not implementation details
- Use coverage reports to find untested code paths
- Exclude generated code from coverage (already configured)

## Troubleshooting

### Java Tests Fail to Compile

```bash
# Regenerate Protobuf code
make gen-proto-java

# Clean and rebuild
cd apps/hello-service
./gradlew clean build
```

### Go Coverage Script Fails

```bash
# Ensure bc is installed (for floating point comparison)
# macOS
brew install bc

# Ubuntu/Debian
sudo apt-get install bc
```

### Frontend Tests Timeout

```bash
# Increase timeout in vitest.config.ts
export default defineConfig({
  test: {
    testTimeout: 10000  // 10 seconds
  }
})
```

## Adding Tests to New Services

When creating a new service from templates:

1. Copy test structure from template
2. Update test class/package names
3. Add service-specific test cases
4. Verify coverage meets thresholds
5. Update CI workflow if needed

See `templates/java-service/` and `templates/go-service/` for examples.
