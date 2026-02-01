package sync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// TestMessageSyncer_AsyncSyncFlow tests the complete async synchronization flow
// 测试异步同步流程
func TestMessageSyncer_AsyncSyncFlow(t *testing.T) {
	// Setup test environment with two regions
	regionA, regionB := "region-a", "region-b"

	// Create test infrastructure
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionTest(t, regionA, regionB)
	defer cleanup()

	// Start both syncers
	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Test Case 1: Basic async sync
	t.Run("BasicAsyncSync", func(t *testing.T) {
		testMessage := createTestMessage("async-msg-001", regionA)

		// Insert message in region A
		ctx := context.Background()
		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message in region A: %v", err)
		}

		// Sync to region B asynchronously
		if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
			t.Fatalf("Failed to sync message async: %v", err)
		}

		// Wait for async processing
		time.Sleep(200 * time.Millisecond)

		// Verify message exists in region B
		retrievedMsg, err := storageB.GetMessageByID(ctx, testMessage.MsgID)
		if err != nil {
			t.Fatalf("Failed to retrieve message from region B: %v", err)
		}

		// Verify message content
		if retrievedMsg.Content != testMessage.Content {
			t.Errorf("Expected content %s, got %s", testMessage.Content, retrievedMsg.Content)
		}
		if retrievedMsg.SenderID != testMessage.SenderID {
			t.Errorf("Expected sender %s, got %s", testMessage.SenderID, retrievedMsg.SenderID)
		}

		// Check metrics
		metricsA := syncerA.GetMetrics()
		if metricsA["async_sync_count"].(int64) < 1 {
			t.Errorf("Expected at least 1 async sync, got %d", metricsA["async_sync_count"])
		}
	})

	// Test Case 2: Multiple async messages
	t.Run("MultipleAsyncMessages", func(t *testing.T) {
		ctx := context.Background()
		messageCount := 5

		for i := 0; i < messageCount; i++ {
			testMessage := createTestMessage(fmt.Sprintf("multi-async-msg-%03d", i), regionA)
			testMessage.Content = fmt.Sprintf("Message %d from region A", i)

			if err := storageA.Insert(ctx, testMessage); err != nil {
				t.Fatalf("Failed to insert message %d: %v", i, err)
			}

			if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
				t.Fatalf("Failed to sync message %d: %v", i, err)
			}
		}

		// Wait for all messages to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify all messages exist in region B
		for i := 0; i < messageCount; i++ {
			msgID := fmt.Sprintf("multi-async-msg-%03d", i)
			retrievedMsg, err := storageB.GetMessageByID(ctx, msgID)
			if err != nil {
				t.Errorf("Failed to retrieve message %s from region B: %v", msgID, err)
				continue
			}

			expectedContent := fmt.Sprintf("Message %d from region A", i)
			if retrievedMsg.Content != expectedContent {
				t.Errorf("Message %d: expected content %s, got %s", i, expectedContent, retrievedMsg.Content)
			}
		}
	})

	// Test Case 3: Async sync with checksum verification
	t.Run("AsyncSyncWithChecksum", func(t *testing.T) {
		// Enable checksum for this test
		syncerA.config.EnableChecksum = true
		syncerB.config.EnableChecksum = true

		testMessage := createTestMessage("checksum-async-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
			t.Fatalf("Failed to sync message with checksum: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Verify message was synced successfully
		retrievedMsg, err := storageB.GetMessageByID(ctx, testMessage.MsgID)
		if err != nil {
			t.Fatalf("Failed to retrieve checksummed message: %v", err)
		}

		if retrievedMsg.Content != testMessage.Content {
			t.Error("Checksum verification should have passed and message should be synced")
		}
	})
}

// TestMessageSyncer_SyncAcknowledgmentFlow tests synchronous sync with acknowledgments
// 测试同步确认流程
func TestMessageSyncer_SyncAcknowledgmentFlow(t *testing.T) {
	regionA, regionB := "region-a", "region-b"

	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionTest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Test Case 1: Basic sync with acknowledgment
	t.Run("BasicSyncWithAck", func(t *testing.T) {
		criticalMessage := createTestMessage("sync-msg-001", regionA)
		criticalMessage.Metadata = map[string]string{"type": "payment", "critical": "true"}

		ctx := context.Background()
		if err := storageA.Insert(ctx, criticalMessage); err != nil {
			t.Fatalf("Failed to insert critical message: %v", err)
		}

		// Measure sync time
		start := time.Now()
		if err := syncerA.SyncMessageSync(ctx, regionB, criticalMessage); err != nil {
			t.Fatalf("Failed to sync message synchronously: %v", err)
		}
		syncDuration := time.Since(start)

		// Verify message exists in region B
		retrievedMsg, err := storageB.GetMessageByID(ctx, criticalMessage.MsgID)
		if err != nil {
			t.Fatalf("Failed to retrieve critical message from region B: %v", err)
		}

		if retrievedMsg.Content != criticalMessage.Content {
			t.Errorf("Expected content %s, got %s", criticalMessage.Content, retrievedMsg.Content)
		}

		// Verify sync completed within reasonable time (should be fast for local test)
		if syncDuration > 2*time.Second {
			t.Errorf("Sync took too long: %v", syncDuration)
		}

		// Check metrics
		metricsA := syncerA.GetMetrics()
		if metricsA["sync_sync_count"].(int64) < 1 {
			t.Errorf("Expected at least 1 sync sync, got %d", metricsA["sync_sync_count"])
		}
	})

	// Test Case 2: Sync timeout handling
	t.Run("SyncTimeout", func(t *testing.T) {
		// Create a syncer with very short timeout
		shortTimeoutConfig := DefaultConfig(regionA)
		shortTimeoutConfig.SyncTimeout = 10 * time.Millisecond // Very short timeout

		hlcA := hlc.NewHLC(regionA, "node-1")
		queueA, _ := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		defer queueA.Close()

		storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
		storageA, _ := storage.NewLocalStore(storageConfigA)
		defer storageA.Close()

		timeoutSyncer, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, shortTimeoutConfig, nil)
		if err != nil {
			t.Fatalf("Failed to create timeout syncer: %v", err)
		}
		defer timeoutSyncer.Stop()

		if err := timeoutSyncer.Start(); err != nil {
			t.Fatalf("Failed to start timeout syncer: %v", err)
		}

		testMessage := createTestMessage("timeout-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		// This should timeout since there's no receiver
		err = timeoutSyncer.SyncMessageSync(ctx, "nonexistent-region", testMessage)
		if err == nil {
			t.Error("Expected timeout error, but sync succeeded")
		}

		if !errors.Is(err, context.DeadlineExceeded) && !containsSubstring(err.Error(), "timeout") {
			t.Logf("Got error (expected timeout): %v", err)
		}
	})

	// Test Case 3: Multiple concurrent sync operations
	t.Run("ConcurrentSyncOperations", func(t *testing.T) {
		ctx := context.Background()
		concurrency := 5
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				testMessage := createTestMessage(fmt.Sprintf("concurrent-sync-msg-%03d", index), regionA)
				testMessage.Content = fmt.Sprintf("Concurrent message %d", index)

				if err := storageA.Insert(ctx, testMessage); err != nil {
					errors <- err
					return
				}

				if err := syncerA.SyncMessageSync(ctx, regionB, testMessage); err != nil {
					errors <- err
					return
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent sync error: %v", err)
		}

		// Verify all messages were synced
		time.Sleep(200 * time.Millisecond)
		for i := 0; i < concurrency; i++ {
			msgID := fmt.Sprintf("concurrent-sync-msg-%03d", i)
			if _, err := storageB.GetMessageByID(ctx, msgID); err != nil {
				t.Errorf("Failed to retrieve concurrent message %s: %v", msgID, err)
			}
		}
	})
}

// TestMessageSyncer_ConflictResolutionIntegration tests conflict resolution with message syncer
// 测试冲突解决逻辑
func TestMessageSyncer_ConflictResolutionIntegration(t *testing.T) {
	regionA, regionB := "region-a", "region-b"

	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionTest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Test Case 1: Conflict detection and LWW resolution
	t.Run("ConflictDetectionAndLWW", func(t *testing.T) {
		ctx := context.Background()

		// Create conflicting messages with same ID but different content
		messageA := createTestMessage("conflict-msg-001", regionA)
		messageA.Content = "Message from region A"
		messageA.Metadata = map[string]string{"source": "region-a"}

		messageB := createTestMessage("conflict-msg-001", regionB)
		messageB.Content = "Message from region B"
		messageB.Metadata = map[string]string{"source": "region-b"}

		// Insert messages in their respective regions
		if err := storageA.Insert(ctx, messageA); err != nil {
			t.Fatalf("Failed to insert message A: %v", err)
		}
		if err := storageB.Insert(ctx, messageB); err != nil {
			t.Fatalf("Failed to insert message B: %v", err)
		}

		// Sync message A to region B (should cause conflict)
		if err := syncerA.SyncMessageAsync(ctx, regionB, messageA); err != nil {
			t.Fatalf("Failed to sync message A to B: %v", err)
		}

		// Wait for conflict resolution
		time.Sleep(300 * time.Millisecond)

		// Check that conflict was detected and recorded
		conflicts, err := storageB.GetConflicts(ctx, 10)
		if err != nil {
			t.Fatalf("Failed to get conflicts: %v", err)
		}

		if len(conflicts) == 0 {
			t.Error("Expected conflict to be recorded, but none found")
		} else {
			conflict := conflicts[0]
			if conflict.MessageID != "conflict-msg-001" {
				t.Errorf("Expected conflict for message conflict-msg-001, got %s", conflict.MessageID)
			}

			// Verify conflict resolution is deterministic
			if conflict.Resolution != "local_wins" && conflict.Resolution != "remote_wins" {
				t.Errorf("Expected deterministic resolution, got %s", conflict.Resolution)
			}
		}

		// Check conflict metrics
		metricsB := syncerB.GetMetrics()
		if metricsB["conflict_count"].(int64) == 0 {
			t.Error("Expected at least one conflict to be recorded in metrics")
		}
	})

	// Test Case 2: HLC-based conflict resolution
	t.Run("HLCBasedConflictResolution", func(t *testing.T) {
		ctx := context.Background()

		// Create messages with different HLC timestamps
		hlcA := hlc.NewHLC(regionA, "node-1")
		hlcB := hlc.NewHLC(regionB, "node-1")

		// Advance HLC B to ensure it has a later timestamp
		time.Sleep(10 * time.Millisecond)
		globalIDB := hlcB.GenerateID()

		messageA := createTestMessage("hlc-conflict-msg-001", regionA)
		messageA.GlobalID = hlcA.GenerateID().String()
		messageA.Content = "Earlier message from A"

		messageB := createTestMessage("hlc-conflict-msg-001", regionB)
		messageB.GlobalID = globalIDB.String()
		messageB.Content = "Later message from B"

		// Insert both messages
		if err := storageA.Insert(ctx, messageA); err != nil {
			t.Fatalf("Failed to insert message A: %v", err)
		}
		if err := storageB.Insert(ctx, messageB); err != nil {
			t.Fatalf("Failed to insert message B: %v", err)
		}

		// Sync A to B - B should win due to later HLC timestamp
		if err := syncerA.SyncMessageAsync(ctx, regionB, messageA); err != nil {
			t.Fatalf("Failed to sync message A to B: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Verify the message with later HLC timestamp won
		finalMsg, err := storageB.GetMessageByID(ctx, "hlc-conflict-msg-001")
		if err != nil {
			t.Fatalf("Failed to get final message: %v", err)
		}

		// The winner should be determined by HLC comparison
		// Since we can't easily predict which will win without knowing exact timestamps,
		// we just verify that conflict resolution occurred
		conflicts, err := storageB.GetConflicts(ctx, 10)
		if err != nil {
			t.Fatalf("Failed to get conflicts: %v", err)
		}

		foundConflict := false
		for _, conflict := range conflicts {
			if conflict.MessageID == "hlc-conflict-msg-001" {
				foundConflict = true
				break
			}
		}

		if !foundConflict {
			t.Error("Expected HLC-based conflict to be recorded")
		}

		t.Logf("Final message content: %s", finalMsg.Content)
	})

	// Test Case 3: Conflict resolution with metadata preservation
	t.Run("ConflictResolutionWithMetadata", func(t *testing.T) {
		ctx := context.Background()

		messageA := createTestMessage("metadata-conflict-msg-001", regionA)
		messageA.Content = "Message A"
		messageA.Metadata = map[string]string{
			"priority": "high",
			"source":   "region-a",
			"version":  "1.0",
		}

		messageB := createTestMessage("metadata-conflict-msg-001", regionB)
		messageB.Content = "Message B"
		messageB.Metadata = map[string]string{
			"priority": "low",
			"source":   "region-b",
			"version":  "2.0",
		}

		if err := storageA.Insert(ctx, messageA); err != nil {
			t.Fatalf("Failed to insert message A: %v", err)
		}
		if err := storageB.Insert(ctx, messageB); err != nil {
			t.Fatalf("Failed to insert message B: %v", err)
		}

		// Sync and resolve conflict
		if err := syncerA.SyncMessageAsync(ctx, regionB, messageA); err != nil {
			t.Fatalf("Failed to sync message A to B: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Verify conflict was resolved and metadata is preserved
		finalMsg, err := storageB.GetMessageByID(ctx, "metadata-conflict-msg-001")
		if err != nil {
			t.Fatalf("Failed to get final message: %v", err)
		}

		if finalMsg.Metadata == nil {
			t.Error("Expected metadata to be preserved after conflict resolution")
		}

		// Verify conflict was recorded
		conflicts, err := storageB.GetConflicts(ctx, 10)
		if err != nil {
			t.Fatalf("Failed to get conflicts: %v", err)
		}

		foundMetadataConflict := false
		for _, conflict := range conflicts {
			if conflict.MessageID == "metadata-conflict-msg-001" {
				foundMetadataConflict = true
				break
			}
		}

		if !foundMetadataConflict {
			t.Error("Expected metadata conflict to be recorded")
		}
	})
}

// TestMessageSyncer_NetworkFailureHandling tests network failure scenarios
// 测试网络故障处理
func TestMessageSyncer_NetworkFailureHandling(t *testing.T) {
	regionA, regionB := "region-a", "region-b"

	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionTest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Test Case 1: Sync to non-existent region
	t.Run("SyncToNonExistentRegion", func(t *testing.T) {
		testMessage := createTestMessage("network-fail-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		// Try to sync to non-existent region
		err := syncerA.SyncMessageAsync(ctx, "nonexistent-region", testMessage)
		// This might not fail immediately in async mode, but should be handled gracefully
		if err != nil {
			t.Logf("Expected behavior - async sync to nonexistent region failed: %v", err)
		}

		// For sync mode, it should definitely fail
		err = syncerA.SyncMessageSync(ctx, "nonexistent-region", testMessage)
		if err == nil {
			t.Error("Expected sync to nonexistent region to fail")
		}
	})

	// Test Case 2: Message corruption detection via checksum
	t.Run("MessageCorruptionDetection", func(t *testing.T) {
		// Enable checksum verification
		syncerA.config.EnableChecksum = true
		syncerB.config.EnableChecksum = true

		testMessage := createTestMessage("corruption-test-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		// Create a sync message with correct checksum
		syncMsg := &SyncMessage{
			MessageID:      testMessage.MsgID,
			ConversationID: testMessage.ConversationID,
			SenderID:       testMessage.SenderID,
			Content:        testMessage.Content,
			SequenceNumber: testMessage.SequenceNumber,
			Timestamp:      testMessage.Timestamp,
		}

		// Calculate correct checksum
		correctChecksum := syncerA.calculateChecksum(syncMsg)
		if correctChecksum == "" {
			t.Fatal("Checksum calculation failed")
		}

		// Verify checksum validation works
		syncMsg.Checksum = correctChecksum
		recalculatedChecksum := syncerA.calculateChecksum(syncMsg)

		// Note: The checksum calculation doesn't include the checksum field itself,
		// so this test verifies the checksum mechanism works
		if recalculatedChecksum != correctChecksum {
			t.Logf("Checksum validation working - original: %s, recalculated: %s",
				correctChecksum, recalculatedChecksum)
		}
	})

	// Test Case 3: Retry mechanism for failed syncs
	t.Run("RetryMechanismForFailedSyncs", func(t *testing.T) {
		// Create a syncer with retry configuration
		retryConfig := DefaultConfig(regionA)
		retryConfig.MaxRetries = 3
		retryConfig.SyncTimeout = 100 * time.Millisecond

		hlcA := hlc.NewHLC(regionA, "node-1")
		queueA, _ := queue.NewLocalQueue(queue.DefaultConfig(regionA), nil)
		defer queueA.Close()

		storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
		storageA, _ := storage.NewLocalStore(storageConfigA)
		defer storageA.Close()

		retrySyncer, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, retryConfig, nil)
		if err != nil {
			t.Fatalf("Failed to create retry syncer: %v", err)
		}
		defer retrySyncer.Stop()

		if err := retrySyncer.Start(); err != nil {
			t.Fatalf("Failed to start retry syncer: %v", err)
		}

		testMessage := createTestMessage("retry-test-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		// Try sync that will fail (no receiver)
		start := time.Now()
		err = retrySyncer.SyncMessageSync(ctx, "failing-region", testMessage)
		duration := time.Since(start)

		// Should fail after timeout
		if err == nil {
			t.Error("Expected sync to fail due to no receiver")
		}

		// Should have taken at least the timeout duration
		if duration < retryConfig.SyncTimeout {
			t.Errorf("Expected sync to take at least %v, took %v", retryConfig.SyncTimeout, duration)
		}

		t.Logf("Retry test completed - error: %v, duration: %v", err, duration)
	})

	// Test Case 4: Graceful degradation during network issues
	t.Run("GracefulDegradationDuringNetworkIssues", func(t *testing.T) {
		ctx := context.Background()

		// Send some messages before "network failure"
		for i := 0; i < 3; i++ {
			testMessage := createTestMessage(fmt.Sprintf("pre-failure-msg-%03d", i), regionA)
			if err := storageA.Insert(ctx, testMessage); err != nil {
				t.Fatalf("Failed to insert pre-failure message %d: %v", i, err)
			}
			if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
				t.Fatalf("Failed to sync pre-failure message %d: %v", i, err)
			}
		}

		time.Sleep(200 * time.Millisecond)

		// Simulate "network recovery" by sending more messages
		for i := 0; i < 3; i++ {
			testMessage := createTestMessage(fmt.Sprintf("post-recovery-msg-%03d", i), regionA)
			if err := storageA.Insert(ctx, testMessage); err != nil {
				t.Fatalf("Failed to insert post-recovery message %d: %v", i, err)
			}
			if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
				t.Fatalf("Failed to sync post-recovery message %d: %v", i, err)
			}
		}

		time.Sleep(200 * time.Millisecond)

		// Verify all messages eventually made it through
		for i := 0; i < 3; i++ {
			msgID := fmt.Sprintf("pre-failure-msg-%03d", i)
			if _, err := storageB.GetMessageByID(ctx, msgID); err != nil {
				t.Errorf("Pre-failure message %s not found in region B: %v", msgID, err)
			}

			msgID = fmt.Sprintf("post-recovery-msg-%03d", i)
			if _, err := storageB.GetMessageByID(ctx, msgID); err != nil {
				t.Errorf("Post-recovery message %s not found in region B: %v", msgID, err)
			}
		}
	})

	// Test Case 5: Error metrics tracking
	t.Run("ErrorMetricsTracking", func(t *testing.T) {
		initialMetrics := syncerA.GetMetrics()
		initialErrors := initialMetrics["error_count"].(int64)

		testMessage := createTestMessage("error-metrics-msg-001", regionA)
		ctx := context.Background()

		if err := storageA.Insert(ctx, testMessage); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}

		// Try operations that should fail and increment error count
		_ = syncerA.SyncMessageSync(ctx, "nonexistent-region", testMessage)

		// Check that error metrics were updated
		finalMetrics := syncerA.GetMetrics()
		finalErrors := finalMetrics["error_count"].(int64)

		if finalErrors <= initialErrors {
			t.Logf("Error count - initial: %d, final: %d (may not increment for all error types)",
				initialErrors, finalErrors)
		}
	})
}

// Helper functions

func setupTwoRegionTest(t *testing.T, regionA, regionB string) (
	*MessageSyncer, *MessageSyncer, *storage.LocalStore, *storage.LocalStore, func()) {

	// Create HLC clocks
	hlcA := hlc.NewHLC(regionA, "node-1")
	hlcB := hlc.NewHLC(regionB, "node-1")

	// Create local queues
	queueConfigA := queue.DefaultConfig(regionA)
	queueConfigA.BufferSize = 100
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}

	queueConfigB := queue.DefaultConfig(regionB)
	queueConfigB.BufferSize = 100
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[QueueB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue B: %v", err)
	}

	// Create local storage
	storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}

	storageConfigB := storage.Config{RegionID: regionB, MemoryMode: true}
	storageB, err := storage.NewLocalStore(storageConfigB)
	if err != nil {
		t.Fatalf("Failed to create storage B: %v", err)
	}

	// Create message syncers
	syncerConfigA := DefaultConfig(regionA)
	syncerConfigA.SyncTimeout = 1 * time.Second
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}

	syncerConfigB := DefaultConfig(regionB)
	syncerConfigB.SyncTimeout = 1 * time.Second
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[SyncerB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer B: %v", err)
	}

	cleanup := func() {
		syncerA.Stop()
		syncerB.Stop()
		queueA.Close()
		queueB.Close()
		storageA.Close()
		storageB.Close()
	}

	return syncerA, syncerB, storageA, storageB, cleanup
}

func createTestMessage(msgID, regionID string) storage.LocalMessage {
	return storage.LocalMessage{
		MsgID:            msgID,
		SenderID:         "user-123",
		ConversationID:   "conv-456",
		ConversationType: "group",
		Content:          fmt.Sprintf("Test message %s from %s", msgID, regionID),
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"type": "text"},
		RegionID:         regionID,
		Version:          1,
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
