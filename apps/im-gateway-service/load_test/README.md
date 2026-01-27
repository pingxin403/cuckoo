# Load Testing for IM Gateway Service

This directory contains load testing scenarios for the IM Gateway Service using k6.

## Test Scenarios

### 1. Connection Load Test (`connection-load-test.js`)
Tests 100K concurrent WebSocket connections per Gateway node.

**Target Metrics**:
- Connection success rate > 95%
- P95 connection time < 5 seconds
- P99 message latency < 200ms
- P95 session duration > 30 minutes

### 2. Message Throughput Test (`message-throughput-test.js`)
Tests message throughput (10K messages/sec target).

**Target Metrics**:
- Throughput > 10,000 msg/sec
- P99 latency < 200ms
- Message success rate > 99%

### 3. Cluster Load Test (`cluster-load-test.js`)
Tests 10M concurrent users across 100-node cluster.

**Target Metrics**:
- 100K connections per node
- Cross-cluster message routing
- P99 latency < 200ms

## Quick Start

### Prerequisites
```bash
# Install k6
brew install k6  # macOS
# or see https://k6.io/docs/getting-started/installation/

# Configure system limits
ulimit -n 1000000
```

### Run Tests

**Quick Test (reduced scale)**:
```bash
./run-load-tests.sh quick
```

**Connection Load Test**:
```bash
./run-load-tests.sh connection
```

**Message Throughput Test**:
```bash
./run-load-tests.sh throughput
```

**All Tests**:
```bash
./run-load-tests.sh all
```

### Environment Variables

```bash
export GATEWAY_HOST="gateway.example.com"
export GATEWAY_PORT="8080"
export AUTH_TOKEN="your-test-token"
export MESSAGE_SIZE="1024"  # 1KB messages
```

## Documentation

See [LOAD_TEST_GUIDE.md](./LOAD_TEST_GUIDE.md) for comprehensive documentation including:
- Detailed test scenarios
- System configuration
- Performance tuning
- Monitoring during tests
- Troubleshooting
- CI/CD integration

## Results

Test results are saved to `results/YYYYMMDD_HHMMSS/` with:
- JSON output files
- Summary JSON files
- Test logs

## CI/CD Integration

See the Load Test Guide for GitHub Actions and GitLab CI examples.
