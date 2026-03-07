package clockdrift

import (
	"testing"
	"time"
)

func TestRingBuffer_NewRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{"positive capacity", 10, 10},
		{"zero capacity", 0, 100},      // default
		{"negative capacity", -5, 100}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRingBuffer(tt.capacity)
			if rb.Capacity() != tt.want {
				t.Errorf("Capacity() = %d, want %d", rb.Capacity(), tt.want)
			}
			if rb.Size() != 0 {
				t.Errorf("Size() = %d, want 0", rb.Size())
			}
		})
	}
}

func TestRingBuffer_PushAndGetAll(t *testing.T) {
	rb := NewRingBuffer(3)

	// Test empty buffer
	if samples := rb.GetAll(); samples != nil {
		t.Errorf("GetAll() on empty buffer = %v, want nil", samples)
	}

	// Push first sample
	sample1 := ClockSample{
		Timestamp: time.Now(),
		Offset:    10 * time.Millisecond,
		Source:    "ntp1",
	}
	rb.Push(sample1)

	samples := rb.GetAll()
	if len(samples) != 1 {
		t.Fatalf("len(GetAll()) = %d, want 1", len(samples))
	}
	if samples[0].Offset != sample1.Offset {
		t.Errorf("samples[0].Offset = %v, want %v", samples[0].Offset, sample1.Offset)
	}

	// Push second and third samples
	sample2 := ClockSample{
		Timestamp: time.Now().Add(time.Second),
		Offset:    20 * time.Millisecond,
		Source:    "ntp2",
	}
	sample3 := ClockSample{
		Timestamp: time.Now().Add(2 * time.Second),
		Offset:    30 * time.Millisecond,
		Source:    "ntp3",
	}
	rb.Push(sample2)
	rb.Push(sample3)

	samples = rb.GetAll()
	if len(samples) != 3 {
		t.Fatalf("len(GetAll()) = %d, want 3", len(samples))
	}

	// Verify chronological order
	if samples[0].Offset != sample1.Offset {
		t.Errorf("samples[0].Offset = %v, want %v", samples[0].Offset, sample1.Offset)
	}
	if samples[1].Offset != sample2.Offset {
		t.Errorf("samples[1].Offset = %v, want %v", samples[1].Offset, sample2.Offset)
	}
	if samples[2].Offset != sample3.Offset {
		t.Errorf("samples[2].Offset = %v, want %v", samples[2].Offset, sample3.Offset)
	}
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := NewRingBuffer(3)

	// Push 5 samples (more than capacity)
	samples := []ClockSample{
		{Timestamp: time.Now(), Offset: 10 * time.Millisecond, Source: "ntp1"},
		{Timestamp: time.Now().Add(time.Second), Offset: 20 * time.Millisecond, Source: "ntp2"},
		{Timestamp: time.Now().Add(2 * time.Second), Offset: 30 * time.Millisecond, Source: "ntp3"},
		{Timestamp: time.Now().Add(3 * time.Second), Offset: 40 * time.Millisecond, Source: "ntp4"},
		{Timestamp: time.Now().Add(4 * time.Second), Offset: 50 * time.Millisecond, Source: "ntp5"},
	}

	for _, s := range samples {
		rb.Push(s)
	}

	// Should only have last 3 samples
	result := rb.GetAll()
	if len(result) != 3 {
		t.Fatalf("len(GetAll()) = %d, want 3", len(result))
	}

	// Verify we have samples 3, 4, 5 (oldest samples 1, 2 were overwritten)
	if result[0].Offset != 30*time.Millisecond {
		t.Errorf("result[0].Offset = %v, want 30ms", result[0].Offset)
	}
	if result[1].Offset != 40*time.Millisecond {
		t.Errorf("result[1].Offset = %v, want 40ms", result[1].Offset)
	}
	if result[2].Offset != 50*time.Millisecond {
		t.Errorf("result[2].Offset = %v, want 50ms", result[2].Offset)
	}

	// Size should be capped at capacity
	if rb.Size() != 3 {
		t.Errorf("Size() = %d, want 3", rb.Size())
	}
}

func TestRingBuffer_GetSince(t *testing.T) {
	rb := NewRingBuffer(10)

	now := time.Now()
	samples := []ClockSample{
		{Timestamp: now.Add(-3 * time.Hour), Offset: 10 * time.Millisecond, Source: "ntp1"},
		{Timestamp: now.Add(-2 * time.Hour), Offset: 20 * time.Millisecond, Source: "ntp2"},
		{Timestamp: now.Add(-1 * time.Hour), Offset: 30 * time.Millisecond, Source: "ntp3"},
		{Timestamp: now.Add(-30 * time.Minute), Offset: 40 * time.Millisecond, Source: "ntp4"},
		{Timestamp: now, Offset: 50 * time.Millisecond, Source: "ntp5"},
	}

	for _, s := range samples {
		rb.Push(s)
	}

	// Get samples from last 90 minutes
	since := now.Add(-90 * time.Minute)
	result := rb.GetSince(since)

	// Should get samples 3, 4, 5
	if len(result) != 3 {
		t.Fatalf("len(GetSince()) = %d, want 3", len(result))
	}

	if result[0].Offset != 30*time.Millisecond {
		t.Errorf("result[0].Offset = %v, want 30ms", result[0].Offset)
	}
	if result[1].Offset != 40*time.Millisecond {
		t.Errorf("result[1].Offset = %v, want 40ms", result[1].Offset)
	}
	if result[2].Offset != 50*time.Millisecond {
		t.Errorf("result[2].Offset = %v, want 50ms", result[2].Offset)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)

	// Push some samples
	for i := 0; i < 3; i++ {
		rb.Push(ClockSample{
			Timestamp: time.Now(),
			Offset:    time.Duration(i) * time.Millisecond,
			Source:    "ntp",
		})
	}

	if rb.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", rb.Size())
	}

	// Clear buffer
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Size() after Clear() = %d, want 0", rb.Size())
	}

	if samples := rb.GetAll(); samples != nil {
		t.Errorf("GetAll() after Clear() = %v, want nil", samples)
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := NewRingBuffer(100)
	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				rb.Push(ClockSample{
					Timestamp: time.Now(),
					Offset:    time.Duration(id*100+j) * time.Millisecond,
					Source:    "ntp",
				})
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				rb.GetAll()
				rb.GetSince(time.Now().Add(-time.Hour))
				rb.Size()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Verify buffer is in valid state
	if rb.Size() > rb.Capacity() {
		t.Errorf("Size() = %d exceeds Capacity() = %d", rb.Size(), rb.Capacity())
	}
}
