package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Standalone version of ConflictResolver for testing without external dependencies

// StandaloneGlobalID represents a global ID without external dependencies
type StandaloneGlobalID struct {
	RegionID string `json:"region_id"`
	HLC      string `json:"hlc"`      // HLC timestamp in format "physical-logical"
	Sequence int64  `json:"sequence"` // Local sequence number
}

// StandaloneMessageVersion represents a message version for conflict resolution
type StandaloneMessageVersion struct {
	GlobalID       StandaloneGlobalID `json:"global_id"`
	MessageID      string             `json:"message_id"`
	Content        string             `json:"content"`
	Timestamp      int64              `json:"timestamp"`
	RegionID       string             `json:"region_id"`
	Version        int64              `json:"version"`
	SequenceNumber int64              `json:"sequence_number"`
	Metadata       map[string]string  `json:"metadata,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// StandaloneConflictResolution represents the result of conflict resolution
type StandaloneConflictResolution struct {
	MessageID        string                   `json:"message_id"`
	LocalVersion     StandaloneMessageVersion `json:"local_version"`
	RemoteVersion    StandaloneMessageVersion `json:"remote_version"`
	Winner           StandaloneMessageVersion `json:"winner"`
	Resolution       string                   `json:"resolution"` // "local_wins", "remote_wins", "no_conflict"
	ResolutionReason string                   `json:"resolution_reason"`
	ConflictTime     time.Time                `json:"conflict_time"`
	ResolutionTimeUs int64                    `json:"resolution_time_us"`
}

// StandaloneConflictResolver handles conflict resolution using LWW strategy
type StandaloneConflictResolver struct {
	regionID string
	logger   *log.Logger

	// Metrics
	conflictCount       int64
	lwwResolutionCount  int64
	localWinsCount      int64
	remoteWinsCount     int64
	resolutionTimeSum   int64
	resolutionTimeCount int64

	// Configuration
	enableDetailedLogging bool

	// State management
	mu sync.RWMutex
}

// NewStandaloneConflictResolver creates a new standalone conflict resolver
func NewStandaloneConflictResolver(regionID string, enableLogging bool, logger *log.Logger) *StandaloneConflictResolver {
	if logger == nil {
		logger = log.New(log.Writer(), "[StandaloneConflictResolver] ", log.LstdFlags|log.Lshortfile)
	}

	resolver := &StandaloneConflictResolver{
		regionID:              regionID,
		logger:                logger,
		enableDetailedLogging: enableLogging,
	}

	logger.Printf("StandaloneConflictResolver initialized for region %s", regionID)
	return resolver
}

// ResolveConflict resolves conflicts between local and remote message versions using LWW strategy
func (cr *StandaloneConflictResolver) ResolveConflict(
	ctx context.Context,
	localVersion, remoteVersion StandaloneMessageVersion,
) (*StandaloneConflictResolution, error) {
	startTime := time.Now()

	resolution := &StandaloneConflictResolution{
		MessageID:     localVersion.MessageID,
		LocalVersion:  localVersion,
		RemoteVersion: remoteVersion,
		ConflictTime:  startTime,
	}

	// Compare Global IDs using HLC comparison
	cmp := cr.compareGlobalID(localVersion.GlobalID, remoteVersion.GlobalID)

	if cmp == 0 {
		// IDs are identical, no conflict
		resolution.Winner = localVersion
		resolution.Resolution = "no_conflict"
		resolution.ResolutionReason = "identical global IDs"

		if cr.enableDetailedLogging {
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

		if cr.enableDetailedLogging {
			cr.logger.Printf("Conflict resolved for message %s: LOCAL WINS (local_hlc=%s > remote_hlc=%s)",
				localVersion.MessageID, localVersion.GlobalID.HLC, remoteVersion.GlobalID.HLC)
		}
	} else {
		// Remote version has later timestamp, remote wins
		resolution.Winner = remoteVersion
		resolution.Resolution = "remote_wins"
		resolution.ResolutionReason = "remote version has later HLC timestamp"
		atomic.AddInt64(&cr.remoteWinsCount, 1)

		if cr.enableDetailedLogging {
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

// compareGlobalID compares two StandaloneGlobalIDs and returns:
// -1 if id1 < id2, 0 if id1 == id2, 1 if id1 > id2
func (cr *StandaloneConflictResolver) compareGlobalID(id1, id2 StandaloneGlobalID) int {
	// 1. Compare HLC timestamps (format: "physical-logical")
	cmp := strings.Compare(id1.HLC, id2.HLC)
	if cmp != 0 {
		return cmp
	}

	// 2. HLC timestamps are identical, use RegionID as tiebreaker
	cmp = strings.Compare(id1.RegionID, id2.RegionID)
	if cmp != 0 {
		return cmp
	}

	// 3. Same region, compare sequence numbers
	if id1.Sequence < id2.Sequence {
		return -1
	} else if id1.Sequence > id2.Sequence {
		return 1
	}

	// 4. Completely identical
	return 0
}

// logConflictDetails logs detailed conflict information
func (cr *StandaloneConflictResolver) logConflictDetails(resolution *StandaloneConflictResolution) {
	if !cr.enableDetailedLogging {
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
func (cr *StandaloneConflictResolver) GetMetrics() map[string]interface{} {
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

	return map[string]interface{}{
		"region_id":              cr.regionID,
		"total_conflicts":        totalConflicts,
		"lww_resolutions":        lwwResolutions,
		"local_wins":             localWins,
		"remote_wins":            remoteWins,
		"avg_resolution_time_us": avgResolutionTime,
		"resolution_count":       resolutionTimeCount,
	}
}

// ResetMetrics resets all conflict metrics
func (cr *StandaloneConflictResolver) ResetMetrics() {
	atomic.StoreInt64(&cr.conflictCount, 0)
	atomic.StoreInt64(&cr.lwwResolutionCount, 0)
	atomic.StoreInt64(&cr.localWinsCount, 0)
	atomic.StoreInt64(&cr.remoteWinsCount, 0)
	atomic.StoreInt64(&cr.resolutionTimeSum, 0)
	atomic.StoreInt64(&cr.resolutionTimeCount, 0)

	cr.logger.Printf("Conflict metrics reset for region %s", cr.regionID)
}

// LogMetrics logs current metrics
func (cr *StandaloneConflictResolver) LogMetrics() {
	metrics := cr.GetMetrics()

	cr.logger.Printf("CONFLICT METRICS for region %s:", cr.regionID)
	cr.logger.Printf("  Total conflicts: %d", metrics["total_conflicts"])
	cr.logger.Printf("  LWW resolutions: %d", metrics["lww_resolutions"])
	cr.logger.Printf("  Local wins: %d", metrics["local_wins"])
	cr.logger.Printf("  Remote wins: %d", metrics["remote_wins"])
	cr.logger.Printf("  Avg resolution time: %.2f μs", metrics["avg_resolution_time_us"])
}

// GetConflictResolutionStrategy returns the strategy used for conflict resolution
func (cr *StandaloneConflictResolver) GetConflictResolutionStrategy() string {
	return "LWW" // Last Write Wins based on HLC timestamps
}

// ValidateConflictResolution validates that a conflict resolution is consistent
func (cr *StandaloneConflictResolver) ValidateConflictResolution(resolution *StandaloneConflictResolution) error {
	if resolution == nil {
		return fmt.Errorf("resolution cannot be nil")
	}

	if resolution.MessageID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	if resolution.LocalVersion.MessageID != resolution.RemoteVersion.MessageID {
		return fmt.Errorf("local and remote versions must have same message ID")
	}

	if resolution.Resolution != "local_wins" &&
		resolution.Resolution != "remote_wins" &&
		resolution.Resolution != "no_conflict" {
		return fmt.Errorf("invalid resolution type: %s", resolution.Resolution)
	}

	return nil
}

// IsConflictResolutionDeterministic checks if conflict resolution is deterministic
func (cr *StandaloneConflictResolver) IsConflictResolutionDeterministic(
	localVersion, remoteVersion StandaloneMessageVersion,
) bool {
	cmp := cr.compareGlobalID(localVersion.GlobalID, remoteVersion.GlobalID)
	return cmp != 0 // If comparison is not equal, resolution is deterministic
}

// String returns a string representation of the conflict resolver
func (cr *StandaloneConflictResolver) String() string {
	metrics := cr.GetMetrics()
	return fmt.Sprintf("StandaloneConflictResolver{region=%s, strategy=LWW, conflicts=%d, local_wins=%d, remote_wins=%d}",
		cr.regionID, metrics["total_conflicts"], metrics["local_wins"], metrics["remote_wins"])
}
