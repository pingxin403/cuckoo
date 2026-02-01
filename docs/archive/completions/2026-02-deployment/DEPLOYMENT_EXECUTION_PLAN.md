# Multi-Region Deployment Execution Plan

## Overview

This document provides a step-by-step execution plan for deploying the multi-region active-active architecture. Follow these steps in order to set up a complete multi-region environment.

## Prerequisites

- Docker and Docker Compose installed
- Sufficient system resources (16GB RAM, 50GB disk space recommended)
- Network connectivity between regions (if deploying to actual geographic locations)
- Access to DNS management (for production deployments)

## Deployment Phases

### Phase 1: Local Development Environment (Docker Compose)

**Estimated Time**: 30 minutes

This phase sets up a complete multi-region environment on a single machine using Docker Compose.

#### Step 1.1: Start Infrastructure Services

```bash
# Navigate to deployment directory
cd deploy/docker

# Start shared infrastructure (MySQL, Redis, Kafka, etcd)
docker compose -f docker-compose.infra.yml up -d

# Wait for services to be ready (2-3 minutes)
echo "Waiting for infrastructure to be ready..."
sleep 120

# Verify infrastructure health
docker compose -f docker-compose.infra.yml ps
```

**Expected Output**: All services should show "Up" status

#### Step 1.2: Start Multi-Region Services

```bash
# Start Region A and Region B services
./start-multi-region.sh start

# Verify services are running
./start-multi-region.sh test
```

**Expected Output**: 
```
✅ Region A IM Service: healthy
✅ Region A Gateway Service: healthy
✅ Region B IM Service: healthy
✅ Region B Gateway Service: healthy
```

#### Step 1.3: Verify Cross-Region Communication

```bash
# Test Region A to Region B connectivity
docker exec im-service-region-a ping -c 3 im-service-region-b

# Test Region B to Region A connectivity
docker exec im-service-region-b ping -c 3 im-service-region-a

# Check service discovery in etcd
docker exec etcd etcdctl get /im/services/ --prefix
```

**Expected Output**: Successful ping responses and service registrations in etcd

---

### Phase 2: Configure Data Replication (Simulated)

**Estimated Time**: 15 minutes

In the Docker Compose environment, we simulate replication using shared infrastructure. For production, follow the INFRASTRUCTURE_SETUP_GUIDE.md.

#### Step 2.1: Verify MySQL Connectivity

```bash
# Test MySQL connection from Region A
docker exec im-service-region-a nc -zv mysql 3306

# Test MySQL connection from Region B
docker exec im-service-region-b nc -zv mysql 3306

# Verify database exists
docker exec mysql mysql -uim_service -pim_service_password -e "SHOW DATABASES LIKE 'im_chat';"
```

#### Step 2.2: Verify Redis Connectivity

```bash
# Test Redis from Region A (DB 2)
docker exec im-service-region-a redis-cli -h redis PING

# Test Redis from Region B (DB 3)
docker exec im-service-region-b redis-cli -h redis PING

# Verify separate DBs
docker exec redis redis-cli INFO keyspace
```

#### Step 2.3: Verify Kafka Connectivity

```bash
# List Kafka topics
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Verify offline_msg topic exists
docker exec kafka kafka-topics --describe --topic offline_msg --bootstrap-server localhost:9092

# Check consumer groups
docker exec kafka kafka-consumer-groups --list --bootstrap-server localhost:9092
```

**Expected Output**: Both region consumer groups should be listed:
- `im-service-region-a-offline-workers`
- `im-service-region-b-offline-workers`

---

### Phase 3: Configure Routing and Load Balancing

**Estimated Time**: 10 minutes

#### Step 3.1: Verify Envoy Configuration (if using)

```bash
# Check if Envoy is running
docker compose -f docker-compose.services.yml ps envoy

# Test Envoy admin interface
curl http://localhost:9901/stats

# Verify cluster health
curl http://localhost:9901/clusters
```

#### Step 3.2: Test Geo-Routing

```bash
# Test routing to Region A (via header)
curl -H "X-Target-Region: region-a" http://localhost:8184/health

# Test routing to Region B (via header)
curl -H "X-Target-Region: region-b" http://localhost:8284/health

# Test default routing (should use Region A)
curl http://localhost:8184/health
```

#### Step 3.3: Test Traffic Switching CLI

```bash
# Build traffic CLI tool
cd apps/im-service/cmd/traffic-cli
go build -o traffic-cli .

# Test traffic status
./traffic-cli status

# Simulate traffic switch (dry-run)
./traffic-cli switch --from region-a --to region-b --percentage 10 --dry-run
```

---

### Phase 4: Import Monitoring and Observability

**Estimated Time**: 20 minutes

#### Step 4.1: Start Observability Stack

```bash
# Start Prometheus, Grafana, and related services
cd deploy/docker
docker compose -f docker-compose.observability.yml up -d

# Wait for services to start
sleep 30

# Verify Prometheus is running
curl http://localhost:9090/-/healthy

# Verify Grafana is running
curl http://localhost:3000/api/health
```

#### Step 4.2: Configure Prometheus Targets

```bash
# Verify Prometheus is scraping multi-region services
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job | contains("im-service"))'
```

**Expected Targets**:
- `im-service-region-a:9090`
- `im-service-region-b:9090`
- `im-gateway-service-region-a:9090`
- `im-gateway-service-region-b:9090`

#### Step 4.3: Import Grafana Dashboards

```bash
# Login to Grafana (default: admin/admin)
# URL: http://localhost:3000

# Import dashboard via API
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @../../docs/multi-region-demo/grafana-dashboard.json

# Or manually:
# 1. Open http://localhost:3000
# 2. Go to Dashboards > Import
# 3. Upload docs/multi-region-demo/grafana-dashboard.json
```

#### Step 4.4: Verify Metrics Collection

```bash
# Check multi-region metrics in Prometheus
curl 'http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms' | jq

# Check HLC metrics
curl 'http://localhost:9090/api/v1/query?query=hlc_physical_time' | jq

# Check conflict metrics
curl 'http://localhost:9090/api/v1/query?query=cross_region_conflicts_total' | jq
```

#### Step 4.5: Configure Alertmanager

```bash
# Verify Alertmanager is running
curl http://localhost:9093/-/healthy

# Check alert rules
curl http://localhost:9090/api/v1/rules | jq '.data.groups[] | select(.name == "multi_region_alerts")'

# Test alert (trigger high latency)
# This will be done in chaos testing phase
```

---

### Phase 5: Execute Chaos Engineering Tests

**Estimated Time**: 45 minutes

#### Step 5.1: Run Basic Functionality Test

```bash
cd deploy/mvp
./scripts/chaos-test.sh basic
```

**Expected Output**:
```
✅ Message sent to Region A
✅ Message sent to Region B
✅ Messages synchronized across regions
```

#### Step 5.2: Run Network Latency Test

```bash
./scripts/chaos-test.sh latency
```

**Expected Output**:
```
📡 Injecting network latency: 200ms ± 50ms
✅ Message delivered with high latency
✅ Sync latency within acceptable range
```

#### Step 5.3: Run Network Partition Test

```bash
./scripts/chaos-test.sh partition
```

**Expected Output**:
```
🚫 Network partition activated
✅ Messages queued during partition
✅ Messages synchronized after recovery
```

#### Step 5.4: Run Region Failover Test

```bash
# Test Region A failover
./scripts/chaos-test.sh failover-a

# Wait for recovery
sleep 60

# Test Region B failover
./scripts/chaos-test.sh failover-b
```

**Expected Output**:
```
💥 Region A stopped
✅ Region B continues to operate
✅ Messages delivered via Region B
🔧 Region A restored
✅ Messages synchronized after recovery
```

#### Step 5.5: Run Split-Brain Prevention Test

```bash
./scripts/chaos-test.sh split-brain
```

**Expected Output**:
```
🏛️ Arbiter status: healthy
🚫 Network partition activated
✅ Minority region enters read-only mode
✅ Majority region continues to operate
```

#### Step 5.6: Monitor Metrics During Tests

```bash
# Open Grafana dashboard
open http://localhost:3000

# Watch metrics in real-time
./scripts/chaos-test.sh monitor
```

**Key Metrics to Watch**:
- Cross-region sync latency (should be < 500ms)
- Conflict rate (should be < 0.1%)
- Failover events (should be logged)
- Message throughput (should remain stable)

---

### Phase 6: Performance Tuning

**Estimated Time**: 30 minutes

#### Step 6.1: Baseline Performance Measurement

```bash
# Run end-to-end tests to establish baseline
cd tests/e2e/multi-region
./run-e2e-tests.sh

# Record baseline metrics
echo "Baseline Metrics:" > performance-baseline.txt
curl 'http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms' >> performance-baseline.txt
```

#### Step 6.2: Tune Kafka Consumer Settings

```bash
# Edit docker-compose.services.yml
# Adjust these environment variables for im-service:

# BATCH_SIZE=100          # Increase to 200 for higher throughput
# BATCH_TIMEOUT=5s        # Decrease to 3s for lower latency
# MAX_RETRIES=5           # Keep at 5 for reliability

# Restart services to apply changes
docker compose -f docker-compose.services.yml restart im-service-region-a im-service-region-b
```

#### Step 6.3: Tune Database Connection Pool

```bash
# Edit docker-compose.services.yml
# Adjust these environment variables:

# DB_MAX_OPEN_CONNS=25    # Increase to 50 for higher concurrency
# DB_MAX_IDLE_CONNS=5     # Increase to 10 to reduce connection overhead

# Restart services
docker compose -f docker-compose.services.yml restart im-service-region-a im-service-region-b
```

#### Step 6.4: Tune Redis Settings

```bash
# Edit docker-compose.infra.yml
# Add Redis configuration:

# redis:
#   command: redis-server --maxmemory 2gb --maxmemory-policy allkeys-lru

# Restart Redis
docker compose -f docker-compose.infra.yml restart redis
```

#### Step 6.5: Measure Performance Improvement

```bash
# Run tests again
cd tests/e2e/multi-region
./run-e2e-tests.sh

# Compare with baseline
echo "After Tuning:" > performance-tuned.txt
curl 'http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms' >> performance-tuned.txt

# Calculate improvement
diff performance-baseline.txt performance-tuned.txt
```

#### Step 6.6: Document Optimal Settings

```bash
# Create tuning report
cat > PERFORMANCE_TUNING_RESULTS.md << 'EOF'
# Performance Tuning Results

## Baseline Metrics
- Sync Latency P99: XXXms
- Message Throughput: XXX msg/s
- Conflict Rate: X.XX%

## Tuned Settings
- Kafka Batch Size: 200
- Kafka Batch Timeout: 3s
- DB Max Open Conns: 50
- DB Max Idle Conns: 10
- Redis Max Memory: 2GB

## Improved Metrics
- Sync Latency P99: XXXms (XX% improvement)
- Message Throughput: XXX msg/s (XX% improvement)
- Conflict Rate: X.XX% (no change)

## Recommendations
- Monitor memory usage with increased connection pool
- Consider horizontal scaling if throughput > 10K msg/s
- Review batch size if latency requirements change
EOF
```

---

## Verification Checklist

Use this checklist to verify successful deployment:

### Infrastructure
- [ ] MySQL is running and accessible from both regions
- [ ] Redis is running with separate DBs for each region
- [ ] Kafka is running with offline_msg topic created
- [ ] etcd is running and services are registered
- [ ] Network connectivity between regions is working

### Services
- [ ] IM Service Region A is healthy
- [ ] IM Service Region B is healthy
- [ ] IM Gateway Region A is healthy
- [ ] IM Gateway Region B is healthy
- [ ] Auth Service is healthy
- [ ] User Service is healthy

### Data Replication
- [ ] Messages written to Region A appear in Region B
- [ ] Messages written to Region B appear in Region A
- [ ] Replication latency is < 500ms
- [ ] No data loss during replication

### Routing and Load Balancing
- [ ] Geo-routing directs requests to correct region
- [ ] Traffic switching CLI works correctly
- [ ] Health checks detect unhealthy regions
- [ ] Automatic failover works (if enabled)

### Monitoring and Observability
- [ ] Prometheus is scraping all services
- [ ] Grafana dashboards are imported and displaying data
- [ ] Alertmanager is configured and receiving alerts
- [ ] Multi-region metrics are being collected

### Chaos Engineering
- [ ] Basic functionality test passes
- [ ] Network latency test passes
- [ ] Network partition test passes
- [ ] Region failover test passes
- [ ] Split-brain prevention test passes

### Performance
- [ ] Sync latency P99 < 500ms
- [ ] Conflict rate < 0.1%
- [ ] Message throughput meets requirements
- [ ] Resource utilization is acceptable

---

## Troubleshooting

### Services Won't Start

**Problem**: Services fail to start or crash immediately

**Solution**:
```bash
# Check logs
docker compose -f docker-compose.services.yml logs im-service-region-a

# Common issues:
# 1. Infrastructure not ready - wait longer or check infra health
# 2. Port conflicts - check if ports are already in use
# 3. Resource constraints - increase Docker memory/CPU limits
```

### Cross-Region Communication Fails

**Problem**: Services can't communicate across regions

**Solution**:
```bash
# Check network connectivity
docker network inspect monorepo-network
docker network inspect region-a-network
docker network inspect region-b-network

# Verify services are on correct networks
docker inspect im-service-region-a | jq '.[0].NetworkSettings.Networks'

# Test connectivity
docker exec im-service-region-a ping im-service-region-b
```

### Metrics Not Appearing

**Problem**: Grafana dashboards show no data

**Solution**:
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Verify services are exposing metrics
curl http://localhost:8184/metrics
curl http://localhost:8284/metrics

# Check Prometheus configuration
docker exec prometheus cat /etc/prometheus/prometheus.yml
```

### High Replication Latency

**Problem**: Sync latency > 500ms

**Solution**:
```bash
# Check network latency
docker exec im-service-region-a ping -c 10 im-service-region-b

# Check Kafka lag
docker exec kafka kafka-consumer-groups --describe \
  --group im-service-region-a-offline-workers \
  --bootstrap-server localhost:9092

# Tune batch settings (see Phase 6)
```

---

## Next Steps

After completing this deployment:

1. **Run Load Tests**: Use tools like k6 or JMeter to test under load
2. **Security Hardening**: Enable TLS, configure firewalls, rotate credentials
3. **Backup and Recovery**: Set up automated backups and test recovery procedures
4. **Documentation**: Update runbooks with environment-specific details
5. **Training**: Train operations team on monitoring and troubleshooting

---

## Production Deployment Differences

When deploying to production, consider these differences:

### Infrastructure
- Use managed services (RDS, ElastiCache, MSK) instead of Docker containers
- Deploy to actual geographic regions (e.g., us-east-1, us-west-2)
- Use VPC peering or Transit Gateway for cross-region connectivity
- Enable encryption at rest and in transit

### DNS and Load Balancing
- Use Route 53 with geolocation routing policies
- Configure health checks with appropriate thresholds
- Set up CloudFront or similar CDN for static assets
- Enable DDoS protection

### Monitoring
- Use CloudWatch or Datadog for centralized monitoring
- Set up PagerDuty or similar for on-call alerting
- Enable audit logging for compliance
- Configure log retention policies

### Security
- Use AWS Secrets Manager or HashiCorp Vault for credentials
- Enable IAM roles for service authentication
- Configure security groups and NACLs
- Enable AWS GuardDuty for threat detection

### Scaling
- Use Auto Scaling Groups for horizontal scaling
- Configure appropriate instance types based on workload
- Set up read replicas for database scaling
- Use Kafka partitioning for message queue scaling

---

## References

- [Infrastructure Setup Guide](./INFRASTRUCTURE_SETUP_GUIDE.md)
- [Multi-Region Deployment Guide](./MULTI_REGION_DEPLOYMENT.md)
- [Performance Tuning Guide](../../docs/multi-region-demo/operations/PERFORMANCE_TUNING_GUIDE.md)
- [Troubleshooting Handbook](../../docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md)
- [Chaos Test Scripts](../mvp/scripts/chaos-test.sh)
