// Package clockdrift provides clock drift detection and calibration functionality
// for multi-region active-active systems.
package clockdrift

import (
	"sync"
	"time"
)

// ClockSample represents a single clock drift measurement
type ClockSample struct {
	Timestamp time.Time     `json:"timestamp"` // When the sample was taken
	Offset    time.Duration `json:"offset"`    // Offset from NTP server
	Source    string        `json:"source"`    // NTP server address
}

// RingBuffer is a thread-safe circular buffer for storing clock samples
type RingBuffer struct {
	mu       sync.RWMutex
	capacity int
	samples  []ClockSample
	head     int // Index of the next write position
	size     int // Current number of samples
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 100 // default capacity
	}
	return &RingBuffer{
		capacity: capacity,
		samples:  make([]ClockSample, capacity),
		head:     0,
		size:     0,
	}
}

// Push adds a new sample to the buffer
// If the buffer is full, the oldest sample is overwritten
func (rb *RingBuffer) Push(sample ClockSample) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.samples[rb.head] = sample
	rb.head = (rb.head + 1) % rb.capacity

	if rb.size < rb.capacity {
		rb.size++
	}
}

// GetAll returns all samples in chronological order (oldest first)
func (rb *RingBuffer) GetAll() []ClockSample {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return nil
	}

	result := make([]ClockSample, rb.size)

	if rb.size < rb.capacity {
		// Buffer not full yet, samples are from 0 to size-1
		copy(result, rb.samples[:rb.size])
	} else {
		// Buffer is full, samples wrap around
		// Copy from head to end (oldest samples)
		n := copy(result, rb.samples[rb.head:])
		// Copy from start to head (newest samples)
		copy(result[n:], rb.samples[:rb.head])
	}

	return result
}

// GetSince returns all samples since the specified time
func (rb *RingBuffer) GetSince(since time.Time) []ClockSample {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return nil
	}

	allSamples := rb.getAllUnsafe()
	result := make([]ClockSample, 0, rb.size)

	for _, sample := range allSamples {
		if sample.Timestamp.After(since) || sample.Timestamp.Equal(since) {
			result = append(result, sample)
		}
	}

	return result
}

// Size returns the current number of samples in the buffer
func (rb *RingBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Capacity returns the maximum capacity of the buffer
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}

// Clear removes all samples from the buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.size = 0
}

// getAllUnsafe returns all samples without locking (must be called with lock held)
func (rb *RingBuffer) getAllUnsafe() []ClockSample {
	if rb.size == 0 {
		return nil
	}

	result := make([]ClockSample, rb.size)

	if rb.size < rb.capacity {
		copy(result, rb.samples[:rb.size])
	} else {
		n := copy(result, rb.samples[rb.head:])
		copy(result[n:], rb.samples[:rb.head])
	}

	return result
}
