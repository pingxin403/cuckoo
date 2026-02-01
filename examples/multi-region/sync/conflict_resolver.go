package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
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

	// State management
	mu sync.RWMutex
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

// DetectAndResolveConflict detects conflicts and resolves them for a remote message
func (cr *ConflictResolver) DetectAndResolveConflict(
	ctx context.Context,
	localStorage *storage.LocalStore,
	remoteMessage storage.LocalMessage,
) (*ConflictResolution, error) {
	// Check if message exists locally
	localMessage, err := localStorage.GetMessageByID(ctx, remoteMessage.MsgID)
	if err != nil {
		// Message doesn't exist locally, no conflict
		return &ConflictResolution{
			MessageID:        remoteMessage.MsgID,
			RemoteVersion:    cr.storageMessageToVersion(remoteMessage),
			Winner:           cr.storageMessageToVersion(remoteMessage),
			Resolution:       "no_conflict",
			ResolutionReason: "message does not exist locally",
			ConflictTime:     time.Now(),
			ResolutionTimeUs: 0,
		}, nil
	}

	// Convert storage messages to versions for comparison
	localVersion := cr.storageMessageToVersion(*localMessage)
	remoteVersion := cr.storageMessageToVersion(remoteMessage)

	// Resolve conflict using LWW strategy
	return cr.ResolveConflict(ctx, localVersion, remoteVersion)
}

// storageMessageToVersion converts a storage.LocalMessage to MessageVersion
func (cr *ConflictResolver) storageMessageToVersion(msg storage.LocalMessage) MessageVersion {
	// Parse GlobalID from string format
	globalID := cr.parseGlobalIDFromString(msg.GlobalID, msg.RegionID)

	return MessageVersion{
		GlobalID:       globalID,
		MessageID:      msg.MsgID,
		Content:        msg.Content,
		Timestamp:      msg.Timestamp,
		RegionID:       msg.RegionID,
		Version:        msg.Version,
		SequenceNumber: msg.SequenceNumber,
		Metadata:       msg.Metadata,
		CreatedAt:      msg.CreatedAt,
	}
}

// parseGlobalIDFromString parses a GlobalID from its string representation
func (cr *ConflictResolver) parseGlobalIDFromString(globalIDStr, regionID string) hlc.GlobalID {
	// Expected format: "region-hlc-sequence" or just the HLC part
	// For now, use the GlobalID string as HLC for comparison
	// In a production system, you'd want more robust parsing

	if globalIDStr == "" {
		// Fallback to current timestamp if no GlobalID
		return hlc.GlobalID{
			RegionID: regionID,
			HLC:      fmt.Sprintf("%d-0", time.Now().UnixMilli()),
			Sequence: 0,
		}
	}

	// Simple parsing - assume the globalIDStr is in the correct format
	// In production, you'd want to properly parse the components
	return hlc.GlobalID{
		RegionID: regionID,
		HLC:      globalIDStr, // Use the full string as HLC for now
		Sequence: 0,           // Would be parsed from the string in production
	}
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

// RecordConflict records a conflict for monitoring and metrics
func (cr *ConflictResolver) RecordConflict(
	ctx context.Context,
	localStorage *storage.LocalStore,
	resolution *ConflictResolution,
) error {
	// Convert to storage conflict info
	conflictInfo := storage.ConflictInfo{
		MessageID:     resolution.MessageID,
		LocalVersion:  resolution.LocalVersion.Version,
		RemoteVersion: resolution.RemoteVersion.Version,
		LocalRegion:   resolution.LocalVersion.RegionID,
		RemoteRegion:  resolution.RemoteVersion.RegionID,
		ConflictTime:  resolution.ConflictTime,
		Resolution:    resolution.Resolution,
	}

	// Record in storage for persistence
	if err := localStorage.RecordConflict(ctx, conflictInfo); err != nil {
		cr.logger.Printf("Failed to record conflict in storage: %v", err)
		return fmt.Errorf("failed to record conflict: %w", err)
	}

	cr.logger.Printf("Recorded conflict for message %s with resolution %s",
		resolution.MessageID, resolution.Resolution)
	return nil
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

	// Calculate conflict rate (conflicts per second) - simplified calculation
	// In production, you'd want a more sophisticated rate calculation
	conflictRate := float64(0)
	if totalConflicts > 0 {
		// This is a simplified rate calculation
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

// ValidateConflictResolution validates that a conflict resolution is consistent
func (cr *ConflictResolver) ValidateConflictResolution(resolution *ConflictResolution) error {
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

	// Validate that winner matches resolution
	switch resolution.Resolution {
	case "local_wins":
		if resolution.Winner.MessageID != resolution.LocalVersion.MessageID ||
			resolution.Winner.RegionID != resolution.LocalVersion.RegionID {
			return fmt.Errorf("winner does not match local version for local_wins resolution")
		}
	case "remote_wins":
		if resolution.Winner.MessageID != resolution.RemoteVersion.MessageID ||
			resolution.Winner.RegionID != resolution.RemoteVersion.RegionID {
			return fmt.Errorf("winner does not match remote version for remote_wins resolution")
		}
	case "no_conflict":
		// For no conflict, winner can be either version (typically local)
		if resolution.Winner.MessageID != resolution.MessageID {
			return fmt.Errorf("winner message ID does not match for no_conflict resolution")
		}
	}

	return nil
}

// CompareMessageVersions compares two message versions using HLC ordering
func (cr *ConflictResolver) CompareMessageVersions(v1, v2 MessageVersion) int {
	return hlc.CompareGlobalID(v1.GlobalID, v2.GlobalID)
}

// IsConflictResolutionDeterministic checks if conflict resolution is deterministic
// This is important for ensuring all regions resolve conflicts the same way
func (cr *ConflictResolver) IsConflictResolutionDeterministic(
	localVersion, remoteVersion MessageVersion,
) bool {
	// LWW with HLC is deterministic because:
	// 1. HLC provides total ordering
	// 2. RegionID tiebreaker ensures deterministic resolution even for identical timestamps
	// 3. Sequence number provides final tiebreaker

	cmp := hlc.CompareGlobalID(localVersion.GlobalID, remoteVersion.GlobalID)
	return cmp != 0 // If comparison is not equal, resolution is deterministic
}

// GetConflictResolutionStrategy returns the strategy used for conflict resolution
func (cr *ConflictResolver) GetConflictResolutionStrategy() string {
	return "LWW" // Last Write Wins based on HLC timestamps
}

// String returns a string representation of the conflict resolver
func (cr *ConflictResolver) String() string {
	metrics := cr.GetMetrics()
	return fmt.Sprintf("ConflictResolver{region=%s, strategy=LWW, conflicts=%d, local_wins=%d, remote_wins=%d}",
		cr.regionID, metrics.TotalConflicts, metrics.LocalWins, metrics.RemoteWins)
}
