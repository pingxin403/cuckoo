# Multi-Region Deployment Guide

## Overview

This document outlines the strategy for deploying the URL Shortener Service across multiple regions for high availability and disaster recovery.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Global Load Balancer                  │
│                    (Cloudflare/AWS Global)                   │
└─────────────────────┬─────────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │  US-EAST│   │ EU-WEST │   │ AP-SOUTH│
   └────┬────┘   └────┬────┘   └────┬────┘
        │            │            │
        ▼            ▼            ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │Shortener│   │Shortener│   │Shortener│
   │ Service │   │ Service │   │ Service │
   └────┬────┘   └────┬────┘   └────┬────┘
        │            │            │
        └────────────┼────────────┘
                     ▼
           ┌─────────────────┐
           │  Data Sync      │
           │ (MySQL Binlog)  │
           └─────────────────┘
```

## Deployment Strategy

### 1. Regional Services

Each region runs independent shortener service instances:
- **US-EAST**: Primary region
- **EU-WEST**: European traffic
- **AP-SOUTH**: Asia-Pacific traffic

### 2. Data Synchronization

- **MySQL**: Cross-region replication using binlog
- **Redis**: Local caches, periodic warm-up from primary
- **Analytics**: Async Kafka-based event collection

### 3. Traffic Routing

- **GeoDNS**: Route users to nearest region
- **Health Checks**: Automatic failover on regional failure
- **Consistent Hashing**: Same short-code accessible globally

## Kubernetes Configuration

### Regional Deployments

Each region has its own K8s deployment:
- `deploy/k8s/services/shortener-service/us-east/`
- `deploy/k8s/services/shortener-service/eu-west/`
- `deploy/k8s/services/shortener-service/ap-south/`

### Global Service (External-DNS)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: shortener-global
spec:
  type: ExternalName
  externalName: shortener.example.com
```

## Failover Procedures

### Primary Region Failure

1. Detect failure via health checks
2. Update DNS to point to backup region
3. Backup region serves traffic
4. Data syncs via async replication

### Recovery

1. Restore primary region
2. Backfill missed writes from backup
3. Gradually shift traffic back

## Implementation Status

- [x] Service supports horizontal scaling
- [x] Health check endpoints (/health, /ready)
- [x] Graceful degradation on cache miss
- [ ] Cross-region replication (future work)
- [ ] Global load balancer configuration (future work)
- [ ] Automated failover (future work)