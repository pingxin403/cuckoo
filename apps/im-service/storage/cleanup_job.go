package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// CleanupJob handles periodic TTL cleanup of expired offline messages
type CleanupJob struct {
	store       *OfflineStore
	batchSize   int
	interval    time.Duration
	stopCh      chan struct{}
	stoppedCh   chan struct{}
	statsLogger StatsLogger
}

// CleanupStats holds statistics about cleanup operations
type CleanupStats struct {
	RunTime           time.Time
	MessagesDeleted   int64
	RemainingExpired  int64
	OldestExpiredTime *time.Time
	Duration          time.Duration
	Error             error
}

// StatsLogger is an interface for logging cleanup statistics
type StatsLogger interface {
	LogCleanupStats(stats CleanupStats)
}

// DefaultStatsLogger is a simple logger that prints to stdout
type DefaultStatsLogger struct{}

// LogCleanupStats logs cleanup statistics
func (l *DefaultStatsLogger) LogCleanupStats(stats CleanupStats) {
	if stats.Error != nil {
		log.Printf("[CleanupJob] ERROR: %v", stats.Error)
		return
	}

	log.Printf("[CleanupJob] Run completed at %s", stats.RunTime.Format(time.RFC3339))
	log.Printf("[CleanupJob] Messages deleted: %d", stats.MessagesDeleted)
	log.Printf("[CleanupJob] Remaining expired: %d", stats.RemainingExpired)
	if stats.OldestExpiredTime != nil {
		log.Printf("[CleanupJob] Oldest expired message: %s", stats.OldestExpiredTime.Format(time.RFC3339))
	}
	log.Printf("[CleanupJob] Duration: %s", stats.Duration)
}

// CleanupJobConfig holds configuration for the cleanup job
type CleanupJobConfig struct {
	BatchSize   int           // Number of messages to delete per run (default: 10000)
	Interval    time.Duration // How often to run cleanup (default: 1 hour)
	StatsLogger StatsLogger   // Logger for cleanup statistics (optional)
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(store *OfflineStore, config CleanupJobConfig) *CleanupJob {
	if config.BatchSize <= 0 {
		config.BatchSize = 10000
	}
	if config.Interval <= 0 {
		config.Interval = 1 * time.Hour
	}
	if config.StatsLogger == nil {
		config.StatsLogger = &DefaultStatsLogger{}
	}

	return &CleanupJob{
		store:       store,
		batchSize:   config.BatchSize,
		interval:    config.Interval,
		stopCh:      make(chan struct{}),
		stoppedCh:   make(chan struct{}),
		statsLogger: config.StatsLogger,
	}
}

// Start starts the cleanup job
func (j *CleanupJob) Start() {
	log.Printf("[CleanupJob] Starting cleanup job (interval: %s, batch size: %d)", j.interval, j.batchSize)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	defer close(j.stoppedCh)

	// Run immediately on start
	j.runCleanup()

	for {
		select {
		case <-ticker.C:
			j.runCleanup()
		case <-j.stopCh:
			log.Printf("[CleanupJob] Stopping cleanup job")
			return
		}
	}
}

// Stop stops the cleanup job
func (j *CleanupJob) Stop() {
	close(j.stopCh)
	<-j.stoppedCh
	log.Printf("[CleanupJob] Cleanup job stopped")
}

// runCleanup performs a single cleanup run
func (j *CleanupJob) runCleanup() {
	startTime := time.Now()
	ctx := context.Background()

	stats := CleanupStats{
		RunTime: startTime,
	}

	// Delete expired messages
	deleted, err := j.store.DeleteExpiredMessages(ctx, j.batchSize)
	if err != nil {
		stats.Error = fmt.Errorf("failed to delete expired messages: %w", err)
		j.statsLogger.LogCleanupStats(stats)
		return
	}
	stats.MessagesDeleted = deleted

	// Get remaining expired message count
	remaining, err := j.store.GetExpiredMessageCount(ctx)
	if err != nil {
		stats.Error = fmt.Errorf("failed to count remaining expired messages: %w", err)
		j.statsLogger.LogCleanupStats(stats)
		return
	}
	stats.RemainingExpired = remaining

	// Get oldest expired message timestamp
	oldest, err := j.store.GetOldestExpiredMessage(ctx)
	if err != nil {
		stats.Error = fmt.Errorf("failed to get oldest expired message: %w", err)
		j.statsLogger.LogCleanupStats(stats)
		return
	}
	stats.OldestExpiredTime = oldest

	stats.Duration = time.Since(startTime)
	j.statsLogger.LogCleanupStats(stats)
}

// RunOnce runs cleanup once (useful for testing or manual triggers)
func (j *CleanupJob) RunOnce() CleanupStats {
	startTime := time.Now()
	ctx := context.Background()

	stats := CleanupStats{
		RunTime: startTime,
	}

	// Delete expired messages
	deleted, err := j.store.DeleteExpiredMessages(ctx, j.batchSize)
	if err != nil {
		stats.Error = fmt.Errorf("failed to delete expired messages: %w", err)
		return stats
	}
	stats.MessagesDeleted = deleted

	// Get remaining expired message count
	remaining, err := j.store.GetExpiredMessageCount(ctx)
	if err != nil {
		stats.Error = fmt.Errorf("failed to count remaining expired messages: %w", err)
		return stats
	}
	stats.RemainingExpired = remaining

	// Get oldest expired message timestamp
	oldest, err := j.store.GetOldestExpiredMessage(ctx)
	if err != nil {
		stats.Error = fmt.Errorf("failed to get oldest expired message: %w", err)
		return stats
	}
	stats.OldestExpiredTime = oldest

	stats.Duration = time.Since(startTime)
	return stats
}
