package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
)

// OfflineStorer defines the interface for offline message storage
type OfflineStorer interface {
	BatchInsert(ctx context.Context, messages []storage.OfflineMessage) error
	Close() error
}

// DedupChecker defines the interface for deduplication checking
type DedupChecker interface {
	CheckDuplicate(ctx context.Context, msgID string) (bool, error)
	MarkProcessed(ctx context.Context, msgID string) error
}

// OfflineMessageEvent represents a message from Kafka offline_msg topic
// Matches the protobuf schema from design.md
type OfflineMessageEvent struct {
	MsgID            string `json:"msg_id"`
	UserID           string `json:"user_id"`
	SenderID         string `json:"sender_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"` // "private" or "group"
	Content          string `json:"content"`
	SequenceNumber   int64  `json:"sequence_number"`
	Timestamp        int64  `json:"timestamp"`
}

// OfflineWorker consumes messages from Kafka and persists them to database
// Requirements: 4.2, 4.6 - Consume from offline_msg topic and batch persist
type OfflineWorker struct {
	consumerGroup sarama.ConsumerGroup
	store         OfflineStorer
	dedupService  DedupChecker
	config        WorkerConfig

	// Metrics
	mu                   sync.RWMutex
	messagesProcessed    int64
	messagesDeduplicated int64
	messagesPersisted    int64
	batchWrites          int64
	errors               int64

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// WorkerConfig holds configuration for the offline worker
type WorkerConfig struct {
	KafkaBrokers  []string
	ConsumerGroup string
	Topic         string
	BatchSize     int
	BatchTimeout  time.Duration
	MaxRetries    int
	RetryBackoff  []time.Duration
	MessageTTL    time.Duration // Default: 7 days
}

// NewOfflineWorker creates a new offline worker
func NewOfflineWorker(
	config WorkerConfig,
	store OfflineStorer,
	dedupService DedupChecker,
) (*OfflineWorker, error) {
	// Set defaults
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 5 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 5
	}
	if len(config.RetryBackoff) == 0 {
		config.RetryBackoff = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	}
	if config.MessageTTL == 0 {
		config.MessageTTL = 7 * 24 * time.Hour // 7 days
	}

	// Configure Sarama
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false // Manual commit after processing
	saramaConfig.Consumer.Return.Errors = true

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(config.KafkaBrokers, config.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &OfflineWorker{
		consumerGroup: consumerGroup,
		store:         store,
		dedupService:  dedupService,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start starts the offline worker
// Requirements: 4.2 - Subscribe to offline_msg topic
func (w *OfflineWorker) Start() error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		handler := &consumerGroupHandler{
			worker: w,
		}

		for {
			// Check if context is cancelled
			if w.ctx.Err() != nil {
				log.Println("Worker context cancelled, stopping consumer")
				return
			}

			// Consume messages
			// This will block until rebalance or error
			err := w.consumerGroup.Consume(w.ctx, []string{w.config.Topic}, handler)
			if err != nil {
				log.Printf("Error consuming messages: %v", err)
				w.incrementErrors()

				// Wait before retrying
				select {
				case <-w.ctx.Done():
					return
				case <-time.After(5 * time.Second):
					continue
				}
			}
		}
	}()

	// Monitor errors
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case <-w.ctx.Done():
				return
			case err := <-w.consumerGroup.Errors():
				if err != nil {
					log.Printf("Consumer group error: %v", err)
					w.incrementErrors()
				}
			}
		}
	}()

	log.Printf("Offline worker started, consuming from topic: %s", w.config.Topic)
	return nil
}

// Stop gracefully stops the offline worker
func (w *OfflineWorker) Stop() error {
	log.Println("Stopping offline worker...")

	// Cancel context to stop consumption
	w.cancel()

	// Wait for goroutines to finish
	w.wg.Wait()

	// Close consumer group
	if err := w.consumerGroup.Close(); err != nil {
		return fmt.Errorf("failed to close consumer group: %w", err)
	}

	log.Println("Offline worker stopped")
	return nil
}

// GetStats returns worker statistics
func (w *OfflineWorker) GetStats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	avgBatchSize := 0.0
	if w.batchWrites > 0 {
		avgBatchSize = float64(w.messagesPersisted) / float64(w.batchWrites)
	}

	return WorkerStats{
		MessagesProcessed:    w.messagesProcessed,
		MessagesDeduplicated: w.messagesDeduplicated,
		MessagesPersisted:    w.messagesPersisted,
		BatchWrites:          w.batchWrites,
		Errors:               w.errors,
		AvgBatchSize:         avgBatchSize,
	}
}

// WorkerStats holds worker statistics
type WorkerStats struct {
	MessagesProcessed    int64
	MessagesDeduplicated int64
	MessagesPersisted    int64
	BatchWrites          int64
	Errors               int64
	AvgBatchSize         float64
}

// incrementProcessed increments the processed message counter
func (w *OfflineWorker) incrementProcessed() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messagesProcessed++
}

// incrementDeduplicated increments the deduplicated message counter
func (w *OfflineWorker) incrementDeduplicated() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messagesDeduplicated++
}

// incrementPersisted increments the persisted message counter
func (w *OfflineWorker) incrementPersisted(count int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messagesPersisted += count
}

// incrementBatchWrites increments the batch write counter
func (w *OfflineWorker) incrementBatchWrites() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.batchWrites++
}

// incrementErrors increments the error counter
func (w *OfflineWorker) incrementErrors() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errors++
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	worker *OfflineWorker
}

// Setup is called when a new session is established
// Requirements: Handle consumer rebalancing
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	log.Printf("Consumer group session started, member ID: %s, generation ID: %d",
		session.MemberID(), session.GenerationID())
	return nil
}

// Cleanup is called when a session is ending
func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	log.Printf("Consumer group session ended, member ID: %s", session.MemberID())
	return nil
}

// ConsumeClaim processes messages from a partition
// Requirements: 4.6 - Batch processing with manual offset commit
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	batch := make([]OfflineMessageEvent, 0, h.worker.config.BatchSize)
	batchTimer := time.NewTimer(h.worker.config.BatchTimeout)
	defer batchTimer.Stop()

	for {
		select {
		case <-session.Context().Done():
			// Session context cancelled, flush remaining batch
			if len(batch) > 0 {
				if err := h.processBatch(session.Context(), batch, session, claim); err != nil {
					log.Printf("Error processing final batch: %v", err)
				}
			}
			return nil

		case <-batchTimer.C:
			// Batch timeout reached, process accumulated messages
			if len(batch) > 0 {
				if err := h.processBatch(session.Context(), batch, session, claim); err != nil {
					log.Printf("Error processing batch on timeout: %v", err)
					h.worker.incrementErrors()
					// Don't return error, continue processing
				}
				batch = batch[:0] // Clear batch
			}
			batchTimer.Reset(h.worker.config.BatchTimeout)

		case message := <-claim.Messages():
			if message == nil {
				continue
			}

			// Parse message
			var event OfflineMessageEvent
			if err := json.Unmarshal(message.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				h.worker.incrementErrors()
				// Mark message as processed even if invalid
				session.MarkMessage(message, "")
				continue
			}

			h.worker.incrementProcessed()

			// Add to batch
			batch = append(batch, event)

			// Process batch if full
			if len(batch) >= h.worker.config.BatchSize {
				if err := h.processBatch(session.Context(), batch, session, claim); err != nil {
					log.Printf("Error processing full batch: %v", err)
					h.worker.incrementErrors()
					// Don't return error, continue processing
				}
				batch = batch[:0] // Clear batch
				batchTimer.Reset(h.worker.config.BatchTimeout)
			}
		}
	}
}

// processBatch processes a batch of messages
// Requirements: 3.9, 3.10, 3.11 - Deduplication before database write
// Requirements: 4.6 - Batch insert with transaction
func (h *consumerGroupHandler) processBatch(
	ctx context.Context,
	events []OfflineMessageEvent,
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	if len(events) == 0 {
		return nil
	}

	// Step 1: Check for duplicates
	// Requirements: 3.9, 3.11 - Check Redis dedup set before database write
	uniqueEvents := make([]OfflineMessageEvent, 0, len(events))
	for _, event := range events {
		isDuplicate, err := h.worker.dedupService.CheckDuplicate(ctx, event.MsgID)
		if err != nil {
			log.Printf("Error checking duplicate for msg_id %s: %v", event.MsgID, err)
			// Continue processing, don't fail entire batch
			uniqueEvents = append(uniqueEvents, event)
			continue
		}

		if isDuplicate {
			// Requirements: 3.10 - Skip if msg_id already exists
			log.Printf("Duplicate message detected: %s, skipping", event.MsgID)
			h.worker.incrementDeduplicated()
			continue
		}

		uniqueEvents = append(uniqueEvents, event)
	}

	if len(uniqueEvents) == 0 {
		// All messages were duplicates, commit offset
		session.Commit()
		return nil
	}

	// Step 2: Convert to storage format
	messages := make([]storage.OfflineMessage, len(uniqueEvents))
	for i, event := range uniqueEvents {
		messages[i] = storage.OfflineMessage{
			MsgID:            event.MsgID,
			UserID:           event.UserID,
			SenderID:         event.SenderID,
			ConversationID:   event.ConversationID,
			ConversationType: event.ConversationType,
			Content:          event.Content,
			SequenceNumber:   event.SequenceNumber,
			Timestamp:        event.Timestamp,
			ExpiresAt:        time.Now().Add(h.worker.config.MessageTTL),
		}
	}

	// Step 3: Batch insert to database with retry
	// Requirements: 4.6 - Single transaction for batch insert
	var lastErr error
	for attempt := 0; attempt <= h.worker.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			backoffIdx := attempt - 1
			if backoffIdx >= len(h.worker.config.RetryBackoff) {
				backoffIdx = len(h.worker.config.RetryBackoff) - 1
			}
			time.Sleep(h.worker.config.RetryBackoff[backoffIdx])
			log.Printf("Retrying batch insert, attempt %d/%d", attempt+1, h.worker.config.MaxRetries)
		}

		err := h.worker.store.BatchInsert(ctx, messages)
		if err == nil {
			// Success!
			break
		}

		lastErr = err
		log.Printf("Error inserting batch (attempt %d/%d): %v", attempt+1, h.worker.config.MaxRetries+1, err)
	}

	if lastErr != nil {
		// Requirements: Rollback and retry on failure
		// Kafka will redeliver messages since we don't commit offset
		return fmt.Errorf("failed to insert batch after %d retries: %w", h.worker.config.MaxRetries, lastErr)
	}

	// Step 4: Mark messages as processed in dedup set
	// Requirements: 3.11 - Add to dedup set after successful write
	for _, event := range uniqueEvents {
		if err := h.worker.dedupService.MarkProcessed(ctx, event.MsgID); err != nil {
			log.Printf("Error marking msg_id %s as processed: %v", event.MsgID, err)
			// Don't fail the batch, dedup is best-effort
		}
	}

	// Step 5: Commit Kafka offset
	// Requirements: Manual offset commit after processing
	session.Commit()

	// Update metrics
	h.worker.incrementPersisted(int64(len(uniqueEvents)))
	h.worker.incrementBatchWrites()

	log.Printf("Successfully processed batch of %d messages (%d unique, %d duplicates)",
		len(events), len(uniqueEvents), len(events)-len(uniqueEvents))

	return nil
}
