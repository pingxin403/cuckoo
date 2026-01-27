package metrics

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// OTelConfig holds configuration for OpenTelemetry metrics
type OTelConfig struct {
	ServiceName       string
	ServiceVersion    string
	Environment       string
	Namespace         string
	OTLPEndpoint      string
	PrometheusEnabled bool
	Insecure          bool
}

// OTelMetricsCollector implements Collector using OpenTelemetry Metrics SDK
type OTelMetricsCollector struct {
	config   OTelConfig
	meter    otelmetric.Meter
	provider *metric.MeterProvider

	// Instrument caches
	counters   sync.Map // map[string]otelmetric.Int64Counter
	histograms sync.Map // map[string]otelmetric.Float64Histogram

	// Gauge support (OTel uses observable gauges)
	gaugeValues sync.Map // map[string]*atomic.Float64
	gauges      sync.Map // map[string]otelmetric.Float64ObservableGauge

	// Prometheus exporter for HTTP handler
	promExporter *otelprom.Exporter
	promRegistry *prometheus.Registry

	// Internal observability metrics
	internalMetrics *internalMetrics

	// Shutdown function
	shutdownFunc func(context.Context) error
}

// internalMetrics tracks internal observability metrics
type internalMetrics struct {
	// Export failures counter
	exportFailures atomic.Int64

	// Instrument creation failures counter
	instrumentFailures atomic.Int64

	// Total operations counter
	totalOperations atomic.Int64

	// Cached instruments count
	cachedCounters   atomic.Int64
	cachedHistograms atomic.Int64
	cachedGauges     atomic.Int64
}

// gaugeValue wraps an atomic uint64 for gauge storage (stores float64 bits)
type gaugeValue struct {
	value atomic.Uint64
}

// Load loads the float64 value
func (g *gaugeValue) Load() float64 {
	return math.Float64frombits(g.value.Load())
}

// Store stores the float64 value
func (g *gaugeValue) Store(val float64) {
	g.value.Store(math.Float64bits(val))
}

// CompareAndSwap performs a compare-and-swap operation
func (g *gaugeValue) CompareAndSwap(old, new float64) bool {
	return g.value.CompareAndSwap(math.Float64bits(old), math.Float64bits(new))
}

// NewOTelMetricsCollector creates a new OpenTelemetry metrics collector
func NewOTelMetricsCollector(config OTelConfig) (*OTelMetricsCollector, error) {
	ctx := context.Background()

	// Create resource with service attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create metric readers
	var readers []metric.Reader

	// OTLP exporter (if endpoint provided)
	if config.OTLPEndpoint != "" {
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(config.OTLPEndpoint),
		}
		if config.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}

		otlpExporter, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		readers = append(readers, metric.NewPeriodicReader(otlpExporter))
	}

	// Prometheus exporter (if enabled)
	var promExporter *otelprom.Exporter
	var promRegistry *prometheus.Registry
	if config.PrometheusEnabled {
		// Create a new Prometheus registry
		promRegistry = prometheus.NewRegistry()

		// Create the OTel Prometheus exporter with the registry
		promExporter, err = otelprom.New(otelprom.WithRegisterer(promRegistry))
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)
	}

	// Create meter provider
	opts := []metric.Option{metric.WithResource(res)}
	for _, reader := range readers {
		opts = append(opts, metric.WithReader(reader))
	}
	provider := metric.NewMeterProvider(opts...)

	// Create meter
	meterName := config.ServiceName
	if config.Namespace != "" {
		meterName = config.Namespace
	}
	meter := provider.Meter(meterName)

	collector := &OTelMetricsCollector{
		config:          config,
		meter:           meter,
		provider:        provider,
		promExporter:    promExporter,
		promRegistry:    promRegistry,
		internalMetrics: &internalMetrics{},
		shutdownFunc:    provider.Shutdown,
	}

	// Register internal observability metrics
	collector.registerInternalMetrics()

	return collector, nil
}

// IncrementCounter increments a counter by 1
func (o *OTelMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	// Don't increment totalOperations here, AddCounter will do it
	o.AddCounter(name, 1, labels)
}

// AddCounter adds a value to a counter
func (o *OTelMetricsCollector) AddCounter(name string, value float64, labels map[string]string) {
	o.internalMetrics.totalOperations.Add(1)
	counter := o.getOrCreateCounter(name)
	attrs := labelsToAttributes(labels)
	counter.Add(context.Background(), int64(value), otelmetric.WithAttributes(attrs...))
}

// SetGauge sets a gauge to a specific value
func (o *OTelMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	o.internalMetrics.totalOperations.Add(1)
	key := makeGaugeKey(name, labels)

	// Get or create gauge value storage
	val, _ := o.gaugeValues.LoadOrStore(key, &gaugeValue{})
	gv := val.(*gaugeValue)
	gv.Store(value)

	// Ensure observable gauge is registered
	o.getOrCreateObservableGauge(name, labels)
}

// IncrementGauge increments a gauge by 1
func (o *OTelMetricsCollector) IncrementGauge(name string, labels map[string]string) {
	o.internalMetrics.totalOperations.Add(1)
	key := makeGaugeKey(name, labels)

	// Get or create gauge value storage
	val, _ := o.gaugeValues.LoadOrStore(key, &gaugeValue{})
	gv := val.(*gaugeValue)

	// Atomic increment
	for {
		old := gv.Load()
		if gv.CompareAndSwap(old, old+1) {
			break
		}
	}

	// Ensure observable gauge is registered
	o.getOrCreateObservableGauge(name, labels)
}

// DecrementGauge decrements a gauge by 1
func (o *OTelMetricsCollector) DecrementGauge(name string, labels map[string]string) {
	o.internalMetrics.totalOperations.Add(1)
	key := makeGaugeKey(name, labels)

	// Get or create gauge value storage
	val, _ := o.gaugeValues.LoadOrStore(key, &gaugeValue{})
	gv := val.(*gaugeValue)

	// Atomic decrement
	for {
		old := gv.Load()
		if gv.CompareAndSwap(old, old-1) {
			break
		}
	}

	// Ensure observable gauge is registered
	o.getOrCreateObservableGauge(name, labels)
}

// RecordHistogram records a value in a histogram
func (o *OTelMetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	o.internalMetrics.totalOperations.Add(1)
	histogram := o.getOrCreateHistogram(name)
	attrs := labelsToAttributes(labels)
	histogram.Record(context.Background(), value, otelmetric.WithAttributes(attrs...))
}

// RecordDuration records a duration in a histogram
func (o *OTelMetricsCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	o.RecordHistogram(name, duration.Seconds(), labels)
}

// Handler returns an HTTP handler for exposing metrics
func (o *OTelMetricsCollector) Handler() http.Handler {
	if o.promRegistry != nil {
		// Use promhttp.HandlerFor with the registry that contains the OTel exporter
		return promhttp.HandlerFor(o.promRegistry, promhttp.HandlerOpts{})
	}
	// Return empty handler if Prometheus is not enabled
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Prometheus exporter not enabled"))
	})
}

// Shutdown gracefully shuts down the metrics collector
func (o *OTelMetricsCollector) Shutdown(ctx context.Context) error {
	if o.shutdownFunc != nil {
		return o.shutdownFunc(ctx)
	}
	return nil
}

// getOrCreateCounter gets or creates a counter instrument
func (o *OTelMetricsCollector) getOrCreateCounter(name string) otelmetric.Int64Counter {
	if val, ok := o.counters.Load(name); ok {
		return val.(otelmetric.Int64Counter)
	}

	counter, err := o.meter.Int64Counter(
		o.formatMetricName(name),
		otelmetric.WithDescription(fmt.Sprintf("Counter metric: %s", name)),
	)
	if err != nil {
		// Track instrument creation failure
		o.internalMetrics.instrumentFailures.Add(1)
		// Return a no-op counter on error
		counter, _ = o.meter.Int64Counter("noop")
	} else {
		o.internalMetrics.cachedCounters.Add(1)
	}

	o.counters.Store(name, counter)
	return counter
}

// getOrCreateHistogram gets or creates a histogram instrument
func (o *OTelMetricsCollector) getOrCreateHistogram(name string) otelmetric.Float64Histogram {
	if val, ok := o.histograms.Load(name); ok {
		return val.(otelmetric.Float64Histogram)
	}

	histogram, err := o.meter.Float64Histogram(
		o.formatMetricName(name),
		otelmetric.WithDescription(fmt.Sprintf("Histogram metric: %s", name)),
	)
	if err != nil {
		// Track instrument creation failure
		o.internalMetrics.instrumentFailures.Add(1)
		// Return a no-op histogram on error
		histogram, _ = o.meter.Float64Histogram("noop")
	} else {
		o.internalMetrics.cachedHistograms.Add(1)
	}

	o.histograms.Store(name, histogram)
	return histogram
}

// getOrCreateObservableGauge gets or creates an observable gauge
func (o *OTelMetricsCollector) getOrCreateObservableGauge(name string, labels map[string]string) {
	key := makeGaugeKey(name, labels)

	if _, ok := o.gauges.Load(key); ok {
		return
	}

	attrs := labelsToAttributes(labels)

	gauge, err := o.meter.Float64ObservableGauge(
		o.formatMetricName(name),
		otelmetric.WithDescription(fmt.Sprintf("Gauge metric: %s", name)),
		otelmetric.WithFloat64Callback(func(ctx context.Context, observer otelmetric.Float64Observer) error {
			if val, ok := o.gaugeValues.Load(key); ok {
				gv := val.(*gaugeValue)
				observer.Observe(gv.Load(), otelmetric.WithAttributes(attrs...))
			}
			return nil
		}),
	)
	if err != nil {
		// Track instrument creation failure
		o.internalMetrics.instrumentFailures.Add(1)
		return
	}

	o.internalMetrics.cachedGauges.Add(1)
	o.gauges.Store(key, gauge)
}

// formatMetricName formats a metric name with namespace
func (o *OTelMetricsCollector) formatMetricName(name string) string {
	if o.config.Namespace != "" {
		return fmt.Sprintf("%s.%s", o.config.Namespace, name)
	}
	return name
}

// Helper functions

// labelsToAttributes converts a label map to OTel attributes
func labelsToAttributes(labels map[string]string) []attribute.KeyValue {
	if len(labels) == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, len(labels))
	for k, v := range labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}

// makeGaugeKey creates a unique key for a gauge with labels
func makeGaugeKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Sort labels for consistent keys
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteByte('|')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(labels[k])
	}

	return sb.String()
}

// registerInternalMetrics registers internal observability metrics
func (o *OTelMetricsCollector) registerInternalMetrics() {
	// Register observable gauges for internal metrics
	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.export_failures",
		otelmetric.WithDescription("Number of metric export failures"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.exportFailures.Load())
			return nil
		}),
	)

	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.instrument_failures",
		otelmetric.WithDescription("Number of metric instrument creation failures"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.instrumentFailures.Load())
			return nil
		}),
	)

	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.total_operations",
		otelmetric.WithDescription("Total number of metric operations"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.totalOperations.Load())
			return nil
		}),
	)

	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.cached_counters",
		otelmetric.WithDescription("Number of cached counter instruments"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.cachedCounters.Load())
			return nil
		}),
	)

	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.cached_histograms",
		otelmetric.WithDescription("Number of cached histogram instruments"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.cachedHistograms.Load())
			return nil
		}),
	)

	_, _ = o.meter.Int64ObservableGauge(
		"otel.metrics.cached_gauges",
		otelmetric.WithDescription("Number of cached gauge instruments"),
		otelmetric.WithInt64Callback(func(ctx context.Context, observer otelmetric.Int64Observer) error {
			observer.Observe(o.internalMetrics.cachedGauges.Load())
			return nil
		}),
	)
}

// GetInternalMetrics returns internal metrics for monitoring (used for testing)
func (o *OTelMetricsCollector) GetInternalMetrics() map[string]int64 {
	return map[string]int64{
		"export_failures":     o.internalMetrics.exportFailures.Load(),
		"instrument_failures": o.internalMetrics.instrumentFailures.Load(),
		"total_operations":    o.internalMetrics.totalOperations.Load(),
		"cached_counters":     o.internalMetrics.cachedCounters.Load(),
		"cached_histograms":   o.internalMetrics.cachedHistograms.Load(),
		"cached_gauges":       o.internalMetrics.cachedGauges.Load(),
	}
}
