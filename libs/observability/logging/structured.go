package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// StructuredLogger implements Logger with structured logging
type StructuredLogger struct {
	config Config
	level  Level
	output io.Writer
	fields map[string]interface{}
	mu     sync.Mutex
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config Config) *StructuredLogger {
	var output io.Writer
	switch config.Output {
	case "stderr":
		output = os.Stderr
	default:
		output = os.Stdout
	}

	return &StructuredLogger{
		config: config,
		level:  ParseLevel(config.Level),
		output: output,
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > DebugLevel {
		return
	}
	l.log(ctx, DebugLevel, msg, keysAndValues...)
}

// Info logs an info message
func (l *StructuredLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > InfoLevel {
		return
	}
	l.log(ctx, InfoLevel, msg, keysAndValues...)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > WarnLevel {
		return
	}
	l.log(ctx, WarnLevel, msg, keysAndValues...)
}

// Error logs an error message
func (l *StructuredLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.level > ErrorLevel {
		return
	}
	l.log(ctx, ErrorLevel, msg, keysAndValues...)
}

// With returns a logger with additional fields
func (l *StructuredLogger) With(keysAndValues ...interface{}) Logger {
	newLogger := &StructuredLogger{
		config: l.config,
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			newLogger.fields[key] = keysAndValues[i+1]
		}
	}

	return newLogger
}

// Sync flushes any buffered log entries
func (l *StructuredLogger) Sync() error {
	if syncer, ok := l.output.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// log writes a log entry
func (l *StructuredLogger) log(ctx context.Context, level Level, msg string, keysAndValues ...interface{}) {
	entry := make(map[string]interface{})

	// Add timestamp
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)

	// Add level
	entry["level"] = level.String()

	// Add service name
	if l.config.ServiceName != "" {
		entry["service"] = l.config.ServiceName
	}

	// Add message
	entry["message"] = msg

	// Add existing fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Add context fields (trace_id, span_id, etc.)
	// TODO: Extract from context when tracing is implemented

	// Add key-value pairs
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			entry[key] = keysAndValues[i+1]
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config.Format == "json" {
		l.writeJSON(entry)
	} else {
		l.writeText(entry)
	}
}

// writeJSON writes a log entry in JSON format
func (l *StructuredLogger) writeJSON(entry map[string]interface{}) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}
	_, _ = l.output.Write(data)
	_, _ = l.output.Write([]byte("\n"))
}

// writeText writes a log entry in text format
func (l *StructuredLogger) writeText(entry map[string]interface{}) {
	// Format: timestamp level [service] message key=value key=value
	timestamp := entry["timestamp"]
	level := entry["level"]
	service := entry["service"]
	message := entry["message"]

	output := fmt.Sprintf("%s %s", timestamp, level)
	if service != nil {
		output += fmt.Sprintf(" [%s]", service)
	}
	output += fmt.Sprintf(" %s", message)

	// Add other fields
	for k, v := range entry {
		if k == "timestamp" || k == "level" || k == "service" || k == "message" {
			continue
		}
		output += fmt.Sprintf(" %s=%v", k, v)
	}

	_, _ = l.output.Write([]byte(output))
	_, _ = l.output.Write([]byte("\n"))
}
