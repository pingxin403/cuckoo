# Multi-Region Docker Compose Configuration

## Overview

This document describes the multi-region Docker Compose configuration for the IM system, supporting active-active deployment across two regions (Region A and Region B).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Shared Infrastructure                     │
│  (MySQL, Redis, Kafka, etcd, Auth Service, User Service)   │
└─────────────────────────────────────────────────────────────┘
                    │                    │
        ┌───────────┴──────────┐    ┌───┴──────────────┐
        │   Region A Network   │    │  Region B Network │
        │   (172.20.0.0/16)    │    │  (172.21.0.0/16)  │
        └──────────────────────┘    └───────────────────┘
                    │                    │
        ┌───────────▼──────────┐    ┌───▼───────────────┐
        │  im-service-region-a │    │ im-service-region-b│
        │  Port: 9194 (gRPC)   │    │ Port: 9294 (gRPC)  │
        │  Port: 8184 (HTTP)   │    │ Port: 8284 (HTTP)  │
        └──────────────────────┘    └────────────────────┘
                    │                    │
        ┌───────────▼──────────┐    ┌───▼───────────────┐
        │im-gateway-region-a   │    │im-gateway-region-b │
        │  Port: 9197 (gRPC)   │    │ Port: 9297 (gRPC)  │
        │  Port: 8182 (WS)     │    │ Port: 8282 (WS)    │
        └──────────────────────┘    └────────────────────┘
```

## Services

### Region A (Primary - Beijing)

#### im-service-region-a
- **Container**: `im-service-region-a`
- **Ports**:
  - `9194:9094` - gRPC API
  - `8184:8080` - HTTP (health checks, metrics)
- **Region Config**:
  - `REGION_ID=region-a`
  - `REGION_NAME=Primary Region (Beijing)`
  - `PEER_REGIONS=region-b`
  - `CROSS_REGION_ENABLED=true`
- **Networks**: `monorepo-network`, `region-a-network`

#### im-gateway-service-region-a
- **Container**: `im-gateway-service-region-a`
- **Ports**:
  - `9197:9097` - gRPC API
  - `8182:8080` - WebSocket
- **Region Config**:
  - `REGION_ID=region-a`
  - `ROUTING_ENABLED=true`
  - `PEER_REGION_B_ENDPOINT=http://im-gateway-service-region-b:8080`
  - `FAILOVER_ENABLED=true`
- **Networks**: `monorepo-network`, `region-a-network`

### Region B (Secondary - Shanghai)

#### im-service-region-b
- **Container**: `im-service-region-b`
- **Ports**:
  - `9294:9094` - gRPC API
  - `8284:8080` - HTTP (health checks, metrics)
- **Region Config**:
  - `REGION_ID=region-b`
  - `REGION_NAME=Secondary Region (Shanghai)`
  - `PEER_REGIONS=region-a`
  - `CROSS_REGION_ENABLED=true`
- **Networks**: `monorepo-network`, `region-b-network`
- **Redis DB**: Uses DB 3 (separate from Region A's DB 2)

#### im-gateway-service-region-b
- **Container**: `im-gateway-service-region-b`
- **Ports**:
  - `9297:9097` - gRPC API
  - `8282:8080` - WebSocket
- **Region Config**:
  - `REGION_ID=region-b`
  - `ROUTING_ENABLED=true`
  - `PEER_REGION_A_ENDPOINT=http://im-gateway-service-region-a:8080`
  - `FAILOVER_ENABLED=true`
- **Networks**: `monorepo-network`, `region-b-network`

## Network Configuration

### monorepo-network
- **Type**: External network
- **Purpose**: Shared infrastructure communication
- **Services**: All services connect to this network

### region-a-network
- **Type**: Bridge network
- **Subnet**: `172.20.0.0/16`
- **Purpose**: Region A service isolation
- **Services**: `im-service-region-a`, `im-gateway-service-region-a`

### region-b-network
- **Type**: Bridge network
- **Subnet**: `172.21.0.0/16`
- **Purpose**: Region B service isolation
- **Services**: `im-service-region-b`, `im-gateway-service-region-b`

## Key Features

### 1. Cross-Region Configuration
- **Enabled**: `CROSS_REGION_ENABLED=true`
- **Peer Discovery**: Each region knows about its peer
- **Sync Interval**: `100ms` for low-latency synchronization
- **Failover Timeout**: `30s` for automatic failover

### 2. Conflict Resolution
- **Strategy**: Last Write Wins (LWW)
- **Config**: `CONFLICT_RESOLUTION=lww`
- **Based on**: HLC (Hybrid Logical Clock) timestamps

### 3. Health Checks
- **Interval**: 15 seconds
- **Timeout**: 3 seconds
- **Retries**: 3 attempts
- **Start Period**: 20 seconds

### 4. Routing and Failover
- **Routing**: Enabled in both gateway services
- **Health Check Interval**: 30 seconds
- **Automatic Failover**: Enabled
- **Peer Endpoints**: Configured for cross-region communication

## Usage

### Start All Services (Including Multi-Region)

```bash
# Start infrastructure first
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# Start all services including multi-region
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml up -d
```

### Start Only Multi-Region Services

```bash
# Start Region A services
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml \
               up -d im-service-region-a im-gateway-service-region-a

# Start Region B services
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml \
               up -d im-service-region-b im-gateway-service-region-b
```

### Check Service Health

```bash
# Region A
curl http://localhost:8184/health  # IM Service Region A
curl http://localhost:8182/health  # Gateway Region A

# Region B
curl http://localhost:8284/health  # IM Service Region B
curl http://localhost:8282/health  # Gateway Region B
```

### View Logs

```bash
# Region A logs
docker compose -f deploy/docker/docker-compose.services.yml \
               logs -f im-service-region-a im-gateway-service-region-a

# Region B logs
docker compose -f deploy/docker/docker-compose.services.yml \
               logs -f im-service-region-b im-gateway-service-region-b
```

### Stop Multi-Region Services

```bash
# Stop all multi-region services
docker compose -f deploy/docker/docker-compose.services.yml \
               stop im-service-region-a im-gateway-service-region-a \
                    im-service-region-b im-gateway-service-region-b
```

## Testing Multi-Region Setup

### 1. Test Cross-Region Communication

```bash
# Send message via Region A
curl -X POST http://localhost:8182/ws \
  -H "Content-Type: application/json" \
  -d '{"type":"message","content":"Hello from Region A"}'

# Verify message received in Region B
curl http://localhost:8282/messages
```

### 2. Test Failover

```bash
# Stop Region A gateway
docker compose -f deploy/docker/docker-compose.services.yml \
               stop im-gateway-service-region-a

# Verify Region B takes over
curl http://localhost:8282/health
# Should return healthy status

# Restart Region A
docker compose -f deploy/docker/docker-compose.services.yml \
               start im-gateway-service-region-a
```

### 3. Monitor Cross-Region Metrics

```bash
# Region A metrics
curl http://localhost:8184/metrics | grep cross_region

# Region B metrics
curl http://localhost:8284/metrics | grep cross_region
```

## Environment Variables Reference

### Region Configuration
- `REGION_ID`: Unique region identifier (e.g., `region-a`, `region-b`)
- `REGION_NAME`: Human-readable region name
- `NODE_ID`: Node identifier within the region
- `CROSS_REGION_ENABLED`: Enable cross-region features (`true`/`false`)
- `PEER_REGIONS`: Comma-separated list of peer region IDs

### Synchronization
- `SYNC_INTERVAL`: Interval for cross-region sync (e.g., `100ms`)
- `FAILOVER_TIMEOUT`: Timeout for failover detection (e.g., `30s`)
- `CONFLICT_RESOLUTION`: Conflict resolution strategy (`lww`)

### Routing (Gateway)
- `ROUTING_ENABLED`: Enable intelligent routing (`true`/`false`)
- `PEER_REGION_*_ENDPOINT`: Peer region gateway endpoint URL
- `HEALTH_CHECK_INTERVAL`: Health check interval (e.g., `30s`)
- `FAILOVER_ENABLED`: Enable automatic failover (`true`/`false`)

## Troubleshooting

### Services Not Starting

```bash
# Check service logs
docker compose -f deploy/docker/docker-compose.services.yml \
               logs im-service-region-a

# Check network connectivity
docker network inspect region-a-network
docker network inspect region-b-network
```

### Cross-Region Communication Issues

```bash
# Test connectivity between regions
docker exec im-service-region-a ping im-service-region-b
docker exec im-gateway-service-region-a curl http://im-gateway-service-region-b:8080/health
```

### Health Check Failures

```bash
# Check service health directly
docker exec im-service-region-a wget --spider -q http://localhost:8080/health
echo $?  # Should return 0 if healthy

# Check dependencies
docker compose -f deploy/docker/docker-compose.infra.yml ps
```

## Next Steps

1. **Database Migration**: Add multi-region fields to database schema
2. **Monitoring**: Set up Grafana dashboards for multi-region metrics
3. **Alerting**: Configure Prometheus alerts for cross-region issues
4. **Load Testing**: Run load tests to verify multi-region performance
5. **Production Deployment**: Deploy to actual multi-region infrastructure

## Related Documentation

- [Multi-Region Integration Guide](../../apps/MULTI_REGION_INTEGRATION_COMPLETE.md)
- [Integration Summary](../../apps/INTEGRATION_SUMMARY.md)
- [IM Service Multi-Region Config](../../apps/im-service/config/multi-region-example.yaml)
- [Gateway Multi-Region Config](../../apps/im-gateway-service/config/multi-region-example.yaml)
