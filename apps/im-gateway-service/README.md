# IM Gateway Service

The IM Gateway Service is a stateless WebSocket gateway that manages real-time connections for the IM Chat System. It handles 100K+ concurrent connections per node and routes messages between clients and the IM Service.

## Architecture

### Core Components

1. **GatewayService**: Main service managing WebSocket connections
2. **PushService**: Handles pushing messages from IM Service to clients
3. **KafkaConsumer**: Consumes group messages from Kafka and broadcasts to local clients
4. **CacheManager**: Manages local caches for user-to-gateway mappings and group membership

### Key Features

- ✅ WebSocket connection management (100K+ concurrent connections)
- ✅ JWT authentication via Auth Service
- ✅ User registration in etcd Registry
- ✅ Heartbeat mechanism (30s ping, 90s timeout)
- ✅ Message routing to IM Service
- ✅ Multi-device support
- ✅ Rate limiting (100 messages/second per user)
- ✅ Local caching with TTL and Watch-based invalidation
- ✅ Kafka consumer for group message broadcast
- ✅ Graceful shutdown with connection notification

## Message Flow

### Client → Server (Send Message)

```
Client → WebSocket → Gateway → IM Service → (Registry/Kafka)
```

1. Client sends message via WebSocket
2. Gateway validates and rate-limits
3. Gateway forwards to IM Service for routing
4. IM Service determines delivery path (online/offline)
5. Gateway sends ACK back to client

### Server → Client (Receive Message)

```
IM Service → Gateway (gRPC) → WebSocket → Client
```

1. IM Service calls PushMessage gRPC
2. Gateway finds user's WebSocket connection(s)
3. Gateway pushes message to all devices
4. Gateway waits for delivery ACK (5s timeout)
5. Gateway returns delivery status to IM Service

### Group Messages

```
IM Service → Kafka → Gateway (Consumer) → WebSocket → Clients
```

1. IM Service publishes to Kafka `group_msg` topic
2. All Gateway nodes consume the message
3. Each Gateway filters for locally-connected members
4. Gateway pushes to local WebSocket connections

## Configuration

### GatewayConfig

```go
type GatewayConfig struct {
    // Connection settings
    HeartbeatInterval time.Duration // Default: 30s
    HeartbeatTimeout  time.Duration // Default: 90s
    ReadBufferSize    int           // Default: 4096
    WriteBufferSize   int           // Default: 4096
    
    // Message settings
    MaxMessageSize    int64         // Default: 10KB
    WriteWait         time.Duration // Default: 10s
    PongWait          time.Duration // Default: 60s
    PingPeriod        time.Duration // Default: 54s
    
    // Registry settings
    RegistryTTL       time.Duration // Default: 90s
    RegistryRenewInterval time.Duration // Default: 30s
    
    // Rate limiting
    MaxMessagesPerSecond int // Default: 100
}
```

### KafkaConfig

```go
type KafkaConfig struct {
    Brokers       []string
    GroupID       string
    Topic         string
    MinBytes      int
    MaxBytes      int
    CommitInterval time.Duration
}
```

## WebSocket Protocol

### Client Messages

```json
{
  "type": "send_msg",
  "msg_id": "uuid",
  "recipient": "user_123" or "group_456",
  "content": "Hello, world!",
  "timestamp": 1234567890
}
```

```json
{
  "type": "ack",
  "msg_id": "uuid"
}
```

```json
{
  "type": "heartbeat"
}
```

### Server Messages

```json
{
  "type": "message",
  "msg_id": "uuid",
  "sender": "user_123",
  "content": "Hello, world!",
  "timestamp": 1234567890,
  "sequence_number": 12345
}
```

```json
{
  "type": "ack",
  "msg_id": "uuid",
  "sequence_number": 12345,
  "timestamp": 1234567890
}
```

```json
{
  "type": "error",
  "error_code": "RATE_LIMIT_EXCEEDED",
  "error_message": "Too many messages",
  "timestamp": 1234567890
}
```

## Connection Lifecycle

1. **Connect**: Client initiates WebSocket connection with JWT token
2. **Authenticate**: Gateway validates token via Auth Service
3. **Register**: Gateway registers user in etcd Registry
4. **Active**: Connection maintained with heartbeat (ping/pong)
5. **Disconnect**: Connection closed, user unregistered from Registry

## Caching Strategy

### User-to-Gateway Cache

- **TTL**: 5 minutes
- **Invalidation**: etcd Watch on `/registry/users/` prefix
- **Purpose**: Fast lookup for message routing

### Group Membership Cache

- **TTL**: 5 minutes
- **Invalidation**: Membership change events via Kafka
- **Small groups (<1,000)**: Cache all members
- **Large groups (>1,000)**: Cache only locally-connected members

## Scalability

### Horizontal Scaling

- Stateless design (all state in etcd)
- Add more Gateway nodes to handle more connections
- Each node handles 100K+ concurrent connections
- Target: 8KB memory per connection

### Load Balancing

- Use Higress/Envoy for WebSocket load balancing
- Sticky sessions not required (clients can reconnect to any node)
- Registry maintains user-to-gateway mappings

## Monitoring

### Key Metrics

- Active connection count
- Message throughput (messages/second)
- Delivery latency (P50, P95, P99)
- ACK timeout rate
- Cache hit rate
- Kafka consumer lag

## Dependencies

- **Auth Service**: JWT token validation
- **IM Service**: Message routing
- **User Service**: Group membership queries
- **etcd**: User registry
- **Redis**: Distributed cache
- **Kafka**: Group message broadcast
- **MySQL**: (indirect) Offline message storage

## Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: im-gateway-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: im-gateway-service
        image: im-gateway-service:latest
        ports:
        - containerPort: 9093  # gRPC
        - containerPort: 8080  # WebSocket
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Environment Variables

- `AUTH_SERVICE_ADDR`: Auth Service gRPC address
- `IM_SERVICE_ADDR`: IM Service gRPC address
- `USER_SERVICE_ADDR`: User Service gRPC address
- `ETCD_ENDPOINTS`: etcd cluster endpoints
- `REDIS_ADDR`: Redis address
- `KAFKA_BROKERS`: Kafka broker addresses
- `GATEWAY_NODE_ID`: Unique node identifier

## Testing

### Unit Tests

```bash
make test APP=im-gateway-service
```

### Property-Based Tests

```bash
go test -tags=property ./service
```

### Integration Tests

```bash
./scripts/run-integration-tests.sh
```

## TODO

- [ ] Implement proper origin checking for WebSocket upgrade
- [ ] Add Bloom filter for large group membership checks
- [ ] Implement metrics collection (Prometheus)
- [ ] Add distributed tracing (OpenTelemetry)
- [ ] Implement connection pooling for gRPC clients
- [ ] Add circuit breaker for external service calls
- [ ] Implement backpressure handling
- [ ] Add comprehensive logging

## References

- [Requirements Document](../../.kiro/specs/im-chat-system/requirements.md)
- [Design Document](../../.kiro/specs/im-chat-system/design.md)
- [Task List](../../.kiro/specs/im-chat-system/tasks.md)
