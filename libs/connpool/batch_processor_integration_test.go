//go:build integration
// +build integration

package connpool

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

func TestBatchProcessorWithKafka_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Kafka producer
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionSnappy

	brokers := []string{"localhost:9092"}
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		t.Skipf("Kafka not available: %v", err)
	}
	defer producer.Close()

	// Create batch processor with Kafka
	bpConfig := DefaultBatchProcessorConfig()
	bpConfig.MessageBatchSize = 5
	bpConfig.MessageFlushInterval = 100 * time.Millisecond

	bp := NewBatchProcessorWithKafka(bpConfig, producer)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer bp.Stop()

	// Add messages
	for i := 0; i < 10; i++ {
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

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()
	if metrics.TotalMessagesBatched != 10 {
		t.Errorf("Expected 10 messages batched, got %d", metrics.TotalMessagesBatched)
	}

	if metrics.TotalBatchesFlushed == 0 {
		t.Error("Expected at least one batch to be flushed")
	}
}

func TestBatchProcessorWithRedis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// Test connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Create batch processor with Redis
	bpConfig := DefaultBatchProcessorConfig()
	bpConfig.OfflineBatchSize = 5
	bpConfig.OfflineFlushInterval = 100 * time.Millisecond

	bp := NewBatchProcessorWithRedis(bpConfig, redisClient)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start batch processor: %v", err)
	}
	defer bp.Stop()

	// Add offline messages
	for i := 0; i < 10; i++ {
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

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()
	if metrics.TotalOfflineBatched != 10 {
		t.Errorf("Expected 10 offline messages batched, got %d", metrics.TotalOfflineBatched)
	}

	if metrics.TotalBatchesFlushed == 0 {
		t.Error("Expected at least one batch to be flushed")
	}
}

func TestBatchProcessor_HighThroughput_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 100
	config.MessageFlushInterval = 50 * time.Millisecond
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer bp.Stop()

	// Simulate high throughput
	numMessages := 1000
	done := make(chan bool)

	go func() {
		for i := 0; i < numMessages; i++ {
			msg := BatchMessage{
				ID:             string(rune('A' + i%26)),
				RegionID:       "region-a",
				GlobalID:       "global-" + string(rune('A'+i%26)),
				ConversationID: "conv-1",
				SenderID:       "user-1",
				Content:        "test message",
				Timestamp:      time.Now().UnixMilli(),
				Priority:       i % 10, // Varying priorities
				CreatedAt:      time.Now(),
			}

			if err := bp.AddMessage(msg); err != nil {
				t.Errorf("Failed to add message: %v", err)
			}
		}
		done <- true
	}()

	// Wait for completion
	<-done

	// Wait for all batches to be processed
	time.Sleep(1 * time.Second)

	// Check metrics
	metrics := bp.GetMetrics()

	if metrics.TotalMessagesBatched != int64(numMessages) {
		t.Errorf("Expected %d messages batched, got %d", numMessages, metrics.TotalMessagesBatched)
	}

	if metrics.TotalBatchesFlushed == 0 {
		t.Error("Expected batches to be flushed")
	}

	if metrics.AvgBatchSize == 0 {
		t.Error("Expected average batch size to be calculated")
	}

	t.Logf("Processed %d messages in %d batches", numMessages, metrics.TotalBatchesFlushed)
	t.Logf("Average batch size: %.2f", metrics.AvgBatchSize)
	t.Logf("Average flush duration: %v", metrics.AvgFlushDuration)
}

func TestBatchProcessor_MixedOperations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 10
	config.OfflineBatchSize = 10
	config.ReconcileBatchSize = 10
	config.MessageFlushInterval = 100 * time.Millisecond
	config.OfflineFlushInterval = 100 * time.Millisecond
	config.ReconcileFlushInterval = 100 * time.Millisecond

	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer bp.Stop()

	// Add mixed operations concurrently
	done := make(chan bool, 3)

	// Add messages
	go func() {
		for i := 0; i < 20; i++ {
			msg := BatchMessage{
				ID:        "M" + string(rune('A'+i%26)),
				Priority:  1,
				CreatedAt: time.Now(),
			}
			bp.AddMessage(msg)
		}
		done <- true
	}()

	// Add offline messages
	go func() {
		for i := 0; i < 20; i++ {
			msg := BatchOfflineMessage{
				ID:        "O" + string(rune('A'+i%26)),
				CreatedAt: time.Now(),
			}
			bp.AddOfflineMessage(msg)
		}
		done <- true
	}()

	// Add reconcile items
	go func() {
		for i := 0; i < 20; i++ {
			item := BatchReconcileItem{
				GlobalID:  "R" + string(rune('A'+i%26)),
				Priority:  1,
				CreatedAt: time.Now(),
			}
			bp.AddReconcileItem(item)
		}
		done <- true
	}()

	// Wait for all operations
	for i := 0; i < 3; i++ {
		<-done
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check metrics
	metrics := bp.GetMetrics()

	if metrics.TotalMessagesBatched != 20 {
		t.Errorf("Expected 20 messages batched, got %d", metrics.TotalMessagesBatched)
	}

	if metrics.TotalOfflineBatched != 20 {
		t.Errorf("Expected 20 offline messages batched, got %d", metrics.TotalOfflineBatched)
	}

	if metrics.TotalReconcileBatched != 20 {
		t.Errorf("Expected 20 reconcile items batched, got %d", metrics.TotalReconcileBatched)
	}

	t.Logf("Total batches flushed: %d", metrics.TotalBatchesFlushed)
	t.Logf("Average batch size: %.2f", metrics.AvgBatchSize)
	t.Logf("Average flush duration: %v", metrics.AvgFlushDuration)
}

func TestBatchProcessor_StressTest_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 50
	config.MessageFlushInterval = 50 * time.Millisecond
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer bp.Stop()

	// Stress test with multiple concurrent producers
	numProducers := 10
	messagesPerProducer := 100
	done := make(chan bool, numProducers)

	startTime := time.Now()

	for p := 0; p < numProducers; p++ {
		go func(producerID int) {
			for i := 0; i < messagesPerProducer; i++ {
				msg := BatchMessage{
					ID:        string(rune('A' + (producerID*messagesPerProducer+i)%26)),
					Priority:  i % 10,
					CreatedAt: time.Now(),
				}
				bp.AddMessage(msg)
			}
			done <- true
		}(p)
	}

	// Wait for all producers
	for i := 0; i < numProducers; i++ {
		<-done
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	duration := time.Since(startTime)

	// Check metrics
	metrics := bp.GetMetrics()

	expectedTotal := int64(numProducers * messagesPerProducer)
	if metrics.TotalMessagesBatched != expectedTotal {
		t.Errorf("Expected %d messages batched, got %d", expectedTotal, metrics.TotalMessagesBatched)
	}

	throughput := float64(expectedTotal) / duration.Seconds()

	t.Logf("Stress test results:")
	t.Logf("  Total messages: %d", expectedTotal)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f messages/sec", throughput)
	t.Logf("  Total batches: %d", metrics.TotalBatchesFlushed)
	t.Logf("  Average batch size: %.2f", metrics.AvgBatchSize)
	t.Logf("  Average flush duration: %v", metrics.AvgFlushDuration)
	t.Logf("  Errors: %d", metrics.TotalBatchErrors)
}

func TestBatchProcessor_GracefulShutdown_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultBatchProcessorConfig()
	config.MessageBatchSize = 100 // Large size to prevent auto-flush
	config.MessageFlushInterval = 10 * time.Second
	bp := NewBatchProcessor(config)

	if err := bp.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Add messages
	for i := 0; i < 50; i++ {
		msg := BatchMessage{
			ID:        string(rune('A' + i%26)),
			Priority:  1,
			CreatedAt: time.Now(),
		}
		bp.AddMessage(msg)
	}

	// Stop should flush remaining messages
	if err := bp.Stop(); err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	// Check that all batches are empty
	bp.messageBatchMu.Lock()
	batchSize := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	if batchSize != 0 {
		t.Errorf("Expected batch to be flushed on shutdown, but has %d items", batchSize)
	}

	// Check metrics
	metrics := bp.GetMetrics()
	if metrics.TotalMessagesBatched != 50 {
		t.Errorf("Expected 50 messages batched, got %d", metrics.TotalMessagesBatched)
	}
}
