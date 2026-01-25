# Infrastructure Integration Tests Guide

## Overview

This document describes the infrastructure integration tests for the IM Chat System. These tests validate the resilience and failover capabilities of the underlying infrastructure components.

## Test Coverage

### 1. Etcd Cluster Failover (`TestEtcdClusterFailover`)

**Purpose**: Validate etcd cluster resilience and leader election

**What it tests**:
- ✅ Connection to etcd cluster
- ✅ Write operations to etcd
- ✅ Read operations from etcd
- ✅ Leader election status
- ✅ Watch mechanism for change detection
- ✅ Failover detection

**Requirements validated**: 10.3 (etcd cluster resilience)

**Expected behavior**:
- Cluster should remain available even if one node fails
- Leader election should occur automatically
- Watch events should be delivered reliably
- Data should be consistent across nodes

### 2. Kafka Broker Failover (`TestKafkaBrokerFailover`)

**Purpose**: Validate Kafka broker resilience and message replication

**What it tests**:
- ✅ Topic creation with replication
- ✅ Message production to Kafka
- ✅ Message consumption from Kafka
- ✅ Message ordering and delivery
- ✅ Broker failover handling

**Requirements validated**: 10.3 (Kafka cluster resilience)

**Expected behavior**:
- Messages should be delivered reliably
- Consumers should receive all messages
- System should handle broker failures gracefully
- Message ordering should be preserved within partitions

### 3. Redis Failover (`TestRedisFailover`)

**Purpose**: Validate Redis resilience and persistence

**What it tests**:
- ✅ Connection to Redis
- ✅ Write operations with TTL
- ✅ Read operations
- ✅ TTL functionality
- ✅ Persistence configuration
- ✅ Connection pooling with concurrent requests

**Requirements validated**: 10.3 (Redis resilience)

**Expected behavior**:
- Redis should handle concurrent requests
- TTL should work correctly
- Persistence should be configured
- Connection pool should handle multiple clients

### 4. MySQL Connection Pooling (`TestMySQLConnectionPooling`)

**Purpose**: Validate MySQL connection pooling and resilience

**What it tests**:
- ✅ Connection pool configuration
- ✅ Concurrent database operations
- ✅ Connection pool statistics
- ✅ Query timeout handling
- ✅ Connection reuse

**Requirements validated**: 10.4 (MySQL connection pooling)

**Expected behavior**:
- Connection pool should handle 25 max open connections
- Concurrent inserts should succeed
- Timeouts should be handled gracefully
- Connection pool should reuse connections efficiently

**Connection pool configuration**:
```go
db.SetMaxOpenConns(25)      // Maximum open connections
db.SetMaxIdleConns(5)       // Maximum idle connections
db.SetConnMaxLifetime(5 * time.Minute)  // Connection lifetime
```

### 5. Network Partition Scenario (`TestNetworkPartitionScenario`)

**Purpose**: Validate system behavior during network partitions

**What it tests**:
- ✅ Timeout handling for etcd
- ✅ Timeout handling for Redis
- ✅ Graceful degradation
- ✅ Error detection

**Requirements validated**: 10.3 (network partition resilience)

**Expected behavior**:
- System should detect network partitions via timeouts
- Operations should fail gracefully
- Errors should be properly reported
- System should recover when partition is resolved

## Running the Tests

### Prerequisites

1. Docker and Docker Compose installed
2. Go 1.21+ installed
3. Infrastructure services running (MySQL, Redis, etcd, Kafka)

### Run All Infrastructure Tests

```bash
cd apps/im-service
go test -v -tags=integration ./integration_test/... -run TestEtcd -timeout 10m
go test -v -tags=integration ./integration_test/... -run TestKafka -timeout 10m
go test -v -tags=integration ./integration_test/... -run TestRedis -timeout 10m
go test -v -tags=integration ./integration_test/... -run TestMySQL -timeout 10m
go test -v -tags=integration ./integration_test/... -run TestNetwork -timeout 10m
```

### Run Specific Test

```bash
# Test etcd failover
go test -v -tags=integration ./integration_test/... -run TestEtcdClusterFailover

# Test Kafka failover
go test -v -tags=integration ./integration_test/... -run TestKafkaBrokerFailover

# Test Redis failover
go test -v -tags=integration ./integration_test/... -run TestRedisFailover

# Test MySQL connection pooling
go test -v -tags=integration ./integration_test/... -run TestMySQLConnectionPooling

# Test network partition
go test -v -tags=integration ./integration_test/... -run TestNetworkPartitionScenario
```

### Run with Docker Compose

```bash
cd apps/im-service/integration_test
./run-infrastructure-tests.sh
```

## Test Environment Setup

### Docker Compose Configuration

The infrastructure tests use the same Docker Compose environment as the E2E tests:

```yaml
services:
  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: im_chat

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    ports:
      - "2379:2379"
      - "2380:2380"

  zookeeper:
    image: confluentinc/cp-zookeeper:7.4.0
    ports:
      - "2181:2181"

  kafka:
    image: confluentinc/cp-kafka:7.4.0
    ports:
      - "9092:9092"
```

### Environment Variables

```bash
export MYSQL_ADDR="root:password@tcp(localhost:3306)/im_chat"
export REDIS_ADDR="localhost:6379"
export ETCD_ADDR="localhost:2379"
export KAFKA_ADDR="localhost:9092"
```

## Failover Testing Scenarios

### Etcd Cluster Failover

**Scenario 1: Single Node Failure**
1. Start 3-node etcd cluster
2. Write data to cluster
3. Stop one node
4. Verify cluster still operational
5. Verify data still accessible
6. Restart node
7. Verify cluster recovers

**Scenario 2: Leader Failure**
1. Identify current leader
2. Stop leader node
3. Verify new leader elected
4. Verify writes still work
5. Restart old leader
6. Verify cluster stable

### Kafka Broker Failover

**Scenario 1: Producer Failover**
1. Start producing messages
2. Stop one broker
3. Verify producer continues
4. Verify messages delivered
5. Restart broker
6. Verify rebalancing

**Scenario 2: Consumer Failover**
1. Start consuming messages
2. Stop one broker
3. Verify consumer continues
4. Verify no message loss
5. Restart broker
6. Verify rebalancing

### Redis Failover

**Scenario 1: Master Failure**
1. Write data to master
2. Stop master
3. Verify replica promoted (if configured)
4. Verify data accessible
5. Restart old master
6. Verify replication resumes

**Scenario 2: Connection Pool Exhaustion**
1. Create many concurrent connections
2. Verify pool limits enforced
3. Verify connections reused
4. Verify no connection leaks

### MySQL Connection Pool

**Scenario 1: Connection Exhaustion**
1. Create 25+ concurrent queries
2. Verify pool limits enforced
3. Verify queries queued
4. Verify no connection leaks

**Scenario 2: Long-Running Query**
1. Execute long query
2. Verify timeout enforced
3. Verify connection released
4. Verify pool recovers

## Monitoring and Metrics

### Etcd Metrics

```bash
# Check cluster health
etcdctl endpoint health --cluster

# Check cluster status
etcdctl endpoint status --cluster

# Check member list
etcdctl member list
```

### Kafka Metrics

```bash
# List topics
kafka-topics --bootstrap-server localhost:9092 --list

# Describe topic
kafka-topics --bootstrap-server localhost:9092 --describe --topic test-topic

# Check consumer groups
kafka-consumer-groups --bootstrap-server localhost:9092 --list
```

### Redis Metrics

```bash
# Check Redis info
redis-cli INFO

# Check persistence
redis-cli INFO persistence

# Check replication
redis-cli INFO replication

# Monitor commands
redis-cli MONITOR
```

### MySQL Metrics

```sql
-- Check connection pool
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';

-- Check slow queries
SHOW VARIABLES LIKE 'slow_query_log';
SHOW STATUS LIKE 'Slow_queries';

-- Check table status
SHOW TABLE STATUS FROM im_chat;
```

## Troubleshooting

### Etcd Issues

**Problem**: Connection timeout
```
Solution: Check etcd cluster health, verify network connectivity
```

**Problem**: Leader election fails
```
Solution: Ensure at least 2 nodes are running (quorum)
```

### Kafka Issues

**Problem**: Topic creation fails
```
Solution: Check Zookeeper connection, verify broker health
```

**Problem**: Messages not delivered
```
Solution: Check consumer group, verify topic exists, check broker logs
```

### Redis Issues

**Problem**: Connection refused
```
Solution: Check Redis is running, verify port 6379 is accessible
```

**Problem**: Out of memory
```
Solution: Check maxmemory setting, verify eviction policy
```

### MySQL Issues

**Problem**: Too many connections
```
Solution: Increase max_connections, check for connection leaks
```

**Problem**: Slow queries
```
Solution: Add indexes, optimize queries, check slow query log
```

## Best Practices

### 1. Test Isolation

- Each test should be independent
- Clean up test data after each test
- Use unique identifiers (timestamps) for test data
- Don't rely on test execution order

### 2. Timeout Handling

- Always use context with timeout
- Set reasonable timeout values
- Handle timeout errors gracefully
- Log timeout occurrences

### 3. Error Handling

- Check all error returns
- Log errors with context
- Don't fail silently
- Provide meaningful error messages

### 4. Resource Cleanup

- Always defer cleanup operations
- Use `defer` for resource cleanup
- Handle cleanup errors
- Don't leave test data behind

### 5. Concurrent Testing

- Test concurrent operations
- Use goroutines for concurrency
- Use channels for synchronization
- Verify thread safety

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run Infrastructure Tests
  run: |
    cd apps/im-service
    go test -v -tags=integration ./integration_test/... \
      -run "TestEtcd|TestKafka|TestRedis|TestMySQL|TestNetwork" \
      -timeout 15m
```

### GitLab CI

```yaml
infrastructure-tests:
  stage: test
  script:
    - cd apps/im-service
    - go test -v -tags=integration ./integration_test/... 
        -run "TestEtcd|TestKafka|TestRedis|TestMySQL|TestNetwork"
        -timeout 15m
```

## Performance Benchmarks

### Expected Performance

- **Etcd**: < 10ms for single write, < 5ms for single read
- **Kafka**: > 10,000 messages/sec throughput
- **Redis**: < 1ms for single operation
- **MySQL**: < 10ms for simple query with connection pool

### Load Testing

```bash
# Etcd load test
for i in {1..1000}; do
  etcdctl put /test/key$i value$i
done

# Redis load test
redis-benchmark -t set,get -n 100000 -q

# MySQL load test
sysbench --test=oltp --mysql-user=root --mysql-password=password \
  --mysql-db=im_chat --oltp-table-size=10000 prepare
```

## References

- [Etcd Documentation](https://etcd.io/docs/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Redis Documentation](https://redis.io/documentation)
- [MySQL Documentation](https://dev.mysql.com/doc/)
- [Go Testing Package](https://pkg.go.dev/testing)
