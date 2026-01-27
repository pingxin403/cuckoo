package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStructuredLogger(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
	}

	logger := NewStructuredLogger(config)
	require.NotNil(t, logger)
	assert.Equal(t, config, logger.config)
	assert.Equal(t, InfoLevel, logger.level)
}

func TestStructuredLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{
			ServiceName: "test",
			Level:       "debug",
			Format:      "json",
		},
		level:  DebugLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	ctx := context.Background()

	// Test all log levels
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 4)

	// Verify each line is valid JSON
	for _, line := range lines {
		var entry map[string]interface{}
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err)
		assert.Contains(t, entry, "level")
		assert.Contains(t, entry, "message")
		assert.Contains(t, entry, "timestamp")
	}
}

func TestStructuredLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel Level
		logFunc     func(*StructuredLogger, context.Context)
		shouldLog   bool
	}{
		{
			name:        "debug level logs debug",
			loggerLevel: DebugLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Debug(ctx, "test") },
			shouldLog:   true,
		},
		{
			name:        "info level filters debug",
			loggerLevel: InfoLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Debug(ctx, "test") },
			shouldLog:   false,
		},
		{
			name:        "info level logs info",
			loggerLevel: InfoLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Info(ctx, "test") },
			shouldLog:   true,
		},
		{
			name:        "warn level filters info",
			loggerLevel: WarnLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Info(ctx, "test") },
			shouldLog:   false,
		},
		{
			name:        "error level filters warn",
			loggerLevel: ErrorLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Warn(ctx, "test") },
			shouldLog:   false,
		},
		{
			name:        "error level logs error",
			loggerLevel: ErrorLevel,
			logFunc:     func(l *StructuredLogger, ctx context.Context) { l.Error(ctx, "test") },
			shouldLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &StructuredLogger{
				config: Config{Format: "json"},
				level:  tt.loggerLevel,
				output: &buf,
				fields: make(map[string]interface{}),
			}

			tt.logFunc(logger, context.Background())

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestStructuredLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{
			ServiceName: "test",
			Format:      "json",
		},
		level:  InfoLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	// Create logger with additional fields
	childLogger := logger.With("request_id", "123", "user_id", "456")
	childLogger.Info(context.Background(), "test message")

	output := buf.String()
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	require.NoError(t, err)

	assert.Equal(t, "123", entry["request_id"])
	assert.Equal(t, "456", entry["user_id"])
	assert.Equal(t, "test message", entry["message"])
}

func TestStructuredLogger_KeyValuePairs(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{Format: "json"},
		level:  InfoLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	logger.Info(context.Background(), "test message",
		"key1", "value1",
		"key2", 42,
		"key3", true,
	)

	output := buf.String()
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	require.NoError(t, err)

	assert.Equal(t, "value1", entry["key1"])
	assert.Equal(t, float64(42), entry["key2"]) // JSON numbers are float64
	assert.Equal(t, true, entry["key3"])
}

func TestStructuredLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{
			ServiceName: "test-service",
			Format:      "json",
		},
		level:  InfoLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	logger.Info(context.Background(), "test message", "key", "value")

	output := buf.String()
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	require.NoError(t, err)

	// Verify required fields
	assert.Contains(t, entry, "timestamp")
	assert.Contains(t, entry, "level")
	assert.Contains(t, entry, "service")
	assert.Contains(t, entry, "message")
	assert.Equal(t, "test-service", entry["service"])
	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "test message", entry["message"])
	assert.Equal(t, "value", entry["key"])
}

func TestStructuredLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{
			ServiceName: "test-service",
			Format:      "text",
		},
		level:  InfoLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	logger.Info(context.Background(), "test message", "key", "value")

	output := buf.String()

	// Verify text format contains expected elements
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "[test-service]")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
}

func TestStructuredLogger_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		config: Config{Format: "json"},
		level:  InfoLevel,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	// Simulate concurrent logging
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.Info(context.Background(), "test", "id", id, "count", j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify output is not empty and no panics occurred
	assert.NotEmpty(t, buf.String())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"error", ErrorLevel},
		{"invalid", InfoLevel}, // defaults to info
		{"", InfoLevel},        // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := ParseLevel(tt.input)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}
