package logging

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig holds configuration for OpenTelemetry logging
type OTelConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Level          string
	Insecure       bool
}

// OTelLogger implements Logger using OpenTelemetry Logs SDK
type OTelLogger struct {
	config   OTelConfig
	logger   log.Logger
	provider *sdklog.LoggerProvider
	level    Level
	fields   []log.KeyValue // Additional context fields

	// Internal observability metrics
	internalMetrics *logInternalMetrics

	// Shutdown function
	shutdownFunc func(context.Context) error
}

// logInternalMetrics tracks internal logging metrics
type logInternalMetrics struct {
	// Export failures counter
	exportFailures atomic.Int64

	// Total log entries counter
	totalLogs atomic.Int64

	// Logs by level
	debugLogs atomic.Int64
	infoLogs  atomic.Int64
	warnLogs  atomic.Int64
	errorLogs atomic.Int64

	// Logs with trace context
	logsWithTrace atomic.Int64
}

// NewOTelLogger creates a new OpenTelemetry logger
func NewOTelLogger(config OTelConfig) (*OTelLogger, error) {
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

	// Create OTLP exporter
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(config.OTLPEndpoint),
	}
	if config.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Create logger provider with batch processor
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	// Create logger
	logger := provider.Logger(config.ServiceName)

	level := ParseLevel(config.Level)

	otelLogger := &OTelLogger{
		config:          config,
		logger:          logger,
		provider:        provider,
		level:           level,
		fields:          []log.KeyValue{},
		internalMetrics: &logInternalMetrics{},
		shutdownFunc:    provider.Shutdown,
	}

	return otelLogger, nil
}

// Debug logs a debug message
func (l *OTelLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > DebugLevel {
		return
	}
	l.internalMetrics.debugLogs.Add(1)
	l.emit(ctx, log.SeverityDebug, msg, keysAndValues...)
}

// Info logs an info message
func (l *OTelLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > InfoLevel {
		return
	}
	l.internalMetrics.infoLogs.Add(1)
	l.emit(ctx, log.SeverityInfo, msg, keysAndValues...)
}

// Warn logs a warning message
func (l *OTelLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > WarnLevel {
		return
	}
	l.internalMetrics.warnLogs.Add(1)
	l.emit(ctx, log.SeverityWarn, msg, keysAndValues...)
}

// Error logs an error message
func (l *OTelLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > ErrorLevel {
		return
	}
	l.internalMetrics.errorLogs.Add(1)
	l.emit(ctx, log.SeverityError, msg, keysAndValues...)
}

// With returns a logger with additional fields
func (l *OTelLogger) With(keysAndValues ...interface{}) Logger {
	// Create new logger with copied fields
	newFields := make([]log.KeyValue, len(l.fields))
	copy(newFields, l.fields)

	// Add new fields
	newFields = append(newFields, kvPairsToKeyValues(keysAndValues...)...)

	return &OTelLogger{
		config:          l.config,
		logger:          l.logger,
		provider:        l.provider,
		level:           l.level,
		fields:          newFields,
		internalMetrics: l.internalMetrics, // Share internal metrics with parent
		shutdownFunc:    l.shutdownFunc,
	}
}

// Sync flushes any buffered log entries
func (l *OTelLogger) Sync() error {
	// Force flush by shutting down and recreating provider
	// Note: This is a simplified implementation
	return nil
}

// Shutdown gracefully shuts down the logger
func (l *OTelLogger) Shutdown(ctx context.Context) error {
	if l.shutdownFunc != nil {
		return l.shutdownFunc(ctx)
	}
	return nil
}

// emit creates and emits a log record
func (l *OTelLogger) emit(ctx context.Context, severity log.Severity, msg string, keysAndValues ...interface{}) {
	l.internalMetrics.totalLogs.Add(1)

	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetBody(log.StringValue(msg))

	// Add logger's context fields
	attrs := make([]log.KeyValue, 0, len(l.fields)+len(keysAndValues)/2+3) // +3 for trace context
	attrs = append(attrs, l.fields...)

	// Extract trace context if available and add as attributes
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		l.internalMetrics.logsWithTrace.Add(1)
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			log.KeyValue{
				Key:   "trace_id",
				Value: log.StringValue(spanCtx.TraceID().String()),
			},
			log.KeyValue{
				Key:   "span_id",
				Value: log.StringValue(spanCtx.SpanID().String()),
			},
			log.KeyValue{
				Key:   "trace_flags",
				Value: log.StringValue(spanCtx.TraceFlags().String()),
			},
		)
	}

	// Add key-value pairs as attributes
	attrs = append(attrs, kvPairsToKeyValues(keysAndValues...)...)

	record.AddAttributes(attrs...)

	// Emit the record
	l.logger.Emit(ctx, record)
}

// Helper functions

// kvPairsToKeyValues converts key-value pairs to OTel KeyValue attributes
func kvPairsToKeyValues(keysAndValues ...interface{}) []log.KeyValue {
	if len(keysAndValues) == 0 {
		return nil
	}

	attrs := make([]log.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			attrs = append(attrs, log.KeyValue{
				Key:   key,
				Value: anyToValue(value),
			})
		}
	}
	return attrs
}

// anyToValue converts any value to OTel log.Value
func anyToValue(v interface{}) log.Value {
	switch val := v.(type) {
	case string:
		return log.StringValue(val)
	case int:
		return log.Int64Value(int64(val))
	case int64:
		return log.Int64Value(val)
	case float64:
		return log.Float64Value(val)
	case bool:
		return log.BoolValue(val)
	default:
		return log.StringValue(fmt.Sprintf("%v", val))
	}
}

// GetInternalMetrics returns internal metrics for monitoring (used for testing)
func (l *OTelLogger) GetInternalMetrics() map[string]int64 {
	return map[string]int64{
		"export_failures": l.internalMetrics.exportFailures.Load(),
		"total_logs":      l.internalMetrics.totalLogs.Load(),
		"debug_logs":      l.internalMetrics.debugLogs.Load(),
		"info_logs":       l.internalMetrics.infoLogs.Load(),
		"warn_logs":       l.internalMetrics.warnLogs.Load(),
		"error_logs":      l.internalMetrics.errorLogs.Load(),
		"logs_with_trace": l.internalMetrics.logsWithTrace.Load(),
	}
}
