# Auth Service - Deployment Guide

## Overview

The Auth Service provides JWT token validation and refresh functionality for the IM Chat System.

## Infrastructure Requirements

### Production Requirements

**Per Node**:
- CPU: 2 cores
- Memory: 4GB RAM
- Storage: 20GB SSD

**Cluster**: 3+ nodes for high availability

## Service Dependencies

### Optional Services

#### 1. Redis (Token Caching)
- **Purpose**: Cache validated tokens
- **Configuration**:
  ```yaml
  REDIS_ADDR: redis-master:6379
  REDIS_PASSWORD: ${REDIS_PASSWORD}
  TOKEN_CACHE_TTL: 5m
  ```

## Configuration Parameters

### Environment Variables

```bash
# Server
SERVER_GRPC_PORT=9095
SERVER_HTTP_PORT=8095

# JWT
JWT_SECRET=${JWT_SECRET}
JWT_ISSUER=im-chat-system
JWT_EXPIRY=24h
REFRESH_TOKEN_EXPIRY=168h  # 7 days

# Token validation
MAX_DEVICE_LIMIT=5

# Observability
METRICS_ENABLED=true
LOG_LEVEL=info
```

### Configuration File

**config.yaml**:
```yaml
server:
  grpc_port: 9095
  http_port: 8095

jwt:
  secret: ${JWT_SECRET}
  issuer: im-chat-system
  expiry: 24h
  refresh_token_expiry: 168h

validation:
  max_device_limit: 5

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
  name: auth-service
  namespace: im-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: auth-service
  template:
    metadata:
      labels:
        app: auth-service
    spec:
      containers:
      - name: auth-service
        image: auth-service:latest
        ports:
        - containerPort: 9095
          name: grpc
        - containerPort: 8095
          name: http
        env:
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: auth-secret
              key: jwt-secret
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8095
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8095
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Scaling Guidelines

### Horizontal Scaling

Scale based on:
- Request rate > 10K req/sec per node
- CPU usage > 70%
- P99 latency > 50ms

```bash
kubectl scale deployment auth-service --replicas=5 -n im-system
```

## Health Checks

### Liveness Probe
```bash
curl http://localhost:8095/health
```

### Readiness Probe
```bash
curl http://localhost:8095/ready
```

## References

- [API Documentation](./API.md)
- [Testing Guide](./TESTING.md)
