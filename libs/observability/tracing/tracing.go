package tracing

import (
	"context"
)

// Tracer defines the interface for distributed tracing
type Tracer interface {
	// StartSpan starts a new span
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// Shutdown gracefully shuts down the tracer
	Shutdown(ctx context.Context) error
}

// Span represents a span in a trace
type Span interface {
	// End ends the span
	End()

	// SetAttribute sets an attribute on the span
	SetAttribute(key string, value interface{})

	// SetAttributes sets multiple attributes on the span
	SetAttributes(attributes map[string]interface{})

	// RecordError records an error on the span
	RecordError(err error)

	// SetStatus sets the status of the span
	SetStatus(code StatusCode, description string)
}

// StatusCode represents the status of a span
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// SpanOption configures a span
type SpanOption func(*SpanConfig)

// SpanConfig holds configuration for a span
type SpanConfig struct {
	Attributes map[string]interface{}
	Kind       SpanKind
}

// SpanKind represents the kind of span
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
)

// WithAttributes returns a SpanOption that sets attributes
func WithAttributes(attributes map[string]interface{}) SpanOption {
	return func(c *SpanConfig) {
		if c.Attributes == nil {
			c.Attributes = make(map[string]interface{})
		}
		for k, v := range attributes {
			c.Attributes[k] = v
		}
	}
}

// WithSpanKind returns a SpanOption that sets the span kind
func WithSpanKind(kind SpanKind) SpanOption {
	return func(c *SpanConfig) {
		c.Kind = kind
	}
}

// Config holds configuration for tracer
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	SampleRate     float64
}
