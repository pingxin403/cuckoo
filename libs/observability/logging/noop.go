package logging

import "context"

// NoOpLogger is a no-op implementation of Logger
type NoOpLogger struct{}

// NewNoOpLogger creates a new no-op logger
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Debug does nothing
func (n *NoOpLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// Info does nothing
func (n *NoOpLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// Warn does nothing
func (n *NoOpLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// Error does nothing
func (n *NoOpLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// With returns the same no-op logger
func (n *NoOpLogger) With(keysAndValues ...interface{}) Logger {
	return n
}

// Sync does nothing
func (n *NoOpLogger) Sync() error {
	return nil
}
