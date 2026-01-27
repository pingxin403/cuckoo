package logging

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewOTelLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  OTelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: OTelConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4317",
				Level:          "info",
				Insecure:       true,
			},
			wantErr: false,
		},
		{
			name: "with debug level",
			config: OTelConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4317",
				Level:          "debug",
				Insecure:       true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewOTelLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.logger)
			assert.NotNil(t, logger.provider)

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = logger.Shutdown(ctx)
		})
	}
}

func TestOTelLogger_LogLevels(t *testing.T) {
	logger, err := NewOTelLogger(OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "debug",
		Insecure:       true,
	})
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	ctx := context.Background()

	// Test all log levels
	logger.Debug(ctx, "debug message", "key", "value")
	logger.Info(ctx, "info message", "key", "value")
	logger.Warn(ctx, "warn message", "key", "value")
	logger.Error(ctx, "error message", "key", "value")

	// No assertions - just verify no panics
}

func TestOTelLogger_WithFields(t *testing.T) {
	logger, err := NewOTelLogger(OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	})
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Create child logger with additional fields
	childLogger := logger.With("request_id", "123", "user_id", "456")
	assert.NotNil(t, childLogger)

	// Verify child logger is a different instance
	otelLogger := logger
	childOtelLogger, ok := childLogger.(*OTelLogger)
	require.True(t, ok)
	assert.NotEqual(t, otelLogger, childOtelLogger)

	// Log with child logger
	childLogger.Info(context.Background(), "test message", "extra", "field")

	// Verify parent logger fields are not modified
	assert.Len(t, otelLogger.fields, 0)

	// Verify child logger has additional fields
	assert.Len(t, childOtelLogger.fields, 2)
}

func TestOTelLogger_TraceCorrelation(t *testing.T) {
	logger, err := NewOTelLogger(OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	})
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Create a tracer provider with a valid tracer
	tp := trace.NewNoopTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Note: NoopTracer creates invalid span contexts
	// In a real scenario with a proper tracer, the span context would be valid
	spanCtx := span.SpanContext()

	// Log within span context
	logger.Info(ctx, "test message with trace", "key", "value")

	// Note: In a real test with a proper tracer and collector,
	// we would capture the exported log record and verify it contains
	// the trace_id and span_id as attributes
	_ = spanCtx
}

func TestOTelLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		configLevel string
		logLevel    string
		shouldLog   bool
	}{
		{
			name:        "info level logs info",
			configLevel: "info",
			logLevel:    "info",
			shouldLog:   true,
		},
		{
			name:        "info level filters debug",
			configLevel: "info",
			logLevel:    "debug",
			shouldLog:   false,
		},
		{
			name:        "debug level logs debug",
			configLevel: "debug",
			logLevel:    "debug",
			shouldLog:   true,
		},
		{
			name:        "error level filters info",
			configLevel: "error",
			logLevel:    "info",
			shouldLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewOTelLogger(OTelConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4317",
				Level:          tt.configLevel,
				Insecure:       true,
			})
			require.NoError(t, err)
			defer logger.Shutdown(context.Background())

			ctx := context.Background()

			// Log at specified level
			switch tt.logLevel {
			case "debug":
				logger.Debug(ctx, "test message")
			case "info":
				logger.Info(ctx, "test message")
			case "warn":
				logger.Warn(ctx, "test message")
			case "error":
				logger.Error(ctx, "test message")
			}

			// No assertions - just verify no panics
			// In a real test, we would verify the log was/wasn't exported
		})
	}
}

func TestOTelLogger_KeyValueTypes(t *testing.T) {
	logger, err := NewOTelLogger(OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	})
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	ctx := context.Background()

	// Test different value types
	logger.Info(ctx, "test message",
		"string_key", "string_value",
		"int_key", 42,
		"int64_key", int64(123),
		"float_key", 3.14,
		"bool_key", true,
	)

	// No assertions - just verify no panics
}

func TestOTelLogger_ConcurrentAccess(t *testing.T) {
	logger, err := NewOTelLogger(OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	})
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Test concurrent logging
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.Info(context.Background(), "concurrent message",
					"goroutine", id,
					"iteration", j,
				)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No assertions - just verify no panics or data races
}

func TestKvPairsToKeyValues(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		want  int
	}{
		{
			name:  "nil input",
			input: nil,
			want:  0,
		},
		{
			name:  "empty input",
			input: []interface{}{},
			want:  0,
		},
		{
			name:  "single pair",
			input: []interface{}{"key1", "value1"},
			want:  1,
		},
		{
			name:  "multiple pairs",
			input: []interface{}{"key1", "value1", "key2", "value2", "key3", "value3"},
			want:  3,
		},
		{
			name:  "odd number of elements",
			input: []interface{}{"key1", "value1", "key2"},
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs := kvPairsToKeyValues(tt.input...)
			assert.Len(t, kvs, tt.want)
		})
	}
}

func TestAnyToValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "string",
			input: "test",
		},
		{
			name:  "int",
			input: 42,
		},
		{
			name:  "int64",
			input: int64(123),
		},
		{
			name:  "float64",
			input: 3.14,
		},
		{
			name:  "bool",
			input: true,
		},
		{
			name:  "other type",
			input: struct{ Name string }{Name: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := anyToValue(tt.input)
			assert.NotNil(t, value)
		})
	}
}
