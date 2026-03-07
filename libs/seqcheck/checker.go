// Package seqcheck provides message sequence checking and gap detection functionality
// for multi-region active-active IM systems.
package seqcheck

import (
	"fmt"
	"sync"
)

// GapRange represents a range of missing sequence numbers
type GapRange struct {
	ConversationID string `json:"conversation_id"`
	StartSeq       int64  `json:"start_seq"`
	EndSeq         int64  `json:"end_seq"`
}

// String returns a string representation of the gap range
func (g GapRange) String() string {
	if g.StartSeq == g.EndSeq {
		return fmt.Sprintf("[%d]", g.StartSeq)
	}
	return fmt.Sprintf("[%d-%d]", g.StartSeq, g.EndSeq)
}

// ConversationTracker tracks sequence numbers for a single conversation
type ConversationTracker struct {
	conversationID string
	maxSeq         int64
	received       map[int64]bool
	gaps           []GapRange
	retryCount     map[string]int // key: "startSeq-endSeq"
}

// newConversationTracker creates a new conversation tracker
func newConversationTracker(conversationID string) *ConversationTracker {
	return &ConversationTracker{
		conversationID: conversationID,
		maxSeq:         0,
		received:       make(map[int64]bool),
		gaps:           make([]GapRange, 0),
		retryCount:     make(map[string]int),
	}
}

// SequenceChecker checks message sequence numbers and detects gaps
type SequenceChecker struct {
	mu            sync.RWMutex
	conversations map[string]*ConversationTracker
	maxRetries    int
	gapCallback   func(gaps []GapRange)
}

// NewSequenceChecker creates a new sequence checker
func NewSequenceChecker(maxRetries int, gapCallback func(gaps []GapRange)) *SequenceChecker {
	if maxRetries <= 0 {
		maxRetries = 3 // default max retries
	}
	return &SequenceChecker{
		conversations: make(map[string]*ConversationTracker),
		maxRetries:    maxRetries,
		gapCallback:   gapCallback,
	}
}

// RecordSequence records a received message sequence number and returns newly detected gaps
func (sc *SequenceChecker) RecordSequence(conversationID string, seq int64) []GapRange {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	tracker := sc.getOrCreateTracker(conversationID)

	// If sequence number is less than or equal to maxSeq, mark as received
	if seq <= tracker.maxSeq {
		tracker.received[seq] = true
		sc.updateGaps(tracker)
		return nil
	}

	// If there's a gap between maxSeq and current seq
	var newGaps []GapRange
	if seq > tracker.maxSeq+1 {
		newGap := GapRange{
			ConversationID: conversationID,
			StartSeq:       tracker.maxSeq + 1,
			EndSeq:         seq - 1,
		}
		tracker.gaps = append(tracker.gaps, newGap)
		newGaps = append(newGaps, newGap)

		// Trigger callback if configured
		if sc.gapCallback != nil {
			go sc.gapCallback(newGaps)
		}
	}

	// Update maxSeq and mark current sequence as received
	tracker.maxSeq = seq
	tracker.received[seq] = true

	return newGaps
}

// GetGaps returns all unfilled gaps for a conversation
func (sc *SequenceChecker) GetGaps(conversationID string) []GapRange {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	tracker, exists := sc.conversations[conversationID]
	if !exists {
		return nil
	}

	// Return a copy to avoid external modification
	gaps := make([]GapRange, len(tracker.gaps))
	copy(gaps, tracker.gaps)
	return gaps
}

// FillGap marks a gap as filled (idempotent operation)
func (sc *SequenceChecker) FillGap(conversationID string, seq int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	tracker, exists := sc.conversations[conversationID]
	if !exists {
		return
	}

	// Mark sequence as received
	tracker.received[seq] = true

	// Update gaps to remove filled sequences
	sc.updateGaps(tracker)
}

// ShouldFullSync determines if a full sync should be triggered
// Returns true if any gap has failed retry attempts >= maxRetries
func (sc *SequenceChecker) ShouldFullSync(conversationID string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	tracker, exists := sc.conversations[conversationID]
	if !exists {
		return false
	}

	for _, gap := range tracker.gaps {
		key := gapKey(gap)
		if tracker.retryCount[key] >= sc.maxRetries {
			return true
		}
	}

	return false
}

// RecordRetryFailure records a failed retry attempt for a gap
func (sc *SequenceChecker) RecordRetryFailure(conversationID string, gap GapRange) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	tracker, exists := sc.conversations[conversationID]
	if !exists {
		return
	}

	key := gapKey(gap)
	tracker.retryCount[key]++
}

// getOrCreateTracker gets or creates a conversation tracker (must be called with lock held)
func (sc *SequenceChecker) getOrCreateTracker(conversationID string) *ConversationTracker {
	tracker, exists := sc.conversations[conversationID]
	if !exists {
		tracker = newConversationTracker(conversationID)
		sc.conversations[conversationID] = tracker
	}
	return tracker
}

// updateGaps updates the gaps list by removing filled sequences (must be called with lock held)
func (sc *SequenceChecker) updateGaps(tracker *ConversationTracker) {
	newGaps := make([]GapRange, 0, len(tracker.gaps))

	for _, gap := range tracker.gaps {
		// Check if all sequences in this gap are now received
		allFilled := true
		for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
			if !tracker.received[seq] {
				allFilled = false
				break
			}
		}

		if !allFilled {
			// Gap still exists, keep it
			newGaps = append(newGaps, gap)
		} else {
			// Gap is filled, remove retry count
			key := gapKey(gap)
			delete(tracker.retryCount, key)
		}
	}

	tracker.gaps = newGaps
}

// gapKey generates a unique key for a gap range
func gapKey(gap GapRange) string {
	return fmt.Sprintf("%d-%d", gap.StartSeq, gap.EndSeq)
}
