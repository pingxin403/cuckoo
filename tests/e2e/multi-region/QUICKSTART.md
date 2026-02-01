# Multi-Region E2E Tests - Quick Start Guide

## TL;DR

```bash
cd tests/e2e/multi-region
./run-e2e-tests.sh
```

That's it! The script will:
1. ✅ Check prerequisites (Docker, Go, etc.)
2. ✅ Start multi-region infrastructure
3. ✅ Wait for services to be ready
4. ✅ Run all verification tests
5. ✅ Generate test report
6. ✅ Clean up automatically

## What Gets Tested

### 7 Comprehensive Test Scenarios

1. **Cross-Region Message Routing** - Geo-based routing and health checks
2. **IM Service Multi-Region** - Service extensions and HLC sync
3. **etcd Coordination** - Distributed service discovery and locking
4. **Failover Mechanisms** - Automatic failure detection and recovery
5. **HLC Global IDs** - Unique ID generation and causal ordering
6. **Conflict Resolution** - LWW strategy and deterministic resolution
7. **Sync Latency** - Cross-region replication performance

## Prerequisites

- Docker & Docker Compose
- Go 1.21+
- ~5 minutes for test execution

## Commands

```bash
# Run everything (recommended)
./run-e2e-tests.sh

# Or run step-by-step
./run-e2e-tests.sh start   # Start infrastructure
./run-e2e-tests.sh test    # Run tests
./run-e2e-tests.sh stop    # Stop infrastructure

# Debugging
./run-e2e-tests.sh logs    # View service logs
```

## Expected Results

```
✅ All 7 test scenarios pass
✅ Test duration: ~45 seconds
✅ All services healthy
✅ Performance metrics within targets
```

## If Tests Fail

1. Check service logs: `./run-e2e-tests.sh logs`
2. Verify services are running: `cd ../../deploy/docker && ./start-multi-region.sh status`
3. Check the [troubleshooting guide](./README.md#troubleshooting)

## What's Running

### Region A (Primary - Beijing)
- IM Service: `localhost:9194` (gRPC), `localhost:8184` (HTTP)
- Gateway: `localhost:9197` (gRPC), `localhost:8182` (WebSocket)
- Redis DB: 2

### Region B (Secondary - Shanghai)
- IM Service: `localhost:9294` (gRPC), `localhost:8284` (HTTP)
- Gateway: `localhost:9297` (gRPC), `localhost:8282` (WebSocket)
- Redis DB: 3

### Shared Infrastructure
- MySQL, Redis, Kafka, etcd

## Performance Targets

| Metric | Target | Typical |
|--------|--------|---------|
| Local Redis Write | < 100ms | 1-5ms |
| Cross-Region Routing | < 50ms | 10-20ms |
| HLC Generation | < 1ms | 0.1-0.5ms |
| Conflict Resolution | < 10ms | 1-3ms |
| Failover Detection | < 35s | 30-35s |
| End-to-End Sync | < 500ms | 100-200ms |

## More Information

- [Full README](./README.md) - Complete documentation
- [Task Summary](./TASK_10.1_SUMMARY.md) - Implementation details
- [Deployment Guide](../../../deploy/docker/MULTI_REGION_DEPLOYMENT.md) - Infrastructure setup

## Support

Questions? Check the [README troubleshooting section](./README.md#troubleshooting) or run:

```bash
./run-e2e-tests.sh help
```
