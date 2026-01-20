package logger

import (
	"go.uber.org/zap"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
)

// InitLogger initializes the global logger
// In production, use JSON encoding for structured logs
// In development, use console encoding for readability
func InitLogger(development bool) error {
	var err error
	if development {
		Log, err = zap.NewDevelopment()
	} else {
		Log, err = zap.NewProduction()
	}
	if err != nil {
		return err
	}
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
