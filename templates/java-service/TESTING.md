# Testing Guide - {{SERVICE_NAME}}

This document provides comprehensive testing guidance for the {{SERVICE_NAME}}.

## Test Structure

```
src/test/java/
└── com/pingxin403/cuckoo/{{service_name}}/
    └── service/
        ├── {{ServiceName}}ImplTest.java          # Unit tests
        └── {{ServiceName}}PropertyTest.java      # Property-based tests (jqwik)
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
- **Overall coverage**: 80% minimum
- **Service classes**: 90% minimum

These thresholds are verified in CI and will fail the build if not met.

## Unit Testing

### Test Framework

- **JUnit 5** (Jupiter) for test structure
- **AssertJ** for fluent assertions
- **Mockito** for mocking dependencies

### Example Unit Test

```java
package com.pingxin403.cuckoo.{{service_name}}.service;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

class {{ServiceName}}ImplTest {
    
    private {{ServiceName}}Impl service;
    
    @BeforeEach
    void setUp() {
        service = new {{ServiceName}}Impl();
    }
    
    @Test
    void methodName_withValidInput_returnsExpectedResult() {
        // Arrange
        Request request = Request.newBuilder()
            .setField("value")
            .build();
        
        // Act
        Response response = service.methodName(request);
        
        // Assert
        assertThat(response).isNotNull();
        assertThat(response.getField()).isEqualTo("expected");
    }
    
    @Test
    void methodName_withInvalidInput_throwsException() {
        // Arrange
        Request request = Request.newBuilder()
            .setField("")
            .build();
        
        // Act & Assert
        assertThatThrownBy(() -> service.methodName(request))
            .isInstanceOf(IllegalArgumentException.class)
            .hasMessageContaining("field cannot be empty");
    }
}
```

### Parameterized Tests

Use `@ParameterizedTest` for testing multiple scenarios:

```java
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.junit.jupiter.params.provider.ValueSource;

@ParameterizedTest
@CsvSource({
    "'input1', 'output1'",
    "'input2', 'output2'",
    "'input3', 'output3'"
})
void methodName_variousInputs_returnsExpectedOutput(String input, String expected) {
    Request request = Request.newBuilder()
        .setField(input)
        .build();
    
    Response response = service.methodName(request);
    
    assertThat(response.getField()).isEqualTo(expected);
}

@ParameterizedTest
@ValueSource(strings = {"", " ", "  "})
void methodName_withBlankInput_throwsException(String input) {
    Request request = Request.newBuilder()
        .setField(input)
        .build();
    
    assertThatThrownBy(() -> service.methodName(request))
        .isInstanceOf(IllegalArgumentException.class);
}
```

## Property-Based Testing

### Framework

The service uses **jqwik** for property-based testing, which generates random test data to verify properties hold across many inputs.

### Example Property Test

```java
package com.pingxin403.cuckoo.{{service_name}}.service;

import net.jqwik.api.*;

class {{ServiceName}}PropertyTest {
    
    @Property
    void methodName_neverReturnsNull(@ForAll String input) {
        // Arrange
        {{ServiceName}}Impl service = new {{ServiceName}}Impl();
        Request request = Request.newBuilder()
            .setField(input)
            .build();
        
        // Act
        Response response = service.methodName(request);
        
        // Assert - Property: response is never null
        assertThat(response).isNotNull();
    }
    
    @Property
    void methodName_outputLengthNeverExceedsInput(
        @ForAll @StringLength(min = 1, max = 100) String input
    ) {
        // Arrange
        {{ServiceName}}Impl service = new {{ServiceName}}Impl();
        Request request = Request.newBuilder()
            .setField(input)
            .build();
        
        // Act
        Response response = service.methodName(request);
        
        // Assert - Property: output length <= input length
        assertThat(response.getField().length()).isLessThanOrEqualTo(input.length());
    }
    
    @Property
    void methodName_isIdempotent(@ForAll String input) {
        // Arrange
        {{ServiceName}}Impl service = new {{ServiceName}}Impl();
        Request request = Request.newBuilder()
            .setField(input)
            .build();
        
        // Act
        Response first = service.methodName(request);
        Response second = service.methodName(request);
        
        // Assert - Property: calling twice gives same result
        assertThat(first).isEqualTo(second);
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

4. **Commutativity**: Order doesn't matter
   ```java
   @Property
   void operationIsCommutative(@ForAll int a, @ForAll int b) {
       assertThat(service.combine(a, b)).isEqualTo(service.combine(b, a));
   }
   ```

## Integration Testing

For integration tests that require external services (databases, message queues, etc.):

### Test Containers

Use Testcontainers for integration tests:

```java
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.MySQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

@Testcontainers
class {{ServiceName}}IntegrationTest {
    
    @Container
    static MySQLContainer<?> mysql = new MySQLContainer<>("mysql:8.0")
        .withDatabaseName("testdb")
        .withUsername("test")
        .withPassword("test");
    
    @Container
    static GenericContainer<?> redis = new GenericContainer<>("redis:7-alpine")
        .withExposedPorts(6379);
    
    @Test
    void integrationTest() {
        // Test with real MySQL and Redis containers
        String jdbcUrl = mysql.getJdbcUrl();
        String redisHost = redis.getHost();
        Integer redisPort = redis.getFirstMappedPort();
        
        // Configure service with test containers
        // Run integration test
    }
}
```

## Mocking

### Mockito Basics

```java
import org.mockito.Mock;
import org.mockito.InjectMocks;
import org.mockito.junit.jupiter.MockitoExtension;
import org.junit.jupiter.api.extension.ExtendWith;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class ServiceWithDependenciesTest {
    
    @Mock
    private DependencyService dependencyService;
    
    @InjectMocks
    private {{ServiceName}}Impl service;
    
    @Test
    void testWithMock() {
        // Arrange
        when(dependencyService.getData()).thenReturn("mocked data");
        
        Request request = Request.newBuilder().build();
        
        // Act
        Response response = service.methodName(request);
        
        // Assert
        assertThat(response).isNotNull();
        
        // Verify interactions
        verify(dependencyService).getData();
        verify(dependencyService, times(1)).getData();
        verify(dependencyService, never()).deleteData();
    }
    
    @Test
    void testWithArgumentCaptor() {
        // Arrange
        ArgumentCaptor<String> captor = ArgumentCaptor.forClass(String.class);
        
        // Act
        service.methodName(request);
        
        // Assert
        verify(dependencyService).processData(captor.capture());
        assertThat(captor.getValue()).isEqualTo("expected value");
    }
}
```

## Best Practices

### Test Organization

1. **Arrange-Act-Assert (AAA)**: Structure tests clearly
2. **One assertion per test**: Keep tests focused (or related assertions)
3. **Descriptive names**: Use `methodName_scenario_expectedResult` format
4. **Test edge cases**: Empty strings, nulls, boundaries, max values
5. **Use test fixtures**: Extract common setup to `@BeforeEach`
6. **Clean up resources**: Use `@AfterEach` for cleanup

### Coverage Guidelines

- **Focus on business logic**: Don't test getters/setters
- **Test error paths**: Verify exception handling
- **Test boundaries**: Min/max values, empty collections
- **Property tests for algorithms**: Use jqwik for complex logic
- **Integration tests for flows**: Verify end-to-end scenarios

### Performance

- **Keep tests fast**: Unit tests should run in milliseconds
- **Isolate slow tests**: Use `@Tag("slow")` for integration tests
- **Parallel execution**: Enable in `gradle.properties`:
  ```properties
  org.gradle.parallel=true
  ```
- **Use test containers wisely**: Only for integration tests

## Continuous Integration

Tests run automatically in CI on:
- Pull requests
- Commits to main branch
- Scheduled nightly builds

CI enforces:
- All tests must pass
- Coverage thresholds must be met
- No test failures allowed
- Code quality checks pass (Checkstyle, SpotBugs)

## Troubleshooting

### Tests Fail Locally But Pass in CI

- Check Java version: `java -version`
- Clean build: `./gradlew clean test`
- Check for test order dependencies
- Verify environment variables

### Coverage Below Threshold

```bash
# Generate detailed coverage report
./gradlew test jacocoTestReport

# Open HTML report to see uncovered lines
open build/reports/jacoco/test/html/index.html

# Check coverage for specific package
./gradlew test jacocoTestReport --info | grep "{{service_name}}"
```

### Property Tests Fail Intermittently

- Set a fixed seed in `jqwik.properties`:
  ```properties
  jqwik.seed = 42
  ```
- Reduce number of tries for debugging:
  ```properties
  jqwik.tries.default = 10
  ```
- Check for non-deterministic behavior (time, random, threads)

### Slow Test Execution

```bash
# Run tests in parallel
./gradlew test --parallel

# Profile test execution
./gradlew test --profile

# Open profile report
open build/reports/profile/profile-*.html
```

### Mockito Issues

```bash
# Common issues:
# 1. Verify called on non-mock: Ensure @Mock annotation is present
# 2. Stubbing not working: Check method signature matches exactly
# 3. Argument matchers: Use all matchers or all concrete values, not mixed
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
