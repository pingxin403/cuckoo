# Flash Sale Service (秒杀服务)

High-concurrency flash sale (seckill) system built with Spring Boot, Redis, Kafka, and MySQL.

## Overview

This service handles flash sale scenarios with extreme concurrency requirements:
- **100K+ QPS** peak traffic handling
- **Atomic inventory management** using Redis Lua scripts
- **Traffic shaping** via Kafka message queue
- **Multi-layer anti-fraud** and rate limiting

## Architecture

The service follows the "Three-Layer Funnel Model":

```
┌─────────────────────────────────────────────────────────────┐
│                    Layer 1: Anti-Fraud                       │
│  - L1: Gateway IP rate limiting (10 QPS/IP)                 │
│  - L2: User behavior analysis                               │
│  - L3: Device fingerprint risk assessment                   │
│  Blocks 90%+ of bot/fraudulent traffic                      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Layer 2: Queue                            │
│  - Token bucket rate limiting                               │
│  - User-friendly queue experience                           │
│  - Estimated wait time calculation                          │
│  Controls entry rate to protect backend                     │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Layer 3: Inventory                        │
│  - Redis Lua atomic stock deduction                         │
│  - Strong consistency guarantee                             │
│  - Zero overselling                                         │
│  50K+ QPS per Redis instance                                │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Async Processing                          │
│  - Kafka message queue for order creation                   │
│  - Batch database writes (100 records/batch)                │
│  - Dead letter queue for failed messages                    │
│  Converts 100K QPS to 2K TPS database writes                │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Framework**: Spring Boot 3.x, Java 17+
- **Cache**: Redis 7.x with Lua scripts
- **Message Queue**: Kafka (KRaft mode)
- **Database**: MySQL 8.x
- **Service Discovery**: etcd
- **Observability**: Prometheus, Grafana, Jaeger

## Quick Start

### Prerequisites

Start the infrastructure services:

```bash
# From project root
docker compose -f deploy/docker/docker-compose.infra.yml up -d
```

### Initialize Database

Create the flash_sale database and user:

```sql
CREATE DATABASE IF NOT EXISTS flash_sale;
CREATE USER IF NOT EXISTS 'flash_sale_user'@'%' IDENTIFIED BY 'flash_sale_password';
GRANT ALL PRIVILEGES ON flash_sale.* TO 'flash_sale_user'@'%';
FLUSH PRIVILEGES;
```

### Run the Service

```bash
cd apps/flash-sale-service
./gradlew bootRun
```

The service will start on:
- HTTP: `http://localhost:8084`
- gRPC: `localhost:9094`
- Actuator: `http://localhost:9091`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_HOST` | Redis server host | `localhost` |
| `REDIS_PORT` | Redis server port | `6379` |
| `MYSQL_HOST` | MySQL server host | `localhost` |
| `MYSQL_PORT` | MySQL server port | `3306` |
| `MYSQL_DATABASE` | Database name | `flash_sale` |
| `KAFKA_BOOTSTRAP_SERVERS` | Kafka brokers | `localhost:9093` |
| `ETCD_ENDPOINTS` | etcd endpoints | `http://localhost:2379` |

### Profiles

- `local`: Local development with Docker infrastructure
- `staging`: Staging environment
- `production`: Production environment with full HA
- `testing`: Unit/integration tests with Testcontainers

## API Endpoints

### Flash Sale

- `POST /api/seckill/{skuId}` - Submit flash sale request
- `GET /api/seckill/status/{orderId}` - Query order status

### Activity Management

- `POST /api/activities` - Create flash sale activity
- `GET /api/activities/{activityId}` - Get activity details
- `PUT /api/activities/{activityId}` - Update activity
- `DELETE /api/activities/{activityId}` - Delete activity

### Health & Metrics

- `GET /actuator/health` - Health check
- `GET /actuator/prometheus` - Prometheus metrics

## Redis Data Structures

```
# Inventory
stock:sku_{skuId}              -> Integer (remaining stock)
sold:sku_{skuId}               -> Integer (sold count)

# Token Bucket
token_bucket:{skuId}           -> Integer (available tokens)
token_bucket_last:{skuId}      -> Long (last refill timestamp)

# User Purchase Limit
user_purchase:{skuId}:{userId} -> Integer (purchased count)

# Order Status Cache
order_status:{orderId}         -> String (PENDING/PAID/CANCELLED)

# Deduplication
dedup:order:{orderId}          -> "1" (TTL: 7 days)
```

## Kafka Topics

| Topic | Partitions | Retention | Purpose |
|-------|------------|-----------|---------|
| `seckill-orders` | 100 | 7 days | Order messages |
| `seckill-dlq` | 10 | 30 days | Dead letter queue |

## Testing

```bash
# Run all tests
./gradlew test

# Run with coverage report
./gradlew test jacocoTestReport

# Run integration tests (requires Docker)
./gradlew integrationTest
```

## Performance Targets

| Metric | Target |
|--------|--------|
| Redis stock deduction | ≥50K QPS/instance |
| Gateway concurrent connections | ≥100K |
| Kafka throughput | ≥1M msg/s |
| Database writes | ≥2K TPS (batch) |
| P99 response time | <200ms |

## License

Internal use only.
