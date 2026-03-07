package health

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:      serviceName,
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
		FailureThreshold: 3,
		LivenessConfig:   DefaultLivenessConfig(),
		CircuitBreakerConfig: DefaultCircuitBreakerConfig(),
		RecoveryConfig:   DefaultRecoveryConfig(),
	}
}

// DefaultLivenessConfig returns LivenessConfig with sensible defaults
func DefaultLivenessConfig() LivenessConfig {
	return LivenessConfig{
		HeartbeatInterval: 1 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
		MemoryLimit:       4 * 1024 * 1024 * 1024, // 4GB
		GoroutineLimit:    10000,
	}
}

// DefaultCircuitBreakerConfig returns CircuitBreakerConfig with sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:     3,
		Timeout:         30 * time.Second,
		HalfOpenTimeout: 10 * time.Second,
	}
}

// DefaultRecoveryConfig returns RecoveryConfig with sensible defaults
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		Enabled:        true,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
}

// LoadConfigFromEnv loads configuration from environment variables
// It starts with default values and overrides with environment variables if present
func LoadConfigFromEnv(serviceName string) (Config, error) {
	config := DefaultConfig(serviceName)

	// Health check configuration
	if val := os.Getenv("HEALTH_CHECK_INTERVAL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid HEALTH_CHECK_INTERVAL: %w", err)
		}
		config.CheckInterval = duration
	}

	if val := os.Getenv("HEALTH_CHECK_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid HEALTH_CHECK_TIMEOUT: %w", err)
		}
		config.DefaultTimeout = duration
	}

	if val := os.Getenv("HEALTH_FAILURE_THRESHOLD"); val != "" {
		threshold, err := strconv.Atoi(val)
		if err != nil {
			return config, fmt.Errorf("invalid HEALTH_FAILURE_THRESHOLD: %w", err)
		}
		config.FailureThreshold = threshold
	}

	if val := os.Getenv("HEALTH_HEALTHY_SCORE"); val != "" {
		score, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return config, fmt.Errorf("invalid HEALTH_HEALTHY_SCORE: %w", err)
		}
		config.HealthyScore = score
	}

	if val := os.Getenv("HEALTH_DEGRADED_SCORE"); val != "" {
		score, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return config, fmt.Errorf("invalid HEALTH_DEGRADED_SCORE: %w", err)
		}
		config.DegradedScore = score
	}

	// Liveness probe configuration
	if val := os.Getenv("LIVENESS_HEARTBEAT_INTERVAL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid LIVENESS_HEARTBEAT_INTERVAL: %w", err)
		}
		config.LivenessConfig.HeartbeatInterval = duration
	}

	if val := os.Getenv("LIVENESS_HEARTBEAT_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid LIVENESS_HEARTBEAT_TIMEOUT: %w", err)
		}
		config.LivenessConfig.HeartbeatTimeout = duration
	}

	if val := os.Getenv("LIVENESS_MEMORY_LIMIT"); val != "" {
		limit, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return config, fmt.Errorf("invalid LIVENESS_MEMORY_LIMIT: %w", err)
		}
		config.LivenessConfig.MemoryLimit = limit
	}

	if val := os.Getenv("LIVENESS_GOROUTINE_LIMIT"); val != "" {
		limit, err := strconv.Atoi(val)
		if err != nil {
			return config, fmt.Errorf("invalid LIVENESS_GOROUTINE_LIMIT: %w", err)
		}
		config.LivenessConfig.GoroutineLimit = limit
	}

	// Circuit breaker configuration
	if val := os.Getenv("CIRCUIT_BREAKER_MAX_FAILURES"); val != "" {
		maxFailures, err := strconv.Atoi(val)
		if err != nil {
			return config, fmt.Errorf("invalid CIRCUIT_BREAKER_MAX_FAILURES: %w", err)
		}
		config.CircuitBreakerConfig.MaxFailures = maxFailures
	}

	if val := os.Getenv("CIRCUIT_BREAKER_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid CIRCUIT_BREAKER_TIMEOUT: %w", err)
		}
		config.CircuitBreakerConfig.Timeout = duration
	}

	if val := os.Getenv("CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT: %w", err)
		}
		config.CircuitBreakerConfig.HalfOpenTimeout = duration
	}

	// Auto-recovery configuration
	if val := os.Getenv("AUTO_RECOVERY_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return config, fmt.Errorf("invalid AUTO_RECOVERY_ENABLED: %w", err)
		}
		config.RecoveryConfig.Enabled = enabled
	}

	if val := os.Getenv("AUTO_RECOVERY_MAX_RETRIES"); val != "" {
		maxRetries, err := strconv.Atoi(val)
		if err != nil {
			return config, fmt.Errorf("invalid AUTO_RECOVERY_MAX_RETRIES: %w", err)
		}
		config.RecoveryConfig.MaxRetries = maxRetries
	}

	if val := os.Getenv("AUTO_RECOVERY_INITIAL_BACKOFF"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid AUTO_RECOVERY_INITIAL_BACKOFF: %w", err)
		}
		config.RecoveryConfig.InitialBackoff = duration
	}

	if val := os.Getenv("AUTO_RECOVERY_MAX_BACKOFF"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return config, fmt.Errorf("invalid AUTO_RECOVERY_MAX_BACKOFF: %w", err)
		}
		config.RecoveryConfig.MaxBackoff = duration
	}

	if val := os.Getenv("AUTO_RECOVERY_BACKOFF_FACTOR"); val != "" {
		factor, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return config, fmt.Errorf("invalid AUTO_RECOVERY_BACKOFF_FACTOR: %w", err)
		}
		config.RecoveryConfig.BackoffFactor = factor
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return config, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("ServiceName is required")
	}

	if c.CheckInterval <= 0 {
		return fmt.Errorf("CheckInterval must be positive, got %v", c.CheckInterval)
	}

	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("DefaultTimeout must be positive, got %v", c.DefaultTimeout)
	}

	if c.HealthyScore < 0 || c.HealthyScore > 1 {
		return fmt.Errorf("HealthyScore must be between 0 and 1, got %v", c.HealthyScore)
	}

	if c.DegradedScore < 0 || c.DegradedScore > 1 {
		return fmt.Errorf("DegradedScore must be between 0 and 1, got %v", c.DegradedScore)
	}

	if c.DegradedScore >= c.HealthyScore {
		return fmt.Errorf("DegradedScore (%v) must be less than HealthyScore (%v)", c.DegradedScore, c.HealthyScore)
	}

	if c.FailureThreshold < 1 {
		return fmt.Errorf("FailureThreshold must be at least 1, got %v", c.FailureThreshold)
	}

	// Validate LivenessConfig
	if err := c.LivenessConfig.Validate(); err != nil {
		return fmt.Errorf("invalid LivenessConfig: %w", err)
	}

	// Validate CircuitBreakerConfig
	if err := c.CircuitBreakerConfig.Validate(); err != nil {
		return fmt.Errorf("invalid CircuitBreakerConfig: %w", err)
	}

	// Validate RecoveryConfig
	if err := c.RecoveryConfig.Validate(); err != nil {
		return fmt.Errorf("invalid RecoveryConfig: %w", err)
	}

	return nil
}

// Validate checks if the LivenessConfig is valid
func (lc LivenessConfig) Validate() error {
	if lc.HeartbeatInterval <= 0 {
		return fmt.Errorf("HeartbeatInterval must be positive, got %v", lc.HeartbeatInterval)
	}

	if lc.HeartbeatTimeout <= 0 {
		return fmt.Errorf("HeartbeatTimeout must be positive, got %v", lc.HeartbeatTimeout)
	}

	if lc.HeartbeatTimeout <= lc.HeartbeatInterval {
		return fmt.Errorf("HeartbeatTimeout (%v) must be greater than HeartbeatInterval (%v)", lc.HeartbeatTimeout, lc.HeartbeatInterval)
	}

	if lc.MemoryLimit == 0 {
		return fmt.Errorf("MemoryLimit must be positive")
	}

	if lc.GoroutineLimit <= 0 {
		return fmt.Errorf("GoroutineLimit must be positive, got %v", lc.GoroutineLimit)
	}

	return nil
}

// Validate checks if the CircuitBreakerConfig is valid
func (cbc CircuitBreakerConfig) Validate() error {
	if cbc.MaxFailures < 1 {
		return fmt.Errorf("MaxFailures must be at least 1, got %v", cbc.MaxFailures)
	}

	if cbc.Timeout <= 0 {
		return fmt.Errorf("Timeout must be positive, got %v", cbc.Timeout)
	}

	if cbc.HalfOpenTimeout <= 0 {
		return fmt.Errorf("HalfOpenTimeout must be positive, got %v", cbc.HalfOpenTimeout)
	}

	return nil
}

// Validate checks if the RecoveryConfig is valid
func (rc RecoveryConfig) Validate() error {
	if !rc.Enabled {
		// If recovery is disabled, no need to validate other fields
		return nil
	}

	if rc.MaxRetries < 1 {
		return fmt.Errorf("MaxRetries must be at least 1, got %v", rc.MaxRetries)
	}

	if rc.InitialBackoff <= 0 {
		return fmt.Errorf("InitialBackoff must be positive, got %v", rc.InitialBackoff)
	}

	if rc.MaxBackoff <= 0 {
		return fmt.Errorf("MaxBackoff must be positive, got %v", rc.MaxBackoff)
	}

	if rc.MaxBackoff < rc.InitialBackoff {
		return fmt.Errorf("MaxBackoff (%v) must be greater than or equal to InitialBackoff (%v)", rc.MaxBackoff, rc.InitialBackoff)
	}

	if rc.BackoffFactor <= 1.0 {
		return fmt.Errorf("BackoffFactor must be greater than 1.0, got %v", rc.BackoffFactor)
	}

	return nil
}
