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

func TestMessageSyncer_AsyncSync(t *testing.T) {
	// Setup test environment
	regionA := "region-a"
	regionB := "region-b"

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
	defer queueA.Close()

	queueConfigB := queue.DefaultConfig(regionB)
	queueConfigB.BufferSize = 100
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[QueueB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue B: %v", err)
	}
	defer queueB.Close()

	// Create local storage
	storageConfigA := storage.Config{
		RegionID:   regionA,
		MemoryMode: true,
	}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}
	defer storageA.Close()

	storageConfigB := storage.Config{
		RegionID:   regionB,
		MemoryMode: true,
	}
	storageB, err := storage.NewLocalStore(storageConfigB)
	if err != nil {
		t.Fatalf("Failed to create storage B: %v", err)
	}
	defer storageB.Close()

	// Create message syncers
	syncerConfigA := DefaultConfig(regionA)
	syncerConfigA.SyncTimeout = 2 * time.Second
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	syncerConfigB := DefaultConfig(regionB)
	syncerConfigB.SyncTimeout = 2 * time.Second
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[SyncerB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer B: %v", err)
	}
	defer syncerB.Stop()

	// Start syncers
	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}

	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Create test message
	testMessage := storage.LocalMessage{
		MsgID:            "test-msg-001",
		SenderID:         "user-123",
		ConversationID:   "conv-456",
		ConversationType: "group",
		Content:          "Hello from region A!",
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"type": "text"},
		RegionID:         regionA,
		Version:          1,
	}

	// Insert message in region A
	ctx := context.Background()
	if err := storageA.Insert(ctx, testMessage); err != nil {
		t.Fatalf("Failed to insert test message in region A: %v", err)
	}

	// Sync message to region B asynchronously
	if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
		t.Fatalf("Failed to sync message async: %v", err)
	}

	// Wait for message to be processed
	time.Sleep(500 * time.Millisecond)

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
	if metricsA["async_sync_count"].(int64) != 1 {
		t.Errorf("Expected 1 async sync, got %d", metricsA["async_sync_count"])
	}

	t.Logf("Async sync test passed - message successfully synced from %s to %s", regionA, regionB)
}

func TestMessageSyncer_SyncSync(t *testing.T) {
	// Setup test environment (similar to async test)
	regionA := "region-a"
	regionB := "region-b"

	hlcA := hlc.NewHLC(regionA, "node-1")
	hlcB := hlc.NewHLC(regionB, "node-1")

	queueConfigA := queue.DefaultConfig(regionA)
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}
	defer queueA.Close()

	queueConfigB := queue.DefaultConfig(regionB)
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[QueueB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue B: %v", err)
	}
	defer queueB.Close()

	storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}
	defer storageA.Close()

	storageConfigB := storage.Config{RegionID: regionB, MemoryMode: true}
	storageB, err := storage.NewLocalStore(storageConfigB)
	if err != nil {
		t.Fatalf("Failed to create storage B: %v", err)
	}
	defer storageB.Close()

	syncerConfigA := DefaultConfig(regionA)
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	syncerConfigB := DefaultConfig(regionB)
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[SyncerB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer B: %v", err)
	}
	defer syncerB.Stop()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}

	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Create critical business message
	criticalMessage := storage.LocalMessage{
		MsgID:            "critical-msg-001",
		SenderID:         "system",
		ConversationID:   "payment-conv-789",
		ConversationType: "private",
		Content:          "Payment processed: $100.00",
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"type": "payment", "critical": "true"},
		RegionID:         regionA,
		Version:          1,
	}

	// Insert message in region A
	ctx := context.Background()
	if err := storageA.Insert(ctx, criticalMessage); err != nil {
		t.Fatalf("Failed to insert critical message in region A: %v", err)
	}

	// Sync message to region B synchronously
	start := time.Now()
	if err := syncerA.SyncMessageSync(ctx, regionB, criticalMessage); err != nil {
		t.Fatalf("Failed to sync message sync: %v", err)
	}
	syncDuration := time.Since(start)

	// Verify message exists in region B
	retrievedMsg, err := storageB.GetMessageByID(ctx, criticalMessage.MsgID)
	if err != nil {
		t.Fatalf("Failed to retrieve critical message from region B: %v", err)
	}

	// Verify message content
	if retrievedMsg.Content != criticalMessage.Content {
		t.Errorf("Expected content %s, got %s", criticalMessage.Content, retrievedMsg.Content)
	}

	// Check that sync completed within reasonable time
	if syncDuration > 5*time.Second {
		t.Errorf("Sync took too long: %v", syncDuration)
	}

	// Check metrics
	metricsA := syncerA.GetMetrics()
	if metricsA["sync_sync_count"].(int64) != 1 {
		t.Errorf("Expected 1 sync sync, got %d", metricsA["sync_sync_count"])
	}

	t.Logf("Sync sync test passed - critical message synced in %v", syncDuration)
}

func TestMessageSyncer_ConflictResolution(t *testing.T) {
	// Setup test environment
	regionA := "region-a"
	regionB := "region-b"

	hlcA := hlc.NewHLC(regionA, "node-1")
	hlcB := hlc.NewHLC(regionB, "node-1")

	// Create slightly different timestamps to simulate conflict
	time.Sleep(10 * time.Millisecond)

	queueConfigA := queue.DefaultConfig(regionA)
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}
	defer queueA.Close()

	queueConfigB := queue.DefaultConfig(regionB)
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[QueueB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue B: %v", err)
	}
	defer queueB.Close()

	storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}
	defer storageA.Close()

	storageConfigB := storage.Config{RegionID: regionB, MemoryMode: true}
	storageB, err := storage.NewLocalStore(storageConfigB)
	if err != nil {
		t.Fatalf("Failed to create storage B: %v", err)
	}
	defer storageB.Close()

	syncerConfigA := DefaultConfig(regionA)
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	syncerConfigB := DefaultConfig(regionB)
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[SyncerB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer B: %v", err)
	}
	defer syncerB.Stop()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}

	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Create conflicting messages with same ID but different content
	messageA := storage.LocalMessage{
		MsgID:            "conflict-msg-001",
		SenderID:         "user-123",
		ConversationID:   "conv-456",
		ConversationType: "group",
		Content:          "Message from region A",
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"source": "region-a"},
		RegionID:         regionA,
		GlobalID:         hlcA.GenerateID().String(),
		Version:          1,
	}

	messageB := storage.LocalMessage{
		MsgID:            "conflict-msg-001", // Same ID
		SenderID:         "user-123",
		ConversationID:   "conv-456",
		ConversationType: "group",
		Content:          "Message from region B", // Different content
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"source": "region-b"},
		RegionID:         regionB,
		GlobalID:         hlcB.GenerateID().String(),
		Version:          1,
	}

	// Insert messages in their respective regions
	ctx := context.Background()
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

	// Wait for processing
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

		t.Logf("Conflict detected and resolved: %s", conflict.Resolution)
	}

	// Check metrics
	metricsA := syncerA.GetMetrics()
	metricsB := syncerB.GetMetrics()

	totalConflicts := metricsA["conflict_count"].(int64) + metricsB["conflict_count"].(int64)
	if totalConflicts == 0 {
		t.Error("Expected at least one conflict to be recorded in metrics")
	}

	t.Logf("Conflict resolution test passed - %d conflicts detected and resolved", totalConflicts)
}

func TestMessageSyncer_HLCIntegration(t *testing.T) {
	// Test that HLC clocks are properly updated during sync
	regionA := "region-a"
	regionB := "region-b"

	hlcA := hlc.NewHLC(regionA, "node-1")
	hlcB := hlc.NewHLC(regionB, "node-1")

	// Record initial timestamps
	initialA := hlcA.GetCurrentTimestamp()
	initialB := hlcB.GetCurrentTimestamp()

	queueConfigA := queue.DefaultConfig(regionA)
	queueA, err := queue.NewLocalQueue(queueConfigA, log.New(os.Stdout, "[QueueA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue A: %v", err)
	}
	defer queueA.Close()

	queueConfigB := queue.DefaultConfig(regionB)
	queueB, err := queue.NewLocalQueue(queueConfigB, log.New(os.Stdout, "[QueueB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create queue B: %v", err)
	}
	defer queueB.Close()

	storageConfigA := storage.Config{RegionID: regionA, MemoryMode: true}
	storageA, err := storage.NewLocalStore(storageConfigA)
	if err != nil {
		t.Fatalf("Failed to create storage A: %v", err)
	}
	defer storageA.Close()

	storageConfigB := storage.Config{RegionID: regionB, MemoryMode: true}
	storageB, err := storage.NewLocalStore(storageConfigB)
	if err != nil {
		t.Fatalf("Failed to create storage B: %v", err)
	}
	defer storageB.Close()

	syncerConfigA := DefaultConfig(regionA)
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	syncerConfigB := DefaultConfig(regionB)
	syncerB, err := NewMessageSyncer(regionB, hlcB, queueB, storageB, syncerConfigB,
		log.New(os.Stdout, "[SyncerB] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer B: %v", err)
	}
	defer syncerB.Stop()

	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer A: %v", err)
	}

	if err := syncerB.Start(); err != nil {
		t.Fatalf("Failed to start syncer B: %v", err)
	}

	// Create message with HLC timestamp
	testMessage := storage.LocalMessage{
		MsgID:            "hlc-test-msg-001",
		SenderID:         "user-123",
		ConversationID:   "conv-456",
		ConversationType: "group",
		Content:          "HLC test message",
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		RegionID:         regionA,
		GlobalID:         hlcA.GenerateID().String(),
		Version:          1,
	}

	// Sync message from A to B
	ctx := context.Background()
	if err := storageA.Insert(ctx, testMessage); err != nil {
		t.Fatalf("Failed to insert test message: %v", err)
	}

	if err := syncerA.SyncMessageAsync(ctx, regionB, testMessage); err != nil {
		t.Fatalf("Failed to sync message: %v", err)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check that HLC clocks have been updated
	finalA := hlcA.GetCurrentTimestamp()
	finalB := hlcB.GetCurrentTimestamp()

	if finalA == initialA {
		t.Error("HLC clock A should have been updated")
	}

	if finalB == initialB {
		t.Error("HLC clock B should have been updated")
	}

	t.Logf("HLC integration test passed - clocks updated from %s->%s (A) and %s->%s (B)",
		initialA, finalA, initialB, finalB)
}

func TestMessageSyncer_Checksum(t *testing.T) {
	// Test checksum calculation and verification
	regionA := "region-a"

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
	syncerConfigA.EnableChecksum = true
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	// Create test sync message
	syncMsg := &SyncMessage{
		MessageID:      "checksum-test-001",
		ConversationID: "conv-123",
		SenderID:       "user-456",
		Content:        "Test message for checksum",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	// Calculate checksum
	checksum1 := syncerA.calculateChecksum(syncMsg)
	checksum2 := syncerA.calculateChecksum(syncMsg)

	// Checksums should be identical for same message
	if checksum1 != checksum2 {
		t.Errorf("Checksums should be identical: %s != %s", checksum1, checksum2)
	}

	// Modify message and verify checksum changes
	syncMsg.Content = "Modified content"
	checksum3 := syncerA.calculateChecksum(syncMsg)

	if checksum1 == checksum3 {
		t.Error("Checksum should change when message content changes")
	}

	// Verify checksum is not empty when enabled
	if checksum1 == "" {
		t.Error("Checksum should not be empty when enabled")
	}

	t.Logf("Checksum test passed - checksums: %s, %s, %s", checksum1, checksum2, checksum3)
}

func TestMessageSyncer_Metrics(t *testing.T) {
	// Test metrics collection
	regionA := "region-a"

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
	syncerA, err := NewMessageSyncer(regionA, hlcA, queueA, storageA, syncerConfigA,
		log.New(os.Stdout, "[SyncerA] ", log.LstdFlags))
	if err != nil {
		t.Fatalf("Failed to create syncer A: %v", err)
	}
	defer syncerA.Stop()

	// Get initial metrics
	initialMetrics := syncerA.GetMetrics()

	// Verify initial state
	if initialMetrics["region_id"] != regionA {
		t.Errorf("Expected region_id %s, got %s", regionA, initialMetrics["region_id"])
	}

	if initialMetrics["async_sync_count"].(int64) != 0 {
		t.Errorf("Expected initial async_sync_count 0, got %d", initialMetrics["async_sync_count"])
	}

	if initialMetrics["started"].(bool) != false {
		t.Error("Expected syncer to not be started initially")
	}

	// Start syncer and check metrics
	if err := syncerA.Start(); err != nil {
		t.Fatalf("Failed to start syncer: %v", err)
	}

	updatedMetrics := syncerA.GetMetrics()
	if updatedMetrics["started"].(bool) != true {
		t.Error("Expected syncer to be started after Start()")
	}

	t.Logf("Metrics test passed - initial: %+v, updated: %+v", initialMetrics, updatedMetrics)
}
