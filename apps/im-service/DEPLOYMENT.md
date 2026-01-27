# IM Service - Deployment Guide

## Overview

The IM Service is the core message routing service that handles private and group message delivery, offline message storage, and read receipts.

## Table of Contents

1. [Infrastructure Requirements](#infrastructure-requirements)
2. [Service Dependencies](#service-dependencies)
3. [Configuration Parameters](#configuration-parameters)
4. [Deployment Methods](#deployment-methods)
5. [Scaling Guidelines](#scaling-guidelines)
6. [Operational Runbooks](#operational-runbooks)
7. [Troubleshooting](#troubleshooting)

## Infrastructure Requirements

### Minimum Requirements (Development)

**Single Node**:
- CPU: 4 cores
- Memory: 8GB RAM
- Storage: 50GB SSD
- Network: 1 Gbps
- OS: Linux (Ubuntu 20.04+, CentOS 8+)

### Production Requirements

**Per IM Service Node**:
- CPU: 8 cores (3.0 GHz+)
- Memory: 16GB RAM
- Storage: 100GB SSD
- Network: 10 Gbps

**Cluster Configuration**:
- IM Service Nodes: 3+ (for high availability)
- MySQL: Master-replica setup
- Redis: 3 nodes (master + 2 replicas)
- Kafka: 3+ brokers
- etcd: 3 nodes

## Service Dependencies

### Required Services

#### 1. MySQL (Message Storage)
- **Purpose**: Offline message storage, sequence snapshots, read receipts
- **Version**: 8.0+
- **Configuration**:
  ```yaml
  DB_HOST: mysql-master:3306
  DB_NAME: im_chat
  DB_USER: im_service
  DB_PASSWORD: ${DB_PASSWORD}
  DB_MAX_OPEN_CONNS: 25
  DB_MAX_IDLE_CONNS: 5
  ```

#### 2. Redis (Sequence Generator & Deduplication)
- **Purpose**: Sequence number generation, message deduplication
- **Version**: 6.0+
- **Configuration**:
  ```yaml
  REDIS_ADDR: redis-master:6379
  REDIS_PASSWORD: ${REDIS_PASSWORD}
  REDIS_DB: 0
  REDIS_POOL_SIZE: 100
  ```

#### 3. Kafka (Message Bus)
- **Purpose**: Group messages, offline messages, membership changes
- **Version**: 2.8+
- **Topics**: `group_msg`, `offline_msg`, `membership_change`
- **Configuration**:
  ```yaml
  KAFKA_BROKERS: kafka-1:9092,kafka-2:9092,kafka-3:9092
  KAFKA_GROUP_ID: im-service-group
  ```

#### 4. etcd (Service Registry)
- **Purpose**: User-to-gateway mapping lookup
- **Version**: 3.5+
- **Configuration**:
  ```yaml
  ETCD_ENDPOINTS: etcd-1:2379,etcd-2:2379,etcd-3:2379
  ```

#### 5. Gateway Service (Message Delivery)
- **Purpose**: Push messages to connected clients
- **Version**: v1.0+
- **Configuration**:
  ```yaml
  GATEWAY_SERVICE_ADDR: im-gateway-service:9093
  ```

#### 6. User Service (User Profiles)
- **Purpose**: User profile lookup, group membership validation
- **Version**: v1.0+
- **Configuration**:
  ```yaml
  USER_SERVICE_ADDR: user-service:9096
  ```

### Optional Services

#### 1. KMS (Key Management)
- **Purpose**: Encryption key management
- **Configuration**:
  ```yaml
  KMS_ENDPOINT: kms:8080
  KMS_KEY_ID: ${KMS_KEY_ID}
  ```

## Configuration Parameters

### Environment Variables

#### Server Configuration
```bash
# Server ports
SERVER_GRPC_PORT=9094          # gRPC port
SERVER_HTTP_PORT=8094          # HTTP port for health checks

# Service limits
MAX_MESSAGE_SIZE=10240         # Max message size (10KB)
MAX_RETRY_ATTEMPTS=3           # Max retry attempts for delivery
RETRY_TIMEOUT=5s               # Timeout per retry attempt
```

#### Database Configuration
```bash
# MySQL
DB_HOST=mysql-master:3306
DB_NAME=im_chat
DB_USER=im_service
DB_PASSWORD=${DB_PASSWORD}
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=1h

# Connection pooling
DB_CONN_TIMEOUT=10s
DB_READ_TIMEOUT=30s
DB_WRITE_TIMEOUT=30s
```

#### Redis Configuration
```bash
# Redis
REDIS_ADDR=redis-master:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0
REDIS_POOL_SIZE=100
REDIS_MIN_IDLE_CONNS=10
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s

# Sequence generator
SEQUENCE_SNAPSHOT_INTERVAL=10000  # Snapshot every 10K messages
```

#### Kafka Configuration
```bash
# Kafka brokers
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092

# Producer settings
KAFKA_PRODUCER_TIMEOUT=10s
KAFKA_PRODUCER_RETRY_MAX=3
KAFKA_PRODUCER_BATCH_SIZE=16384

# Consumer settings (Offline Worker)
KAFKA_CONSUMER_GROUP_ID=im-service-offline-worker
KAFKA_CONSUMER_AUTO_OFFSET_RESET=latest
KAFKA_CONSUMER_SESSION_TIMEOUT=30s

# Topics
KAFKA_TOPIC_GROUP_MSG=group_msg
KAFKA_TOPIC_OFFLINE_MSG=offline_msg
KAFKA_TOPIC_MEMBERSHIP_CHANGE=membership_change
```

#### Offline Worker Configuration
```bash
# Worker settings
OFFLINE_WORKER_ENABLED=true
OFFLINE_WORKER_BATCH_SIZE=100
OFFLINE_WORKER_BATCH_TIMEOUT=5s
OFFLINE_WORKER_CONCURRENCY=10
```

#### Sensitive Word Filter Configuration
```bash
# Filter settings
FILTER_ENABLED=true
FILTER_ACTION=replace          # block, replace, audit
FILTER_WORDLIST_PATH=/etc/im-service/wordlist.txt
```

#### Encryption Configuration
```bash
# Encryption
ENCRYPTION_ENABLED=true
KMS_ENDPOINT=kms:8080
KMS_KEY_ID=${KMS_KEY_ID}
DEK_CACHE_TTL=1h
KEY_ROTATION_DAYS=90
```

#### Observability Configuration
```bash
# Metrics
METRICS_ENABLED=true
METRICS_PORT=9090
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Configuration File

**config.yaml**:
```yaml
server:
  grpc_port: 9094
  http_port: 8094
  max_message_size: 10240
  max_retry_attempts: 3
  retry_timeout: 5s

database:
  host: mysql-master:3306
  name: im_chat
  user: im_service
  password: ${DB_PASSWORD}
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 1h

redis:
  addr: redis-master:6379
  password: ${REDIS_PASSWORD}
  db: 0
  pool_size: 100
  min_idle_conns: 10

kafka:
  brokers:
    - kafka-1:9092
    - kafka-2:9092
    - kafka-3:9092
  producer:
    timeout: 10s
    retry_max: 3
    batch_size: 16384
  consumer:
    group_id: im-service-offline-worker
    auto_offset_reset: latest
    session_timeout: 30s
  topics:
    group_msg: group_msg
    offline_msg: offline_msg
    membership_change: membership_change

offline_worker:
  enabled: true
  batch_size: 100
  batch_timeout: 5s
  concurrency: 10

filter:
  enabled: true
  action: replace
  wordlist_path: /etc/im-service/wordlist.txt

encryption:
  enabled: true
  kms_endpoint: kms:8080
  kms_key_id: ${KMS_KEY_ID}
  dek_cache_ttl: 1h
  key_rotation_days: 90

services:
  gateway:
    addr: im-gateway-service:9093
    timeout: 5s
  user:
    addr: user-service:9096
    timeout: 3s

observability:
  metrics_enabled: true
  metrics_port: 9090
  otel_endpoint: http://otel-collector:4317
  log_level: info
  log_format: json
```

## Deployment Methods

### 1. Docker Deployment

#### Build Image
```bash
cd apps/im-service
docker build -t im-service:latest .
```

#### Run Container
```bash
docker run -d \
  --name im-service \
  -p 9094:9094 \
  -p 8094:8094 \
  -e DB_HOST=mysql-master:3306 \
  -e DB_PASSWORD=${DB_PASSWORD} \
  -e REDIS_ADDR=redis-master:6379 \
  -e REDIS_PASSWORD=${REDIS_PASSWORD} \
  -e KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092 \
  -e ETCD_ENDPOINTS=etcd-1:2379,etcd-2:2379,etcd-3:2379 \
  -e GATEWAY_SERVICE_ADDR=im-gateway-service:9093 \
  -e USER_SERVICE_ADDR=user-service:9096 \
  im-service:latest
```

### 2. Kubernetes Deployment

#### Deployment Manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: im-service
  namespace: im-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: im-service
  template:
    metadata:
      labels:
        app: im-service
    spec:
      containers:
      - name: im-service
        image: im-service:latest
        ports:
        - containerPort: 9094
          name: grpc
        - containerPort: 8094
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
        - name: KAFKA_BROKERS
          value: "kafka-1:9092,kafka-2:9092,kafka-3:9092"
        - name: ETCD_ENDPOINTS
          value: "etcd-1:2379,etcd-2:2379,etcd-3:2379"
        - name: GATEWAY_SERVICE_ADDR
          value: "im-gateway-service:9093"
        - name: USER_SERVICE_ADDR
          value: "user-service:9096"
        resources:
          requests:
            cpu: "4"
            memory: "8Gi"
          limits:
            cpu: "8"
            memory: "16Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8094
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8094
          initialDelaySeconds: 10
          periodSeconds: 5
```

#### Service Manifest
```yaml
apiVersion: v1
kind: Service
metadata:
  name: im-service
  namespace: im-system
spec:
  type: ClusterIP
  selector:
    app: im-service
  ports:
  - name: grpc
    port: 9094
    targetPort: 9094
    protocol: TCP
  - name: http
    port: 8094
    targetPort: 8094
    protocol: TCP
```

### 3. Binary Deployment

#### Systemd Service
```ini
# /etc/systemd/system/im-service.service
[Unit]
Description=IM Service
After=network.target mysql.service redis.service

[Service]
Type=simple
User=im-service
Group=im-service
WorkingDirectory=/opt/im-service
ExecStart=/opt/im-service/bin/im-service
Restart=always
RestartSec=10

Environment="DB_HOST=mysql-master:3306"
Environment="DB_PASSWORD=${DB_PASSWORD}"
Environment="REDIS_ADDR=redis-master:6379"
Environment="KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092"

[Install]
WantedBy=multi-user.target
```

## Scaling Guidelines

### Horizontal Scaling

#### When to Scale Out
- Message routing latency > 200ms (P99)
- CPU usage > 70% sustained
- Kafka consumer lag > 10,000 messages
- Database connection pool exhausted

#### Scaling Process
```bash
kubectl scale deployment im-service --replicas=5 -n im-system
```

### Vertical Scaling

#### When to Scale Up
- Memory usage > 80%
- Database query performance degradation
- Frequent GC pauses

## Operational Runbooks

### Runbook 1: Handle High Kafka Lag

**Detection**: Kafka consumer lag > 10,000 messages

**Response**:
1. Check offline worker status
2. Increase worker concurrency
3. Scale IM Service replicas
4. Monitor lag reduction

### Runbook 2: Handle Database Connection Exhaustion

**Detection**: Database connection errors

**Response**:
1. Check active connections
2. Increase connection pool size
3. Identify slow queries
4. Optimize query performance

### Runbook 3: Handle Message Delivery Failures

**Detection**: High message delivery failure rate

**Response**:
1. Check Gateway Service health
2. Check Registry (etcd) connectivity
3. Verify Kafka broker health
4. Check message routing logs

## Troubleshooting

### Issue 1: High Message Latency

**Symptoms**: P99 latency > 200ms

**Solutions**:
1. Check database query performance
2. Verify Redis response times
3. Check Kafka producer lag
4. Optimize message routing logic

### Issue 2: Offline Worker Lag

**Symptoms**: Kafka consumer lag increasing

**Solutions**:
1. Increase worker concurrency
2. Increase batch size
3. Scale IM Service replicas
4. Check database write performance

### Issue 3: Sequence Number Gaps

**Symptoms**: Non-monotonic sequence numbers

**Solutions**:
1. Check Redis connectivity
2. Verify snapshot mechanism
3. Check for Redis failover events
4. Review sequence generator logs

## Health Checks

### Liveness Probe
```bash
curl http://localhost:8094/health
```

### Readiness Probe
```bash
curl http://localhost:8094/ready
```

### Metrics Endpoint
```bash
curl http://localhost:8094/metrics
```

## References

- [API Documentation](./API.md)
- [Integration Testing Guide](./integration_test/README.md)
- [Monitoring Guide](../../deploy/docker/MONITORING_SUMMARY.md)
