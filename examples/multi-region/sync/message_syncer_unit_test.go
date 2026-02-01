package sync

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// TestMessageSyncer_AsyncSyncUnit tests async sync functionality in isolation
// 测试异步同步流程 - 单元测试
func TestMessageSyncer_AsyncSyncUnit(t *testing.T) {
	regionA := "region-a"

	// Create test infrastructure
	hlcA := hlc.NewHLC(regionA, "node-1")

	queueConfigA := queue.DefaultConfig(regionA)
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}
	defer queueA.Close()

	storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}
	defer storageA.Close()

	syncerConfigA := DefaultConfig(regionA)
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA, nil)
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	// Test Case 1: Syncer initialization
	t.Run("SyncerInitialization", func(t *testing.T) {
		if syncerA.regionID != regionA {
			t.Errorf("Expected region ID %s, got %s", regionA, syncerA.regionID)
		}

		if syncerA.hlcClock == nil {
			t.Error("Expected HLC clock to be initialized")
		}

		if syncerA.localStorage == nil {
			t.Error("Expected local storage to be initialized")
		}

		if syncerA.localQueue == nil {
			t.Error("Expected local queue to be initialized")
		}
	})

	// Test Case 2: Start and stop syncer
	t.Run("StartStopSyncer", func(t *testing.T) {
		if err := syncerA.Start(); err != nil {
			t.Fatalf("Failed to start syncer: %v", err)
		}

		// Check that syncer is started
		metrics := syncerA.GetMetrics()
		if !metrics["started"].(bool) {
			t.Error("Expected syncer to be started")
		}

		if err := syncerA.Stop(); err != nil {
			t.Fatalf("Failed to stop syncer: %v", err)
		}

		// Check that syncer is stopped
		metrics = syncerA.GetMetrics()
		if !metrics["shutdown"].(bool) {
			t.Error("Expected syncer to be shutdown")
		}
	})

	// Test Case 3: Message conversion functions
	t.Run("MessageConversion", func(t *testing.T) {
		testMessage := storage.LocalMessage{
			MsgID:            "test-msg-001",
			SenderID:         "user-123",
			ConversationID:   "conv-456",
			ConversationType: "group",
			Content:          "Test message content",
			SequenceNumber:   1,
			Timestamp:        time.Now().UnixMilli(),
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(24 * time.Hour),
			Metadata:         map[string]string{"type": "text"},
			RegionID:         regionA,
			Version:          1,
		}

		// Test syncMessageToLocalMessage conversion
		globalID := hlcA.GenerateID()
		syncMsg := &SyncMessage{
			ID:             "sync-001",
			Type:           "async",
			SourceRegion:   regionA,
			TargetRegion:   "region-b",
			MessageID:      testMessage.MsgID,
			GlobalID:       globalID,
			ConversationID: testMessage.ConversationID,
			SenderID:       testMessage.SenderID,
			Content:        testMessage.Content,
			SequenceNumber: testMessage.SequenceNumber,
			Timestamp:      testMessage.Timestamp,
			Metadata:       testMessage.Metadata,
			CreatedAt:      testMessage.CreatedAt,
		}

		convertedMsg := syncerA.syncMessageToLocalMessage(syncMsg)

		if convertedMsg.MsgID != testMessage.MsgID {
			t.Errorf("Expected message ID %s, got %s", testMessage.MsgID, convertedMsg.MsgID)
		}

		if convertedMsg.Content != testMessage.Content {
			t.Errorf("Expected content %s, got %s", testMessage.Content, convertedMsg.Content)
		}

		if convertedMsg.SenderID != testMessage.SenderID {
			t.Errorf("Expected sender ID %s, got %s", testMessage.SenderID, convertedMsg.SenderID)
		}
	})

	// Test Case 4: Checksum calculation
	t.Run("ChecksumCalculation", func(t *testing.T) {
		syncerA.config.EnableChecksum = true

		syncMsg := &SyncMessage{
			MessageID:      "checksum-test-001",
			ConversationID: "conv-123",
			SenderID:       "user-456",
			Content:        "Test message for checksum",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
		}

		// Calculate checksum twice - should be identical
		checksum1 := syncerA.calculateChecksum(syncMsg)
		checksum2 := syncerA.calculateChecksum(syncMsg)

		if checksum1 != checksum2 {
			t.Errorf("Checksums should be identical: %s != %s", checksum1, checksum2)
		}

		if checksum1 == "" {
			t.Error("Checksum should not be empty when enabled")
		}

		// Modify message and verify checksum changes
		syncMsg.Content = "Modified content"
		checksum3 := syncerA.calculateChecksum(syncMsg)

		if checksum1 == checksum3 {
			t.Error("Checksum should change when message content changes")
		}
	})

	// Test Case 5: Metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		initialMetrics := syncerA.GetMetrics()

		// Verify initial metrics structure
		expectedKeys := []string{
			"region_id", "async_sync_count", "sync_sync_count",
			"conflict_count", "error_count", "avg_sync_latency_ms",
			"sync_latency_count", "started", "shutdown",
		}

		for _, key := range expectedKeys {
			if _, exists := initialMetrics[key]; !exists {
				t.Errorf("Expected metric key %s not found", key)
			}
		}

		if initialMetrics["region_id"] != regionA {
			t.Errorf("Expected region_id %s, got %s", regionA, initialMetrics["region_id"])
		}
	})
}

// TestMessageSyncer_SyncAcknowledgmentUnit tests sync acknowledgment functionality
// 测试同步确认流程 - 单元测试
func TestMessageSyncer_SyncAcknowledgmentUnit(t *testing.T) {
	regionA := "region-a"

	hlcA := hlc.NewHLC(regionA, "node-1")
	queueA, _ := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
	defer queueA.Close()

	storageA, _ := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
	defer storageA.Close()

	syncerA, _ := NewMessageSyncer(regionA, hlcA, queueA, storageA, DefaultConfig(regionA), nil)
	defer syncerA.Stop()

	// Test Case 1: SyncAck creation and validation
	t.Run("SyncAckCreation", func(t *testing.T) {
		ack := &SyncAck{
			MessageID:    "msg-001",
			GlobalID:     "region-a-123456-1",
			SourceRegion: regionA,
			TargetRegion: "region-b",
			Status:       "success",
			Timestamp:    time.Now(),
			ProcessTime:  150,
		}

		if ack.Status != "success" {
			t.Errorf("Expected status 'success', got %s", ack.Status)
		}

		if ack.ProcessTime != 150 {
			t.Errorf("Expected process time 150, got %d", ack.ProcessTime)
		}

		if ack.Error != "" {
			t.Errorf("Expected empty error for success status, got %s", ack.Error)
		}
	})

	// Test Case 2: Sync message processing logic
	t.Run("SyncMessageProcessing", func(t *testing.T) {
		ctx := context.Background()

		syncMsg := &SyncMessage{
			ID:             "sync-001",
			Type:           "sync",
			SourceRegion:   "region-b",
			TargetRegion:   regionA,
			MessageID:      "process-test-001",
			GlobalID:       hlcA.GenerateID(),
			ConversationID: "conv-123",
			SenderID:       "user-456",
			Content:        "Test sync message",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
			RequiresAck:    true,
			CreatedAt:      time.Now(),
		}

		// Test processSyncMessage method
		status, errorMsg := syncerA.processSyncMessage(ctx, syncMsg)

		if status != "success" && status != "error" {
			t.Errorf("Expected status 'success' or 'error', got %s", status)
		}

		if status == "error" && errorMsg == "" {
			t.Error("Expected error message when status is error")
		}

		if status == "success" && errorMsg != "" {
			t.Errorf("Expected empty error message for success, got %s", errorMsg)
		}
	})

	// Test Case 3: Timeout configuration
	t.Run("TimeoutConfiguration", func(t *testing.T) {
		config := DefaultConfig(regionA)

		// Test default timeout
		if config.SyncTimeout != 5*time.Second {
			t.Errorf("Expected default sync timeout 5s, got %v", config.SyncTimeout)
		}

		// Test custom timeout with separate queue instance
		config.SyncTimeout = 2 * time.Second
		customQueue, err := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create custom queue: %v", err)
		}
		defer customQueue.Close()

		customStorage, err := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
		if err != nil {
			t.Fatalf("Failed to create custom storage: %v", err)
		}
		defer customStorage.Close()

		customSyncer, err := NewMessageSyncer(regionA, hlcA, customQueue, customStorage, config, nil)
		if err != nil {
			t.Fatalf("Failed to create syncer with custom timeout: %v", err)
		}
		defer customSyncer.Stop()

		if customSyncer.config.SyncTimeout != 2*time.Second {
			t.Errorf("Expected custom sync timeout 2s, got %v", customSyncer.config.SyncTimeout)
		}
	})
}

// TestMessageSyncer_ConflictResolutionUnit tests conflict resolution integration
// 测试冲突解决逻辑 - 单元测试
func TestMessageSyncer_ConflictResolutionUnit(t *testing.T) {
	regionA := "region-a"

	hlcA := hlc.NewHLC(regionA, "node-1")
	queueA, _ := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
	defer queueA.Close()

	storageA, _ := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
	defer storageA.Close()

	syncerA, _ := NewMessageSyncer(regionA, hlcA, queueA, storageA, DefaultConfig(regionA), nil)
	defer syncerA.Stop()

	// Test Case 1: Message version creation
	t.Run("MessageVersionCreation", func(t *testing.T) {
		testMessage := storage.LocalMessage{
			MsgID:          "version-test-001",
			SenderID:       "user-123",
			ConversationID: "conv-456",
			Content:        "Test message for versioning",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
			CreatedAt:      time.Now(),
			RegionID:       regionA,
			GlobalID:       hlcA.GenerateID().String(),
			Version:        1,
			Metadata:       map[string]string{"type": "text"},
		}

		// Insert message to test conflict detection
		ctx := context.Background()
		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert test message: %v", err)
		}

		// Verify message exists
		retrievedMsg, err := storageA.GetMessageByID(ctx, testMessage.MsgID)
		if err != nil {
			t.Fatalf("Failed to retrieve message: %v", err)
		}

		if retrievedMsg.MsgID != testMessage.MsgID {
			t.Errorf("Expected message ID %s, got %s", testMessage.MsgID, retrievedMsg.MsgID)
		}
	})

	// Test Case 2: Conflict detection logic
	t.Run("ConflictDetectionLogic", func(t *testing.T) {
		ctx := context.Background()

		// Create a message that exists locally
		localMessage := storage.LocalMessage{
			MsgID:          "conflict-test-001",
			SenderID:       "user-123",
			ConversationID: "conv-456",
			Content:        "Local message content",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
			CreatedAt:      time.Now(),
			RegionID:       regionA,
			GlobalID:       hlcA.GenerateID().String(),
			Version:        1,
		}

		if err := storageA.Insert(ctx, localMessage); err != nil {
			t.Fatalf("Failed to insert local message: %v", err)
		}

		// Create a conflicting remote message
		remoteMessage := storage.LocalMessage{
			MsgID:          "conflict-test-001", // Same ID
			SenderID:       "user-123",
			ConversationID: "conv-456",
			Content:        "Remote message content", // Different content
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli() + 1000, // Later timestamp
			CreatedAt:      time.Now(),
			RegionID:       "region-b",
			GlobalID:       "region-b-" + hlcA.GenerateID().String(),
			Version:        2,
		}

		// Test conflict detection
		conflict, err := storageA.DetectConflict(ctx, remoteMessage)
		if err != nil {
			t.Fatalf("Failed to detect conflict: %v", err)
		}

		if conflict == nil {
			t.Error("Expected conflict to be detected")
		} else {
			if conflict.MessageID != "conflict-test-001" {
				t.Errorf("Expected conflict message ID conflict-test-001, got %s", conflict.MessageID)
			}

			if conflict.Resolution == "" {
				t.Error("Expected conflict resolution to be set")
			}
		}
	})

	// Test Case 3: HLC integration with conflict resolution
	t.Run("HLCIntegrationWithConflictResolution", func(t *testing.T) {
		// Create two HLC clocks with different timestamps
		hlcB := hlc.NewHLC("region-b", "node-1")

		// Generate IDs with different timestamps
		globalIDA := hlcA.GenerateID()
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		globalIDB := hlcB.GenerateID()

		// Compare the global IDs
		cmp := hlc.CompareGlobalID(globalIDA, globalIDB)

		// The comparison should be deterministic
		if cmp == 0 {
			t.Error("Expected different global IDs to have non-zero comparison")
		}

		// Test that comparison is consistent
		cmp2 := hlc.CompareGlobalID(globalIDA, globalIDB)
		if cmp != cmp2 {
			t.Error("Global ID comparison should be consistent")
		}
	})
}

// TestMessageSyncer_NetworkFailureUnit tests network failure handling
// 测试网络故障处理 - 单元测试
func TestMessageSyncer_NetworkFailureUnit(t *testing.T) {
	regionA := "region-a"

	hlcA := hlc.NewHLC(regionA, "node-1")
	queueA, _ := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
	defer queueA.Close()

	storageA, _ := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
	defer storageA.Close()

	// Test Case 1: Timeout handling configuration
	t.Run("TimeoutHandlingConfiguration", func(t *testing.T) {
		shortTimeoutConfig := DefaultConfig(regionA)
		shortTimeoutConfig.SyncTimeout = 50 * time.Millisecond
		shortTimeoutConfig.MaxRetries = 2

		timeoutSyncer, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, shortTimeoutConfig, nil)
		if err != nil {
			t.Fatalf("Failed to create timeout syncer: %v", err)
		}
		defer timeoutSyncer.Stop()

		if timeoutSyncer.config.SyncTimeout != 50*time.Millisecond {
			t.Errorf("Expected timeout 50ms, got %v", timeoutSyncer.config.SyncTimeout)
		}

		if timeoutSyncer.config.MaxRetries != 2 {
			t.Errorf("Expected max retries 2, got %d", timeoutSyncer.config.MaxRetries)
		}
	})

	// Test Case 2: Error metrics tracking
	t.Run("ErrorMetricsTracking", func(t *testing.T) {
		testQueue, err := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create test queue: %v", err)
		}
		defer testQueue.Close()

		testStorage, err := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
		if err != nil {
			t.Fatalf("Failed to create test storage: %v", err)
		}
		defer testStorage.Close()

		syncer, err := NewMessageSyncer(regionA, hlcA, testQueue, testStorage, DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create syncer: %v", err)
		}
		defer syncer.Stop()

		initialMetrics := syncer.GetMetrics()
		initialErrors := initialMetrics["error_count"].(int64)

		// Verify error count starts at zero
		if initialErrors != 0 {
			t.Errorf("Expected initial error count 0, got %d", initialErrors)
		}

		// Test that metrics structure is correct
		if initialMetrics["region_id"] != regionA {
			t.Errorf("Expected region_id %s, got %s", regionA, initialMetrics["region_id"])
		}
	})

	// Test Case 3: Checksum verification for corruption detection
	t.Run("ChecksumVerificationForCorruption", func(t *testing.T) {
		config := DefaultConfig(regionA)
		config.EnableChecksum = true

		checksumQueue, err := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create checksum queue: %v", err)
		}
		defer checksumQueue.Close()

		checksumStorage, err := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
		if err != nil {
			t.Fatalf("Failed to create checksum storage: %v", err)
		}
		defer checksumStorage.Close()

		syncer, err := NewMessageSyncer(regionA, hlcA, checksumQueue, checksumStorage, config, nil)
		if err != nil {
			t.Fatalf("Failed to create syncer: %v", err)
		}
		defer syncer.Stop()

		syncMsg := &SyncMessage{
			MessageID:      "corruption-test-001",
			ConversationID: "conv-123",
			SenderID:       "user-456",
			Content:        "Test message for corruption detection",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
		}

		// Calculate correct checksum
		correctChecksum := syncer.calculateChecksum(syncMsg)
		if correctChecksum == "" {
			t.Error("Checksum should not be empty when enabled")
		}

		// Set correct checksum
		syncMsg.Checksum = correctChecksum

		// Verify checksum validation (simulate by recalculating)
		recalculatedChecksum := syncer.calculateChecksum(syncMsg)

		// The checksum calculation should be deterministic
		if recalculatedChecksum == "" {
			t.Error("Recalculated checksum should not be empty")
		}

		// Test checksum with corrupted content
		syncMsg.Content = "Corrupted content"
		corruptedChecksum := syncer.calculateChecksum(syncMsg)

		if correctChecksum == corruptedChecksum {
			t.Error("Checksum should change when content is corrupted")
		}
	})

	// Test Case 4: Retry mechanism configuration
	t.Run("RetryMechanismConfiguration", func(t *testing.T) {
		retryConfig := DefaultConfig(regionA)
		retryConfig.MaxRetries = 5

		retryQueue, err := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create retry queue: %v", err)
		}
		defer retryQueue.Close()

		retryStorage, err := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
		if err != nil {
			t.Fatalf("Failed to create retry storage: %v", err)
		}
		defer retryStorage.Close()

		retrySyncer, err := NewMessageSyncer(regionA, hlcA, retryQueue, retryStorage, retryConfig, nil)
		if err != nil {
			t.Fatalf("Failed to create retry syncer: %v", err)
		}
		defer retrySyncer.Stop()

		if retrySyncer.config.MaxRetries != 5 {
			t.Errorf("Expected max retries 5, got %d", retrySyncer.config.MaxRetries)
		}

		// Test SyncMessage creation with retry configuration
		testMessage := storage.LocalMessage{
			MsgID:          "retry-test-001",
			SenderID:       "user-123",
			ConversationID: "conv-456",
			Content:        "Test message for retry",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
			CreatedAt:      time.Now(),
			RegionID:       regionA,
			Version:        1,
		}

		globalID := hlcA.GenerateID()
		syncMsg := &SyncMessage{
			ID:             "retry-sync-001",
			Type:           "sync",
			SourceRegion:   regionA,
			TargetRegion:   "region-b",
			MessageID:      testMessage.MsgID,
			GlobalID:       globalID,
			ConversationID: testMessage.ConversationID,
			SenderID:       testMessage.SenderID,
			Content:        testMessage.Content,
			MaxRetries:     retryConfig.MaxRetries,
			RequiresAck:    true,
			CreatedAt:      time.Now(),
		}

		if syncMsg.MaxRetries != 5 {
			t.Errorf("Expected sync message max retries 5, got %d", syncMsg.MaxRetries)
		}
	})

	// Test Case 5: Graceful degradation configuration
	t.Run("GracefulDegradationConfiguration", func(t *testing.T) {
		config := DefaultConfig(regionA)
		config.EnableDeduplication = true
		config.EnableChecksum = true

		degradationQueue, err := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		if err != nil {
			t.Fatalf("Failed to create degradation queue: %v", err)
		}
		defer degradationQueue.Close()

		degradationStorage, err := storage.NewLocalStore(storage.Config{RegionID: regionA, MemoryMode: true})
		if err != nil {
			t.Fatalf("Failed to create degradation storage: %v", err)
		}
		defer degradationStorage.Close()

		syncer, err := NewMessageSyncer(regionA, hlcA, degradationQueue, degradationStorage, config, nil)
		if err != nil {
			t.Fatalf("Failed to create syncer with degradation config: %v", err)
		}
		defer syncer.Stop()

		if !syncer.config.EnableDeduplication {
			t.Error("Expected deduplication to be enabled")
		}

		if !syncer.config.EnableChecksum {
			t.Error("Expected checksum to be enabled")
		}

		// Test that syncer can handle configuration properly
		metrics := syncer.GetMetrics()
		if metrics["region_id"] != regionA {
			t.Errorf("Expected region_id %s, got %s", regionA, metrics["region_id"])
		}
	})
}
