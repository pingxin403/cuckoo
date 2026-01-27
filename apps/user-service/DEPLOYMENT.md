# User Service - Deployment Guide

## Overview

The User Service provides user profile and group membership management for the IM Chat System.

## Infrastructure Requirements

### Production Requirements

**Per Node**:
- CPU: 4 cores
- Memory: 8GB RAM
- Storage: 50GB SSD

**Cluster**: 3+ nodes for high availability

## Service Dependencies

### Required Services

#### 1. MySQL (User Data Storage)
- **Purpose**: User profiles, group membership
- **Configuration**:
  ```yaml
  DB_HOST: mysql-master:3306
  DB_NAME: im_chat
  DB_USER: user_service
  DB_PASSWORD: ${DB_PASSWORD}
  DB_MAX_OPEN_CONNS: 25
  DB_MAX_IDLE_CONNS: 5
  ```

#### 2. Redis (Caching)
- **Purpose**: Cache user profiles and group membership
- **Configuration**:
  ```yaml
  REDIS_ADDR: redis-master:6379
  REDIS_PASSWORD: ${REDIS_PASSWORD}
  CACHE_USER_TTL: 10m
  CACHE_GROUP_TTL: 5m
  ```

## Configuration Parameters

### Environment Variables

```bash
# Server
SERVER_GRPC_PORT=9096
SERVER_HTTP_PORT=8096

# Database
DB_HOST=mysql-master:3306
DB_NAME=im_chat
DB_USER=user_service
DB_PASSWORD=${DB_PASSWORD}
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Cache
REDIS_ADDR=redis-master:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
CACHE_USER_TTL=10m
CACHE_GROUP_TTL=5m

# Pagination
MAX_PAGE_SIZE=1000
DEFAULT_PAGE_SIZE=100

# Observability
METRICS_ENABLED=true
LOG_LEVEL=info
```

### Configuration File

**config.yaml**:
```yaml
server:
  grpc_port: 9096
  http_port: 8096

database:
  host: mysql-master:3306
  name: im_chat
  user: user_service
  password: ${DB_PASSWORD}
  max_open_conns: 25
  max_idle_conns: 5

cache:
  redis_addr: redis-master:6379
  redis_password: ${REDIS_PASSWORD}
  user_ttl: 10m
  group_ttl: 5m

pagination:
  max_page_size: 1000
  default_page_size: 100

observability:
  metrics_enabled: true
  log_level: info
```

## Deployment Methods

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: im-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
    spec:
      containers:
      - name: user-service
        image: user-service:latest
        ports:
        - containerPort: 9096
          name: grpc
        - containerPort: 8096
          name: http
        env:
        - name: DB_HOST
          value: "mysql-master:3306"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        - name: REDIS_ADDR
          value: "redis-master:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        resources:
          requests:
            cpu: "2"
            memory: "4Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8096
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8096
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Scaling Guidelines

### Horizontal Scaling

Scale based on:
- Request rate > 5K req/sec per node
- CPU usage > 70%
- Database connection pool exhausted

```bash
kubectl scale deployment user-service --replicas=5 -n im-system
```

## Health Checks

### Liveness Probe
```bash
curl http://localhost:8096/health
```

### Readiness Probe
```bash
curl http://localhost:8096/ready
```

## References

- [API Documentation](./API.md)
- [Testing Guide](./TESTING.md)
