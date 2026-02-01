package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Message represents a message in the queue system
type Message struct {
	ID         string            `json:"id"`
	Topic      string            `json:"topic"`
	Key        string            `json:"key,omitempty"`
	Value      []byte            `json:"value"`
	Headers    map[string]string `json:"headers,omitempty"`
	Timestamp  int64             `json:"timestamp"`
	RegionID   string            `json:"region_id"`
	GlobalID   string            `json:"global_id"`
	Partition  int               `json:"partition"`
	Offset     int64             `json:"offset"`
	Retry      int               `json:"retry"`
	MaxRetries int               `json:"max_retries"`
	DeadLetter bool              `json:"dead_letter"`
}

// SyncEvent represents a cross-region synchronization event
type SyncEvent struct {
	Type         string `json:"type"`
	SourceRegion string `json:"source_region"`
	TargetRegion string `json:"target_region"`
	MessageID    string `json:"message_id"`
	GlobalID     string `json:"global_id"`
	Data         []byte `json:"data"`
	Timestamp    int64  `json:"timestamp"`
	Checksum     string `json:"checksum,omitempty"`
}

// Consumer represents a message consumer
type Consumer interface {
	Consume(ctx context.Context, handler MessageHandler) error
	Close() error
}

// Producer represents a message producer
type Producer interface {
	Produce(ctx context.Context, message *Message) error
	ProduceSync(ctx context.Context, message *Message) error
	Close() error
}

// MessageHandler handles consumed messages
type MessageHandler func(ctx context.Context, message *Message) error

// LocalQueue implements a simplified message queue using Go channels
type LocalQueue struct {
	regionID     string
	topics       map[string]*Topic
	consumers    map[string]*LocalConsumer
	producers    map[string]*LocalProducer
	syncChannels map[string]chan *SyncEvent // Cross-region sync channels

	// Message deduplication
	messageCache map[string]time.Time // messageID -> timestamp
	cacheTTL     time.Duration

	// Metrics
	messageCount     int64
	syncEventCount   int64
	errorCount       int64
	partitionCounter int64 // Separate counter for round-robin partitioning

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger *log.Logger
	closed bool // Track if queue is closed

	// Configuration
	config Config
}

// Topic represents a message topic with multiple partitions
type Topic struct {
	Name       string
	Partitions []*Partition
	mu         sync.RWMutex
}

// Partition represents a single partition within a topic
type Partition struct {
	ID       int
	Messages chan *Message
	Offset   int64
	mu       sync.RWMutex
}

// Config holds local queue configuration
type Config struct {
	RegionID            string        `json:"region_id"`
	BufferSize          int           `json:"buffer_size"`
	PartitionCount      int           `json:"partition_count"`
	MessageTTL          time.Duration `json:"message_ttl"`
	SyncInterval        time.Duration `json:"sync_interval"`
	MaxRetries          int           `json:"max_retries"`
	EnablePersistence   bool          `json:"enable_persistence"`
	PersistencePath     string        `json:"persistence_path"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	DeduplicationTTL    time.Duration `json:"deduplication_ttl"`
}

// DefaultConfig returns default configuration
func DefaultConfig(regionID string) Config {
	return Config{
		RegionID:            regionID,
		BufferSize:          10000,
		PartitionCount:      3,
		MessageTTL:          24 * time.Hour,
		SyncInterval:        100 * time.Millisecond,
		MaxRetries:          3,
		EnablePersistence:   false,
		EnableDeduplication: true,
		DeduplicationTTL:    5 * time.Minute,
	}
}

// NewLocalQueue creates a new local message queue
func NewLocalQueue(config Config, logger *log.Logger) (*LocalQueue, error) {
	if logger == nil {
		logger = log.New(log.Writer(), "[LocalQueue] ", log.LstdFlags|log.Lshortfile)
	}

	ctx, cancel := context.WithCancel(context.Background())

	queue := &LocalQueue{
		regionID:     config.RegionID,
		topics:       make(map[string]*Topic),
		consumers:    make(map[string]*LocalConsumer),
		producers:    make(map[string]*LocalProducer),
		syncChannels: make(map[string]chan *SyncEvent),
		messageCache: make(map[string]time.Time),
		cacheTTL:     config.DeduplicationTTL,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		config:       config,
	}

	// Start background tasks
	queue.wg.Add(2)
	go queue.cleanupExpiredMessages()
	go queue.cleanupMessageCache()

	logger.Printf("LocalQueue initialized for region %s", config.RegionID)
	return queue, nil
}

// CreateTopic creates a new topic with specified partitions
func (lq *LocalQueue) CreateTopic(topicName string) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	if _, exists := lq.topics[topicName]; exists {
		return fmt.Errorf("topic %s already exists", topicName)
	}

	topic := &Topic{
		Name:       topicName,
		Partitions: make([]*Partition, lq.config.PartitionCount),
	}

	// Initialize partitions
	for i := 0; i < lq.config.PartitionCount; i++ {
		topic.Partitions[i] = &Partition{
			ID:       i,
			Messages: make(chan *Message, lq.config.BufferSize),
			Offset:   0,
		}
	}

	lq.topics[topicName] = topic
	lq.logger.Printf("Created topic %s with %d partitions", topicName, lq.config.PartitionCount)
	return nil
}

// CreateSyncChannel creates a cross-region sync channel
func (lq *LocalQueue) CreateSyncChannel(targetRegion string) error {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	channelName := fmt.Sprintf("%s_to_%s", lq.regionID, targetRegion)
	if _, exists := lq.syncChannels[channelName]; exists {
		return fmt.Errorf("sync channel %s already exists", channelName)
	}

	lq.syncChannels[channelName] = make(chan *SyncEvent, lq.config.BufferSize)
	lq.logger.Printf("Created sync channel: %s", channelName)
	return nil
}

// NewProducer creates a new message producer
func (lq *LocalQueue) NewProducer(producerID string) (Producer, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	if _, exists := lq.producers[producerID]; exists {
		return nil, fmt.Errorf("producer %s already exists", producerID)
	}

	producer := &LocalProducer{
		id:    producerID,
		queue: lq,
	}

	lq.producers[producerID] = producer
	lq.logger.Printf("Created producer: %s", producerID)
	return producer, nil
}

// NewConsumer creates a new message consumer
func (lq *LocalQueue) NewConsumer(consumerID, topicName string) (Consumer, error) {
	lq.mu.Lock()
	defer lq.mu.Unlock()

	if _, exists := lq.consumers[consumerID]; exists {
		return nil, fmt.Errorf("consumer %s already exists", consumerID)
	}

	topic, exists := lq.topics[topicName]
	if !exists {
		return nil, fmt.Errorf("topic %s does not exist", topicName)
	}

	consumer := &LocalConsumer{
		id:    consumerID,
		topic: topic,
		queue: lq,
	}

	lq.consumers[consumerID] = consumer
	lq.logger.Printf("Created consumer: %s for topic: %s", consumerID, topicName)
	return consumer, nil
}

// SendSyncEvent sends a synchronization event to another region
func (lq *LocalQueue) SendSyncEvent(targetRegion string, event *SyncEvent) error {
	lq.mu.RLock()
	channelName := fmt.Sprintf("%s_to_%s", lq.regionID, targetRegion)
	syncChan, exists := lq.syncChannels[channelName]
	lq.mu.RUnlock()

	if !exists {
		return fmt.Errorf("sync channel to %s does not exist", targetRegion)
	}

	event.SourceRegion = lq.regionID
	event.TargetRegion = targetRegion
	event.Timestamp = time.Now().UnixMilli()

	select {
	case syncChan <- event:
		atomic.AddInt64(&lq.syncEventCount, 1)
		lq.logger.Printf("Sent sync event %s to %s", event.Type, targetRegion)
		return nil
	case <-lq.ctx.Done():
		return fmt.Errorf("queue is shutting down")
	default:
		atomic.AddInt64(&lq.errorCount, 1)
		return fmt.Errorf("sync channel to %s is full", targetRegion)
	}
}

// ReceiveSyncEvents receives synchronization events from another region
func (lq *LocalQueue) ReceiveSyncEvents(sourceRegion string, handler func(*SyncEvent) error) error {
	lq.mu.RLock()
	channelName := fmt.Sprintf("%s_to_%s", sourceRegion, lq.regionID)
	syncChan, exists := lq.syncChannels[channelName]
	lq.mu.RUnlock()

	if !exists {
		return fmt.Errorf("sync channel from %s does not exist", sourceRegion)
	}

	lq.wg.Add(1)
	go func() {
		defer lq.wg.Done()

		for {
			select {
			case event := <-syncChan:
				if err := handler(event); err != nil {
					lq.logger.Printf("Error handling sync event from %s: %v", sourceRegion, err)
					atomic.AddInt64(&lq.errorCount, 1)
				} else {
					lq.logger.Printf("Processed sync event %s from %s", event.Type, sourceRegion)
				}
			case <-lq.ctx.Done():
				lq.logger.Printf("Stopped receiving sync events from %s", sourceRegion)
				return
			}
		}
	}()

	return nil
}

// isDuplicate checks if a message is a duplicate
func (lq *LocalQueue) isDuplicate(messageID string) bool {
	if !lq.config.EnableDeduplication {
		return false
	}

	lq.mu.RLock()
	timestamp, exists := lq.messageCache[messageID]
	lq.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if the cached entry is still valid
	return time.Since(timestamp) < lq.cacheTTL
}

// markMessageSeen marks a message as seen for deduplication
func (lq *LocalQueue) markMessageSeen(messageID string) {
	if !lq.config.EnableDeduplication {
		return
	}

	lq.mu.Lock()
	lq.messageCache[messageID] = time.Now()
	lq.mu.Unlock()
}

// getPartition determines which partition to use for a message
func (lq *LocalQueue) getPartition(topic *Topic, key string) *Partition {
	if key == "" {
		// Round-robin if no key
		partitionID := int(atomic.AddInt64(&lq.partitionCounter, 1)) % len(topic.Partitions)
		return topic.Partitions[partitionID]
	}

	// Hash-based partitioning
	hash := 0
	for _, c := range key {
		hash = hash*31 + int(c)
	}
	partitionID := hash % len(topic.Partitions)
	if partitionID < 0 {
		partitionID = -partitionID
	}

	return topic.Partitions[partitionID]
}

// cleanupExpiredMessages removes expired messages (background task)
func (lq *LocalQueue) cleanupExpiredMessages() {
	defer lq.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// In a real implementation, we would clean up expired messages
			// For this simplified version, we just log the cleanup
			lq.logger.Printf("Cleanup task: message count=%d, sync events=%d, errors=%d",
				atomic.LoadInt64(&lq.messageCount),
				atomic.LoadInt64(&lq.syncEventCount),
				atomic.LoadInt64(&lq.errorCount))
		case <-lq.ctx.Done():
			lq.logger.Printf("Stopped cleanup task")
			return
		}
	}
}

// cleanupMessageCache removes expired entries from message cache
func (lq *LocalQueue) cleanupMessageCache() {
	defer lq.wg.Done()

	ticker := time.NewTicker(lq.cacheTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lq.mu.Lock()
			now := time.Now()
			for messageID, timestamp := range lq.messageCache {
				if now.Sub(timestamp) > lq.cacheTTL {
					delete(lq.messageCache, messageID)
				}
			}
			lq.mu.Unlock()
		case <-lq.ctx.Done():
			lq.logger.Printf("Stopped cache cleanup task")
			return
		}
	}
}

// GetStats returns queue statistics
func (lq *LocalQueue) GetStats() map[string]interface{} {
	lq.mu.RLock()
	defer lq.mu.RUnlock()

	stats := map[string]interface{}{
		"region_id":        lq.regionID,
		"topics":           len(lq.topics),
		"consumers":        len(lq.consumers),
		"producers":        len(lq.producers),
		"sync_channels":    len(lq.syncChannels),
		"message_count":    atomic.LoadInt64(&lq.messageCount),
		"sync_event_count": atomic.LoadInt64(&lq.syncEventCount),
		"error_count":      atomic.LoadInt64(&lq.errorCount),
		"cache_size":       len(lq.messageCache),
	}

	// Add topic-specific stats
	topicStats := make(map[string]interface{})
	for name, topic := range lq.topics {
		topic.mu.RLock()
		partitionStats := make([]map[string]interface{}, len(topic.Partitions))
		for i, partition := range topic.Partitions {
			partition.mu.RLock()
			partitionStats[i] = map[string]interface{}{
				"id":            partition.ID,
				"offset":        partition.Offset,
				"buffer_length": len(partition.Messages),
				"buffer_cap":    cap(partition.Messages),
			}
			partition.mu.RUnlock()
		}
		topicStats[name] = map[string]interface{}{
			"partitions": partitionStats,
		}
		topic.mu.RUnlock()
	}
	stats["topic_details"] = topicStats

	return stats
}

// Close gracefully shuts down the queue
func (lq *LocalQueue) Close() error {
	lq.mu.Lock()
	if lq.closed {
		lq.mu.Unlock()
		return nil // Already closed
	}
	lq.closed = true
	lq.mu.Unlock()

	lq.logger.Printf("Shutting down LocalQueue for region %s", lq.regionID)

	// Cancel context to stop background tasks
	lq.cancel()

	// Close all producers
	lq.mu.Lock()
	for id, producer := range lq.producers {
		if err := producer.Close(); err != nil {
			lq.logger.Printf("Error closing producer %s: %v", id, err)
		}
	}

	// Close all consumers
	for id, consumer := range lq.consumers {
		if err := consumer.Close(); err != nil {
			lq.logger.Printf("Error closing consumer %s: %v", id, err)
		}
	}

	// Close all topic partitions
	for _, topic := range lq.topics {
		for _, partition := range topic.Partitions {
			select {
			case <-partition.Messages:
				// Channel already closed
			default:
				close(partition.Messages)
			}
		}
	}

	// Close sync channels
	for name, ch := range lq.syncChannels {
		select {
		case <-ch:
			// Channel already closed
		default:
			close(ch)
			lq.logger.Printf("Closed sync channel: %s", name)
		}
	}
	lq.mu.Unlock()

	// Wait for background tasks to finish
	lq.wg.Wait()

	lq.logger.Printf("LocalQueue shutdown complete")
	return nil
}

// LocalProducer implements the Producer interface
type LocalProducer struct {
	id    string
	queue *LocalQueue
	mu    sync.Mutex
}

// Produce sends a message asynchronously
func (lp *LocalProducer) Produce(ctx context.Context, message *Message) error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	// Check for duplicates
	if lp.queue.isDuplicate(message.ID) {
		lp.queue.logger.Printf("Duplicate message detected: %s", message.ID)
		return nil // Silently ignore duplicates
	}

	// Get topic
	lp.queue.mu.RLock()
	topic, exists := lp.queue.topics[message.Topic]
	lp.queue.mu.RUnlock()

	if !exists {
		return fmt.Errorf("topic %s does not exist", message.Topic)
	}

	// Set message metadata
	message.RegionID = lp.queue.regionID
	message.Timestamp = time.Now().UnixMilli()
	if message.MaxRetries == 0 {
		message.MaxRetries = lp.queue.config.MaxRetries
	}

	// Get partition
	partition := lp.queue.getPartition(topic, message.Key)

	// Set partition and offset
	partition.mu.Lock()
	message.Partition = partition.ID
	message.Offset = atomic.AddInt64(&partition.Offset, 1)
	partition.mu.Unlock()

	// Send message to partition
	select {
	case partition.Messages <- message:
		lp.queue.markMessageSeen(message.ID)
		atomic.AddInt64(&lp.queue.messageCount, 1)
		lp.queue.logger.Printf("Produced message %s to topic %s partition %d",
			message.ID, message.Topic, message.Partition)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		atomic.AddInt64(&lp.queue.errorCount, 1)
		return fmt.Errorf("partition %d buffer is full", partition.ID)
	}
}

// ProduceSync sends a message synchronously (same as Produce for this implementation)
func (lp *LocalProducer) ProduceSync(ctx context.Context, message *Message) error {
	return lp.Produce(ctx, message)
}

// Close closes the producer
func (lp *LocalProducer) Close() error {
	lp.queue.logger.Printf("Closed producer: %s", lp.id)
	return nil
}

// LocalConsumer implements the Consumer interface
type LocalConsumer struct {
	id    string
	topic *Topic
	queue *LocalQueue
	mu    sync.Mutex
}

// Consume starts consuming messages from the topic
func (lc *LocalConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.queue.logger.Printf("Consumer %s started consuming from topic %s", lc.id, lc.topic.Name)

	// Start consuming from all partitions
	for _, partition := range lc.topic.Partitions {
		lc.queue.wg.Add(1)
		go lc.consumePartition(ctx, partition, handler)
	}

	return nil
}

// consumePartition consumes messages from a specific partition
func (lc *LocalConsumer) consumePartition(ctx context.Context, partition *Partition, handler MessageHandler) {
	defer lc.queue.wg.Done()

	for {
		select {
		case message := <-partition.Messages:
			if err := lc.handleMessage(ctx, message, handler); err != nil {
				lc.queue.logger.Printf("Error handling message %s: %v", message.ID, err)
				atomic.AddInt64(&lc.queue.errorCount, 1)
			}
		case <-ctx.Done():
			lc.queue.logger.Printf("Consumer %s stopped consuming partition %d", lc.id, partition.ID)
			return
		case <-lc.queue.ctx.Done():
			lc.queue.logger.Printf("Consumer %s stopped due to queue shutdown", lc.id)
			return
		}
	}
}

// handleMessage handles a single message with retry logic
func (lc *LocalConsumer) handleMessage(ctx context.Context, message *Message, handler MessageHandler) error {
	for attempt := 0; attempt <= message.MaxRetries; attempt++ {
		err := handler(ctx, message)
		if err == nil {
			lc.queue.logger.Printf("Successfully processed message %s", message.ID)
			return nil
		}

		if attempt < message.MaxRetries {
			lc.queue.logger.Printf("Message %s failed (attempt %d/%d): %v",
				message.ID, attempt+1, message.MaxRetries+1, err)

			// Exponential backoff
			backoff := time.Duration(attempt+1) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Mark as dead letter after all retries exhausted
	message.DeadLetter = true
	lc.queue.logger.Printf("Message %s moved to dead letter queue after %d attempts",
		message.ID, message.MaxRetries+1)

	return fmt.Errorf("message %s failed after %d attempts", message.ID, message.MaxRetries+1)
}

// Close closes the consumer
func (lc *LocalConsumer) Close() error {
	lc.queue.logger.Printf("Closed consumer: %s", lc.id)
	return nil
}

// Helper functions for message serialization

// MarshalMessage serializes a message to JSON
func MarshalMessage(message *Message) ([]byte, error) {
	return json.Marshal(message)
}

// UnmarshalMessage deserializes a message from JSON
func UnmarshalMessage(data []byte) (*Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	return &message, err
}

// MarshalSyncEvent serializes a sync event to JSON
func MarshalSyncEvent(event *SyncEvent) ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalSyncEvent deserializes a sync event from JSON
func UnmarshalSyncEvent(data []byte) (*SyncEvent, error) {
	var event SyncEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}
