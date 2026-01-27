# Deduplication Service

The deduplication service provides Redis-based message deduplication for the IM chat system. It ensures exactly-once message display by tracking processed message IDs.

## Features

- **O(1) Duplicate Detection**: Fast lookup using Redis SET operations
- **Automatic TTL**: Deduplication records expire after 7 days (configurable)
- **Atomic Operations**: Thread-safe check-and-mark operations using Redis SETNX
- **Connection Pooling**: Efficient Redis connection management via go-redis client
- **Error Handling**: Graceful handling of Redis connection failures

## Requirements

Validates the following requirements from the IM Chat System specification:

- **Requirement 8.1**: Deduplication using Redis SET with O(1) lookup
- **Requirement 8.2**: 7-day TTL on deduplication records
- **Requirement 8.3**: Atomic check-and-mark operations

## Usage

### Basic Setup

```go
import "github.com/pingxin403/cuckoo/apps/im-service/dedup"

// Create configuration
cfg := dedup.Config{
    RedisAddr:     "localhost:6379",
    RedisPassword: "",
    RedisDB:       0,
    TTL:           7 * 24 * time.Hour, // Optional, defaults to 7 days
}

// Create service
service := dedup.NewDedupService(cfg)
defer service.Close()
```

### Check for Duplicates

```go
ctx := context.Background()
msgID := "msg-12345"

// Check if message has been processed before
isDuplicate, err := service.CheckDuplicate(ctx, msgID)
if err != nil {
    log.Printf("Failed to check duplicate: %v", err)
    return
}

if isDuplicate {
    log.Printf("Message %s is a duplicate, skipping", msgID)
    return
}
```

### Mark Message as Processed

```go
// Mark message as processed
err := service.MarkProcessed(ctx, msgID)
if err != nil {
    log.Printf("Failed to mark message as processed: %v", err)
    return
}
```

### Atomic Check and Mark

For best performance and consistency, use the atomic `CheckAndMark` operation:

```go
// Atomically check for duplicate and mark if not duplicate
isDuplicate, err := service.CheckAndMark(ctx, msgID)
if err != nil {
    log.Printf("Failed to check and mark: %v", err)
    return
}

if isDuplicate {
    log.Printf("Message %s is a duplicate, skipping", msgID)
    return
}

// Process the message (it's not a duplicate)
processMessage(msgID)
```

### Health Check

```go
// Check if Redis connection is alive
err := service.Ping(ctx)
if err != nil {
    log.Printf("Redis connection failed: %v", err)
}
```

## Configuration

### Config Fields

- `RedisAddr` (string): Redis server address (e.g., "localhost:6379")
- `RedisPassword` (string): Redis password (optional)
- `RedisDB` (int): Redis database number (default: 0)
- `TTL` (time.Duration): Time-to-live for deduplication records (default: 7 days)

### Custom TTL

```go
cfg := dedup.Config{
    RedisAddr: "localhost:6379",
    TTL:       24 * time.Hour, // 1 day instead of default 7 days
}
```

## Redis Key Format

Deduplication records are stored in Redis with the following key format:

```
dedup:msg:{message_id}
```

Example: `dedup:msg:msg-12345`

## Performance

- **Lookup Time**: O(1) using Redis EXISTS command
- **Mark Time**: O(1) using Redis SET command
- **Atomic Check-and-Mark**: O(1) using Redis SETNX command
- **Memory Usage**: ~100 bytes per message ID (including Redis overhead)
- **TTL Cleanup**: Automatic by Redis, no manual cleanup required

## Error Handling

The service handles the following error scenarios:

1. **Redis Connection Failure**: Returns error, caller should retry or route to offline channel
2. **Network Timeout**: Returns error after Redis client timeout
3. **Redis Server Error**: Returns error with details from Redis

## Testing

### Unit Tests

Run unit tests with mock Redis (miniredis):

```bash
go test ./dedup/... -v
```

### Property-Based Tests

Run property-based tests (100 iterations each):

```bash
go test ./dedup/... -tags=property -v
```

### Coverage

Check test coverage:

```bash
go test ./dedup/... -cover
```

## Properties Validated

The service validates the following correctness properties:

1. **Exactly-Once Display**: First check returns false (not duplicate), subsequent checks return true
2. **TTL Expiration**: Records expire after configured TTL
3. **Concurrent Consistency**: Exactly one concurrent operation succeeds in marking as new
4. **Independent Tracking**: Different message IDs are tracked independently
5. **Check-Mark Consistency**: CheckDuplicate and MarkProcessed are consistent
6. **Key Format Consistency**: dedupKey produces consistent keys for same message ID

## Integration with IM Service

The deduplication service is used in the following scenarios:

1. **Offline Worker**: Check for duplicates before writing to offline message storage
2. **Message Router**: Prevent duplicate message delivery
3. **ACK-Offline Race**: Handle race condition between ACK and offline write

Example integration:

```go
// In offline worker
isDup, err := dedupService.CheckAndMark(ctx, msg.MsgID)
if err != nil {
    log.Printf("Dedup check failed: %v", err)
    // Continue with write (fail-open for availability)
}

if isDup {
    log.Printf("Skipping duplicate message: %s", msg.MsgID)
    return nil
}

// Write to offline storage
err = offlineStore.Write(ctx, msg)
```

## Monitoring

Key metrics to monitor:

- **Duplicate Rate**: Percentage of messages detected as duplicates
- **Redis Latency**: P50, P95, P99 latency for Redis operations
- **Error Rate**: Percentage of operations that fail
- **Memory Usage**: Redis memory usage for dedup keys

## Troubleshooting

### High Duplicate Rate

If duplicate rate is unexpectedly high:

1. Check if clients are retrying messages too aggressively
2. Verify message ID generation is unique
3. Check for clock skew issues

### Redis Connection Errors

If seeing frequent Redis connection errors:

1. Check Redis server health
2. Verify network connectivity
3. Check Redis connection pool settings
4. Consider increasing timeout values

### Memory Issues

If Redis memory usage is too high:

1. Verify TTL is set correctly (default 7 days)
2. Check if message volume is higher than expected
3. Consider reducing TTL if acceptable
4. Monitor Redis eviction policy

## References

- [IM Chat System Requirements](../.kiro/specs/im-chat-system/requirements.md)
- [IM Chat System Design](../.kiro/specs/im-chat-system/design.md)
- [Redis SET Commands](https://redis.io/commands/set/)
- [Redis SETNX Command](https://redis.io/commands/setnx/)
