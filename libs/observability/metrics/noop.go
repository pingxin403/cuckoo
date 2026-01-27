package metrics

import (
	"net/http"
	"time"
)

// NoOpCollector is a no-op implementation of Collector
type NoOpCollector struct{}

// NewNoOpCollector creates a new no-op collector
func NewNoOpCollector() *NoOpCollector {
	return &NoOpCollector{}
}

// IncrementCounter does nothing
func (n *NoOpCollector) IncrementCounter(name string, labels map[string]string) {}

// AddCounter does nothing
func (n *NoOpCollector) AddCounter(name string, value float64, labels map[string]string) {}

// SetGauge does nothing
func (n *NoOpCollector) SetGauge(name string, value float64, labels map[string]string) {}

// IncrementGauge does nothing
func (n *NoOpCollector) IncrementGauge(name string, labels map[string]string) {}

// DecrementGauge does nothing
func (n *NoOpCollector) DecrementGauge(name string, labels map[string]string) {}

// RecordHistogram does nothing
func (n *NoOpCollector) RecordHistogram(name string, value float64, labels map[string]string) {}

// RecordDuration does nothing
func (n *NoOpCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
}

// Handler returns a handler that returns empty metrics
func (n *NoOpCollector) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("# No metrics enabled\n"))
	})
}
