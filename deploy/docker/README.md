# Docker Compose Deployment

Local development deployment using Docker Compose.

## Structure

```
deploy/docker/
├── docker-compose.infra.yml      # Infrastructure only (databases, caches, message queues)
├── docker-compose.services.yml   # Application services only
└── README.md                      # This file
```

## Quick Start

### Start Everything

```bash
# From project root - use both compose files
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml up -d

# Or use Makefile (recommended)
make dev-up
```

### Start Infrastructure Only

```bash
docker compose -f deploy/docker/docker-compose.infra.yml up -d
```

### Start Services Only

```bash
# Requires infrastructure to be running
docker compose -f deploy/docker/docker-compose.services.yml up -d
```

## Infrastructure Components

| Component | Container | Ports | Purpose |
|-----------|-----------|-------|---------|
| MySQL (Shortener) | shortener-mysql | 3306 | Shortener service database |
| Redis (Shortener) | shortener-redis | 6379 | Shortener service cache |
| MySQL (IM) | im-mysql | 3307 | IM chat database |
| Redis (IM) | im-redis | 6380 | IM deduplication & caching |
| etcd-1 | im-etcd-1 | 2379, 2380 | IM registry node 1 |
| etcd-2 | im-etcd-2 | 2381, 2382 | IM registry node 2 |
| etcd-3 | im-etcd-3 | 2383, 2384 | IM registry node 3 |
| Kafka-1 | im-kafka-1 | 9093, 19093 | IM message bus node 1 |
| Kafka-2 | im-kafka-2 | 9094, 19094 | IM message bus node 2 |
| Kafka-3 | im-kafka-3 | 9095, 19095 | IM message bus node 3 |

## Application Services

| Service | Container | Ports | Dependencies |
|---------|-----------|-------|--------------|
| Hello Service | hello-service | 9090 | None |
| TODO Service | todo-service | 9091 | hello-service |
| Shortener Service | shortener-service | 9092 (gRPC), 8081 (HTTP) | mysql, redis |
| Envoy Gateway | envoy-gateway | 8080, 9901 | shortener-service |

## Common Commands

```bash
# View logs
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml logs -f [service-name]

# Stop all services
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml down

# Stop and remove volumes (WARNING: deletes all data!)
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml down -v

# Rebuild services
docker compose -f deploy/docker/docker-compose.services.yml build [service-name]

# Restart a service
docker compose -f deploy/docker/docker-compose.services.yml restart [service-name]

# Check service health
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml ps
```

**Tip**: Use Makefile commands for easier access:
```bash
make dev-up          # Start everything
make dev-down        # Stop everything
make dev-restart     # Restart services
make infra-up        # Start infrastructure only
make services-up     # Start services only
```

## Development Workflow

### 1. Start Infrastructure

```bash
docker compose -f deploy/docker/docker-compose.infra.yml up -d
```

### 2. Verify Infrastructure

```bash
# Check all containers are healthy
docker compose -f deploy/docker/docker-compose.infra.yml ps

# Test MySQL
docker exec shortener-mysql mysql -u shortener_user -pshortener_password -e "SELECT 1"

# Test Redis
docker exec shortener-redis redis-cli PING

# Test etcd
docker exec im-etcd-1 etcdctl endpoint health

# Test Kafka
docker exec im-kafka-1 kafka-topics --list --bootstrap-server localhost:9092
```

### 3. Start Services

```bash
docker compose -f deploy/docker/docker-compose.services.yml up -d
```

### 4. Test Services

```bash
# Test hello-service
grpcurl -plaintext localhost:9090 list

# Test todo-service
grpcurl -plaintext localhost:9091 list

# Test shortener-service
curl http://localhost:8081/health
```

## Troubleshooting

### Services won't start

```bash
# Check logs
docker compose logs [service-name]

# Check if ports are already in use
lsof -i :9090  # or any other port
```

### Database connection issues

```bash
# Verify MySQL is running
docker exec shortener-mysql mysqladmin ping -h localhost -u root -proot_password

# Check network
docker network inspect monorepo-network
```

### Kafka issues

```bash
# Check Kafka cluster status
docker exec im-kafka-1 kafka-metadata --bootstrap-server localhost:9092 --describe --replication

# Recreate topics
docker compose -f deploy/docker/docker-compose.infra.yml restart kafka-init
```

## Clean Up

```bash
# Stop all services (using Makefile - recommended)
make dev-down

# Or use docker compose directly
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml down

# Remove volumes (WARNING: deletes all data!)
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml down -v

# Remove images
docker rmi hello-service:latest todo-service:latest shortener-service:latest
```

## Environment Variables

Override default values by creating a `.env` file in the project root:

```env
# MySQL
MYSQL_ROOT_PASSWORD=your_password
MYSQL_DATABASE=shortener
MYSQL_USER=shortener_user
MYSQL_PASSWORD=your_password

# IM MySQL
IM_MYSQL_ROOT_PASSWORD=your_password
IM_MYSQL_DATABASE=im_chat
IM_MYSQL_USER=im_service
IM_MYSQL_PASSWORD=your_password

# Service Ports
HELLO_SERVICE_PORT=9090
TODO_SERVICE_PORT=9091
SHORTENER_SERVICE_PORT=9092
```

## Production Considerations

This Docker Compose setup is for **local development only**. For production:

1. Use Kubernetes deployment (see `deploy/k8s/`)
2. Use managed services for databases (RDS, Cloud SQL)
3. Use managed Redis (ElastiCache, Cloud Memorystore)
4. Use managed Kafka (MSK, Confluent Cloud)
5. Enable TLS/SSL for all connections
6. Use secrets management (Vault, AWS Secrets Manager)
7. Enable monitoring and logging
8. Configure proper resource limits
9. Set up backups and disaster recovery

## Related Documentation

- [Kubernetes Deployment](../k8s/README.md)
- [IM Chat System Infrastructure](../../apps/im-chat-system/README.md)
- [Shortener Service](../../apps/shortener-service/README.md)
