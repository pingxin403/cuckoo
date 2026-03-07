# Multi-Region Database Initialization Guide

## Overview

This guide explains how to initialize the MySQL database with multi-region support for the IM system. The initialization script applies schema changes from task 18.1 and validates the setup.

## Task 18.2: 创建数据库初始化脚本

### Purpose

The `init-multi-region-db.sh` script automates the following tasks:

1. **Apply Schema Changes**: Executes the migration file to add multi-region fields
2. **Verify Table Structure**: Confirms all required fields are present
3. **Verify Indexes**: Ensures multi-region indexes are created
4. **Insert Test Data**: Adds sample data to validate multi-region fields
5. **Test Queries**: Validates query performance with multi-region patterns

## Prerequisites

### 1. MySQL Running

Ensure MySQL is running via Docker Compose:

```bash
# Start MySQL infrastructure
docker compose -f deploy/docker/docker-compose.infra.yml up -d mysql

# Wait for MySQL to be healthy
docker compose -f deploy/docker/docker-compose.infra.yml ps mysql
```

### 2. Database Credentials

Default credentials (can be overridden with environment variables):

- **Host**: `localhost`
- **Port**: `3306`
- **User**: `im_service`
- **Password**: `im_service_password`
- **Database**: `im_chat`

## Usage

### Basic Usage

Run the script from the `deploy/docker` directory:

```bash
cd deploy/docker
./init-multi-region-db.sh
```

### Custom Configuration

Override default settings using environment variables:

```bash
# Custom MySQL connection
MYSQL_HOST=192.168.1.100 \
MYSQL_PORT=3307 \
MYSQL_USER=custom_user \
MYSQL_PASSWORD=custom_pass \
MYSQL_DATABASE=custom_db \
./init-multi-region-db.sh
```

### Using Docker Exec

If running inside a Docker container:

```bash
docker compose -f deploy/docker/docker-compose.infra.yml exec mysql bash
cd /docker-entrypoint-initdb.d
./init-multi-region-db.sh
```

## What the Script Does

### Step 1: Connection Check

Verifies MySQL is accessible with the provided credentials:

```
[INFO] Checking MySQL connection...
[SUCCESS] MySQL connection successful
```

### Step 2: Database Verification

Confirms the `im_chat` database exists:

```
[INFO] Checking if database 'im_chat' exists...
[SUCCESS] Database 'im_chat' exists
```

### Step 3: Schema Migration

Applies the migration file `004_offline_messages_partitioning.sql`:

```
[INFO] Applying schema migration from ../../apps/im-service/migrations/004_offline_messages_partitioning.sql...
[SUCCESS] Schema migration applied successfully
```

This migration:
- Drops and recreates the `offline_messages` table
- Adds multi-region fields: `region_id`, `global_id`, `sync_status`, `synced_at`
- Creates indexes: `idx_region_sync_status`, `idx_global_id`
- Sets up 16 hash partitions by `user_id`

### Step 4: Field Verification

Checks that all multi-region fields are present:

```
[INFO] Checking multi-region fields...
[SUCCESS]   ✓ Field 'region_id' exists
[SUCCESS]   ✓ Field 'global_id' exists
[SUCCESS]   ✓ Field 'sync_status' exists
[SUCCESS]   ✓ Field 'synced_at' exists
```

### Step 5: Index Verification

Confirms multi-region indexes are created:

```
[INFO] Verifying multi-region indexes...
[SUCCESS]   ✓ Index 'idx_region_sync_status' exists
[SUCCESS]   ✓ Index 'idx_global_id' exists
```

### Step 6: Partitioning Verification

Validates the table has 16 partitions:

```
[INFO] Verifying table partitioning...
[SUCCESS] Table has 16 partitions (as expected)
```

### Step 7: Table Structure Display

Shows the complete table structure:

```
[INFO] Table structure:

  Field              Type                                    Null    Key     Default
  id                 bigint                                  NO      PRI     NULL
  msg_id             varchar(36)                             NO      UNI     NULL
  user_id            varchar(64)                             NO      PRI     NULL
  region_id          varchar(50)                             NO      MUL     default
  global_id          varchar(255)                            YES     MUL     NULL
  sync_status        enum('pending','synced','conflict')     YES             pending
  synced_at          timestamp                               YES             NULL
  ...
```

### Step 8: Index Display

Shows all indexes on the table:

```
[INFO] Table indexes:

  Table              Non_unique  Key_name                    Seq_in_index  Column_name
  offline_messages   0           PRIMARY                     1             id
  offline_messages   0           PRIMARY                     2             user_id
  offline_messages   0           idx_msg_id                  1             msg_id
  offline_messages   1           idx_region_sync_status      1             region_id
  offline_messages   1           idx_region_sync_status      2             sync_status
  offline_messages   1           idx_region_sync_status      3             created_at
  offline_messages   1           idx_global_id               1             global_id
  ...
```

### Step 9: Test Data Verification

Displays sample multi-region data:

```
[INFO] Verifying test data...
[SUCCESS] Test data inserted: 3 rows

[INFO] Sample multi-region data:

  msg_id                                user_id   region_id   global_id                   sync_status
  a1b2c3d4-...                          user001   region-a    region-a-1234567890-0-1     synced
  e5f6g7h8-...                          user001   region-a    region-a-1234567891-0-2     pending
  i9j0k1l2-...                          user002   region-b    region-b-1234567892-0-3     synced
```

### Step 10: Additional Test Data

Inserts more test data for validation:

```
[INFO] Inserting additional test data for multi-region validation...
[SUCCESS] Additional test data inserted successfully
```

### Step 11: Query Testing

Tests multi-region query patterns:

```
[INFO] Testing multi-region query patterns...

[INFO] Test 1: Query pending messages for region-a
[SUCCESS]   Found 2 pending messages in region-a

[INFO] Test 2: Query by global_id
[SUCCESS]   Found 4 messages with region-a global_id

[INFO] Test 3: Query conflict messages
[SUCCESS]   Found 0 conflict messages

[INFO] Test 4: Query performance analysis
  id  select_type  table              type   possible_keys              key                      rows
  1   SIMPLE       offline_messages   ref    idx_region_sync_status     idx_region_sync_status   2
```

## Schema Changes (Task 18.1)

### New Fields

| Field Name | Type | Default | Description |
|------------|------|---------|-------------|
| `region_id` | VARCHAR(50) NOT NULL | 'default' | Source region identifier |
| `global_id` | VARCHAR(255) | NULL | HLC-based global ID for cross-region ordering |
| `sync_status` | ENUM('pending', 'synced', 'conflict') | 'pending' | Synchronization status |
| `synced_at` | TIMESTAMP NULL | NULL | Sync completion timestamp |

### New Indexes

1. **idx_region_sync_status (region_id, sync_status, created_at)**
   - Purpose: Fast lookup of pending sync messages by region
   - Query pattern: `WHERE region_id = ? AND sync_status = ? ORDER BY created_at`

2. **idx_global_id (global_id)**
   - Purpose: Fast lookup by HLC-based global ID
   - Query pattern: `WHERE global_id = ?`

### Partitioning

- **Strategy**: HASH partitioning by `CRC32(user_id)`
- **Partitions**: 16 partitions for even distribution
- **Benefit**: Improved query performance for user-specific queries

## Verification Steps

### Manual Verification

After running the script, verify the changes manually:

```bash
# Connect to MySQL
mysql -h localhost -u im_service -pim_service_password im_chat

# Check table structure
DESCRIBE offline_messages;

# Check indexes
SHOW INDEX FROM offline_messages;

# Check partitions
SELECT TABLE_NAME, PARTITION_NAME, TABLE_ROWS 
FROM INFORMATION_SCHEMA.PARTITIONS 
WHERE TABLE_NAME = 'offline_messages';

# Query test data
SELECT msg_id, user_id, region_id, global_id, sync_status 
FROM offline_messages;
```

### Query Performance Testing

Test multi-region query patterns:

```sql
-- Test 1: Find pending sync messages for region-a
EXPLAIN SELECT * FROM offline_messages 
WHERE region_id = 'region-a' AND sync_status = 'pending' 
ORDER BY created_at LIMIT 1000;

-- Test 2: Query by global_id
EXPLAIN SELECT * FROM offline_messages 
WHERE global_id = 'region-a-1234567890-0-1';

-- Test 3: Cross-region message ordering
SELECT * FROM offline_messages 
WHERE conversation_id = 'private:user001_user002' 
ORDER BY global_id;

-- Test 4: Update sync status
UPDATE offline_messages 
SET sync_status = 'synced', synced_at = NOW() 
WHERE region_id = 'region-a' AND sync_status = 'pending' 
LIMIT 1000;
```

## Troubleshooting

### Connection Failed

**Error**: `Cannot connect to MySQL at localhost:3306`

**Solution**:
```bash
# Check if MySQL is running
docker compose -f deploy/docker/docker-compose.infra.yml ps mysql

# Check MySQL logs
docker compose -f deploy/docker/docker-compose.infra.yml logs mysql

# Restart MySQL
docker compose -f deploy/docker/docker-compose.infra.yml restart mysql
```

### Database Not Found

**Error**: `Database 'im_chat' does not exist`

**Solution**:
```bash
# Create database manually
mysql -h localhost -u root -proot_password -e "CREATE DATABASE IF NOT EXISTS im_chat;"

# Grant permissions
mysql -h localhost -u root -proot_password -e "GRANT ALL PRIVILEGES ON im_chat.* TO 'im_service'@'%';"
```

### Migration File Not Found

**Error**: `Migration file not found: ../../apps/im-service/migrations/004_offline_messages_partitioning.sql`

**Solution**:
```bash
# Check current directory
pwd

# Should be in deploy/docker
cd deploy/docker

# Verify migration file exists
ls -la ../../apps/im-service/migrations/004_offline_messages_partitioning.sql
```

### Permission Denied

**Error**: `Permission denied: ./init-multi-region-db.sh`

**Solution**:
```bash
# Make script executable
chmod +x deploy/docker/init-multi-region-db.sh

# Run again
./init-multi-region-db.sh
```

### Fields Already Exist

**Warning**: If running the script multiple times, the migration will drop and recreate the table.

**Note**: This is expected behavior for local development. In production, use incremental migrations.

## Integration with Docker Compose

### Option 1: Run During Container Startup

Add to `docker-compose.infra.yml`:

```yaml
mysql:
  image: mysql:8.0
  volumes:
    - ./init-multi-region-db.sh:/docker-entrypoint-initdb.d/02-init-multi-region.sh:ro
    - ../../apps/im-service/migrations:/docker-entrypoint-initdb.d/migrations:ro
```

### Option 2: Run After Container Startup

```bash
# Start MySQL
docker compose -f deploy/docker/docker-compose.infra.yml up -d mysql

# Wait for MySQL to be ready
sleep 10

# Run initialization script
cd deploy/docker
./init-multi-region-db.sh
```

### Option 3: Run Inside Container

```bash
# Copy script to container
docker cp deploy/docker/init-multi-region-db.sh mysql:/tmp/

# Execute inside container
docker exec -it mysql bash -c "cd /tmp && ./init-multi-region-db.sh"
```

## Next Steps

After successful initialization:

### 1. Verify Integration Tests

```bash
cd apps/im-service
go test ./integration_test/hlc_integration_test.go -v
go test ./integration_test/conflict_resolution_integration_test.go -v
```

### 2. Start Multi-Region Services

```bash
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml \
               up -d im-service-region-a im-service-region-b
```

### 3. Test Cross-Region Message Flow

```bash
# Send message via region-a
curl -X POST http://localhost:8184/api/messages \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user001",
    "content": "Test message from region-a",
    "conversation_id": "private:user001_user002"
  }'

# Query messages from region-b
curl http://localhost:8284/api/messages?user_id=user001
```

### 4. Monitor Sync Status

```bash
# Check pending sync messages
mysql -h localhost -u im_service -pim_service_password im_chat \
  -e "SELECT COUNT(*) as pending_count FROM offline_messages WHERE sync_status='pending';"

# Check sync latency
mysql -h localhost -u im_service -pim_service_password im_chat \
  -e "SELECT region_id, AVG(TIMESTAMPDIFF(SECOND, created_at, synced_at)) as avg_sync_latency_sec 
      FROM offline_messages 
      WHERE sync_status='synced' AND synced_at IS NOT NULL 
      GROUP BY region_id;"
```

## Requirements Satisfied

✅ **Task 18.2 Requirements**:
- ✅ Create `deploy/docker/init-multi-region-db.sh`
- ✅ Automatically apply schema changes to local MySQL
- ✅ Verify table structure and index creation success
- ✅ Insert test data to validate multi-region fields

✅ **Related Requirements**:
- ✅ Requirement 2.1: HLC Global ID support (global_id field)
- ✅ Requirement 2.2: LWW Conflict Resolution (sync_status field)
- ✅ Requirement 1.1: Cross-region message replication (region_id field)

## Related Files

- **Script**: `deploy/docker/init-multi-region-db.sh`
- **Migration**: `apps/im-service/migrations/004_offline_messages_partitioning.sql`
- **Task 18.1 Summary**: `apps/im-service/migrations/TASK_18.1_COMPLETION_SUMMARY.md`
- **Design Doc**: `.kiro/specs/multi-region-active-active/design.md`
- **Requirements**: `.kiro/specs/multi-region-active-active/requirements.md`

## Summary

The `init-multi-region-db.sh` script provides a comprehensive, automated solution for initializing the MySQL database with multi-region support. It applies schema changes, verifies the setup, inserts test data, and validates query performance—all essential steps for local multi-region testing.

**Status**: ✅ Task 18.2 Complete
**Next Task**: 18.3 验证数据库 schema 变更
