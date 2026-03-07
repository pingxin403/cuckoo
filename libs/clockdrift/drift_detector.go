package clockdrift

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/prometheus/client_golang/prometheus"
)

// DriftDetector monitors clock drift and calibrates HLC when necessary
type DriftDetector struct {
	mu sync.RWMutex

	// Configuration
	ntpServer       string
	checkInterval   time.Duration
	threshold       time.Duration
	maxOffset       time.Duration
	historyDuration time.Duration

	// State
	history      *RingBuffer
	lastCheck    time.Time
	lastOffset   time.Duration
	calibrateHLC func(offset time.Duration) error

	// Metrics
	registry               *prometheus.Registry
	currentOffsetGauge     prometheus.Gauge
	calibrationCounter     prometheus.Counter
	thresholdBreachCounter prometheus.Counter
	checkLatencyHistogram  prometheus.Histogram
}

// Config holds configuration for DriftDetector
type Config struct {
	NTPServer       string        // NTP server address (default: "pool.ntp.org")
	CheckInterval   time.Duration // How often to check drift (default: 30s)
	Threshold       time.Duration // Threshold for triggering calibration (default: 500ms)
	MaxOffset       time.Duration // Maximum offset before refusing calibration (default: 10s)
	HistoryDuration time.Duration // How long to keep history (default: 24h)
	HistoryCapacity int           // Ring buffer capacity (default: 2880 = 24h at 30s interval)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		NTPServer:       "pool.ntp.org",
		CheckInterval:   30 * time.Second,
		Threshold:       500 * time.Millisecond,
		MaxOffset:       10 * time.Second,
		HistoryDuration: 24 * time.Hour,
		HistoryCapacity: 2880, // 24h * 3600s / 30s
	}
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(cfg Config, calibrateHLC func(offset time.Duration) error) *DriftDetector {
	if cfg.NTPServer == "" {
		cfg = DefaultConfig()
	}

	dd := &DriftDetector{
		ntpServer:       cfg.NTPServer,
		checkInterval:   cfg.CheckInterval,
		threshold:       cfg.Threshold,
		maxOffset:       cfg.MaxOffset,
		historyDuration: cfg.HistoryDuration,
		history:         NewRingBuffer(cfg.HistoryCapacity),
		calibrateHLC:    calibrateHLC,
	}

	dd.initMetrics()
	return dd
}

// initMetrics initializes Prometheus metrics
func (dd *DriftDetector) initMetrics() {
	// Create a new registry for this detector instance
	dd.registry = prometheus.NewRegistry()

	dd.currentOffsetGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "clock_drift_current_offset_ms",
		Help: "Current clock offset from NTP server in milliseconds",
	})

	dd.calibrationCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "clock_drift_calibration_total",
		Help: "Total number of HLC calibrations performed",
	})

	dd.thresholdBreachCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "clock_drift_threshold_breach_total",
		Help: "Total number of times clock drift exceeded threshold",
	})

	dd.checkLatencyHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "clock_drift_check_latency_ms",
		Help:    "Latency of NTP check operations in milliseconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1ms to 512ms
	})

	// Register metrics with the custom registry
	dd.registry.MustRegister(dd.currentOffsetGauge)
	dd.registry.MustRegister(dd.calibrationCounter)
	dd.registry.MustRegister(dd.thresholdBreachCounter)
	dd.registry.MustRegister(dd.checkLatencyHistogram)
}

// Start begins periodic drift detection
func (dd *DriftDetector) Start(ctx context.Context) error {
	ticker := time.NewTicker(dd.checkInterval)
	defer ticker.Stop()

	// Perform initial check
	if err := dd.CheckNow(); err != nil {
		return fmt.Errorf("initial drift check failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := dd.CheckNow(); err != nil {
				// Log error but continue checking
				fmt.Printf("drift check failed: %v\n", err)
			}
		}
	}
}

// CheckNow performs an immediate drift check
func (dd *DriftDetector) CheckNow() error {
	startTime := time.Now()

	// Query NTP server
	response, err := ntp.Query(dd.ntpServer)
	if err != nil {
		return fmt.Errorf("NTP query failed: %w", err)
	}

	offset := response.ClockOffset
	checkLatency := time.Since(startTime)

	// Record metrics
	dd.checkLatencyHistogram.Observe(float64(checkLatency.Milliseconds()))
	dd.currentOffsetGauge.Set(float64(offset.Milliseconds()))

	// Store sample in history
	sample := ClockSample{
		Timestamp: time.Now(),
		Offset:    offset,
		Source:    dd.ntpServer,
	}

	dd.mu.Lock()
	dd.history.Push(sample)
	dd.lastCheck = sample.Timestamp
	dd.lastOffset = offset
	dd.mu.Unlock()

	// Check if offset exceeds threshold
	absOffset := offset
	if absOffset < 0 {
		absOffset = -absOffset
	}

	if absOffset > dd.threshold {
		dd.thresholdBreachCounter.Inc()

		// Only calibrate if offset is within acceptable range
		if absOffset <= dd.maxOffset {
			if dd.calibrateHLC != nil {
				if err := dd.calibrateHLC(offset); err != nil {
					return fmt.Errorf("HLC calibration failed: %w", err)
				}
				dd.calibrationCounter.Inc()
			}
		} else {
			// Offset too large, only alert
			return fmt.Errorf("clock offset too large: %v (max: %v)", offset, dd.maxOffset)
		}
	}

	return nil
}

// GetHistory returns clock drift history for the specified duration
func (dd *DriftDetector) GetHistory(duration time.Duration) []ClockSample {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	since := time.Now().Add(-duration)
	return dd.history.GetSince(since)
}

// GetLastOffset returns the most recent clock offset
func (dd *DriftDetector) GetLastOffset() (time.Duration, time.Time) {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return dd.lastOffset, dd.lastCheck
}

// CalibrateHLC adjusts HLC based on detected drift
func (dd *DriftDetector) CalibrateHLC(sample ClockSample) error {
	if dd.calibrateHLC == nil {
		return fmt.Errorf("no HLC calibration function configured")
	}

	absOffset := sample.Offset
	if absOffset < 0 {
		absOffset = -absOffset
	}

	// Only calibrate if within acceptable range
	if absOffset > dd.maxOffset {
		return fmt.Errorf("offset %v exceeds max %v, refusing to calibrate", sample.Offset, dd.maxOffset)
	}

	return dd.calibrateHLC(sample.Offset)
}

// GetRegistry returns the Prometheus registry for this detector
func (dd *DriftDetector) GetRegistry() *prometheus.Registry {
	return dd.registry
}
