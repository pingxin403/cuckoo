package analytics

import (
	"testing"
	"time"
)

// Test event buffering
// Requirements: 7.2
func TestAnalyticsWriter_EventBuffering(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   2,
		BufferSize:   10,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	// Log events
	for i := 0; i < 5; i++ {
		event := ClickEvent{
			ShortCode: "test123",
			Timestamp: time.Now(),
			SourceIP:  "192.168.1.1",
			UserAgent: "test-agent",
		}
		aw.LogClick(event)
	}

	// Check stats
	stats := aw.Stats()
	bufferUsed := stats["buffer_used"].(int)

	// Buffer should have events (may be less than 5 if workers processed some)
	if bufferUsed < 0 || bufferUsed > 5 {
		t.Fatalf("Expected buffer_used between 0 and 5, got %d", bufferUsed)
	}
}

// Test worker pool processing
// Requirements: 7.2
func TestAnalyticsWriter_WorkerPool(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   4,
		BufferSize:   100,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	// Verify worker count
	stats := aw.Stats()
	if stats["num_workers"].(int) != 4 {
		t.Fatalf("Expected 4 workers, got %d", stats["num_workers"].(int))
	}
}

// Test Kafka failure handling
// Requirements: 7.5
func TestAnalyticsWriter_KafkaFailureHandling(t *testing.T) {
	// Use invalid Kafka broker to simulate failure
	config := Config{
		KafkaBrokers: []string{"invalid-broker:9092"},
		Topic:        "test-clicks",
		NumWorkers:   1,
		BufferSize:   10,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	// Log event - should not panic even if Kafka is unavailable
	event := ClickEvent{
		ShortCode: "test123",
		Timestamp: time.Now(),
		SourceIP:  "192.168.1.1",
		UserAgent: "test-agent",
	}
	aw.LogClick(event)

	// Give worker time to attempt write
	time.Sleep(100 * time.Millisecond)

	// Should not panic - failure is handled gracefully
}

// Test buffer full scenario
// Requirements: 7.5
func TestAnalyticsWriter_BufferFull(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   0, // No workers to process events
		BufferSize:   5,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	// Fill buffer beyond capacity
	for i := 0; i < 10; i++ {
		event := ClickEvent{
			ShortCode: "test123",
			Timestamp: time.Now(),
			SourceIP:  "192.168.1.1",
			UserAgent: "test-agent",
		}
		aw.LogClick(event)
	}

	// Check stats
	stats := aw.Stats()
	bufferUsed := stats["buffer_used"].(int)
	bufferSize := stats["buffer_size"].(int)

	// Buffer should not exceed capacity
	if bufferUsed > bufferSize {
		t.Fatalf("Buffer used (%d) exceeds capacity (%d)", bufferUsed, bufferSize)
	}
}

// Test graceful shutdown
// Requirements: 7.2
func TestAnalyticsWriter_GracefulShutdown(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   2,
		BufferSize:   100,
	}
	aw := NewAnalyticsWriter(config)

	// Log some events
	for i := 0; i < 10; i++ {
		event := ClickEvent{
			ShortCode: "test123",
			Timestamp: time.Now(),
			SourceIP:  "192.168.1.1",
			UserAgent: "test-agent",
		}
		aw.LogClick(event)
	}

	// Close should complete without hanging
	done := make(chan error, 1)
	go func() {
		done <- aw.Close()
	}()

	select {
	case err := <-done:
		// Close completed (may have error due to Kafka connection, but should not hang)
		if err != nil {
			t.Logf("Close returned error (expected in test): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not complete within 5 seconds")
	}
}

// Test stats reporting
// Requirements: 7.2
func TestAnalyticsWriter_Stats(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   3,
		BufferSize:   50,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	stats := aw.Stats()

	// Verify stats structure
	if stats["buffer_size"].(int) != 50 {
		t.Fatalf("Expected buffer_size 50, got %d", stats["buffer_size"].(int))
	}
	if stats["num_workers"].(int) != 3 {
		t.Fatalf("Expected num_workers 3, got %d", stats["num_workers"].(int))
	}

	// Buffer utilization should be between 0 and 1
	utilization := stats["buffer_utilization"].(float64)
	if utilization < 0 || utilization > 1 {
		t.Fatalf("Expected buffer_utilization between 0 and 1, got %f", utilization)
	}
}

// Test default configuration values
// Requirements: 7.2
func TestAnalyticsWriter_DefaultConfig(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		// NumWorkers, BufferSize, Topic not specified - should use defaults
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	stats := aw.Stats()

	// Check defaults
	if stats["num_workers"].(int) != 4 {
		t.Fatalf("Expected default num_workers 4, got %d", stats["num_workers"].(int))
	}
	if stats["buffer_size"].(int) != 10000 {
		t.Fatalf("Expected default buffer_size 10000, got %d", stats["buffer_size"].(int))
	}
}

// Test click event structure
// Requirements: 7.3
func TestClickEvent_Structure(t *testing.T) {
	event := ClickEvent{
		ShortCode: "abc1234",
		Timestamp: time.Now(),
		SourceIP:  "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		Referer:   "https://example.com",
	}

	// Verify all fields are set
	if event.ShortCode == "" {
		t.Fatal("ShortCode should not be empty")
	}
	if event.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}
	if event.SourceIP == "" {
		t.Fatal("SourceIP should not be empty")
	}
	if event.UserAgent == "" {
		t.Fatal("UserAgent should not be empty")
	}
}

// Test concurrent logging
// Requirements: 7.2
func TestAnalyticsWriter_ConcurrentLogging(t *testing.T) {
	config := Config{
		KafkaBrokers: []string{"localhost:9092"},
		Topic:        "test-clicks",
		NumWorkers:   4,
		BufferSize:   100,
	}
	aw := NewAnalyticsWriter(config)
	defer func() { _ = aw.Close() }()

	// Log events concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				event := ClickEvent{
					ShortCode: "test123",
					Timestamp: time.Now(),
					SourceIP:  "192.168.1.1",
					UserAgent: "test-agent",
				}
				aw.LogClick(event)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
}
