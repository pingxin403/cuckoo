package reconcile

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RepairStrategy defines the strategy for repairing differences
type RepairStrategy string

const (
	// RepairStrategyPull fetches missing data from remote
	RepairStrategyPull RepairStrategy = "pull"

	// RepairStrategyPush sends local data to remote
	RepairStrategyPush RepairStrategy = "push"

	// RepairStrategyBidirectional performs both pull and push
	RepairStrategyBidirectional RepairStrategy = "bidirectional"
)

// IncrementalRepairer performs incremental data repair operations
type IncrementalRepairer struct {
	config   RepairConfig
	store    MessageStore
	provider RemoteTreeProvider

	mu               sync.RWMutex
	repairQueue      []RepairTask
	completedRepairs int64
	failedRepairs    int64
}

// RepairConfig holds configuration for incremental repair
type RepairConfig struct {
	RegionID       string
	Strategy       RepairStrategy
	BatchSize      int           // Number of messages to repair in one batch
	RetryAttempts  int           // Number of retry attempts for failed repairs
	RetryDelay     time.Duration // Delay between retry attempts
	MaxQueueSize   int           // Maximum size of repair queue
	EnablePriority bool          // Enable priority-based repair
}

// RepairTask represents a single repair task
type RepairTask struct {
	GlobalID     string
	Operation    string // "fetch", "update", "delete"
	Priority     int    // Higher priority tasks are processed first
	Attempts     int    // Number of attempts made
	LastAttempt  time.Time
	Error        error
	TargetRegion string
}

// RepairResult represents the result of a repair operation
type RepairResult struct {
	GlobalID string
	Success  bool
	Error    error
	Duration time.Duration
	Attempts int
}

// RepairBatchResult represents the result of a batch repair operation
type RepairBatchResult struct {
	TotalTasks int
	Successful int
	Failed     int
	Duration   time.Duration
	Results    []RepairResult
}

// NewIncrementalRepairer creates a new incremental repairer
func NewIncrementalRepairer(config RepairConfig, store MessageStore, provider RemoteTreeProvider) *IncrementalRepairer {
	return &IncrementalRepairer{
		config:      config,
		store:       store,
		provider:    provider,
		repairQueue: make([]RepairTask, 0),
	}
}

// QueueRepair adds a repair task to the queue
func (ir *IncrementalRepairer) QueueRepair(task RepairTask) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if len(ir.repairQueue) >= ir.config.MaxQueueSize {
		return fmt.Errorf("repair queue is full (max: %d)", ir.config.MaxQueueSize)
	}

	ir.repairQueue = append(ir.repairQueue, task)

	// Sort by priority if enabled
	if ir.config.EnablePriority {
		ir.sortQueueByPriority()
	}

	return nil
}

// QueueRepairs adds multiple repair tasks to the queue
func (ir *IncrementalRepairer) QueueRepairs(tasks []RepairTask) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if len(ir.repairQueue)+len(tasks) > ir.config.MaxQueueSize {
		return fmt.Errorf("repair queue would exceed max size (current: %d, adding: %d, max: %d)",
			len(ir.repairQueue), len(tasks), ir.config.MaxQueueSize)
	}

	ir.repairQueue = append(ir.repairQueue, tasks...)

	if ir.config.EnablePriority {
		ir.sortQueueByPriority()
	}

	return nil
}

// ProcessBatch processes a batch of repair tasks from the queue
func (ir *IncrementalRepairer) ProcessBatch(ctx context.Context) (*RepairBatchResult, error) {
	startTime := time.Now()

	ir.mu.Lock()
	batchSize := ir.config.BatchSize
	if batchSize > len(ir.repairQueue) {
		batchSize = len(ir.repairQueue)
	}

	if batchSize == 0 {
		ir.mu.Unlock()
		return &RepairBatchResult{
			TotalTasks: 0,
			Duration:   time.Since(startTime),
		}, nil
	}

	// Take batch from queue
	batch := make([]RepairTask, batchSize)
	copy(batch, ir.repairQueue[:batchSize])
	ir.repairQueue = ir.repairQueue[batchSize:]
	ir.mu.Unlock()

	// Process batch
	result := &RepairBatchResult{
		TotalTasks: len(batch),
		Results:    make([]RepairResult, 0, len(batch)),
	}

	var wg sync.WaitGroup
	resultCh := make(chan RepairResult, len(batch))

	for _, task := range batch {
		wg.Add(1)
		go func(t RepairTask) {
			defer wg.Done()

			taskResult := ir.processTask(ctx, t)
			resultCh <- taskResult
		}(task)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	for taskResult := range resultCh {
		result.Results = append(result.Results, taskResult)
		if taskResult.Success {
			result.Successful++
			ir.mu.Lock()
			ir.completedRepairs++
			ir.mu.Unlock()
		} else {
			result.Failed++
			ir.mu.Lock()
			ir.failedRepairs++
			ir.mu.Unlock()

			// Re-queue failed task if retries remain
			if taskResult.Attempts < ir.config.RetryAttempts {
				task := RepairTask{
					GlobalID:    taskResult.GlobalID,
					Operation:   "fetch", // Assuming fetch operation
					Priority:    1,       // Lower priority for retries
					Attempts:    taskResult.Attempts + 1,
					LastAttempt: time.Now(),
					Error:       taskResult.Error,
				}
				_ = ir.QueueRepair(task) // Ignore error if queue is full
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// processTask processes a single repair task
func (ir *IncrementalRepairer) processTask(ctx context.Context, task RepairTask) RepairResult {
	startTime := time.Now()
	result := RepairResult{
		GlobalID: task.GlobalID,
		Attempts: task.Attempts + 1,
	}

	// Add retry delay if this is a retry
	if task.Attempts > 0 {
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.Duration = time.Since(startTime)
			return result
		case <-time.After(ir.config.RetryDelay):
		}
	}

	// Perform repair based on operation
	var err error
	switch task.Operation {
	case "fetch":
		err = ir.fetchAndStore(ctx, task.GlobalID, task.TargetRegion)
	case "update":
		err = ir.updateMessage(ctx, task.GlobalID, task.TargetRegion)
	case "delete":
		err = ir.deleteMessage(ctx, task.GlobalID)
	default:
		err = fmt.Errorf("unknown operation: %s", task.Operation)
	}

	result.Success = (err == nil)
	result.Error = err
	result.Duration = time.Since(startTime)

	return result
}

// fetchAndStore fetches a message from remote and stores it locally
func (ir *IncrementalRepairer) fetchAndStore(ctx context.Context, globalID string, targetRegion string) error {
	// Fetch from remote
	messages, err := ir.provider.GetRemoteMessages(ctx, targetRegion, []string{globalID})
	if err != nil {
		return fmt.Errorf("failed to fetch message: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("message not found: %s", globalID)
	}

	// Store locally
	msg := messages[0]
	if err := ir.store.StoreMessage(ctx, &msg); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

// updateMessage updates a local message with remote version
func (ir *IncrementalRepairer) updateMessage(ctx context.Context, globalID string, targetRegion string) error {
	// Fetch remote version
	messages, err := ir.provider.GetRemoteMessages(ctx, targetRegion, []string{globalID})
	if err != nil {
		return fmt.Errorf("failed to fetch message: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("message not found: %s", globalID)
	}

	// Update local
	msg := messages[0]
	if err := ir.store.StoreMessage(ctx, &msg); err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// deleteMessage deletes a local message
func (ir *IncrementalRepairer) deleteMessage(ctx context.Context, globalID string) error {
	if err := ir.store.DeleteMessage(ctx, globalID); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// sortQueueByPriority sorts the repair queue by priority (descending)
func (ir *IncrementalRepairer) sortQueueByPriority() {
	// Simple bubble sort for small queues
	// In production, use a priority queue data structure
	n := len(ir.repairQueue)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if ir.repairQueue[j].Priority < ir.repairQueue[j+1].Priority {
				ir.repairQueue[j], ir.repairQueue[j+1] = ir.repairQueue[j+1], ir.repairQueue[j]
			}
		}
	}
}

// GetQueueSize returns the current size of the repair queue
func (ir *IncrementalRepairer) GetQueueSize() int {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	return len(ir.repairQueue)
}

// GetStats returns statistics about repair operations
func (ir *IncrementalRepairer) GetStats() map[string]interface{} {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	return map[string]interface{}{
		"queue_size":        len(ir.repairQueue),
		"completed_repairs": ir.completedRepairs,
		"failed_repairs":    ir.failedRepairs,
		"success_rate":      ir.calculateSuccessRate(),
	}
}

// calculateSuccessRate calculates the success rate of repairs
func (ir *IncrementalRepairer) calculateSuccessRate() float64 {
	total := ir.completedRepairs + ir.failedRepairs
	if total == 0 {
		return 0.0
	}
	return float64(ir.completedRepairs) / float64(total) * 100.0
}

// ClearQueue clears all pending repair tasks
func (ir *IncrementalRepairer) ClearQueue() {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.repairQueue = make([]RepairTask, 0)
}

// RepairDifferences creates repair tasks from a DiffResult
func (ir *IncrementalRepairer) RepairDifferences(diff *DiffResult, targetRegion string, priority int) error {
	tasks := make([]RepairTask, 0)

	// Create tasks for missing messages in local
	for _, globalID := range diff.MissingInLocal {
		tasks = append(tasks, RepairTask{
			GlobalID:     globalID,
			Operation:    "fetch",
			Priority:     priority,
			TargetRegion: targetRegion,
		})
	}

	// Create tasks for conflicts (update with remote version)
	for _, globalID := range diff.Conflicts {
		tasks = append(tasks, RepairTask{
			GlobalID:     globalID,
			Operation:    "update",
			Priority:     priority + 1, // Higher priority for conflicts
			TargetRegion: targetRegion,
		})
	}

	return ir.QueueRepairs(tasks)
}

// DefaultRepairConfig returns a default repair configuration
func DefaultRepairConfig(regionID string) RepairConfig {
	return RepairConfig{
		RegionID:       regionID,
		Strategy:       RepairStrategyPull,
		BatchSize:      50,
		RetryAttempts:  3,
		RetryDelay:     5 * time.Second,
		MaxQueueSize:   10000,
		EnablePriority: true,
	}
}
