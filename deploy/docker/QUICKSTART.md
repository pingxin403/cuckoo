# Multi-Region Deployment Quick Start

## 🚀 One-Command Deployment

Deploy the complete multi-region environment with a single command:

```bash
cd deploy/docker
./deploy-multi-region.sh deploy
```

This will:
1. ✅ Check prerequisites (Docker, memory, etc.)
2. ✅ Deploy infrastructure (MySQL, Redis, Kafka, etcd)
3. ✅ Deploy multi-region services (Region A & B)
4. ✅ Verify cross-region communication
5. ✅ Deploy observability stack (Prometheus, Grafana)
6. ✅ Run basic health tests

**Estimated Time**: 5-10 minutes

---

## 📋 Prerequisites

- Docker 20.10+ installed
- Docker Compose V2 installed
- 16GB RAM available (minimum 8GB)
- 50GB disk space
- Ports available: 3000, 3307, 6380, 8080-8284, 9090-9297

---

## 🎯 Quick Commands

### Deploy Everything
```bash
./deploy-multi-region.sh deploy
```

### Verify Deployment
```bash
./deploy-multi-region.sh verify
```

### Show Summary
```bash
./deploy-multi-region.sh summary
```

### Cleanup
```bash
./deploy-multi-region.sh cleanup
```

---

## 🔍 Manual Step-by-Step (Alternative)

If you prefer manual control:

### Step 1: Start Infrastructure
```bash
docker compose -f docker-compose.infra.yml up -d
sleep 120  # Wait for services to be ready
```

### Step 2: Start Multi-Region Services
```bash
./start-multi-region.sh start
```

### Step 3: Verify Health
```bash
./start-multi-region.sh test
```

### Step 4: Start Observability (Optional)
```bash
docker compose -f docker-compose.observability.yml up -d
```

---

## 🧪 Run Tests

### Basic Functionality Test
```bash
cd deploy/mvp
./scripts/chaos-test.sh basic
```

### Network Latency Test
```bash
./scripts/chaos-test.sh latency
```

### Failover Test
```bash
./scripts/chaos-test.sh failover-a
```

### All Tests
```bash
./scripts/chaos-test.sh all
```

### End-to-End Tests
```bash
cd tests/e2e/multi-region
./run-e2e-tests.sh
```

---

## 📊 Access Services

### Application Services

| Service | Region A | Region B |
|---------|----------|----------|
| IM Service HTTP | http://localhost:8184 | http://localhost:8284 |
| IM Service gRPC | localhost:9194 | localhost:9294 |
| Gateway WebSocket | ws://localhost:8182 | ws://localhost:8282 |
| Gateway gRPC | localhost:9197 | localhost:9297 |

### Infrastructure

| Service | Endpoint | Credentials |
|---------|----------|-------------|
| MySQL | localhost:3307 | im_service / im_service_password |
| Redis | localhost:6380 | (no password) |
| Kafka | localhost:9093 | (no auth) |
| etcd | localhost:2379 | (no auth) |

### Monitoring

| Service | URL | Credentials |
|---------|-----|-------------|
| Prometheus | http://localhost:9090 | (no auth) |
| Grafana | http://localhost:3000 | admin / admin |
| Alertmanager | http://localhost:9093 | (no auth) |

---

## 🔧 Common Operations

### View Logs
```bash
# All services
./start-multi-region.sh logs

# Specific service
docker compose -f docker-compose.services.yml logs -f im-service-region-a
```

### Check Service Health
```bash
# Region A
curl http://localhost:8184/health

# Region B
curl http://localhost:8284/health

# Cross-region health
curl http://localhost:8184/health/cross-region
```

### Check Metrics
```bash
# Region A metrics
curl http://localhost:8184/metrics | grep hlc

# Region B metrics
curl http://localhost:8284/metrics | grep hlc

# Prometheus query
curl 'http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms'
```

### Test Traffic Switching
```bash
cd apps/im-service/cmd/traffic-cli

# Build CLI
go build -o traffic-cli .

# Check status
./traffic-cli status

# Switch 10% traffic to Region B
./traffic-cli switch --from region-a --to region-b --percentage 10
```

### Service Discovery
```bash
# List all registered services
docker exec etcd etcdctl get /im/services/ --prefix

# Watch for changes
docker exec etcd etcdctl watch /im/services/ --prefix
```

---

## 📈 Import Grafana Dashboards

### Option 1: Via API
```bash
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @../../docs/multi-region-demo/grafana-dashboard.json
```

### Option 2: Via UI
1. Open http://localhost:3000
2. Login with admin/admin
3. Go to Dashboards → Import
4. Upload `docs/multi-region-demo/grafana-dashboard.json`

---

## 🐛 Troubleshooting

### Services Won't Start

**Check logs:**
```bash
docker compose -f docker-compose.services.yml logs im-service-region-a
```

**Common issues:**
- Infrastructure not ready → Wait longer or restart infrastructure
- Port conflicts → Check if ports are in use: `lsof -i :8184`
- Memory limits → Increase Docker memory in Docker Desktop settings

### Can't Connect Between Regions

**Test connectivity:**
```bash
docker exec im-service-region-a ping im-service-region-b
```

**Check networks:**
```bash
docker network ls | grep region
docker network inspect region-a-network
```

### Metrics Not Showing

**Check Prometheus targets:**
```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health == "down")'
```

**Verify metrics endpoints:**
```bash
curl http://localhost:8184/metrics
curl http://localhost:8284/metrics
```

### High Latency

**Check Kafka consumer lag:**
```bash
docker exec kafka kafka-consumer-groups --describe \
  --group im-service-region-a-offline-workers \
  --bootstrap-server localhost:9092
```

**Monitor sync latency:**
```bash
curl 'http://localhost:9090/api/v1/query?query=cross_region_sync_latency_ms' | jq
```

---

## 🧹 Cleanup

### Stop All Services
```bash
./deploy-multi-region.sh cleanup
```

### Or Manually
```bash
# Stop multi-region services
./start-multi-region.sh stop

# Stop observability
docker compose -f docker-compose.observability.yml down

# Stop infrastructure
docker compose -f docker-compose.infra.yml down

# Remove volumes (optional - deletes all data)
docker compose -f docker-compose.infra.yml down -v
```

---

## 📚 Next Steps

1. **Explore Monitoring**
   - Open Grafana: http://localhost:3000
   - View multi-region dashboards
   - Set up custom alerts

2. **Run Chaos Tests**
   - Test network partitions
   - Test region failovers
   - Verify data consistency

3. **Performance Tuning**
   - Adjust batch sizes
   - Tune connection pools
   - Optimize replication intervals

4. **Production Planning**
   - Review [INFRASTRUCTURE_SETUP_GUIDE.md](./INFRASTRUCTURE_SETUP_GUIDE.md)
   - Plan actual geographic deployment
   - Configure DNS routing
   - Set up monitoring and alerting

---

## 📖 Documentation

- **Deployment Plan**: [DEPLOYMENT_EXECUTION_PLAN.md](./DEPLOYMENT_EXECUTION_PLAN.md)
- **Infrastructure Setup**: [INFRASTRUCTURE_SETUP_GUIDE.md](./INFRASTRUCTURE_SETUP_GUIDE.md)
- **Multi-Region Guide**: [README.multi-region.md](./README.multi-region.md)
- **Performance Tuning**: [../../docs/multi-region-demo/operations/PERFORMANCE_TUNING_GUIDE.md](../../docs/multi-region-demo/operations/PERFORMANCE_TUNING_GUIDE.md)
- **Troubleshooting**: [../../docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md](../../docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md)

---

## 🆘 Getting Help

If you encounter issues:

1. Check the [Troubleshooting Handbook](../../docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md)
2. Review service logs: `./start-multi-region.sh logs`
3. Verify health: `./deploy-multi-region.sh verify`
4. Check GitHub issues or create a new one

---

## ✅ Success Criteria

Your deployment is successful when:

- ✅ All services show "healthy" status
- ✅ Cross-region ping succeeds
- ✅ Services registered in etcd
- ✅ Metrics visible in Prometheus
- ✅ Grafana dashboards display data
- ✅ Basic chaos tests pass
- ✅ Sync latency < 500ms

**Congratulations! Your multi-region environment is ready! 🎉**
