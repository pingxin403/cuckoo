package logging

import "context"

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message with key-value pairs
	Debug(ctx context.Context, msg string, keysAndValues ...interface{})

	// Info logs an info message with key-value pairs
	Info(ctx context.Context, msg string, keysAndValues ...interface{})

	// Warn logs a warning message with key-value pairs
	Warn(ctx context.Context, msg string, keysAndValues ...interface{})

	// Error logs an error message with key-value pairs
	Error(ctx context.Context, msg string, keysAndValues ...interface{})

	// With returns a logger with additional fields
	With(keysAndValues ...interface{}) Logger

	// Sync flushes any buffered log entries
	Sync() error
}

// Config holds configuration for logger
type Config struct {
	ServiceName string
	Level       string
	Format      string // "json" or "text"
	Output      string // "stdout" or "stderr"
}

// Level represents log level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// ParseLevel parses a log level string
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// String returns the string representation of a level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "info"
	}
}
