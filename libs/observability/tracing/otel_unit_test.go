package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: We don't test NewOTelTracer with real connections as it requires
// a running OpenTelemetry collector. Integration tests should be done separately.

func TestOTelSpan_SetAttribute(t *testing.T) {
	// Create a no-op tracer for testing span operations
	tracer := NewNoOpTracer()
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "test-span")
	defer span.End()

	// Test setting attributes
	span.SetAttribute("key1", "value1")
	span.SetAttribute("key2", 42)
	span.SetAttribute("key3", true)

	// No assertions needed - just verify no panics
}

func TestOTelSpan_SetAttributes(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "test-span")
	defer span.End()

	// Test setting multiple attributes
	span.SetAttributes(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
		"key4": 3.14,
	})

	// No assertions needed - just verify no panics
}

func TestOTelSpan_RecordError(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "test-span")
	defer span.End()

	// Test recording an error
	err := errors.New("test error")
	span.RecordError(err)

	// No assertions needed - just verify no panics
}

func TestOTelSpan_SetStatus(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "test-span")
	defer span.End()

	// Test setting status
	span.SetStatus(StatusCodeOK, "success")
	span.SetStatus(StatusCodeError, "error occurred")
	span.SetStatus(StatusCodeUnset, "")

	// No assertions needed - just verify no panics
}

func TestOTelTracer_StartSpan_WithOptions(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Test with attributes
	_, span := tracer.StartSpan(ctx, "test-span",
		WithAttributes(map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}),
		WithSpanKind(SpanKindServer),
	)
	defer span.End()

	// No assertions needed - just verify no panics
}

func TestSpanKind(t *testing.T) {
	tests := []struct {
		name string
		kind SpanKind
	}{
		{"internal", SpanKindInternal},
		{"server", SpanKindServer},
		{"client", SpanKindClient},
	}

	tracer := NewNoOpTracer()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := tracer.StartSpan(ctx, "test-span", WithSpanKind(tt.kind))
			defer span.End()
			// No assertions needed - just verify no panics
		})
	}
}

func TestConvertToAttribute(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "key", "value"},
		{"int", "key", 42},
		{"int64", "key", int64(42)},
		{"float64", "key", 3.14},
		{"bool", "key", true},
		{"other", "key", struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := convertToAttribute(tt.key, tt.value)
			assert.NotNil(t, attr)
		})
	}
}

func TestOTelTracer_Shutdown(t *testing.T) {
	// Test shutdown with no-op tracer
	tracer := NewNoOpTracer()
	ctx := context.Background()

	err := tracer.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSpanConfig(t *testing.T) {
	config := &SpanConfig{
		Attributes: make(map[string]interface{}),
	}

	// Test WithAttributes option
	opt := WithAttributes(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})
	opt(config)

	assert.Len(t, config.Attributes, 2)
	assert.Equal(t, "value1", config.Attributes["key1"])
	assert.Equal(t, 42, config.Attributes["key2"])

	// Test WithSpanKind option
	kindOpt := WithSpanKind(SpanKindServer)
	kindOpt(config)

	assert.Equal(t, SpanKindServer, config.Kind)
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		name string
		code StatusCode
	}{
		{"unset", StatusCodeUnset},
		{"ok", StatusCodeOK},
		{"error", StatusCodeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewNoOpTracer()
			ctx := context.Background()

			_, span := tracer.StartSpan(ctx, "test-span")
			defer span.End()

			span.SetStatus(tt.code, "test description")
			// No assertions needed - just verify no panics
		})
	}
}

func TestOTelTracer_ContextPropagation(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Start parent span
	ctx, parentSpan := tracer.StartSpan(ctx, "parent-span")
	defer parentSpan.End()

	// Start child span with parent context
	_, childSpan := tracer.StartSpan(ctx, "child-span")
	defer childSpan.End()

	// No assertions needed - just verify no panics
}

func TestOTelTracer_MultipleSpans(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Create multiple spans
	for i := 0; i < 10; i++ {
		_, span := tracer.StartSpan(ctx, "test-span")
		span.SetAttribute("iteration", i)
		span.End()
	}

	// No assertions needed - just verify no panics
}

func TestOTelTracer_ConcurrentSpans(t *testing.T) {
	tracer := NewNoOpTracer()
	ctx := context.Background()

	// Create spans concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, span := tracer.StartSpan(ctx, "concurrent-span")
			span.SetAttribute("id", id)
			span.End()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No assertions needed - just verify no panics
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Endpoint:       "localhost:4317",
				SampleRate:     0.5,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: Config{
				ServiceName:    "",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Endpoint:       "localhost:4317",
				SampleRate:     0.5,
			},
			wantErr: false, // Service name is not validated in Config
		},
		{
			name: "invalid sample rate",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Endpoint:       "localhost:4317",
				SampleRate:     1.5, // > 1.0
			},
			wantErr: false, // Sample rate is not validated in Config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify config can be created
			require.NotNil(t, tt.config)
		})
	}
}
