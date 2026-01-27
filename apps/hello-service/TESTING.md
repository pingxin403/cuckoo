# Testing Guide - Hello Service

This document provides comprehensive testing guidance for the Hello Service.

## Test Structure

```
src/test/java/
└── com/pingxin403/cuckoo/hello/
    └── service/
        ├── HelloServiceImplTest.java          # Unit tests
        └── HelloServicePropertyTest.java      # Property-based tests (jqwik)
```

## Running Tests

### Quick Test Commands

```bash
# Run all tests
./gradlew test

# Run tests with coverage report
./gradlew test jacocoTestReport

# Verify coverage thresholds
./gradlew test jacocoTestCoverageVerification

# Run only unit tests (exclude property tests)
./gradlew test --tests '*Test' --exclude-tests '*PropertyTest'

# Run only property-based tests
./gradlew test --tests '*PropertyTest'

# Run tests in continuous mode (watch for changes)
./gradlew test --continuous
```

### Coverage Reports

After running tests with coverage, reports are generated at:
- **HTML Report**: `build/reports/jacoco/test/html/index.html`
- **XML Report**: `build/reports/jacoco/test/jacocoTestReport.xml`

Open the HTML report in your browser:
```bash
open build/reports/jacoco/test/html/index.html
```

## Coverage Requirements

The service enforces test coverage thresholds:
- **Overall coverage**: 30% minimum
- **Service classes**: 50% minimum

These thresholds are verified in CI and will fail the build if not met.

## Unit Testing

### Test Framework

- **JUnit 5** (Jupiter) for test structure
- **AssertJ** for fluent assertions
- **Mockito** for mocking dependencies

### Example Unit Test

```java
package com.pingxin403.cuckoo.hello.service;

import org.junit.jupiter.api.Test;
import static org.assertj.core.api.Assertions.assertThat;

class HelloServiceImplTest {
    
    @Test
    void sayHello_withName_returnsPersonalizedGreeting() {
        // Arrange
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName("Alice")
            .build();
        
        // Act
        HelloResponse response = service.sayHello(request);
        
        // Assert
        assertThat(response.getMessage()).isEqualTo("Hello, Alice!");
    }
    
    @Test
    void sayHello_withEmptyName_returnsDefaultGreeting() {
        // Arrange
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName("")
            .build();
        
        // Act
        HelloResponse response = service.sayHello(request);
        
        // Assert
        assertThat(response.getMessage()).isEqualTo("Hello, World!");
    }
}
```

### Parameterized Tests

Use `@ParameterizedTest` for testing multiple scenarios:

```java
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

@ParameterizedTest
@CsvSource({
    "'Alice', 'Hello, Alice!'",
    "'Bob', 'Hello, Bob!'",
    "'', 'Hello, World!'",
    "' ', 'Hello, World!'"
})
void sayHello_variousInputs_returnsExpectedGreeting(String name, String expected) {
    HelloServiceImpl service = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder()
        .setName(name)
        .build();
    
    HelloResponse response = service.sayHello(request);
    
    assertThat(response.getMessage()).isEqualTo(expected);
}
```

## Property-Based Testing

### Framework

The service uses **jqwik** for property-based testing, which generates random test data to verify properties hold across many inputs.

### Example Property Test

```java
package com.pingxin403.cuckoo.hello.service;

import net.jqwik.api.*;

class HelloServicePropertyTest {
    
    @Property
    void sayHello_neverReturnsNull(@ForAll String name) {
        // Arrange
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName(name)
            .build();
        
        // Act
        HelloResponse response = service.sayHello(request);
        
        // Assert - Property: response is never null
        assertThat(response).isNotNull();
        assertThat(response.getMessage()).isNotNull();
    }
    
    @Property
    void sayHello_messageAlwaysStartsWithHello(@ForAll String name) {
        // Arrange
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName(name)
            .build();
        
        // Act
        HelloResponse response = service.sayHello(request);
        
        // Assert - Property: message always starts with "Hello"
        assertThat(response.getMessage()).startsWith("Hello");
    }
    
    @Property
    void sayHello_withNonEmptyName_includesNameInResponse(
        @ForAll @StringLength(min = 1, max = 100) String name
    ) {
        // Arrange
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName(name)
            .build();
        
        // Act
        HelloResponse response = service.sayHello(request);
        
        // Assert - Property: non-empty names appear in response
        assertThat(response.getMessage()).contains(name);
    }
}
```

### Property Test Configuration

Configure jqwik in `src/test/resources/jqwik.properties`:

```properties
# Number of tries per property (default: 1000)
jqwik.tries.default = 100

# Seed for reproducible tests
# jqwik.seed = 42

# Report only failures
jqwik.reporting.onlyFailures = true
```

### Common Property Patterns

1. **Invariants**: Properties that always hold
   ```java
   @Property
   void resultIsNeverNull(@ForAll Input input) {
       assertThat(service.process(input)).isNotNull();
   }
   ```

2. **Idempotence**: Applying operation twice gives same result
   ```java
   @Property
   void operationIsIdempotent(@ForAll Input input) {
       Result first = service.process(input);
       Result second = service.process(input);
       assertThat(first).isEqualTo(second);
   }
   ```

3. **Round-trip**: Encode then decode returns original
   ```java
   @Property
   void encodeDecodeRoundTrip(@ForAll Data data) {
       String encoded = service.encode(data);
       Data decoded = service.decode(encoded);
       assertThat(decoded).isEqualTo(data);
   }
   ```

## Integration Testing

For integration tests that require external services (databases, message queues, etc.):

### Test Containers

Use Testcontainers for integration tests:

```java
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

@Testcontainers
class HelloServiceIntegrationTest {
    
    @Container
    static GenericContainer<?> redis = new GenericContainer<>("redis:7-alpine")
        .withExposedPorts(6379);
    
    @Test
    void integrationTest() {
        // Test with real Redis container
    }
}
```

## Mocking

### Mockito Basics

```java
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.junit.jupiter.api.extension.ExtendWith;

@ExtendWith(MockitoExtension.class)
class ServiceWithDependenciesTest {
    
    @Mock
    private DependencyService dependencyService;
    
    @Test
    void testWithMock() {
        // Arrange
        when(dependencyService.getData()).thenReturn("mocked data");
        
        HelloServiceImpl service = new HelloServiceImpl(dependencyService);
        
        // Act & Assert
        // ...
        
        // Verify interactions
        verify(dependencyService).getData();
    }
}
```

## Best Practices

### Test Organization

1. **Arrange-Act-Assert (AAA)**: Structure tests clearly
2. **One assertion per test**: Keep tests focused
3. **Descriptive names**: Use `methodName_scenario_expectedResult` format
4. **Test edge cases**: Empty strings, nulls, boundaries
5. **Use test fixtures**: Extract common setup to `@BeforeEach`

### Coverage Guidelines

- **Focus on business logic**: Don't test getters/setters
- **Test error paths**: Verify exception handling
- **Test boundaries**: Min/max values, empty collections
- **Property tests for algorithms**: Use jqwik for complex logic

### Performance

- **Keep tests fast**: Unit tests should run in milliseconds
- **Isolate slow tests**: Use `@Tag("slow")` for integration tests
- **Parallel execution**: Enable in `gradle.properties`:
  ```properties
  org.gradle.parallel=true
  ```

## Continuous Integration

Tests run automatically in CI on:
- Pull requests
- Commits to main branch
- Scheduled nightly builds

CI enforces:
- All tests must pass
- Coverage thresholds must be met
- No test failures allowed

## Troubleshooting

### Tests Fail Locally But Pass in CI

- Check Java version: `java -version`
- Clean build: `./gradlew clean test`
- Check for test order dependencies

### Coverage Below Threshold

```bash
# Generate detailed coverage report
./gradlew test jacocoTestReport

# Open HTML report to see uncovered lines
open build/reports/jacoco/test/html/index.html
```

### Property Tests Fail Intermittently

- Set a fixed seed in `jqwik.properties`
- Reduce number of tries for debugging
- Check for non-deterministic behavior

### Slow Test Execution

```bash
# Run tests in parallel
./gradlew test --parallel

# Profile test execution
./gradlew test --profile
```

## Resources

- [JUnit 5 User Guide](https://junit.org/junit5/docs/current/user-guide/)
- [jqwik User Guide](https://jqwik.net/docs/current/user-guide.html)
- [AssertJ Documentation](https://assertj.github.io/doc/)
- [Mockito Documentation](https://javadoc.io/doc/org.mockito/mockito-core/latest/org/mockito/Mockito.html)
- [Testcontainers](https://www.testcontainers.org/)
- [Monorepo Testing Guide](../../docs/development/TESTING_GUIDE.md)

## Support

For questions about testing:
- Review existing test examples in the codebase
- Check the monorepo testing documentation
- Contact backend-java-team
