package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PrometheusCollector implements Collector using Prometheus format
type PrometheusCollector struct {
	config Config

	// Counters
	countersMu sync.RWMutex
	counters   map[string]*counter

	// Gauges
	gaugesMu sync.RWMutex
	gauges   map[string]*gauge

	// Histograms
	histogramsMu sync.RWMutex
	histograms   map[string]*histogram
}

type counter struct {
	values map[string]float64 // label hash -> value
	mu     sync.RWMutex
}

type gauge struct {
	values map[string]float64 // label hash -> value
	mu     sync.RWMutex
}

type histogram struct {
	buckets []float64
	counts  map[string][]int64 // label hash -> bucket counts
	sums    map[string]float64 // label hash -> sum
	totals  map[string]int64   // label hash -> total count
	mu      sync.RWMutex
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector(config Config) *PrometheusCollector {
	return &PrometheusCollector{
		config:     config,
		counters:   make(map[string]*counter),
		gauges:     make(map[string]*gauge),
		histograms: make(map[string]*histogram),
	}
}

// IncrementCounter increments a counter by 1
func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
	p.AddCounter(name, 1, labels)
}

// AddCounter adds a value to a counter
func (p *PrometheusCollector) AddCounter(name string, value float64, labels map[string]string) {
	p.countersMu.Lock()
	c, exists := p.counters[name]
	if !exists {
		c = &counter{values: make(map[string]float64)}
		p.counters[name] = c
	}
	p.countersMu.Unlock()

	labelHash := hashLabels(labels)
	c.mu.Lock()
	c.values[labelHash] += value
	c.mu.Unlock()
}

// SetGauge sets a gauge to a specific value
func (p *PrometheusCollector) SetGauge(name string, value float64, labels map[string]string) {
	p.gaugesMu.Lock()
	g, exists := p.gauges[name]
	if !exists {
		g = &gauge{values: make(map[string]float64)}
		p.gauges[name] = g
	}
	p.gaugesMu.Unlock()

	labelHash := hashLabels(labels)
	g.mu.Lock()
	g.values[labelHash] = value
	g.mu.Unlock()
}

// IncrementGauge increments a gauge by 1
func (p *PrometheusCollector) IncrementGauge(name string, labels map[string]string) {
	p.gaugesMu.Lock()
	g, exists := p.gauges[name]
	if !exists {
		g = &gauge{values: make(map[string]float64)}
		p.gauges[name] = g
	}
	p.gaugesMu.Unlock()

	labelHash := hashLabels(labels)
	g.mu.Lock()
	g.values[labelHash]++
	g.mu.Unlock()
}

// DecrementGauge decrements a gauge by 1
func (p *PrometheusCollector) DecrementGauge(name string, labels map[string]string) {
	p.gaugesMu.Lock()
	g, exists := p.gauges[name]
	if !exists {
		g = &gauge{values: make(map[string]float64)}
		p.gauges[name] = g
	}
	p.gaugesMu.Unlock()

	labelHash := hashLabels(labels)
	g.mu.Lock()
	g.values[labelHash]--
	g.mu.Unlock()
}

// RecordHistogram records a value in a histogram
func (p *PrometheusCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	p.histogramsMu.Lock()
	h, exists := p.histograms[name]
	if !exists {
		h = &histogram{
			buckets: defaultBuckets(),
			counts:  make(map[string][]int64),
			sums:    make(map[string]float64),
			totals:  make(map[string]int64),
		}
		p.histograms[name] = h
	}
	p.histogramsMu.Unlock()

	labelHash := hashLabels(labels)
	h.mu.Lock()
	defer h.mu.Unlock()

	// Initialize buckets if needed
	if _, exists := h.counts[labelHash]; !exists {
		h.counts[labelHash] = make([]int64, len(h.buckets))
	}

	// Update buckets
	for i, bucket := range h.buckets {
		if value <= bucket {
			h.counts[labelHash][i]++
		}
	}

	// Update sum and count
	h.sums[labelHash] += value
	h.totals[labelHash]++
}

// RecordDuration records a duration in a histogram
func (p *PrometheusCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	p.RecordHistogram(name, duration.Seconds(), labels)
}

// Handler returns an HTTP handler that exposes metrics in Prometheus format
func (p *PrometheusCollector) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Write counters
		p.countersMu.RLock()
		for name, c := range p.counters {
			metricName := p.formatMetricName(name)
			_, _ = fmt.Fprintf(w, "# HELP %s Counter metric\n", metricName)
			_, _ = fmt.Fprintf(w, "# TYPE %s counter\n", metricName)

			c.mu.RLock()
			for labelHash, value := range c.values {
				labels := parseLabels(labelHash)
				if len(labels) > 0 {
					_, _ = fmt.Fprintf(w, "%s{%s} %.2f\n", metricName, formatLabels(labels), value)
				} else {
					_, _ = fmt.Fprintf(w, "%s %.2f\n", metricName, value)
				}
			}
			c.mu.RUnlock()
		}
		p.countersMu.RUnlock()

		// Write gauges
		p.gaugesMu.RLock()
		for name, g := range p.gauges {
			metricName := p.formatMetricName(name)
			_, _ = fmt.Fprintf(w, "# HELP %s Gauge metric\n", metricName)
			_, _ = fmt.Fprintf(w, "# TYPE %s gauge\n", metricName)

			g.mu.RLock()
			for labelHash, value := range g.values {
				labels := parseLabels(labelHash)
				if len(labels) > 0 {
					_, _ = fmt.Fprintf(w, "%s{%s} %.2f\n", metricName, formatLabels(labels), value)
				} else {
					_, _ = fmt.Fprintf(w, "%s %.2f\n", metricName, value)
				}
			}
			g.mu.RUnlock()
		}
		p.gaugesMu.RUnlock()

		// Write histograms
		p.histogramsMu.RLock()
		for name, h := range p.histograms {
			metricName := p.formatMetricName(name)
			_, _ = fmt.Fprintf(w, "# HELP %s Histogram metric\n", metricName)
			_, _ = fmt.Fprintf(w, "# TYPE %s histogram\n", metricName)

			h.mu.RLock()
			for labelHash := range h.counts {
				labels := parseLabels(labelHash)
				labelStr := ""
				if len(labels) > 0 {
					labelStr = formatLabels(labels)
				}

				// Write buckets
				for i, bucket := range h.buckets {
					bucketLabel := fmt.Sprintf("le=\"%.6f\"", bucket)
					if labelStr != "" {
						_, _ = fmt.Fprintf(w, "%s_bucket{%s,%s} %d\n", metricName, labelStr, bucketLabel, h.counts[labelHash][i])
					} else {
						_, _ = fmt.Fprintf(w, "%s_bucket{%s} %d\n", metricName, bucketLabel, h.counts[labelHash][i])
					}
				}

				// Write +Inf bucket
				infLabel := "le=\"+Inf\""
				if labelStr != "" {
					_, _ = fmt.Fprintf(w, "%s_bucket{%s,%s} %d\n", metricName, labelStr, infLabel, h.totals[labelHash])
				} else {
					_, _ = fmt.Fprintf(w, "%s_bucket{%s} %d\n", metricName, infLabel, h.totals[labelHash])
				}

				// Write sum and count
				if labelStr != "" {
					_, _ = fmt.Fprintf(w, "%s_sum{%s} %.6f\n", metricName, labelStr, h.sums[labelHash])
					_, _ = fmt.Fprintf(w, "%s_count{%s} %d\n", metricName, labelStr, h.totals[labelHash])
				} else {
					_, _ = fmt.Fprintf(w, "%s_sum %.6f\n", metricName, h.sums[labelHash])
					_, _ = fmt.Fprintf(w, "%s_count %d\n", metricName, h.totals[labelHash])
				}
			}
			h.mu.RUnlock()
		}
		p.histogramsMu.RUnlock()
	})
}

// formatMetricName formats a metric name with namespace
func (p *PrometheusCollector) formatMetricName(name string) string {
	if p.config.Namespace != "" {
		return fmt.Sprintf("%s_%s", p.config.Namespace, name)
	}
	return name
}

// Helper functions

func hashLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	// Simple hash: concatenate key=value pairs
	result := ""
	for k, v := range labels {
		result += fmt.Sprintf("%s=%s,", k, v)
	}
	return result
}

func parseLabels(hash string) map[string]string {
	if hash == "" {
		return nil
	}
	// Parse back from hash (simplified)
	return map[string]string{"_hash": hash}
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	result := ""
	first := true
	for k, v := range labels {
		if k == "_hash" {
			continue
		}
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%s=\"%s\"", k, v)
		first = false
	}
	return result
}

func defaultBuckets() []float64 {
	// Default buckets: 10ms, 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s, 10s
	return []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0}
}
