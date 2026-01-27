# Load Testing Guide for IM Gateway Service

## Overview

This guide describes how to perform load testing on the IM Gateway Service to validate performance under realistic load conditions.

## Test Scenarios

### 1. Connection Load Test (`connection-load-test.js`)

**Purpose**: Test 100K concurrent WebSocket connections per Gateway node

**Validates**: Requirements 6.1, 9.1

**Test Profile**:
- Ramp up from 0 to 100K connections over 15 minutes
- Hold 100K connections for 30 minutes
- Ramp down over 5 minutes
- Total duration: 50 minutes

**Metrics**:
- Connection success rate (target: > 95%)
- Connection establishment time (target: P95 < 5s)
- Message latency (target: P99 < 200ms)
- Session duration (target: P95 > 30 min)

**Resource Requirements**:
- Load generator: 8 CPU cores, 16GB RAM
- Gateway node: 16 CPU cores, 32GB RAM
- Network: 10 Gbps

### 2. Message Throughput Test (`message-throughput-test.js`)

**Purpose**: Test message throughput (messages/sec)

**Validates**: Requirements 1.1, 17.1

**Test Profile**:
- 1,000 concurrent users
- Each user sends 10 messages/sec
- Total: 10,000 messages/sec
- Duration: 10 minutes
- Total messages: 6,000,000

**Metrics**:
- Message throughput (target: > 10K msg/sec)
- Message latency (target: P99 < 200ms)
- Message success rate (target: > 99%)

**Resource Requirements**:
- Load generator: 4 CPU cores, 8GB RAM
- Gateway node: 16 CPU cores, 32GB RAM
- Network: 10 Gbps

### 3. Cluster Load Test (`cluster-load-test.js`)

**Purpose**: Test 10M concurrent users across cluster

**Validates**: Requirements 9.1

**Test Profile**:
- 100 Gateway nodes
- 100K connections per node
- Total: 10M concurrent connections
- Duration: 1 hour

**Metrics**:
- Total concurrent connections
- Connection distribution across nodes
- Message routing latency
- System resource usage

**Resource Requirements**:
- Load generators: 100 instances (8 cores, 16GB each)
- Gateway cluster: 100 nodes (16 cores, 32GB each)
- Infrastructure: etcd (3 nodes), Kafka (3 brokers), Redis (3 nodes)

## Prerequisites

### 1. Install k6

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6:latest
```

### 2. System Configuration

**Load Generator**:
```bash
# Increase file descriptor limit
ulimit -n 1000000

# Increase ephemeral port range
sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"

# Enable TCP reuse
sudo sysctl -w net.ipv4.tcp_tw_reuse=1

# Increase connection tracking
sudo sysctl -w net.netfilter.nf_conntrack_max=1000000
```

**Gateway Server**:
```bash
# Increase file descriptor limit
ulimit -n 1000000

# Increase connection backlog
sudo sysctl -w net.core.somaxconn=65535

# Increase receive buffer
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.rmem_default=134217728

# Increase send buffer
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.core.wmem_default=134217728
```

### 3. Environment Variables

```bash
export GATEWAY_HOST="gateway.example.com"
export GATEWAY_PORT="8080"
export AUTH_TOKEN="your-test-token"
export MESSAGE_SIZE="1024"  # 1KB messages
```

## Running Load Tests

### Connection Load Test

**Small Scale (1K connections)**:
```bash
k6 run --vus 1000 --duration 5m connection-load-test.js
```

**Medium Scale (10K connections)**:
```bash
k6 run connection-load-test.js \
  --stage 1m:10000 \
  --stage 5m:10000 \
  --stage 1m:0
```

**Large Scale (100K connections)**:
```bash
k6 run connection-load-test.js
```

**With Custom Configuration**:
```bash
k6 run connection-load-test.js \
  -e GATEWAY_HOST=gateway.example.com \
  -e GATEWAY_PORT=8080 \
  -e AUTH_TOKEN=your-token
```

### Message Throughput Test

**Standard Test (10K msg/sec)**:
```bash
k6 run message-throughput-test.js
```

**High Throughput (50K msg/sec)**:
```bash
k6 run message-throughput-test.js \
  --vus 5000 \
  --duration 10m
```

**Custom Message Size**:
```bash
k6 run message-throughput-test.js \
  -e MESSAGE_SIZE=4096  # 4KB messages
```

### Cluster Load Test

**Distributed Load Test**:
```bash
# On each load generator
k6 run cluster-load-test.js \
  --vus 1000 \
  --duration 1h \
  -e GATEWAY_HOST=gateway-node-${NODE_ID}.example.com
```

## Monitoring During Load Tests

### 1. Real-Time Metrics

**k6 Dashboard**:
```bash
k6 run --out influxdb=http://localhost:8086/k6 connection-load-test.js
```

**Grafana Dashboard**:
- Import k6 dashboard: https://grafana.com/grafana/dashboards/2587
- View real-time metrics during test execution

### 2. System Metrics

**Gateway Server**:
```bash
# CPU usage
top -p $(pgrep im-gateway)

# Memory usage
ps aux | grep im-gateway

# Network connections
ss -s
netstat -an | grep ESTABLISHED | wc -l

# File descriptors
lsof -p $(pgrep im-gateway) | wc -l
```

**Infrastructure**:
```bash
# etcd
etcdctl endpoint health --cluster
etcdctl endpoint status --cluster

# Redis
redis-cli INFO stats
redis-cli INFO memory

# Kafka
kafka-topics --bootstrap-server localhost:9092 --describe
```

### 3. Application Metrics

**Prometheus Queries**:
```promql
# Active connections
im_gateway_active_connections

# Message latency P99
histogram_quantile(0.99, rate(im_gateway_message_latency_bucket[5m]))

# Message throughput
rate(im_gateway_messages_sent_total[1m])

# Error rate
rate(im_gateway_errors_total[1m])
```

## Interpreting Results

### Connection Load Test Results

**Success Criteria**:
- ✅ Connection success rate > 95%
- ✅ P95 connection time < 5 seconds
- ✅ P99 message latency < 200ms
- ✅ P95 session duration > 30 minutes
- ✅ Memory per connection < 8KB

**Example Output**:
```
Connection Load Test Summary
============================

Connections:
  Total Attempts: 100000
  Success Rate: 98.50%
  Errors: 1500

Connection Duration:
  Min: 45.23ms
  Avg: 1234.56ms
  P95: 3456.78ms
  Max: 8901.23ms

Message Latency:
  P50: 45.67ms
  P95: 123.45ms
  P99: 178.90ms
```

### Message Throughput Test Results

**Success Criteria**:
- ✅ Throughput > 10,000 msg/sec
- ✅ P99 latency < 200ms
- ✅ Message success rate > 99%
- ✅ No memory leaks
- ✅ CPU usage < 80%

**Example Output**:
```
Message Throughput Test Summary
================================

Messages:
  Sent: 6000000
  Received: 5994000
  Errors: 6000
  Throughput: 10000.00 msg/sec

Message Latency:
  Min: 12.34ms
  Avg: 67.89ms
  P50: 56.78ms
  P95: 145.67ms
  P99: 189.01ms
  Max: 456.78ms

Success Rate:
  Message Success: 99.90%
```

## Performance Tuning

### 1. Gateway Configuration

**Increase Connection Limits**:
```yaml
# config.yaml
server:
  max_connections: 100000
  read_buffer_size: 4096
  write_buffer_size: 4096
  
websocket:
  handshake_timeout: 10s
  read_timeout: 90s
  write_timeout: 10s
  
connection_pool:
  max_idle_conns: 1000
  max_open_conns: 10000
```

### 2. System Tuning

**Linux Kernel Parameters**:
```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.tcp_keepalive_intvl = 30
```

### 3. Application Tuning

**Go Runtime**:
```bash
# Increase GOMAXPROCS
export GOMAXPROCS=16

# Increase memory limit
export GOMEMLIMIT=30GiB

# Enable CPU profiling
export GODEBUG=gctrace=1
```

## Troubleshooting

### Issue 1: Connection Timeouts

**Symptoms**:
- High connection failure rate
- Slow connection establishment

**Solutions**:
1. Increase `somaxconn` and `tcp_max_syn_backlog`
2. Increase Gateway `handshake_timeout`
3. Check network bandwidth
4. Verify load balancer configuration

### Issue 2: High Latency

**Symptoms**:
- P99 latency > 200ms
- Slow message delivery

**Solutions**:
1. Check CPU usage (should be < 80%)
2. Verify network latency
3. Check Redis/etcd response times
4. Optimize message routing logic
5. Enable connection pooling

### Issue 3: Memory Leaks

**Symptoms**:
- Memory usage continuously increasing
- OOM errors

**Solutions**:
1. Check for goroutine leaks (`pprof`)
2. Verify connection cleanup
3. Check message buffer sizes
4. Monitor GC frequency
5. Use memory profiling

### Issue 4: Connection Drops

**Symptoms**:
- Connections closing unexpectedly
- High reconnection rate

**Solutions**:
1. Increase `tcp_keepalive_time`
2. Verify heartbeat mechanism
3. Check load balancer timeout
4. Monitor network stability
5. Verify Registry TTL

## Best Practices

### 1. Gradual Ramp-Up

- Start with small load (1K connections)
- Gradually increase to target load
- Monitor metrics at each stage
- Identify bottlenecks early

### 2. Realistic Test Data

- Use realistic message sizes (1-4KB)
- Simulate real user behavior
- Include message bursts
- Test different message types

### 3. Long-Duration Tests

- Run tests for at least 1 hour
- Monitor for memory leaks
- Check connection stability
- Verify resource cleanup

### 4. Distributed Testing

- Use multiple load generators
- Distribute load evenly
- Synchronize test start times
- Aggregate results

### 5. Baseline Comparison

- Establish performance baseline
- Compare results over time
- Track performance regressions
- Document optimizations

## CI/CD Integration

### GitHub Actions

```yaml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
            --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
            sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Run Connection Load Test
        run: |
          cd apps/im-gateway-service/load_test
          k6 run --vus 1000 --duration 5m connection-load-test.js
      
      - name: Run Throughput Test
        run: |
          cd apps/im-gateway-service/load_test
          k6 run --vus 100 --duration 5m message-throughput-test.js
      
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: apps/im-gateway-service/load_test/summary.json
```

## References

- [k6 Documentation](https://k6.io/docs/)
- [WebSocket Load Testing](https://k6.io/docs/using-k6/protocols/websockets/)
- [Performance Tuning Guide](https://k6.io/docs/misc/fine-tuning-os/)
- [Grafana k6 Dashboard](https://grafana.com/grafana/dashboards/2587)
