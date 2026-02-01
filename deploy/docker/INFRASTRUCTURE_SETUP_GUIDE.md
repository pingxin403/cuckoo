# Multi-Region Infrastructure Setup Guide

## Overview

This guide provides detailed instructions for setting up cross-region data synchronization infrastructure for the multi-region active-active architecture. These are operational tasks that should be performed by DevOps/SRE teams.

## Task 7: Cross-Region Data Synchronization

### 7.1 MySQL Cross-Region Replication

#### Overview
Configure MySQL master-slave replication between Region A and Region B for the `im_chat` database.

#### Prerequisites
- MySQL 8.0+ installed in both regions
- Network connectivity between regions
- Sufficient disk space for binary logs

#### Implementation Steps

**1. Configure Master (Region A)**

```sql
-- Enable binary logging
-- Edit /etc/mysql/my.cnf
[mysqld]
server-id = 1
log_bin = /var/log/mysql/mysql-bin.log
binlog_format = ROW
binlog_do_db = im_chat
expire_logs_days = 7
max_binlog_size = 100M

-- Restart MySQL
sudo systemctl restart mysql

-- Create replication user
CREATE USER 'repl_user'@'%' IDENTIFIED BY 'strong_password';
GRANT REPLICATION SLAVE ON *.* TO 'repl_user'@'%';
FLUSH PRIVILEGES;

-- Get master status
SHOW MASTER STATUS;
-- Note: File and Position values
```

**2. Configure Slave (Region B)**

```sql
-- Edit /etc/mysql/my.cnf
[mysqld]
server-id = 2
relay-log = /var/log/mysql/mysql-relay-bin
log_bin = /var/log/mysql/mysql-bin.log
binlog_format = ROW
read_only = 1

-- Restart MySQL
sudo systemctl restart mysql

-- Configure replication
CHANGE MASTER TO
  MASTER_HOST='region-a-mysql-host',
  MASTER_USER='repl_user',
  MASTER_PASSWORD='strong_password',
  MASTER_LOG_FILE='mysql-bin.000001',  -- From SHOW MASTER STATUS
  MASTER_LOG_POS=154;                   -- From SHOW MASTER STATUS

-- Start replication
START SLAVE;

-- Verify replication status
SHOW SLAVE STATUS\G
-- Check: Slave_IO_Running: Yes, Slave_SQL_Running: Yes
```

**3. Application Configuration**

Update `apps/im-service/storage/offline_store.go` to support read-write splitting:

```go
type OfflineStore struct {
    writeDB  *sql.DB  // Master (Region A)
    readDB   *sql.DB  // Slave (Region B) or Master
    regionID string
}

// Use writeDB for INSERT, UPDATE, DELETE
// Use readDB for SELECT queries
```

**4. Monitoring**

```bash
# Check replication lag
SELECT TIMESTAMPDIFF(SECOND, ts, NOW()) AS replication_lag 
FROM (SELECT MAX(timestamp) AS ts FROM im_chat.offline_messages) t;

# Monitor replication status
SHOW SLAVE STATUS\G | grep -E "Seconds_Behind_Master|Slave_IO_Running|Slave_SQL_Running"
```

#### Verification

```bash
# Insert test data in Region A
mysql -h region-a-mysql -u im_service -p im_chat -e \
  "INSERT INTO offline_messages (user_id, content) VALUES ('test', 'replication test');"

# Verify in Region B (should appear within 1-2 seconds)
mysql -h region-b-mysql -u im_service -p im_chat -e \
  "SELECT * FROM offline_messages WHERE user_id='test' ORDER BY created_at DESC LIMIT 1;"
```

#### Troubleshooting

**Replication Stopped**
```sql
-- Check error
SHOW SLAVE STATUS\G

-- Skip one error (if safe)
STOP SLAVE;
SET GLOBAL SQL_SLAVE_SKIP_COUNTER = 1;
START SLAVE;

-- Or reset and restart
STOP SLAVE;
RESET SLAVE;
-- Reconfigure with CHANGE MASTER TO
START SLAVE;
```

**High Replication Lag**
- Check network latency between regions
- Increase `slave_parallel_workers` for parallel replication
- Optimize slow queries on slave
- Consider semi-synchronous replication for critical data

---

### 7.2 Redis Cross-Region Synchronization

#### Overview
Configure Redis replication between Region A and Region B for session and cache data.

#### Implementation Options

**Option A: Redis Replication (Simple)**

```bash
# Region B (Slave) - redis.conf
replicaof region-a-redis-host 6379
masterauth strong_password
replica-read-only yes
```

**Option B: Redis Cluster with Cross-Region Replication**

```bash
# Create Redis cluster spanning both regions
redis-cli --cluster create \
  region-a-redis-1:6379 \
  region-a-redis-2:6379 \
  region-a-redis-3:6379 \
  region-b-redis-1:6379 \
  region-b-redis-2:6379 \
  region-b-redis-3:6379 \
  --cluster-replicas 1
```

**Option C: Redis Enterprise Active-Active (Recommended for Production)**

Use Redis Enterprise CRDT (Conflict-free Replicated Data Types) for true active-active replication.

#### Application Configuration

Update Redis connection to support multi-region:

```go
// apps/im-service/dedup/dedup.go
type DedupService struct {
    localRedis  *redis.Client  // Region-local Redis
    remoteRedis *redis.Client  // Remote region Redis (optional)
    regionID    string
}

// Check both local and remote for deduplication
func (d *DedupService) IsDuplicate(msgID string) bool {
    // Check local first (fast)
    if d.localRedis.Exists(ctx, msgID).Val() > 0 {
        return true
    }
    
    // Check remote if configured (slower)
    if d.remoteRedis != nil {
        if d.remoteRedis.Exists(ctx, msgID).Val() > 0 {
            return true
        }
    }
    
    return false
}
```

#### Verification

```bash
# Set key in Region A
redis-cli -h region-a-redis SET test:key "value from region-a"

# Verify in Region B (should appear within 100ms)
redis-cli -h region-b-redis GET test:key
```

#### Monitoring

```bash
# Check replication status
redis-cli -h region-b-redis INFO replication

# Monitor replication lag
redis-cli -h region-b-redis INFO replication | grep master_repl_offset
```

---

### 7.3 Kafka Cross-Region Replication

#### Overview
Configure Kafka MirrorMaker 2.0 for cross-cluster replication of the `offline_msg` topic.

#### Prerequisites
- Kafka 2.4+ in both regions
- Network connectivity between Kafka clusters
- Sufficient disk space for replicated topics

#### Implementation Steps

**1. Install MirrorMaker 2.0**

```bash
# MirrorMaker 2.0 is included in Kafka distribution
cd $KAFKA_HOME
```

**2. Configure MirrorMaker 2.0**

Create `mm2.properties`:

```properties
# Cluster definitions
clusters = region-a, region-b

# Region A cluster
region-a.bootstrap.servers = region-a-kafka-1:9092,region-a-kafka-2:9092,region-a-kafka-3:9092

# Region B cluster
region-b.bootstrap.servers = region-b-kafka-1:9092,region-b-kafka-2:9092,region-b-kafka-3:9092

# Replication flows
region-a->region-b.enabled = true
region-b->region-a.enabled = true

# Topic whitelist
region-a->region-b.topics = offline_msg, read_receipt_events
region-b->region-a.topics = offline_msg, read_receipt_events

# Replication settings
replication.factor = 3
sync.topic.configs.enabled = true
refresh.topics.enabled = true
refresh.topics.interval.seconds = 60

# Consumer group replication
sync.group.offsets.enabled = true
sync.group.offsets.interval.seconds = 60

# Checkpoints
checkpoints.topic.replication.factor = 3
offset-syncs.topic.replication.factor = 3
```

**3. Start MirrorMaker 2.0**

```bash
# Start MirrorMaker 2.0
$KAFKA_HOME/bin/connect-mirror-maker.sh mm2.properties

# Or use systemd
sudo systemctl start kafka-mirror-maker
```

**4. Application Configuration**

Update Kafka consumer/producer to handle replicated topics:

```go
// apps/im-service/worker/offline_worker.go
type OfflineWorker struct {
    localConsumer  *kafka.Consumer   // Consume from local cluster
    remoteProducer *kafka.Producer   // Produce to remote cluster (optional)
    regionID       string
}

// Process message and optionally replicate
func (w *OfflineWorker) processMessage(msg *kafka.Message) error {
    // Process locally
    err := w.processOfflineMessage(msg)
    if err != nil {
        return err
    }
    
    // Replicate to remote region (if not already replicated by MirrorMaker)
    if w.remoteProducer != nil && !w.isReplicatedMessage(msg) {
        return w.replicateToRemote(msg)
    }
    
    return nil
}
```

#### Verification

```bash
# Produce message in Region A
kafka-console-producer --bootstrap-server region-a-kafka:9092 --topic offline_msg
> {"msg_id": "test-123", "content": "test message"}

# Verify in Region B (should appear within 500ms)
kafka-console-consumer --bootstrap-server region-b-kafka:9092 \
  --topic region-a.offline_msg --from-beginning --max-messages 1
```

#### Monitoring

```bash
# Check MirrorMaker lag
kafka-consumer-groups --bootstrap-server region-b-kafka:9092 --describe --group mirrormaker2-cluster

# Monitor replication metrics
curl http://mirrormaker-host:8083/connectors/MirrorSourceConnector/status
```

---

### 7.4 HLC Integration (✅ Completed)

HLC has been successfully integrated into the sequence generator. See:
- `apps/im-service/sequence/sequence_generator.go`
- `apps/im-service/hlc/hlc.go`

**Key Features:**
- Global ID generation: `{region_id}-{hlc_timestamp}-{logical_counter}`
- Remote clock synchronization: `UpdateHLCFromRemote()`
- Causal ordering support

---

## Task 8: Failover and Traffic Management

### 8.1 Multi-Region Health Checks

#### Implementation

Health check endpoints have been extended to support multi-region:

**IM Service Health Check**
```bash
# Local health
curl http://localhost:8184/health

# Cross-region health
curl http://localhost:8184/health/cross-region
```

**Response:**
```json
{
  "overall": "healthy",
  "region": "region-a",
  "local_health": {
    "mysql": "healthy",
    "redis": "healthy",
    "kafka": "healthy",
    "etcd": "healthy"
  },
  "cross_region": {
    "region-b": "healthy"
  },
  "timestamp": 1706180400000
}
```

#### etcd-based Distributed Health Checks

Services register their health status in etcd:

```bash
# Check service health in etcd
etcdctl get /im/health/region-a/im-service --prefix

# Watch for health changes
etcdctl watch /im/health/ --prefix
```

#### Automatic Failover

Implement automatic failover based on health checks:

```go
// Pseudo-code for automatic failover
func (hc *HealthChecker) MonitorAndFailover() {
    for {
        health := hc.CheckCrossRegionHealth()
        
        if !health.IsHealthy("region-a") {
            // Trigger failover to region-b
            hc.trafficManager.FailoverTo("region-b")
            hc.notifyOps("Failover triggered: region-a unhealthy")
        }
        
        time.Sleep(30 * time.Second)
    }
}
```

---

### 8.2 DNS-based Traffic Management

#### Overview
Use DNS geo-routing to direct users to the nearest healthy region.

#### Implementation Options

**Option A: AWS Route 53 (Recommended)**

```hcl
# Terraform configuration
resource "aws_route53_record" "im_service" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "im-api.example.com"
  type    = "A"

  # Region A
  set_identifier = "region-a"
  geolocation_routing_policy {
    continent = "AS"
    country   = "CN"
    subdivision = "BJ"  # Beijing
  }
  alias {
    name                   = aws_lb.region_a.dns_name
    zone_id                = aws_lb.region_a.zone_id
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "im_service_region_b" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "im-api.example.com"
  type    = "A"

  # Region B
  set_identifier = "region-b"
  geolocation_routing_policy {
    continent = "AS"
    country   = "CN"
    subdivision = "SH"  # Shanghai
  }
  alias {
    name                   = aws_lb.region_b.dns_name
    zone_id                = aws_lb.region_b.zone_id
    evaluate_target_health = true
  }
}

# Health checks
resource "aws_route53_health_check" "region_a" {
  fqdn              = "region-a-health.example.com"
  port              = 80
  type              = "HTTPS"
  resource_path     = "/health"
  failure_threshold = 3
  request_interval  = 30
}
```

**Option B: Envoy-based Routing**

Update `deploy/docker/envoy-config.yaml`:

```yaml
static_resources:
  clusters:
  - name: im_service_region_a
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    health_checks:
    - timeout: 1s
      interval: 10s
      unhealthy_threshold: 3
      healthy_threshold: 2
      http_health_check:
        path: "/health"
    load_assignment:
      cluster_name: im_service_region_a
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: im-service-region-a
                port_value: 8080

  - name: im_service_region_b
    # Similar configuration for region-b

  listeners:
  - name: listener_0
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 8080
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          route_config:
            virtual_hosts:
            - name: backend
              domains: ["*"]
              routes:
              - match:
                  prefix: "/"
                  headers:
                  - name: "x-region"
                    exact_match: "region-a"
                route:
                  cluster: im_service_region_a
              - match:
                  prefix: "/"
                  headers:
                  - name: "x-region"
                    exact_match: "region-b"
                route:
                  cluster: im_service_region_b
              - match:
                  prefix: "/"
                route:
                  weighted_clusters:
                    clusters:
                    - name: im_service_region_a
                      weight: 50
                    - name: im_service_region_b
                      weight: 50
```

#### Manual Traffic Switching

Use the traffic CLI tool:

```bash
# Gradual migration (10% to region-b)
./apps/im-service/cmd/traffic-cli/traffic-cli switch \
  --from region-a --to region-b --percentage 10

# Full failover
./apps/im-service/cmd/traffic-cli/traffic-cli switch \
  --from region-a --to region-b --percentage 100

# Rollback
./apps/im-service/cmd/traffic-cli/traffic-cli switch \
  --from region-b --to region-a --percentage 100
```

---

### 8.3 WebSocket Failover Support

#### Implementation

The Gateway Service already supports failover through the geo router:

**Reconnect Message:**
```json
{
  "type": "reconnect",
  "endpoint": "wss://region-b.example.com:8080/ws",
  "reason": "region_failover",
  "timestamp": 1706180400000
}
```

**Client Handling:**
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'reconnect') {
    // Close current connection
    ws.close();
    
    // Reconnect to new endpoint
    connectToEndpoint(message.endpoint);
  }
};
```

**Session State Migration:**

Session state is stored in etcd, so it's automatically available in the new region:

```go
// Gateway service automatically loads session from etcd
func (gs *GatewayService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Authenticate and get user_id
    userID := authenticateUser(r)
    
    // Load session from etcd (available in all regions)
    session := gs.registry.GetUserSession(userID)
    
    // Continue with WebSocket handling
    // ...
}
```

---

## Task 9: Multi-Region Observability

### 9.1 Observability Library Extension

#### Implementation

The observability library has been extended to support region labels:

```go
// libs/observability/metrics.go
type MultiRegionMetrics struct {
    obs      observability.Observability
    regionID string
}

func (m *MultiRegionMetrics) RecordSyncLatency(targetRegion string, latencyMs float64) {
    m.obs.Metrics().RecordHistogram("cross_region_sync_latency_ms", latencyMs, map[string]string{
        "source_region": m.regionID,
        "target_region": targetRegion,
    })
}

func (m *MultiRegionMetrics) RecordConflictEvent(conflictType string) {
    m.obs.Metrics().IncrementCounter("cross_region_conflicts_total", map[string]string{
        "region":        m.regionID,
        "conflict_type": conflictType,
    })
}
```

#### Prometheus Metrics

Multi-region metrics are exposed on the `/metrics` endpoint:

```prometheus
# Cross-region sync latency
cross_region_sync_latency_ms{source_region="region-a",target_region="region-b"} 45.2

# Conflict events
cross_region_conflicts_total{region="region-a",conflict_type="message"} 12

# Failover events
failover_events_total{from_region="region-a",to_region="region-b"} 1

# Region health
region_health_status{region="region-a"} 1
region_health_status{region="region-b"} 1
```

---

### 9.2 Grafana Multi-Region Dashboards

#### Implementation

Grafana dashboards have been created for multi-region monitoring:

**Dashboard Location:** `docs/multi-region-demo/monitoring-dashboard.md`

**Key Panels:**
1. Cross-Region Sync Latency (P50, P95, P99)
2. Conflict Rate by Region
3. Failover Events Timeline
4. Region Health Status
5. Message Throughput by Region
6. HLC Clock Drift

#### Import Dashboard

```bash
# Import Grafana dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @docs/multi-region-demo/grafana-dashboard.json
```

#### Prometheus Alert Rules

Alert rules have been configured in `deploy/docker/prometheus-alerts.yml`:

```yaml
groups:
  - name: multi_region_alerts
    rules:
      - alert: HighCrossRegionSyncLatency
        expr: histogram_quantile(0.99, cross_region_sync_latency_ms) > 500
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cross-region sync latency too high"
          
      - alert: CrossRegionConflictRateHigh
        expr: rate(cross_region_conflicts_total[5m]) > 0.001
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Cross-region conflict rate > 0.1%"
          
      - alert: RegionUnhealthy
        expr: region_health_status == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Region {{ $labels.region }} is unhealthy"
```

---

### 9.3 Data Reconciliation

#### Overview

Data reconciliation ensures consistency between regions by periodically comparing and synchronizing data.

#### Implementation Strategy

**Option A: Merkle Tree-based Reconciliation**

```go
// reconcile/merkle_tree.go
type MerkleTree struct {
    root     *MerkleNode
    regionID string
}

func (mt *MerkleTree) BuildTree(messages []Message) {
    // Build Merkle tree from message hashes
    // ...
}

func (mt *MerkleTree) Compare(remoteTree *MerkleTree) []string {
    // Compare trees and return list of divergent message IDs
    // ...
}

func (mt *MerkleTree) Reconcile(divergentIDs []string) error {
    // Fetch and sync divergent messages
    // ...
}
```

**Option B: Timestamp-based Reconciliation**

```go
// reconcile/timestamp_reconciler.go
type TimestampReconciler struct {
    localDB  *sql.DB
    remoteDB *sql.DB
}

func (tr *TimestampReconciler) Reconcile(since time.Time) error {
    // Compare messages modified since timestamp
    localMessages := tr.getMessagesSince(tr.localDB, since)
    remoteMessages := tr.getMessagesSince(tr.remoteDB, since)
    
    // Find differences
    diff := tr.computeDiff(localMessages, remoteMessages)
    
    // Sync differences
    return tr.syncDifferences(diff)
}
```

#### Scheduled Reconciliation

```bash
# Cron job for hourly reconciliation
0 * * * * /usr/local/bin/reconcile-data --region-a region-a-db --region-b region-b-db
```

#### Monitoring

```prometheus
# Reconciliation metrics
data_reconciliation_runs_total{region="region-a"} 24
data_reconciliation_differences_found{region="region-a"} 5
data_reconciliation_sync_duration_seconds{region="region-a"} 12.5
```

---

## Summary

### Completed Tasks

✅ **Task 7.4**: HLC integration into sequence generator  
✅ **Task 8.1**: Multi-region health checks (code implemented)  
✅ **Task 8.2**: Traffic management CLI tool (implemented)  
✅ **Task 8.3**: WebSocket failover support (implemented)  
✅ **Task 9.1**: Observability library extension (implemented)  
✅ **Task 9.2**: Grafana dashboards (documented)

### Infrastructure Tasks (Operational)

📋 **Task 7.1**: MySQL replication (configuration guide provided)  
📋 **Task 7.2**: Redis replication (configuration guide provided)  
📋 **Task 7.3**: Kafka MirrorMaker (configuration guide provided)  
📋 **Task 9.3**: Data reconciliation (implementation strategy provided)

### Next Steps

1. **Deploy Infrastructure**: Follow the configuration guides above to set up MySQL, Redis, and Kafka replication in your target environment
2. **Test Failover**: Use the chaos engineering scripts to test failover scenarios
3. **Monitor Metrics**: Set up Grafana dashboards and Prometheus alerts
4. **Tune Performance**: Adjust replication intervals and batch sizes based on observed metrics
5. **Document Runbooks**: Create operational runbooks for common scenarios (failover, scaling, troubleshooting)

### Production Checklist

- [ ] MySQL master-slave replication configured and tested
- [ ] Redis replication or cluster configured
- [ ] Kafka MirrorMaker 2.0 deployed and monitored
- [ ] DNS geo-routing configured (Route 53 or equivalent)
- [ ] Health checks configured with appropriate thresholds
- [ ] Grafana dashboards imported and customized
- [ ] Prometheus alerts configured and tested
- [ ] Runbooks created for operational procedures
- [ ] Load testing performed across regions
- [ ] Disaster recovery procedures documented and tested

## References

- [Multi-Region Design Document](../../.kiro/specs/multi-region-active-active/design.md)
- [Multi-Region Requirements](../../.kiro/specs/multi-region-active-active/requirements.md)
- [Docker Compose Multi-Region Setup](./README.multi-region.md)
- [Traffic CLI Tool](../../apps/im-service/cmd/traffic-cli/README.md)
- [Monitoring Dashboard](../../docs/multi-region-demo/monitoring-dashboard.md)
