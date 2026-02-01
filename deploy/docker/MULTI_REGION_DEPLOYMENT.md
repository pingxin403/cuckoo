# Multi-Region Docker Compose Deployment Guide

## Overview

This guide explains how to deploy and test the multi-region active-active architecture using Docker Compose. The setup simulates two geographic regions (Region A and Region B) with independent service instances that communicate and synchronize data across regions.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Shared Infrastructure                        │
│  (MySQL, Redis, Kafka, etcd, Auth Service, User Service)       │
└─────────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┴─────────────┐
                │                           │
        ┌───────▼────────┐          ┌───────▼────────┐
        │   Region A     │          │   Region B     │
        │   (Primary)    │◄────────►│  (Secondary)   │
        │                │          │                │
        │  Port: 9194    │          │  Port: 9294    │
        │  Port: 9197    │          │  Port: 9297    │
        └────────────────┘          └────────────────┘
```

## Services

### Region A (Primary - Beijing)

1. **im-service-region-a**
   - gRPC Port: `9194` (mapped from internal 9094)
   - HTTP Port: `8184` (mapped from internal 8080)
   - Region ID: `region-a`
   - Consumer Group: `im-service-region-a-offline-workers`
   - Redis DB: `2`

2. **im-gateway-service-region-a**
   - gRPC Port: `9197` (mapped from internal 9097)
   - WebSocket Port: `8182` (mapped from internal 8080)
   - Region ID: `region-a`
   - Peer Endpoint: `http://im-gateway-service-region-b:8080`

### Region B (Secondary - Shanghai)

1. **im-service-region-b**
   - gRPC Port: `9294` (mapped from internal 9094)
   - HTTP Port: `8284` (mapped from internal 8080)
   - Region ID: `region-b`
   - Consumer Group: `im-service-region-b-offline-workers`
   - Redis DB: `3`

2. **im-gateway-service-region-b**
   - gRPC Port: `9297` (mapped from internal 9097)
   - WebSocket Port: `8282` (mapped from internal 8080)
   - Region ID: `region-b`
   - Peer Endpoint: `http://im-gateway-service-region-a:8080`

### Shared Infrastructure

All regions share the following infrastructure services:
- **MySQL**: Database for persistent storage
- **Redis**: Cache and session storage
- **Kafka**: Message queue for async communication
- **etcd**: Service discovery and coordination
- **Auth Service**: Authentication
- **User Service**: User management

## Network Configuration

### Networks

1. **monorepo-network**: Main network for all services
2. **region-a-network**: Isolated network for Region A services
   - Subnet: `172.20.0.0/16`
3. **region-b-network**: Isolated network for Region B services
   - Subnet: `172.21.0.0/16`

Services in each region are connected to both the main network (for shared infrastructure) and their region-specific network (for isolation).

## Configuration

### Environment Variables

#### Region-Specific Configuration

Both regions use the following configuration variables:

```bash
# Region Identity
REGION_ID=region-a|region-b
REGION_NAME=Primary Region (Beijing)|Secondary Region (Shanghai)
NODE_ID=node-1

# Cross-Region Settings
CROSS_REGION_ENABLED=true
PEER_REGIONS=region-b|region-a
SYNC_INTERVAL=100ms
FAILOVER_TIMEOUT=30s
CONFLICT_RESOLUTION=lww

# Routing (Gateway only)
ROUTING_ENABLED=true
PEER_REGION_X_ENDPOINT=http://im-gateway-service-region-x:8080
HEALTH_CHECK_INTERVAL=30s
FAILOVER_ENABLED=true
```

#### Key Differences Between Regions

| Configuration | Region A | Region B |
|--------------|----------|----------|
| REGION_ID | region-a | region-b |
| REGION_NAME | Primary Region (Beijing) | Secondary Region (Shanghai) |
| PEER_REGIONS | region-b | region-a |
| Redis DB | 2 | 3 |
| Consumer Group | im-service-region-a-offline-workers | im-service-region-b-offline-workers |

## Deployment

### Prerequisites

1. Ensure infrastructure services are running:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml up -d
   ```

2. Verify infrastructure health:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml ps
   ```

### Start Multi-Region Services

1. **Start all services (including multi-region)**:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml \
                  -f deploy/docker/docker-compose.services.yml up -d
   ```

2. **Start only multi-region services**:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml \
                  -f deploy/docker/docker-compose.services.yml up -d \
                  im-service-region-a \
                  im-gateway-service-region-a \
                  im-service-region-b \
                  im-gateway-service-region-b
   ```

3. **View logs**:
   ```bash
   # Region A logs
   docker compose -f deploy/docker/docker-compose.services.yml logs -f im-service-region-a
   docker compose -f deploy/docker/docker-compose.services.yml logs -f im-gateway-service-region-a
   
   # Region B logs
   docker compose -f deploy/docker/docker-compose.services.yml logs -f im-service-region-b
   docker compose -f deploy/docker/docker-compose.services.yml logs -f im-gateway-service-region-b
   ```

### Stop Services

```bash
# Stop all services
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml down

# Stop only multi-region services
docker compose -f deploy/docker/docker-compose.services.yml stop \
               im-service-region-a \
               im-gateway-service-region-a \
               im-service-region-b \
               im-gateway-service-region-b
```

## Testing

### Health Checks

Check the health of each region:

```bash
# Region A
curl http://localhost:8184/health
curl http://localhost:8182/health  # Gateway

# Region B
curl http://localhost:8284/health
curl http://localhost:8282/health  # Gateway
```

### Cross-Region Communication

Test cross-region message synchronization:

```bash
# Send message to Region A
curl -X POST http://localhost:8182/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "user1",
    "receiver_id": "user2",
    "content": "Hello from Region A"
  }'

# Verify message appears in Region B
curl http://localhost:8282/api/v1/messages?user_id=user2
```

### Failover Testing

1. **Stop Region A**:
   ```bash
   docker compose -f deploy/docker/docker-compose.services.yml stop \
                  im-service-region-a im-gateway-service-region-a
   ```

2. **Verify Region B handles traffic**:
   ```bash
   curl http://localhost:8282/health
   ```

3. **Restart Region A**:
   ```bash
   docker compose -f deploy/docker/docker-compose.services.yml start \
                  im-service-region-a im-gateway-service-region-a
   ```

### Network Latency Simulation

To simulate cross-region network latency, you can use `tc` (traffic control):

```bash
# Add 50ms latency to Region B
docker exec im-service-region-b tc qdisc add dev eth0 root netem delay 50ms

# Remove latency
docker exec im-service-region-b tc qdisc del dev eth0 root
```

## Monitoring

### Service Discovery

Check service registration in etcd:

```bash
# List all registered services
docker exec etcd etcdctl get /im/services/ --prefix

# Check Region A services
docker exec etcd etcdctl get /im/services/region-a/ --prefix

# Check Region B services
docker exec etcd etcdctl get /im/services/region-b/ --prefix
```

### Metrics

Access Prometheus metrics:

```bash
# Region A metrics
curl http://localhost:8184/metrics
curl http://localhost:8182/metrics  # Gateway

# Region B metrics
curl http://localhost:8284/metrics
curl http://localhost:8282/metrics  # Gateway
```

### Kafka Topics

Monitor cross-region message flow:

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

### Services Not Starting

1. **Check dependencies**:
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml ps
   ```

2. **Check logs**:
   ```bash
   docker compose -f deploy/docker/docker-compose.services.yml logs im-service-region-a
   ```

3. **Verify network connectivity**:
   ```bash
   docker exec im-service-region-a ping im-service-region-b
   ```

### Cross-Region Communication Issues

1. **Check network configuration**:
   ```bash
   docker network inspect region-a-network
   docker network inspect region-b-network
   ```

2. **Verify peer endpoints**:
   ```bash
   docker exec im-gateway-service-region-a env | grep PEER
   docker exec im-gateway-service-region-b env | grep PEER
   ```

3. **Test connectivity**:
   ```bash
   docker exec im-gateway-service-region-a curl http://im-gateway-service-region-b:8080/health
   ```

### Database Issues

1. **Check MySQL connection**:
   ```bash
   docker exec im-service-region-a nc -zv mysql 3306
   ```

2. **Verify database schema**:
   ```bash
   docker exec mysql mysql -uim_service -pim_service_password im_chat -e "SHOW TABLES;"
   ```

### Redis Issues

1. **Check Redis connectivity**:
   ```bash
   docker exec im-service-region-a redis-cli -h redis ping
   ```

2. **Verify Redis DB separation**:
   ```bash
   # Region A uses DB 2
   docker exec redis redis-cli -n 2 KEYS "*"
   
   # Region B uses DB 3
   docker exec redis redis-cli -n 3 KEYS "*"
   ```

## Performance Tuning

### Resource Limits

Add resource limits to prevent resource exhaustion:

```yaml
services:
  im-service-region-a:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Connection Pooling

Adjust database connection pool settings:

```bash
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
```

### Kafka Consumer Tuning

Optimize Kafka consumer performance:

```bash
BATCH_SIZE=200
BATCH_TIMEOUT=3s
```

## Production Considerations

### Security

1. **Use secrets management** for sensitive data (passwords, tokens)
2. **Enable TLS** for cross-region communication
3. **Implement authentication** between regions
4. **Use network policies** to restrict traffic

### High Availability

1. **Deploy multiple nodes** per region
2. **Use external load balancers** for traffic distribution
3. **Implement health checks** at multiple levels
4. **Configure automatic restart** policies

### Monitoring

1. **Set up Prometheus** for metrics collection
2. **Configure Grafana** dashboards for visualization
3. **Implement alerting** for critical events
4. **Enable distributed tracing** with OpenTelemetry

### Backup and Recovery

1. **Regular database backups** with point-in-time recovery
2. **Kafka topic replication** across regions
3. **Redis persistence** configuration
4. **Disaster recovery procedures** documentation

## References

- [Multi-Region Architecture Design](../../.kiro/specs/multi-region-active-active/design.md)
- [Requirements Document](../../.kiro/specs/multi-region-active-active/requirements.md)
- [Integration Guide](../../apps/MULTI_REGION_INTEGRATION_COMPLETE.md)
- [IM Service Configuration](../../apps/im-service/config/multi-region-example.yaml)
- [Gateway Configuration](../../apps/im-gateway-service/config/multi-region-example.yaml)

## Next Steps

1. **Run integration tests** to verify multi-region functionality
2. **Implement database migration** to add multi-region fields
3. **Deploy to staging environment** for end-to-end testing
4. **Create monitoring dashboards** for production readiness
5. **Document operational procedures** for production deployment
