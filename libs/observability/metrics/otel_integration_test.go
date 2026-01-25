package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOTelMetricsCollector(t *testing.T) {
	tests := []struct {
		name    string
		config  OTelConfig
		wantErr bool
	}{
		{
			name: "Prometheus only",
			config: OTelConfig{
				ServiceName:       "test-service",
				ServiceVersion:    "1.0.0",
				Environment:       "test",
				PrometheusEnabled: true,
			},
			wantErr: false,
		},
		{
			name: "OTLP only",
			config: OTelConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4317",
				Insecure:       true,
			},
			wantErr: false,
		},
		{
			name: "Both exporters",
			config: OTelConfig{
				ServiceName:       "test-service",
				ServiceVersion:    "1.0.0",
				Environment:       "test",
				OTLPEndpoint:      "localhost:4317",
				PrometheusEnabled: true,
				Insecure:          true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := NewOTelMetricsCollector(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, collector)
			assert.NotNil(t, collector.meter)
			assert.NotNil(t, collector.provider)

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = collector.Shutdown(ctx)
		})
	}
}

func TestOTelMetricsCollector_Counter(t *testing.T) {
	collector, err := NewOTelMetricsCollector(OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		PrometheusEnabled: true,
	})
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Test IncrementCounter
	collector.IncrementCounter("test_counter", map[string]string{
		"label1": "value1",
	})

	// Test AddCounter
	collector.AddCounter("test_counter", 5, map[string]string{
		"label1": "value1",
	})

	// Verify counter was created
	val, ok := collector.counters.Load("test_counter")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestOTelMetricsCollector_Gauge(t *testing.T) {
	collector, err := NewOTelMetricsCollector(OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		PrometheusEnabled: true,
	})
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	labels := map[string]string{"label1": "value1"}

	// Test SetGauge
	collector.SetGauge("test_gauge", 42.5, labels)

	// Verify gauge value
	key := makeGaugeKey("test_gauge", labels)
	val, ok := collector.gaugeValues.Load(key)
	require.True(t, ok)
	gv := val.(*gaugeValue)
	assert.Equal(t, 42.5, gv.Load())

	// Test IncrementGauge
	collector.IncrementGauge("test_gauge", labels)
	assert.Equal(t, 43.5, gv.Load())

	// Test DecrementGauge
	collector.DecrementGauge("test_gauge", labels)
	assert.Equal(t, 42.5, gv.Load())
}

func TestOTelMetricsCollector_Histogram(t *testing.T) {
	collector, err := NewOTelMetricsCollector(OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		PrometheusEnabled: true,
	})
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Test RecordHistogram
	collector.RecordHistogram("test_histogram", 0.123, map[string]string{
		"label1": "value1",
	})

	// Test RecordDuration
	collector.RecordDuration("test_duration", 100*time.Millisecond, map[string]string{
		"label1": "value1",
	})

	// Verify histogram was created
	val, ok := collector.histograms.Load("test_histogram")
	assert.True(t, ok)
	assert.NotNil(t, val)

	val, ok = collector.histograms.Load("test_duration")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestOTelMetricsCollector_Handler(t *testing.T) {
	t.Run("Prometheus enabled", func(t *testing.T) {
		collector, err := NewOTelMetricsCollector(OTelConfig{
			ServiceName:       "test-service",
			ServiceVersion:    "1.0.0",
			Environment:       "test",
			PrometheusEnabled: true,
		})
		require.NoError(t, err)
		defer collector.Shutdown(context.Background())

		handler := collector.Handler()
		assert.NotNil(t, handler)
	})

	t.Run("Prometheus disabled", func(t *testing.T) {
		collector, err := NewOTelMetricsCollector(OTelConfig{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			OTLPEndpoint:   "localhost:4317",
			Insecure:       true,
		})
		require.NoError(t, err)
		defer collector.Shutdown(context.Background())

		handler := collector.Handler()
		assert.NotNil(t, handler)
	})
}

func TestOTelMetricsCollector_Namespace(t *testing.T) {
	collector, err := NewOTelMetricsCollector(OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		Namespace:         "custom_namespace",
		PrometheusEnabled: true,
	})
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Test that namespace is applied
	collector.IncrementCounter("test_counter", nil)

	// Verify counter was created
	val, ok := collector.counters.Load("test_counter")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestOTelMetricsCollector_ConcurrentAccess(t *testing.T) {
	collector, err := NewOTelMetricsCollector(OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		PrometheusEnabled: true,
	})
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Test concurrent counter operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.IncrementCounter("concurrent_counter", map[string]string{
					"worker": "test",
				})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent gauge operations
	labels := map[string]string{"worker": "test"}
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.IncrementGauge("concurrent_gauge", labels)
				collector.DecrementGauge("concurrent_gauge", labels)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify gauge value is consistent (should be 0 after equal increments/decrements)
	key := makeGaugeKey("concurrent_gauge", labels)
	val, ok := collector.gaugeValues.Load(key)
	require.True(t, ok)
	gv := val.(*gaugeValue)
	assert.Equal(t, 0.0, gv.Load())
}

func TestLabelsToAttributes(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   int
	}{
		{
			name:   "nil labels",
			labels: nil,
			want:   0,
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   0,
		},
		{
			name: "single label",
			labels: map[string]string{
				"key1": "value1",
			},
			want: 1,
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := labelsToAttributes(tt.labels)
			assert.Len(t, attrs, tt.want)
		})
	}
}

func TestMakeGaugeKey(t *testing.T) {
	tests := []struct {
		name   string
		metric string
		labels map[string]string
		want   string
	}{
		{
			name:   "no labels",
			metric: "test_metric",
			labels: nil,
			want:   "test_metric",
		},
		{
			name:   "single label",
			metric: "test_metric",
			labels: map[string]string{"key1": "value1"},
			want:   "test_metric|key1=value1",
		},
		{
			name:   "multiple labels sorted",
			metric: "test_metric",
			labels: map[string]string{
				"key2": "value2",
				"key1": "value1",
			},
			want: "test_metric|key1=value1,key2=value2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeGaugeKey(tt.metric, tt.labels)
			assert.Equal(t, tt.want, got)
		})
	}
}
