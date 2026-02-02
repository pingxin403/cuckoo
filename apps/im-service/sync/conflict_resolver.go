package sync

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/pingxin403/cuckoo/libs/hlc"
)

// ConflictResolver handles conflict detection and resolution using LWW strategy
type ConflictResolver struct {
	regionID string
	logger   *log.Logger

	// Metrics
	conflictCount       int64
	lwwResolutionCount  int64
	localWinsCount      int64
	remoteWinsCount     int64
	resolutionTimeSum   int64 // Total resolution time in microseconds
	resolutionTimeCount int64

	// Configuration
	config ConflictResolverConfig
}

// ConflictResolverConfig holds configuration for the conflict resolver
type ConflictResolverConfig struct {
	RegionID              string        `json:"region_id"`
	EnableDetailedLogging bool          `json:"enable_detailed_logging"`
	MetricsInterval       time.Duration `json:"metrics_interval"`
	MaxConflictHistory    int           `json:"max_conflict_history"`
}

// MessageVersion represents a version of a message for conflict resolution
type MessageVersion struct {
	GlobalID       hlc.GlobalID      `json:"global_id"`
	MessageID      string            `json:"message_id"`
	Content        string            `json:"content"`
	Timestamp      int64             `json:"timestamp"`
	RegionID       string            `json:"region_id"`
	Version        int64             `json:"version"`
	SequenceNumber int64             `json:"sequence_number"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// ConflictResolution represents the result of conflict resolution
type ConflictResolution struct {
	MessageID        string         `json:"message_id"`
	LocalVersion     MessageVersion `json:"local_version"`
	RemoteVersion    MessageVersion `json:"remote_version"`
	Winner           MessageVersion `json:"winner"`
	Resolution       string         `json:"resolution"` // "local_wins", "remote_wins", "no_conflict"
	ResolutionReason string         `json:"resolution_reason"`
	ConflictTime     time.Time      `json:"conflict_time"`
	ResolutionTimeUs int64          `json:"resolution_time_us"`
}

// ConflictMetrics holds conflict resolution metrics
type ConflictMetrics struct {
	RegionID            string    `json:"region_id"`
	TotalConflicts      int64     `json:"total_conflicts"`
	LWWResolutions      int64     `json:"lww_resolutions"`
	LocalWins           int64     `json:"local_wins"`
	RemoteWins          int64     `json:"remote_wins"`
	AvgResolutionTimeUs float64   `json:"avg_resolution_time_us"`
	ConflictRate        float64   `json:"conflict_rate"` // Conflicts per second
	LastUpdated         time.Time `json:"last_updated"`
}

// DefaultConflictResolverConfig returns default configuration
func DefaultConflictResolverConfig(regionID string) ConflictResolverConfig {
	return ConflictResolverConfig{
		RegionID:              regionID,
		EnableDetailedLogging: true,
		MetricsInterval:       30 * time.Second,
		MaxConflictHistory:    1000,
	}
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(config ConflictResolverConfig, logger *log.Logger) *ConflictResolver {
	if logger == nil {
		logger = log.New(log.Writer(), "[ConflictResolver] ", log.LstdFlags|log.Lshortfile)
	}

	resolver := &ConflictResolver{
		regionID: config.RegionID,
		logger:   logger,
		config:   config,
	}

	logger.Printf("ConflictResolver initialized for region %s", config.RegionID)
	return resolver
}

// ResolveConflict resolves conflicts between local and remote message versions using LWW strategy
func (cr *ConflictResolver) ResolveConflict(
	ctx context.Context,
	localVersion, remoteVersion MessageVersion,
) (*ConflictResolution, error) {
	startTime := time.Now()

	resolution := &ConflictResolution{
		MessageID:     localVersion.MessageID,
		LocalVersion:  localVersion,
		RemoteVersion: remoteVersion,
		ConflictTime:  startTime,
	}

	// Compare Global IDs using HLC comparison
	cmp := hlc.CompareGlobalID(localVersion.GlobalID, remoteVersion.GlobalID)

	if cmp == 0 {
		// IDs are identical, no conflict
		resolution.Winner = localVersion
		resolution.Resolution = "no_conflict"
		resolution.ResolutionReason = "identical global IDs"

		if cr.config.EnableDetailedLogging {
			cr.logger.Printf("No conflict for message %s: identical global IDs",
				localVersion.MessageID)
		}

		resolution.ResolutionTimeUs = time.Since(startTime).Microseconds()
		return resolution, nil
	}

	// Record conflict metrics
	atomic.AddInt64(&cr.conflictCount, 1)
	atomic.AddInt64(&cr.lwwResolutionCount, 1)

	// LWW: HLC timestamp comparison determines winner
	if cmp > 0 {
		// Local version has later timestamp, local wins
		resolution.Winner = localVersion
		resolution.Resolution = "local_wins"
		resolution.ResolutionReason = "local version has later HLC timestamp"
		atomic.AddInt64(&cr.localWinsCount, 1)

		if cr.config.EnableDetailedLogging {
			cr.logger.Printf("Conflict resolved for message %s: LOCAL WINS (local_hlc=%s > remote_hlc=%s)",
				localVersion.MessageID, localVersion.GlobalID.HLC, remoteVersion.GlobalID.HLC)
		}
	} else {
		// Remote version has later timestamp, remote wins
		resolution.Winner = remoteVersion
		resolution.Resolution = "remote_wins"
		resolution.ResolutionReason = "remote version has later HLC timestamp"
		atomic.AddInt64(&cr.remoteWinsCount, 1)

		if cr.config.EnableDetailedLogging {
			cr.logger.Printf("Conflict resolved for message %s: REMOTE WINS (remote_hlc=%s > local_hlc=%s)",
				localVersion.MessageID, remoteVersion.GlobalID.HLC, localVersion.GlobalID.HLC)
		}
	}

	// Record resolution time
	resolutionTime := time.Since(startTime).Microseconds()
	resolution.ResolutionTimeUs = resolutionTime
	atomic.AddInt64(&cr.resolutionTimeSum, resolutionTime)
	atomic.AddInt64(&cr.resolutionTimeCount, 1)

	// Log conflict details
	cr.logConflictDetails(resolution)

	return resolution, nil
}

// logConflictDetails logs detailed conflict information
func (cr *ConflictResolver) logConflictDetails(resolution *ConflictResolution) {
	if !cr.config.EnableDetailedLogging {
		return
	}

	cr.logger.Printf("CONFLICT DETAILS for message %s:", resolution.MessageID)
	cr.logger.Printf("  Local:  region=%s, hlc=%s, version=%d, timestamp=%d",
		resolution.LocalVersion.RegionID,
		resolution.LocalVersion.GlobalID.HLC,
		resolution.LocalVersion.Version,
		resolution.LocalVersion.Timestamp)
	cr.logger.Printf("  Remote: region=%s, hlc=%s, version=%d, timestamp=%d",
		resolution.RemoteVersion.RegionID,
		resolution.RemoteVersion.GlobalID.HLC,
		resolution.RemoteVersion.Version,
		resolution.RemoteVersion.Timestamp)
	cr.logger.Printf("  Winner: %s (%s)",
		resolution.Resolution, resolution.ResolutionReason)
	cr.logger.Printf("  Resolution time: %d μs", resolution.ResolutionTimeUs)
}

// GetMetrics returns current conflict resolution metrics
func (cr *ConflictResolver) GetMetrics() ConflictMetrics {
	totalConflicts := atomic.LoadInt64(&cr.conflictCount)
	lwwResolutions := atomic.LoadInt64(&cr.lwwResolutionCount)
	localWins := atomic.LoadInt64(&cr.localWinsCount)
	remoteWins := atomic.LoadInt64(&cr.remoteWinsCount)
	resolutionTimeSum := atomic.LoadInt64(&cr.resolutionTimeSum)
	resolutionTimeCount := atomic.LoadInt64(&cr.resolutionTimeCount)

	avgResolutionTime := float64(0)
	if resolutionTimeCount > 0 {
		avgResolutionTime = float64(resolutionTimeSum) / float64(resolutionTimeCount)
	}

	conflictRate := float64(0)
	if totalConflicts > 0 {
		conflictRate = float64(totalConflicts) / time.Since(time.Now().Add(-cr.config.MetricsInterval)).Seconds()
	}

	return ConflictMetrics{
		RegionID:            cr.regionID,
		TotalConflicts:      totalConflicts,
		LWWResolutions:      lwwResolutions,
		LocalWins:           localWins,
		RemoteWins:          remoteWins,
		AvgResolutionTimeUs: avgResolutionTime,
		ConflictRate:        conflictRate,
		LastUpdated:         time.Now(),
	}
}

// ResetMetrics resets all conflict metrics (useful for testing)
func (cr *ConflictResolver) ResetMetrics() {
	atomic.StoreInt64(&cr.conflictCount, 0)
	atomic.StoreInt64(&cr.lwwResolutionCount, 0)
	atomic.StoreInt64(&cr.localWinsCount, 0)
	atomic.StoreInt64(&cr.remoteWinsCount, 0)
	atomic.StoreInt64(&cr.resolutionTimeSum, 0)
	atomic.StoreInt64(&cr.resolutionTimeCount, 0)

	cr.logger.Printf("Conflict metrics reset for region %s", cr.regionID)
}

// LogMetrics logs current metrics
func (cr *ConflictResolver) LogMetrics() {
	metrics := cr.GetMetrics()

	cr.logger.Printf("CONFLICT METRICS for region %s:", cr.regionID)
	cr.logger.Printf("  Total conflicts: %d", metrics.TotalConflicts)
	cr.logger.Printf("  LWW resolutions: %d", metrics.LWWResolutions)
	cr.logger.Printf("  Local wins: %d", metrics.LocalWins)
	cr.logger.Printf("  Remote wins: %d", metrics.RemoteWins)
	cr.logger.Printf("  Avg resolution time: %.2f μs", metrics.AvgResolutionTimeUs)
	cr.logger.Printf("  Conflict rate: %.4f conflicts/sec", metrics.ConflictRate)
}

// String returns a string representation of the conflict resolver
func (cr *ConflictResolver) String() string {
	metrics := cr.GetMetrics()
	return fmt.Sprintf("ConflictResolver{region=%s, strategy=LWW, conflicts=%d, local_wins=%d, remote_wins=%d}",
		cr.regionID, metrics.TotalConflicts, metrics.LocalWins, metrics.RemoteWins)
}
