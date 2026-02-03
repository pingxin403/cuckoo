# Multi-Region Docker Compose Setup

## Quick Start

The fastest way to get started with multi-region deployment:

```bash
# Start all multi-region services
./deploy/docker/start-multi-region.sh start

# Check service health
./deploy/docker/start-multi-region.sh test

# View logs
./deploy/docker/start-multi-region.sh logs

# Stop services
./deploy/docker/start-multi-region.sh stop
```

## What's Included

This Docker Compose configuration provides a complete multi-region active-active deployment with:

### Region A (Primary - Beijing)
- **im-service-region-a**: Message routing and offline persistence
  - gRPC: `localhost:9194`
  - HTTP: `localhost:8184`
- **im-gateway-service-region-a**: WebSocket gateway
  - gRPC: `localhost:9197`
  - WebSocket: `localhost:8182`

### Region B (Secondary - Shanghai)
- **im-service-region-b**: Message routing and offline persistence
  - gRPC: `localhost:9294`
  - HTTP: `localhost:8284`
- **im-gateway-service-region-b**: WebSocket gateway
  - gRPC: `localhost:9297`
  - WebSocket: `localhost:8282`

### Shared Infrastructure
- MySQL (database)
- Redis (cache)
- Kafka (message queue)
- etcd (service discovery)
- Auth Service
- User Service

## Architecture Highlights

### Cross-Region Communication
- Services in different regions can communicate directly
- Each region has its own network for isolation
- All regions share the main `monorepo-network` for infrastructure access

### Data Synchronization
- **HLC (Hybrid Logical Clock)**: Global ID generation with causal ordering
- **LWW (Last Write Wins)**: Conflict resolution strategy
- **Kafka**: Async message replication between regions
- **Redis**: Separate DBs per region (Region A: DB 2, Region B: DB 3)

### Service Discovery
- etcd-based service registration with region awareness
- Health checks every 15 seconds
- Automatic failover support

## Configuration

### Key Environment Variables

Each region is configured with:

```bash
# Region Identity
REGION_ID=region-a|region-b
REGION_NAME=Primary Region (Beijing)|Secondary Region (Shanghai)

# Cross-Region Settings
CROSS_REGION_ENABLED=true
PEER_REGIONS=region-b|region-a
SYNC_INTERVAL=100ms
FAILOVER_TIMEOUT=30s
CONFLICT_RESOLUTION=lww

# Routing (Gateway)
ROUTING_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
FAILOVER_ENABLED=true
```

### Port Mapping

| Service | Region A | Region B |
|---------|----------|----------|
| IM Service gRPC | 9194 | 9294 |
| IM Service HTTP | 8184 | 8284 |
| Gateway gRPC | 9197 | 9297 |
| Gateway WebSocket | 8182 | 8282 |

## Usage Examples

### Manual Docker Compose Commands

```bash
# Start infrastructure
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# Start multi-region services
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml up -d \
               im-service-region-a \
               im-gateway-service-region-a \
               im-service-region-b \
               im-gateway-service-region-b

# View logs for specific region
docker compose -f deploy/docker/docker-compose.services.yml logs -f im-service-region-a

# Stop multi-region services
docker compose -f deploy/docker/docker-compose.services.yml stop \
               im-service-region-a im-gateway-service-region-a \
               im-service-region-b im-gateway-service-region-b
```

### Health Checks

```bash
# Check Region A
curl http://localhost:8184/health
curl http://localhost:8182/health

# Check Region B
curl http://localhost:8284/health
curl http://localhost:8282/health
```

### Service Discovery

```bash
# List all registered services
docker exec etcd etcdctl get /im/services/ --prefix

# Check Region A services
docker exec etcd etcdctl get /im/services/region-a/ --prefix

# Check Region B services
docker exec etcd etcdctl get /im/services/region-b/ --prefix
```

### Cross-Region Testing

```bash
# Test connectivity from Region A to Region B
docker exec im-service-region-a ping im-service-region-b

# Test connectivity from Region B to Region A
docker exec im-service-region-b ping im-service-region-a

# Check gateway peer connectivity
docker exec im-gateway-service-region-a curl http://im-gateway-service-region-b:8080/health
```

## Testing Scenarios

### 1. Normal Operation

Both regions running and synchronizing:

```bash
# Start all services
./deploy/docker/start-multi-region.sh start

# Verify both regions are healthy
./deploy/docker/start-multi-region.sh test
```

### 2. Region Failover

Simulate Region A failure:

```bash
# Stop Region A
docker compose -f deploy/docker/docker-compose.services.yml stop \
               im-service-region-a im-gateway-service-region-a

# Verify Region B continues to operate
curl http://localhost:8284/health

# Restart Region A
docker compose -f deploy/docker/docker-compose.services.yml start \
               im-service-region-a im-gateway-service-region-a
```

### 3. Network Latency Simulation

Add artificial latency to simulate geographic distance:

```bash
# Add 50ms latency to Region B
docker exec im-service-region-b tc qdisc add dev eth0 root netem delay 50ms

# Test with latency
./deploy/docker/start-multi-region.sh test

# Remove latency
docker exec im-service-region-b tc qdisc del dev eth0 root
```

### 4. Split-Brain Scenario

Test network partition between regions:

```bash
# Block traffic from Region A to Region B
docker exec im-service-region-a iptables -A OUTPUT -d im-service-region-b -j DROP

# Verify each region operates independently
curl http://localhost:8184/health
curl http://localhost:8284/health

# Restore connectivity
docker exec im-service-region-a iptables -D OUTPUT -d im-service-region-b -j DROP
```

## Monitoring

### Metrics

Access Prometheus metrics from each service:

```bash
# Region A
curl http://localhost:8184/metrics
curl http://localhost:8182/metrics

# Region B
curl http://localhost:8284/metrics
curl http://localhost:8282/metrics
```

### Logs

View logs from all multi-region services:

```bash
# All services
./deploy/docker/start-multi-region.sh logs

# Specific service
docker compose -f deploy/docker/docker-compose.services.yml logs -f im-service-region-a
```

### Kafka Messages

Monitor message flow:

```bash
# List topics
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Monitor offline_msg topic
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic offline_msg \
  --from-beginning
```

## Troubleshooting

### Services Won't Start

1. Check infrastructure is running:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml ps
   ```

2. Check logs for errors:
   ```bash
   docker compose -f deploy/docker/docker-compose.services.yml logs im-service-region-a
   ```

3. Verify network connectivity:
   ```bash
   docker network ls | grep region
   ```

### Cross-Region Communication Fails

1. Check network configuration:
   ```bash
   docker network inspect region-a-network
   docker network inspect region-b-network
   ```

2. Test connectivity:
   ```bash
   docker exec im-service-region-a ping im-service-region-b
   ```

3. Check firewall rules:
   ```bash
   docker exec im-service-region-a iptables -L
   ```

### Database Connection Issues

1. Verify MySQL is running:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml ps mysql
   ```

2. Test connection from service:
   ```bash
   docker exec im-service-region-a nc -zv mysql 3306
   ```

3. Check database credentials:
   ```bash
   docker exec mysql mysql -uim_service -pim_service_password im_chat -e "SELECT 1;"
   ```

## Performance Tuning

### Resource Limits

Add resource constraints to prevent resource exhaustion:

```yaml
services:
  im-service-region-a:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
```

### Connection Pooling

Adjust database connection pool:

```bash
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
```

### Kafka Tuning

Optimize message processing:

```bash
BATCH_SIZE=200
BATCH_TIMEOUT=3s
```

## Next Steps

1. **Run Integration Tests**: Verify multi-region functionality
   ```bash
   cd apps/im-service/integration_test
   go test -v
   ```

2. **Database Migration**: Add multi-region fields to database schema
   ```bash
   # Add region_id, global_id, sync_status columns
   ```

3. **Monitoring Setup**: Create Grafana dashboards for multi-region metrics

4. **Production Deployment**: Deploy to actual geographic regions with proper networking

## References

- [Detailed Deployment Guide](./MULTI_REGION_DEPLOYMENT.md)
- [Integration Guide](../../apps/MULTI_REGION_INTEGRATION_COMPLETE.md)

## Support

For issues or questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review logs: `./deploy/docker/start-multi-region.sh logs`
3. Run health checks: `./deploy/docker/start-multi-region.sh test`
4. Consult the [detailed deployment guide](./MULTI_REGION_DEPLOYMENT.md)
