package health

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test-service")

	if config.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName to be 'test-service', got %s", config.ServiceName)
	}

	if config.CheckInterval != 5*time.Second {
		t.Errorf("Expected CheckInterval to be 5s, got %v", config.CheckInterval)
	}

	if config.DefaultTimeout != 100*time.Millisecond {
		t.Errorf("Expected DefaultTimeout to be 100ms, got %v", config.DefaultTimeout)
	}

	if config.HealthyScore != 0.8 {
		t.Errorf("Expected HealthyScore to be 0.8, got %v", config.HealthyScore)
	}

	if config.DegradedScore != 0.5 {
		t.Errorf("Expected DegradedScore to be 0.5, got %v", config.DegradedScore)
	}

	if config.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold to be 3, got %v", config.FailureThreshold)
	}

	// Validate default config is valid
	if err := config.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestDefaultLivenessConfig(t *testing.T) {
	config := DefaultLivenessConfig()

	if config.HeartbeatInterval != 1*time.Second {
		t.Errorf("Expected HeartbeatInterval to be 1s, got %v", config.HeartbeatInterval)
	}

	if config.HeartbeatTimeout != 10*time.Second {
		t.Errorf("Expected HeartbeatTimeout to be 10s, got %v", config.HeartbeatTimeout)
	}

	if config.MemoryLimit != 4*1024*1024*1024 {
		t.Errorf("Expected MemoryLimit to be 4GB, got %v", config.MemoryLimit)
	}

	if config.GoroutineLimit != 10000 {
		t.Errorf("Expected GoroutineLimit to be 10000, got %v", config.GoroutineLimit)
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Default liveness config should be valid, got error: %v", err)
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.MaxFailures != 3 {
		t.Errorf("Expected MaxFailures to be 3, got %v", config.MaxFailures)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout to be 30s, got %v", config.Timeout)
	}

	if config.HalfOpenTimeout != 10*time.Second {
		t.Errorf("Expected HalfOpenTimeout to be 10s, got %v", config.HalfOpenTimeout)
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Default circuit breaker config should be valid, got error: %v", err)
	}
}

func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %v", config.MaxRetries)
	}

	if config.InitialBackoff != 1*time.Second {
		t.Errorf("Expected InitialBackoff to be 1s, got %v", config.InitialBackoff)
	}

	if config.MaxBackoff != 30*time.Second {
		t.Errorf("Expected MaxBackoff to be 30s, got %v", config.MaxBackoff)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %v", config.BackoffFactor)
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Default recovery config should be valid, got error: %v", err)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"HEALTH_CHECK_INTERVAL",
		"HEALTH_CHECK_TIMEOUT",
		"HEALTH_FAILURE_THRESHOLD",
		"HEALTH_HEALTHY_SCORE",
		"HEALTH_DEGRADED_SCORE",
		"LIVENESS_HEARTBEAT_INTERVAL",
		"LIVENESS_HEARTBEAT_TIMEOUT",
		"LIVENESS_MEMORY_LIMIT",
		"LIVENESS_GOROUTINE_LIMIT",
		"CIRCUIT_BREAKER_MAX_FAILURES",
		"CIRCUIT_BREAKER_TIMEOUT",
		"CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT",
		"AUTO_RECOVERY_ENABLED",
		"AUTO_RECOVERY_MAX_RETRIES",
		"AUTO_RECOVERY_INITIAL_BACKOFF",
		"AUTO_RECOVERY_MAX_BACKOFF",
		"AUTO_RECOVERY_BACKOFF_FACTOR",
	}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Cleanup function to restore environment
	cleanup := func() {
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}
	defer cleanup()

	t.Run("default values when no env vars set", func(t *testing.T) {
		// Clear all env vars
		for _, key := range envVars {
			os.Unsetenv(key)
		}

		config, err := LoadConfigFromEnv("test-service")
		if err != nil {
			t.Fatalf("LoadConfigFromEnv failed: %v", err)
		}

		defaultConfig := DefaultConfig("test-service")
		if config.CheckInterval != defaultConfig.CheckInterval {
			t.Errorf("Expected CheckInterval to be %v, got %v", defaultConfig.CheckInterval, config.CheckInterval)
		}
	})

	t.Run("override with environment variables", func(t *testing.T) {
		os.Setenv("HEALTH_CHECK_INTERVAL", "10s")
		os.Setenv("HEALTH_CHECK_TIMEOUT", "200ms")
		os.Setenv("HEALTH_FAILURE_THRESHOLD", "5")
		os.Setenv("HEALTH_HEALTHY_SCORE", "0.9")
		os.Setenv("HEALTH_DEGRADED_SCORE", "0.6")
		os.Setenv("LIVENESS_HEARTBEAT_INTERVAL", "2s")
		os.Setenv("LIVENESS_HEARTBEAT_TIMEOUT", "20s")
		os.Setenv("LIVENESS_MEMORY_LIMIT", "8589934592") // 8GB
		os.Setenv("LIVENESS_GOROUTINE_LIMIT", "20000")
		os.Setenv("CIRCUIT_BREAKER_MAX_FAILURES", "5")
		os.Setenv("CIRCUIT_BREAKER_TIMEOUT", "60s")
		os.Setenv("CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT", "20s")
		os.Setenv("AUTO_RECOVERY_ENABLED", "false")
		os.Setenv("AUTO_RECOVERY_MAX_RETRIES", "5")
		os.Setenv("AUTO_RECOVERY_INITIAL_BACKOFF", "2s")
		os.Setenv("AUTO_RECOVERY_MAX_BACKOFF", "60s")
		os.Setenv("AUTO_RECOVERY_BACKOFF_FACTOR", "3.0")

		config, err := LoadConfigFromEnv("test-service")
		if err != nil {
			t.Fatalf("LoadConfigFromEnv failed: %v", err)
		}

		if config.CheckInterval != 10*time.Second {
			t.Errorf("Expected CheckInterval to be 10s, got %v", config.CheckInterval)
		}

		if config.DefaultTimeout != 200*time.Millisecond {
			t.Errorf("Expected DefaultTimeout to be 200ms, got %v", config.DefaultTimeout)
		}

		if config.FailureThreshold != 5 {
			t.Errorf("Expected FailureThreshold to be 5, got %v", config.FailureThreshold)
		}

		if config.HealthyScore != 0.9 {
			t.Errorf("Expected HealthyScore to be 0.9, got %v", config.HealthyScore)
		}

		if config.DegradedScore != 0.6 {
			t.Errorf("Expected DegradedScore to be 0.6, got %v", config.DegradedScore)
		}

		if config.LivenessConfig.HeartbeatInterval != 2*time.Second {
			t.Errorf("Expected HeartbeatInterval to be 2s, got %v", config.LivenessConfig.HeartbeatInterval)
		}

		if config.LivenessConfig.HeartbeatTimeout != 20*time.Second {
			t.Errorf("Expected HeartbeatTimeout to be 20s, got %v", config.LivenessConfig.HeartbeatTimeout)
		}

		if config.LivenessConfig.MemoryLimit != 8589934592 {
			t.Errorf("Expected MemoryLimit to be 8GB, got %v", config.LivenessConfig.MemoryLimit)
		}

		if config.LivenessConfig.GoroutineLimit != 20000 {
			t.Errorf("Expected GoroutineLimit to be 20000, got %v", config.LivenessConfig.GoroutineLimit)
		}

		if config.CircuitBreakerConfig.MaxFailures != 5 {
			t.Errorf("Expected MaxFailures to be 5, got %v", config.CircuitBreakerConfig.MaxFailures)
		}

		if config.CircuitBreakerConfig.Timeout != 60*time.Second {
			t.Errorf("Expected Timeout to be 60s, got %v", config.CircuitBreakerConfig.Timeout)
		}

		if config.CircuitBreakerConfig.HalfOpenTimeout != 20*time.Second {
			t.Errorf("Expected HalfOpenTimeout to be 20s, got %v", config.CircuitBreakerConfig.HalfOpenTimeout)
		}

		if config.RecoveryConfig.Enabled {
			t.Error("Expected Enabled to be false")
		}

		if config.RecoveryConfig.MaxRetries != 5 {
			t.Errorf("Expected MaxRetries to be 5, got %v", config.RecoveryConfig.MaxRetries)
		}

		if config.RecoveryConfig.InitialBackoff != 2*time.Second {
			t.Errorf("Expected InitialBackoff to be 2s, got %v", config.RecoveryConfig.InitialBackoff)
		}

		if config.RecoveryConfig.MaxBackoff != 60*time.Second {
			t.Errorf("Expected MaxBackoff to be 60s, got %v", config.RecoveryConfig.MaxBackoff)
		}

		if config.RecoveryConfig.BackoffFactor != 3.0 {
			t.Errorf("Expected BackoffFactor to be 3.0, got %v", config.RecoveryConfig.BackoffFactor)
		}
	})

	t.Run("invalid environment variables", func(t *testing.T) {
		testCases := []struct {
			name   string
			envVar string
			value  string
		}{
			{"invalid duration", "HEALTH_CHECK_INTERVAL", "invalid"},
			{"invalid int", "HEALTH_FAILURE_THRESHOLD", "not-a-number"},
			{"invalid float", "HEALTH_HEALTHY_SCORE", "not-a-float"},
			{"invalid bool", "AUTO_RECOVERY_ENABLED", "not-a-bool"},
			{"invalid uint", "LIVENESS_MEMORY_LIMIT", "-1"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Clear all env vars
				for _, key := range envVars {
					os.Unsetenv(key)
				}

				os.Setenv(tc.envVar, tc.value)

				_, err := LoadConfigFromEnv("test-service")
				if err == nil {
					t.Errorf("Expected error for invalid %s, got nil", tc.envVar)
				}
			})
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultConfig("test-service")
		if err := config.Validate(); err != nil {
			t.Errorf("Valid config should not return error, got: %v", err)
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		config := DefaultConfig("")
		if err := config.Validate(); err == nil {
			t.Error("Expected error for empty service name")
		}
	})

	t.Run("invalid check interval", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.CheckInterval = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero check interval")
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.DefaultTimeout = -1
		if err := config.Validate(); err == nil {
			t.Error("Expected error for negative timeout")
		}
	})

	t.Run("invalid healthy score", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.HealthyScore = 1.5
		if err := config.Validate(); err == nil {
			t.Error("Expected error for healthy score > 1")
		}
	})

	t.Run("invalid degraded score", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.DegradedScore = -0.1
		if err := config.Validate(); err == nil {
			t.Error("Expected error for negative degraded score")
		}
	})

	t.Run("degraded score >= healthy score", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.DegradedScore = 0.9
		config.HealthyScore = 0.8
		if err := config.Validate(); err == nil {
			t.Error("Expected error when degraded score >= healthy score")
		}
	})

	t.Run("invalid failure threshold", func(t *testing.T) {
		config := DefaultConfig("test-service")
		config.FailureThreshold = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero failure threshold")
		}
	})
}

func TestLivenessConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultLivenessConfig()
		if err := config.Validate(); err != nil {
			t.Errorf("Valid config should not return error, got: %v", err)
		}
	})

	t.Run("invalid heartbeat interval", func(t *testing.T) {
		config := DefaultLivenessConfig()
		config.HeartbeatInterval = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero heartbeat interval")
		}
	})

	t.Run("invalid heartbeat timeout", func(t *testing.T) {
		config := DefaultLivenessConfig()
		config.HeartbeatTimeout = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero heartbeat timeout")
		}
	})

	t.Run("timeout <= interval", func(t *testing.T) {
		config := DefaultLivenessConfig()
		config.HeartbeatInterval = 10 * time.Second
		config.HeartbeatTimeout = 5 * time.Second
		if err := config.Validate(); err == nil {
			t.Error("Expected error when timeout <= interval")
		}
	})

	t.Run("zero memory limit", func(t *testing.T) {
		config := DefaultLivenessConfig()
		config.MemoryLimit = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero memory limit")
		}
	})

	t.Run("invalid goroutine limit", func(t *testing.T) {
		config := DefaultLivenessConfig()
		config.GoroutineLimit = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero goroutine limit")
		}
	})
}

func TestCircuitBreakerConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		if err := config.Validate(); err != nil {
			t.Errorf("Valid config should not return error, got: %v", err)
		}
	})

	t.Run("invalid max failures", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		config.MaxFailures = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero max failures")
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		config.Timeout = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero timeout")
		}
	})

	t.Run("invalid half-open timeout", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		config.HalfOpenTimeout = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero half-open timeout")
		}
	})
}

func TestRecoveryConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		if err := config.Validate(); err != nil {
			t.Errorf("Valid config should not return error, got: %v", err)
		}
	})

	t.Run("disabled recovery skips validation", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.Enabled = false
		config.MaxRetries = 0 // Invalid, but should be ignored
		if err := config.Validate(); err != nil {
			t.Errorf("Disabled recovery should skip validation, got error: %v", err)
		}
	})

	t.Run("invalid max retries", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.MaxRetries = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero max retries")
		}
	})

	t.Run("invalid initial backoff", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.InitialBackoff = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero initial backoff")
		}
	})

	t.Run("invalid max backoff", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.MaxBackoff = 0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for zero max backoff")
		}
	})

	t.Run("max backoff < initial backoff", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.InitialBackoff = 10 * time.Second
		config.MaxBackoff = 5 * time.Second
		if err := config.Validate(); err == nil {
			t.Error("Expected error when max backoff < initial backoff")
		}
	})

	t.Run("invalid backoff factor", func(t *testing.T) {
		config := DefaultRecoveryConfig()
		config.BackoffFactor = 1.0
		if err := config.Validate(); err == nil {
			t.Error("Expected error for backoff factor <= 1.0")
		}
	})
}
