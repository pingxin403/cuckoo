# Batch Operation Optimization Guide

## Overview

The Batch Processor provides comprehensive batch operation optimization for cross-region multi-active architectures. It optimizes three critical operations:

1. **Message Batch Synchronization** - Efficient cross-region message sync
2. **Offline Message Batch Writes** - Optimized database batch inserts
3. **Reconciliation Batch Processing** - Efficient data consistency checks

## Features

### Core Capabilities

- **Automatic Batching**: Collects items and flushes based on size or time
- **Priority Ordering**: Processes high-priority items first
- **Retry Logic**: Automatic retry with configurable limits
- **Concurrent Processing**: Multiple batch types processed in parallel
- **Metrics Collection**: Comprehensive performance monitoring
- **Graceful Shutdown**: Flushes remaining batches on stop

### Performance Optimizations

- **Compression**: Optional message compression for bandwidth savings
- **Pipelining**: Redis pipeline support for batch operations
- **Concurrent Batches**: Process multiple batches simultaneously
- **Smart Flushing**: Size-based and time-based flush triggers

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"
    "github.com/cuckoo-org/cuckoo/libs/connpool"
)

func main() {
    // Create batch processor with default config
    config := connpool.DefaultBatchProcessorConfig()
    bp := connpool.NewBatchProcessor(config)
    
    // Start the processor
    if err := bp.Start(); err != nil {
        panic(err)
    }
    defer bp.Stop()
    
    // Add messages to batch
    msg := connpool.BatchMessage{
        ID:             "msg-001",
        RegionID:       "region-a",
        GlobalID:       "global-001",
        ConversationID: "conv-123",
        SenderID:       "user-456",
        Content:        "Hello, World!",
        Timestamp:      time.Now().UnixMilli(),
        Priority:       5,
        CreatedAt:      time.Now(),
    }
    
    if err := bp.AddMessage(msg); err != nil {
        panic(err)
    }
    
    // Messages are automatically flushed based on:
    // 1. Batch size (default: 100 messages)
    // 2. Flush interval (default: 100ms)
}
```

### Integration with Pool Manager

```go
package main

import (
    "github.com/cuckoo-org/cuckoo/libs/connpool"
)

func main() {
    // Create pool manager
    poolConfig := connpool.DefaultPoolConfig()
    pm := connpool.NewPoolManager(poolConfig)
    defer pm.Stop()
    
    // Create batch processor
    batchConfig := connpool.DefaultBatchProcessorConfig()
    bp, err := pm.GetOrCreateBatchProcessor("main", batchConfig)
    if err != nil {
        panic(err)
    }
    
    // Start pool manager (starts all processors)
    if err := pm.Start(); err != nil {
        panic(err)
    }
    
    // Use batch processor
    msg := connpool.BatchMessage{
        ID:       "msg-001",
        Priority: 5,
    }
    bp.AddMessage(msg)
    
    // Get metrics from all processors
    allMetrics := pm.GetAllBatchProcessorMetrics()
    for name, metrics := range allMetrics {
        println("Processor:", name)
        println("  Total batched:", metrics.TotalMessagesBatched)
        println("  Avg batch size:", metrics.AvgBatchSize)
    }
}
```

## Configuration

### Batch Processor Configuration

```go
type BatchProcessorConfig struct {
    // Message batch configuration
    MessageBatchSize     int           // Maximum messages per batch (default: 100)
    MessageFlushInterval time.Duration // Maximum time before flushing (default: 100ms)
    MessageMaxRetries    int           // Maximum retry attempts (default: 3)
    
    // Offline message batch configuration
    OfflineBatchSize     int           // Maximum offline messages per batch (default: 50)
    OfflineFlushInterval time.Duration // Maximum time before flushing (default: 200ms)
    OfflineMaxRetries    int           // Maximum retry attempts (default: 3)
    
    // Reconcile batch configuration
    ReconcileBatchSize     int           // Maximum reconcile items per batch (default: 100)
    ReconcileFlushInterval time.Duration // Maximum time before flushing (default: 500ms)
    ReconcileMaxRetries    int           // Maximum retry attempts (default: 3)
    
    // Performance tuning
    EnableCompression    bool // Enable message compression (default: true)
    EnablePipelining     bool // Enable Redis pipelining (default: true)
    MaxConcurrentBatches int  // Maximum concurrent batch operations (default: 10)
    
    // Monitoring
    EnableMetrics      bool          // Enable metrics collection (default: true)
    MetricsInterval    time.Duration // Metrics reporting interval (default: 30s)
    EnableHealthChecks bool          // Enable health checks (default: true)
}
```

### Custom Configuration

```go
// Create custom configuration
config := connpool.BatchProcessorConfig{
    // Optimize for high throughput
    MessageBatchSize:     200,
    MessageFlushInterval: 50 * time.Millisecond,
    MessageMaxRetries:    5,
    
    // Optimize for database writes
    OfflineBatchSize:     100,
    OfflineFlushInterval: 100 * time.Millisecond,
    OfflineMaxRetries:    3,
    
    // Optimize for reconciliation
    ReconcileBatchSize:     150,
    ReconcileFlushInterval: 250 * time.Millisecond,
    ReconcileMaxRetries:    5,
    
    // Enable all optimizations
    EnableCompression:    true,
    EnablePipelining:     true,
    MaxConcurrentBatches: 20,
    
    // Enable monitoring
    EnableMetrics:      true,
    MetricsInterval:    10 * time.Second,
    EnableHealthChecks: true,
}

bp := connpool.NewBatchProcessor(config)
```

## Usage Examples

### 1. Message Batch Synchronization

```go
// Add messages for cross-region sync
for _, msg := range messages {
    batchMsg := connpool.BatchMessage{
        ID:             msg.ID,
        RegionID:       "region-a",
        GlobalID:       msg.GlobalID,
        ConversationID: msg.ConversationID,
        SenderID:       msg.SenderID,
        Content:        msg.Content,
        Timestamp:      msg.Timestamp,
        Priority:       msg.Priority, // Higher priority = processed first
        CreatedAt:      time.Now(),
    }
    
    if err := bp.AddMessage(batchMsg); err != nil {
        log.Printf("Failed to add message: %v", err)
    }
}

// Messages are automatically batched and flushed
// based on size (100 messages) or time (100ms)
```

### 2. Offline Message Batch Writes

```go
// Add offline messages for batch database insert
for _, msg := range offlineMessages {
    batchMsg := connpool.BatchOfflineMessage{
        ID:             msg.ID,
        UserID:         msg.UserID,
        SenderID:       msg.SenderID,
        ConversationID: msg.ConversationID,
        Content:        msg.Content,
        SequenceNumber: msg.SequenceNumber,
        Timestamp:      msg.Timestamp,
        ExpiresAt:      msg.ExpiresAt,
        RegionID:       "region-a",
        GlobalID:       msg.GlobalID,
        CreatedAt:      time.Now(),
    }
    
    if err := bp.AddOfflineMessage(batchMsg); err != nil {
        log.Printf("Failed to add offline message: %v", err)
    }
}

// Offline messages are batched and written to database
// using efficient batch INSERT statements
```

### 3. Reconciliation Batch Processing

```go
// Add reconciliation items for batch processing
for _, item := range reconcileItems {
    batchItem := connpool.BatchReconcileItem{
        GlobalID:     item.GlobalID,
        Operation:    item.Operation, // "add", "update", "delete"
        SourceRegion: "region-a",
        TargetRegion: "region-b",
        MessageData:  item.Data,
        Priority:     item.Priority,
        CreatedAt:    time.Now(),
    }
    
    if err := bp.AddReconcileItem(batchItem); err != nil {
        log.Printf("Failed to add reconcile item: %v", err)
    }
}

// Reconciliation items are batched and processed
// efficiently with priority ordering
```

### 4. Manual Flush

```go
// Flush all pending batches immediately
ctx := context.Background()
if err := bp.FlushAll(ctx); err != nil {
    log.Printf("Failed to flush batches: %v", err)
}

// Useful for:
// - Graceful shutdown
// - End of processing window
// - Before critical operations
```

### 5. Metrics Monitoring

```go
// Get current metrics
metrics := bp.GetMetrics()

fmt.Printf("Batch Processor Metrics:\n")
fmt.Printf("  Messages batched: %d\n", metrics.TotalMessagesBatched)
fmt.Printf("  Offline batched: %d\n", metrics.TotalOfflineBatched)
fmt.Printf("  Reconcile batched: %d\n", metrics.TotalReconcileBatched)
fmt.Printf("  Batches flushed: %d\n", metrics.TotalBatchesFlushed)
fmt.Printf("  Batch errors: %d\n", metrics.TotalBatchErrors)
fmt.Printf("  Avg batch size: %.2f\n", metrics.AvgBatchSize)
fmt.Printf("  Avg flush duration: %v\n", metrics.AvgFlushDuration)
fmt.Printf("  Last flush: %v\n", metrics.LastFlushTime)
fmt.Printf("  Current message batch: %d\n", metrics.CurrentMessageBatch)
fmt.Printf("  Current offline batch: %d\n", metrics.CurrentOfflineBatch)
fmt.Printf("  Current reconcile batch: %d\n", metrics.CurrentReconcileBatch)
```

## Integration Examples

### With Kafka

```go
import (
    "github.com/IBM/sarama"
    "github.com/cuckoo-org/cuckoo/libs/connpool"
)

// Create Kafka producer
kafkaConfig := sarama.NewConfig()
kafkaConfig.Producer.Return.Successes = true
kafkaConfig.Producer.Compression = sarama.CompressionSnappy

producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, kafkaConfig)
if err != nil {
    panic(err)
}
defer producer.Close()

// Create batch processor with Kafka
bpConfig := connpool.DefaultBatchProcessorConfig()
bp := connpool.NewBatchProcessorWithKafka(bpConfig, producer)

if err := bp.Start(); err != nil {
    panic(err)
}
defer bp.Stop()

// Add messages - they will be batched and sent to Kafka
msg := connpool.BatchMessage{
    ID:       "msg-001",
    Content:  "Hello Kafka!",
    Priority: 5,
}
bp.AddMessage(msg)
```

### With Redis

```go
import (
    "github.com/redis/go-redis/v9"
    "github.com/cuckoo-org/cuckoo/libs/connpool"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    DB:   0,
})
defer redisClient.Close()

// Create batch processor with Redis
bpConfig := connpool.DefaultBatchProcessorConfig()
bp := connpool.NewBatchProcessorWithRedis(bpConfig, redisClient)

if err := bp.Start(); err != nil {
    panic(err)
}
defer bp.Stop()

// Add offline messages - they will be batched and written to Redis
msg := connpool.BatchOfflineMessage{
    ID:      "offline-001",
    UserID:  "user-123",
    Content: "Offline message",
}
bp.AddOfflineMessage(msg)
```

## Performance Tuning

### Batch Size Optimization

```go
// For high throughput scenarios
config := connpool.DefaultBatchProcessorConfig()
config.MessageBatchSize = 200      // Larger batches
config.MessageFlushInterval = 50 * time.Millisecond  // Faster flush

// For low latency scenarios
config.MessageBatchSize = 50       // Smaller batches
config.MessageFlushInterval = 10 * time.Millisecond  // Very fast flush

// For balanced performance
config.MessageBatchSize = 100      // Default
config.MessageFlushInterval = 100 * time.Millisecond // Default
```

### Concurrent Processing

```go
// Enable more concurrent batch operations
config := connpool.DefaultBatchProcessorConfig()
config.MaxConcurrentBatches = 20   // Process up to 20 batches concurrently

// This is useful for:
// - High throughput scenarios
// - Multiple batch types
// - Cross-region synchronization
```

### Compression and Pipelining

```go
// Enable all optimizations
config := connpool.DefaultBatchProcessorConfig()
config.EnableCompression = true    // Compress messages (saves bandwidth)
config.EnablePipelining = true     // Use Redis pipelining (faster writes)

// Compression is useful for:
// - Cross-region sync (reduces bandwidth)
// - Large message payloads
// - Network-constrained environments

// Pipelining is useful for:
// - Redis batch operations
// - High throughput writes
// - Reducing round-trip latency
```

## Monitoring and Metrics

### Available Metrics

```go
type BatchProcessorMetrics struct {
    TotalMessagesBatched  int64         // Total messages added to batch
    TotalOfflineBatched   int64         // Total offline messages added
    TotalReconcileBatched int64         // Total reconcile items added
    TotalBatchesFlushed   int64         // Total batches flushed
    TotalBatchErrors      int64         // Total batch errors
    AvgBatchSize          float64       // Average batch size
    AvgFlushDuration      time.Duration // Average flush duration
    LastFlushTime         time.Time     // Last flush timestamp
    CurrentMessageBatch   int           // Current message batch size
    CurrentOfflineBatch   int           // Current offline batch size
    CurrentReconcileBatch int           // Current reconcile batch size
}
```

### Prometheus Integration

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    batchSizeGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "batch_processor_batch_size",
            Help: "Current batch size",
        },
        []string{"type"},
    )
    
    batchFlushCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "batch_processor_flushes_total",
            Help: "Total number of batch flushes",
        },
        []string{"type"},
    )
    
    batchFlushDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "batch_processor_flush_duration_seconds",
            Help:    "Batch flush duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"type"},
    )
)

// Update metrics periodically
func updateMetrics(bp *connpool.BatchProcessor) {
    metrics := bp.GetMetrics()
    
    batchSizeGauge.WithLabelValues("message").Set(float64(metrics.CurrentMessageBatch))
    batchSizeGauge.WithLabelValues("offline").Set(float64(metrics.CurrentOfflineBatch))
    batchSizeGauge.WithLabelValues("reconcile").Set(float64(metrics.CurrentReconcileBatch))
    
    batchFlushCounter.WithLabelValues("all").Add(float64(metrics.TotalBatchesFlushed))
    
    batchFlushDuration.WithLabelValues("all").Observe(metrics.AvgFlushDuration.Seconds())
}
```

## Best Practices

### 1. Choose Appropriate Batch Sizes

```go
// For message sync (cross-region)
config.MessageBatchSize = 100      // Balance latency and throughput
config.MessageFlushInterval = 100 * time.Millisecond

// For offline messages (database writes)
config.OfflineBatchSize = 50       // Smaller batches for faster writes
config.OfflineFlushInterval = 200 * time.Millisecond

// For reconciliation (data consistency)
config.ReconcileBatchSize = 100    // Larger batches for efficiency
config.ReconcileFlushInterval = 500 * time.Millisecond
```

### 2. Use Priority Ordering

```go
// High priority messages are processed first
highPriorityMsg := connpool.BatchMessage{
    ID:       "urgent-001",
    Priority: 10,  // Higher priority
}

normalMsg := connpool.BatchMessage{
    ID:       "normal-001",
    Priority: 5,   // Normal priority
}

lowPriorityMsg := connpool.BatchMessage{
    ID:       "low-001",
    Priority: 1,   // Lower priority
}

// Messages are sorted by priority before processing
bp.AddMessage(highPriorityMsg)
bp.AddMessage(normalMsg)
bp.AddMessage(lowPriorityMsg)
```

### 3. Handle Errors Gracefully

```go
// Add error handling
if err := bp.AddMessage(msg); err != nil {
    log.Printf("Failed to add message: %v", err)
    // Implement fallback logic
    // - Retry with backoff
    // - Store in dead letter queue
    // - Alert monitoring system
}

// Monitor error metrics
metrics := bp.GetMetrics()
if metrics.TotalBatchErrors > threshold {
    log.Printf("High error rate detected: %d errors", metrics.TotalBatchErrors)
    // Take corrective action
}
```

### 4. Graceful Shutdown

```go
// Ensure all batches are flushed on shutdown
func shutdown(bp *connpool.BatchProcessor) {
    log.Println("Shutting down batch processor...")
    
    // Flush all pending batches
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := bp.FlushAll(ctx); err != nil {
        log.Printf("Error flushing batches: %v", err)
    }
    
    // Stop the processor
    if err := bp.Stop(); err != nil {
        log.Printf("Error stopping processor: %v", err)
    }
    
    log.Println("Batch processor stopped")
}
```

### 5. Monitor Performance

```go
// Set up periodic monitoring
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := bp.GetMetrics()
        
        // Log metrics
        log.Printf("Batch Processor Stats:")
        log.Printf("  Total batched: %d", 
            metrics.TotalMessagesBatched + 
            metrics.TotalOfflineBatched + 
            metrics.TotalReconcileBatched)
        log.Printf("  Avg batch size: %.2f", metrics.AvgBatchSize)
        log.Printf("  Avg flush duration: %v", metrics.AvgFlushDuration)
        
        // Alert on anomalies
        if metrics.AvgFlushDuration > 1*time.Second {
            log.Printf("WARNING: High flush duration detected")
        }
        
        if metrics.TotalBatchErrors > 0 {
            log.Printf("WARNING: Batch errors detected: %d", metrics.TotalBatchErrors)
        }
    }
}()
```

## Troubleshooting

### High Latency

**Problem**: Batch flush duration is too high

**Solutions**:
1. Reduce batch size
2. Increase concurrent batches
3. Enable compression
4. Check network latency
5. Optimize database queries

```go
// Reduce batch size
config.MessageBatchSize = 50  // Smaller batches

// Increase concurrency
config.MaxConcurrentBatches = 20

// Enable compression
config.EnableCompression = true
```

### Memory Usage

**Problem**: High memory consumption

**Solutions**:
1. Reduce batch sizes
2. Decrease flush intervals
3. Monitor current batch sizes
4. Implement backpressure

```go
// Reduce batch sizes
config.MessageBatchSize = 50
config.OfflineBatchSize = 25
config.ReconcileBatchSize = 50

// Faster flush intervals
config.MessageFlushInterval = 50 * time.Millisecond

// Monitor memory
metrics := bp.GetMetrics()
if metrics.CurrentMessageBatch > 80 {
    log.Println("WARNING: Message batch approaching limit")
}
```

### Batch Errors

**Problem**: High error rate

**Solutions**:
1. Check retry configuration
2. Verify network connectivity
3. Check database/Kafka health
4. Review error logs

```go
// Increase retries
config.MessageMaxRetries = 5
config.OfflineMaxRetries = 5
config.ReconcileMaxRetries = 5

// Monitor errors
metrics := bp.GetMetrics()
errorRate := float64(metrics.TotalBatchErrors) / float64(metrics.TotalBatchesFlushed)
if errorRate > 0.01 {  // 1% error rate
    log.Printf("High error rate: %.2f%%", errorRate*100)
}
```

## Performance Benchmarks

### Throughput

```
Batch Size: 100
Flush Interval: 100ms
Concurrent Batches: 10

Results:
- Messages/sec: ~10,000
- Avg batch size: 95
- Avg flush duration: 15ms
- Memory usage: ~50MB
```

### Latency

```
Batch Size: 50
Flush Interval: 50ms
Concurrent Batches: 5

Results:
- P50 latency: 25ms
- P95 latency: 75ms
- P99 latency: 150ms
- Throughput: ~5,000 messages/sec
```

## Conclusion

The Batch Processor provides a comprehensive solution for optimizing batch operations in cross-region multi-active architectures. By following this guide and best practices, you can achieve:

- **High Throughput**: Process thousands of messages per second
- **Low Latency**: Sub-100ms batch processing
- **Reliability**: Automatic retries and error handling
- **Observability**: Comprehensive metrics and monitoring
- **Scalability**: Concurrent batch processing

For more information, see:
- [Connection Pool README](README.md)
- [Cache Warming Guide](CACHE_WARMING_GUIDE.md)
- [Task 12.1 Completion Summary](TASK_12.1_COMPLETION_SUMMARY.md)
- [Task 12.2 Completion Summary](TASK_12.2_COMPLETION_SUMMARY.md)
