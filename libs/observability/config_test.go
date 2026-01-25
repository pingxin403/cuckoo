package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "empty config gets defaults",
			config: Config{},
			expected: Config{
				ServiceName:        "unknown-service",
				ServiceVersion:     "unknown",
				Environment:        "development",
				MetricsPort:        9090,
				MetricsPath:        "/metrics",
				MetricsNamespace:   "unknown-service",
				TracingSampleRate:  0.1,
				LogLevel:           "info",
				LogFormat:          "json",
				LogOutput:          "stdout",
				OTLPProtocol:       "grpc",
				OTLPTimeout:        10 * time.Second,
				OTLPBatchSize:      512,
				OTLPExportInterval: 5 * time.Second,
			},
		},
		{
			name: "custom values preserved",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "production",
				MetricsPort:    8080,
				LogLevel:       "debug",
			},
			expected: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "production",
				MetricsPort:        8080,
				MetricsPath:        "/metrics",
				MetricsNamespace:   "test-service",
				TracingSampleRate:  0.1,
				LogLevel:           "debug",
				LogFormat:          "json",
				LogOutput:          "stdout",
				OTLPProtocol:       "grpc",
				OTLPTimeout:        10 * time.Second,
				OTLPBatchSize:      512,
				OTLPExportInterval: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.WithDefaults()
			assert.Equal(t, tt.expected.ServiceName, result.ServiceName)
			assert.Equal(t, tt.expected.ServiceVersion, result.ServiceVersion)
			assert.Equal(t, tt.expected.Environment, result.Environment)
			assert.Equal(t, tt.expected.MetricsPort, result.MetricsPort)
			assert.Equal(t, tt.expected.MetricsPath, result.MetricsPath)
			assert.Equal(t, tt.expected.LogLevel, result.LogLevel)
			assert.Equal(t, tt.expected.OTLPProtocol, result.OTLPProtocol)
			assert.Equal(t, tt.expected.OTLPTimeout, result.OTLPTimeout)
			assert.Equal(t, tt.expected.OTLPBatchSize, result.OTLPBatchSize)
			assert.Equal(t, tt.expected.OTLPExportInterval, result.OTLPExportInterval)
		})
	}
}

func TestConfig_UnifiedEndpointFallback(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected struct {
			metricsEndpoint string
			tracingEndpoint string
			logsEndpoint    string
		}
	}{
		{
			name: "unified endpoint applies to all signals",
			config: Config{
				OTLPEndpoint: "localhost:4317",
			},
			expected: struct {
				metricsEndpoint string
				tracingEndpoint string
				logsEndpoint    string
			}{
				metricsEndpoint: "localhost:4317",
				tracingEndpoint: "localhost:4317",
				logsEndpoint:    "localhost:4317",
			},
		},
		{
			name: "specific endpoints override unified",
			config: Config{
				OTLPEndpoint:        "localhost:4317",
				OTLPMetricsEndpoint: "metrics.example.com:4317",
				TracingEndpoint:     "traces.example.com:4317",
			},
			expected: struct {
				metricsEndpoint string
				tracingEndpoint string
				logsEndpoint    string
			}{
				metricsEndpoint: "metrics.example.com:4317",
				tracingEndpoint: "traces.example.com:4317",
				logsEndpoint:    "localhost:4317",
			},
		},
		{
			name: "no unified endpoint, specific endpoints used",
			config: Config{
				OTLPMetricsEndpoint: "metrics.example.com:4317",
				TracingEndpoint:     "traces.example.com:4317",
				OTLPLogsEndpoint:    "logs.example.com:4317",
			},
			expected: struct {
				metricsEndpoint string
				tracingEndpoint string
				logsEndpoint    string
			}{
				metricsEndpoint: "metrics.example.com:4317",
				tracingEndpoint: "traces.example.com:4317",
				logsEndpoint:    "logs.example.com:4317",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.WithDefaults()
			assert.Equal(t, tt.expected.metricsEndpoint, result.OTLPMetricsEndpoint)
			assert.Equal(t, tt.expected.tracingEndpoint, result.TracingEndpoint)
			assert.Equal(t, tt.expected.logsEndpoint, result.OTLPLogsEndpoint)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				ServiceName:        "test-service",
				EnableMetrics:      true,
				MetricsPort:        9090,
				MetricsPath:        "/metrics",
				EnableTracing:      true,
				TracingEndpoint:    "localhost:4317",
				TracingSampleRate:  0.5,
				LogLevel:           "info",
				LogFormat:          "json",
				OTLPProtocol:       "grpc",
				OTLPTimeout:        10 * time.Second,
				OTLPBatchSize:      512,
				OTLPExportInterval: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: Config{
				ServiceName: "",
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "invalid metrics port - too low",
			config: Config{
				ServiceName:   "test",
				EnableMetrics: true,
				MetricsPort:   0,
				MetricsPath:   "/metrics",
			},
			wantErr: true,
			errMsg:  "invalid metrics port",
		},
		{
			name: "invalid metrics port - too high",
			config: Config{
				ServiceName:   "test",
				EnableMetrics: true,
				MetricsPort:   70000,
				MetricsPath:   "/metrics",
			},
			wantErr: true,
			errMsg:  "invalid metrics port",
		},
		{
			name: "missing tracing endpoint when enabled",
			config: Config{
				ServiceName:   "test",
				EnableTracing: true,
			},
			wantErr: true,
			errMsg:  "tracing endpoint is required",
		},
		{
			name: "invalid tracing sample rate - negative",
			config: Config{
				ServiceName:       "test",
				EnableTracing:     true,
				TracingEndpoint:   "localhost:4317",
				TracingSampleRate: -0.1,
			},
			wantErr: true,
			errMsg:  "invalid tracing sample rate",
		},
		{
			name: "invalid tracing sample rate - too high",
			config: Config{
				ServiceName:       "test",
				EnableTracing:     true,
				TracingEndpoint:   "localhost:4317",
				TracingSampleRate: 1.5,
			},
			wantErr: true,
			errMsg:  "invalid tracing sample rate",
		},
		{
			name: "invalid log level",
			config: Config{
				ServiceName: "test",
				LogLevel:    "invalid",
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "invalid log format",
			config: Config{
				ServiceName: "test",
				LogLevel:    "info",
				LogFormat:   "xml",
			},
			wantErr: true,
			errMsg:  "invalid log format",
		},
		{
			name: "invalid OTLP protocol",
			config: Config{
				ServiceName:  "test",
				LogLevel:     "info",
				LogFormat:    "json",
				OTLPProtocol: "tcp",
			},
			wantErr: true,
			errMsg:  "invalid OTLP protocol",
		},
		{
			name: "invalid OTLP timeout",
			config: Config{
				ServiceName: "test",
				LogLevel:    "info",
				LogFormat:   "json",
				OTLPTimeout: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid OTLP timeout",
		},
		{
			name: "invalid OTLP batch size",
			config: Config{
				ServiceName:   "test",
				LogLevel:      "info",
				LogFormat:     "json",
				OTLPBatchSize: -1,
			},
			wantErr: true,
			errMsg:  "invalid OTLP batch size",
		},
		{
			name: "invalid OTLP export interval",
			config: Config{
				ServiceName:        "test",
				LogLevel:           "info",
				LogFormat:          "json",
				OTLPExportInterval: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid OTLP export interval",
		},
		{
			name: "invalid pprof block profile rate",
			config: Config{
				ServiceName:           "test",
				LogLevel:              "info",
				LogFormat:             "json",
				PprofBlockProfileRate: -1,
			},
			wantErr: true,
			errMsg:  "invalid pprof block profile rate",
		},
		{
			name: "invalid pprof mutex profile fraction",
			config: Config{
				ServiceName:               "test",
				LogLevel:                  "info",
				LogFormat:                 "json",
				PprofMutexProfileFraction: -1,
			},
			wantErr: true,
			errMsg:  "invalid pprof mutex profile fraction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_OTLPDefaults(t *testing.T) {
	config := Config{}.WithDefaults()

	assert.Equal(t, "grpc", config.OTLPProtocol)
	assert.Equal(t, 10*time.Second, config.OTLPTimeout)
	assert.Equal(t, 512, config.OTLPBatchSize)
	assert.Equal(t, 5*time.Second, config.OTLPExportInterval)
}

func TestConfig_ResourceAttributes(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		ResourceAttributes: map[string]string{
			"team":        "platform",
			"environment": "staging",
		},
	}

	assert.NotNil(t, config.ResourceAttributes)
	assert.Equal(t, "platform", config.ResourceAttributes["team"])
	assert.Equal(t, "staging", config.ResourceAttributes["environment"])
}
