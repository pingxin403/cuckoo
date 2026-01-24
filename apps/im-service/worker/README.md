# Offline Worker

The offline worker consumes messages from the Kafka `offline_msg` topic and persists them to the MySQL database for offline message delivery.

## Architecture

```
Kafka (offline_msg) → Offline Worker → Redis (dedup) → MySQL (offline_messages)
                                    ↓
                              Commit Offset
```

## Features

### 1. Kafka Consumer (Task 12.1)
- **Consumer Group**: `offline-worker-group` for load balancing
- **Topic**: `offline_msg` with 64 partitions
- **Manual Offset Commit**: Only commits after successful database write
- **Rebalancing**: Handles consumer group rebalancing gracefully
- **Error Handling**: Retries on failure, Kafka redelivers on crash

### 2. Deduplication (Task 12.2)
- **Redis Check**: Checks `dedup:msg:{msg_id}` before database write
- **Skip Duplicates**: Skips messages already in Redis dedup set
- **Mark Processed**: Adds msg_id to Redis after successful write
- **Race Condition**: Handles ACK-offline race condition (Requirement 3.10)
- **TTL**: 7-day TTL on dedup records

### 3. Batch Processing (Task 12.3)
- **Batch Size**: 100 messages or 5 seconds (whichever comes first)
- **Single Transaction**: All messages in batch inserted in one transaction
- **Atomic Commit**: Kafka offset committed only after successful write
- **Rollback**: On failure, transaction rolls back and Kafka redelivers
- **Retry Logic**: Up to 5 retries with exponential backoff

## Configuration

```go
config := WorkerConfig{
    KafkaBrokers:  []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
    ConsumerGroup: "offline-worker-group",
    Topic:         "offline_msg",
    BatchSize:     100,
    BatchTimeout:  5 * time.Second,
    MaxRetries:    5,
    RetryBackoff:  []time.Duration{1*time.Second, 2*time.Second, 4*time.Second, 8*time.Second, 16*time.Second},
    MessageTTL:    7 * 24 * time.Hour, // 7 days
}
```

## Usage

```go
// Create dependencies
store, err := storage.NewOfflineStore(storageConfig)
dedupService := dedup.NewDedupService(dedupConfig)

// Create worker
worker, err := worker.NewOfflineWorker(workerConfig, store, dedupService)
if err != nil {
    log.Fatal(err)
}

// Start worker
if err := worker.Start(); err != nil {
    log.Fatal(err)
}

// Graceful shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

if err := worker.Stop(); err != nil {
    log.Fatal(err)
}
```

## Metrics

The worker tracks the following metrics:

- **MessagesProcessed**: Total messages consumed from Kafka
- **MessagesDeduplicated**: Messages skipped due to duplicate detection
- **MessagesPersisted**: Messages successfully written to database
- **BatchWrites**: Number of batch write operations
- **Errors**: Total errors encountered
- **AvgBatchSize**: Average number of messages per batch

```go
stats := worker.GetStats()
fmt.Printf("Processed: %d, Persisted: %d, Duplicates: %d, Avg Batch: %.2f\n",
    stats.MessagesProcessed,
    stats.MessagesPersisted,
    stats.MessagesDeduplicated,
    stats.AvgBatchSize)
```

## Message Flow

1. **Consume**: Worker consumes message from Kafka `offline_msg` topic
2. **Parse**: Unmarshal JSON to `OfflineMessageEvent`
3. **Accumulate**: Add to in-memory batch buffer
4. **Trigger**: Process batch when size reaches 100 or timeout reaches 5s
5. **Deduplicate**: Check each msg_id against Redis dedup set
6. **Filter**: Remove duplicates from batch
7. **Insert**: Batch insert unique messages to MySQL in single transaction
8. **Mark**: Add msg_ids to Redis dedup set (TTL=7 days)
9. **Commit**: Commit Kafka offset to mark messages as processed
10. **Retry**: On failure, rollback and retry up to 5 times

## Error Handling

### Database Errors
- **Retry**: Up to 5 retries with exponential backoff (1s, 2s, 4s, 8s, 16s)
- **Rollback**: Transaction rolls back on error
- **Redeliver**: Kafka offset not committed, messages redelivered

### Deduplication Errors
- **Best Effort**: Dedup check errors don't fail the batch
- **Continue**: Message processed even if dedup check fails
- **Log**: Errors logged for monitoring

### Consumer Errors
- **Reconnect**: Automatically reconnects to Kafka on connection loss
- **Rebalance**: Handles consumer group rebalancing
- **Graceful**: Flushes remaining batch on shutdown

## ACK-Offline Race Condition

**Scenario**: Message routed to offline channel, then delayed ACK arrives

**Solution**:
1. Offline worker checks Redis dedup set before database write
2. If msg_id exists (ACK arrived first), skip database write
3. Client performs final deduplication using local SQLite
4. Result: At-most-once database write, exactly-once display

**Example**:
```
T0: User B offline, message routed to Kafka offline_msg
T1: Offline worker consumes message
T2: Offline worker checks Redis dedup set → not found
T3: Delayed ACK arrives, adds msg_id to Redis dedup set
T4: Offline worker writes to database (race lost)
T5: Client fetches offline messages, deduplicates using local SQLite
```

## Requirements Validation

- **Requirement 4.2**: Consume messages from Kafka offline_msg topic ✓
- **Requirement 4.6**: Batch write messages to database ✓
- **Requirement 3.9**: Check Redis dedup set before database write ✓
- **Requirement 3.10**: Skip if msg_id already exists ✓
- **Requirement 3.11**: Add to dedup set after successful write ✓

## Testing

See `offline_worker_test.go` for unit tests and `offline_worker_property_test.go` for property-based tests.

Target coverage: 90%
