# Multi-Region End-to-End Verification Tests

## Overview


## Test Coverage


The test suite validates:

1. **Cross-Region Message Routing** (Requirement 3.1)
   - Geo-based routing decisions
   - Health-aware routing
   - Latency measurement
   - Routing metrics collection

2. **IM Service Multi-Region Functionality** (Requirements 1.1, 2.1)
   - Service accessibility in both regions
   - Region-aware sequence generation
   - HLC synchronization between regions
   - Global ID generation with region prefix

3. **etcd Distributed Coordination** (Requirement 6.4)
   - Service registration in multiple regions
   - Cross-region service discovery
   - Distributed locking for coordination
   - Conflict-free concurrent operations

4. **Failover Mechanisms** (Requirements 4.1, 4.2)
   - Health check detection
   - Automatic failover on region failure
   - Routing decision changes during failure
   - Service recovery detection

5. **HLC Global ID Generation** (Requirement 2.1)
   - Unique ID generation across regions
   - Monotonicity guarantees
   - Cross-region clock synchronization
   - Causal ordering preservation

6. **Conflict Resolution** (Requirement 2.2)
   - LWW (Last Write Wins) strategy
   - Deterministic conflict resolution
   - Region ID tiebreaker
   - Conflict metrics recording

7. **Cross-Region Sync Latency** (Requirement 1.1)
   - Local write latency measurement
   - Network latency estimation
   - End-to-end sync latency validation
   - P99 < 500ms requirement verification

## Prerequisites

### Infrastructure Requirements

1. **Docker Compose Multi-Region Setup**
   ```bash
   # Start multi-region infrastructure
   cd deploy/docker
   ./start-multi-region.sh start
   ```

2. **Required Services**
   - Region A: IM Service (port 9194), Gateway (port 8182), Redis (DB 2), etcd
   - Region B: IM Service (port 9294), Gateway (port 8282), Redis (DB 3), etcd
   - Shared: MySQL, Kafka, etcd

3. **Network Configuration**
   - Region A network: 172.20.0.0/16
   - Region B network: 172.21.0.0/16
   - Shared network: monorepo-network

### Software Requirements

- Go 1.21+
- Docker & Docker Compose
- Redis CLI (for debugging)
- etcdctl (for debugging)

## Running the Tests

### Quick Start

```bash
# Run all end-to-end verification tests
./run-e2e-tests.sh
```

### Manual Execution

```bash
# 1. Start multi-region infrastructure
cd deploy/docker
./start-multi-region.sh start

# 2. Wait for services to be ready
./start-multi-region.sh test

# 3. Run the tests
cd tests/e2e/multi-region
go test -v -tags=e2e -timeout 15m

# 4. Cleanup
cd deploy/docker
./start-multi-region.sh stop
```

### Running Specific Tests

```bash
# Run only cross-region routing tests
go test -v -tags=e2e -run TestEndToEndMultiRegionVerification/CrossRegionMessageRouting

# Run only failover tests
go test -v -tags=e2e -run TestEndToEndMultiRegionVerification/FailoverMechanisms

# Run only HLC tests
go test -v -tags=e2e -run TestEndToEndMultiRegionVerification/HLCGlobalIDGeneration
```

## Environment Variables

Configure the test environment using these variables:

```bash
# Region A configuration
export REGION_A_IM_SERVICE_ADDR="localhost:9194"
export REGION_A_GATEWAY_ADDR="localhost:8182"
export REGION_A_REDIS_ADDR="localhost:6379"
export REGION_A_ETCD_ADDR="localhost:2379"

# Region B configuration
export REGION_B_IM_SERVICE_ADDR="localhost:9294"
export REGION_B_GATEWAY_ADDR="localhost:8282"
export REGION_B_REDIS_ADDR="localhost:6379"
export REGION_B_ETCD_ADDR="localhost:2379"

# Shared infrastructure
export SHARED_ETCD_ADDR="localhost:2379"
```

## Test Architecture

### Test Flow

```
┌─────────────────────────────────────────────────────────────┐
│                  Test Orchestrator                          │
│              (end_to_end_verification_test.go)              │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐  ┌──────▼──────┐
│   Region A     │  │   Region B     │  │   Shared    │
│   Components   │  │   Components   │  │   etcd      │
└────────────────┘  └────────────────┘  └─────────────┘
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐         │
│ IM Service     │  │ IM Service     │         │
│ Gateway        │  │ Gateway        │         │
│ Redis (DB 2)   │  │ Redis (DB 3)   │         │
│ etcd Client    │  │ etcd Client    │         │
│ HLC            │  │ HLC            │         │
│ Conflict Res.  │  │ Conflict Res.  │         │
│ Geo Router     │  │ Geo Router     │         │
└────────────────┘  └────────────────┘         │
                                               │
                    ┌──────────────────────────┘
                    │
        ┌───────────▼───────────┐
        │  Validation & Metrics │
        └───────────────────────┘
```

### Test Components

1. **MultiRegionTestEnvironment**: Manages all test infrastructure
   - Region A components (services, clients, routers)
   - Region B components (services, clients, routers)
   - Shared infrastructure (etcd)
   - Cleanup functions

2. **Test Cases**: Individual verification tests
   - Each test validates specific requirements
   - Tests are independent and can run in any order
   - Proper setup and teardown for each test

3. **Helper Functions**: Utility functions for common operations
   - Service readiness checks
   - Sequence generation
   - Environment variable handling

## Expected Results

### Success Criteria

All tests should pass with the following validations:

✅ **Cross-Region Routing**
- Geo router routes to local region when healthy
- Peer region health is detected correctly
- Routing latency is measured
- Routing metrics are collected

✅ **IM Service Multi-Region**
- Both regions are accessible via gRPC
- Sequence IDs contain region identifiers
- HLC synchronization works correctly
- Region B HLC advances after receiving Region A's HLC

✅ **etcd Coordination**
- Services register in their respective regions
- Cross-region service discovery works
- Distributed locks prevent concurrent access
- Lock acquisition is exclusive

✅ **Failover Mechanisms**
- Health checks detect region failures
- Routing decisions change during failures
- Services recover after restoration
- Health status updates correctly

✅ **HLC Global IDs**
- 1000+ unique IDs generated without collision
- HLC values are monotonically increasing
- Cross-region synchronization maintains causality
- Causal ordering is preserved

✅ **Conflict Resolution**
- Conflicts are detected correctly
- LWW strategy selects higher HLC
- Resolution is deterministic
- Conflict metrics are recorded

✅ **Sync Latency**
- Local Redis writes < 100ms
- Estimated P99 sync latency < 500ms
- All latency components measured
- Requirement validation passes

### Performance Metrics

Expected performance characteristics:

| Metric | Target | Typical |
|--------|--------|---------|
| Local Redis Write | < 100ms | 1-5ms |
| Cross-Region Routing | < 50ms | 10-20ms |
| HLC Generation | < 1ms | 0.1-0.5ms |
| Conflict Resolution | < 10ms | 1-3ms |
| Health Check Interval | 30s | 30s |
| Failover Detection | < 35s | 30-35s |
| End-to-End Sync (estimated) | < 500ms | 100-200ms |

## Troubleshooting

### Common Issues

#### 1. Services Not Ready

**Symptom**: Tests fail with connection errors

**Solution**:
```bash
# Check service status
cd deploy/docker
./start-multi-region.sh status

# Check service health
./start-multi-region.sh test

# View service logs
./start-multi-region.sh logs
```

#### 2. Redis Connection Failed

**Symptom**: `failed to connect to Redis`

**Solution**:
```bash
# Check Redis is running
docker ps | grep redis

# Test Redis connectivity
redis-cli -h localhost -p 6379 ping

# Check Redis DB configuration
redis-cli -h localhost -p 6379 SELECT 2
redis-cli -h localhost -p 6379 SELECT 3
```

#### 3. etcd Connection Failed

**Symptom**: `failed to connect to etcd`

**Solution**:
```bash
# Check etcd is running
docker ps | grep etcd

# Test etcd connectivity
docker exec etcd etcdctl endpoint health

# Check etcd endpoints
docker exec etcd etcdctl member list
```

#### 4. Geo Router Health Check Timeout

**Symptom**: Health checks don't complete in time

**Solution**:
- Increase health check interval in test
- Verify network connectivity between regions
- Check gateway service logs for errors

#### 5. HLC Synchronization Issues

**Symptom**: HLC values don't synchronize correctly

**Solution**:
- Check system clock synchronization (NTP)
- Verify HLC update logic
- Check for clock drift between regions

### Debug Mode

Enable verbose logging:

```bash
# Run tests with verbose output
go test -v -tags=e2e -timeout 15m

# Enable Go race detector
go test -v -tags=e2e -race -timeout 15m

# Run with debug logging
GODEBUG=gctrace=1 go test -v -tags=e2e -timeout 15m
```

### Manual Verification

Verify infrastructure manually:

```bash
# Check Region A IM Service
curl http://localhost:8184/health

# Check Region B IM Service
curl http://localhost:8284/health

# Check Region A Gateway
curl http://localhost:8182/health

# Check Region B Gateway
curl http://localhost:8282/health

# Check etcd service registry
docker exec etcd etcdctl get /im/services/ --prefix

# Check Redis keys
redis-cli -h localhost -p 6379 --scan --pattern "test:*"
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Multi-Region E2E Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Start Multi-Region Infrastructure
        run: |
          cd deploy/docker
          ./start-multi-region.sh start
          ./start-multi-region.sh test
      
      - name: Run E2E Tests
        run: |
          cd tests/e2e/multi-region
          go test -v -tags=e2e -timeout 15m
      
      - name: Cleanup
        if: always()
        run: |
          cd deploy/docker
          ./start-multi-region.sh stop
```

### GitLab CI

```yaml
e2e-multi-region:
  stage: test
  image: golang:1.21
  services:
    - docker:dind
  script:
    - cd deploy/docker
    - ./start-multi-region.sh start
    - ./start-multi-region.sh test
    - cd ../../tests/e2e/multi-region
    - go test -v -tags=e2e -timeout 15m
  after_script:
    - cd deploy/docker
    - ./start-multi-region.sh stop
  only:
    - main
    - develop
    - merge_requests
```

## Next Steps


   - Load testing with cross-region traffic
   - Latency measurement under load
   - Consistency verification with concurrent writes

   - Update deployment documentation
   - Create operational runbooks
   - Document troubleshooting procedures

3. **Phase 2 Tasks**: Production readiness
   - Database migration scripts
   - Monitoring dashboards
   - Alerting rules
   - Capacity planning

## References

- [Docker Deployment Guide](../../../deploy/docker/MULTI_REGION_DEPLOYMENT.md)
- [Integration Guide](../../../apps/MULTI_REGION_INTEGRATION_COMPLETE.md)

## Support

For issues or questions:
1. Check this README's troubleshooting section
2. Review service logs: `./start-multi-region.sh logs`
3. Check service health: `./start-multi-region.sh test`
4. Consult the [deployment guide](../../../deploy/docker/MULTI_REGION_DEPLOYMENT.md)
5. Open an issue with test output and logs
