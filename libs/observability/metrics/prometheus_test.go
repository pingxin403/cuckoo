package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheusCollector(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Namespace:      "test",
	}

	collector := NewPrometheusCollector(config)
	require.NotNil(t, collector)
	assert.Equal(t, config, collector.config)
	assert.NotNil(t, collector.counters)
	assert.NotNil(t, collector.gauges)
	assert.NotNil(t, collector.histograms)
}

func TestPrometheusCollector_Counter(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Increment counter
	collector.IncrementCounter("requests_total", map[string]string{"method": "GET"})
	collector.IncrementCounter("requests_total", map[string]string{"method": "GET"})
	collector.IncrementCounter("requests_total", map[string]string{"method": "POST"})

	// Add to counter
	collector.AddCounter("bytes_total", 100, nil)
	collector.AddCounter("bytes_total", 50, nil)

	// Verify counters exist
	assert.Len(t, collector.counters, 2)
	assert.Contains(t, collector.counters, "requests_total")
	assert.Contains(t, collector.counters, "bytes_total")
}

func TestPrometheusCollector_Gauge(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Set gauge
	collector.SetGauge("active_connections", 10, nil)
	collector.SetGauge("active_connections", 20, nil)

	// Increment gauge
	collector.IncrementGauge("goroutines", nil)
	collector.IncrementGauge("goroutines", nil)

	// Decrement gauge
	collector.DecrementGauge("goroutines", nil)

	// Verify gauges exist
	assert.Len(t, collector.gauges, 2)
	assert.Contains(t, collector.gauges, "active_connections")
	assert.Contains(t, collector.gauges, "goroutines")
}

func TestPrometheusCollector_Histogram(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Record histogram values
	collector.RecordHistogram("request_duration_seconds", 0.05, map[string]string{"method": "GET"})
	collector.RecordHistogram("request_duration_seconds", 0.15, map[string]string{"method": "GET"})
	collector.RecordHistogram("request_duration_seconds", 0.5, map[string]string{"method": "POST"})

	// Record duration
	collector.RecordDuration("processing_time", 100*time.Millisecond, nil)

	// Verify histograms exist
	assert.Len(t, collector.histograms, 2)
	assert.Contains(t, collector.histograms, "request_duration_seconds")
	assert.Contains(t, collector.histograms, "processing_time")
}

func TestPrometheusCollector_Handler(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Add some metrics
	collector.IncrementCounter("requests_total", map[string]string{"method": "GET"})
	collector.SetGauge("active_connections", 42, nil)
	collector.RecordHistogram("request_duration_seconds", 0.1, nil)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler := collector.Handler()
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; version=0.0.4", w.Header().Get("Content-Type"))

	body := w.Body.String()

	// Verify counter is present
	assert.Contains(t, body, "# TYPE test_requests_total counter")
	assert.Contains(t, body, "test_requests_total")

	// Verify gauge is present
	assert.Contains(t, body, "# TYPE test_active_connections gauge")
	assert.Contains(t, body, "test_active_connections 42.00")

	// Verify histogram is present
	assert.Contains(t, body, "# TYPE test_request_duration_seconds histogram")
	assert.Contains(t, body, "test_request_duration_seconds_bucket")
	assert.Contains(t, body, "test_request_duration_seconds_sum")
	assert.Contains(t, body, "test_request_duration_seconds_count")
}

func TestPrometheusCollector_HandlerFormat(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})
	collector.IncrementCounter("requests_total", nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := collector.Handler()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	lines := strings.Split(body, "\n")

	// Verify Prometheus format
	helpFound := false
	typeFound := false
	metricFound := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP test_requests_total") {
			helpFound = true
		}
		if strings.HasPrefix(line, "# TYPE test_requests_total counter") {
			typeFound = true
		}
		if strings.HasPrefix(line, "test_requests_total") && !strings.HasPrefix(line, "# ") {
			metricFound = true
		}
	}

	assert.True(t, helpFound, "HELP line should be present")
	assert.True(t, typeFound, "TYPE line should be present")
	assert.True(t, metricFound, "Metric line should be present")
}

func TestPrometheusCollector_ConcurrentAccess(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Simulate concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.IncrementCounter("requests_total", nil)
				collector.SetGauge("active_connections", float64(j), nil)
				collector.RecordHistogram("request_duration_seconds", 0.1, nil)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no panics and metrics exist
	assert.NotEmpty(t, collector.counters)
	assert.NotEmpty(t, collector.gauges)
	assert.NotEmpty(t, collector.histograms)
}

func TestPrometheusCollector_HistogramBuckets(t *testing.T) {
	collector := NewPrometheusCollector(Config{Namespace: "test"})

	// Record values in different buckets
	collector.RecordHistogram("latency", 0.005, nil) // < 10ms
	collector.RecordHistogram("latency", 0.025, nil) // < 50ms
	collector.RecordHistogram("latency", 0.075, nil) // < 100ms
	collector.RecordHistogram("latency", 0.15, nil)  // < 200ms
	collector.RecordHistogram("latency", 0.3, nil)   // < 500ms
	collector.RecordHistogram("latency", 0.7, nil)   // < 1s
	collector.RecordHistogram("latency", 1.5, nil)   // < 2s
	collector.RecordHistogram("latency", 3.0, nil)   // < 5s
	collector.RecordHistogram("latency", 7.0, nil)   // < 10s
	collector.RecordHistogram("latency", 15.0, nil)  // > 10s

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := collector.Handler()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify all buckets are present
	assert.Contains(t, body, "le=\"0.010000\"")
	assert.Contains(t, body, "le=\"0.050000\"")
	assert.Contains(t, body, "le=\"0.100000\"")
	assert.Contains(t, body, "le=\"0.200000\"")
	assert.Contains(t, body, "le=\"0.500000\"")
	assert.Contains(t, body, "le=\"1.000000\"")
	assert.Contains(t, body, "le=\"2.000000\"")
	assert.Contains(t, body, "le=\"5.000000\"")
	assert.Contains(t, body, "le=\"10.000000\"")
	assert.Contains(t, body, "le=\"+Inf\"")

	// Verify sum and count
	assert.Contains(t, body, "test_latency_sum")
	assert.Contains(t, body, "test_latency_count 10")
}

func TestPrometheusCollector_MetricNameFormatting(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		metric    string
		expected  string
	}{
		{
			name:      "with namespace",
			namespace: "myapp",
			metric:    "requests_total",
			expected:  "myapp_requests_total",
		},
		{
			name:      "without namespace",
			namespace: "",
			metric:    "requests_total",
			expected:  "requests_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewPrometheusCollector(Config{Namespace: tt.namespace})
			collector.IncrementCounter(tt.metric, nil)

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			w := httptest.NewRecorder()

			handler := collector.Handler()
			handler.ServeHTTP(w, req)

			body := w.Body.String()
			assert.Contains(t, body, tt.expected)
		})
	}
}
