# IM Gateway Service Dependency Integration Tests

This directory contains integration tests that validate the IM Gateway Service's interactions with dependent services (Auth Service, User Service, IM Service) and infrastructure components.

## Overview

These tests verify:
- **Service Integration**: Gateway → Auth Service, Gateway → User Service, IM Service → Gateway
- **Infrastructure Integration**: Offline Worker → Database, Kafka message bus
- **Resilience**: Circuit breaker behavior, service unavailability handling, timeout handling
- **Health Checks**: Service availability monitoring

## Test Structure

### Test Files

- `service_dependency_test.go` - Main integration test suite with service dependency tests

### Test Categories

1. **Gateway → Auth Service Integration**
   - Valid token validation
   - Service unavailability handling
   - Timeout handling

2. **Gateway → User Service Integration**
   - Get user profile
   - Batch get users
   - Validate group membership
   - Service retry on failure

3. **IM Service → Gateway Integration**
   - Route private messages
   - Route group messages
   - Concurrent message routing

4. **Offline Worker → Database Integration**
   - Offline message persistence
   - High volume offline messages

5. **Circuit Breaker Tests**
   - Circuit breaker on repeated failures

6. **Service Health Checks**
   - All services healthy

## Running the Tests

### Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ installed
- Sufficient system resources (4GB+ RAM recommended)

### Quick Start

Run all service dependency integration tests:

```bash
cd apps/im-gateway-service/integration_test
./run-integration-tests.sh
```

### Manual Test Execution

If you prefer to run tests manually:

1. Start the test environment:
```bash
docker-compose -f docker-compose.test.yml up -d
```

2. Wait for services to be healthy (check with `docker-compose ps`)

3. Create Kafka topics:
```bash
docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic group_msg --partitions 3 --replication-factor 1

docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic offline_msg --partitions 3 --replication-factor 1

docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic membership_change --partitions 3 --replication-factor 1
```

4. Run the tests:
```bash
cd apps/im-gateway-service
export AUTH_SERVICE_ADDR="localhost:9095"
export USER_SERVICE_ADDR="localhost:9096"
export IM_SERVICE_ADDR="localhost:9094"
export REDIS_ADDR="localhost:6379"
export ETCD_ADDR="localhost:2379"
export KAFKA_ADDR="localhost:9092"

go test -v -tags=integration ./integration_test/... -timeout 15m
```

5. Clean up:
```bash
cd integration_test
docker-compose -f docker-compose.test.yml down -v
```

## Test Environment

The test environment includes:

### Services
- **Auth Service** (port 9095) - JWT token validation
- **User Service** (port 9096) - User profile and group membership
- **IM Service** (port 9094) - Message routing
- **IM Gateway Service** (ports 9093, 8080) - WebSocket gateway

### Infrastructure
- **MySQL** (ports 3306, 3307) - User data and offline messages
- **Redis** (port 6379) - Deduplication and caching
- **etcd** (port 2379) - Service registry
- **Kafka** (port 9092) - Message bus
- **Zookeeper** (port 2181) - Kafka coordination

## Environment Variables

The tests use the following environment variables:

- `AUTH_SERVICE_ADDR` - Auth Service address (default: localhost:9095)
- `USER_SERVICE_ADDR` - User Service address (default: localhost:9096)
- `IM_SERVICE_ADDR` - IM Service address (default: localhost:9094)
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `ETCD_ADDR` - etcd address (default: localhost:2379)
- `KAFKA_ADDR` - Kafka address (default: localhost:9092)

## Test Validation

Each test validates specific requirements from the design document:

- **Requirements 14.4**: Service dependency integration
- **Requirements 14.4**: Graceful degradation
- **Requirements 14.4**: Service availability monitoring

## Troubleshooting

### Services Not Starting

If services fail to start:
1. Check Docker logs: `docker-compose -f docker-compose.test.yml logs <service-name>`
2. Verify port availability: `lsof -i :<port>`
3. Increase Docker resource limits

### Tests Timing Out

If tests timeout:
1. Increase test timeout: `go test -timeout 20m`
2. Check service health: `docker-compose -f docker-compose.test.yml ps`
3. Verify network connectivity between containers

### Connection Refused Errors

If you see connection refused errors:
1. Wait longer for services to initialize
2. Check service health checks in docker-compose.test.yml
3. Verify environment variables are set correctly

## CI/CD Integration

These tests can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Service Dependency Integration Tests
  run: |
    cd apps/im-gateway-service/integration_test
    ./run-integration-tests.sh
```

## Performance Considerations

- Tests may take 5-10 minutes to complete
- Docker containers require ~4GB RAM
- Kafka and MySQL initialization can take 30-60 seconds

## Future Enhancements

Potential improvements:
- Add more edge case scenarios
- Add performance benchmarks
- Add chaos engineering tests (random service failures)
- Add network partition tests
- Add load testing scenarios


## Coverage Reports

The test runner automatically generates coverage reports when tests pass:

### Generated Files

- `coverage/integration.out` - Raw coverage data in Go format
- `coverage/integration.html` - Interactive HTML coverage report
- `coverage/integration-junit.xml` - JUnit XML report (requires go-junit-report)

### Viewing Coverage

Open the HTML report in your browser:
```bash
# macOS
open apps/im-gateway-service/coverage/integration.html

# Linux
xdg-open apps/im-gateway-service/coverage/integration.html

# Windows
start apps/im-gateway-service/coverage/integration.html
```

### Coverage Summary

The test runner displays a coverage summary at the end:
```
Coverage Summary:
total:  (statements)    78.5%
```

### Installing go-junit-report

For JUnit XML reports (useful for CI/CD):
```bash
go install github.com/jstemmer/go-junit-report@latest
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Gateway Service Integration Tests

on:
  push:
    paths:
      - 'apps/im-gateway-service/**'
      - 'apps/auth-service/**'
      - 'apps/user-service/**'
      - 'apps/im-service/**'

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Run integration tests
        run: |
          cd apps/im-gateway-service/integration_test
          ./run-integration-tests.sh
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./apps/im-gateway-service/coverage/integration.out
          flags: gateway-integration
```

### GitLab CI Example

```yaml
gateway-integration-test:
  stage: test
  image: golang:1.21
  services:
    - mysql:8.0
    - redis:7-alpine
    - etcd:v3.5.9
    - kafka:7.4.0
  script:
    - cd apps/im-gateway-service/integration_test
    - ./run-integration-tests.sh
  artifacts:
    paths:
      - apps/im-gateway-service/coverage/
    reports:
      coverage_report:
        coverage_format: cobertura
        path: apps/im-gateway-service/coverage/integration.out
```

## Performance Metrics

The test runner collects and displays performance metrics:

- **Test Duration**: Total time to run all tests
- **Total Tests**: Number of test cases executed
- **Success Rate**: Percentage of passing tests
- **Coverage**: Code coverage percentage

Example output:
```
Performance Metrics:
Test Duration: PASS (62.8s)
Total Tests: 12
Coverage: 78.5%
```

## Test Execution Time

Typical execution times:
- Service startup: 60-90 seconds
- Test execution: 30-60 seconds
- Total: 2-3 minutes

Factors affecting execution time:
- Docker container startup
- Service health check intervals
- Network latency between containers
- Number of concurrent test cases
