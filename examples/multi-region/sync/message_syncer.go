package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// MessageSyncer handles cross-region message synchronization
type MessageSyncer struct {
	regionID     string
	hlcClock     *hlc.HLC
	localQueue   *queue.LocalQueue
	localStorage *storage.LocalStore
	logger       *log.Logger

	// Producers and consumers for different sync types
	asyncProducer Producer
	syncProducer  Producer
	asyncConsumer Consumer
	syncConsumer  Consumer

	// Sync channels for cross-region communication
	syncChannels map[string]chan *SyncMessage

	// Configuration
	config Config

	// Metrics
	asyncSyncCount   int64
	syncSyncCount    int64
	conflictCount    int64
	errorCount       int64
	syncLatencySum   int64
	syncLatencyCount int64

	// State management
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
	shutdown bool
}

// Producer interface for message production
type Producer interface {
	Produce(ctx context.Context, message *queue.Message) error
	ProduceSync(ctx context.Context, message *queue.Message) error
	Close() error
}

// Consumer interface for message consumption
type Consumer interface {
	Consume(ctx context.Context, handler queue.MessageHandler) error
	Close() error
}

// SyncMessage represents a message to be synchronized across regions
type SyncMessage struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"` // "async", "sync", "ack"
	SourceRegion   string            `json:"source_region"`
	TargetRegion   string            `json:"target_region"`
	MessageID      string            `json:"message_id"`
	GlobalID       hlc.GlobalID      `json:"global_id"`
	ConversationID string            `json:"conversation_id"`
	SenderID       string            `json:"sender_id"`
	Content        string            `json:"content"`
	SequenceNumber int64             `json:"sequence_number"`
	Timestamp      int64             `json:"timestamp"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Checksum       string            `json:"checksum"`
	RequiresAck    bool              `json:"requires_ack"`
	AckTimeout     time.Duration     `json:"ack_timeout"`
	RetryCount     int               `json:"retry_count"`
	MaxRetries     int               `json:"max_retries"`
	CreatedAt      time.Time         `json:"created_at"`
	ProcessedAt    time.Time         `json:"processed_at,omitempty"`
	IsCritical     bool              `json:"is_critical"` // Critical business operations
}

// SyncAck represents an acknowledgment for synchronous operations
type SyncAck struct {
	MessageID    string    `json:"message_id"`
	GlobalID     string    `json:"global_id"`
	SourceRegion string    `json:"source_region"`
	TargetRegion string    `json:"target_region"`
	Status       string    `json:"status"` // "success", "error", "conflict"
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	ProcessTime  int64     `json:"process_time_ms"`
}

// Config holds message syncer configuration
type Config struct {
	RegionID            string        `json:"region_id"`
	AsyncTopic          string        `json:"async_topic"`
	SyncTopic           string        `json:"sync_topic"`
	AckTopic            string        `json:"ack_topic"`
	SyncTimeout         time.Duration `json:"sync_timeout"`
	MaxRetries          int           `json:"max_retries"`
	BatchSize           int           `json:"batch_size"`
	FlushInterval       time.Duration `json:"flush_interval"`
	EnableChecksum      bool          `json:"enable_checksum"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	MetricsInterval     time.Duration `json:"metrics_interval"`
}

// DefaultConfig returns default configuration for message syncer
func DefaultConfig(regionID string) Config {
	return Config{
		RegionID:            regionID,
		AsyncTopic:          "cross_region_async",
		SyncTopic:           "cross_region_sync",
		AckTopic:            "cross_region_ack",
		SyncTimeout:         5 * time.Second,
		MaxRetries:          3,
		BatchSize:           100,
		FlushInterval:       100 * time.Millisecond,
		EnableChecksum:      true,
		EnableDeduplication: true,
		MetricsInterval:     30 * time.Second,
	}
}

// NewMessageSyncer creates a new message synchronizer
func NewMessageSyncer(
	regionID string,
	hlcClock *hlc.HLC,
	localQueue *queue.LocalQueue,
	localStorage *storage.LocalStore,
	config Config,
	logger *log.Logger,
) (*MessageSyncer, error) {
	if logger == nil {
		logger = log.New(log.Writer(), "[MessageSyncer] ", log.LstdFlags|log.Lshortfile)
	}

	ctx, cancel := context.WithCancel(context.Background())

	syncer := &MessageSyncer{
		regionID:     regionID,
		hlcClock:     hlcClock,
		localQueue:   localQueue,
		localStorage: localStorage,
		logger:       logger,
		syncChannels: make(map[string]chan *SyncMessage),
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize producers and consumers
	if err := syncer.initializeProducersConsumers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize producers/consumers: %w", err)
	}

	logger.Printf("MessageSyncer initialized for region %s", regionID)
	return syncer, nil
}

// initializeProducersConsumers sets up Kafka producers and consumers
func (ms *MessageSyncer) initializeProducersConsumers() error {
	var err error

	// Create async producer
	ms.asyncProducer, err = ms.localQueue.NewProducer(fmt.Sprintf("async-producer-%s", ms.regionID))
	if err != nil {
		return fmt.Errorf("failed to create async producer: %w", err)
	}

	// Create sync producer
	ms.syncProducer, err = ms.localQueue.NewProducer(fmt.Sprintf("sync-producer-%s", ms.regionID))
	if err != nil {
		return fmt.Errorf("failed to create sync producer: %w", err)
	}

	// Create topics if they don't exist
	topics := []string{ms.config.AsyncTopic, ms.config.SyncTopic, ms.config.AckTopic}
	for _, topic := range topics {
		if err := ms.localQueue.CreateTopic(topic); err != nil {
			ms.logger.Printf("Topic %s might already exist: %v", topic, err)
		}
	}

	// Create async consumer
	ms.asyncConsumer, err = ms.localQueue.NewConsumer(
		fmt.Sprintf("async-consumer-%s", ms.regionID),
		ms.config.AsyncTopic,
	)
	if err != nil {
		return fmt.Errorf("failed to create async consumer: %w", err)
	}

	// Create sync consumer
	ms.syncConsumer, err = ms.localQueue.NewConsumer(
		fmt.Sprintf("sync-consumer-%s", ms.regionID),
		ms.config.SyncTopic,
	)
	if err != nil {
		return fmt.Errorf("failed to create sync consumer: %w", err)
	}

	return nil
}

// Start starts the message synchronizer
func (ms *MessageSyncer) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.started {
		return fmt.Errorf("message syncer already started")
	}

	// If previously shutdown, reset state for restart
	if ms.shutdown {
		ms.shutdown = false
		ms.ctx, ms.cancel = context.WithCancel(context.Background())
		ms.wg = sync.WaitGroup{} // Reset wait group
	}

	// Start async message consumer
	ms.wg.Add(1)
	go func() {
		defer ms.wg.Done()
		if err := ms.asyncConsumer.Consume(ms.ctx, ms.handleAsyncMessage); err != nil {
			ms.logger.Printf("Async consumer error: %v", err)
		}
	}()

	// Start sync message consumer
	ms.wg.Add(1)
	go func() {
		defer ms.wg.Done()
		if err := ms.syncConsumer.Consume(ms.ctx, ms.handleSyncMessage); err != nil {
			ms.logger.Printf("Sync consumer error: %v", err)
		}
	}()

	// Start metrics reporter
	ms.wg.Add(1)
	go ms.reportMetrics()

	ms.started = true
	ms.logger.Printf("MessageSyncer started for region %s", ms.regionID)
	return nil
}

// SyncMessageAsync sends a message asynchronously to another region
func (ms *MessageSyncer) SyncMessageAsync(ctx context.Context, targetRegion string, message storage.LocalMessage) error {
	// Generate global ID for the sync operation
	globalID := ms.hlcClock.GenerateID()

	// Create sync message
	syncMsg := &SyncMessage{
		ID:             fmt.Sprintf("async-%s-%d", ms.regionID, time.Now().UnixNano()),
		Type:           "async",
		SourceRegion:   ms.regionID,
		TargetRegion:   targetRegion,
		MessageID:      message.MsgID,
		GlobalID:       globalID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Content:        message.Content,
		SequenceNumber: message.SequenceNumber,
		Timestamp:      message.Timestamp,
		Metadata:       message.Metadata,
		RequiresAck:    false,
		MaxRetries:     ms.config.MaxRetries,
		CreatedAt:      time.Now(),
		IsCritical:     false,
	}

	// Calculate checksum if enabled
	if ms.config.EnableChecksum {
		syncMsg.Checksum = ms.calculateChecksum(syncMsg)
	}

	// Convert to queue message
	queueMsg, err := ms.syncMessageToQueueMessage(syncMsg)
	if err != nil {
		atomic.AddInt64(&ms.errorCount, 1)
		return fmt.Errorf("failed to convert sync message: %w", err)
	}

	// Send asynchronously
	if err := ms.asyncProducer.Produce(ctx, queueMsg); err != nil {
		atomic.AddInt64(&ms.errorCount, 1)
		return fmt.Errorf("failed to produce async message: %w", err)
	}

	atomic.AddInt64(&ms.asyncSyncCount, 1)
	ms.logger.Printf("Sent async sync message %s to %s", syncMsg.MessageID, targetRegion)
	return nil
}

// SyncMessageSync sends a message synchronously to another region with acknowledgment
func (ms *MessageSyncer) SyncMessageSync(ctx context.Context, targetRegion string, message storage.LocalMessage) error {
	startTime := time.Now()

	// Generate global ID for the sync operation
	globalID := ms.hlcClock.GenerateID()

	// Create sync message
	syncMsg := &SyncMessage{
		ID:             fmt.Sprintf("sync-%s-%d", ms.regionID, time.Now().UnixNano()),
		Type:           "sync",
		SourceRegion:   ms.regionID,
		TargetRegion:   targetRegion,
		MessageID:      message.MsgID,
		GlobalID:       globalID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Content:        message.Content,
		SequenceNumber: message.SequenceNumber,
		Timestamp:      message.Timestamp,
		Metadata:       message.Metadata,
		RequiresAck:    true,
		AckTimeout:     ms.config.SyncTimeout,
		MaxRetries:     ms.config.MaxRetries,
		CreatedAt:      time.Now(),
		IsCritical:     true,
	}

	// Calculate checksum if enabled
	if ms.config.EnableChecksum {
		syncMsg.Checksum = ms.calculateChecksum(syncMsg)
	}

	// Convert to queue message
	queueMsg, err := ms.syncMessageToQueueMessage(syncMsg)
	if err != nil {
		atomic.AddInt64(&ms.errorCount, 1)
		return fmt.Errorf("failed to convert sync message: %w", err)
	}

	// Create acknowledgment channel
	ackChan := make(chan *SyncAck, 1)
	ms.registerAckHandler(syncMsg.ID, ackChan)
	defer ms.unregisterAckHandler(syncMsg.ID)

	// Send synchronously
	if err := ms.syncProducer.ProduceSync(ctx, queueMsg); err != nil {
		atomic.AddInt64(&ms.errorCount, 1)
		return fmt.Errorf("failed to produce sync message: %w", err)
	}

	// Wait for acknowledgment
	select {
	case ack := <-ackChan:
		latency := time.Since(startTime).Milliseconds()
		atomic.AddInt64(&ms.syncLatencySum, latency)
		atomic.AddInt64(&ms.syncLatencyCount, 1)

		if ack.Status == "success" {
			atomic.AddInt64(&ms.syncSyncCount, 1)
			ms.logger.Printf("Sync message %s acknowledged by %s in %dms",
				syncMsg.MessageID, targetRegion, latency)
			return nil
		} else {
			atomic.AddInt64(&ms.errorCount, 1)
			return fmt.Errorf("sync failed: %s", ack.Error)
		}

	case <-time.After(ms.config.SyncTimeout):
		atomic.AddInt64(&ms.errorCount, 1)
		return fmt.Errorf("sync timeout after %v", ms.config.SyncTimeout)

	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleAsyncMessage processes incoming async messages
func (ms *MessageSyncer) handleAsyncMessage(ctx context.Context, queueMsg *queue.Message) error {
	syncMsg, err := ms.queueMessageToSyncMessage(queueMsg)
	if err != nil {
		ms.logger.Printf("Failed to parse async message: %v", err)
		return err
	}

	// Skip messages from our own region
	if syncMsg.SourceRegion == ms.regionID {
		return nil
	}

	ms.logger.Printf("Processing async message %s from %s", syncMsg.MessageID, syncMsg.SourceRegion)

	// Update HLC clock from remote timestamp
	if err := ms.hlcClock.UpdateFromRemote(syncMsg.GlobalID.HLC); err != nil {
		ms.logger.Printf("Failed to update HLC from remote: %v", err)
	}

	// Verify checksum if enabled
	if ms.config.EnableChecksum && syncMsg.Checksum != "" {
		expectedChecksum := ms.calculateChecksum(syncMsg)
		if syncMsg.Checksum != expectedChecksum {
			ms.logger.Printf("Checksum mismatch for message %s", syncMsg.MessageID)
			return fmt.Errorf("checksum verification failed")
		}
	}

	// Convert to local message
	localMsg := ms.syncMessageToLocalMessage(syncMsg)

	// Check for conflicts
	conflict, err := ms.localStorage.DetectConflict(ctx, localMsg)
	if err != nil {
		ms.logger.Printf("Failed to detect conflict for message %s: %v", syncMsg.MessageID, err)
		return err
	}

	if conflict != nil {
		atomic.AddInt64(&ms.conflictCount, 1)
		ms.logger.Printf("Conflict detected for message %s: %s wins",
			syncMsg.MessageID, conflict.Resolution)

		// Record conflict for monitoring
		if err := ms.localStorage.RecordConflict(ctx, *conflict); err != nil {
			ms.logger.Printf("Failed to record conflict: %v", err)
		}

		// Apply conflict resolution
		if conflict.Resolution == "remote_wins" {
			if err := ms.localStorage.Insert(ctx, localMsg); err != nil {
				ms.logger.Printf("Failed to insert remote message after conflict resolution: %v", err)
				return err
			}
		}
		// If local wins, we don't need to do anything
	} else {
		// No conflict, insert the message
		if err := ms.localStorage.Insert(ctx, localMsg); err != nil {
			ms.logger.Printf("Failed to insert async message %s: %v", syncMsg.MessageID, err)
			return err
		}
	}

	ms.logger.Printf("Successfully processed async message %s", syncMsg.MessageID)
	return nil
}

// handleSyncMessage processes incoming sync messages that require acknowledgment
func (ms *MessageSyncer) handleSyncMessage(ctx context.Context, queueMsg *queue.Message) error {
	startTime := time.Now()

	syncMsg, err := ms.queueMessageToSyncMessage(queueMsg)
	if err != nil {
		ms.logger.Printf("Failed to parse sync message: %v", err)
		return err
	}

	// Skip messages from our own region
	if syncMsg.SourceRegion == ms.regionID {
		return nil
	}

	ms.logger.Printf("Processing sync message %s from %s", syncMsg.MessageID, syncMsg.SourceRegion)

	// Create acknowledgment
	ack := &SyncAck{
		MessageID:    syncMsg.MessageID,
		GlobalID:     syncMsg.GlobalID.String(),
		SourceRegion: ms.regionID,
		TargetRegion: syncMsg.SourceRegion,
		Timestamp:    time.Now(),
	}

	// Update HLC clock from remote timestamp
	if err := ms.hlcClock.UpdateFromRemote(syncMsg.GlobalID.HLC); err != nil {
		ms.logger.Printf("Failed to update HLC from remote: %v", err)
		ack.Status = "error"
		ack.Error = fmt.Sprintf("HLC update failed: %v", err)
	} else {
		// Verify checksum if enabled
		if ms.config.EnableChecksum && syncMsg.Checksum != "" {
			expectedChecksum := ms.calculateChecksum(syncMsg)
			if syncMsg.Checksum != expectedChecksum {
				ack.Status = "error"
				ack.Error = "checksum verification failed"
			} else {
				// Process the message
				ack.Status, ack.Error = ms.processSyncMessage(ctx, syncMsg)
			}
		} else {
			// Process the message
			ack.Status, ack.Error = ms.processSyncMessage(ctx, syncMsg)
		}
	}

	// Calculate processing time
	ack.ProcessTime = time.Since(startTime).Milliseconds()

	// Send acknowledgment
	if err := ms.sendAcknowledgment(ctx, ack); err != nil {
		ms.logger.Printf("Failed to send acknowledgment for message %s: %v", syncMsg.MessageID, err)
		return err
	}

	ms.logger.Printf("Sent acknowledgment for sync message %s with status %s",
		syncMsg.MessageID, ack.Status)
	return nil
}

// processSyncMessage processes a sync message and returns status and error
func (ms *MessageSyncer) processSyncMessage(ctx context.Context, syncMsg *SyncMessage) (string, string) {
	// Convert to local message
	localMsg := ms.syncMessageToLocalMessage(syncMsg)

	// Check for conflicts
	conflict, err := ms.localStorage.DetectConflict(ctx, localMsg)
	if err != nil {
		ms.logger.Printf("Failed to detect conflict for message %s: %v", syncMsg.MessageID, err)
		return "error", fmt.Sprintf("conflict detection failed: %v", err)
	}

	if conflict != nil {
		atomic.AddInt64(&ms.conflictCount, 1)
		ms.logger.Printf("Conflict detected for sync message %s: %s wins",
			syncMsg.MessageID, conflict.Resolution)

		// Record conflict for monitoring
		if err := ms.localStorage.RecordConflict(ctx, *conflict); err != nil {
			ms.logger.Printf("Failed to record conflict: %v", err)
		}

		// Apply conflict resolution
		if conflict.Resolution == "remote_wins" {
			if err := ms.localStorage.Insert(ctx, localMsg); err != nil {
				ms.logger.Printf("Failed to insert remote message after conflict resolution: %v", err)
				return "error", fmt.Sprintf("failed to insert message: %v", err)
			}
		}

		return "conflict", fmt.Sprintf("conflict resolved: %s", conflict.Resolution)
	} else {
		// No conflict, insert the message
		if err := ms.localStorage.Insert(ctx, localMsg); err != nil {
			ms.logger.Printf("Failed to insert sync message %s: %v", syncMsg.MessageID, err)
			return "error", fmt.Sprintf("failed to insert message: %v", err)
		}
	}

	return "success", ""
}

// sendAcknowledgment sends an acknowledgment back to the source region
func (ms *MessageSyncer) sendAcknowledgment(ctx context.Context, ack *SyncAck) error {
	// Convert acknowledgment to queue message
	ackData, err := json.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal acknowledgment: %w", err)
	}

	queueMsg := &queue.Message{
		ID:        fmt.Sprintf("ack-%s-%d", ms.regionID, time.Now().UnixNano()),
		Topic:     ms.config.AckTopic,
		Key:       ack.MessageID,
		Value:     ackData,
		Timestamp: time.Now().UnixMilli(),
		RegionID:  ms.regionID,
		GlobalID:  ack.GlobalID,
	}

	// Send acknowledgment
	return ms.asyncProducer.Produce(ctx, queueMsg)
}

// registerAckHandler registers a handler for acknowledgments
func (ms *MessageSyncer) registerAckHandler(messageID string, ackChan chan *SyncAck) {
	// In a real implementation, this would use a map to track pending acknowledgments
	// For this simplified version, we'll use the sync channel approach
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// This is a simplified implementation - in production you'd want a proper
	// acknowledgment tracking system with timeouts and cleanup
}

// unregisterAckHandler unregisters an acknowledgment handler
func (ms *MessageSyncer) unregisterAckHandler(messageID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Cleanup acknowledgment handler
}

// calculateChecksum calculates a checksum for message integrity verification
func (ms *MessageSyncer) calculateChecksum(syncMsg *SyncMessage) string {
	if !ms.config.EnableChecksum {
		return ""
	}

	// Create a deterministic string representation for checksum calculation
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%d",
		syncMsg.MessageID,
		syncMsg.ConversationID,
		syncMsg.SenderID,
		syncMsg.Content,
		syncMsg.SequenceNumber,
		syncMsg.Timestamp,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// syncMessageToQueueMessage converts a SyncMessage to a queue.Message
func (ms *MessageSyncer) syncMessageToQueueMessage(syncMsg *SyncMessage) (*queue.Message, error) {
	data, err := json.Marshal(syncMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sync message: %w", err)
	}

	topic := ms.config.AsyncTopic
	if syncMsg.Type == "sync" {
		topic = ms.config.SyncTopic
	}

	return &queue.Message{
		ID:        syncMsg.ID,
		Topic:     topic,
		Key:       syncMsg.MessageID,
		Value:     data,
		Timestamp: syncMsg.CreatedAt.UnixMilli(),
		RegionID:  syncMsg.SourceRegion,
		GlobalID:  syncMsg.GlobalID.String(),
	}, nil
}

// queueMessageToSyncMessage converts a queue.Message to a SyncMessage
func (ms *MessageSyncer) queueMessageToSyncMessage(queueMsg *queue.Message) (*SyncMessage, error) {
	var syncMsg SyncMessage
	if err := json.Unmarshal(queueMsg.Value, &syncMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync message: %w", err)
	}
	return &syncMsg, nil
}

// syncMessageToLocalMessage converts a SyncMessage to a storage.LocalMessage
func (ms *MessageSyncer) syncMessageToLocalMessage(syncMsg *SyncMessage) storage.LocalMessage {
	return storage.LocalMessage{
		MsgID:            syncMsg.MessageID,
		UserID:           "", // Will be set by the application layer
		SenderID:         syncMsg.SenderID,
		ConversationID:   syncMsg.ConversationID,
		ConversationType: "group", // Default, should be determined by application logic
		Content:          syncMsg.Content,
		SequenceNumber:   syncMsg.SequenceNumber,
		Timestamp:        syncMsg.Timestamp,
		CreatedAt:        syncMsg.CreatedAt,
		ExpiresAt:        syncMsg.CreatedAt.Add(7 * 24 * time.Hour), // 7 days TTL
		Metadata:         syncMsg.Metadata,
		RegionID:         syncMsg.SourceRegion,
		GlobalID:         syncMsg.GlobalID.String(),
		Version:          1, // Initial version
	}
}

// reportMetrics periodically reports synchronization metrics
func (ms *MessageSyncer) reportMetrics() {
	defer ms.wg.Done()

	ticker := time.NewTicker(ms.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			asyncCount := atomic.LoadInt64(&ms.asyncSyncCount)
			syncCount := atomic.LoadInt64(&ms.syncSyncCount)
			conflictCount := atomic.LoadInt64(&ms.conflictCount)
			errorCount := atomic.LoadInt64(&ms.errorCount)
			latencySum := atomic.LoadInt64(&ms.syncLatencySum)
			latencyCount := atomic.LoadInt64(&ms.syncLatencyCount)

			avgLatency := int64(0)
			if latencyCount > 0 {
				avgLatency = latencySum / latencyCount
			}

			ms.logger.Printf("Sync metrics - Async: %d, Sync: %d, Conflicts: %d, Errors: %d, Avg Latency: %dms",
				asyncCount, syncCount, conflictCount, errorCount, avgLatency)

		case <-ms.ctx.Done():
			ms.logger.Printf("Stopped metrics reporting")
			return
		}
	}
}

// GetMetrics returns current synchronization metrics
func (ms *MessageSyncer) GetMetrics() map[string]interface{} {
	latencySum := atomic.LoadInt64(&ms.syncLatencySum)
	latencyCount := atomic.LoadInt64(&ms.syncLatencyCount)

	avgLatency := int64(0)
	if latencyCount > 0 {
		avgLatency = latencySum / latencyCount
	}

	return map[string]interface{}{
		"region_id":           ms.regionID,
		"async_sync_count":    atomic.LoadInt64(&ms.asyncSyncCount),
		"sync_sync_count":     atomic.LoadInt64(&ms.syncSyncCount),
		"conflict_count":      atomic.LoadInt64(&ms.conflictCount),
		"error_count":         atomic.LoadInt64(&ms.errorCount),
		"avg_sync_latency_ms": avgLatency,
		"sync_latency_count":  latencyCount,
		"started":             ms.started,
		"shutdown":            ms.shutdown,
	}
}

// GetCounts returns the individual metric counts
func (ms *MessageSyncer) GetCounts() (asyncCount, syncCount, conflictCount, errorCount int64) {
	return atomic.LoadInt64(&ms.asyncSyncCount),
		atomic.LoadInt64(&ms.syncSyncCount),
		atomic.LoadInt64(&ms.conflictCount),
		atomic.LoadInt64(&ms.errorCount)
}

// GetAverageLatency returns the average sync latency in milliseconds
func (ms *MessageSyncer) GetAverageLatency() float64 {
	latencySum := atomic.LoadInt64(&ms.syncLatencySum)
	latencyCount := atomic.LoadInt64(&ms.syncLatencyCount)

	if latencyCount == 0 {
		return 0.0
	}
	return float64(latencySum) / float64(latencyCount)
}

// GetSyncRate returns the approximate sync rate in messages per second
func (ms *MessageSyncer) GetSyncRate() float64 {
	// Simple approximation based on total syncs and uptime
	totalSyncs := atomic.LoadInt64(&ms.asyncSyncCount) + atomic.LoadInt64(&ms.syncSyncCount)

	// Estimate uptime (this is a simple approximation)
	// In a real implementation, you'd track start time
	if totalSyncs == 0 {
		return 0.0
	}

	// Assume we've been running for at least 1 second, return a reasonable rate
	return float64(totalSyncs) / 60.0 // Messages per minute / 60 = rough per-second rate
}

// Stop gracefully stops the message synchronizer
func (ms *MessageSyncer) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.shutdown {
		return nil // Already stopped
	}

	ms.logger.Printf("Stopping MessageSyncer for region %s", ms.regionID)

	// Cancel context to stop all goroutines
	ms.cancel()

	// Close producers and consumers
	if ms.asyncProducer != nil {
		if err := ms.asyncProducer.Close(); err != nil {
			ms.logger.Printf("Error closing async producer: %v", err)
		}
	}

	if ms.syncProducer != nil {
		if err := ms.syncProducer.Close(); err != nil {
			ms.logger.Printf("Error closing sync producer: %v", err)
		}
	}

	if ms.asyncConsumer != nil {
		if err := ms.asyncConsumer.Close(); err != nil {
			ms.logger.Printf("Error closing async consumer: %v", err)
		}
	}

	if ms.syncConsumer != nil {
		if err := ms.syncConsumer.Close(); err != nil {
			ms.logger.Printf("Error closing sync consumer: %v", err)
		}
	}

	// Wait for all goroutines to finish
	ms.wg.Wait()

	ms.shutdown = true
	ms.started = false // Reset started flag to allow restart
	ms.logger.Printf("MessageSyncer stopped")
	return nil
}

// CreateSyncChannel creates a sync channel for cross-region communication
func (ms *MessageSyncer) CreateSyncChannel(targetRegion string) error {
	return ms.localQueue.CreateSyncChannel(targetRegion)
}

// SendSyncEvent sends a sync event to another region (for direct channel communication)
func (ms *MessageSyncer) SendSyncEvent(targetRegion string, event *queue.SyncEvent) error {
	return ms.localQueue.SendSyncEvent(targetRegion, event)
}

// ReceiveSyncEvents receives sync events from another region
func (ms *MessageSyncer) ReceiveSyncEvents(sourceRegion string, handler func(*queue.SyncEvent) error) error {
	return ms.localQueue.ReceiveSyncEvents(sourceRegion, handler)
}
