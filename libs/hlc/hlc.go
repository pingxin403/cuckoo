package hlc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HLC represents a Hybrid Logical Clock
type HLC struct {
	mu           sync.RWMutex
	physicalTime int64  // Physical time in milliseconds
	logicalTime  int64  // Logical counter
	regionID     string // Region identifier
	nodeID       string // Node identifier
	sequence     int64  // Local sequence counter (atomic)
}

// GlobalID represents a globally unique identifier with HLC timestamp
type GlobalID struct {
	RegionID string `json:"region_id"`
	HLC      string `json:"hlc"`      // HLC timestamp in format "physical-logical"
	Sequence int64  `json:"sequence"` // Local sequence number
}

// HLCTimestamp represents the parsed HLC timestamp
type HLCTimestamp struct {
	Physical int64 // Physical time component
	Logical  int64 // Logical time component
}

// NewHLC creates a new HLC instance
func NewHLC(regionID, nodeID string) *HLC {
	return &HLC{
		physicalTime: time.Now().UnixMilli(),
		logicalTime:  0,
		regionID:     regionID,
		nodeID:       nodeID,
		sequence:     0,
	}
}

// GenerateID generates a new global ID with HLC timestamp
func (h *HLC) GenerateID() GlobalID {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UnixMilli()

	if now > h.physicalTime {
		// Physical time advanced, reset logical time
		h.physicalTime = now
		h.logicalTime = 0
	} else {
		// Physical time same or went backwards, increment logical time
		h.logicalTime++
	}

	hlcStr := fmt.Sprintf("%d-%d", h.physicalTime, h.logicalTime)
	sequence := atomic.AddInt64(&h.sequence, 1)

	return GlobalID{
		RegionID: h.regionID,
		HLC:      hlcStr,
		Sequence: sequence,
	}
}

// UpdateFromRemote updates the HLC based on a remote timestamp
// This ensures causal ordering is maintained across regions
func (h *HLC) UpdateFromRemote(remoteHLC string) error {
	remoteTimestamp, err := parseHLC(remoteHLC)
	if err != nil {
		return fmt.Errorf("failed to parse remote HLC: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UnixMilli()

	// Update physical time to max of local, remote, and current wall clock
	maxPhysical := max(h.physicalTime, remoteTimestamp.Physical, now)

	if maxPhysical > h.physicalTime {
		// Physical time advanced
		h.physicalTime = maxPhysical
		if maxPhysical == remoteTimestamp.Physical {
			// If we're using remote physical time, start logical time after remote
			h.logicalTime = remoteTimestamp.Logical + 1
		} else {
			// If we're using wall clock time, reset logical time
			h.logicalTime = 0
		}
	} else if maxPhysical == h.physicalTime {
		// Same physical time, increment logical time beyond remote
		if remoteTimestamp.Logical >= h.logicalTime {
			h.logicalTime = remoteTimestamp.Logical + 1
		} else {
			h.logicalTime++
		}
	}

	return nil
}

// GetCurrentTimestamp returns the current HLC timestamp as a string
func (h *HLC) GetCurrentTimestamp() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return fmt.Sprintf("%d-%d", h.physicalTime, h.logicalTime)
}

// GetPhysicalTime returns the current physical time
func (h *HLC) GetPhysicalTime() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.physicalTime
}

// GetLogicalTime returns the current logical time
func (h *HLC) GetLogicalTime() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.logicalTime
}

// GetRegionID returns the region ID
func (h *HLC) GetRegionID() string {
	return h.regionID
}

// GetNodeID returns the node ID
func (h *HLC) GetNodeID() string {
	return h.nodeID
}

// GetSequence returns the current sequence number
func (h *HLC) GetSequence() int64 {
	return atomic.LoadInt64(&h.sequence)
}

// AdjustForDrift adjusts the HLC based on detected clock drift
// When offset is positive (local clock is ahead), we increase logical counter step
// to compensate for the drift while maintaining monotonicity
func (h *HLC) AdjustForDrift(offset time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Convert offset to milliseconds
	offsetMs := offset.Milliseconds()

	if offsetMs == 0 {
		return nil // No adjustment needed
	}

	// For positive offset (local clock ahead), increase logical counter
	// This ensures generated IDs remain monotonic even after calibration
	if offsetMs > 0 {
		// Increase logical counter by offset amount
		// This compensates for the physical time being ahead
		h.logicalTime += offsetMs
	} else {
		// For negative offset (local clock behind), we don't adjust
		// because the next GenerateID will naturally use the higher wall clock time
		// No action needed - HLC algorithm handles this automatically
	}

	return nil
}

// CompareGlobalID compares two GlobalIDs and returns:
// -1 if id1 < id2
//
//	0 if id1 == id2
//	1 if id1 > id2
func CompareGlobalID(id1, id2 GlobalID) int {
	// 1. Parse and compare HLC timestamps
	hlc1, err1 := parseHLC(id1.HLC)
	hlc2, err2 := parseHLC(id2.HLC)

	// If parsing fails, fall back to string comparison
	if err1 != nil || err2 != nil {
		return strings.Compare(id1.HLC, id2.HLC)
	}

	// 2. Compare physical time first
	if hlc1.Physical != hlc2.Physical {
		if hlc1.Physical < hlc2.Physical {
			return -1
		}
		return 1
	}

	// 3. Physical time same, compare logical time
	if hlc1.Logical != hlc2.Logical {
		if hlc1.Logical < hlc2.Logical {
			return -1
		}
		return 1
	}

	// 4. HLC timestamps are identical, use RegionID as tiebreaker
	if id1.RegionID != id2.RegionID {
		return strings.Compare(id1.RegionID, id2.RegionID)
	}

	// 5. Same region, compare sequence numbers
	if id1.Sequence != id2.Sequence {
		if id1.Sequence < id2.Sequence {
			return -1
		}
		return 1
	}

	// 6. Completely identical
	return 0
}

// parseHLC parses an HLC timestamp string into its components
func parseHLC(hlcStr string) (HLCTimestamp, error) {
	parts := strings.Split(hlcStr, "-")
	if len(parts) != 2 {
		return HLCTimestamp{}, fmt.Errorf("invalid HLC format: %s", hlcStr)
	}

	physical, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return HLCTimestamp{}, fmt.Errorf("invalid physical time: %s", parts[0])
	}

	logical, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return HLCTimestamp{}, fmt.Errorf("invalid logical time: %s", parts[1])
	}

	return HLCTimestamp{
		Physical: physical,
		Logical:  logical,
	}, nil
}

// max returns the maximum of three int64 values
func max(a, b, c int64) int64 {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}

// String returns a string representation of the GlobalID
func (g GlobalID) String() string {
	return fmt.Sprintf("%s-%s-%d", g.RegionID, g.HLC, g.Sequence)
}

// ParseGlobalID parses a string representation of GlobalID back into a GlobalID struct
// Format: "regionID-hlc-sequence" (e.g., "region-a-1234567890-5-42")
func ParseGlobalID(s string) (GlobalID, error) {
	if s == "" {
		return GlobalID{}, fmt.Errorf("empty global ID string")
	}

	// Split by last two dashes to get regionID, HLC, and sequence
	// Format: regionID-physical-logical-sequence
	parts := strings.Split(s, "-")
	if len(parts) < 4 {
		return GlobalID{}, fmt.Errorf("invalid global ID format: %s (expected at least 4 parts)", s)
	}

	// Last part is sequence
	sequence, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return GlobalID{}, fmt.Errorf("invalid sequence in global ID: %w", err)
	}

	// Second to last and third to last are logical and physical time
	hlc := strings.Join(parts[len(parts)-3:len(parts)-1], "-")

	// Everything before that is regionID
	regionID := strings.Join(parts[:len(parts)-3], "-")

	return GlobalID{
		RegionID: regionID,
		HLC:      hlc,
		Sequence: sequence,
	}, nil
}

// String returns a string representation of the HLC
func (h *HLC) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return fmt.Sprintf("HLC{region=%s, node=%s, time=%d-%d, seq=%d}",
		h.regionID, h.nodeID, h.physicalTime, h.logicalTime, atomic.LoadInt64(&h.sequence))
}
