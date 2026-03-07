package clockdrift

import (
	"sort"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 3: 时钟偏移历史环形缓冲区容量不变性
// For any number of Push operations, the buffer size should never exceed capacity,
// and GetAll should return samples in chronological order (oldest first).
// Validates: Requirements 8.2.3
func TestProperty_RingBufferCapacityInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random capacity
		capacity := rapid.IntRange(1, 100).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		// Generate random number of samples to push (may exceed capacity)
		numSamples := rapid.IntRange(0, capacity*3).Draw(t, "numSamples")

		// Generate and push samples
		baseTime := time.Now()
		for i := 0; i < numSamples; i++ {
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(rapid.Int64Range(-1000, 1000).Draw(t, "offset")) * time.Millisecond,
				Source:    "ntp.test.com",
			}
			rb.Push(sample)

			// Verify size never exceeds capacity
			currentSize := rb.Size()
			if currentSize > capacity {
				t.Fatalf("Buffer size %d exceeds capacity %d after %d pushes",
					currentSize, capacity, i+1)
			}
		}

		// Verify final size
		finalSize := rb.Size()
		if finalSize > capacity {
			t.Fatalf("Final buffer size %d exceeds capacity %d", finalSize, capacity)
		}

		// Verify expected size
		expectedSize := numSamples
		if expectedSize > capacity {
			expectedSize = capacity
		}
		if finalSize != expectedSize {
			t.Fatalf("Expected size %d, got %d", expectedSize, finalSize)
		}

		// Verify GetAll returns samples in chronological order
		samples := rb.GetAll()
		if len(samples) != finalSize {
			t.Fatalf("GetAll returned %d samples, expected %d", len(samples), finalSize)
		}

		// Check chronological order
		for i := 1; i < len(samples); i++ {
			if samples[i].Timestamp.Before(samples[i-1].Timestamp) {
				t.Fatalf("Samples not in chronological order: sample[%d]=%v comes before sample[%d]=%v",
					i, samples[i].Timestamp, i-1, samples[i-1].Timestamp)
			}
		}
	})
}

// Property: Ring buffer should maintain FIFO order
func TestProperty_RingBufferFIFOOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 50).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		// Push more samples than capacity to test wraparound
		numSamples := rapid.IntRange(capacity+1, capacity*2).Draw(t, "numSamples")

		baseTime := time.Now()
		pushedSamples := make([]ClockSample, numSamples)

		for i := 0; i < numSamples; i++ {
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(i) * time.Millisecond,
				Source:    "ntp.test.com",
			}
			pushedSamples[i] = sample
			rb.Push(sample)
		}

		// Get all samples
		retrievedSamples := rb.GetAll()

		// Should have exactly capacity samples (oldest ones dropped)
		if len(retrievedSamples) != capacity {
			t.Fatalf("Expected %d samples, got %d", capacity, len(retrievedSamples))
		}

		// Verify we got the most recent samples
		expectedStart := numSamples - capacity
		for i := 0; i < capacity; i++ {
			expected := pushedSamples[expectedStart+i]
			actual := retrievedSamples[i]

			if !actual.Timestamp.Equal(expected.Timestamp) {
				t.Fatalf("Sample %d mismatch: expected timestamp %v, got %v",
					i, expected.Timestamp, actual.Timestamp)
			}
			if actual.Offset != expected.Offset {
				t.Fatalf("Sample %d mismatch: expected offset %v, got %v",
					i, expected.Offset, actual.Offset)
			}
		}
	})
}

// Property: GetSince should return only samples after the specified time
func TestProperty_GetSinceFiltersCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(10, 50).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		numSamples := rapid.IntRange(5, capacity).Draw(t, "numSamples")
		baseTime := time.Now()

		// Push samples
		for i := 0; i < numSamples; i++ {
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(i) * time.Millisecond,
				Source:    "ntp.test.com",
			}
			rb.Push(sample)
		}

		// Choose a random cutoff time
		cutoffIndex := rapid.IntRange(0, numSamples-1).Draw(t, "cutoffIndex")
		cutoffTime := baseTime.Add(time.Duration(cutoffIndex) * time.Second)

		// Get samples since cutoff
		filteredSamples := rb.GetSince(cutoffTime)

		// Verify all returned samples are >= cutoffTime
		for i, sample := range filteredSamples {
			if sample.Timestamp.Before(cutoffTime) {
				t.Fatalf("Sample %d has timestamp %v before cutoff %v",
					i, sample.Timestamp, cutoffTime)
			}
		}

		// Verify we got the expected number of samples
		expectedCount := numSamples - cutoffIndex
		if len(filteredSamples) != expectedCount {
			t.Fatalf("Expected %d samples since cutoff, got %d",
				expectedCount, len(filteredSamples))
		}

		// Verify samples are still in chronological order
		for i := 1; i < len(filteredSamples); i++ {
			if filteredSamples[i].Timestamp.Before(filteredSamples[i-1].Timestamp) {
				t.Fatalf("Filtered samples not in chronological order at index %d", i)
			}
		}
	})
}

// Property: Clear should reset buffer to empty state
func TestProperty_ClearResetsBuffer(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 50).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		// Push random number of samples
		numSamples := rapid.IntRange(1, capacity*2).Draw(t, "numSamples")
		baseTime := time.Now()

		for i := 0; i < numSamples; i++ {
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(i) * time.Millisecond,
				Source:    "ntp.test.com",
			}
			rb.Push(sample)
		}

		// Verify buffer has samples
		if rb.Size() == 0 {
			t.Fatal("Buffer should have samples before clear")
		}

		// Clear buffer
		rb.Clear()

		// Verify buffer is empty
		if rb.Size() != 0 {
			t.Fatalf("Buffer size should be 0 after clear, got %d", rb.Size())
		}

		samples := rb.GetAll()
		if len(samples) != 0 {
			t.Fatalf("GetAll should return empty slice after clear, got %d samples", len(samples))
		}

		sinceTime := baseTime.Add(-1 * time.Hour)
		sinceSamples := rb.GetSince(sinceTime)
		if len(sinceSamples) != 0 {
			t.Fatalf("GetSince should return empty slice after clear, got %d samples", len(sinceSamples))
		}
	})
}

// Property: Concurrent access should be safe
func TestProperty_ConcurrentAccessSafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(10, 50).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		numWriters := rapid.IntRange(2, 5).Draw(t, "numWriters")
		numReaders := rapid.IntRange(2, 5).Draw(t, "numReaders")
		samplesPerWriter := rapid.IntRange(10, 50).Draw(t, "samplesPerWriter")

		done := make(chan bool)

		// Start writers
		baseTime := time.Now()
		for w := 0; w < numWriters; w++ {
			go func(writerID int) {
				for i := 0; i < samplesPerWriter; i++ {
					offsetMs := int64(writerID*samplesPerWriter + i)
					sample := ClockSample{
						Timestamp: baseTime.Add(time.Duration(offsetMs) * time.Millisecond),
						Offset:    time.Duration(writerID*1000+i) * time.Microsecond,
						Source:    "ntp.test.com",
					}
					rb.Push(sample)
				}
				done <- true
			}(w)
		}

		// Start readers
		for r := 0; r < numReaders; r++ {
			go func() {
				for i := 0; i < samplesPerWriter; i++ {
					_ = rb.GetAll()
					_ = rb.Size()
					_ = rb.GetSince(time.Now().Add(-1 * time.Hour))
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numWriters+numReaders; i++ {
			<-done
		}

		// Verify buffer state is consistent
		size := rb.Size()
		samples := rb.GetAll()

		if len(samples) != size {
			t.Fatalf("Inconsistent state: Size()=%d but GetAll() returned %d samples",
				size, len(samples))
		}

		if size > capacity {
			t.Fatalf("Buffer size %d exceeds capacity %d after concurrent access",
				size, capacity)
		}

		// Note: We don't check chronological order here because the RingBuffer
		// doesn't guarantee ordering when samples are pushed concurrently.
		// The buffer only guarantees thread-safety (no data races).
	})
}

// Property: Buffer should handle edge cases correctly
func TestProperty_EdgeCases(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, 10).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		// Test empty buffer
		if rb.Size() != 0 {
			t.Fatal("New buffer should be empty")
		}
		if rb.GetAll() != nil {
			t.Fatal("GetAll on empty buffer should return nil")
		}
		if rb.GetSince(time.Now()) != nil {
			t.Fatal("GetSince on empty buffer should return nil")
		}

		// Push exactly capacity samples
		baseTime := time.Now()
		for i := 0; i < capacity; i++ {
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(i) * time.Millisecond,
				Source:    "ntp.test.com",
			}
			rb.Push(sample)
		}

		// Verify size equals capacity
		if rb.Size() != capacity {
			t.Fatalf("Expected size %d, got %d", capacity, rb.Size())
		}

		// Push one more to trigger wraparound
		extraSample := ClockSample{
			Timestamp: baseTime.Add(time.Duration(capacity) * time.Second),
			Offset:    time.Duration(capacity) * time.Millisecond,
			Source:    "ntp.test.com",
		}
		rb.Push(extraSample)

		// Size should still be capacity
		if rb.Size() != capacity {
			t.Fatalf("Size should remain %d after wraparound, got %d", capacity, rb.Size())
		}

		// Oldest sample should be dropped
		samples := rb.GetAll()
		if samples[0].Timestamp.Equal(baseTime) {
			t.Fatal("Oldest sample should have been dropped after wraparound")
		}
		if !samples[len(samples)-1].Timestamp.Equal(extraSample.Timestamp) {
			t.Fatal("Newest sample should be at the end")
		}
	})
}

// Property: Samples should maintain their data integrity
func TestProperty_DataIntegrity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 20).Draw(t, "capacity")
		rb := NewRingBuffer(capacity)

		numSamples := rapid.IntRange(1, capacity).Draw(t, "numSamples")
		baseTime := time.Now()

		// Generate unique samples
		type sampleKey struct {
			timestamp int64
			offset    int64
		}
		pushedSamples := make(map[sampleKey]ClockSample)

		for i := 0; i < numSamples; i++ {
			offset := rapid.Int64Range(-10000, 10000).Draw(t, "offset")
			sample := ClockSample{
				Timestamp: baseTime.Add(time.Duration(i) * time.Second),
				Offset:    time.Duration(offset) * time.Millisecond,
				Source:    "ntp.test.com",
			}

			key := sampleKey{
				timestamp: sample.Timestamp.Unix(),
				offset:    int64(sample.Offset),
			}
			pushedSamples[key] = sample
			rb.Push(sample)
		}

		// Retrieve samples
		retrievedSamples := rb.GetAll()

		// Verify all retrieved samples match pushed samples
		for _, retrieved := range retrievedSamples {
			key := sampleKey{
				timestamp: retrieved.Timestamp.Unix(),
				offset:    int64(retrieved.Offset),
			}

			original, exists := pushedSamples[key]
			if !exists {
				t.Fatalf("Retrieved sample not found in pushed samples: %v", retrieved)
			}

			if !retrieved.Timestamp.Equal(original.Timestamp) {
				t.Fatalf("Timestamp mismatch: expected %v, got %v",
					original.Timestamp, retrieved.Timestamp)
			}
			if retrieved.Offset != original.Offset {
				t.Fatalf("Offset mismatch: expected %v, got %v",
					original.Offset, retrieved.Offset)
			}
			if retrieved.Source != original.Source {
				t.Fatalf("Source mismatch: expected %v, got %v",
					original.Source, retrieved.Source)
			}
		}
	})
}

// Helper function to verify samples are sorted by timestamp
func isSortedByTimestamp(samples []ClockSample) bool {
	return sort.SliceIsSorted(samples, func(i, j int) bool {
		return samples[i].Timestamp.Before(samples[j].Timestamp) ||
			samples[i].Timestamp.Equal(samples[j].Timestamp)
	})
}
