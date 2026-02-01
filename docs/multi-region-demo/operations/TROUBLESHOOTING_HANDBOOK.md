# Multi-Region Active-Active System - Troubleshooting Handbook

**Version**: 1.0  
**Last Updated**: 2024  
**Maintained By**: Platform Engineering Team

## 📋 Table of Contents

1. [Overview](#overview)
2. [Quick Diagnostic Checklist](#quick-diagnostic-checklist)
3. [Common Issues](#common-issues)
4. [Cross-Region Sync Issues](#cross-region-sync-issues)
5. [Conflict Resolution Issues](#conflict-resolution-issues)
6. [Failover Issues](#failover-issues)
7. [Performance Degradation](#performance-degradation)
8. [Data Consistency Issues](#data-consistency-issues)
9. [Network Issues](#network-issues)
10. [Diagnostic Tools](#diagnostic-tools)

---

## Overview

This handbook provides step-by-step troubleshooting procedures for the multi-region active-active IM chat system. It covers common issues, diagnostic techniques, and resolution strategies specific to distributed multi-region architectures.

### Severity Levels

| Level | Response Time | Impact | Examples |
|-------|---------------|--------|----------|
| **P0 - Critical** | < 5 minutes | Complete service outage | Region down, data loss |
| **P1 - High** | < 15 minutes | Significant degradation | High sync latency, failover issues |
| **P2 - Medium** | < 1 hour | Minor degradation | High conflict rate, cache issues |
| **P3 - Low** | < 4 hours | No user impact | Monitoring gaps, optimization needed |

### Prerequisites

Before troubleshooting, ensure you have:
- Access to Grafana dashboards (http://localhost:3000)
- Access to Prometheus (http://localhost:9090)
- SSH/kubectl access to both regions
- Log aggregation access (Loki)
- Understanding of HLC and conflict resolution mechanisms

---

## Quick Diagnostic Checklist

### 1. System Health Check (2 minutes)

```bash
# Check all services are running
docker ps | grep -E "im-service|im-gateway|mysql|redis|kafka|etcd"

# Check service health endpoints
curl http://localhost:8080/health  # IM Service Region A
curl http://localhost:8081/health  # IM Service Region B
curl http://localhost:9093/health  # IM Gateway Region A
curl http://localhost:9094/health  # IM Gateway Region B

# Check Grafana dashboards
open http://localhost:3000/d/multi-region-overview
```

### 2. Cross-Region Connectivity Check (1 minute)

```bash
# Test network connectivity between regions
docker exec im-service-region-a ping -c 3 im-service-region-b
docker exec im-service-region-b ping -c 3 im-service-region-a

# Check Kafka cross-region replication
docker exec kafka-region-a kafka-topics.sh --list --bootstrap-server kafka-region-b:9092

# Check etcd cluster health
docker exec etcd-region-a etcdctl endpoint health --cluster
```

### 3. Key Metrics Check (1 minute)

```promql
# In Prometheus (http://localhost:9090)

# Cross-region sync latency
histogram_quantile(0.99, cross_region_sync_latency_ms)

# Conflict rate
rate(cross_region_conflicts_total[5m])

# Message throughput
sum(rate(messages_sent_total[5m])) by (region)

# Active connections
sum(active_websocket_connections) by (region)
```

---

## Common Issues

### Issue 1: High Cross-Region Sync Latency

**Symptoms**:
- P99 sync latency > 500ms
- Alert: `HighCrossRegionSyncLatency`
- Users reporting delayed message delivery across regions
- Grafana dashboard shows latency spikes

**Diagnosis Steps**:

1. **Check network latency between regions**:
```bash
# Measure network RTT
docker exec im-service-region-a ping -c 10 im-service-region-b | tail -1

# Expected: avg < 50ms (same cloud provider, different AZs)
# If > 100ms: Network issue
```

2. **Check Kafka consumer lag**:
```bash
docker exec kafka-region-a kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group cross-region-sync-group \
  --describe

# Look for: LAG column
# If LAG > 1000: Consumer is falling behind
```

3. **Check database replication lag**:
```bash
docker exec mysql-region-b mysql -u root -ppassword -e \
  "SHOW SLAVE STATUS\G" | grep Seconds_Behind_Master

# Expected: 0-2 seconds
# If > 5 seconds: Database replication issue
```

4. **Check service resource usage**:
```bash
docker stats im-service-region-a im-service-region-b

# Look for: CPU > 80%, Memory > 80%
# If high: Resource exhaustion
```

**Resolution Steps**:

**Scenario A: Network Latency Issue**
```bash
# 1. Verify network path
traceroute im-service-region-b.internal

# 2. Check for packet loss
mtr -r -c 100 im-service-region-b.internal

# 3. If persistent, contact network team
# 4. Consider enabling compression for cross-region traffic
```

**Scenario B: Kafka Consumer Lag**
```bash
# 1. Scale up consumers
docker-compose up -d --scale im-service-region-b=3

# 2. Increase consumer threads
# Edit config: KAFKA_CONSUMER_THREADS=8

# 3. Check for slow message processing
docker logs im-service-region-b | grep "slow processing"

# 4. Restart consumers if stuck
docker restart im-service-region-b
```

**Scenario C: Database Replication Lag**
```bash
# 1. Check binlog position
docker exec mysql-region-a mysql -u root -ppassword -e \
  "SHOW MASTER STATUS"

# 2. Check replica status
docker exec mysql-region-b mysql -u root -ppassword -e \
  "SHOW SLAVE STATUS\G"

# 3. If replication stopped, restart it
docker exec mysql-region-b mysql -u root -ppassword -e \
  "STOP SLAVE; START SLAVE;"

# 4. If persistent, consider parallel replication
# Edit my.cnf: slave_parallel_workers=4
```

**Scenario D: Resource Exhaustion**
```bash
# 1. Increase resource limits
# Edit docker-compose.yml:
#   resources:
#     limits:
#       cpus: '2.0'
#       memory: 4G

# 2. Restart services
docker-compose restart im-service-region-a im-service-region-b

# 3. Monitor improvement
watch -n 1 'docker stats --no-stream im-service-region-a'
```

**Verification**:
```bash
# 1. Check latency improved
curl http://localhost:9090/api/v1/query?query=histogram_quantile\(0.99,cross_region_sync_latency_ms\)

# 2. Monitor for 10 minutes
# 3. Verify alert cleared in Alertmanager
```

---

### Issue 2: High Conflict Rate

**Symptoms**:
- Conflict rate > 0.1%
- Alert: `HighConflictRate`
- Frequent tiebreaker usage
- Users reporting inconsistent data

**Diagnosis Steps**:

1. **Check conflict metrics**:
```bash
# View conflict rate
curl -s http://localhost:9090/api/v1/query?query=rate\(cross_region_conflicts_total\[5m\]\) | jq

# View conflicts by type
curl -s http://localhost:9090/api/v1/query?query=sum\(cross_region_conflicts_total\)by\(conflict_type\) | jq
```

2. **Analyze conflict logs**:
```bash
# View recent conflicts
docker logs im-service-region-a | grep "conflict detected" | tail -20

# Extract conflict details
docker logs im-service-region-a | grep "conflict detected" | jq -r '.global_id, .conflict_type'
```

3. **Check HLC clock synchronization**:
```bash
# Check NTP sync status
docker exec im-service-region-a ntpq -p
docker exec im-service-region-b ntpq -p

# Check clock offset
docker exec im-service-region-a date +%s
docker exec im-service-region-b date +%s
# Difference should be < 1 second
```

4. **Check tiebreaker usage**:
```bash
# High tiebreaker usage indicates clock sync issues
curl -s http://localhost:9090/api/v1/query?query=rate\(conflict_tiebreaker_used_total\[5m\]\) | jq
```

**Resolution Steps**:

**Scenario A: Clock Synchronization Issue**
```bash
# 1. Force NTP sync
docker exec im-service-region-a ntpdate -u pool.ntp.org
docker exec im-service-region-b ntpdate -u pool.ntp.org

# 2. Restart services to reset HLC
docker restart im-service-region-a im-service-region-b

# 3. Monitor tiebreaker usage decrease
```

**Scenario B: Application Logic Issue**
```bash
# 1. Identify conflicting operations
docker logs im-service-region-a | grep "conflict detected" | \
  jq -r '.operation' | sort | uniq -c | sort -rn

# 2. Review application code for concurrent writes
# 3. Consider adding optimistic locking or versioning
# 4. Implement conflict-free replicated data types (CRDTs) where appropriate
```

**Scenario C: Network Partition Recovery**
```bash
# After network partition, conflicts are expected
# 1. Monitor conflict rate trend
# 2. Should decrease over time as data converges
# 3. If persistent, check for ongoing network issues
```

**Verification**:
```bash
# 1. Conflict rate should drop below 0.1%
# 2. Tiebreaker usage should be minimal
# 3. No user reports of data inconsistency
```

---

### Issue 3: Failover Not Triggering

**Symptoms**:
- Region down but traffic not switching
- Alert: `RegionDown` but no `FailoverEventDetected`
- Users unable to connect
- Manual failover required

**Diagnosis Steps**:

1. **Check health check status**:
```bash
# Check health endpoints
curl -v http://im-gateway-region-a:8080/health
curl -v http://im-gateway-region-b:8080/health

# Check health check configuration
docker exec im-gateway-region-a cat /etc/config/health-check.yaml
```

2. **Check geo-router status**:
```bash
# View router logs
docker logs im-gateway-region-a | grep "geo_router"

# Check routing decisions
docker logs im-gateway-region-a | grep "routing decision"
```

3. **Check etcd leader election**:
```bash
# Check etcd cluster status
docker exec etcd-region-a etcdctl endpoint status --cluster --write-out=table

# Check leader
docker exec etcd-region-a etcdctl endpoint status --write-out=json | \
  jq '.[] | select(.Status.leader == .Status.header.member_id)'
```

4. **Check DNS/load balancer**:
```bash
# Check DNS resolution
nslookup im-gateway.example.com

# Check load balancer health checks
# (Cloud provider specific)
```

**Resolution Steps**:

**Scenario A: Health Check Misconfiguration**
```bash
# 1. Verify health check endpoint
curl -v http://im-gateway-region-a:8080/health

# 2. Check health check interval and threshold
# Edit config: HEALTH_CHECK_INTERVAL=5s, FAILURE_THRESHOLD=3

# 3. Restart gateway
docker restart im-gateway-region-a
```

**Scenario B: Geo-Router Not Detecting Failure**
```bash
# 1. Check router configuration
docker exec im-gateway-region-a env | grep ROUTING

# 2. Enable debug logging
docker exec im-gateway-region-a kill -USR1 1  # Toggle debug

# 3. Manually trigger failover
curl -X POST http://im-gateway-region-a:8080/admin/failover \
  -H "Content-Type: application/json" \
  -d '{"target_region": "region-b"}'
```

**Scenario C: etcd Split-Brain**
```bash
# 1. Check etcd cluster health
docker exec etcd-region-a etcdctl endpoint health --cluster

# 2. If split-brain detected, force leader election
docker exec etcd-region-a etcdctl elect /im/coordination/leader region-a

# 3. Restart etcd cluster if necessary
docker-compose restart etcd-region-a etcd-region-b etcd-arbiter
```

**Verification**:
```bash
# 1. Simulate region failure
docker stop im-service-region-a im-gateway-region-a

# 2. Wait for failover (should be < 30 seconds)
# 3. Check traffic switched to region-b
curl http://im-gateway.example.com/stats | jq '.region'

# 4. Verify alert: FailoverEventDetected
# 5. Restart region-a
docker start im-service-region-a im-gateway-region-a
```

---

### Issue 4: Message Loss During Failover

**Symptoms**:
- Messages sent during failover not delivered
- Alert: `HighMessageLossRate`
- Users reporting missing messages
- Offline queue not capturing messages

**Diagnosis Steps**:

1. **Check message delivery metrics**:
```bash
# Check message loss rate
curl -s http://localhost:9090/api/v1/query?query=rate\(im_gateway_messages_failed_total\[5m\]\) | jq

# Check offline queue size
curl -s http://localhost:9090/api/v1/query?query=im_service_offline_queue_size | jq
```

2. **Check failover timing**:
```bash
# View failover events
docker logs im-gateway-region-a | grep "failover" | tail -10

# Check failover duration
docker logs im-gateway-region-a | grep "failover completed" | jq '.duration_ms'
```

3. **Check message acknowledgment**:
```bash
# Check ACK timeout rate
curl -s http://localhost:9090/api/v1/query?query=rate\(im_gateway_ack_timeout_total\[5m\]\) | jq

# View ACK timeout logs
docker logs im-gateway-region-a | grep "ack timeout"
```

4. **Check database consistency**:
```bash
# Compare message counts between regions
docker exec mysql-region-a mysql -u root -ppassword im_chat -e \
  "SELECT COUNT(*) FROM offline_messages WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)"

docker exec mysql-region-b mysql -u root -ppassword im_chat -e \
  "SELECT COUNT(*) FROM offline_messages WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)"
```

**Resolution Steps**:

**Scenario A: Failover Too Slow**
```bash
# 1. Reduce health check interval
# Edit config: HEALTH_CHECK_INTERVAL=3s (from 5s)

# 2. Reduce failure threshold
# Edit config: FAILURE_THRESHOLD=2 (from 3)

# 3. Enable pre-failover connection draining
# Edit config: ENABLE_CONNECTION_DRAINING=true

# 4. Restart services
docker-compose restart
```

**Scenario B: Offline Queue Not Working**
```bash
# 1. Check Kafka offline_msg topic
docker exec kafka-region-a kafka-topics.sh \
  --describe --topic offline_msg --bootstrap-server localhost:9092

# 2. Check offline worker status
docker logs offline-worker | grep "processing"

# 3. Check database connectivity
docker exec im-service-region-a mysql -h mysql-region-a -u im_service -ppassword -e "SELECT 1"

# 4. Restart offline worker
docker restart offline-worker
```

**Scenario C: Message Acknowledgment Issues**
```bash
# 1. Increase ACK timeout
# Edit config: MESSAGE_ACK_TIMEOUT=10s (from 5s)

# 2. Enable message retry
# Edit config: MESSAGE_RETRY_ENABLED=true, MAX_RETRIES=3

# 3. Implement idempotency keys
# Ensure messages have unique IDs for deduplication
```

**Verification**:
```bash
# 1. Simulate failover with active traffic
./scripts/load-test.sh &
docker stop im-service-region-a

# 2. Wait for failover
sleep 30

# 3. Check message loss rate
curl -s http://localhost:9090/api/v1/query?query=rate\(im_gateway_messages_failed_total\[5m\]\)

# 4. Should be < 0.01%
# 5. Restart region-a
docker start im-service-region-a
```

---

## Cross-Region Sync Issues

### Issue 5: Kafka Replication Lag

**Symptoms**:
- MirrorMaker lag increasing
- Messages not appearing in remote region
- Alert: `HighKafkaReplicationLag`

**Diagnosis**:
```bash
# Check MirrorMaker status
docker logs kafka-mirrormaker | tail -50

# Check replication lag
docker exec kafka-region-a kafka-consumer-groups.sh \
  --bootstrap-server kafka-region-b:9092 \
  --group mirrormaker-cluster --describe

# Check network bandwidth
docker stats kafka-mirrormaker --no-stream
```

**Resolution**:
```bash
# 1. Increase MirrorMaker threads
# Edit config: num.streams=4

# 2. Increase batch size
# Edit config: batch.size=32768

# 3. Enable compression
# Edit config: compression.type=lz4

# 4. Restart MirrorMaker
docker restart kafka-mirrormaker
```

---

### Issue 6: Database Replication Stopped

**Symptoms**:
- Slave_IO_Running: No or Slave_SQL_Running: No
- Data divergence between regions
- Alert: `MySQLReplicationStopped`

**Diagnosis**:
```bash
# Check replication status
docker exec mysql-region-b mysql -u root -ppassword -e "SHOW SLAVE STATUS\G"

# Check error log
docker logs mysql-region-b | grep -i error | tail -20

# Check binlog position
docker exec mysql-region-a mysql -u root -ppassword -e "SHOW MASTER STATUS"
```

**Resolution**:
```bash
# 1. If IO thread stopped
docker exec mysql-region-b mysql -u root -ppassword -e \
  "STOP SLAVE IO_THREAD; START SLAVE IO_THREAD;"

# 2. If SQL thread stopped (duplicate key error)
docker exec mysql-region-b mysql -u root -ppassword -e \
  "STOP SLAVE; SET GLOBAL SQL_SLAVE_SKIP_COUNTER = 1; START SLAVE;"

# 3. If replication completely broken, rebuild replica
# (See runbook: Rebuild MySQL Replica)

# 4. Verify replication resumed
docker exec mysql-region-b mysql -u root -ppassword -e "SHOW SLAVE STATUS\G" | \
  grep -E "Slave_IO_Running|Slave_SQL_Running"
```

---

## Conflict Resolution Issues

### Issue 7: Incorrect Conflict Resolution

**Symptoms**:
- Wrong version of data persisted
- Users reporting data overwrites
- Conflict logs show unexpected winners

**Diagnosis**:
```bash
# View conflict resolution logs
docker logs im-service-region-a | grep "conflict resolved" | jq

# Check HLC timestamps
docker logs im-service-region-a | grep "conflict resolved" | \
  jq -r '.local_hlc, .remote_hlc'

# Verify LWW logic
docker logs im-service-region-a | grep "conflict resolved" | \
  jq -r '.winner, .reason'
```

**Resolution**:
```bash
# 1. Verify HLC implementation
# Check: apps/im-service/hlc/hlc.go

# 2. Verify conflict resolver logic
# Check: apps/im-service/sync/conflict_resolver.go

# 3. If bug found, deploy fix
# 4. For affected data, manual reconciliation may be needed

# 5. Query affected records
docker exec mysql-region-a mysql -u root -ppassword im_chat -e \
  "SELECT * FROM offline_messages WHERE sync_status = 'conflict' LIMIT 10"
```

---

## Failover Issues

### Issue 8: Slow Failover (RTO > 30s)

**Diagnosis**:
```bash
# Measure failover time
time (docker stop im-service-region-a && \
      while ! curl -s http://im-gateway.example.com/health | grep -q region-b; do sleep 1; done)

# Check health check frequency
docker exec im-gateway-region-a env | grep HEALTH_CHECK

# Check DNS TTL
dig im-gateway.example.com | grep TTL
```

**Resolution**:
```bash
# 1. Reduce health check interval to 3s
# 2. Reduce failure threshold to 2
# 3. Reduce DNS TTL to 30s
# 4. Enable connection pre-warming
# 5. Implement graceful shutdown with connection draining
```

---

## Performance Degradation

### Issue 9: High CPU Usage

**Diagnosis**:
```bash
# Check CPU usage
docker stats --no-stream | grep im-service

# Profile CPU
docker exec im-service-region-a curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze profile
go tool pprof -http=:8080 cpu.prof
```

**Resolution**:
```bash
# 1. Identify hot paths in profile
# 2. Optimize code (caching, batching, etc.)
# 3. Scale horizontally
docker-compose up -d --scale im-service-region-a=3

# 4. Increase CPU limits
# Edit docker-compose.yml: cpus: '2.0'
```

---

## Data Consistency Issues

### Issue 10: Data Divergence Between Regions

**Diagnosis**:
```bash
# Compare record counts
docker exec mysql-region-a mysql -u root -ppassword im_chat -e \
  "SELECT COUNT(*), MAX(id) FROM offline_messages"

docker exec mysql-region-b mysql -u root -ppassword im_chat -e \
  "SELECT COUNT(*), MAX(id) FROM offline_messages"

# Check for missing records
# (Implement data reconciliation script)
```

**Resolution**:
```bash
# 1. Run data reconciliation
./scripts/reconcile-data.sh

# 2. Identify root cause (replication lag, conflicts, etc.)
# 3. Fix root cause
# 4. Re-sync missing data
```

---

## Network Issues

### Issue 11: Network Partition

**Symptoms**:
- Regions cannot communicate
- Sync latency infinite
- Split-brain risk

**Diagnosis**:
```bash
# Test connectivity
docker exec im-service-region-a ping -c 5 im-service-region-b

# Check firewall rules
iptables -L -n

# Check network policies (Kubernetes)
kubectl get networkpolicies
```

**Resolution**:
```bash
# 1. Identify network issue (firewall, routing, etc.)
# 2. Fix network connectivity
# 3. Verify etcd arbiter can reach both regions
# 4. Monitor for split-brain
# 5. After recovery, check for conflicts and reconcile data
```

---

## Diagnostic Tools

### Tool 1: Health Check Script

```bash
#!/bin/bash
# health-check.sh

echo "=== Multi-Region Health Check ==="

echo "1. Service Status:"
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "im-service|im-gateway"

echo "2. Cross-Region Connectivity:"
docker exec im-service-region-a ping -c 1 im-service-region-b > /dev/null && echo "✓ Region A -> B" || echo "✗ Region A -> B"
docker exec im-service-region-b ping -c 1 im-service-region-a > /dev/null && echo "✓ Region B -> A" || echo "✗ Region B -> A"

echo "3. Key Metrics:"
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile\(0.99,cross_region_sync_latency_ms\) | \
  jq -r '.data.result[0].value[1] // "N/A"' | xargs -I {} echo "Sync Latency P99: {} ms"

curl -s http://localhost:9090/api/v1/query?query=rate\(cross_region_conflicts_total\[5m\]\) | \
  jq -r '.data.result[0].value[1] // "0"' | xargs -I {} echo "Conflict Rate: {} /sec"

echo "4. Alerts:"
curl -s http://localhost:9093/api/v1/alerts | jq -r '.data[] | select(.state=="firing") | .labels.alertname' | \
  xargs -I {} echo "🔥 {}"

echo "=== Health Check Complete ==="
```

### Tool 2: Conflict Analysis Script

```bash
#!/bin/bash
# analyze-conflicts.sh

echo "=== Conflict Analysis ==="

echo "1. Conflict Rate (last 5 min):"
curl -s http://localhost:9090/api/v1/query?query=rate\(cross_region_conflicts_total\[5m\]\) | jq

echo "2. Conflicts by Type:"
curl -s http://localhost:9090/api/v1/query?query=sum\(cross_region_conflicts_total\)by\(conflict_type\) | jq

echo "3. Recent Conflict Logs:"
docker logs im-service-region-a | grep "conflict detected" | tail -10 | jq

echo "4. Tiebreaker Usage:"
curl -s http://localhost:9090/api/v1/query?query=rate\(conflict_tiebreaker_used_total\[5m\]\) | jq

echo "=== Analysis Complete ==="
```

### Tool 3: Failover Test Script

```bash
#!/bin/bash
# test-failover.sh

echo "=== Failover Test ==="

echo "1. Current active region:"
curl -s http://im-gateway.example.com/stats | jq -r '.region'

echo "2. Stopping Region A..."
docker stop im-service-region-a im-gateway-region-a

echo "3. Waiting for failover..."
start_time=$(date +%s)
while ! curl -s http://im-gateway.example.com/health | grep -q region-b; do
  sleep 1
done
end_time=$(date +%s)
failover_time=$((end_time - start_time))

echo "✓ Failover completed in ${failover_time} seconds"

if [ $failover_time -lt 30 ]; then
  echo "✓ RTO target met (< 30s)"
else
  echo "✗ RTO target missed (>= 30s)"
fi

echo "4. Restarting Region A..."
docker start im-service-region-a im-gateway-region-a

echo "=== Failover Test Complete ==="
```

---

## Escalation Procedures

### When to Escalate

| Situation | Escalate To | Timeframe |
|-----------|-------------|-----------|
| Cannot resolve P0 in 15 min | Senior SRE | Immediate |
| Data loss detected | Engineering Lead | Immediate |
| Multiple regions down | Incident Commander | Immediate |
| Cannot resolve P1 in 1 hour | Team Lead | Within 1 hour |
| Recurring issues | Architecture Team | Next business day |

### Escalation Contacts

- **Primary On-Call**: PagerDuty
- **Senior SRE**: #sre-escalation (Slack)
- **Engineering Lead**: engineering-lead@example.com
- **Incident Commander**: #incident-response (Slack)

---

## Post-Incident Checklist

After resolving an incident:

1. ✅ Document root cause
2. ✅ Update runbook if needed
3. ✅ Create Jira ticket for permanent fix
4. ✅ Update monitoring/alerting if gaps found
5. ✅ Schedule post-mortem (P0/P1 only)
6. ✅ Share learnings with team

---

**Related Documents**:
- [Capacity Planning Guide](./CAPACITY_PLANNING_GUIDE.md)
- [Performance Tuning Guide](./PERFORMANCE_TUNING_GUIDE.md)
- [Monitoring & Alerting Handbook](./MONITORING_ALERTING_HANDBOOK.md)
- [Architecture Overview](../architecture-overview.md)

**Last Updated**: 2024  
**Next Review**: Quarterly
