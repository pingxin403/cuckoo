# IM Gateway Service - Deployment Guide

## Overview

This guide covers deploying the IM Gateway Service in various environments (development, staging, production).

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
- Storage: 20GB SSD
- Network: 1 Gbps
- OS: Linux (Ubuntu 20.04+, CentOS 8+)

**Dependencies**:
- etcd: 1 node
- Redis: 1 node
- Kafka: 1 broker
- MySQL: 1 instance

### Production Requirements

**Per Gateway Node**:
- CPU: 16 cores (3.0 GHz+)
- Memory: 32GB RAM
- Storage: 100GB SSD
- Network: 10 Gbps
- OS: Linux (Ubuntu 20.04+, CentOS 8+)

**Cluster Configuration**:
- Gateway Nodes: 3+ (for high availability)
- Load Balancer: 2+ (active-passive or active-active)
- etcd Cluster: 3 nodes
- Redis Cluster: 3 nodes (master + 2 replicas)
- Kafka Cluster: 3+ brokers
- MySQL: Master-replica setup

**Network Requirements**:
- Low latency between Gateway and etcd (< 5ms)
- Low latency between Gateway and Redis (< 2ms)
- Sufficient bandwidth for WebSocket traffic (estimate: 10KB/connection/min)

### System Configuration

**File Descriptor Limits**:
```bash
# /etc/security/limits.conf
* soft nofile 1000000
* hard nofile 1000000

# Verify
ulimit -n
```

**Kernel Parameters**:
```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1

# Apply changes
sudo sysctl -p
```

## Service Dependencies

### Required Services

#### 1. etcd (Service Registry)
- **Purpose**: User-to-Gateway mapping, service discovery
- **Version**: 3.5+
- **Endpoints**: `etcd-1:2379,etcd-2:2379,etcd-3:2379`
- **Configuration**:
  ```yaml
  ETCD_ENDPOINTS: "etcd-1:2379,etcd-2:2379,etcd-3:2379"
  ETCD_DIAL_TIMEOUT: "5s"
  ETCD_REQUEST_TIMEOUT: "3s"
  ```

#### 2. Redis (Caching & Deduplication)
- **Purpose**: User mapping cache, message deduplication
- **Version**: 6.0+
- **Endpoints**: `redis-master:6379`
- **Configuration**:
  ```yaml
  REDIS_ADDR: "redis-master:6379"
  REDIS_PASSWORD: "${REDIS_PASSWORD}"
  REDIS_DB: 0
  REDIS_POOL_SIZE: 100
  REDIS_MIN_IDLE_CONNS: 10
  ```

#### 3. Kafka (Message Bus)
- **Purpose**: Group message distribution, offline messages
- **Version**: 2.8+
- **Brokers**: `kafka-1:9092,kafka-2:9092,kafka-3:9092`
- **Topics**: `group_msg`, `offline_msg`, `membership_change`
- **Configuration**:
  ```yaml
  KAFKA_BROKERS: "kafka-1:9092,kafka-2:9092,kafka-3:9092"
  KAFKA_GROUP_ID: "im-gateway-group"
  KAFKA_AUTO_OFFSET_RESET: "latest"
  ```

#### 4. Auth Service (Authentication)
- **Purpose**: JWT token validation
- **Version**: v1.0+
- **Endpoint**: `auth-service:9095`
- **Configuration**:
  ```yaml
  AUTH_SERVICE_ADDR: "auth-service:9095"
  AUTH_SERVICE_TIMEOUT: "3s"
  ```

#### 5. User Service (User Profiles)
- **Purpose**: User profile lookup, group membership
- **Version**: v1.0+
- **Endpoint**: `user-service:9096`
- **Configuration**:
  ```yaml
  USER_SERVICE_ADDR: "user-service:9096"
  USER_SERVICE_TIMEOUT: "3s"
  ```

#### 6. IM Service (Message Routing)
- **Purpose**: Message routing and delivery
- **Version**: v1.0+
- **Endpoint**: `im-service:9094`
- **Configuration**:
  ```yaml
  IM_SERVICE_ADDR: "im-service:9094"
  IM_SERVICE_TIMEOUT: "5s"
  ```

### Optional Services

#### 1. Prometheus (Metrics)
- **Purpose**: Metrics collection and alerting
- **Endpoint**: `prometheus:9090`

#### 2. Grafana (Dashboards)
- **Purpose**: Metrics visualization
- **Endpoint**: `grafana:3000`

#### 3. Loki (Logging)
- **Purpose**: Centralized logging
- **Endpoint**: `loki:3100`

## Configuration Parameters

### Environment Variables

#### Server Configuration
```bash
# Server ports
SERVER_GRPC_PORT=9093          # gRPC port for internal communication
SERVER_WS_PORT=8080            # WebSocket port for client connections
SERVER_HTTP_PORT=8081          # HTTP port for health checks

# Server limits
MAX_CONNECTIONS=100000         # Maximum concurrent connections
READ_BUFFER_SIZE=4096          # WebSocket read buffer size (bytes)
WRITE_BUFFER_SIZE=4096         # WebSocket write buffer size (bytes)
```

#### WebSocket Configuration
```bash
# Connection timeouts
WS_HANDSHAKE_TIMEOUT=10s       # WebSocket handshake timeout
WS_READ_TIMEOUT=90s            # Read timeout (must be > heartbeat interval)
WS_WRITE_TIMEOUT=10s           # Write timeout
WS_PING_INTERVAL=30s           # Heartbeat ping interval
WS_PONG_TIMEOUT=60s            # Pong response timeout
```

#### Registry Configuration
```bash
# etcd settings
REGISTRY_TTL=90s               # User registration TTL
REGISTRY_RENEW_INTERVAL=30s    # Lease renewal interval
REGISTRY_PREFIX=/registry/users/
```

#### Cache Configuration
```bash
# Cache TTLs
CACHE_USER_MAPPING_TTL=5m      # User-to-gateway mapping cache TTL
CACHE_GROUP_MEMBERSHIP_TTL=10m # Group membership cache TTL
CACHE_LARGE_GROUP_THRESHOLD=1000  # Threshold for large group optimization
```

#### Performance Configuration
```bash
# Go runtime
GOMAXPROCS=16                  # Number of CPU cores to use
GOMEMLIMIT=30GiB              # Memory limit

# Connection pooling
DB_MAX_OPEN_CONNS=100         # Maximum database connections
DB_MAX_IDLE_CONNS=10          # Idle database connections
GRPC_MAX_CONN_AGE=30m         # gRPC connection max age
```

#### Observability Configuration
```bash
# Metrics
METRICS_ENABLED=true
METRICS_PORT=9090
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317

# Logging
LOG_LEVEL=info                 # debug, info, warn, error
LOG_FORMAT=json                # json or text
```

### Configuration File

**config.yaml**:
```yaml
server:
  grpc_port: 9093
  ws_port: 8080
  http_port: 8081
  max_connections: 100000
  read_buffer_size: 4096
  write_buffer_size: 4096

websocket:
  handshake_timeout: 10s
  read_timeout: 90s
  write_timeout: 10s
  ping_interval: 30s
  pong_timeout: 60s

registry:
  endpoints:
    - etcd-1:2379
    - etcd-2:2379
    - etcd-3:2379
  ttl: 90s
  renew_interval: 30s
  prefix: /registry/users/

cache:
  redis_addr: redis-master:6379
  user_mapping_ttl: 5m
  group_membership_ttl: 10m
  large_group_threshold: 1000

services:
  auth:
    addr: auth-service:9095
    timeout: 3s
  user:
    addr: user-service:9096
    timeout: 3s
  im:
    addr: im-service:9094
    timeout: 5s

kafka:
  brokers:
    - kafka-1:9092
    - kafka-2:9092
    - kafka-3:9092
  group_id: im-gateway-group
  topics:
    group_msg: group_msg
    membership_change: membership_change

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
cd apps/im-gateway-service
docker build -t im-gateway-service:latest .
```

#### Run Container
```bash
docker run -d \
  --name im-gateway \
  -p 8080:8080 \
  -p 9093:9093 \
  -p 8081:8081 \
  -e ETCD_ENDPOINTS=etcd-1:2379,etcd-2:2379,etcd-3:2379 \
  -e REDIS_ADDR=redis-master:6379 \
  -e KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092 \
  -e AUTH_SERVICE_ADDR=auth-service:9095 \
  -e USER_SERVICE_ADDR=user-service:9096 \
  -e IM_SERVICE_ADDR=im-service:9094 \
  --ulimit nofile=1000000:1000000 \
  im-gateway-service:latest
```

#### Docker Compose
```yaml
version: '3.8'

services:
  im-gateway:
    image: im-gateway-service:latest
    ports:
      - "8080:8080"
      - "9093:9093"
      - "8081:8081"
    environment:
      - ETCD_ENDPOINTS=etcd-1:2379,etcd-2:2379,etcd-3:2379
      - REDIS_ADDR=redis-master:6379
      - KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
      - AUTH_SERVICE_ADDR=auth-service:9095
      - USER_SERVICE_ADDR=user-service:9096
      - IM_SERVICE_ADDR=im-service:9094
      - GOMAXPROCS=16
      - GOMEMLIMIT=30GiB
    ulimits:
      nofile:
        soft: 1000000
        hard: 1000000
    deploy:
      resources:
        limits:
          cpus: '16'
          memory: 32G
        reservations:
          cpus: '8'
          memory: 16G
    depends_on:
      - etcd
      - redis
      - kafka
      - auth-service
      - user-service
      - im-service
```

### 2. Kubernetes Deployment

#### Deployment Manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: im-gateway-service
  namespace: im-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: im-gateway-service
  template:
    metadata:
      labels:
        app: im-gateway-service
    spec:
      containers:
      - name: im-gateway
        image: im-gateway-service:latest
        ports:
        - containerPort: 8080
          name: websocket
        - containerPort: 9093
          name: grpc
        - containerPort: 8081
          name: http
        env:
        - name: ETCD_ENDPOINTS
          value: "etcd-1:2379,etcd-2:2379,etcd-3:2379"
        - name: REDIS_ADDR
          value: "redis-master:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: password
        - name: KAFKA_BROKERS
          value: "kafka-1:9092,kafka-2:9092,kafka-3:9092"
        - name: AUTH_SERVICE_ADDR
          value: "auth-service:9095"
        - name: USER_SERVICE_ADDR
          value: "user-service:9096"
        - name: IM_SERVICE_ADDR
          value: "im-service:9094"
        - name: GOMAXPROCS
          value: "16"
        - name: GOMEMLIMIT
          value: "30GiB"
        resources:
          requests:
            cpu: "8"
            memory: "16Gi"
          limits:
            cpu: "16"
            memory: "32Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 5
```

#### Service Manifest
```yaml
apiVersion: v1
kind: Service
metadata:
  name: im-gateway-service
  namespace: im-system
spec:
  type: LoadBalancer
  selector:
    app: im-gateway-service
  ports:
  - name: websocket
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: grpc
    port: 9093
    targetPort: 9093
    protocol: TCP
  - name: http
    port: 8081
    targetPort: 8081
    protocol: TCP
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3 hours
```

#### Deploy to Kubernetes
```bash
# Create namespace
kubectl create namespace im-system

# Create secrets
kubectl create secret generic redis-secret \
  --from-literal=password=your-redis-password \
  -n im-system

# Apply manifests
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# Verify deployment
kubectl get pods -n im-system
kubectl get svc -n im-system
```

### 3. Binary Deployment

#### Build Binary
```bash
cd apps/im-gateway-service
go build -o bin/im-gateway-service main.go
```

#### Create Systemd Service
```ini
# /etc/systemd/system/im-gateway.service
[Unit]
Description=IM Gateway Service
After=network.target

[Service]
Type=simple
User=im-gateway
Group=im-gateway
WorkingDirectory=/opt/im-gateway
ExecStart=/opt/im-gateway/bin/im-gateway-service
Restart=always
RestartSec=10
LimitNOFILE=1000000

Environment="ETCD_ENDPOINTS=etcd-1:2379,etcd-2:2379,etcd-3:2379"
Environment="REDIS_ADDR=redis-master:6379"
Environment="KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092"
Environment="AUTH_SERVICE_ADDR=auth-service:9095"
Environment="USER_SERVICE_ADDR=user-service:9096"
Environment="IM_SERVICE_ADDR=im-service:9094"
Environment="GOMAXPROCS=16"
Environment="GOMEMLIMIT=30GiB"

[Install]
WantedBy=multi-user.target
```

#### Start Service
```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable im-gateway

# Start service
sudo systemctl start im-gateway

# Check status
sudo systemctl status im-gateway

# View logs
sudo journalctl -u im-gateway -f
```

## Scaling Guidelines

### Horizontal Scaling

#### When to Scale Out
- Active connections > 80K per node
- CPU usage > 70% sustained
- Memory usage > 80%
- P99 latency > 200ms
- Connection errors > 1%

#### Scaling Process
1. **Add New Gateway Node**:
   ```bash
   kubectl scale deployment im-gateway-service --replicas=5 -n im-system
   ```

2. **Verify Load Distribution**:
   ```bash
   # Check connection distribution
   kubectl exec -it im-gateway-pod-1 -n im-system -- \
     curl localhost:8081/metrics | grep active_connections
   ```

3. **Monitor Rebalancing**:
   - New connections automatically route to new nodes
   - Existing connections remain on current nodes
   - Users reconnect naturally over time (heartbeat failures, network issues)

#### Connection Rebalancing
- **Passive Rebalancing**: Wait for natural reconnections (recommended)
- **Active Rebalancing**: Gradually close connections on overloaded nodes
  ```bash
  # Drain node gracefully
  kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data
  ```

### Vertical Scaling

#### When to Scale Up
- Memory per connection > 10KB
- CPU bottleneck on message processing
- Network bandwidth saturation

#### Scaling Process
1. Update resource limits in deployment
2. Rolling update to apply changes
3. Monitor performance improvements

### Auto-Scaling

#### Horizontal Pod Autoscaler (HPA)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: im-gateway-hpa
  namespace: im-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: im-gateway-service
  minReplicas: 3
  maxReplicas: 100
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: active_connections
      target:
        type: AverageValue
        averageValue: "80000"
```

## Operational Runbooks

### Runbook 1: Handle Gateway Node Failure

**Scenario**: A Gateway node becomes unresponsive

**Detection**:
- Health check failures
- Prometheus alert: `GatewayNodeDown`
- Increased connection errors

**Response**:
1. **Verify Node Status**:
   ```bash
   kubectl get pods -n im-system | grep im-gateway
   kubectl describe pod im-gateway-pod-1 -n im-system
   ```

2. **Check Logs**:
   ```bash
   kubectl logs im-gateway-pod-1 -n im-system --tail=100
   ```

3. **Restart Pod** (if needed):
   ```bash
   kubectl delete pod im-gateway-pod-1 -n im-system
   ```

4. **Verify Recovery**:
   - Check new pod is running
   - Verify connections are rebalancing
   - Monitor error rates

**Impact**:
- Users on failed node will reconnect automatically
- Reconnection time: 30-90 seconds (heartbeat timeout)
- No message loss (messages queued in Kafka/offline storage)

### Runbook 2: Scale Cluster Up

**Scenario**: Need to handle increased load

**Steps**:
1. **Determine Target Replicas**:
   ```
   Target Replicas = Total Connections / 80,000
   ```

2. **Scale Deployment**:
   ```bash
   kubectl scale deployment im-gateway-service --replicas=10 -n im-system
   ```

3. **Verify New Pods**:
   ```bash
   kubectl get pods -n im-system -w
   ```

4. **Monitor Load Distribution**:
   ```bash
   # Check connections per pod
   for pod in $(kubectl get pods -n im-system -l app=im-gateway-service -o name); do
     echo "$pod:"
     kubectl exec -n im-system $pod -- curl -s localhost:8081/metrics | grep active_connections
   done
   ```

5. **Verify Performance**:
   - Check P99 latency < 200ms
   - Check CPU usage < 70%
   - Check memory usage < 80%

### Runbook 3: Scale Cluster Down

**Scenario**: Reduce capacity during low traffic

**Steps**:
1. **Identify Pods to Remove**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service
   ```

2. **Drain Pod Gracefully**:
   ```bash
   # Mark pod for deletion (stops accepting new connections)
   kubectl delete pod im-gateway-pod-5 -n im-system --grace-period=300
   ```

3. **Monitor Connection Drain**:
   ```bash
   # Watch active connections decrease
   kubectl exec -n im-system im-gateway-pod-5 -- \
     watch -n 5 'curl -s localhost:8081/metrics | grep active_connections'
   ```

4. **Scale Down Deployment**:
   ```bash
   kubectl scale deployment im-gateway-service --replicas=5 -n im-system
   ```

### Runbook 4: Investigate Message Delivery Issues

**Scenario**: Users report messages not being delivered

**Investigation**:
1. **Check Gateway Metrics**:
   ```promql
   # Message delivery success rate
   rate(im_gateway_messages_delivered_total[5m]) / 
   rate(im_gateway_messages_sent_total[5m])
   
   # Message delivery latency
   histogram_quantile(0.99, rate(im_gateway_message_latency_bucket[5m]))
   ```

2. **Check Service Dependencies**:
   ```bash
   # Auth Service
   curl http://auth-service:9095/health
   
   # User Service
   curl http://user-service:9096/health
   
   # IM Service
   curl http://im-service:9094/health
   ```

3. **Check Infrastructure**:
   ```bash
   # etcd health
   etcdctl endpoint health --cluster
   
   # Redis health
   redis-cli ping
   
   # Kafka health
   kafka-topics --bootstrap-server kafka-1:9092 --list
   ```

4. **Check Logs**:
   ```bash
   # Gateway logs
   kubectl logs -n im-system -l app=im-gateway-service --tail=100 | grep ERROR
   
   # IM Service logs
   kubectl logs -n im-system -l app=im-service --tail=100 | grep ERROR
   ```

5. **Common Issues**:
   - **Registry lookup failures**: Check etcd connectivity
   - **Message routing failures**: Check IM Service health
   - **Kafka publish failures**: Check Kafka broker health
   - **Authentication failures**: Check Auth Service health

## Troubleshooting

### Issue 1: High Connection Failure Rate

**Symptoms**:
- Connection success rate < 95%
- Many WebSocket handshake failures

**Diagnosis**:
```bash
# Check connection errors
kubectl logs -n im-system im-gateway-pod-1 | grep "connection error"

# Check system limits
kubectl exec -n im-system im-gateway-pod-1 -- ulimit -n

# Check load balancer
kubectl describe svc im-gateway-service -n im-system
```

**Solutions**:
1. Increase file descriptor limits
2. Check load balancer configuration
3. Verify network connectivity
4. Check Auth Service availability

### Issue 2: High Memory Usage

**Symptoms**:
- Memory usage > 90%
- OOM kills

**Diagnosis**:
```bash
# Check memory usage
kubectl top pod -n im-system -l app=im-gateway-service

# Check connection count
kubectl exec -n im-system im-gateway-pod-1 -- \
  curl localhost:8081/metrics | grep active_connections

# Memory per connection
Memory per connection = Total Memory / Active Connections
```

**Solutions**:
1. Verify memory per connection < 10KB
2. Check for memory leaks (use pprof)
3. Reduce cache TTLs
4. Scale horizontally

### Issue 3: High Latency

**Symptoms**:
- P99 latency > 200ms
- Slow message delivery

**Diagnosis**:
```bash
# Check latency metrics
kubectl exec -n im-system im-gateway-pod-1 -- \
  curl localhost:8081/metrics | grep message_latency

# Check CPU usage
kubectl top pod -n im-system -l app=im-gateway-service

# Check network latency
kubectl exec -n im-system im-gateway-pod-1 -- ping etcd-1
kubectl exec -n im-system im-gateway-pod-1 -- ping redis-master
```

**Solutions**:
1. Check CPU usage (should be < 80%)
2. Verify network latency to dependencies
3. Check Redis/etcd response times
4. Optimize message routing logic
5. Scale horizontally

### Issue 4: Connection Drops

**Symptoms**:
- Frequent disconnections
- High reconnection rate

**Diagnosis**:
```bash
# Check disconnection logs
kubectl logs -n im-system im-gateway-pod-1 | grep "connection closed"

# Check heartbeat configuration
kubectl exec -n im-system im-gateway-pod-1 -- \
  env | grep WS_PING_INTERVAL
```

**Solutions**:
1. Increase heartbeat interval
2. Check load balancer timeout settings
3. Verify network stability
4. Check Registry TTL configuration

## Health Checks

### Liveness Probe
```bash
curl http://localhost:8081/health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-25T10:00:00Z"
}
```

### Readiness Probe
```bash
curl http://localhost:8081/ready
```

**Response**:
```json
{
  "status": "ready",
  "dependencies": {
    "etcd": "healthy",
    "redis": "healthy",
    "kafka": "healthy",
    "auth_service": "healthy",
    "user_service": "healthy",
    "im_service": "healthy"
  }
}
```

### Metrics Endpoint
```bash
curl http://localhost:8081/metrics
```

**Key Metrics**:
- `im_gateway_active_connections`: Current active connections
- `im_gateway_messages_sent_total`: Total messages sent
- `im_gateway_messages_delivered_total`: Total messages delivered
- `im_gateway_message_latency_bucket`: Message delivery latency histogram
- `im_gateway_connection_errors_total`: Total connection errors

## Security Considerations

### TLS Configuration
- Use TLS 1.3 for WebSocket connections
- Configure certificate rotation
- Enforce TLS-only connections

### Authentication
- Validate JWT tokens on every connection
- Extract user_id and device_id from tokens
- Enforce max 5 devices per user

### Network Security
- Use network policies to restrict traffic
- Whitelist IP ranges for admin endpoints
- Enable DDoS protection on load balancer

## Monitoring and Alerting

### Key Metrics to Monitor
1. **Connection Metrics**:
   - Active connections
   - Connection success rate
   - Connection errors

2. **Message Metrics**:
   - Message throughput
   - Message latency (P50, P95, P99)
   - Message delivery success rate

3. **System Metrics**:
   - CPU usage
   - Memory usage
   - Network bandwidth
   - File descriptors

4. **Dependency Metrics**:
   - etcd response time
   - Redis response time
   - Kafka lag
   - Service availability

### Alert Thresholds
- P99 latency > 500ms for 5 minutes (P1)
- Message loss rate > 0.01% (Circuit Breaker)
- ACK timeout rate > 5% (P2)
- Active connections > 90K per node (Warning)
- CPU usage > 80% for 10 minutes (Warning)
- Memory usage > 90% (Critical)

## References

- [Load Testing Guide](./load_test/LOAD_TEST_GUIDE.md)
- [Integration Testing Guide](./integration_test/README.md)
- [Monitoring Guide](../../deploy/docker/MONITORING_SUMMARY.md)
- [Alerting Guide](../../deploy/docker/ALERTING_GUIDE.md)
