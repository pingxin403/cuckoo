# IM Chat System Infrastructure

This directory contains the infrastructure setup for the IM Chat System, including etcd, MySQL, Redis, and Kafka clusters.

## Architecture Overview

The IM Chat System uses the following infrastructure components:

- **etcd Cluster (3 nodes)**: Distributed registry for user-to-gateway mappings
- **MySQL**: Persistent storage for offline messages, users, and groups
- **Redis**: Deduplication, caching, and sequence number generation
- **Kafka Cluster (3 brokers, KRaft mode)**: Message bus for group messages and offline message queue (no Zookeeper required)

## Quick Start

### 1. Start Infrastructure

Start all infrastructure services using Docker Compose:

```bash
# Start infrastructure services (MySQL, Redis, etcd, Kafka)
make infra-up

# Run database migrations for IM Chat System
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase

# Or manually start services
docker compose -f deploy/docker/docker-compose.infra.yml up -d
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase  # Run database migrations
```

### 2. Verify Infrastructure

Run the infrastructure test script:

```bash
./scripts/test-im-infrastructure.sh
```

This will verify:
- ✓ etcd cluster health (3 nodes)
- ✓ MySQL database and tables
- ✓ Redis connectivity and persistence
- ✓ Kafka cluster and topics

### 3. Access Infrastructure

**etcd Cluster:**
```bash
# Access etcd-1
docker exec -it im-etcd-1 etcdctl endpoint health

# List all members
docker exec -it im-etcd-1 etcdctl member list

# Put/Get values
docker exec -it im-etcd-1 etcdctl put /test/key "value"
docker exec -it im-etcd-1 etcdctl get /test/key
```

**MySQL:**
```bash
# Connect to MySQL
docker exec -it im-mysql mysql -u im_service -pim_service_password -D im_chat

# Show tables
docker exec -it im-mysql mysql -u im_service -pim_service_password -D im_chat -e "SHOW TABLES;"

# Query offline messages
docker exec -it im-mysql mysql -u im_service -pim_service_password -D im_chat -e "SELECT * FROM offline_messages LIMIT 10;"
```

**Redis:**
```bash
# Connect to Redis CLI
docker exec -it im-redis redis-cli

# Test commands
docker exec -it im-redis redis-cli PING
docker exec -it im-redis redis-cli SET test:key "value"
docker exec -it im-redis redis-cli GET test:key
```

**Kafka:**
```bash
# List topics
docker exec -it im-kafka-1 kafka-topics --list --bootstrap-server localhost:9092

# Describe topic
docker exec -it im-kafka-1 kafka-topics --describe --topic group_msg --bootstrap-server localhost:9092

# Produce message
echo "test message" | docker exec -i im-kafka-1 kafka-console-producer --bootstrap-server localhost:9092 --topic group_msg

# Consume messages
docker exec -it im-kafka-1 kafka-console-consumer --bootstrap-server localhost:9092 --topic group_msg --from-beginning
```

## Infrastructure Endpoints

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

## Database Migration Management (Liquibase)

### Overview

The IM Chat System uses Liquibase for database schema version control and migration management. This provides:

- **Version Control**: Track all database changes in YAML files
- **Rollback Support**: Easily rollback changes if needed
- **Environment Consistency**: Ensure dev, test, and prod databases are in sync
- **Change Tracking**: Liquibase tracks which changes have been applied

### Migration Files Structure

```
apps/im-chat-system/migrations/
├── liquibase.properties          # Liquibase configuration
├── changelog/
│   ├── db.changelog-master.yaml  # Master changelog (includes all versions)
│   └── v1.0/
│       ├── 001-initial-schema.yaml  # Schema DDL
│       └── 002-sample-data.yaml     # Sample data DML
```

### Common Commands

**Apply Migrations:**
```bash
# Apply all pending migrations
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase
```

**Check Migration Status:**
```bash
# See which changesets have been applied
docker compose -f deploy/docker/docker-compose.infra.yml run --rm im-liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://mysql:3306/im_chat?useSSL=false \
  --username=im_service \
  --password=im_service_password \
  status --verbose
```

**Rollback Changes:**
```bash
# Rollback the last changeset
make im-db-rollback

# Rollback to a specific tag
docker compose run --rm im-liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://im-mysql:3306/im_chat \
  --username=im_service \
  --password=im_service_password \
  rollback v1.0
```

**Validate Changelog:**
```bash
# Validate changelog syntax
make im-db-validate
```

**Generate Diff:**
```bash
# Compare database with changelog
make im-db-diff
```

### Creating New Migrations

1. **Create a new changeset file:**
```bash
# Create new version directory
mkdir -p apps/im-chat-system/migrations/changelog/v1.1

# Create changeset file
touch apps/im-chat-system/migrations/changelog/v1.1/003-add-message-reactions.yaml
```

2. **Write the changeset:**
```yaml
databaseChangeLog:
  - changeSet:
      id: 003-add-message-reactions-table
      author: your-name
      comment: Add table for message reactions (emoji)
      changes:
        - createTable:
            tableName: message_reactions
            columns:
              - column:
                  name: id
                  type: BIGINT
                  autoIncrement: true
                  constraints:
                    primaryKey: true
              - column:
                  name: msg_id
                  type: VARCHAR(36)
                  constraints:
                    nullable: false
              - column:
                  name: user_id
                  type: VARCHAR(64)
                  constraints:
                    nullable: false
              - column:
                  name: reaction
                  type: VARCHAR(50)
                  constraints:
                    nullable: false
              - column:
                  name: created_at
                  type: TIMESTAMP
                  defaultValueComputed: CURRENT_TIMESTAMP
      rollback:
        - dropTable:
            tableName: message_reactions
```

3. **Include in master changelog:**
```yaml
# apps/im-chat-system/migrations/changelog/db.changelog-master.yaml
databaseChangeLog:
  - include:
      file: changelog/v1.0/001-initial-schema.yaml
  - include:
      file: changelog/v1.0/002-sample-data.yaml
  - include:
      file: changelog/v1.1/003-add-message-reactions.yaml  # Add this line
```

4. **Apply the migration:**
```bash
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase
```

### Best Practices

1. **Never modify existing changesets** - Once applied to production, changesets are immutable
2. **Always provide rollback** - Define rollback for each changeset when possible
3. **Use contexts** - Separate test data from production schema using contexts
4. **Descriptive IDs** - Use clear, sequential IDs like `001-create-users-table`
5. **One logical change per changeset** - Keep changesets focused and atomic
6. **Test rollbacks** - Always test rollback before deploying to production

### Contexts

Liquibase contexts allow you to control which changesets run in different environments:

- `test`: Sample data for testing (included by default in docker-compose)
- `prod`: Production-only changes
- No context: Always runs

Example:
```yaml
- changeSet:
    id: 002-insert-sample-users
    context: test  # Only runs with --contexts=test
    changes:
      - insert:
          tableName: users
          ...
```

### Troubleshooting

**Migration fails with "Table already exists":**
```bash
# Clear Liquibase lock
docker compose run --rm im-liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://im-mysql:3306/im_chat \
  --username=im_service \
  --password=im_service_password \
  clear-checksums

# Or reset database (WARNING: Deletes all data!)
docker exec shared-mysql mysql -u root -proot_password -e "DROP DATABASE im_chat; CREATE DATABASE im_chat;"
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase
```

**Check Liquibase logs:**
```bash
docker logs im-liquibase
```

**Manually connect to database:**
```bash
docker exec -it im-mysql mysql -u im_service -pim_service_password -D im_chat
```

## Database Schema

### Tables

1. **offline_messages**: Stores offline messages with 7-day retention
   - Partitioned by user_id (16 partitions)
   - Indexed by user_id, timestamp, conversation_id, sequence_number

2. **groups**: Group metadata
   - Stores group name, creator, member count

3. **group_members**: Group membership
   - Many-to-many relationship between users and groups
   - Supports roles: owner, admin, member

4. **sequence_snapshots**: Backup for Redis sequence numbers
   - Periodic snapshots every 10,000 messages
   - Used for disaster recovery

5. **users**: User profiles
   - Stores username, email, display name, avatar

### Sample Data

The database is initialized with sample data:
- 3 test users (alice, bob, charlie)
- 1 test group with 3 members

## Kafka Topics

| Topic | Partitions | Replication | Retention | Purpose |
|-------|------------|-------------|-----------|---------|
| group_msg | 32 | 3 | 1 hour | Group message broadcast |
| offline_msg | 64 | 3 | 7 days | Offline message queue |
| membership_change | 16 | 3 | 1 hour | Group membership events |

## Configuration

### etcd Configuration
- Cluster size: 3 nodes
- TTL: 90 seconds (auto-cleanup)
- Lease renewal: 30 seconds

### MySQL Configuration
- Max connections: 500
- Connection pooling: 25 max open, 5 max idle
- Partitioning: 16 partitions by user_id hash

### Redis Configuration
- Persistence: AOF + RDB
- Max memory: 2GB
- Eviction policy: allkeys-lru
- TTL for deduplication: 7 days

### Kafka Configuration
- Mode: KRaft (no Zookeeper)
- Cluster size: 3 brokers (also acting as controllers)
- Replication factor: 3
- Min in-sync replicas: 2
- Compression: snappy
- Acks: all (wait for all replicas)

## Monitoring

### Health Checks

All services have health checks configured:

```bash
# Check all container health
docker ps --format "table {{.Names}}\t{{.Status}}"

# Check specific service
docker inspect --format='{{.State.Health.Status}}' im-etcd-1
```

### Logs

View logs for troubleshooting:

```bash
# View etcd logs
docker logs im-etcd-1

# View MySQL logs
docker logs im-mysql

# View Kafka logs
docker logs im-kafka-1

# Follow logs in real-time
docker logs -f im-redis
```

## Cleanup

### Stop Services

```bash
# Stop all IM infrastructure
docker compose stop etcd-1 etcd-2 etcd-3 im-mysql im-redis kafka-1 kafka-2 kafka-3

# Or stop all services
docker compose down
```

### Remove Data

```bash
# Remove all volumes (WARNING: This deletes all data!)
docker compose down -v

# Or remove specific volumes
docker volume rm im-mysql-data im-redis-data im-etcd-1-data im-etcd-2-data im-etcd-3-data im-kafka-1-data im-kafka-2-data im-kafka-3-data
```

## Troubleshooting

### etcd Cluster Issues

If etcd cluster fails to form:

```bash
# Check member list
docker exec im-etcd-1 etcdctl member list

# Check cluster health
docker exec im-etcd-1 etcdctl endpoint health --cluster

# Reset cluster (WARNING: Deletes all data)
docker compose down
docker volume rm im-etcd-1-data im-etcd-2-data im-etcd-3-data
docker compose up -d etcd-1 etcd-2 etcd-3
```

### Kafka Issues (KRaft Mode)

```bash
# Check Kafka cluster status
docker exec im-kafka-1 kafka-metadata --bootstrap-server localhost:9092 --describe --replication

# Check controller status
docker exec im-kafka-1 kafka-metadata --bootstrap-server localhost:9092 --describe --controllers

# Reset Kafka cluster (WARNING: Deletes all data)
docker compose down
docker volume rm im-kafka-1-data im-kafka-2-data im-kafka-3-data
docker compose up -d kafka-1 kafka-2 kafka-3
```

### MySQL Connection Issues

```bash
# Check MySQL is running
docker exec im-mysql mysqladmin ping -h localhost -u root -pim_root_password

# Check user permissions
docker exec im-mysql mysql -u root -pim_root_password -e "SHOW GRANTS FOR 'im_service'@'%';"

# Recreate database
docker exec im-mysql mysql -u root -pim_root_password -e "DROP DATABASE IF EXISTS im_chat; CREATE DATABASE im_chat;"
docker compose restart im-mysql
```

### Kafka Topic Issues

```bash
# Delete and recreate topics
docker exec im-kafka-1 kafka-topics --delete --topic group_msg --bootstrap-server localhost:9092
docker compose restart kafka-init

# Check consumer groups
docker exec im-kafka-1 kafka-consumer-groups --list --bootstrap-server localhost:9092
```

## Next Steps

After infrastructure is running:

1. **Implement Protobuf APIs** (Task 2.1-2.3)
2. **Implement Auth Service** (Task 3.1-3.6)
3. **Implement User Service** (Task 4.1-4.6)
4. **Implement Core IM Services** (Tasks 5-10)

See `tasks.md` for the complete implementation plan.

## References

- [etcd Documentation](https://etcd.io/docs/)
- [MySQL Documentation](https://dev.mysql.com/doc/)
- [Redis Documentation](https://redis.io/documentation)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Design Document](./design.md)
- [Requirements Document](./requirements.md)
