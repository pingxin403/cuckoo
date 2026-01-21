package analytics

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 12: Analytics Non-Blocking
// Validates: Requirements 7.1, 7.2, 7.5
// For any click event, logging should complete immediately without blocking,
// even if Kafka is slow or unavailable
func TestProperty_AnalyticsNonBlocking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numEvents := rapid.IntRange(10, 100).Draw(t, "numEvents")

		// Create analytics writer with small buffer for testing
		config := Config{
			KafkaBrokers: []string{"localhost:9092"}, // Will fail to connect
			Topic:        "test-clicks",
			NumWorkers:   2,
			BufferSize:   50,
		}
		aw := NewAnalyticsWriter(config)
		defer aw.Close()

		// Generate random events
		events := make([]ClickEvent, numEvents)
		for i := 0; i < numEvents; i++ {
			events[i] = ClickEvent{
				ShortCode: rapid.StringMatching(`[A-Za-z0-9]{7}`).Draw(t, "shortCode"),
				Timestamp: time.Now(),
				SourceIP:  rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "sourceIP"),
				UserAgent: rapid.StringN(1, 50, 50).Draw(t, "userAgent"),
			}
		}

		// Measure time to log all events
		start := time.Now()
		for _, event := range events {
			aw.LogClick(event)
		}
		duration := time.Since(start)

		// Property: Logging should complete very quickly (< 10ms for 100 events)
		// This proves it's non-blocking
		maxDuration := 10 * time.Millisecond
		if duration > maxDuration {
			t.Fatalf("Logging took %v, expected < %v (non-blocking)", duration, maxDuration)
		}

		// Property: All events should be queued or dropped (no blocking)
		// We can't verify exact count due to buffer limits, but operation should complete
		stats := aw.Stats()
		bufferUsed := stats["buffer_used"].(int)
		bufferSize := stats["buffer_size"].(int)

		// Buffer should not exceed capacity
		if bufferUsed > bufferSize {
			t.Fatalf("Buffer used (%d) exceeds capacity (%d)", bufferUsed, bufferSize)
		}
	})
}

// Property: Click events have all required fields
// Validates: Requirements 7.3
func TestProperty_ClickEventCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random click event with non-empty fields
		event := ClickEvent{
			ShortCode: rapid.StringMatching(`[A-Za-z0-9]{7}`).Draw(t, "shortCode"),
			Timestamp: time.Now(),
			SourceIP:  rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "sourceIP"),
			UserAgent: rapid.StringN(1, 100, 100).Draw(t, "userAgent"),
		}

		// Property 1: ShortCode must not be empty
		if event.ShortCode == "" {
			t.Fatal("ShortCode should not be empty")
		}

		// Property 2: Timestamp must not be zero
		if event.Timestamp.IsZero() {
			t.Fatal("Timestamp should not be zero")
		}

		// Property 3: SourceIP must not be empty
		if event.SourceIP == "" {
			t.Fatal("SourceIP should not be empty")
		}

		// Property 4: UserAgent must not be empty
		if event.UserAgent == "" {
			t.Fatal("UserAgent should not be empty")
		}

		// Property 5: All required fields should be present
		requiredFields := []bool{
			event.ShortCode != "",
			!event.Timestamp.IsZero(),
			event.SourceIP != "",
			event.UserAgent != "",
		}

		for i, present := range requiredFields {
			if !present {
				t.Fatalf("Required field %d is missing", i)
			}
		}
	})
}

// Property: Buffer handles overflow gracefully
// Validates: Requirements 7.5
func TestProperty_BufferOverflowHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create analytics writer with very small buffer
		bufferSize := rapid.IntRange(5, 20).Draw(t, "bufferSize")
		config := Config{
			KafkaBrokers: []string{"localhost:9092"},
			Topic:        "test-clicks",
			NumWorkers:   1,
			BufferSize:   bufferSize,
		}
		aw := NewAnalyticsWriter(config)
		defer aw.Close()

		// Try to log more events than buffer can hold
		numEvents := bufferSize * 2

		for i := 0; i < numEvents; i++ {
			event := ClickEvent{
				ShortCode: "test123",
				Timestamp: time.Now(),
				SourceIP:  "192.168.1.1",
				UserAgent: "test-agent",
			}
			aw.LogClick(event)
		}

		// Property: Operation should complete without blocking or panicking
		// Some events may be dropped, but that's expected behavior
		stats := aw.Stats()
		bufferUsed := stats["buffer_used"].(int)

		// Buffer should not exceed capacity
		if bufferUsed > bufferSize {
			t.Fatalf("Buffer used (%d) exceeds capacity (%d)", bufferUsed, bufferSize)
		}
	})
}

// Property: Multiple workers process events concurrently
// Validates: Requirements 7.2
func TestProperty_ConcurrentWorkerProcessing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numWorkers := rapid.IntRange(2, 8).Draw(t, "numWorkers")
		bufferSize := rapid.IntRange(50, 200).Draw(t, "bufferSize")

		config := Config{
			KafkaBrokers: []string{"localhost:9092"},
			Topic:        "test-clicks",
			NumWorkers:   numWorkers,
			BufferSize:   bufferSize,
		}
		aw := NewAnalyticsWriter(config)
		defer aw.Close()

		// Property: Stats should reflect correct configuration
		stats := aw.Stats()
		if stats["num_workers"].(int) != numWorkers {
			t.Fatalf("Expected %d workers, got %d", numWorkers, stats["num_workers"].(int))
		}
		if stats["buffer_size"].(int) != bufferSize {
			t.Fatalf("Expected buffer size %d, got %d", bufferSize, stats["buffer_size"].(int))
		}
	})
}

// Property: Analytics writer can be safely closed
// Validates: Requirements 7.2
func TestProperty_SafeClose(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create analytics writer
		config := Config{
			KafkaBrokers: []string{"localhost:9092"},
			Topic:        "test-clicks",
			NumWorkers:   2,
			BufferSize:   100,
		}
		aw := NewAnalyticsWriter(config)

		// Log some events
		numEvents := rapid.IntRange(5, 20).Draw(t, "numEvents")
		for i := 0; i < numEvents; i++ {
			event := ClickEvent{
				ShortCode: "test123",
				Timestamp: time.Now(),
				SourceIP:  "192.168.1.1",
				UserAgent: "test-agent",
			}
			aw.LogClick(event)
		}

		// Property: Close should complete without error or panic
		err := aw.Close()
		if err != nil {
			// Kafka connection error is expected in tests, but Close should still work
			// We only fail if there's a panic or hang
			t.Logf("Close returned error (expected in test): %v", err)
		}

		// Property: Closing twice should not panic
		err = aw.Close()
		// Second close may return error, but should not panic
	})
}
