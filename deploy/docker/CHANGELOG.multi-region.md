# Multi-Region Docker Compose Changes

## Overview

This document describes the changes made to support multi-region active-active deployment in Docker Compose.

## Changes Made

### 1. Updated `docker-compose.services.yml`

Added four new services for multi-region deployment:

#### Region A (Primary - Beijing)
- **im-service-region-a**
  - Container: `im-service-region-a`
  - Ports: 9194 (gRPC), 8184 (HTTP)
  - Region ID: `region-a`
  - Redis DB: 2
  - Consumer Group: `im-service-region-a-offline-workers`
  - Networks: `monorepo-network`, `region-a-network`

- **im-gateway-service-region-a**
  - Container: `im-gateway-service-region-a`
  - Ports: 9197 (gRPC), 8182 (WebSocket)
  - Region ID: `region-a`
  - Peer: `http://im-gateway-service-region-b:8080`
  - Networks: `monorepo-network`, `region-a-network`

#### Region B (Secondary - Shanghai)
- **im-service-region-b**
  - Container: `im-service-region-b`
  - Ports: 9294 (gRPC), 8284 (HTTP)
  - Region ID: `region-b`
  - Redis DB: 3
  - Consumer Group: `im-service-region-b-offline-workers`
  - Networks: `monorepo-network`, `region-b-network`

- **im-gateway-service-region-b**
  - Container: `im-gateway-service-region-b`
  - Ports: 9297 (gRPC), 8282 (WebSocket)
  - Region ID: `region-b`
  - Peer: `http://im-gateway-service-region-a:8080`
  - Networks: `monorepo-network`, `region-b-network`

### 2. Network Configuration

Added two new Docker networks for region isolation:

- **region-a-network**
  - Driver: bridge
  - Subnet: 172.20.0.0/16
  - Purpose: Isolate Region A services

- **region-b-network**
  - Driver: bridge
  - Subnet: 172.21.0.0/16
  - Purpose: Isolate Region B services

### 3. New Documentation Files

Created comprehensive documentation:

- **MULTI_REGION_DEPLOYMENT.md**: Complete deployment guide
  - Architecture overview
  - Service descriptions
  - Configuration details
  - Deployment procedures
  - Testing scenarios
  - Troubleshooting guide
  - Performance tuning
  - Production considerations

- **README.multi-region.md**: Quick start guide
  - Quick start commands
  - Architecture highlights
  - Usage examples
  - Testing scenarios
  - Monitoring instructions
  - Troubleshooting tips

- **CHANGELOG.multi-region.md**: This file
  - Summary of changes
  - Configuration details
  - Migration notes

### 4. Helper Scripts

Created `start-multi-region.sh` script with commands:
- `start`: Start all multi-region services
- `stop`: Stop all multi-region services
- `restart`: Restart services
- `status`: Show service status
- `logs`: View logs
- `test`: Run health checks
- `clean`: Clean up all resources
- `help`: Show usage information

## Configuration Details

### Environment Variables Added

All multi-region services include:

```bash
# Region Identity
REGION_ID=region-a|region-b
REGION_NAME=Primary Region (Beijing)|Secondary Region (Shanghai)
NODE_ID=node-1

# Cross-Region Configuration
CROSS_REGION_ENABLED=true
PEER_REGIONS=region-b|region-a
SYNC_INTERVAL=100ms
FAILOVER_TIMEOUT=30s
CONFLICT_RESOLUTION=lww
```

Gateway services additionally include:

```bash
# Routing Configuration
ROUTING_ENABLED=true
PEER_REGION_X_ENDPOINT=http://im-gateway-service-region-x:8080
HEALTH_CHECK_INTERVAL=30s
FAILOVER_ENABLED=true
```

### Port Allocation

| Service | Region A | Region B | Purpose |
|---------|----------|----------|---------|
| IM Service gRPC | 9194 | 9294 | gRPC API |
| IM Service HTTP | 8184 | 8284 | Health checks, metrics |
| Gateway gRPC | 9197 | 9297 | gRPC API |
| Gateway WebSocket | 8182 | 8282 | WebSocket connections |

### Resource Isolation

- Each region uses a separate Redis database (Region A: DB 2, Region B: DB 3)
- Each region has its own Kafka consumer group
- Each region has its own Docker network for isolation
- All regions share the same MySQL database (with region_id field for data separation)

## Migration Notes

### For Existing Deployments

1. **No Breaking Changes**: Existing single-region services (`im-service`, `im-gateway-service`) remain unchanged
2. **Additive Changes**: Multi-region services are additional services, not replacements
3. **Shared Infrastructure**: Multi-region services use the same infrastructure (MySQL, Redis, Kafka, etcd)

### Deployment Strategy

1. **Phase 1**: Deploy infrastructure (if not already running)
   ```bash
   docker compose -f deploy/docker/docker-compose.infra.yml up -d
   ```

2. **Phase 2**: Deploy multi-region services
   ```bash
   ./deploy/docker/start-multi-region.sh start
   ```

3. **Phase 3**: Verify health and connectivity
   ```bash
   ./deploy/docker/start-multi-region.sh test
   ```

### Database Schema Changes (Future)

The following database changes will be needed for full multi-region support:

```sql
-- Add multi-region fields to offline_messages table
ALTER TABLE offline_messages 
  ADD COLUMN region_id VARCHAR(50),
  ADD COLUMN global_id VARCHAR(255),
  ADD COLUMN sync_status ENUM('pending', 'synced', 'conflict') DEFAULT 'pending',
  ADD INDEX idx_region_id (region_id),
  ADD INDEX idx_global_id (global_id);

-- Add multi-region fields to user_sessions table
ALTER TABLE user_sessions
  ADD COLUMN region_id VARCHAR(50),
  ADD COLUMN cross_region_state JSON,
  ADD INDEX idx_region_id (region_id);
```

## Testing

### Validation Steps

1. **Service Health**:
   ```bash
   ./deploy/docker/start-multi-region.sh test
   ```

2. **Cross-Region Connectivity**:
   ```bash
   docker exec im-service-region-a ping im-service-region-b
   docker exec im-service-region-b ping im-service-region-a
   ```

3. **Service Discovery**:
   ```bash
   docker exec etcd etcdctl get /im/services/ --prefix
   ```

4. **Failover**:
   ```bash
   # Stop Region A
   docker compose -f deploy/docker/docker-compose.services.yml stop im-service-region-a
   
   # Verify Region B continues
   curl http://localhost:8284/health
   
   # Restart Region A
   docker compose -f deploy/docker/docker-compose.services.yml start im-service-region-a
   ```

## Known Limitations

1. **Database Schema**: Multi-region fields not yet added to database tables
2. **Message Synchronization**: Full cross-region message sync implementation pending
3. **Monitoring**: Grafana dashboards for multi-region metrics not yet created
4. **Network Latency**: No built-in latency simulation (requires manual tc commands)

## Next Steps

1. **Database Migration**: Add multi-region fields to database schema
2. **Integration Tests**: Run comprehensive multi-region integration tests
3. **Monitoring**: Create Grafana dashboards for multi-region metrics
4. **Documentation**: Add operational runbooks for production deployment
5. **Performance Testing**: Conduct load testing with cross-region traffic

## References

- [Multi-Region Deployment Guide](./MULTI_REGION_DEPLOYMENT.md)
- [Quick Start Guide](./README.multi-region.md)
- [Integration Guide](../../apps/MULTI_REGION_INTEGRATION_COMPLETE.md)

## Version History

- **v1.0.0** (2024-01-XX): Initial multi-region Docker Compose support
  - Added 4 multi-region services (2 per region)
  - Created region-specific networks
  - Added comprehensive documentation
  - Created helper scripts for deployment and testing
