# IM Service

IM (Instant Messaging) Service provides core message routing and offline message persistence for the chat system.

## Architecture

The IM Service is a **unified service** that runs two components in a single process:

### 1. Message Router (gRPC Server)
- Exposes gRPC API for message routing (port 9094)
- Handles private and group message routing
- Assigns sequence numbers
- Determines Fast Path vs Slow Path delivery
- Runs on main goroutine

### 2. Offline Worker (Background Component)
- Consumes from Kafka `offline_msg` topic
- Persists offline messages to MySQL
- Performs deduplication using Redis
- Runs as background goroutine
- **Starts automatically when IM Service starts**

This integrated design simplifies deployment, reduces operational complexity, and ensures tight coordination between routing and persistence components.

## Internal Components

- **Registry**: User-to-gateway mapping with etcd backend
- **Sequence Generator**: Monotonic sequence number generation with Redis
- **Sequence Backup**: MySQL-based backup for sequence recovery
- **Deduplication**: Redis-based message deduplication
- **Offline Storage**: MySQL-based offline message persistence

## Quick Start

### Local Development

```bash
# Build the service
go build -o bin/im-service .

# Run with default configuration
./bin/im-service

# Run with custom configuration
export GRPC_PORT=9094
export HTTP_PORT=8080
export KAFKA_BROKERS=localhost:9092
export DB_HOST=localhost
export REDIS_ADDR=localhost:6379
export OFFLINE_WORKER_ENABLED=true
./bin/im-service
```

### Testing

```bash
# Fast unit tests (1 second) - recommended for development
./scripts/test-coverage.sh

# Full test suite including property tests (7 minutes)
./scripts/test-coverage.sh --with-property

# Or use make from root
make test APP=im
```

**Note**: Property-based tests are slow due to TTL waits. See [TESTING.md](TESTING.md) for details.

### Docker Deployment

```bash
# Build Docker image
docker build -t im-service:latest .

# Run with Docker Compose
cd deploy/docker
docker-compose -f docker-compose.infra.yml up -d
docker-compose -f docker-compose.services.yml up im-service
```

### Kubernetes Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deploy/k8s/services/im-service/

# Check status
kubectl get pods -l app=im-service
kubectl logs -f deployment/im-service

# Verify worker is running
kubectl logs -f deployment/im-service | grep "Offline worker"
```

## Component Details

### Registry Service
- Manages user-to-gateway mappings in etcd
- Supports multi-device connections
- 90-second TTL with heartbeat renewal
- Watch mechanism for cache invalidation

See [registry/README.md](registry/README.md) for details.

### Sequence Generator
- Generates monotonic sequence numbers using Redis INCR
- Supports private chat and group chat
- MySQL backup every 10,000 messages
- Recovery on Redis failure

See [sequence/README.md](sequence/README.md) for details.

### Offline Worker
- Consumes messages from Kafka `offline_msg` topic
- Batch processing (default: 100 messages or 5 seconds)
- Deduplication using Redis before database write
- Automatic retry with exponential backoff
- Manual offset commit after successful persistence

See [worker/README.md](worker/README.md) for details.

### Offline Storage
- MySQL-based persistent storage for offline messages
- Partitioned by user_id for scalability
- 7-day TTL with automatic cleanup
- Batch insert with transaction support

See [storage/README.md](storage/README.md) for details.

## Configuration

The service is configured via environment variables:

### gRPC Server
- `GRPC_PORT`: gRPC server port (default: 9094)

### HTTP Server
- `HTTP_PORT`: HTTP server port for health checks and metrics (default: 8080)

### Kafka
- `KAFKA_BROKERS`: Comma-separated Kafka broker addresses (default: localhost:9092)
- `KAFKA_CONSUMER_GROUP`: Consumer group name (default: im-service-offline-workers)
- `KAFKA_TOPIC`: Topic to consume from (default: offline_msg)

### Database
- `DB_HOST`: MySQL host (default: localhost)
- `DB_PORT`: MySQL port (default: 3306)
- `DB_USER`: MySQL username (default: im_service)
- `DB_PASSWORD`: MySQL password
- `DB_NAME`: Database name (default: im_chat)
- `DB_MAX_OPEN_CONNS`: Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS`: Max idle connections (default: 5)
- `DB_CONN_MAX_LIFETIME`: Connection max lifetime (default: 5m)

### Redis
- `REDIS_ADDR`: Redis address (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Redis database number (default: 2)

### Offline Worker
- `OFFLINE_WORKER_ENABLED`: Enable/disable offline worker (default: true)
- `BATCH_SIZE`: Batch size for processing (default: 100)
- `BATCH_TIMEOUT`: Batch timeout (default: 5s)
- `MAX_RETRIES`: Max retry attempts (default: 5)
- `RETRY_BACKOFF`: Retry backoff durations (default: 1s,2s,4s,8s,16s)
- `MESSAGE_TTL`: Message time-to-live (default: 168h = 7 days)

## Health Checks

The service exposes HTTP endpoints for monitoring:

- `GET /health`: Liveness probe (always returns 200 OK)
- `GET /ready`: Readiness probe (checks worker health)
- `GET /stats`: JSON statistics about worker performance
- `GET /metrics`: Prometheus-format metrics

## Read Receipt API

The service provides HTTP REST API for read receipt tracking with real-time delivery:

### Mark Message as Read
```bash
POST /api/v1/messages/read
Content-Type: application/json

{
  "msg_id": "550e8400-e29b-41d4-a716-446655440000",
  "reader_id": "user123",
  "sender_id": "user456",
  "conversation_id": "conv789",
  "conversation_type": "private",
  "device_id": "device-uuid"
}
```

**Real-Time Delivery**:
- When a message is marked as read, a read receipt event is published to Kafka topic `read_receipt_events`
- The Gateway Service consumes the event and pushes it to the sender via WebSocket
- All sender's devices receive the read receipt (multi-device sync)
- If the sender is offline, the receipt is stored for later retrieval

### Get Unread Message Count
```bash
GET /api/v1/messages/unread/count?user_id=user123
```

### Get Unread Messages
```bash
GET /api/v1/messages/unread?user_id=user123&limit=50&offset=0
```

### Get Read Receipts for Message
```bash
GET /api/v1/messages/receipts?msg_id=550e8400-e29b-41d4-a716-446655440000
```

### Mark Conversation as Read
```bash
POST /api/v1/conversations/read
Content-Type: application/json

{
  "user_id": "user123",
  "conversation_id": "conv789"
}
```

For detailed API documentation and read receipt delivery architecture, see:
- [Read Receipt Tracking](readreceipt/README.md)
- [Read Receipt Delivery](readreceipt/READ_RECEIPT_DELIVERY.md)

## Requirements Validated

### Message Routing
- **7.1**: Registry with 90-second TTL
- **7.2**: Lease renewal every 30 seconds
- **7.6**: etcd cluster (3 or 5 nodes)
- **7.9**: Watch mechanism for Registry changes
- **15.1**: Multi-device support
- **15.2**: Device ID in Registry
- **16.1**: Monotonic sequence numbers
- **16.2**: Redis-based sequence generation
- **16.6**: Conversation-specific sequences
- **16.7**: MySQL backup for sequences
- **17.3**: Watch-based cache invalidation

### Offline Message Persistence
- **3.9**: Deduplication before database write
- **3.10**: Skip duplicate messages
- **3.11**: Add to dedup set after successful write
- **4.2**: Consume from offline_msg topic
- **4.6**: Batch processing with manual offset commit
- **5.1**: MySQL storage with partitioning
- **5.2**: 7-day TTL for offline messages
- **5.3**: Automatic cleanup of expired messages

### Read Receipts
- **5.1**: Mark messages as read with timestamp
- **5.2**: Update message status to "read"
- **5.3**: Push read receipt to online sender via WebSocket
- **5.4**: Store read receipt for offline sender
- **15.4**: Read receipt sync across devices (multi-device support)
- **5.2**: Update message status to "read"
- **5.3**: Real-time read receipt delivery (via Gateway Service)
- **5.4**: Offline read receipt storage
- **5.5**: Support for private and group chat read receipts

## Test Coverage

- **Unit Tests**: 48 tests (fast, < 1 second)
- **Property Tests**: 14 tests (slow, ~7 minutes)
- **Total**: 62 tests validating correctness

## Scaling

### Horizontal Scaling
- Deploy multiple replicas of im-service
- Each replica runs both gRPC server and offline worker
- Kafka consumer group ensures load balancing across workers
- Example: 3 replicas = 3 gRPC servers + 3 Kafka consumers

### Resource Recommendations
- **CPU**: 500m per replica (shared between routing and worker)
- **Memory**: 512Mi per replica (shared between routing and worker)
- **Kafka Partitions**: 64 (allows up to 64 worker instances)

## Dependencies

- **etcd** v3.6.7: User-to-gateway registry
- **Redis**: Sequence generation and deduplication
- **MySQL**: Sequence backup and offline message storage
- **Kafka**: Message queue for offline messages
- **pgregory.net/rapid**: Property-based testing

## Infrastructure

The IM Service depends on the following infrastructure components:

### etcd Cluster (3 nodes)
- Distributed registry for user-to-gateway mappings
- TTL: 90 seconds (auto-cleanup)
- Lease renewal: 30 seconds

### MySQL
- Persistent storage for offline messages, users, and groups
- Max connections: 500
- Connection pooling: 25 max open, 5 max idle
- Partitioning: 16 partitions by user_id hash

### Redis
- Deduplication, caching, and sequence number generation
- Persistence: AOF + RDB
- Max memory: 2GB
- Eviction policy: allkeys-lru
- TTL for deduplication: 7 days

### Kafka Cluster (3 brokers, KRaft mode)
- Message bus for group messages and offline message queue
- Replication factor: 3
- Min in-sync replicas: 2
- Compression: snappy
- Acks: all (wait for all replicas)

### Database Schema

**Tables:**
1. **offline_messages**: Stores offline messages with 7-day retention
   - Partitioned by user_id (16 partitions)
   - Indexed by user_id, timestamp, conversation_id, sequence_number
   - Added `read_at` column for read receipt tracking

2. **read_receipts**: Tracks read receipts for messages
   - Stores msg_id, reader_id, sender_id, read_at timestamp
   - Supports multi-device read tracking with device_id
   - Unique constraint on (msg_id, reader_id, device_id)

3. **groups**: Group metadata
   - Stores group name, creator, member count

4. **group_members**: Group membership
   - Many-to-many relationship between users and groups
   - Supports roles: owner, admin, member

5. **sequence_snapshots**: Backup for Redis sequence numbers
   - Periodic snapshots every 10,000 messages
   - Used for disaster recovery

6. **users**: User profiles
   - Stores username, email, display name, avatar

### Kafka Topics

| Topic | Partitions | Replication | Retention | Purpose |
|-------|------------|-------------|-----------|---------|
| group_msg | 32 | 3 | 1 hour | Group message broadcast |
| offline_msg | 64 | 3 | 7 days | Offline message queue |
| membership_change | 16 | 3 | 1 hour | Group membership events |

### Infrastructure Endpoints

| Service | Host | Port | Credentials |
|---------|------|------|-------------|
| etcd-1 | localhost | 2379, 2380 | - |
| etcd-2 | localhost | 2381, 2382 | - |
| etcd-3 | localhost | 2383, 2384 | - |
| MySQL | localhost | 3307 | user: `im_service`<br>password: `im_service_password`<br>database: `im_chat` |
| Redis | localhost | 6380 | - |
| Kafka-1 | localhost | 9093 | - |
| Kafka-2 | localhost | 9094 | - |
| Kafka-3 | localhost | 9095 | - |

### Database Migrations

The IM Service uses Liquibase for database schema version control. Migrations are located in `migrations/` directory.

**Apply Migrations:**
```bash
docker compose -f deploy/docker/docker-compose.infra.yml up liquibase
```

**Migration Files Structure:**
```
apps/im-service/migrations/
├── liquibase.properties
├── changelog/
│   ├── db.changelog-master.yaml
│   └── v1.0/
│       ├── 001-initial-schema.yaml
│       └── 002-sample-data.yaml
```

For detailed infrastructure setup and troubleshooting, see the [Infrastructure Guide](../../deploy/docker/README.md).

## Migration from Separate Worker

If you previously deployed offline-worker as a separate service:

1. Stop the separate offline-worker deployment
2. Update im-service to latest version (includes integrated worker)
3. Deploy updated im-service
4. Verify worker is running via logs or `/stats` endpoint
5. Remove obsolete offline-worker deployment files

See [.kiro/specs/im-chat-system/ARCHITECTURE_CLARIFICATION.md](../../.kiro/specs/im-chat-system/ARCHITECTURE_CLARIFICATION.md) for detailed architecture explanation.
