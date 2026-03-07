package clockdrift

import (
	"context"
	"testing"
	"time"
)

func TestDriftDetector_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NTPServer != "pool.ntp.org" {
		t.Errorf("NTPServer = %s, want pool.ntp.org", cfg.NTPServer)
	}
	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("CheckInterval = %v, want 30s", cfg.CheckInterval)
	}
	if cfg.Threshold != 500*time.Millisecond {
		t.Errorf("Threshold = %v, want 500ms", cfg.Threshold)
	}
	if cfg.MaxOffset != 10*time.Second {
		t.Errorf("MaxOffset = %v, want 10s", cfg.MaxOffset)
	}
}

func TestDriftDetector_NewDriftDetector(t *testing.T) {
	calibrateFunc := func(offset time.Duration) error {
		return nil
	}

	cfg := DefaultConfig()
	dd := NewDriftDetector(cfg, calibrateFunc)

	if dd == nil {
		t.Fatal("NewDriftDetector returned nil")
	}

	if dd.ntpServer != cfg.NTPServer {
		t.Errorf("ntpServer = %s, want %s", dd.ntpServer, cfg.NTPServer)
	}

	if dd.history == nil {
		t.Error("history buffer is nil")
	}

	if dd.history.Capacity() != cfg.HistoryCapacity {
		t.Errorf("history capacity = %d, want %d", dd.history.Capacity(), cfg.HistoryCapacity)
	}
}

func TestDriftDetector_GetHistory(t *testing.T) {
	cfg := DefaultConfig()
	dd := NewDriftDetector(cfg, nil)

	// Add some samples manually
	now := time.Now()
	samples := []ClockSample{
		{Timestamp: now.Add(-3 * time.Hour), Offset: 10 * time.Millisecond, Source: "ntp"},
		{Timestamp: now.Add(-2 * time.Hour), Offset: 20 * time.Millisecond, Source: "ntp"},
		{Timestamp: now.Add(-1 * time.Hour), Offset: 30 * time.Millisecond, Source: "ntp"},
		{Timestamp: now, Offset: 40 * time.Millisecond, Source: "ntp"},
	}

	for _, s := range samples {
		dd.history.Push(s)
	}

	// Get history for last 90 minutes
	history := dd.GetHistory(90 * time.Minute)

	// Should get last 2 samples
	if len(history) != 2 {
		t.Fatalf("len(GetHistory(90m)) = %d, want 2", len(history))
	}

	if history[0].Offset != 30*time.Millisecond {
		t.Errorf("history[0].Offset = %v, want 30ms", history[0].Offset)
	}
	if history[1].Offset != 40*time.Millisecond {
		t.Errorf("history[1].Offset = %v, want 40ms", history[1].Offset)
	}
}

func TestDriftDetector_GetLastOffset(t *testing.T) {
	cfg := DefaultConfig()
	dd := NewDriftDetector(cfg, nil)

	// Initially should be zero
	offset, checkTime := dd.GetLastOffset()
	if offset != 0 {
		t.Errorf("initial offset = %v, want 0", offset)
	}
	if !checkTime.IsZero() {
		t.Errorf("initial checkTime should be zero, got %v", checkTime)
	}

	// Simulate a check
	testOffset := 100 * time.Millisecond
	testTime := time.Now()

	dd.mu.Lock()
	dd.lastOffset = testOffset
	dd.lastCheck = testTime
	dd.mu.Unlock()

	offset, checkTime = dd.GetLastOffset()
	if offset != testOffset {
		t.Errorf("offset = %v, want %v", offset, testOffset)
	}
	if !checkTime.Equal(testTime) {
		t.Errorf("checkTime = %v, want %v", checkTime, testTime)
	}
}

func TestDriftDetector_CalibrateHLC(t *testing.T) {
	tests := []struct {
		name       string
		offset     time.Duration
		maxOffset  time.Duration
		wantErr    bool
		wantCalled bool
	}{
		{
			name:       "offset within threshold",
			offset:     100 * time.Millisecond,
			maxOffset:  10 * time.Second,
			wantErr:    false,
			wantCalled: true,
		},
		{
			name:       "offset exceeds max",
			offset:     15 * time.Second,
			maxOffset:  10 * time.Second,
			wantErr:    true,
			wantCalled: false,
		},
		{
			name:       "negative offset within threshold",
			offset:     -100 * time.Millisecond,
			maxOffset:  10 * time.Second,
			wantErr:    false,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calibrateCalled := false
			calibrateFunc := func(offset time.Duration) error {
				calibrateCalled = true
				return nil
			}

			cfg := DefaultConfig()
			cfg.MaxOffset = tt.maxOffset
			dd := NewDriftDetector(cfg, calibrateFunc)

			sample := ClockSample{
				Timestamp: time.Now(),
				Offset:    tt.offset,
				Source:    "test-ntp",
			}

			err := dd.CalibrateHLC(sample)

			if (err != nil) != tt.wantErr {
				t.Errorf("CalibrateHLC() error = %v, wantErr %v", err, tt.wantErr)
			}

			if calibrateCalled != tt.wantCalled {
				t.Errorf("calibrateFunc called = %v, want %v", calibrateCalled, tt.wantCalled)
			}
		})
	}
}

func TestDriftDetector_CalibrateHLC_NoFunction(t *testing.T) {
	cfg := DefaultConfig()
	dd := NewDriftDetector(cfg, nil)

	sample := ClockSample{
		Timestamp: time.Now(),
		Offset:    100 * time.Millisecond,
		Source:    "test-ntp",
	}

	err := dd.CalibrateHLC(sample)
	if err == nil {
		t.Error("CalibrateHLC() with nil function should return error")
	}
}

func TestDriftDetector_Start_ContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CheckInterval = 100 * time.Millisecond

	// Use a mock NTP server that doesn't exist to avoid actual network calls
	cfg.NTPServer = "localhost:9999"

	dd := NewDriftDetector(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start should return when context is cancelled
	err := dd.Start(ctx)

	// Should get context deadline exceeded or cancelled error
	if err == nil {
		t.Error("Start() should return error when context is cancelled")
	}
}

func TestDriftDetector_Metrics(t *testing.T) {
	cfg := DefaultConfig()
	dd := NewDriftDetector(cfg, nil)

	// Verify metrics are initialized
	if dd.currentOffsetGauge == nil {
		t.Error("currentOffsetGauge is nil")
	}
	if dd.calibrationCounter == nil {
		t.Error("calibrationCounter is nil")
	}
	if dd.thresholdBreachCounter == nil {
		t.Error("thresholdBreachCounter is nil")
	}
	if dd.checkLatencyHistogram == nil {
		t.Error("checkLatencyHistogram is nil")
	}
}
