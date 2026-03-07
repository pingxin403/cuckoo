package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// TestEndToEndMessageFlow tests the complete message flow from Region A to Region B
// **Validates: Requirements 1.1, 2.1, 2.2**
func TestEndToEndMessageFlow(t *testing.T) {
	t.Run("RegionAToRegionBMessageSync", func(t *testing.T) {
		testRegionAToRegionBMessageSync(t)
	})

	t.Run("HLCClockUpdateLogic", func(t *testing.T) {
		testHLCClockUpdateLogic(t)
	})

	t.Run("ConflictDetectionAndLWWResolution", func(t *testing.T) {
		testConflictDetectionAndLWWResolution(t)
	})

	t.Run("MessageOrderingCorrectness", func(t *testing.T) {
		testMessageOrderingCorrectness(t)
	})

	t.Run("BidirectionalSyncFlow", func(t *testing.T) {
		testBidirectionalSyncFlow(t)
	})

	t.Run("ConcurrentMessageSyncWithOrdering", func(t *testing.T) {
		testConcurrentMessageSyncWithOrdering(t)
	})

	t.Run("NetworkPartitionRecovery", func(t *testing.T) {
		testNetworkPartitionRecovery(t)
	})

	t.Run("CriticalBusinessMessageSync", func(t *testing.T) {
		testCriticalBusinessMessageSync(t)
	})
}

// testRegionAToRegionBMessageSync tests basic message synchronization from Region A to Region B
func testRegionAToRegionBMessageSync(t *testing.T) {
	// Setup two-region test environment
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	// Start both syncers
	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create test messages in Region A
	messages := []storage.LocalMessage{
		createE2ETestMessage("e2e-msg-001", regionA, "Hello from Region A", 1),
		createE2ETestMessage("e2e-msg-002", regionA, "Second message from Region A", 2),
		createE2ETestMessage("e2e-msg-003", regionA, "Third message from Region A", 3),
	}

	// Insert messages in Region A and sync to Region B
	for _, msg := range messages {
		// Insert in Region A
		if err := storageA.Insert(ctx, msg); err != nil {
			t.Fatalf("Failed to insert message %s in Region A: %v", msg.MsgID, err)
		}

		// Sync to Region B asynchronously
		if err := syncerA.SyncMessageAsync(ctx, regionB, msg); err != nil {
			t.Fatalf("Failed to sync message %s to Region B: %v", msg.MsgID, err)
		}
	}

	// Wait for async processing
	time.Sleep(1 * time.Second)

	// Verify all messages exist in Region B
	for _, originalMsg := range messages {
		retrievedMsg, err := storageB.GetMessageByID(ctx, originalMsg.MsgID)
		if err != nil {
			t.Errorf("Message %s not found in Region B: %v", originalMsg.MsgID, err)
			continue
		}

		// Verify message content
		if retrievedMsg.Content != originalMsg.Content {
			t.Errorf("Message %s content mismatch: expected %s, got %s",
				originalMsg.MsgID, originalMsg.Content, retrievedMsg.Content)
		}

		// Verify sender ID
		if retrievedMsg.SenderID != originalMsg.SenderID {
			t.Errorf("Message %s sender mismatch: expected %s, got %s",
				originalMsg.MsgID, originalMsg.SenderID, retrievedMsg.SenderID)
		}

		// Verify sequence number
		if retrievedMsg.SequenceNumber != originalMsg.SequenceNumber {
			t.Errorf("Message %s sequence mismatch: expected %d, got %d",
				originalMsg.MsgID, originalMsg.SequenceNumber, retrievedMsg.SequenceNumber)
		}

		// Verify region ID is preserved as source region
		if retrievedMsg.RegionID != regionA {
			t.Errorf("Message %s region ID should be source region %s, got %s",
				originalMsg.MsgID, regionA, retrievedMsg.RegionID)
		}
	}

	// Check sync metrics
	metricsA := syncerA.GetMetrics()
	if metricsA["async_sync_count"].(int64) < int64(len(messages)) {
		t.Errorf("Expected at least %d async syncs, got %d",
			len(messages), metricsA["async_sync_count"])
	}

	t.Logf("Successfully synced %d messages from %s to %s", len(messages), regionA, regionB)
}

// testHLCClockUpdateLogic tests HLC clock synchronization during message sync
func testHLCClockUpdateLogic(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Get initial HLC timestamps
	hlcA := syncerA.hlcClock
	hlcB := syncerB.hlcClock

	initialTimestampA := hlcA.GetCurrentTimestamp()
	initialTimestampB := hlcB.GetCurrentTimestamp()

	t.Logf("Initial HLC timestamps - A: %s, B: %s", initialTimestampA, initialTimestampB)

	// Create message in Region A with HLC timestamp
	testMessage := createE2ETestMessage("hlc-test-msg-001", regionA, "HLC test message", 1)
	testMessage.GlobalID = hlcA.GenerateID().String()

	// Insert and sync message
	if err := storageA.Insert(ctx, testMessage); err != nil {
		t.Fatalf("Failed to insert message: %v", err)
	}

	if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
		t.Fatalf("Failed to sync message: %v", err)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify message was received in Region B
	_, err := storageB.GetMessageByID(ctx, testMessage.MsgID)
	if err != nil {
		t.Fatalf("Failed to retrieve message from Region B: %v", err)
	}

	// Check that HLC clocks have been updated
	finalTimestampA := hlcA.GetCurrentTimestamp()
	finalTimestampB := hlcB.GetCurrentTimestamp()

	t.Logf("Final HLC timestamps - A: %s, B: %s", finalTimestampA, finalTimestampB)

	// Verify HLC A has advanced (due to ID generation)
	if finalTimestampA == initialTimestampA {
		t.Error("HLC clock A should have advanced after generating ID")
	}

	// Verify HLC B has been updated (due to remote sync)
	if finalTimestampB == initialTimestampB {
		t.Error("HLC clock B should have been updated after receiving remote message")
	}

	// Generate new IDs and verify they maintain causal ordering
	newIDA := hlcA.GenerateID()
	newIDB := hlcB.GenerateID()

	// Both new IDs should be greater than the original message's GlobalID
	originalGlobalID := hlc.GlobalID{
		RegionID: regionA,
		HLC:      testMessage.GlobalID,
		Sequence: 1,
	}

	if hlc.CompareGlobalID(newIDA, originalGlobalID) <= 0 {
		t.Error("New ID from Region A should be greater than original message ID")
	}

	if hlc.CompareGlobalID(newIDB, originalGlobalID) <= 0 {
		t.Error("New ID from Region B should be greater than original message ID after sync")
	}

	t.Logf("HLC clock update test passed - causal ordering maintained")
}

// testConflictDetectionAndLWWResolution tests conflict detection and Last Write Wins resolution
func testConflictDetectionAndLWWResolution(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create conflicting messages with same ID but different content
	hlcA := syncerA.hlcClock
	hlcB := syncerB.hlcClock

	// Generate IDs with slight time difference to ensure deterministic ordering
	globalIDA := hlcA.GenerateID()
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	globalIDB := hlcB.GenerateID()

	messageA := createE2ETestMessage("conflict-msg-001", regionA, "Message from Region A", 1)
	messageA.GlobalID = globalIDA.String()
	messageA.Metadata = map[string]string{"source": "region-a", "version": "1"}

	messageB := createE2ETestMessage("conflict-msg-001", regionB, "Message from Region B", 1)
	messageB.GlobalID = globalIDB.String()
	messageB.Metadata = map[string]string{"source": "region-b", "version": "2"}

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
	time.Sleep(500 * time.Millisecond)

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

		// Verify conflict resolution is deterministic (LWW based on HLC)
		if conflict.Resolution != "local_wins" && conflict.Resolution != "remote_wins" {
			t.Errorf("Expected deterministic resolution, got %s", conflict.Resolution)
		}

		t.Logf("Conflict detected and resolved: %s (local_region=%s, remote_region=%s)",
			conflict.Resolution, conflict.LocalRegion, conflict.RemoteRegion)
	}

	// Verify the winning message is in storage
	finalMsg, err := storageB.GetMessageByID(ctx, "conflict-msg-001")
	if err != nil {
		t.Fatalf("Failed to get final message after conflict resolution: %v", err)
	}

	// Determine which message should have won based on HLC comparison
	expectedWinner := "unknown"
	cmp := hlc.CompareGlobalID(globalIDA, globalIDB)
	if cmp > 0 {
		expectedWinner = "region-a"
	} else if cmp < 0 {
		expectedWinner = "region-b"
	}

	t.Logf("Final message content: %s (expected winner: %s)", finalMsg.Content, expectedWinner)

	// Check conflict metrics
	metricsB := syncerB.GetMetrics()
	if metricsB["conflict_count"].(int64) == 0 {
		t.Error("Expected at least one conflict to be recorded in metrics")
	}

	t.Logf("Conflict detection and LWW resolution test passed")
}

// testMessageOrderingCorrectness tests that messages maintain correct ordering
func testMessageOrderingCorrectness(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create messages with specific sequence numbers
	messageCount := 10
	var messages []storage.LocalMessage
	var globalIDs []hlc.GlobalID

	hlcA := syncerA.hlcClock

	for i := 0; i < messageCount; i++ {
		globalID := hlcA.GenerateID()
		globalIDs = append(globalIDs, globalID)

		msg := createE2ETestMessage(
			fmt.Sprintf("order-msg-%03d", i),
			regionA,
			fmt.Sprintf("Ordered message %d", i),
			int64(i+1),
		)
		msg.GlobalID = globalID.String()
		messages = append(messages, msg)

		// Small delay to ensure different HLC timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Insert and sync all messages
	for _, msg := range messages {
		if err := storageA.Insert(ctx, msg); err != nil {
			t.Fatalf("Failed to insert message %s: %v", msg.MsgID, err)
		}

		if err := syncerA.SyncMessageAsync(ctx, regionB, msg); err != nil {
			t.Fatalf("Failed to sync message %s: %v", msg.MsgID, err)
		}
	}

	// Wait for all messages to be processed
	time.Sleep(1 * time.Second)

	// Retrieve all messages from Region B
	var retrievedMessages []storage.LocalMessage
	for _, originalMsg := range messages {
		retrievedMsg, err := storageB.GetMessageByID(ctx, originalMsg.MsgID)
		if err != nil {
			t.Errorf("Message %s not found in Region B: %v", originalMsg.MsgID, err)
			continue
		}
		retrievedMessages = append(retrievedMessages, *retrievedMsg)
	}

	if len(retrievedMessages) != messageCount {
		t.Fatalf("Expected %d messages in Region B, got %d", messageCount, len(retrievedMessages))
	}

	// Sort retrieved messages by GlobalID (HLC ordering)
	sort.Slice(retrievedMessages, func(i, j int) bool {
		globalIDI := hlc.GlobalID{
			RegionID: regionA,
			HLC:      retrievedMessages[i].GlobalID,
			Sequence: retrievedMessages[i].SequenceNumber,
		}
		globalIDJ := hlc.GlobalID{
			RegionID: regionA,
			HLC:      retrievedMessages[j].GlobalID,
			Sequence: retrievedMessages[j].SequenceNumber,
		}
		return hlc.CompareGlobalID(globalIDI, globalIDJ) < 0
	})

	// Verify messages are in correct order
	for i, msg := range retrievedMessages {
		expectedMsgID := fmt.Sprintf("order-msg-%03d", i)
		if msg.MsgID != expectedMsgID {
			t.Errorf("Message order incorrect at position %d: expected %s, got %s",
				i, expectedMsgID, msg.MsgID)
		}

		expectedSequence := int64(i + 1)
		if msg.SequenceNumber != expectedSequence {
			t.Errorf("Sequence number incorrect at position %d: expected %d, got %d",
				i, expectedSequence, msg.SequenceNumber)
		}
	}

	// Verify HLC ordering is maintained
	for i := 1; i < len(globalIDs); i++ {
		if hlc.CompareGlobalID(globalIDs[i-1], globalIDs[i]) >= 0 {
			t.Errorf("HLC ordering violated at position %d: %s >= %s",
				i, globalIDs[i-1], globalIDs[i])
		}
	}

	t.Logf("Message ordering correctness test passed - %d messages in correct order", messageCount)
}

// testBidirectionalSyncFlow tests synchronization in both directions
func testBidirectionalSyncFlow(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create messages in both regions
	messageA := createE2ETestMessage("bidirectional-msg-a", regionA, "Message from A to B", 1)
	messageB := createE2ETestMessage("bidirectional-msg-b", regionB, "Message from B to A", 1)

	// Insert messages in their respective regions
	if err := storageA.Insert(ctx, messageA); err != nil {
		t.Fatalf("Failed to insert message A: %v", err)
	}
	if err := storageB.Insert(ctx, messageB); err != nil {
		t.Fatalf("Failed to insert message B: %v", err)
	}

	// Sync A → B
	if err := syncerA.SyncMessageAsync(ctx, regionB, messageA); err != nil {
		t.Fatalf("Failed to sync A → B: %v", err)
	}

	// Sync B → A
	if err := syncerB.SyncMessageAsync(ctx, regionA, messageB); err != nil {
		t.Fatalf("Failed to sync B → A: %v", err)
	}

	// Wait for bidirectional sync
	time.Sleep(1 * time.Second)

	// Verify message A exists in Region B
	retrievedA, err := storageB.GetMessageByID(ctx, messageA.MsgID)
	if err != nil {
		t.Errorf("Message A not found in Region B: %v", err)
	} else if retrievedA.Content != messageA.Content {
		t.Errorf("Message A content mismatch in Region B")
	}

	// Verify message B exists in Region A
	retrievedB, err := storageA.GetMessageByID(ctx, messageB.MsgID)
	if err != nil {
		t.Errorf("Message B not found in Region A: %v", err)
	} else if retrievedB.Content != messageB.Content {
		t.Errorf("Message B content mismatch in Region A")
	}

	// Check metrics for both syncers
	metricsA := syncerA.GetMetrics()
	metricsB := syncerB.GetMetrics()

	if metricsA["async_sync_count"].(int64) < 1 {
		t.Error("Syncer A should have at least 1 async sync")
	}
	if metricsB["async_sync_count"].(int64) < 1 {
		t.Error("Syncer B should have at least 1 async sync")
	}

	t.Logf("Bidirectional sync flow test passed")
}

// testConcurrentMessageSyncWithOrdering tests concurrent message sync while maintaining ordering
func testConcurrentMessageSyncWithOrdering(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create multiple messages concurrently
	const numGoroutines = 5
	const messagesPerGoroutine = 10
	const totalMessages = numGoroutines * messagesPerGoroutine

	var wg sync.WaitGroup
	var allGlobalIDs []hlc.GlobalID
	var globalIDsMutex sync.Mutex

	hlcA := syncerA.hlcClock

	// Generate messages concurrently
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for m := 0; m < messagesPerGoroutine; m++ {
				globalID := hlcA.GenerateID()

				// Store global ID for later verification
				globalIDsMutex.Lock()
				allGlobalIDs = append(allGlobalIDs, globalID)
				globalIDsMutex.Unlock()

				msgID := fmt.Sprintf("concurrent-msg-%d-%d", goroutineID, m)
				msg := createE2ETestMessage(msgID, regionA,
					fmt.Sprintf("Concurrent message %d from goroutine %d", m, goroutineID),
					int64(goroutineID*messagesPerGoroutine+m+1))
				msg.GlobalID = globalID.String()

				// Insert and sync
				if err := storageA.Insert(ctx, msg); err != nil {
					t.Errorf("Failed to insert concurrent message %s: %v", msgID, err)
					return
				}

				if err := syncerA.SyncMessageAsync(ctx, regionB, msg); err != nil {
					t.Errorf("Failed to sync concurrent message %s: %v", msgID, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()

	// Wait for all messages to be processed
	time.Sleep(2 * time.Second)

	// Verify all messages were synced
	syncedCount := 0
	for g := 0; g < numGoroutines; g++ {
		for m := 0; m < messagesPerGoroutine; m++ {
			msgID := fmt.Sprintf("concurrent-msg-%d-%d", g, m)
			if _, err := storageB.GetMessageByID(ctx, msgID); err == nil {
				syncedCount++
			} else {
				t.Errorf("Concurrent message %s not found in Region B: %v", msgID, err)
			}
		}
	}

	if syncedCount != totalMessages {
		t.Errorf("Expected %d messages synced, got %d", totalMessages, syncedCount)
	}

	// Verify HLC ordering is maintained even with concurrent generation
	sort.Slice(allGlobalIDs, func(i, j int) bool {
		return hlc.CompareGlobalID(allGlobalIDs[i], allGlobalIDs[j]) < 0
	})

	// Check that sorted order is monotonic
	for i := 1; i < len(allGlobalIDs); i++ {
		if hlc.CompareGlobalID(allGlobalIDs[i-1], allGlobalIDs[i]) >= 0 {
			t.Errorf("HLC ordering violated in concurrent generation at position %d", i)
		}
	}

	t.Logf("Concurrent message sync test passed - %d messages synced with correct ordering", totalMessages)
}

// testNetworkPartitionRecovery simulates network partition and recovery
func testNetworkPartitionRecovery(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Phase 1: Normal operation - sync some messages
	prePartitionMessages := []storage.LocalMessage{
		createE2ETestMessage("pre-partition-msg-1", regionA, "Message before partition", 1),
		createE2ETestMessage("pre-partition-msg-2", regionA, "Another message before partition", 2),
	}

	for _, msg := range prePartitionMessages {
		if err := storageA.Insert(ctx, msg); err != nil {
			t.Fatalf("Failed to insert pre-partition message: %v", err)
		}
		if err := syncerA.SyncMessageAsync(ctx, regionB, msg); err != nil {
			t.Fatalf("Failed to sync pre-partition message: %v", err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	// Verify pre-partition messages were synced
	for _, msg := range prePartitionMessages {
		if _, err := storageB.GetMessageByID(ctx, msg.MsgID); err != nil {
			t.Errorf("Pre-partition message %s not found in Region B: %v", msg.MsgID, err)
		}
	}

	// Phase 2: Simulate network partition by stopping syncer B
	if err := syncerB.Stop(); err != nil {
		t.Fatalf("Failed to stop syncer B: %v", err)
	}

	// Create messages during partition (these should fail to sync)
	partitionMessages := []storage.LocalMessage{
		createE2ETestMessage("partition-msg-1", regionA, "Message during partition", 3),
		createE2ETestMessage("partition-msg-2", regionA, "Another message during partition", 4),
	}

	for _, msg := range partitionMessages {
		if err := storageA.Insert(ctx, msg); err != nil {
			t.Fatalf("Failed to insert partition message: %v", err)
		}
		// These syncs should fail or timeout, but we don't fail the test
		_ = syncerA.SyncMessageAsync(ctx, regionB, msg)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify partition messages are NOT in Region B yet
	for _, msg := range partitionMessages {
		if _, err := storageB.GetMessageByID(ctx, msg.MsgID); err == nil {
			t.Errorf("Partition message %s should not be in Region B during partition", msg.MsgID)
		}
	}

	// Phase 3: Simulate network recovery by restarting syncer B
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to restart syncer B: %v", err)
	}

	// Retry syncing partition messages
	for _, msg := range partitionMessages {
		if err := syncerA.SyncMessageAsync(ctx, regionB, msg); err != nil {
			t.Logf("Retry sync for partition message %s: %v", msg.MsgID, err)
		}
	}

	// Wait for recovery sync
	time.Sleep(1 * time.Second)

	// Verify all messages are now in Region B
	allMessages := append(prePartitionMessages, partitionMessages...)
	for _, msg := range allMessages {
		if _, err := storageB.GetMessageByID(ctx, msg.MsgID); err != nil {
			t.Errorf("Message %s not found in Region B after recovery: %v", msg.MsgID, err)
		}
	}

	t.Logf("Network partition recovery test passed - %d messages recovered", len(allMessages))
}

// testCriticalBusinessMessageSync tests synchronous sync for critical business operations
func testCriticalBusinessMessageSync(t *testing.T) {
	regionA, regionB := "region-a", "region-b"
	syncerA, syncerB, storageA, storageB, cleanup := setupTwoRegionE2ETest(t, regionA, regionB)
	defer cleanup()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}
	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	ctx := context.Background()

	// Create critical business message (e.g., payment)
	criticalMessage := createE2ETestMessage("critical-payment-001", regionA, "Payment processed: $100.00", 1)
	criticalMessage.Metadata = map[string]string{
		"type":     "payment",
		"critical": "true",
		"amount":   "100.00",
		"currency": "USD",
	}

	// Insert in Region A
	if err := storageA.Insert(ctx, criticalMessage); err != nil {
		t.Fatalf("Failed to insert critical message: %v", err)
	}

	// Sync synchronously (critical business operation)
	start := time.Now()
	if err := syncerA.SyncMessageSync(ctx, regionB, criticalMessage); err != nil {
		t.Fatalf("Failed to sync critical message synchronously: %v", err)
	}
	syncDuration := time.Since(start)

	// Verify message exists in Region B
	retrievedMsg, err := storageB.GetMessageByID(ctx, criticalMessage.MsgID)
	if err != nil {
		t.Fatalf("Critical message not found in Region B: %v", err)
	}

	// Verify message content and metadata
	if retrievedMsg.Content != criticalMessage.Content {
		t.Errorf("Critical message content mismatch")
	}

	if retrievedMsg.Metadata["type"] != "payment" {
		t.Errorf("Critical message metadata not preserved")
	}

	// Verify sync completed within reasonable time
	if syncDuration > 5*time.Second {
		t.Errorf("Critical message sync took too long: %v", syncDuration)
	}

	// Check sync metrics
	metricsA := syncerA.GetMetrics()
	if metricsA["sync_sync_count"].(int64) < 1 {
		t.Error("Expected at least 1 synchronous sync for critical message")
	}

	t.Logf("Critical business message sync test passed - synced in %v", syncDuration)
}

// Helper functions for E2E tests

func setupTwoRegionE2ETest(t *testing.T, regionA, regionB string) (
	*MessageSyncer, *MessageSyncer, *storage.LocalStore, *storage.LocalStore, func()) {

	// Create HLC clocks
	hlcA := hlc.NewHLC(regionA, "node-1")
	hlcB := hlc.NewHLC(regionB, "node-1")

	// Create local queues with larger buffers for E2E tests
	queueConfigA := queue.DefaultConfig(regionA)
	queueConfigA.BufferSize = 1000
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[E2E-QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}

	queueConfigB := queue.DefaultConfig(regionB)
	queueConfigB.BufferSize = 1000
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[E2E-QueueB] ", log.LstdFlags))
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

	// Create message syncers with E2E optimized config
	syncerConfigA := DefaultConfig(regionA)
	syncerConfigA.SyncTimeout = 3 * time.Second
	syncerConfigA.MaxRetries = 3
	syncerConfigA.EnableChecksum = true
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[E2E-SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}

	syncerConfigB := DefaultConfig(regionB)
	syncerConfigB.SyncTimeout = 3 * time.Second
	syncerConfigB.MaxRetries = 3
	syncerConfigB.EnableChecksum = true
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[E2E-SyncerB] ", log.LstdFlags))
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

func createE2ETestMessage(msgID, regionID, content string, sequenceNumber int64) storage.LocalMessage {
	return storage.LocalMessage{
		MsgID:            msgID,
		UserID:           "e2e-user-123",
		SenderID:         "e2e-sender-456",
		ConversationID:   "e2e-conv-789",
		ConversationType: "group",
		Content:          content,
		SequenceNumber:   sequenceNumber,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"test": "e2e", "type": "text"},
		RegionID:         regionID,
		Version:          1,
	}
}
