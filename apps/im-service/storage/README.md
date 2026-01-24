# Offline Message Storage

This package implements offline message storage for the IM Chat System using MySQL with hash partitioning for scalability.

## Database Schema

### offline_messages Table

Stores messages for offline users with the following features:

- **Partitioning**: 16 partitions using HASH(CRC32(user_id)) for even distribution
- **TTL**: Messages expire after 7 days (expires_at column)
- **Indexes**: Optimized for common query patterns

#### Schema

```sql
CREATE TABLE offline_messages (
    id BIGINT AUTO_INCREMENT,
    msg_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conversation_type ENUM('private', 'group') NOT NULL,
    content TEXT NOT NULL,
    sequence_number BIGINT NOT NULL,
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    metadata JSON,
    
    PRIMARY KEY (id, user_id),
    UNIQUE KEY idx_msg_id (msg_id),
    KEY idx_user_timestamp (user_id, timestamp),
    KEY idx_conversation_seq (conversation_id, sequence_number),
    KEY idx_expires_at (expires_at),
    KEY idx_user_conversation (user_id, conversation_id, sequence_number)
) ENGINE=InnoDB
PARTITION BY HASH(CRC32(user_id))
PARTITIONS 16;
```

#### Indexes

1. **PRIMARY KEY (id, user_id)**: Composite key required for partitioning
2. **UNIQUE idx_msg_id**: Ensures message deduplication across all partitions
3. **idx_user_timestamp**: Optimizes user message retrieval sorted by time
4. **idx_conversation_seq**: Optimizes conversation message retrieval sorted by sequence
5. **idx_expires_at**: Optimizes TTL cleanup queries
6. **idx_user_conversation**: Composite index for user + conversation queries

## Partitioning Strategy

### Why Hash Partitioning?

- **Even Distribution**: CRC32 hash ensures uniform distribution across partitions
- **Scalability**: Each partition handles ~625K users (10M / 16)
- **Query Performance**: Partition pruning for user_id queries
- **Future Growth**: Can scale to 64 partitions if needed

### Partition Pruning

Queries with `WHERE user_id = ?` will only scan the relevant partition:

```sql
-- Only scans 1 partition (partition pruning)
SELECT * FROM offline_messages 
WHERE user_id = 'user001' 
ORDER BY timestamp DESC 
LIMIT 100;
```

### Multi-Partition Queries

Some queries require scanning multiple partitions:

```sql
-- Scans all partitions (no partition key in WHERE)
SELECT * FROM offline_messages 
WHERE conversation_id = 'group:group001' 
ORDER BY sequence_number;

-- Scans all partitions (TTL cleanup)
DELETE FROM offline_messages 
WHERE expires_at < NOW() 
LIMIT 10000;
```

## TTL Cleanup

### Cleanup Job

Run hourly via cron job:

```bash
# Cron entry (every hour)
0 * * * * mysql -u im_service -p im_chat < /path/to/ttl_cleanup.sql
```

### Cleanup Strategy

- **Batch Size**: 10,000 messages per run
- **Frequency**: Every hour
- **Retention**: 7 days
- **Logging**: Capture statistics for monitoring

### Monitoring

Track cleanup metrics:

- Messages deleted per run
- Remaining expired messages
- Oldest message timestamp
- Cleanup duration

## Query Performance

### Expected Performance

- **User message retrieval**: < 50ms (with partition pruning)
- **Conversation retrieval**: < 100ms (may scan multiple partitions)
- **Duplicate check**: < 10ms (unique index lookup)
- **TTL cleanup**: < 5s (batch delete 10,000 messages)

### Performance Testing

Run EXPLAIN on common queries:

```sql
-- Test partition pruning
EXPLAIN SELECT * FROM offline_messages 
WHERE user_id = 'user001' 
ORDER BY timestamp DESC 
LIMIT 100;

-- Test conversation query
EXPLAIN SELECT * FROM offline_messages 
WHERE conversation_id = 'private:user001_user002' 
ORDER BY sequence_number;

-- Test duplicate check
EXPLAIN SELECT * FROM offline_messages 
WHERE msg_id = 'some-uuid';
```

### Partition Statistics

Check partition distribution:

```sql
SELECT 
    PARTITION_NAME, 
    TABLE_ROWS,
    AVG_ROW_LENGTH,
    DATA_LENGTH / 1024 / 1024 as SIZE_MB
FROM INFORMATION_SCHEMA.PARTITIONS 
WHERE TABLE_NAME = 'offline_messages'
ORDER BY PARTITION_NAME;
```

## Storage Service

The `offline_store.go` file implements:

- **Batch Insert**: Insert up to 100 messages per transaction
- **Paginated Retrieval**: Cursor-based pagination for efficient fetching
- **Connection Pooling**: Reuse database connections
- **Error Handling**: Graceful handling of database failures

## API Endpoints

### Retrieve Offline Messages

```
GET /api/v1/offline?cursor={last_id}&limit={page_size}
```

Returns messages sorted by sequence_number with cursor-based pagination.

### Get Message Count

```
GET /api/v1/offline/count
```

Returns the count of offline messages for the authenticated user.

## Testing

### Unit Tests

- Test batch insert with various message counts
- Test paginated retrieval with different page sizes
- Test TTL cleanup logic
- Test database error handling
- Target: 90% coverage for storage package

### Property-Based Tests

- **Property 7: Offline Message Ordering Preservation**
  - Messages retrieved in correct sequence_number order
  - Pagination returns all messages without duplicates
  - No messages lost during pagination

## Migration

To apply the partitioning migration:

```bash
# Run migration
mysql -u im_service -p im_chat < migrations/004_offline_messages_partitioning.sql
```

**Note**: In production, migrate existing data before dropping the table.

## Monitoring

Key metrics to monitor:

- **Storage Size**: Total size of offline_messages table
- **Partition Distribution**: Ensure even distribution across partitions
- **Query Latency**: P50, P95, P99 for retrieval queries
- **Cleanup Rate**: Messages deleted per hour
- **Error Rate**: Database connection failures, query errors

## Future Enhancements

- **Archive Old Messages**: Move messages older than 30 days to cold storage
- **Compression**: Compress message content for storage efficiency
- **Read Replicas**: Use read replicas for retrieval queries
- **Sharding**: Shard across multiple databases for > 100M users
