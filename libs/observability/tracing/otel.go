package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// OTelTracer implements Tracer using OpenTelemetry
type OTelTracer struct {
	config   Config
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewOTelTracer creates a new OpenTelemetry tracer
func NewOTelTracer(config Config) (*OTelTracer, error) {
	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithGRPCConn(conn),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create trace provider with sampling
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(provider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := provider.Tracer(config.ServiceName)

	return &OTelTracer{
		config:   config,
		provider: provider,
		tracer:   tracer,
	}, nil
}

// StartSpan starts a new span
func (o *OTelTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	// Apply span options
	config := &SpanConfig{
		Attributes: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(config)
	}

	// Convert span kind
	var spanKind trace.SpanKind
	switch config.Kind {
	case SpanKindServer:
		spanKind = trace.SpanKindServer
	case SpanKindClient:
		spanKind = trace.SpanKindClient
	default:
		spanKind = trace.SpanKindInternal
	}

	// Start OpenTelemetry span
	ctx, otelSpan := o.tracer.Start(ctx, name, trace.WithSpanKind(spanKind))

	// Set attributes
	for k, v := range config.Attributes {
		otelSpan.SetAttributes(convertToAttribute(k, v))
	}

	span := &OTelSpan{
		span: otelSpan,
	}

	return ctx, span
}

// Shutdown gracefully shuts down the tracer
func (o *OTelTracer) Shutdown(ctx context.Context) error {
	if o.provider != nil {
		return o.provider.Shutdown(ctx)
	}
	return nil
}

// OTelSpan implements Span using OpenTelemetry
type OTelSpan struct {
	span trace.Span
}

// End ends the span
func (o *OTelSpan) End() {
	o.span.End()
}

// SetAttribute sets an attribute on the span
func (o *OTelSpan) SetAttribute(key string, value interface{}) {
	o.span.SetAttributes(convertToAttribute(key, value))
}

// SetAttributes sets multiple attributes on the span
func (o *OTelSpan) SetAttributes(attributes map[string]interface{}) {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		attrs = append(attrs, convertToAttribute(k, v))
	}
	o.span.SetAttributes(attrs...)
}

// RecordError records an error on the span
func (o *OTelSpan) RecordError(err error) {
	o.span.RecordError(err)
	o.span.SetStatus(codes.Error, err.Error())
}

// SetStatus sets the status of the span
func (o *OTelSpan) SetStatus(code StatusCode, description string) {
	var otelCode codes.Code
	switch code {
	case StatusCodeOK:
		otelCode = codes.Ok
	case StatusCodeError:
		otelCode = codes.Error
	default:
		otelCode = codes.Unset
	}
	o.span.SetStatus(otelCode, description)
}

// convertToAttribute converts a value to an OpenTelemetry attribute
func convertToAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}
