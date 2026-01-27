package metrics

import (
	"net/http"
	"time"
)

// Collector defines the interface for metrics collection
type Collector interface {
	// Counter operations
	IncrementCounter(name string, labels map[string]string)
	AddCounter(name string, value float64, labels map[string]string)

	// Gauge operations
	SetGauge(name string, value float64, labels map[string]string)
	IncrementGauge(name string, labels map[string]string)
	DecrementGauge(name string, labels map[string]string)

	// Histogram operations
	RecordHistogram(name string, value float64, labels map[string]string)
	RecordDuration(name string, duration time.Duration, labels map[string]string)

	// Handler returns an HTTP handler for exposing metrics
	Handler() http.Handler
}

// Config holds configuration for metrics collector
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Namespace      string
}

// Labels is a convenience type for metric labels
type Labels map[string]string

// Merge merges two label maps
func (l Labels) Merge(other Labels) Labels {
	result := make(Labels, len(l)+len(other))
	for k, v := range l {
		result[k] = v
	}
	for k, v := range other {
		result[k] = v
	}
	return result
}
