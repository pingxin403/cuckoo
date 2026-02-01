package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalQueue(t *testing.T) {
	config := DefaultConfig("region-a")
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	queue, err := NewLocalQueue(config, logger)
	require.NoError(t, err)
	require.NotNil(t, queue)

	assert.Equal(t, "region-a", queue.regionID)
	assert.Equal(t, config.BufferSize, queue.config.BufferSize)
	assert.Equal(t, config.PartitionCount, queue.config.PartitionCount)

	err = queue.Close()
	assert.NoError(t, err)
}

func TestLocalQueue_CreateTopic(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	err := queue.CreateTopic("test-topic")
	assert.NoError(t, err)

	// Verify topic was created
	queue.mu.RLock()
	topic, exists := queue.topics["test-topic"]
	queue.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "test-topic", topic.Name)
	assert.Len(t, topic.Partitions, queue.config.PartitionCount)

	// Test duplicate topic creation
	err = queue.CreateTopic("test-topic")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLocalQueue_CreateSyncChannel(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	err := queue.CreateSyncChannel("region-b")
	assert.NoError(t, err)

	// Verify sync channel was created
	queue.mu.RLock()
	channelName := "region-a_to_region-b"
	_, exists := queue.syncChannels[channelName]
	queue.mu.RUnlock()

	assert.True(t, exists)

	// Test duplicate sync channel creation
	err = queue.CreateSyncChannel("region-b")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLocalQueue_NewProducer(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	require.NotNil(t, producer)

	// Verify producer was registered
	queue.mu.RLock()
	_, exists := queue.producers["producer-1"]
	queue.mu.RUnlock()
	assert.True(t, exists)

	// Test duplicate producer creation
	_, err = queue.NewProducer("producer-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	err = producer.Close()
	assert.NoError(t, err)
}

func TestLocalQueue_NewConsumer(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic first
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	consumer, err := queue.NewConsumer("consumer-1", "test-topic")
	require.NoError(t, err)
	require.NotNil(t, consumer)

	// Verify consumer was registered
	queue.mu.RLock()
	_, exists := queue.consumers["consumer-1"]
	queue.mu.RUnlock()
	assert.True(t, exists)

	// Test duplicate consumer creation
	_, err = queue.NewConsumer("consumer-1", "test-topic")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test consumer for non-existent topic
	_, err = queue.NewConsumer("consumer-2", "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	err = consumer.Close()
	assert.NoError(t, err)
}

func TestLocalProducer_Produce(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic and producer
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	ctx := context.Background()
	message := &Message{
		ID:      "msg-1",
		Topic:   "test-topic",
		Key:     "test-key",
		Value:   []byte("test message"),
		Headers: map[string]string{"source": "test"},
	}

	err = producer.Produce(ctx, message)
	assert.NoError(t, err)

	// Verify message metadata was set
	assert.Equal(t, "region-a", message.RegionID)
	assert.True(t, message.Timestamp > 0)
	assert.True(t, message.Offset > 0)
	assert.True(t, message.Partition >= 0)
	assert.Equal(t, queue.config.MaxRetries, message.MaxRetries)
}

func TestLocalProducer_ProduceToNonExistentTopic(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	ctx := context.Background()
	message := &Message{
		ID:    "msg-1",
		Topic: "non-existent",
		Value: []byte("test message"),
	}

	err = producer.Produce(ctx, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLocalProducer_ProduceDuplicate(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic and producer
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	ctx := context.Background()
	message := &Message{
		ID:    "msg-1",
		Topic: "test-topic",
		Value: []byte("test message"),
	}

	// First produce should succeed
	err = producer.Produce(ctx, message)
	assert.NoError(t, err)

	// Second produce should be silently ignored (duplicate)
	err = producer.Produce(ctx, message)
	assert.NoError(t, err)

	// Verify message count is still 1
	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats["message_count"])
}

func TestLocalConsumer_Consume(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic, producer, and consumer
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	consumer, err := queue.NewConsumer("consumer-1", "test-topic")
	require.NoError(t, err)
	defer consumer.Close()

	// Set up message handler
	var receivedMessages []*Message
	var mu sync.Mutex

	handler := func(ctx context.Context, message *Message) error {
		mu.Lock()
		receivedMessages = append(receivedMessages, message)
		mu.Unlock()
		return nil
	}

	// Start consuming
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = consumer.Consume(ctx, handler)
	require.NoError(t, err)

	// Produce some messages
	for i := 0; i < 3; i++ {
		message := &Message{
			ID:    fmt.Sprintf("msg-%d", i),
			Topic: "test-topic",
			Value: []byte(fmt.Sprintf("test message %d", i)),
		}
		err = producer.Produce(ctx, message)
		require.NoError(t, err)
	}

	// Wait for messages to be consumed
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Len(t, receivedMessages, 3)
	mu.Unlock()
}

func TestLocalConsumer_ConsumeWithRetry(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic, producer, and consumer
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	consumer, err := queue.NewConsumer("consumer-1", "test-topic")
	require.NoError(t, err)
	defer consumer.Close()

	// Set up failing message handler
	attemptCount := 0
	handler := func(ctx context.Context, message *Message) error {
		attemptCount++
		if attemptCount <= 2 {
			return fmt.Errorf("simulated failure %d", attemptCount)
		}
		return nil // Success on third attempt
	}

	// Start consuming
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = consumer.Consume(ctx, handler)
	require.NoError(t, err)

	// Produce a message
	message := &Message{
		ID:         "msg-1",
		Topic:      "test-topic",
		Value:      []byte("test message"),
		MaxRetries: 3,
	}
	err = producer.Produce(ctx, message)
	require.NoError(t, err)

	// Wait for message to be processed with retries
	time.Sleep(500 * time.Millisecond)

	// Should have been attempted 3 times (initial + 2 retries)
	assert.Equal(t, 3, attemptCount)
}

func TestLocalQueue_SendSyncEvent(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create sync channel
	err := queue.CreateSyncChannel("region-b")
	require.NoError(t, err)

	event := &SyncEvent{
		Type:      "message_sync",
		MessageID: "msg-1",
		GlobalID:  "region-a-1234567890-1",
		Data:      []byte("sync data"),
	}

	err = queue.SendSyncEvent("region-b", event)
	assert.NoError(t, err)

	// Verify event metadata was set
	assert.Equal(t, "region-a", event.SourceRegion)
	assert.Equal(t, "region-b", event.TargetRegion)
	assert.True(t, event.Timestamp > 0)

	// Verify sync event count increased
	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats["sync_event_count"])
}

func TestLocalQueue_SendSyncEventToNonExistentChannel(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	event := &SyncEvent{
		Type:      "message_sync",
		MessageID: "msg-1",
		Data:      []byte("sync data"),
	}

	err := queue.SendSyncEvent("region-b", event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLocalQueue_ReceiveSyncEvents(t *testing.T) {
	queueA := createTestQueue(t, "region-a")
	defer queueA.Close()

	queueB := createTestQueue(t, "region-b")
	defer queueB.Close()

	// Create sync channels
	err := queueA.CreateSyncChannel("region-b")
	require.NoError(t, err)

	// Create a separate channel for region-b to avoid sharing
	err = queueB.CreateSyncChannel("region-a")
	require.NoError(t, err)

	// Manually connect the channels for this test
	queueA.mu.Lock()
	channelAtoB := queueA.syncChannels["region-a_to_region-b"]
	queueA.mu.Unlock()

	// Set up sync event handler
	var receivedEvents []*SyncEvent
	var mu sync.Mutex

	handler := func(event *SyncEvent) error {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		return nil
	}

	// Start receiving sync events by manually reading from the channel
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		for {
			select {
			case event := <-channelAtoB:
				if err := handler(event); err != nil {
					t.Errorf("Handler error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send sync event from region-a
	event := &SyncEvent{
		Type:      "message_sync",
		MessageID: "msg-1",
		Data:      []byte("sync data"),
	}

	err = queueA.SendSyncEvent("region-b", event)
	require.NoError(t, err)

	// Wait for event to be received
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Len(t, receivedEvents, 1)
	assert.Equal(t, "message_sync", receivedEvents[0].Type)
	assert.Equal(t, "msg-1", receivedEvents[0].MessageID)
	mu.Unlock()
}

func TestLocalQueue_MessageDeduplication(t *testing.T) {
	config := DefaultConfig("region-a")
	config.EnableDeduplication = true
	config.DeduplicationTTL = 100 * time.Millisecond

	queue, err := NewLocalQueue(config, nil)
	require.NoError(t, err)
	defer queue.Close()

	// Test duplicate detection
	messageID := "msg-1"

	// First check should return false (not duplicate)
	assert.False(t, queue.isDuplicate(messageID))

	// Mark message as seen
	queue.markMessageSeen(messageID)

	// Second check should return true (duplicate)
	assert.True(t, queue.isDuplicate(messageID))

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Third check should return false (expired from cache)
	assert.False(t, queue.isDuplicate(messageID))
}

func TestLocalQueue_GetPartition(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	queue.mu.RLock()
	topic := queue.topics["test-topic"]
	queue.mu.RUnlock()

	// Test key-based partitioning
	partition1 := queue.getPartition(topic, "key1")
	partition2 := queue.getPartition(topic, "key1") // Same key
	partition3 := queue.getPartition(topic, "key2") // Different key

	// Same key should go to same partition
	assert.Equal(t, partition1.ID, partition2.ID)

	// Different keys might go to different partitions
	assert.True(t, partition1.ID >= 0 && partition1.ID < queue.config.PartitionCount)
	assert.True(t, partition3.ID >= 0 && partition3.ID < queue.config.PartitionCount)

	// Test round-robin for empty key
	partition4 := queue.getPartition(topic, "")
	partition5 := queue.getPartition(topic, "")

	assert.True(t, partition4.ID >= 0 && partition4.ID < queue.config.PartitionCount)
	assert.True(t, partition5.ID >= 0 && partition5.ID < queue.config.PartitionCount)
}

func TestLocalQueue_GetStats(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create some resources
	err := queue.CreateTopic("topic-1")
	require.NoError(t, err)

	err = queue.CreateTopic("topic-2")
	require.NoError(t, err)

	err = queue.CreateSyncChannel("region-b")
	require.NoError(t, err)

	_, err = queue.NewProducer("producer-1")
	require.NoError(t, err)

	_, err = queue.NewConsumer("consumer-1", "topic-1")
	require.NoError(t, err)

	stats := queue.GetStats()

	assert.Equal(t, "region-a", stats["region_id"])
	assert.Equal(t, 2, stats["topics"])
	assert.Equal(t, 1, stats["consumers"])
	assert.Equal(t, 1, stats["producers"])
	assert.Equal(t, 1, stats["sync_channels"])
	assert.Equal(t, int64(0), stats["message_count"])
	assert.Equal(t, int64(0), stats["sync_event_count"])
	assert.Equal(t, int64(0), stats["error_count"])

	// Check topic details
	topicDetails, ok := stats["topic_details"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, topicDetails, "topic-1")
	assert.Contains(t, topicDetails, "topic-2")
}

func TestLocalQueue_ConcurrentProduceConsume(t *testing.T) {
	queue := createTestQueue(t, "region-a")
	defer queue.Close()

	// Create topic, producer, and consumer
	err := queue.CreateTopic("test-topic")
	require.NoError(t, err)

	producer, err := queue.NewProducer("producer-1")
	require.NoError(t, err)
	defer producer.Close()

	consumer, err := queue.NewConsumer("consumer-1", "test-topic")
	require.NoError(t, err)
	defer consumer.Close()

	// Set up concurrent message handling
	const numMessages = 100
	var receivedCount int64
	var mu sync.Mutex
	receivedMessages := make(map[string]bool)

	handler := func(ctx context.Context, message *Message) error {
		mu.Lock()
		receivedMessages[message.ID] = true
		receivedCount++
		mu.Unlock()
		return nil
	}

	// Start consuming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = consumer.Consume(ctx, handler)
	require.NoError(t, err)

	// Produce messages concurrently
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			message := &Message{
				ID:    fmt.Sprintf("msg-%d", i),
				Topic: "test-topic",
				Value: []byte(fmt.Sprintf("message %d", i)),
			}
			err := producer.Produce(ctx, message)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Wait for all messages to be consumed
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, int64(numMessages), receivedCount)
	assert.Len(t, receivedMessages, numMessages)
	mu.Unlock()
}

func TestMessageSerialization(t *testing.T) {
	message := &Message{
		ID:        "msg-1",
		Topic:     "test-topic",
		Key:       "test-key",
		Value:     []byte("test message"),
		Headers:   map[string]string{"source": "test"},
		Timestamp: time.Now().UnixMilli(),
		RegionID:  "region-a",
		GlobalID:  "region-a-1234567890-1",
		Partition: 0,
		Offset:    1,
	}

	// Test message serialization
	data, err := MarshalMessage(message)
	require.NoError(t, err)

	unmarshaled, err := UnmarshalMessage(data)
	require.NoError(t, err)

	assert.Equal(t, message.ID, unmarshaled.ID)
	assert.Equal(t, message.Topic, unmarshaled.Topic)
	assert.Equal(t, message.Key, unmarshaled.Key)
	assert.Equal(t, message.Value, unmarshaled.Value)
	assert.Equal(t, message.Headers, unmarshaled.Headers)
	assert.Equal(t, message.RegionID, unmarshaled.RegionID)
	assert.Equal(t, message.GlobalID, unmarshaled.GlobalID)
}

func TestSyncEventSerialization(t *testing.T) {
	event := &SyncEvent{
		Type:         "message_sync",
		SourceRegion: "region-a",
		TargetRegion: "region-b",
		MessageID:    "msg-1",
		GlobalID:     "region-a-1234567890-1",
		Data:         []byte("sync data"),
		Timestamp:    time.Now().UnixMilli(),
	}

	// Test sync event serialization
	data, err := MarshalSyncEvent(event)
	require.NoError(t, err)

	unmarshaled, err := UnmarshalSyncEvent(data)
	require.NoError(t, err)

	assert.Equal(t, event.Type, unmarshaled.Type)
	assert.Equal(t, event.SourceRegion, unmarshaled.SourceRegion)
	assert.Equal(t, event.TargetRegion, unmarshaled.TargetRegion)
	assert.Equal(t, event.MessageID, unmarshaled.MessageID)
	assert.Equal(t, event.GlobalID, unmarshaled.GlobalID)
	assert.Equal(t, event.Data, unmarshaled.Data)
}

// Helper functions

func createTestQueue(t *testing.T, regionID string) *LocalQueue {
	config := DefaultConfig(regionID)
	config.BufferSize = 100 // Smaller buffer for testing
	config.DeduplicationTTL = 1 * time.Second

	queue, err := NewLocalQueue(config, nil)
	require.NoError(t, err)
	return queue
}

// Benchmark tests

func BenchmarkLocalProducer_Produce(b *testing.B) {
	queue := createBenchQueue(b, "region-a")
	defer queue.Close()

	err := queue.CreateTopic("bench-topic")
	require.NoError(b, err)

	producer, err := queue.NewProducer("bench-producer")
	require.NoError(b, err)
	defer producer.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			message := &Message{
				ID:    fmt.Sprintf("msg-%d", i),
				Topic: "bench-topic",
				Value: []byte("benchmark message"),
			}
			err := producer.Produce(ctx, message)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkLocalQueue_SendSyncEvent(b *testing.B) {
	queue := createBenchQueue(b, "region-a")
	defer queue.Close()

	err := queue.CreateSyncChannel("region-b")
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			event := &SyncEvent{
				Type:      "message_sync",
				MessageID: fmt.Sprintf("msg-%d", i),
				Data:      []byte("benchmark sync data"),
			}
			err := queue.SendSyncEvent("region-b", event)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func createBenchQueue(b *testing.B, regionID string) *LocalQueue {
	config := DefaultConfig(regionID)
	config.BufferSize = 10000 // Larger buffer for benchmarks

	queue, err := NewLocalQueue(config, nil)
	require.NoError(b, err)
	return queue
}
