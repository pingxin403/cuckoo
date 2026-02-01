package sync

import (
	"fmt"
	"log"
)

// This file demonstrates how the MessageSyncer integrates with the existing
// HLC, queue, and storage components in the multi-region active-active system.

// ExampleIntegration shows how to set up and use the MessageSyncer
func ExampleIntegration() {
	// This is a conceptual example showing the integration pattern
	// In practice, you would import the actual packages:
	// import "github.com/cuckoo-org/cuckoo/libs/hlc"
	// import "github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	// import "github.com/cuckoo-org/cuckoo/examples/mvp/storage"

	fmt.Println("=== Multi-Region Message Synchronizer Integration Example ===")

	// 1. Setup HLC clocks for two regions
	fmt.Println("\n1. Setting up HLC clocks...")
	// hlcA := hlc.NewHLC("region-a", "node-1")
	// hlcB := hlc.NewHLC("region-b", "node-1")

	// 2. Setup local queues for cross-region communication
	fmt.Println("2. Setting up local queues...")
	// queueConfigA := queue.DefaultConfig("region-a")
	// queueA, _ := queue.NewLocalQueue(queueConfigA, logger)
	//
	// queueConfigB := queue.DefaultConfig("region-b")
	// queueB, _ := queue.NewLocalQueue(queueConfigB, logger)

	// 3. Setup local storage for message persistence
	fmt.Println("3. Setting up local storage...")
	// storageConfigA := storage.Config{RegionID: "region-a", MemoryMode: true}
	// storageA, _ := storage.NewLocalStore(storageConfigA)
	//
	// storageConfigB := storage.Config{RegionID: "region-b", MemoryMode: true}
	// storageB, _ := storage.NewLocalStore(storageConfigB)

	// 4. Create message syncers
	fmt.Println("4. Creating message syncers...")
	// syncerConfigA := DefaultConfig("region-a")
	// syncerA, _ := NewMessageSyncer("region-a", hlcA, queueA, storageA, syncerConfigA, logger)
	//
	// syncerConfigB := DefaultConfig("region-b")
	// syncerB, _ := NewMessageSyncer("region-b", hlcB, queueB, storageB, syncerConfigB, logger)

	// 5. Start syncers
	fmt.Println("5. Starting syncers...")
	// syncerA.Start()
	// syncerB.Start()

	// 6. Create and sync messages
	fmt.Println("6. Syncing messages...")
	// Regular message (async sync)
	// message := storage.LocalMessage{
	//     MsgID:            "msg-001",
	//     SenderID:         "user-123",
	//     ConversationID:   "conv-456",
	//     Content:          "Hello from region A!",
	//     SequenceNumber:   1,
	//     Timestamp:        time.Now().UnixMilli(),
	// }
	// syncerA.SyncMessageAsync(context.Background(), "region-b", message)

	// Critical message (sync with acknowledgment)
	// criticalMessage := storage.LocalMessage{
	//     MsgID:            "payment-001",
	//     SenderID:         "system",
	//     ConversationID:   "payment-conv",
	//     Content:          "Payment processed: $100.00",
	//     SequenceNumber:   1,
	//     Timestamp:        time.Now().UnixMilli(),
	// }
	// syncerA.SyncMessageSync(context.Background(), "region-b", criticalMessage)

	// 7. Monitor metrics
	fmt.Println("7. Monitoring metrics...")
	// metrics := syncerA.GetMetrics()
	// fmt.Printf("Async syncs: %d, Sync syncs: %d, Conflicts: %d\n",
	//     metrics["async_sync_count"],
	//     metrics["sync_sync_count"],
	//     metrics["conflict_count"])

	fmt.Println("\n=== Integration Example Complete ===")
}

// ExampleConflictResolution demonstrates how conflicts are detected and resolved
func ExampleConflictResolution() {
	fmt.Println("\n=== Conflict Resolution Example ===")

	// Scenario: Same message ID updated in both regions simultaneously
	fmt.Println("Scenario: Concurrent updates to same message")

	// Region A updates message-001 at HLC timestamp 1640995200000-0
	fmt.Println("Region A: message-001 updated at HLC 1640995200000-0")

	// Region B updates message-001 at HLC timestamp 1640995200000-1
	fmt.Println("Region B: message-001 updated at HLC 1640995200000-1")

	// When Region A syncs to Region B:
	// 1. Conflict detected (same message ID, different versions)
	// 2. HLC comparison: 1640995200000-1 > 1640995200000-0
	// 3. Resolution: Region B wins (higher logical counter)
	// 4. Region A's version is updated to match Region B
	// 5. Conflict is logged for monitoring

	fmt.Println("Resolution: Region B wins (higher HLC logical counter)")
	fmt.Println("Conflict logged and metrics updated")

	// Example conflict info that would be recorded:
	// conflict := ConflictInfo{
	//     MessageID:     "message-001",
	//     LocalVersion:  1,
	//     RemoteVersion: 1,
	//     LocalRegion:   "region-a",
	//     RemoteRegion:  "region-b",
	//     Resolution:    "remote_wins",
	//     ConflictTime:  time.Now(),
	// }

	fmt.Println("=== Conflict Resolution Complete ===")
}

// ExampleNetworkPartition demonstrates behavior during network issues
func ExampleNetworkPartition() {
	fmt.Println("\n=== Network Partition Handling Example ===")

	// Scenario: Network partition between regions
	fmt.Println("Scenario: Network partition occurs")

	// 1. Messages continue to be processed locally
	fmt.Println("1. Local processing continues in both regions")

	// 2. Sync messages are queued locally (buffered)
	fmt.Println("2. Cross-region sync messages buffered locally")

	// 3. Network partition heals
	fmt.Println("3. Network partition heals")

	// 4. Buffered messages are synchronized
	fmt.Println("4. Buffered messages synchronized")

	// 5. Conflicts detected and resolved using HLC timestamps
	fmt.Println("5. Conflicts resolved using HLC ordering")

	// 6. System returns to normal operation
	fmt.Println("6. Normal operation resumed")

	fmt.Println("=== Network Partition Handling Complete ===")
}

// ExampleMetricsAndMonitoring shows the monitoring capabilities
func ExampleMetricsAndMonitoring() {
	fmt.Println("\n=== Metrics and Monitoring Example ===")

	// Key metrics exposed by MessageSyncer:
	metrics := map[string]interface{}{
		"region_id":           "region-a",
		"async_sync_count":    int64(1250),
		"sync_sync_count":     int64(45),
		"conflict_count":      int64(3),
		"error_count":         int64(2),
		"avg_sync_latency_ms": int64(150),
		"started":             true,
		"shutdown":            false,
	}

	fmt.Println("Current metrics:")
	for key, value := range metrics {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Alerting thresholds (example):
	fmt.Println("\nAlerting thresholds:")
	fmt.Println("  - Sync latency P99 > 500ms: WARNING")
	fmt.Println("  - Conflict rate > 0.1%: CRITICAL")
	fmt.Println("  - Error rate > 1%: WARNING")
	fmt.Println("  - Queue buffer > 80%: WARNING")

	fmt.Println("=== Metrics and Monitoring Complete ===")
}

// ExampleUsagePatterns shows common usage patterns
func ExampleUsagePatterns() {
	fmt.Println("\n=== Usage Patterns Example ===")

	// Pattern 1: Regular chat messages (async)
	fmt.Println("Pattern 1: Regular chat messages")
	fmt.Println("  - Use SyncMessageAsync()")
	fmt.Println("  - Fire-and-forget semantics")
	fmt.Println("  - High throughput, eventual consistency")
	fmt.Println("  - RPO ≈ 0 (<1 second)")

	// Pattern 2: Critical business operations (sync)
	fmt.Println("\nPattern 2: Critical business operations")
	fmt.Println("  - Use SyncMessageSync()")
	fmt.Println("  - Wait for acknowledgment")
	fmt.Println("  - Strong consistency guarantees")
	fmt.Println("  - RPO = 0 (no data loss)")

	// Pattern 3: Batch operations
	fmt.Println("\nPattern 3: Batch operations")
	fmt.Println("  - Process multiple messages together")
	fmt.Println("  - Improved throughput")
	fmt.Println("  - Reduced network overhead")

	// Pattern 4: Monitoring and alerting
	fmt.Println("\nPattern 4: Monitoring and alerting")
	fmt.Println("  - Regular metrics collection")
	fmt.Println("  - Threshold-based alerting")
	fmt.Println("  - Conflict rate monitoring")
	fmt.Println("  - Performance tracking")

	fmt.Println("=== Usage Patterns Complete ===")
}

// RunAllExamples runs all integration examples
func RunAllExamples() {
	logger := log.New(log.Writer(), "[Example] ", log.LstdFlags)
	logger.Println("Starting MessageSyncer integration examples...")

	ExampleIntegration()
	ExampleConflictResolution()
	ExampleNetworkPartition()
	ExampleMetricsAndMonitoring()
	ExampleUsagePatterns()

	logger.Println("All examples completed successfully!")
}
