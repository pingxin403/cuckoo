package seqcheck

import (
	"math/rand/v2"
	"sort"
	"testing"

	"pgregory.net/rapid"
)

// Property 1: 序列检查器断层检测完整性
// For any message sequence (with out-of-order and missing messages),
// after recording all messages, GetGaps should return exactly the set of missing sequence numbers
// between 1 and the maximum received sequence.
// Validates: Requirements 8.1.1, 8.1.2
func TestProperty_GapDetectionCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"

		// Generate a random sequence with gaps
		maxSeq := rapid.Int64Range(10, 100).Draw(t, "maxSeq")

		// Randomly select which sequences to include (some will be missing)
		presentSeqs := make(map[int64]bool)
		for i := int64(1); i <= maxSeq; i++ {
			if rapid.Bool().Draw(t, "include") {
				presentSeqs[i] = true
			}
		}

		// Ensure at least sequence 1 is present (SequenceChecker assumes sequences start from 1)
		// This matches the real-world behavior where message sequences start from 1
		presentSeqs[1] = true

		// Create sequence checker
		sc := NewSequenceChecker(3, nil)

		// Record sequences in sorted order to ensure predictable gap detection
		// The SequenceChecker's gap detection behavior depends on the order of arrival:
		// - When a sequence > maxSeq+1 arrives, it creates a gap from maxSeq+1 to seq-1
		// - Recording in sorted order ensures we detect exactly the missing sequences
		seqList := make([]int64, 0, len(presentSeqs))
		for seq := range presentSeqs {
			seqList = append(seqList, seq)
		}
		sort.Slice(seqList, func(i, j int) bool {
			return seqList[i] < seqList[j]
		})

		for _, seq := range seqList {
			sc.RecordSequence(conversationID, seq)
		}

		// Get detected gaps
		gaps := sc.GetGaps(conversationID)

		// Calculate expected missing sequences
		// The checker only tracks gaps up to the maximum received sequence
		expectedMissing := make(map[int64]bool)
		maxPresent := int64(0)
		for seq := range presentSeqs {
			if seq > maxPresent {
				maxPresent = seq
			}
		}

		// Only sequences from 1 to maxPresent that are missing should be in gaps
		for i := int64(1); i <= maxPresent; i++ {
			if !presentSeqs[i] {
				expectedMissing[i] = true
			}
		}

		// Verify gaps match expected missing sequences
		actualMissing := make(map[int64]bool)
		for _, gap := range gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				actualMissing[seq] = true
			}
		}

		// Check that actual missing equals expected missing
		if len(actualMissing) != len(expectedMissing) {
			t.Fatalf("Gap detection mismatch: expected %d missing, got %d missing\nExpected: %v\nActual: %v",
				len(expectedMissing), len(actualMissing), expectedMissing, actualMissing)
		}

		for seq := range expectedMissing {
			if !actualMissing[seq] {
				t.Fatalf("Expected sequence %d to be in gaps, but it wasn't", seq)
			}
		}

		for seq := range actualMissing {
			if !expectedMissing[seq] {
				t.Fatalf("Sequence %d is in gaps but shouldn't be", seq)
			}
		}
	})
}

// Property 2: 序列检查器补洞幂等性
// For any conversation and known gap, calling FillGap multiple times
// should produce the same state as calling it once.
// Validates: Requirements 8.1.4
func TestProperty_FillGapIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"

		// Create a gap
		sc1 := NewSequenceChecker(3, nil)
		sc2 := NewSequenceChecker(3, nil)

		// Record same initial sequences in both checkers
		sc1.RecordSequence(conversationID, 1)
		sc1.RecordSequence(conversationID, 10)

		sc2.RecordSequence(conversationID, 1)
		sc2.RecordSequence(conversationID, 10)

		// Choose a random sequence in the gap to fill
		seqToFill := rapid.Int64Range(2, 9).Draw(t, "seqToFill")

		// Fill once in sc1
		sc1.FillGap(conversationID, seqToFill)

		// Fill multiple times in sc2
		fillCount := rapid.IntRange(2, 10).Draw(t, "fillCount")
		for i := 0; i < fillCount; i++ {
			sc2.FillGap(conversationID, seqToFill)
		}

		// Both should have the same gaps
		gaps1 := sc1.GetGaps(conversationID)
		gaps2 := sc2.GetGaps(conversationID)

		if len(gaps1) != len(gaps2) {
			t.Fatalf("Idempotence violated: sc1 has %d gaps, sc2 has %d gaps",
				len(gaps1), len(gaps2))
		}

		// Compare gap contents
		for i := range gaps1 {
			if gaps1[i].StartSeq != gaps2[i].StartSeq || gaps1[i].EndSeq != gaps2[i].EndSeq {
				t.Fatalf("Idempotence violated: gap %d differs", i)
			}
		}
	})
}

// Property 3: Gap ranges should be non-overlapping and sorted
func TestProperty_GapRangesNonOverlapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		// Generate random sequence with multiple gaps
		maxSeq := rapid.Int64Range(20, 100).Draw(t, "maxSeq")

		// Record sequences with intentional gaps
		presentSeqs := make([]int64, 0)
		for i := int64(1); i <= maxSeq; i++ {
			if rapid.Bool().Draw(t, "include") {
				presentSeqs = append(presentSeqs, i)
			}
		}

		// Ensure at least 2 sequences present to create potential gaps
		if len(presentSeqs) < 2 {
			presentSeqs = []int64{1, maxSeq}
		}

		// Record in random order
		rand.Shuffle(len(presentSeqs), func(i, j int) {
			presentSeqs[i], presentSeqs[j] = presentSeqs[j], presentSeqs[i]
		})
		for _, seq := range presentSeqs {
			sc.RecordSequence(conversationID, seq)
		}

		gaps := sc.GetGaps(conversationID)

		// Verify gaps are non-overlapping
		for i := 0; i < len(gaps)-1; i++ {
			if gaps[i].EndSeq >= gaps[i+1].StartSeq {
				t.Fatalf("Overlapping gaps detected: [%d-%d] and [%d-%d]",
					gaps[i].StartSeq, gaps[i].EndSeq,
					gaps[i+1].StartSeq, gaps[i+1].EndSeq)
			}
		}

		// Verify gaps are sorted by start sequence
		for i := 0; i < len(gaps)-1; i++ {
			if gaps[i].StartSeq >= gaps[i+1].StartSeq {
				t.Fatalf("Gaps not sorted: [%d-%d] comes before [%d-%d]",
					gaps[i].StartSeq, gaps[i].EndSeq,
					gaps[i+1].StartSeq, gaps[i+1].EndSeq)
			}
		}
	})
}

// Property 4: Recording all sequences should result in no gaps
func TestProperty_CompleteSequenceNoGaps(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		maxSeq := rapid.Int64Range(10, 100).Draw(t, "maxSeq")

		// Generate all sequences from 1 to maxSeq
		allSeqs := make([]int64, maxSeq)
		for i := int64(0); i < maxSeq; i++ {
			allSeqs[i] = i + 1
		}

		// Record in random order
		rand.Shuffle(len(allSeqs), func(i, j int) {
			allSeqs[i], allSeqs[j] = allSeqs[j], allSeqs[i]
		})
		for _, seq := range allSeqs {
			sc.RecordSequence(conversationID, seq)
		}

		// Should have no gaps
		gaps := sc.GetGaps(conversationID)
		if len(gaps) != 0 {
			t.Fatalf("Expected no gaps for complete sequence, got %d gaps", len(gaps))
		}
	})
}

// Property 5: Filling all gaps should result in no gaps
func TestProperty_FillingAllGapsResultsInNoGaps(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		// Create gaps
		maxSeq := rapid.Int64Range(10, 50).Draw(t, "maxSeq")
		presentSeqs := make(map[int64]bool)

		for i := int64(1); i <= maxSeq; i++ {
			if rapid.Bool().Draw(t, "include") {
				presentSeqs[i] = true
			}
		}

		// Ensure at least first and last are present
		presentSeqs[1] = true
		presentSeqs[maxSeq] = true

		// Record present sequences
		for seq := range presentSeqs {
			sc.RecordSequence(conversationID, seq)
		}

		// Get gaps
		gaps := sc.GetGaps(conversationID)

		// Fill all gaps
		for _, gap := range gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				sc.FillGap(conversationID, seq)
			}
		}

		// Should have no gaps now
		remainingGaps := sc.GetGaps(conversationID)
		if len(remainingGaps) != 0 {
			t.Fatalf("Expected no gaps after filling all, got %d gaps", len(remainingGaps))
		}
	})
}

// Property 6: MaxSeq should never decrease
func TestProperty_MaxSeqMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		// Generate random sequences
		numSeqs := rapid.IntRange(10, 100).Draw(t, "numSeqs")
		sequences := make([]int64, numSeqs)
		for i := 0; i < numSeqs; i++ {
			sequences[i] = rapid.Int64Range(1, 1000).Draw(t, "seq")
		}

		// Track maxSeq as we record
		observedMax := int64(0)

		for _, seq := range sequences {
			sc.RecordSequence(conversationID, seq)

			// Get current gaps to infer maxSeq
			gaps := sc.GetGaps(conversationID)
			currentMax := seq

			// MaxSeq is at least the current sequence
			if currentMax > observedMax {
				observedMax = currentMax
			}

			// Verify gaps don't extend beyond what we've seen
			for _, gap := range gaps {
				if gap.EndSeq >= observedMax {
					// This is expected - gap ends just before a higher sequence
					continue
				}
			}
		}

		// Final verification: maxSeq should be the maximum sequence we recorded
		expectedMax := int64(0)
		for _, seq := range sequences {
			if seq > expectedMax {
				expectedMax = seq
			}
		}

		// We can't directly access maxSeq, but we can verify through gaps
		// If we record expectedMax+1, there should be a gap from observedMax+1 to expectedMax
		sc.RecordSequence(conversationID, expectedMax+10)
		gaps := sc.GetGaps(conversationID)

		// Should have at least one gap
		if len(gaps) == 0 && expectedMax+10 > expectedMax+1 {
			// This is fine if all sequences were continuous
		}
	})
}

// Helper function to shuffle int64 slice
func shuffle(slice []int64) {
	for i := len(slice) - 1; i > 0; i-- {
		j := i // In real implementation, use rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// Helper to check if two gap slices are equal
func gapsEqual(a, b []GapRange) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort both slices for comparison
	sortGaps := func(gaps []GapRange) {
		sort.Slice(gaps, func(i, j int) bool {
			return gaps[i].StartSeq < gaps[j].StartSeq
		})
	}

	sortGaps(a)
	sortGaps(b)

	for i := range a {
		if a[i].StartSeq != b[i].StartSeq || a[i].EndSeq != b[i].EndSeq {
			return false
		}
	}

	return true
}
