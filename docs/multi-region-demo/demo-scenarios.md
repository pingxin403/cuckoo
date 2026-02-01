# Interactive Demo Scenarios

## 🎯 Overview

This document provides step-by-step interactive demonstrations of the multi-region active-active architecture. Each scenario showcases specific capabilities and can be run independently.

## 🚀 Prerequisites

### Environment Setup

```bash
# 1. Clone the repository
git clone <repository-url>
cd <repository-name>

# 2. Start the multi-region environment
cd deploy/docker
./start-multi-region.sh

# 3. Verify services are running
docker-compose ps

# Expected output:
# - im-service-region-a (running)
# - im-service-region-b (running)
# - im-gateway-region-a (running)
# - im-gateway-region-b (running)
# - mysql, redis, kafka, etcd (running)
```

### Access Points

- **Region-A Gateway**: http://localhost:8080
- **Region-B Gateway**: http://localhost:8081
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## 📋 Demo Scenarios

### Scenario 1: Normal Cross-Region Message Sync

**Goal**: Demonstrate message synchronization between regions

**Duration**: 5 minutes

**Steps**:

```bash
# 1. Send a message in Region-A
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "user1",
    "receiver_id": "user2",
    "content": "Hello from Region-A"
  }'

# Expected response:
# {
#   "message_id": "region-a-1704067200000-0-1",
#   "global_id": "region-a-1704067200000-0-1",
#   "status": "sent",
#   "timestamp": 1704067200000
# }

# 2. Wait for synchronization (< 500ms)
sleep 1

# 3. Query the message from Region-B
curl http://localhost:8081/api/v1/messages/region-a-1704067200000-0-1

# Expected response:
# {
#   "message_id": "region-a-1704067200000-0-1",
#   "global_id": "region-a-1704067200000-0-1",
#   "sender_id": "user1",
#   "receiver_id": "user2",
#   "content": "Hello from Region-A",
#   "region_id": "region-a",
#   "sync_status": "synced",
#   "timestamp": 1704067200000
# }

# 4. Check sync latency metrics
curl http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms

# Expected: P99 < 500ms
```

**Key Observations**:
- ✅ Message appears in Region-B within 500ms
- ✅ Global ID preserves region origin (region-a)
- ✅ Sync status updated to "synced"
- ✅ HLC timestamp maintains causality

---

### Scenario 2: Conflict Resolution (LWW)

**Goal**: Demonstrate conflict detection and resolution using LWW strategy

**Duration**: 10 minutes

**Steps**:

```bash
# 1. Create a message in Region-A
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message_id": "msg-001",
    "sender_id": "user1",
    "receiver_id": "user2",
    "content": "Original message"
  }'

# 2. Wait for sync
sleep 1

# 3. Simulate network partition (pause sync)
docker exec im-service-region-a tc qdisc add dev eth0 root netem loss 100%
docker exec im-service-region-b tc qdisc add dev eth0 root netem loss 100%

# 4. Update message in Region-A
curl -X PUT http://localhost:8080/api/v1/messages/msg-001 \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Updated by Region-A"
  }'

# 5. Update message in Region-B (concurrent)
curl -X PUT http://localhost:8081/api/v1/messages/msg-001 \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Updated by Region-B"
  }'

# 6. Restore network (resume sync)
docker exec im-service-region-a tc qdisc del dev eth0 root
docker exec im-service-region-b tc qdisc del dev eth0 root

# 7. Wait for conflict resolution
sleep 2

# 8. Check final state in both regions
curl http://localhost:8080/api/v1/messages/msg-001
curl http://localhost:8081/api/v1/messages/msg-001

# Expected: Both regions converge to the same version (higher HLC wins)

# 9. Check conflict metrics
curl http://localhost:9090/api/v1/query?query=cross_region_conflicts_total

# Expected: conflict counter incremented
```

**Key Observations**:
- ✅ Conflict detected when sync resumes
- ✅ LWW strategy selects version with higher HLC
- ✅ Both regions converge to same final state
- ✅ Conflict logged and metrics updated

---

### Scenario 3: Automatic Failover

**Goal**: Demonstrate automatic traffic failover on region failure

**Duration**: 5 minutes

**Steps**:

```bash
# 1. Establish WebSocket connection to Region-A
wscat -c ws://localhost:8080/ws

# 2. Send a message
> {"type": "send", "content": "Hello"}

# Expected: Message sent successfully

# 3. Simulate Region-A failure (in another terminal)
docker stop im-service-region-a im-gateway-region-a

# 4. Observe automatic reconnection
# Expected: Client receives reconnect message with Region-B endpoint

# 5. Client reconnects to Region-B
wscat -c ws://localhost:8081/ws

# 6. Send another message
> {"type": "send", "content": "Hello from Region-B"}

# Expected: Message sent successfully

# 7. Check failover metrics
curl http://localhost:9090/api/v1/query?query=failover_events_total

# Expected: failover event recorded

# 8. Check failover duration
curl http://localhost:9090/api/v1/query?query=failover_duration_ms

# Expected: RTO < 30 seconds

# 9. Restore Region-A
docker start im-service-region-a im-gateway-region-a
```

**Key Observations**:
- ✅ Failover detected within 15 seconds
- ✅ Client automatically reconnected to Region-B
- ✅ No message loss during failover
- ✅ RTO < 30 seconds achieved

---

### Scenario 4: Geo-Routing

**Goal**: Demonstrate intelligent geo-routing based on health checks

**Duration**: 5 minutes

**Steps**:

```bash
# 1. Check routing decision for healthy regions
curl http://localhost:8080/api/v1/routing/decision

# Expected response:
# {
#   "local_region": "region-a",
#   "local_healthy": true,
#   "remote_region": "region-b",
#   "remote_healthy": true,
#   "route_to": "region-a",
#   "reason": "local region healthy"
# }

# 2. Simulate Region-A unhealthy
curl -X POST http://localhost:8080/api/v1/health/set \
  -d '{"healthy": false}'

# 3. Check routing decision again
curl http://localhost:8080/api/v1/routing/decision

# Expected response:
# {
#   "local_region": "region-a",
#   "local_healthy": false,
#   "remote_region": "region-b",
#   "remote_healthy": true,
#   "route_to": "region-b",
#   "reason": "local region unhealthy, failover to remote"
# }

# 4. Send a message (should be routed to Region-B)
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "user1",
    "receiver_id": "user2",
    "content": "Routed to Region-B"
  }'

# 5. Verify message was processed in Region-B
curl http://localhost:8081/api/v1/messages | grep "Routed to Region-B"

# 6. Restore Region-A health
curl -X POST http://localhost:8080/api/v1/health/set \
  -d '{"healthy": true}'
```

**Key Observations**:
- ✅ Routing decision based on health checks
- ✅ Automatic failover to healthy region
- ✅ Traffic resumes to local region when healthy
- ✅ No manual intervention required

---

### Scenario 5: HLC Causality Preservation

**Goal**: Demonstrate HLC preserves message causality

**Duration**: 10 minutes

**Steps**:

```bash
# 1. Send message A in Region-A
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "user1",
    "receiver_id": "user2",
    "content": "Message A"
  }'

# Response: global_id = "region-a-1000-0-1"

# 2. Wait for sync
sleep 1

# 3. Send message B in Region-B (causally after A)
curl -X POST http://localhost:8081/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "user2",
    "receiver_id": "user1",
    "content": "Message B (reply to A)"
  }'

# Response: global_id = "region-b-1001-0-1"
# Note: HLC timestamp > A's timestamp (causality preserved)

# 4. Query messages in order
curl http://localhost:8080/api/v1/messages?order=global_id

# Expected: Messages ordered by HLC (A before B)

# 5. Verify causality in both regions
curl http://localhost:8080/api/v1/messages?order=global_id
curl http://localhost:8081/api/v1/messages?order=global_id

# Expected: Same order in both regions
```

**Key Observations**:
- ✅ HLC timestamps preserve causality
- ✅ Messages ordered consistently across regions
- ✅ No coordination needed between regions
- ✅ Causality maintained even with clock skew

---

### Scenario 6: Performance Benchmarking

**Goal**: Measure cross-region sync latency and throughput

**Duration**: 15 minutes

**Steps**:

```bash
# 1. Run performance test script
cd tests/e2e/multi-region
./run-performance-test.sh

# The script will:
# - Send 10,000 messages to Region-A
# - Measure sync latency to Region-B
# - Calculate throughput
# - Generate latency distribution

# 2. View results
cat performance-results.txt

# Expected output:
# Total messages: 10,000
# Sync latency P50: 150ms
# Sync latency P95: 350ms
# Sync latency P99: 480ms
# Throughput: 5,000 msg/s
# Success rate: 99.99%

# 3. View latency histogram
cat latency-histogram.txt

# 4. Check Prometheus metrics
curl http://localhost:9090/api/v1/query?query=histogram_quantile(0.99,cross_region_sync_latency_ms)

# 5. View Grafana dashboard
open http://localhost:3000/d/multi-region-performance
```

**Key Observations**:
- ✅ P99 sync latency < 500ms
- ✅ Throughput > 5,000 msg/s
- ✅ Success rate > 99.9%
- ✅ Performance meets SLA requirements

---

## 🎥 Recording Demos

### Using asciinema

```bash
# Install asciinema
brew install asciinema  # macOS
apt-get install asciinema  # Ubuntu

# Record a demo
asciinema rec demo-scenario-1.cast

# Run your demo commands
# ...

# Stop recording
exit

# Play back
asciinema play demo-scenario-1.cast

# Upload to asciinema.org
asciinema upload demo-scenario-1.cast
```

### Using screen recording

```bash
# macOS: Use QuickTime Player or Screenshot app
# Linux: Use SimpleScreenRecorder or OBS Studio
# Windows: Use OBS Studio or Windows Game Bar
```

## 📊 Monitoring During Demos

### Key Metrics to Watch

1. **Cross-Region Sync Latency**
   ```promql
   histogram_quantile(0.99, cross_region_sync_latency_ms)
   ```

2. **Conflict Rate**
   ```promql
   rate(cross_region_conflicts_total[5m])
   ```

3. **Failover Events**
   ```promql
   increase(failover_events_total[1m])
   ```

4. **Active Connections**
   ```promql
   active_websocket_connections
   ```

### Grafana Dashboards

- **Multi-Region Overview**: http://localhost:3000/d/multi-region-overview
- **Performance Metrics**: http://localhost:3000/d/multi-region-performance
- **Conflict Analysis**: http://localhost:3000/d/multi-region-conflicts

## 🐛 Troubleshooting

### Common Issues

**Issue 1**: Services not starting

```bash
# Check logs
docker-compose logs im-service-region-a

# Restart services
docker-compose restart
```

**Issue 2**: Sync not working

```bash
# Check Kafka connectivity
docker exec im-service-region-a kafka-topics --list --bootstrap-server kafka:9092

# Check network connectivity
docker exec im-service-region-a ping im-service-region-b
```

**Issue 3**: High latency

```bash
# Check network delay
docker exec im-service-region-a tc qdisc show dev eth0

# Remove artificial delay
docker exec im-service-region-a tc qdisc del dev eth0 root
```

## 📚 Additional Resources

- [Architecture Overview](./architecture-overview.md)
- [Monitoring Dashboard](./monitoring-dashboard.md)
- [Troubleshooting Guide](./troubleshooting-guide.md)
- [Performance Benchmarks](./performance-demo.md)

## 🤝 Contributing

To add new demo scenarios:

1. Document the scenario in this file
2. Create automation scripts in `scripts/`
3. Add test cases in `tests/e2e/multi-region/`
4. Update the main README

---

**Next Steps**: Try the [Chaos Engineering Demo](./chaos-engineering-demo.md) for advanced fault injection scenarios.
