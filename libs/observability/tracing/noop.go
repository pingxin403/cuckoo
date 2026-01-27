package tracing

import "context"

// NoOpTracer is a no-op implementation of Tracer
type NoOpTracer struct{}

// NewNoOpTracer creates a new no-op tracer
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// StartSpan returns a no-op span
func (n *NoOpTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &NoOpSpan{}
}

// Shutdown does nothing
func (n *NoOpTracer) Shutdown(ctx context.Context) error {
	return nil
}

// NoOpSpan is a no-op implementation of Span
type NoOpSpan struct{}

// End does nothing
func (n *NoOpSpan) End() {}

// SetAttribute does nothing
func (n *NoOpSpan) SetAttribute(key string, value interface{}) {}

// SetAttributes does nothing
func (n *NoOpSpan) SetAttributes(attributes map[string]interface{}) {}

// RecordError does nothing
func (n *NoOpSpan) RecordError(err error) {}

// SetStatus does nothing
func (n *NoOpSpan) SetStatus(code StatusCode, description string) {}
