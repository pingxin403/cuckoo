# Integration Testing Strategy

**Status**: Implemented  
**Owner**: Platform Team  
**Last Updated**: 2026-01-20

## Overview

Integration testing strategy for verifying services in real environments with actual dependencies (databases, caches, other services) running in Docker containers.

## Philosophy

**Real Environment Testing**:
- Tests run against actual MySQL, Redis, and other services
- No mocking of infrastructure dependencies
- Validates complete service behavior including:
  - Database operations
  - Cache interactions
  - Service-to-service communication
  - HTTP/gRPC endpoints

**Complementary to Unit Tests**:
- Unit tests: Fast feedback, isolated components, mocked dependencies
- Integration tests: Real environment, end-to-end flows, actual dependencies
- Both are essential for comprehensive testing

## Architecture

### Test Environment

**Docker Compose Based**:
- Uses root `docker-compose.yml` for service orchestration
- Shared infrastructure (MySQL, Redis) across all services
- Each service has its own integration test suite
- Automatic cleanup after tests

**Service Dependencies**:
```yaml
# Example from docker-compose.yml
services:
  mysql:
    image: mysql:8.0
    ports: ["3306:3306"]
    healthcheck: [...]
  
  redis:
    image: redis:7.2-alpine
    ports: ["6379:6379"]
    healthcheck: [...]
  
  shortener-service:
    depends_on:
      mysql: {condition: service_healthy}
      redis: {condition: service_healthy}
    healthcheck: [...]
```

### Test Runner Scripts

Each service has `scripts/run-integration-tests.sh`:

**Responsibilities**:
1. Build service Docker image
2. Start required dependencies
3. Wait for all services to be healthy
4. Run integration tests
5. Show logs on failure
6. Clean up containers

**Example Structure**:
```bash
#!/bin/bash
set -e

# 1. Build service
docker compose build service-name

# 2. Start dependencies
docker compose up -d mysql redis

# 3. Wait for health
wait_for_healthy mysql
wait_for_healthy redis

# 4. Start service
docker compose up -d service-name
wait_for_healthy service-name

# 5. Run tests
GRPC_ADDR="localhost:9092" go test -v ./integration_test/...

# 6. Cleanup
docker compose stop service-name
```

## Service-Specific Integration Tests

### Shortener Service (Go)

**Location**: `apps/shortener-service/integration_test/`

**Test Coverage**:
1. **End-to-End Flow**: Create → Get → Redirect → Delete
2. **Custom Short Code**: User-provided short codes
3. **Link Expiration**: Time-based expiration handling
4. **Cache Warming**: Performance validation
5. **URL Validation**: Security and input validation
6. **Concurrent Operations**: Thread safety
7. **Health Checks**: Liveness and readiness probes

**Dependencies**:
- MySQL 8.0 (port 3306)
- Redis 7.2 (port 6379)

**Endpoints Tested**:
- gRPC: `localhost:9092`
- HTTP: `localhost:8081`
- Metrics: `localhost:9090`

**Key Validations**:
- Database persistence
- Cache hit/miss behavior
- HTTP redirect status codes (302, 410)
- Security headers
- Response times (<100ms for cached redirects)

**Running Tests**:
```bash
cd apps/shortener-service
./scripts/run-integration-tests.sh
```

**Test Results**:
- 7/7 tests passing
- Duration: ~3.5 seconds
- 100% success rate

### Hello Service (Java/Spring Boot)

**Location**: `apps/hello-service/src/test/java/.../integration/`

**Test Coverage**:
1. **Basic Greeting**: Valid name input
2. **Empty Name**: Default "World" greeting
3. **Special Characters**: Unicode, accents, symbols
4. **Long Names**: Large input handling
5. **Concurrent Requests**: Thread safety
6. **Service Availability**: Health check
7. **Response Time**: Performance validation (<100ms)

**Dependencies**:
- None (stateless service)

**Endpoints Tested**:
- gRPC: `localhost:9090`

**Key Validations**:
- Correct message formatting
- Unicode support
- Concurrent request handling
- Response time performance

**Running Tests**:
```bash
cd apps/hello-service
./scripts/run-integration-tests.sh
```

### TODO Service (Go)

**Location**: `apps/todo-service/integration_test/`

**Test Coverage**:
1. **End-to-End Flow**: Create → List → Update → Delete
2. **Multiple TODOs**: Batch operations
3. **Error Handling**: Nonexistent TODO operations
4. **Concurrent Operations**: Thread safety and ID uniqueness
5. **Empty List**: Edge case handling
6. **Service Availability**: Health check

**Dependencies**:
- Hello Service (port 9090) - for service-to-service communication

**Endpoints Tested**:
- gRPC: `localhost:9091`

**Key Validations**:
- CRUD operations correctness
- ID uniqueness
- Concurrent operation safety
- Error responses for invalid operations
- Service-to-service communication

**Running Tests**:
```bash
cd apps/todo-service
./scripts/run-integration-tests.sh
```

### Web App (React/TypeScript)

**Status**: Future enhancement

**Planned Coverage**:
1. **E2E User Flows**: Browser-based testing
2. **API Integration**: gRPC-Web communication
3. **Error Handling**: Network failures, timeouts
4. **State Management**: React Query cache behavior

**Tools**:
- Playwright or Cypress for E2E testing
- Mock Service Worker for API mocking (optional)

## Test Patterns

### Go Integration Tests

**Setup Pattern**:
```go
package integration_test

import (
    "context"
    "os"
    "testing"
    "time"
    
    pb "github.com/pingxin403/cuckoo/apps/service/gen/pb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

var grpcAddr = getEnv("GRPC_ADDR", "localhost:9092")

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func setupClient(t *testing.T) (pb.ServiceClient, *grpc.ClientConn) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    conn, err := grpc.DialContext(ctx, grpcAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    
    return pb.NewServiceClient(conn), conn
}

func TestEndToEndFlow(t *testing.T) {
    client, conn := setupClient(t)
    defer conn.Close()
    
    ctx := context.Background()
    
    // Test implementation
}
```

**HTTP Testing Pattern**:
```go
func TestHTTPEndpoint(t *testing.T) {
    baseURL := getEnv("BASE_URL", "http://localhost:8080")
    
    resp, err := http.Get(baseURL + "/health")
    if err != nil {
        t.Fatalf("Health check failed: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}
```

### Java Integration Tests

**Setup Pattern**:
```java
@SpringBootTest
@ActiveProfiles("test")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class ServiceIntegrationTest {
    
    private static ManagedChannel channel;
    private static ServiceGrpc.ServiceBlockingStub blockingStub;
    
    private static final String GRPC_HOST = 
        System.getenv().getOrDefault("GRPC_HOST", "localhost");
    private static final int GRPC_PORT = 
        Integer.parseInt(System.getenv().getOrDefault("GRPC_PORT", "9090"));
    
    @BeforeAll
    static void setUp() {
        channel = ManagedChannelBuilder
            .forAddress(GRPC_HOST, GRPC_PORT)
            .usePlaintext()
            .build();
        
        blockingStub = ServiceGrpc.newBlockingStub(channel);
    }
    
    @AfterAll
    static void tearDown() throws InterruptedException {
        if (channel != null) {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        }
    }
    
    @Test
    @Order(1)
    @DisplayName("Test basic functionality")
    void testBasicFunctionality() {
        // Test implementation
    }
}
```

## Best Practices

### Test Organization

**Directory Structure**:
```
apps/service-name/
├── integration_test/          # Go services
│   └── integration_test.go
├── src/test/java/.../integration/  # Java services
│   └── ServiceIntegrationTest.java
└── scripts/
    └── run-integration-tests.sh
```

**Naming Conventions**:
- Go: `integration_test` package, `*_test.go` files
- Java: `integration` package, `*IntegrationTest.java` files
- Test methods: Descriptive names explaining what is tested

### Test Independence

**Each Test Should**:
- Be runnable independently
- Clean up its own data
- Not depend on test execution order (where possible)
- Have clear setup and teardown

**Avoid**:
- Shared mutable state between tests
- Hardcoded IDs or values
- Assumptions about existing data

### Performance Considerations

**Fast Tests**:
- Use health checks to wait for services
- Set reasonable timeouts (5-10 seconds)
- Run tests in parallel where possible
- Clean up only what's necessary

**Typical Execution Times**:
- Shortener service: ~3.5 seconds
- Hello service: ~5 seconds (includes Java startup)
- TODO service: ~4 seconds

### Error Handling

**Test Failures Should**:
- Show clear error messages
- Display service logs automatically
- Indicate which service failed
- Suggest potential fixes

**Example**:
```bash
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo "[ERROR] Integration tests failed!"
    echo "[INFO] Showing service logs:"
    docker compose logs service-name
    exit $TEST_EXIT_CODE
fi
```

## CI/CD Integration

### GitHub Actions Workflow

**Integration Test Job**:
```yaml
integration-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
    
    - name: Run integration tests
      run: |
        cd apps/${{ matrix.service }}
        ./scripts/run-integration-tests.sh
  
  strategy:
    matrix:
      service: [shortener-service, hello-service, todo-service]
```

**When to Run**:
- On pull requests (changed services only)
- On main branch (all services)
- Nightly builds (full suite)

### Local Development

**Before Committing**:
```bash
# Run integration tests for changed service
cd apps/your-service
./scripts/run-integration-tests.sh
```

**Before Creating PR**:
```bash
# Run all integration tests
for service in shortener-service hello-service todo-service; do
    cd apps/$service
    ./scripts/run-integration-tests.sh
    cd ../..
done
```

## Troubleshooting

### Common Issues

**1. Service Won't Start**:
```bash
# Check service logs
docker compose logs service-name

# Check health status
docker compose ps service-name

# Verify dependencies
docker compose ps mysql redis
```

**2. Tests Timeout**:
- Increase timeout in test code
- Check service health checks
- Verify network connectivity
- Check resource constraints

**3. Port Conflicts**:
```bash
# Check what's using the port
lsof -i :9092

# Stop conflicting services
docker compose down
```

**4. Database Connection Fails**:
- Verify MySQL is healthy
- Check connection string
- Verify credentials
- Check network configuration

### Debug Mode

**Enable Verbose Logging**:
```bash
# Go services
LOG_LEVEL=debug ./scripts/run-integration-tests.sh

# Java services
SPRING_PROFILES_ACTIVE=debug ./scripts/run-integration-tests.sh
```

**Keep Containers Running**:
```bash
# Comment out cleanup in script
# trap cleanup EXIT

# Or run manually
docker compose up -d service-name
go test -v ./integration_test/...
# Containers stay running for inspection
```

## Metrics and Reporting

### Test Metrics

**Tracked Metrics**:
- Test execution time
- Success rate
- Coverage (integration + unit)
- Flakiness rate

**Example Output**:
```
========================================
Shortener Service - Integration Tests
========================================

[INFO] Building shortener-service...
[INFO] Starting MySQL and Redis...
[INFO] MySQL is ready
[INFO] Redis is ready
[INFO] Starting shortener-service...
[INFO] Shortener service is ready

Running integration tests...

=== RUN   TestEndToEndFlow
--- PASS: TestEndToEndFlow (0.05s)
=== RUN   TestCustomShortCode
--- PASS: TestCustomShortCode (0.01s)
...

PASS
ok      github.com/pingxin403/cuckoo/apps/shortener-service/integration_test 3.497s

[INFO] ✓ All integration tests passed!
```

### Coverage Reports

**Combined Coverage**:
- Unit test coverage: Fast feedback
- Integration test coverage: Real behavior
- Total coverage: Comprehensive view

**Reporting**:
```bash
# Generate combined coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Future Enhancements

### Planned Improvements

1. **Web App E2E Tests**:
   - Playwright integration
   - Visual regression testing
   - Accessibility testing

2. **Performance Testing**:
   - Load testing with k6
   - Stress testing
   - Latency benchmarks

3. **Chaos Engineering**:
   - Service failure simulation
   - Network partition testing
   - Database failover testing

4. **Test Data Management**:
   - Fixture management
   - Test data builders
   - Database seeding

5. **Parallel Execution**:
   - Run tests in parallel
   - Isolated test databases
   - Faster feedback

## References

- [Testing Guide](../../docs/TESTING_GUIDE.md)
- [Shortener Service Integration Tests](../../apps/shortener-service/INTEGRATION_TEST_SUMMARY.md)
- [Docker Compose Configuration](../../docker-compose.yml)
- [Quality Practices](./quality-practices.md)

