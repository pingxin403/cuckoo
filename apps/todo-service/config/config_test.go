package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original APP_ENV
	originalEnv := os.Getenv("APP_ENV")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("APP_ENV", originalEnv)
		} else {
			_ = os.Unsetenv("APP_ENV")
		}
	}()

	t.Run("Load default config", func(t *testing.T) {
		// Set APP_ENV to local
		_ = os.Setenv("APP_ENV", "local")

		cfg, err := Load()
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify defaults
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 9091, cfg.Server.GRPCPort)
		assert.Equal(t, 8080, cfg.Server.HTTPPort)
		assert.Equal(t, "todo-service", cfg.Observability.ServiceName)
	})

	t.Run("Load with environment variable override", func(t *testing.T) {
		// Set environment variable
		_ = os.Setenv("SERVER_GRPC_PORT", "8888")
		defer func() { _ = os.Unsetenv("SERVER_GRPC_PORT") }()

		cfg, err := Load()
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify override
		assert.Equal(t, 8888, cfg.Server.GRPCPort)
	})

	t.Run("Validate required fields", func(t *testing.T) {
		cfg, err := Load()
		require.NoError(t, err)

		// Verify required fields are set
		assert.NotEmpty(t, cfg.Server.Host)
		assert.Greater(t, cfg.Server.GRPCPort, 0)
		assert.Greater(t, cfg.Server.HTTPPort, 0)
		assert.NotEmpty(t, cfg.Observability.ServiceName)
	})
}

func TestSetTodoServiceDefaults(t *testing.T) {
	t.Run("Set defaults", func(t *testing.T) {
		// Create a mock loader (we can't easily test the internal function directly,
		// but we can verify defaults are set through Load())
		cfg, err := Load()
		require.NoError(t, err)

		// Verify all defaults are set
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 9091, cfg.Server.GRPCPort)
		assert.Equal(t, 8080, cfg.Server.HTTPPort)
		assert.Equal(t, "todo-service", cfg.Observability.ServiceName)
		assert.Equal(t, "info", cfg.Observability.LogLevel)
		assert.Equal(t, "json", cfg.Observability.LogFormat)
	})
}

func TestConfigStruct(t *testing.T) {
	t.Run("Config struct fields", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Host:     "localhost",
				GRPCPort: 9091,
				HTTPPort: 8080,
			},
		}

		assert.Equal(t, "localhost", cfg.Server.Host)
		assert.Equal(t, 9091, cfg.Server.GRPCPort)
		assert.Equal(t, 8080, cfg.Server.HTTPPort)
	})
}
