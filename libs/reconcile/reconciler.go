package reconcile

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MessageStore defines the interface for accessing message data
type MessageStore interface {
	// GetMessagesForReconciliation retrieves messages for reconciliation
	GetMessagesForReconciliation(ctx context.Context, startTime, endTime time.Time) ([]MessageData, error)

	// GetMessageByGlobalID retrieves a specific message by its GlobalID
	GetMessageByGlobalID(ctx context.Context, globalID string) (*MessageData, error)

	// StoreMessage stores or updates a message
	StoreMessage(ctx context.Context, msg *MessageData) error

	// DeleteMessage deletes a message by GlobalID
	DeleteMessage(ctx context.Context, globalID string) error
}

// RemoteTreeProvider defines the interface for fetching remote Merkle trees
type RemoteTreeProvider interface {
	// GetRemoteTree fetches the Merkle tree from a remote region
	GetRemoteTree(ctx context.Context, regionID string, startTime, endTime time.Time) (*MerkleTree, error)

	// GetRemoteMessages fetches specific messages from a remote region
	GetRemoteMessages(ctx context.Context, regionID string, globalIDs []string) ([]MessageData, error)
}

// ReconcilerConfig holds configuration for the reconciler
type ReconcilerConfig struct {
	RegionID           string        // Current region ID
	CheckInterval      time.Duration // How often to run reconciliation
	TimeWindow         time.Duration // Time window for each reconciliation run
	MaxConcurrentFixes int           // Maximum concurrent repair operations
	EnableAutoRepair   bool          // Whether to automatically repair differences
	DryRun             bool          // If true, only detect differences without repairing
}

// Reconciler performs periodic data reconciliation between regions
type Reconciler struct {
	config   ReconcilerConfig
	store    MessageStore
	provider RemoteTreeProvider

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}

	// Metrics
	lastRunTime   time.Time
	lastRunStats  *ReconcileStats
	totalRuns     int64
	totalRepaired int64
	totalFailed   int64

	// Reporting and alerting
	reportGenerator *ReportGenerator
	alertManager    *AlertManager
}

// RepairOperation represents a single repair operation
type RepairOperation struct {
	GlobalID  string
	Operation string // "add", "update", "delete"
	Message   *MessageData
	Error     error
}

// NewReconciler creates a new reconciler instance
func NewReconciler(config ReconcilerConfig, store MessageStore, provider RemoteTreeProvider) *Reconciler {
	return &Reconciler{
		config:   config,
		store:    store,
		provider: provider,
		stopCh:   make(chan struct{}),
	}
}

// NewReconcilerWithReporting creates a reconciler with reporting and alerting
func NewReconcilerWithReporting(
	config ReconcilerConfig,
	store MessageStore,
	provider RemoteTreeProvider,
	reportGenerator *ReportGenerator,
	alertManager *AlertManager,
) *Reconciler {
	return &Reconciler{
		config:          config,
		store:           store,
		provider:        provider,
		stopCh:          make(chan struct{}),
		reportGenerator: reportGenerator,
		alertManager:    alertManager,
	}
}

// Start starts the reconciliation process
func (r *Reconciler) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("reconciler already running")
	}
	r.running = true
	r.mu.Unlock()

	go r.reconcileLoop(ctx)
	return nil
}

// Stop stops the reconciliation process
func (r *Reconciler) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return fmt.Errorf("reconciler not running")
	}

	close(r.stopCh)
	r.running = false
	return nil
}

// IsRunning returns whether the reconciler is currently running
func (r *Reconciler) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// GetLastRunStats returns statistics from the last reconciliation run
func (r *Reconciler) GetLastRunStats() *ReconcileStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastRunStats
}

// GetTotalStats returns cumulative statistics
func (r *Reconciler) GetTotalStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"total_runs":     r.totalRuns,
		"total_repaired": r.totalRepaired,
		"total_failed":   r.totalFailed,
		"last_run_time":  r.lastRunTime,
	}
}

// reconcileLoop runs the reconciliation process periodically
func (r *Reconciler) reconcileLoop(ctx context.Context) {
	ticker := time.NewTicker(r.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			// Run reconciliation
			stats, err := r.RunReconciliation(ctx)
			if err != nil {
				// Log error but continue
				continue
			}

			r.mu.Lock()
			r.lastRunTime = time.Now()
			r.lastRunStats = stats
			r.totalRuns++
			r.totalRepaired += int64(stats.Repaired)
			r.totalFailed += int64(stats.Failed)
			r.mu.Unlock()
		}
	}
}

// RunReconciliation performs a single reconciliation run
func (r *Reconciler) RunReconciliation(ctx context.Context) (*ReconcileStats, error) {
	stats := &ReconcileStats{
		StartTime: time.Now(),
	}

	// Define time window for this reconciliation
	endTime := time.Now()
	startTime := endTime.Add(-r.config.TimeWindow)

	// Build local Merkle tree
	localMessages, err := r.store.GetMessagesForReconciliation(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get local messages: %w", err)
	}

	// Compute hashes for local messages
	for i := range localMessages {
		localMessages[i].Hash = ComputeMessageHash(localMessages[i])
	}

	localTree := NewMerkleTree(r.config.RegionID, localMessages)

	// Get remote tree (assuming single remote region for now)
	// In production, this would iterate over all peer regions
	remoteRegionID := r.getPeerRegionID()
	remoteTree, err := r.provider.GetRemoteTree(ctx, remoteRegionID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote tree: %w", err)
	}

	// Find differences
	diff, err := localTree.FindDifferences(ctx, remoteTree)
	if err != nil {
		return nil, fmt.Errorf("failed to find differences: %w", err)
	}

	stats.MessagesChecked = diff.TotalChecked
	stats.Differences = diff.DiffCount

	// Collect repair results for reporting
	var repairResults []RepairResult
	var errors []ErrorRecord

	// Repair differences if enabled
	if r.config.EnableAutoRepair && !r.config.DryRun {
		repaired, failed, results := r.repairDifferencesWithResults(ctx, diff, remoteRegionID)
		stats.Repaired = repaired
		stats.Failed = failed
		repairResults = results
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	// Update internal state
	r.mu.Lock()
	r.lastRunTime = stats.EndTime
	r.lastRunStats = stats
	r.totalRuns++
	r.totalRepaired += int64(stats.Repaired)
	r.totalFailed += int64(stats.Failed)
	r.mu.Unlock()

	// Generate report if report generator is configured
	if r.reportGenerator != nil {
		timeWindow := TimeWindow{
			StartTime: startTime,
			EndTime:   endTime,
		}
		report, err := r.reportGenerator.GenerateReport(
			r.config.RegionID,
			remoteRegionID,
			stats,
			diff,
			repairResults,
			errors,
			timeWindow,
		)
		if err != nil {
			// Log error but don't fail reconciliation
			fmt.Printf("Failed to generate report: %v\n", err)
		} else {
			// Evaluate alerts if alert manager is configured
			if r.alertManager != nil {
				alerts, err := r.alertManager.EvaluateReport(ctx, report)
				if err != nil {
					fmt.Printf("Failed to evaluate alerts: %v\n", err)
				} else if len(alerts) > 0 {
					fmt.Printf("Generated %d alerts for report %s\n", len(alerts), report.ReportID)
				}
			}
		}
	}

	return stats, nil
}

// repairDifferences repairs the differences found between local and remote trees
func (r *Reconciler) repairDifferences(ctx context.Context, diff *DiffResult, remoteRegionID string) (int, int) {
	var repaired, failed int
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent repairs
	sem := make(chan struct{}, r.config.MaxConcurrentFixes)

	// Repair missing messages in local
	for _, globalID := range diff.MissingInLocal {
		wg.Add(1)
		go func(gid string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			if err := r.repairMissingInLocal(ctx, gid, remoteRegionID); err != nil {
				failed++
			} else {
				repaired++
			}
		}(globalID)
	}

	// Repair conflicts (use remote version as source of truth)
	for _, globalID := range diff.Conflicts {
		wg.Add(1)
		go func(gid string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			if err := r.repairConflict(ctx, gid, remoteRegionID); err != nil {
				failed++
			} else {
				repaired++
			}
		}(globalID)
	}

	wg.Wait()

	return repaired, failed
}

// repairDifferencesWithResults repairs differences and returns detailed results
func (r *Reconciler) repairDifferencesWithResults(ctx context.Context, diff *DiffResult, remoteRegionID string) (int, int, []RepairResult) {
	var repaired, failed int
	var wg sync.WaitGroup
	results := make([]RepairResult, 0)
	resultsMu := sync.Mutex{}

	// Use semaphore to limit concurrent repairs
	sem := make(chan struct{}, r.config.MaxConcurrentFixes)

	// Repair missing messages in local
	for _, globalID := range diff.MissingInLocal {
		wg.Add(1)
		go func(gid string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			startTime := time.Now()
			err := r.repairMissingInLocal(ctx, gid, remoteRegionID)
			duration := time.Since(startTime)

			result := RepairResult{
				GlobalID: gid,
				Success:  err == nil,
				Error:    err,
				Duration: duration,
				Attempts: 1,
			}

			resultsMu.Lock()
			results = append(results, result)
			if err == nil {
				repaired++
			} else {
				failed++
			}
			resultsMu.Unlock()
		}(globalID)
	}

	// Repair conflicts (use remote version as source of truth)
	for _, globalID := range diff.Conflicts {
		wg.Add(1)
		go func(gid string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			startTime := time.Now()
			err := r.repairConflict(ctx, gid, remoteRegionID)
			duration := time.Since(startTime)

			result := RepairResult{
				GlobalID: gid,
				Success:  err == nil,
				Error:    err,
				Duration: duration,
				Attempts: 1,
			}

			resultsMu.Lock()
			results = append(results, result)
			if err == nil {
				repaired++
			} else {
				failed++
			}
			resultsMu.Unlock()
		}(globalID)
	}

	wg.Wait()

	return repaired, failed, results
}

// repairMissingInLocal fetches a missing message from remote and stores it locally
func (r *Reconciler) repairMissingInLocal(ctx context.Context, globalID string, remoteRegionID string) error {
	// Fetch message from remote
	messages, err := r.provider.GetRemoteMessages(ctx, remoteRegionID, []string{globalID})
	if err != nil {
		return fmt.Errorf("failed to fetch remote message: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("message not found in remote: %s", globalID)
	}

	// Store message locally
	msg := messages[0]
	if err := r.store.StoreMessage(ctx, &msg); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

// repairConflict resolves a conflict by using the remote version
func (r *Reconciler) repairConflict(ctx context.Context, globalID string, remoteRegionID string) error {
	// Fetch remote version
	messages, err := r.provider.GetRemoteMessages(ctx, remoteRegionID, []string{globalID})
	if err != nil {
		return fmt.Errorf("failed to fetch remote message: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("message not found in remote: %s", globalID)
	}

	// Update local version with remote data
	msg := messages[0]
	if err := r.store.StoreMessage(ctx, &msg); err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// getPeerRegionID returns the peer region ID
// In a real implementation, this would support multiple peer regions
func (r *Reconciler) getPeerRegionID() string {
	// Simple logic: if we're region-a, peer is region-b, and vice versa
	if r.config.RegionID == "region-a" {
		return "region-b"
	}
	return "region-a"
}

// RunOnDemandReconciliation performs an on-demand reconciliation for a specific time range
func (r *Reconciler) RunOnDemandReconciliation(ctx context.Context, startTime, endTime time.Time, targetRegion string) (*ReconcileStats, *DiffResult, error) {
	stats := &ReconcileStats{
		StartTime: time.Now(),
	}

	// Build local Merkle tree
	localMessages, err := r.store.GetMessagesForReconciliation(ctx, startTime, endTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local messages: %w", err)
	}

	// Compute hashes
	for i := range localMessages {
		localMessages[i].Hash = ComputeMessageHash(localMessages[i])
	}

	localTree := NewMerkleTree(r.config.RegionID, localMessages)

	// Get remote tree
	remoteTree, err := r.provider.GetRemoteTree(ctx, targetRegion, startTime, endTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get remote tree: %w", err)
	}

	// Find differences
	diff, err := localTree.FindDifferences(ctx, remoteTree)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find differences: %w", err)
	}

	stats.MessagesChecked = diff.TotalChecked
	stats.Differences = diff.DiffCount

	// Repair if enabled
	if r.config.EnableAutoRepair && !r.config.DryRun {
		repaired, failed := r.repairDifferences(ctx, diff, targetRegion)
		stats.Repaired = repaired
		stats.Failed = failed
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	return stats, diff, nil
}

// DefaultReconcilerConfig returns a default reconciler configuration
func DefaultReconcilerConfig(regionID string) ReconcilerConfig {
	return ReconcilerConfig{
		RegionID:           regionID,
		CheckInterval:      1 * time.Hour,  // Run every hour
		TimeWindow:         24 * time.Hour, // Check last 24 hours
		MaxConcurrentFixes: 10,             // Max 10 concurrent repairs
		EnableAutoRepair:   true,           // Auto-repair enabled
		DryRun:             false,          // Actually perform repairs
	}
}
