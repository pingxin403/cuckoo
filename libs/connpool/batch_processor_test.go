package connpool

import (
	"context"
	"testing"
	"time"
)

func TestNewBatchProcessor(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	bp := NewBatchProcessor(config)

	if bp == nil {
		t.Fatal("Expected batch processor to be created")
	}

	if bp.config.MessageBatchSize != config.MessageBatchSize {
		t.Errorf("Expected message batch size %d, got %d", config.MessageBatchSize, bp.config.MessageBatchSize)
	}
}

func TestBatchProcessor_AddMessage(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 5
	bp := NewBatchProcessor(config)

	// Add messages
	for i := 0; i < 3; i++ {
		msg := BatchMessage{
			ID:             string(rune('A' + i)),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			ConversationID: "conv-1",
			SenderID:       "user-1",
			Content:        "test message",
			Timestamp:      time.Now().UnixMilli(),
			Priority:       1,
			CreatedAt:      time.Now(),
		}

		if err := bp.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	bp.messageBatchMu.Lock()
	batchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	if batchSize != 3 {
		t.Errorf("Expected batch size 3, got %d", batchSize)
	}

	totalBatched := bp.totalMessagesBatched
	if totalBatched != 3 {
		t.Errorf("Expected total batched 3, got %d", totalBatched)
	}
}

func TestBatchProcessor_AddOfflineMessage(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.OfflineBatchSize = 5
	bp := NewBatchProcessor(config)

	// Add offline messages
	for i := 0; i < 3; i++ {
		msg := BatchOfflineMessage{
			ID:             string(rune('A' + i)),
			UserID:         "user-1",
			SenderID:       "user-2",
			ConversationID: "conv-1",
			Content:        "offline message",
			SequenceNumber: int64(i + 1),
			Timestamp:      time.Now().UnixMilli(),
			ExpiresAt:      time.Now().Add(24 * time.Hour),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			CreatedAt:      time.Now(),
		}

		if err := bp.AddOfflineMessage(msg); err != nil {
			t.Fatalf("Failed to add offline message: %v", err)
		}
	}

	bp.offlineBatchMu.Lock()
	batchSize := len(bp.offlineBatch)
	bp.offlineBatchMu.Unlock()

	if batchSize != 3 {
		t.Errorf("Expected batch size 3, got %d", batchSize)
	}

	totalBatched := bp.totalOfflineBatched
	if totalBatched != 3 {
		t.Errorf("Expected total batched 3, got %d", totalBatched)
	}
}

func TestBatchProcessor_AddReconcileItem(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.ReconcileBatchSize = 5
	bp := NewBatchProcessor(config)

	// Add reconcile items
	for i := 0; i < 3; i++ {
		item := BatchReconcileItem{
			GlobalID:     "global-" + string(rune('A'+i)),
			Operation:    "add",
			SourceRegion: "region-a",
			TargetRegion: "region-b",
			Priority:     1,
			CreatedAt:    time.Now(),
		}

		if err := bp.AddReconcileItem(item); err != nil {
			t.Fatalf("Failed to add reconcile item: %v", err)
		}
	}

	bp.reconcileBatchMu.Lock()
	batchSize := len(bp.reconcileBatch)
	bp.reconcileBatchMu.Unlock()

	if batchSize != 3 {
		t.Errorf("Expected batch size 3, got %d", batchSize)
	}

	totalBatched := bp.totalReconcileBatched
	if totalBatched != 3 {
		t.Errorf("Expected total batched 3, got %d", totalBatched)
	}
}

func TestBatchProcessor_AutoFlushOnSize(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 3
	config.MessageFlushInterval = 10 * time.Second // Long interval to test size-based flush
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer bp.Stop()

	// Add messages to trigger auto-flush
	for i := 0; i < 3; i++ {
		msg := BatchMessage{
			ID:             string(rune('A' + i)),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			ConversationID: "conv-1",
			SenderID:       "user-1",
			Content:        "test message",
			Timestamp:      time.Now().UnixMilli(),
			Priority:       1,
			CreatedAt:      time.Now(),
		}

		if err := bp.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	// Wait for auto-flush
	time.Sleep(200 * time.Millisecond)

	// Check that batch was flushed
	bp.messageBatchMu.Lock()
	batchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected batch to be flushed, but has %d items", batchSize)
	}
}

func TestBatchProcessor_PeriodicFlush(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 100 // Large size to prevent size-based flush
	config.MessageFlushInterval = 100 * time.Millisecond
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer bp.Stop()

	// Add a few messages
	for i := 0; i < 3; i++ {
		msg := BatchMessage{
			ID:             string(rune('A' + i)),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			ConversationID: "conv-1",
			SenderID:       "user-1",
			Content:        "test message",
			Timestamp:      time.Now().UnixMilli(),
			Priority:       1,
			CreatedAt:      time.Now(),
		}

		if err := bp.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	// Wait for periodic flush
	time.Sleep(200 * time.Millisecond)

	// Check that batch was flushed
	bp.messageBatchMu.Lock()
	batchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected batch to be flushed, but has %d items", batchSize)
	}
}

func TestBatchProcessor_PriorityOrdering(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	bp := NewBatchProcessor(config)

	// Create messages with different priorities
	messages := []BatchMessage{
		{ID: "A", Priority: 1},
		{ID: "B", Priority: 5},
		{ID: "C", Priority: 3},
		{ID: "D", Priority: 10},
		{ID: "E", Priority: 2},
	}

	// Sort by priority
	bp.sortMessagesByPriority(messages)

	// Check ordering (highest priority first)
	expectedOrder := []string{"D", "B", "C", "E", "A"}
	for i, msg := range messages {
		if msg.ID != expectedOrder[i] {
			t.Errorf("Expected message %s at position %d, got %s", expectedOrder[i], i, msg.ID)
		}
	}
}

func TestBatchProcessor_GetMetrics(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 2
	config.MessageFlushInterval = 50 * time.Millisecond
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer bp.Stop()

	// Add messages
	for i := 0; i < 5; i++ {
		msg := BatchMessage{
			ID:             string(rune('A' + i)),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			ConversationID: "conv-1",
			SenderID:       "user-1",
			Content:        "test message",
			Timestamp:      time.Now().UnixMilli(),
			Priority:       1,
			CreatedAt:      time.Now(),
		}

		if err := bp.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	// Wait for flushes
	time.Sleep(200 * time.Millisecond)

	// Get metrics
	metrics := bp.GetMetrics()

	if metrics.TotalMessagesBatched != 5 {
		t.Errorf("Expected 5 messages batched, got %d", metrics.TotalMessagesBatched)
	}

	if metrics.TotalBatchesFlushed == 0 {
		t.Error("Expected at least one batch to be flushed")
	}

	if metrics.AvgBatchSize == 0 {
		t.Error("Expected average batch size to be calculated")
	}
}

func TestBatchProcessor_FlushAll(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 100 // Large size to prevent auto-flush
	config.MessageFlushInterval = 10 * time.Second
	bp := NewBatchProcessor(config)

	// Add messages
	for i := 0; i < 5; i++ {
		msg := BatchMessage{
			ID:             string(rune('A' + i)),
			RegionID:       "region-a",
			GlobalID:       "global-" + string(rune('A'+i)),
			ConversationID: "conv-1",
			SenderID:       "user-1",
			Content:        "test message",
			Timestamp:      time.Now().UnixMilli(),
			Priority:       1,
			CreatedAt:      time.Now(),
		}

		if err := bp.AddMessage(msg); err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}
	}

	// Manually flush all
	ctx := context.Background()
	if err := bp.FlushAll(ctx); err != nil {
		t.Fatalf("Failed to flush all: %v", err)
	}

	// Check that all batches are empty
	bp.messageBatchMu.Lock()
	messageBatchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	bp.offlineBatchMu.Lock()
	offlineBatchSize := len(bp.offlineBatch)
	bp.offlineBatchMu.Unlock()

	bp.reconcileBatchMu.Lock()
	reconcileBatchSize := len(bp.reconcileBatch)
	bp.reconcileBatchMu.Unlock()

	if messageBatchSize != 0 {
		t.Errorf("Expected message batch to be empty, got %d items", messageBatchSize)
	}
	if offlineBatchSize != 0 {
		t.Errorf("Expected offline batch to be empty, got %d items", offlineBatchSize)
	}
	if reconcileBatchSize != 0 {
		t.Errorf("Expected reconcile batch to be empty, got %d items", reconcileBatchSize)
	}
}

func TestBatchProcessor_Lifecycle(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	bp := NewBatchProcessor(config)

	// Start
	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Add some messages
	for i := 0; i < 3; i++ {
		msg := BatchMessage{
			ID:        string(rune('A' + i)),
			RegionID:  "region-a",
			GlobalID:  "global-" + string(rune('A'+i)),
			Priority:  1,
			CreatedAt: time.Now(),
		}
		bp.AddMessage(msg)
	}

	// Stop (should flush remaining batches)
	if err := bp.Stop(); err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	// Verify all batches are flushed
	bp.messageBatchMu.Lock()
	batchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected batch to be flushed on stop, but has %d items", batchSize)
	}
}

func TestBatchProcessor_ConcurrentAdds(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 100
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer bp.Stop()

	// Concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				msg := BatchMessage{
					ID:        string(rune('A' + id*10 + j)),
					RegionID:  "region-a",
					GlobalID:  "global-" + string(rune('A'+id*10+j)),
					Priority:  1,
					CreatedAt: time.Now(),
				}
				bp.AddMessage(msg)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()
	if metrics.TotalMessagesBatched != 100 {
		t.Errorf("Expected 100 messages batched, got %d", metrics.TotalMessagesBatched)
	}
}

func TestBatchProcessor_RetryLogic(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageMaxRetries = 2
	bp := NewBatchProcessor(config)

	// Create messages with different retry counts
	messages := []BatchMessage{
		{ID: "A", RetryCount: 0},
		{ID: "B", RetryCount: 1},
		{ID: "C", RetryCount: 2}, // Should fail (at max retries)
	}

	// Process batch
	result := bp.processMessageBatch(context.Background(), messages)

	// Check results
	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", result.SuccessCount)
	}

	if result.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", result.FailureCount)
	}

	if len(result.RetryRequired) != 1 {
		t.Errorf("Expected 1 retry required, got %d", len(result.RetryRequired))
	}
}

func TestDefaultBatchProcessorConfig(t *testing.T) {
	config := DefaultBatchProcessorConfig()

	if config.MessageBatchSize <= 0 {
		t.Error("Expected positive message batch size")
	}

	if config.MessageFlushInterval <= 0 {
		t.Error("Expected positive message flush interval")
	}

	if config.OfflineBatchSize <= 0 {
		t.Error("Expected positive offline batch size")
	}

	if config.ReconcileBatchSize <= 0 {
		t.Error("Expected positive reconcile batch size")
	}

	if !config.EnableMetrics {
		t.Error("Expected metrics to be enabled by default")
	}

	if !config.EnableCompression {
		t.Error("Expected compression to be enabled by default")
	}

	if !config.EnablePipelining {
		t.Error("Expected pipelining to be enabled by default")
	}
}

func TestBatchProcessor_MultipleFlushTypes(t *testing.T) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 2
	config.OfflineBatchSize = 2
	config.ReconcileBatchSize = 2
	config.MessageFlushInterval = 50 * time.Millisecond
	config.OfflineFlushInterval = 50 * time.Millisecond
	config.ReconcileFlushInterval = 50 * time.Millisecond
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer bp.Stop()

	// Add different types of items
	bp.AddMessage(BatchMessage{ID: "M1", Priority: 1, CreatedAt: time.Now()})
	bp.AddOfflineMessage(BatchOfflineMessage{ID: "O1", CreatedAt: time.Now()})
	bp.AddReconcileItem(BatchReconcileItem{GlobalID: "R1", Priority: 1, CreatedAt: time.Now()})

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()

	if metrics.TotalMessagesBatched != 1 {
		t.Errorf("Expected 1 message batched, got %d", metrics.TotalMessagesBatched)
	}

	if metrics.TotalOfflineBatched != 1 {
		t.Errorf("Expected 1 offline message batched, got %d", metrics.TotalOfflineBatched)
	}

	if metrics.TotalReconcileBatched != 1 {
		t.Errorf("Expected 1 reconcile item batched, got %d", metrics.TotalReconcileBatched)
	}
}

// Benchmark tests
func BenchmarkBatchProcessor_AddMessage(b *testing.B) {
	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 10000 // Large size to prevent flushes during benchmark
	bp := NewBatchProcessor(config)

	msg := BatchMessage{
		ID:             "test",
		RegionID:       "region-a",
		GlobalID:       "global-test",
		ConversationID: "conv-1",
		SenderID:       "user-1",
		Content:        "test message",
		Timestamp:      time.Now().UnixMilli(),
		Priority:       1,
		CreatedAt:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.AddMessage(msg)
	}
}

func BenchmarkBatchProcessor_FlushMessageBatch(b *testing.B) {
	config := DefaultBatchProcessorConfig()
	bp := NewBatchProcessor(config)

	// Prepare batch
	for i := 0; i < config.MessageBatchSize; i++ {
		msg := BatchMessage{
			ID:        string(rune('A' + i%26)),
			Priority:  1,
			CreatedAt: time.Now(),
		}
		bp.messageBatch = append(bp.messageBatch, msg)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.flushMessageBatch(ctx)
		// Refill batch for next iteration
		for j := 0; j < config.MessageBatchSize; j++ {
			bp.messageBatch = append(bp.messageBatch, BatchMessage{
				ID:        string(rune('A' + j%26)),
				Priority:  1,
				CreatedAt: time.Now(),
			})
		}
	}
}
