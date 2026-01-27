package observability

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for observability
type Config struct {
	// Service identification
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Metrics configuration
	EnableMetrics     bool
	MetricsPort       int
	MetricsPath       string
	MetricsNamespace  string
	UseOTelMetrics    bool // Use OpenTelemetry Metrics SDK
	PrometheusEnabled bool // Enable Prometheus exporter (can be used with OTel)

	// Tracing configuration
	EnableTracing     bool
	TracingEndpoint   string // OTLP endpoint for traces
	TracingSampleRate float64

	// Logging configuration
	LogLevel    string
	LogFormat   string
	LogOutput   string
	UseOTelLogs bool // Use OpenTelemetry Logs SDK

	// pprof configuration
	EnablePprof               bool
	PprofBlockProfileRate     int // nanoseconds, 0 = disabled
	PprofMutexProfileFraction int // 1/rate, 0 = disabled

	// Unified OTLP configuration
	OTLPEndpoint        string        // Unified endpoint for all signals (metrics, traces, logs)
	OTLPMetricsEndpoint string        // Specific endpoint for metrics (overrides OTLPEndpoint)
	OTLPLogsEndpoint    string        // Specific endpoint for logs (overrides OTLPEndpoint)
	OTLPProtocol        string        // "grpc" or "http"
	OTLPTimeout         time.Duration // Timeout for OTLP export
	OTLPBatchSize       int           // Batch size for OTLP export
	OTLPExportInterval  time.Duration // Export interval for OTLP
	OTLPInsecure        bool          // Use insecure connection (for development)

	// Custom resource attributes
	ResourceAttributes map[string]string
}

// WithDefaults returns a config with default values applied
func (c Config) WithDefaults() Config {
	if c.ServiceName == "" {
		c.ServiceName = getEnv("SERVICE_NAME", "unknown-service")
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = getEnv("SERVICE_VERSION", "unknown")
	}
	if c.Environment == "" {
		c.Environment = getEnv("DEPLOYMENT_ENVIRONMENT", "development")
	}

	// Metrics defaults
	if c.MetricsPort == 0 {
		c.MetricsPort = getEnvInt("METRICS_PORT", 9090)
	}
	if c.MetricsPath == "" {
		c.MetricsPath = getEnv("METRICS_PATH", "/metrics")
	}
	if c.MetricsNamespace == "" {
		c.MetricsNamespace = c.ServiceName
	}

	// Tracing defaults
	if c.TracingEndpoint == "" {
		c.TracingEndpoint = getEnv("TRACING_ENDPOINT", "")
	}
	if c.TracingSampleRate == 0 {
		c.TracingSampleRate = getEnvFloat("TRACING_SAMPLE_RATE", 0.1)
	}

	// Logging defaults
	if c.LogLevel == "" {
		c.LogLevel = getEnv("LOG_LEVEL", "info")
	}
	if c.LogFormat == "" {
		c.LogFormat = getEnv("LOG_FORMAT", "json")
	}
	if c.LogOutput == "" {
		c.LogOutput = getEnv("LOG_OUTPUT", "stdout")
	}

	// pprof defaults
	// EnablePprof defaults to false (opt-in for security)
	// Block profile rate: 0 = disabled, 1 = every block event, higher = sample rate
	// Mutex profile fraction: 0 = disabled, 1 = every mutex event, higher = 1/rate sampling
	if c.PprofBlockProfileRate == 0 {
		c.PprofBlockProfileRate = getEnvInt("PPROF_BLOCK_PROFILE_RATE", 0)
	}
	if c.PprofMutexProfileFraction == 0 {
		c.PprofMutexProfileFraction = getEnvInt("PPROF_MUTEX_PROFILE_FRACTION", 0)
	}

	// OTLP defaults
	if c.OTLPEndpoint == "" {
		c.OTLPEndpoint = getEnv("OTLP_ENDPOINT", "")
	}
	if c.OTLPProtocol == "" {
		c.OTLPProtocol = getEnv("OTLP_PROTOCOL", "grpc")
	}
	if c.OTLPTimeout == 0 {
		c.OTLPTimeout = getEnvDuration("OTLP_TIMEOUT", 10*time.Second)
	}
	if c.OTLPBatchSize == 0 {
		c.OTLPBatchSize = getEnvInt("OTLP_BATCH_SIZE", 512)
	}
	if c.OTLPExportInterval == 0 {
		c.OTLPExportInterval = getEnvDuration("OTLP_EXPORT_INTERVAL", 5*time.Second)
	}

	// Unified endpoint fallback logic
	// If OTLPEndpoint is set, use it as default for signal-specific endpoints
	if c.OTLPEndpoint != "" {
		if c.OTLPMetricsEndpoint == "" {
			c.OTLPMetricsEndpoint = c.OTLPEndpoint
		}
		if c.TracingEndpoint == "" {
			c.TracingEndpoint = c.OTLPEndpoint
		}
		if c.OTLPLogsEndpoint == "" {
			c.OTLPLogsEndpoint = c.OTLPEndpoint
		}
	}

	// Allow environment variable overrides for signal-specific endpoints
	if envMetrics := getEnv("OTLP_METRICS_ENDPOINT", ""); envMetrics != "" {
		c.OTLPMetricsEndpoint = envMetrics
	}
	if envLogs := getEnv("OTLP_LOGS_ENDPOINT", ""); envLogs != "" {
		c.OTLPLogsEndpoint = envLogs
	}

	return c
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if c.EnableMetrics {
		if c.MetricsPort < 1 || c.MetricsPort > 65535 {
			return fmt.Errorf("invalid metrics port: %d (must be between 1 and 65535)", c.MetricsPort)
		}
		if c.MetricsPath == "" {
			return fmt.Errorf("metrics path is required")
		}
	}

	if c.EnableTracing {
		if c.TracingEndpoint == "" {
			return fmt.Errorf("tracing endpoint is required when tracing is enabled")
		}
		if c.TracingSampleRate < 0 || c.TracingSampleRate > 1 {
			return fmt.Errorf("invalid tracing sample rate: %f (must be between 0 and 1)", c.TracingSampleRate)
		}
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.LogFormat)
	}

	// OTLP configuration validation
	if c.OTLPProtocol != "" && c.OTLPProtocol != "grpc" && c.OTLPProtocol != "http" {
		return fmt.Errorf("invalid OTLP protocol: %s (must be grpc or http)", c.OTLPProtocol)
	}

	if c.OTLPTimeout < 0 {
		return fmt.Errorf("invalid OTLP timeout: %v (must be non-negative)", c.OTLPTimeout)
	}

	if c.OTLPBatchSize < 0 {
		return fmt.Errorf("invalid OTLP batch size: %d (must be non-negative)", c.OTLPBatchSize)
	}

	if c.OTLPExportInterval < 0 {
		return fmt.Errorf("invalid OTLP export interval: %v (must be non-negative)", c.OTLPExportInterval)
	}

	// pprof configuration validation
	if c.PprofBlockProfileRate < 0 {
		return fmt.Errorf("invalid pprof block profile rate: %d (must be non-negative)", c.PprofBlockProfileRate)
	}

	if c.PprofMutexProfileFraction < 0 {
		return fmt.Errorf("invalid pprof mutex profile fraction: %d (must be non-negative)", c.PprofMutexProfileFraction)
	}

	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
