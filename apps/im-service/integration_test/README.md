# IM Service Integration Tests

This directory contains end-to-end integration tests for the IM Service that validate the complete message flow across all system components.

## Overview

The integration tests verify:
- **Complete message flow**: Send → Route → Deliver
- **Online user message delivery**: Fast path routing
- **Offline user message storage**: Slow path with Kafka and MySQL
- **Group message broadcast**: Kafka-based fan-out
- **Multi-device delivery**: Registry-based routing
- **Message deduplication**: Redis-based duplicate detection
- **Sequence number generation**: Monotonic ordering
- **Sensitive word filtering**: Content moderation

## Architecture

The integration tests use Docker Compose to spin up a complete test environment:

```
┌─────────────────────────────────────────────────────────────┐
│                    Integration Test Suite                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        IM Service                            │
│                      (gRPC Server)                           │
└─────────────────────────────────────────────────────────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
    ┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
    │ MySQL  │    │ Redis  │    │  etcd  │    │ Kafka  │
    └────────┘    └────────┘    └────────┘    └────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ installed
- Ports available: 3306 (MySQL), 6379 (Redis), 2379/2380 (etcd), 9092 (Kafka), 9094 (IM Service)

## Running Tests

### Quick Start

Run all integration tests:

```bash
./run-integration-tests.sh
```

This script will:
1. Start all infrastructure services (MySQL, Redis, etcd, Kafka)
2. Wait for services to be healthy
3. Create required Kafka topics
4. Build and start the IM Service
5. Run all integration tests
6. Clean up all containers and volumes

### Manual Testing

If you want to run tests manually:

```bash
# 1. Start infrastructure
docker-compose -f docker-compose.test.yml up -d

# 2. Wait for services to be ready (check health)
docker-compose -f docker-compose.test.yml ps

# 3. Create Kafka topics
docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic group_msg --partitions 3 --replication-factor 1

docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic offline_msg --partitions 3 --replication-factor 1

# 4. Run tests
cd ..
go test -v -tags=integration ./integration_test/... -timeout 10m

# 5. Cleanup
cd integration_test
docker-compose -f docker-compose.test.yml down -v
```

## Test Cases

### TestEndToEndPrivateMessageFlow
**Validates**: Requirements 1.1, 1.2, 3.1

Tests the complete private message flow:
1. Register sender and recipient in Registry (simulate online users)
2. Send private message via gRPC
3. Verify message is NOT stored in offline_messages (recipient is online)
4. Verify deduplication entry exists in Redis

### TestOfflineMessageStorage
**Validates**: Requirements 4.1, 4.2, 4.3

Tests offline message handling:
1. Send message to offline user (not in Registry)
2. Wait for Kafka consumer to process message
3. Verify message is stored in offline_messages table
4. Verify sequence number is assigned

### TestGroupMessageBroadcast
**Validates**: Requirements 2.1, 2.2, 2.3

Tests group message broadcasting:
1. Send group message via gRPC
2. Verify message is published to Kafka group_msg topic
3. Verify sequence number is assigned

### TestMessageDeduplication
**Validates**: Requirements 8.1, 8.2, 8.3

Tests duplicate message detection:
1. Send message first time
2. Send same message again (duplicate)
3. Verify both succeed but deduplication is applied

### TestSequenceNumberMonotonicity
**Validates**: Requirements 16.1, 16.2

Tests sequence number generation:
1. Send multiple messages in sequence
2. Verify sequence numbers are strictly increasing
3. Verify no gaps or duplicates

### TestSensitiveWordFiltering
**Validates**: Requirements 11.4, 17.4, 17.5

Tests content moderation:
1. Send message with sensitive words
2. Verify message is still delivered (filtered, not blocked)
3. Verify filtering is applied

## Environment Variables

The tests use the following environment variables (with defaults):

- `IM_SERVICE_ADDR`: IM Service gRPC address (default: `localhost:9094`)
- `MYSQL_ADDR`: MySQL connection string (default: `root:password@tcp(localhost:3306)/im_chat`)
- `REDIS_ADDR`: Redis address (default: `localhost:6379`)
- `ETCD_ADDR`: etcd address (default: `localhost:2379`)
- `KAFKA_ADDR`: Kafka broker address (default: `localhost:9092`)

## Troubleshooting

### Services not starting

Check service logs:
```bash
docker-compose -f docker-compose.test.yml logs <service-name>
```

### Tests timing out

Increase the test timeout:
```bash
go test -v -tags=integration ./integration_test/... -timeout 20m
```

### Port conflicts

If ports are already in use, modify `docker-compose.test.yml` to use different ports.

### Kafka topics not created

Manually create topics:
```bash
docker-compose -f docker-compose.test.yml exec kafka kafka-topics \
    --create --bootstrap-server localhost:9092 \
    --topic <topic-name> --partitions 3 --replication-factor 1
```

## CI/CD Integration

To run integration tests in CI/CD:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    cd apps/im-service/integration_test
    ./run-integration-tests.sh
```

## Coverage

Integration tests complement unit tests by verifying:
- Cross-service communication
- Infrastructure integration
- End-to-end workflows
- Real-world scenarios

For unit test coverage, see `../TESTING.md`.

## Contributing

When adding new integration tests:
1. Follow the existing test structure
2. Use descriptive test names
3. Add cleanup logic (defer statements)
4. Document what requirements are validated
5. Keep tests independent and idempotent

## References

- [IM Chat System Requirements](../../../.kiro/specs/im-chat-system/requirements.md)
- [IM Chat System Design](../../../.kiro/specs/im-chat-system/design.md)
- [Unit Testing Guide](../TESTING.md)



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
open apps/im-service/coverage/integration.html

# Linux
xdg-open apps/im-service/coverage/integration.html

# Windows
start apps/im-service/coverage/integration.html
```

### Coverage Summary

The test runner displays a coverage summary at the end:
```
Coverage Summary:
total:  (statements)    85.2%
```

### Installing go-junit-report

For JUnit XML reports (useful for CI/CD):
```bash
go install github.com/jstemmer/go-junit-report@latest
```

## CI/CD Integration

Example CI/CD configurations are provided:

### GitHub Actions

See `.github-workflows-example.yml` for a complete GitHub Actions workflow that:
- Runs integration tests
- Generates coverage reports
- Uploads coverage to Codecov
- Stores coverage artifacts

Copy to `.github/workflows/im-service-integration.yml` to enable.

### GitLab CI

See `.gitlab-ci-example.yml` for a complete GitLab CI configuration that:
- Runs integration tests with all services
- Generates coverage reports
- Stores coverage artifacts
- Displays coverage in merge requests

Copy to `.gitlab-ci.yml` to enable.

### CI/CD Features

Both configurations include:
- ✅ Automatic service startup and health checks
- ✅ Coverage report generation
- ✅ Coverage artifact storage
- ✅ Service log collection on failure
- ✅ Timeout protection (30 minutes)
- ✅ Conditional execution (only on relevant file changes)

## Performance Metrics

The test runner collects and displays performance metrics:

- **Test Duration**: Total time to run all tests
- **Total Tests**: Number of test cases executed
- **Success Rate**: Percentage of passing tests
- **Coverage**: Code coverage percentage

Example output:
```
Performance Metrics:
Test Duration: PASS (45.2s)
Total Tests: 6
Coverage: 85.2%
```
