package connpool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

// BatchProcessor handles batch operations for cross-region scenarios
type BatchProcessor struct {
	config BatchProcessorConfig

	// Batch buffers
	messageBatchMu sync.Mutex
	messageBatch   []BatchMessage
	messageTicker  *time.Ticker

	offlineBatchMu sync.Mutex
	offlineBatch   []BatchOfflineMessage
	offlineTicker  *time.Ticker

	reconcileBatchMu sync.Mutex
	reconcileBatch   []BatchReconcileItem
	reconcileTicker  *time.Ticker

	// Metrics
	mu                    sync.RWMutex
	totalMessagesBatched  int64
	totalOfflineBatched   int64
	totalReconcileBatched int64
	totalBatchesFlushed   int64
	totalBatchErrors      int64
	avgBatchSize          float64
	avgFlushDuration      time.Duration
	lastFlushTime         time.Time
	batchSizeSum          int64
	batchSizeCount        int64
	flushDurationSum      int64
	flushDurationCount    int64

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BatchProcessorConfig holds configuration for batch processing
type BatchProcessorConfig struct {
	// Message batch configuration
	MessageBatchSize     int           // Maximum messages per batch
	MessageFlushInterval time.Duration // Maximum time before flushing
	MessageMaxRetries    int           // Maximum retry attempts

	// Offline message batch configuration
	OfflineBatchSize     int           // Maximum offline messages per batch
	OfflineFlushInterval time.Duration // Maximum time before flushing
	OfflineMaxRetries    int           // Maximum retry attempts

	// Reconcile batch configuration
	ReconcileBatchSize     int           // Maximum reconcile items per batch
	ReconcileFlushInterval time.Duration // Maximum time before flushing
	ReconcileMaxRetries    int           // Maximum retry attempts

	// Performance tuning
	EnableCompression    bool // Enable message compression
	EnablePipelining     bool // Enable Redis pipelining
	MaxConcurrentBatches int  // Maximum concurrent batch operations

	// Monitoring
	EnableMetrics      bool          // Enable metrics collection
	MetricsInterval    time.Duration // Metrics reporting interval
	EnableHealthChecks bool          // Enable health checks
}

// BatchMessage represents a message to be batched for synchronization
type BatchMessage struct {
	ID             string
	RegionID       string
	GlobalID       string
	ConversationID string
	SenderID       string
	Content        string
	Timestamp      int64
	Priority       int // Higher priority = processed first
	RetryCount     int
	CreatedAt      time.Time
}

// BatchOfflineMessage represents an offline message to be batched
type BatchOfflineMessage struct {
	ID             string
	UserID         string
	SenderID       string
	ConversationID string
	Content        string
	SequenceNumber int64
	Timestamp      int64
	ExpiresAt      time.Time
	RegionID       string
	GlobalID       string
	RetryCount     int
	CreatedAt      time.Time
}

// BatchReconcileItem represents a reconciliation item to be batched
type BatchReconcileItem struct {
	GlobalID     string
	Operation    string // "add", "update", "delete"
	SourceRegion string
	TargetRegion string
	MessageData  interface{}
	Priority     int
	RetryCount   int
	CreatedAt    time.Time
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	TotalItems    int
	SuccessCount  int
	FailureCount  int
	Duration      time.Duration
	BatchSize     int
	Errors        []error
	RetryRequired []interface{}
}

// BatchProcessorMetrics holds batch processor metrics
type BatchProcessorMetrics struct {
	TotalMessagesBatched  int64
	TotalOfflineBatched   int64
	TotalReconcileBatched int64
	TotalBatchesFlushed   int64
	TotalBatchErrors      int64
	AvgBatchSize          float64
	AvgFlushDuration      time.Duration
	LastFlushTime         time.Time
	CurrentMessageBatch   int
	CurrentOfflineBatch   int
	CurrentReconcileBatch int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config BatchProcessorConfig) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		config:         config,
		messageBatch:   make([]BatchMessage, 0, config.MessageBatchSize),
		offlineBatch:   make([]BatchOfflineMessage, 0, config.OfflineBatchSize),
		reconcileBatch: make([]BatchReconcileItem, 0, config.ReconcileBatchSize),
		ctx:            ctx,
		cancel:         cancel,
	}

	return bp
}

// Start starts the batch processor
func (bp *BatchProcessor) Start() error {
	// Start message batch ticker
	bp.messageTicker = time.NewTicker(bp.config.MessageFlushInterval)
	bp.wg.Add(1)
	go bp.messageFlushLoop()

	// Start offline batch ticker
	bp.offlineTicker = time.NewTicker(bp.config.OfflineFlushInterval)
	bp.wg.Add(1)
	go bp.offlineFlushLoop()

	// Start reconcile batch ticker
	bp.reconcileTicker = time.NewTicker(bp.config.ReconcileFlushInterval)
	bp.wg.Add(1)
	go bp.reconcileFlushLoop()

	// Start metrics reporter if enabled
	if bp.config.EnableMetrics {
		bp.wg.Add(1)
		go bp.metricsLoop()
	}

	return nil
}

// Stop stops the batch processor and flushes remaining batches
func (bp *BatchProcessor) Stop() error {
	// Cancel context
	bp.cancel()

	// Stop tickers
	if bp.messageTicker != nil {
		bp.messageTicker.Stop()
	}
	if bp.offlineTicker != nil {
		bp.offlineTicker.Stop()
	}
	if bp.reconcileTicker != nil {
		bp.reconcileTicker.Stop()
	}

	// Flush remaining batches
	bp.flushMessageBatch(context.Background())
	bp.flushOfflineBatch(context.Background())
	bp.flushReconcileBatch(context.Background())

	// Wait for all goroutines to finish
	bp.wg.Wait()

	return nil
}

// AddMessage adds a message to the batch
func (bp *BatchProcessor) AddMessage(msg BatchMessage) error {
	bp.messageBatchMu.Lock()
	defer bp.messageBatchMu.Unlock()

	bp.messageBatch = append(bp.messageBatch, msg)
	atomic.AddInt64(&bp.totalMessagesBatched, 1)

	// Flush if batch is full
	if len(bp.messageBatch) >= bp.config.MessageBatchSize {
		go bp.flushMessageBatch(bp.ctx)
	}

	return nil
}

// AddOfflineMessage adds an offline message to the batch
func (bp *BatchProcessor) AddOfflineMessage(msg BatchOfflineMessage) error {
	bp.offlineBatchMu.Lock()
	defer bp.offlineBatchMu.Unlock()

	bp.offlineBatch = append(bp.offlineBatch, msg)
	atomic.AddInt64(&bp.totalOfflineBatched, 1)

	// Flush if batch is full
	if len(bp.offlineBatch) >= bp.config.OfflineBatchSize {
		go bp.flushOfflineBatch(bp.ctx)
	}

	return nil
}

// AddReconcileItem adds a reconciliation item to the batch
func (bp *BatchProcessor) AddReconcileItem(item BatchReconcileItem) error {
	bp.reconcileBatchMu.Lock()
	defer bp.reconcileBatchMu.Unlock()

	bp.reconcileBatch = append(bp.reconcileBatch, item)
	atomic.AddInt64(&bp.totalReconcileBatched, 1)

	// Flush if batch is full
	if len(bp.reconcileBatch) >= bp.config.ReconcileBatchSize {
		go bp.flushReconcileBatch(bp.ctx)
	}

	return nil
}

// messageFlushLoop periodically flushes message batches
func (bp *BatchProcessor) messageFlushLoop() {
	defer bp.wg.Done()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-bp.messageTicker.C:
			bp.flushMessageBatch(bp.ctx)
		}
	}
}

// offlineFlushLoop periodically flushes offline message batches
func (bp *BatchProcessor) offlineFlushLoop() {
	defer bp.wg.Done()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-bp.offlineTicker.C:
			bp.flushOfflineBatch(bp.ctx)
		}
	}
}

// reconcileFlushLoop periodically flushes reconciliation batches
func (bp *BatchProcessor) reconcileFlushLoop() {
	defer bp.wg.Done()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-bp.reconcileTicker.C:
			bp.flushReconcileBatch(bp.ctx)
		}
	}
}

// flushMessageBatch flushes the current message batch
func (bp *BatchProcessor) flushMessageBatch(ctx context.Context) {
	bp.messageBatchMu.Lock()
	if len(bp.messageBatch) == 0 {
		bp.messageBatchMu.Unlock()
		return
	}

	// Take ownership of current batch
	batch := bp.messageBatch
	bp.messageBatch = make([]BatchMessage, 0, bp.config.MessageBatchSize)
	bp.messageBatchMu.Unlock()

	startTime := time.Now()

	// Sort by priority (higher priority first)
	bp.sortMessagesByPriority(batch)

	// Process batch
	result := bp.processMessageBatch(ctx, batch)

	// Update metrics
	duration := time.Since(startTime)
	bp.updateMetrics(len(batch), result.SuccessCount, result.FailureCount, duration)

	// Handle retries
	if len(result.RetryRequired) > 0 {
		bp.handleMessageRetries(result.RetryRequired)
	}
}

// flushOfflineBatch flushes the current offline message batch
func (bp *BatchProcessor) flushOfflineBatch(ctx context.Context) {
	bp.offlineBatchMu.Lock()
	if len(bp.offlineBatch) == 0 {
		bp.offlineBatchMu.Unlock()
		return
	}

	// Take ownership of current batch
	batch := bp.offlineBatch
	bp.offlineBatch = make([]BatchOfflineMessage, 0, bp.config.OfflineBatchSize)
	bp.offlineBatchMu.Unlock()

	startTime := time.Now()

	// Process batch
	result := bp.processOfflineBatch(ctx, batch)

	// Update metrics
	duration := time.Since(startTime)
	bp.updateMetrics(len(batch), result.SuccessCount, result.FailureCount, duration)

	// Handle retries
	if len(result.RetryRequired) > 0 {
		bp.handleOfflineRetries(result.RetryRequired)
	}
}

// flushReconcileBatch flushes the current reconciliation batch
func (bp *BatchProcessor) flushReconcileBatch(ctx context.Context) {
	bp.reconcileBatchMu.Lock()
	if len(bp.reconcileBatch) == 0 {
		bp.reconcileBatchMu.Unlock()
		return
	}

	// Take ownership of current batch
	batch := bp.reconcileBatch
	bp.reconcileBatch = make([]BatchReconcileItem, 0, bp.config.ReconcileBatchSize)
	bp.reconcileBatchMu.Unlock()

	startTime := time.Now()

	// Sort by priority (higher priority first)
	bp.sortReconcileByPriority(batch)

	// Process batch
	result := bp.processReconcileBatch(ctx, batch)

	// Update metrics
	duration := time.Since(startTime)
	bp.updateMetrics(len(batch), result.SuccessCount, result.FailureCount, duration)

	// Handle retries
	if len(result.RetryRequired) > 0 {
		bp.handleReconcileRetries(result.RetryRequired)
	}
}

// processMessageBatch processes a batch of messages
func (bp *BatchProcessor) processMessageBatch(ctx context.Context, batch []BatchMessage) BatchResult {
	result := BatchResult{
		TotalItems: len(batch),
		BatchSize:  len(batch),
		Errors:     make([]error, 0),
	}

	// In a real implementation, this would:
	// 1. Send messages to Kafka in batch
	// 2. Use Kafka producer batching
	// 3. Handle compression if enabled
	// 4. Track success/failure per message

	// Simulate batch processing
	for _, msg := range batch {
		// Simulate processing
		if msg.RetryCount < bp.config.MessageMaxRetries {
			result.SuccessCount++
		} else {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("max retries exceeded for message %s", msg.ID))
			result.RetryRequired = append(result.RetryRequired, msg)
		}
	}

	return result
}

// processOfflineBatch processes a batch of offline messages
func (bp *BatchProcessor) processOfflineBatch(ctx context.Context, batch []BatchOfflineMessage) BatchResult {
	result := BatchResult{
		TotalItems: len(batch),
		BatchSize:  len(batch),
		Errors:     make([]error, 0),
	}

	// In a real implementation, this would:
	// 1. Use database batch insert (INSERT INTO ... VALUES (...), (...), ...)
	// 2. Use prepared statements for efficiency
	// 3. Handle transaction rollback on errors
	// 4. Use connection pooling

	// Simulate batch processing
	for _, msg := range batch {
		// Simulate processing
		if msg.RetryCount < bp.config.OfflineMaxRetries {
			result.SuccessCount++
		} else {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("max retries exceeded for offline message %s", msg.ID))
			result.RetryRequired = append(result.RetryRequired, msg)
		}
	}

	return result
}

// processReconcileBatch processes a batch of reconciliation items
func (bp *BatchProcessor) processReconcileBatch(ctx context.Context, batch []BatchReconcileItem) BatchResult {
	result := BatchResult{
		TotalItems: len(batch),
		BatchSize:  len(batch),
		Errors:     make([]error, 0),
	}

	// In a real implementation, this would:
	// 1. Group operations by type (add/update/delete)
	// 2. Use batch operations for each type
	// 3. Handle conflicts and retries
	// 4. Update reconciliation status

	// Simulate batch processing
	for _, item := range batch {
		// Simulate processing
		if item.RetryCount < bp.config.ReconcileMaxRetries {
			result.SuccessCount++
		} else {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("max retries exceeded for reconcile item %s", item.GlobalID))
			result.RetryRequired = append(result.RetryRequired, item)
		}
	}

	return result
}

// sortMessagesByPriority sorts messages by priority (higher first)
func (bp *BatchProcessor) sortMessagesByPriority(batch []BatchMessage) {
	// Simple bubble sort for small batches
	// In production, use sort.Slice
	for i := 0; i < len(batch)-1; i++ {
		for j := 0; j < len(batch)-i-1; j++ {
			if batch[j].Priority < batch[j+1].Priority {
				batch[j], batch[j+1] = batch[j+1], batch[j]
			}
		}
	}
}

// sortReconcileByPriority sorts reconcile items by priority (higher first)
func (bp *BatchProcessor) sortReconcileByPriority(batch []BatchReconcileItem) {
	// Simple bubble sort for small batches
	// In production, use sort.Slice
	for i := 0; i < len(batch)-1; i++ {
		for j := 0; j < len(batch)-i-1; j++ {
			if batch[j].Priority < batch[j+1].Priority {
				batch[j], batch[j+1] = batch[j+1], batch[j]
			}
		}
	}
}

// handleMessageRetries handles retry logic for failed messages
func (bp *BatchProcessor) handleMessageRetries(retryItems []interface{}) {
	for _, item := range retryItems {
		if msg, ok := item.(BatchMessage); ok {
			msg.RetryCount++
			if msg.RetryCount < bp.config.MessageMaxRetries {
				// Re-add to batch
				bp.AddMessage(msg)
			}
		}
	}
}

// handleOfflineRetries handles retry logic for failed offline messages
func (bp *BatchProcessor) handleOfflineRetries(retryItems []interface{}) {
	for _, item := range retryItems {
		if msg, ok := item.(BatchOfflineMessage); ok {
			msg.RetryCount++
			if msg.RetryCount < bp.config.OfflineMaxRetries {
				// Re-add to batch
				bp.AddOfflineMessage(msg)
			}
		}
	}
}

// handleReconcileRetries handles retry logic for failed reconcile items
func (bp *BatchProcessor) handleReconcileRetries(retryItems []interface{}) {
	for _, item := range retryItems {
		if reconcileItem, ok := item.(BatchReconcileItem); ok {
			reconcileItem.RetryCount++
			if reconcileItem.RetryCount < bp.config.ReconcileMaxRetries {
				// Re-add to batch
				bp.AddReconcileItem(reconcileItem)
			}
		}
	}
}

// updateMetrics updates batch processing metrics
func (bp *BatchProcessor) updateMetrics(batchSize, successCount, failureCount int, duration time.Duration) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	atomic.AddInt64(&bp.totalBatchesFlushed, 1)
	if failureCount > 0 {
		atomic.AddInt64(&bp.totalBatchErrors, 1)
	}

	// Update batch size metrics
	atomic.AddInt64(&bp.batchSizeSum, int64(batchSize))
	atomic.AddInt64(&bp.batchSizeCount, 1)
	bp.avgBatchSize = float64(bp.batchSizeSum) / float64(bp.batchSizeCount)

	// Update flush duration metrics
	atomic.AddInt64(&bp.flushDurationSum, duration.Milliseconds())
	atomic.AddInt64(&bp.flushDurationCount, 1)
	bp.avgFlushDuration = time.Duration(bp.flushDurationSum/bp.flushDurationCount) * time.Millisecond

	bp.lastFlushTime = time.Now()
}

// metricsLoop periodically reports metrics
func (bp *BatchProcessor) metricsLoop() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-ticker.C:
			metrics := bp.GetMetrics()
			// In production, this would send metrics to monitoring system
			_ = metrics
		}
	}
}

// GetMetrics returns current batch processor metrics
func (bp *BatchProcessor) GetMetrics() BatchProcessorMetrics {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	bp.messageBatchMu.Lock()
	currentMessageBatch := len(bp.messageBatch)
	bp.messageBatchMu.Unlock()

	bp.offlineBatchMu.Lock()
	currentOfflineBatch := len(bp.offlineBatch)
	bp.offlineBatchMu.Unlock()

	bp.reconcileBatchMu.Lock()
	currentReconcileBatch := len(bp.reconcileBatch)
	bp.reconcileBatchMu.Unlock()

	return BatchProcessorMetrics{
		TotalMessagesBatched:  atomic.LoadInt64(&bp.totalMessagesBatched),
		TotalOfflineBatched:   atomic.LoadInt64(&bp.totalOfflineBatched),
		TotalReconcileBatched: atomic.LoadInt64(&bp.totalReconcileBatched),
		TotalBatchesFlushed:   atomic.LoadInt64(&bp.totalBatchesFlushed),
		TotalBatchErrors:      atomic.LoadInt64(&bp.totalBatchErrors),
		AvgBatchSize:          bp.avgBatchSize,
		AvgFlushDuration:      bp.avgFlushDuration,
		LastFlushTime:         bp.lastFlushTime,
		CurrentMessageBatch:   currentMessageBatch,
		CurrentOfflineBatch:   currentOfflineBatch,
		CurrentReconcileBatch: currentReconcileBatch,
	}
}

// FlushAll flushes all pending batches immediately
func (bp *BatchProcessor) FlushAll(ctx context.Context) error {
	bp.flushMessageBatch(ctx)
	bp.flushOfflineBatch(ctx)
	bp.flushReconcileBatch(ctx)
	return nil
}

// DefaultBatchProcessorConfig returns a default batch processor configuration
func DefaultBatchProcessorConfig() BatchProcessorConfig {
	return BatchProcessorConfig{
		// Message batch configuration
		MessageBatchSize:     100,
		MessageFlushInterval: 100 * time.Millisecond,
		MessageMaxRetries:    3,

		// Offline message batch configuration
		OfflineBatchSize:     50,
		OfflineFlushInterval: 200 * time.Millisecond,
		OfflineMaxRetries:    3,

		// Reconcile batch configuration
		ReconcileBatchSize:     100,
		ReconcileFlushInterval: 500 * time.Millisecond,
		ReconcileMaxRetries:    3,

		// Performance tuning
		EnableCompression:    true,
		EnablePipelining:     true,
		MaxConcurrentBatches: 10,

		// Monitoring
		EnableMetrics:      true,
		MetricsInterval:    30 * time.Second,
		EnableHealthChecks: true,
	}
}

// BatchProcessorWithKafka creates a batch processor with Kafka integration
type BatchProcessorWithKafka struct {
	*BatchProcessor
	producer sarama.SyncProducer
}

// NewBatchProcessorWithKafka creates a batch processor with Kafka integration
func NewBatchProcessorWithKafka(config BatchProcessorConfig, producer sarama.SyncProducer) *BatchProcessorWithKafka {
	return &BatchProcessorWithKafka{
		BatchProcessor: NewBatchProcessor(config),
		producer:       producer,
	}
}

// BatchProcessorWithRedis creates a batch processor with Redis integration
type BatchProcessorWithRedis struct {
	*BatchProcessor
	redis *redis.Client
}

// NewBatchProcessorWithRedis creates a batch processor with Redis integration
func NewBatchProcessorWithRedis(config BatchProcessorConfig, redisClient *redis.Client) *BatchProcessorWithRedis {
	return &BatchProcessorWithRedis{
		BatchProcessor: NewBatchProcessor(config),
		redis:          redisClient,
	}
}
