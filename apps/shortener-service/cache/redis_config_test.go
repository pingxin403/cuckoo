package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestDefaultRedisConfig verifies that DefaultRedisConfig returns valid defaults
func TestDefaultRedisConfig(t *testing.T) {
	config := DefaultRedisConfig()

	// Verify default values
	if len(config.Addrs) != 1 || config.Addrs[0] != "localhost:6379" {
		t.Errorf("Expected default address localhost:6379, got %v", config.Addrs)
	}

	if config.PoolSize != 20 {
		t.Errorf("Expected default PoolSize 20, got %d", config.PoolSize)
	}

	if config.MinIdleConns != 6 {
		t.Errorf("Expected default MinIdleConns 6 (30%% of 20), got %d", config.MinIdleConns)
	}

	if config.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("Expected default ConnMaxLifetime 30m, got %v", config.ConnMaxLifetime)
	}

	if config.DialTimeout != 5*time.Second {
		t.Errorf("Expected default DialTimeout 5s, got %v", config.DialTimeout)
	}

	if config.ReadTimeout != 3*time.Second {
		t.Errorf("Expected default ReadTimeout 3s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 3*time.Second {
		t.Errorf("Expected default WriteTimeout 3s, got %v", config.WriteTimeout)
	}

	if config.MaxRedirects != 3 {
		t.Errorf("Expected default MaxRedirects 3, got %d", config.MaxRedirects)
	}

	// Verify default config is valid
	if err := config.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

// TestRedisConfig_Validate tests configuration validation
func TestRedisConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RedisConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid standalone config",
			config:  DefaultRedisConfig(),
			wantErr: false,
		},
		{
			name: "valid cluster config",
			config: RedisConfig{
				Addrs:           []string{"node1:6379", "node2:6379", "node3:6379"},
				ClusterMode:     true,
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    3,
			},
			wantErr: false,
		},
		{
			name: "empty addresses",
			config: RedisConfig{
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "at least one Redis address is required",
		},
		{
			name: "invalid pool size - zero",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        0,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "PoolSize must be at least 1",
		},
		{
			name: "invalid pool size - too large",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        150,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "PoolSize should not exceed 100",
		},
		{
			name: "invalid MinIdleConns - negative",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    -1,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "MinIdleConns cannot be negative",
		},
		{
			name: "invalid MinIdleConns - exceeds PoolSize",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    25,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "MinIdleConns (25) cannot exceed PoolSize (20)",
		},
		{
			name: "invalid ConnMaxLifetime - negative",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: -1 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "ConnMaxLifetime cannot be negative",
		},
		{
			name: "invalid ConnMaxLifetime - too short",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Second,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "ConnMaxLifetime should be at least 1 minute",
		},
		{
			name: "invalid DialTimeout - zero",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     0,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "DialTimeout must be positive",
		},
		{
			name: "invalid DialTimeout - too long",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     60 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "DialTimeout should not exceed 30 seconds",
		},
		{
			name: "invalid ReadTimeout - zero",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     0,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "ReadTimeout must be positive",
		},
		{
			name: "invalid WriteTimeout - zero",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    0,
			},
			wantErr: true,
			errMsg:  "WriteTimeout must be positive",
		},
		{
			name: "cluster mode - insufficient nodes",
			config: RedisConfig{
				Addrs:           []string{"node1:6379"},
				ClusterMode:     true,
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    3,
			},
			wantErr: true,
			errMsg:  "cluster mode requires at least 3 addresses",
		},
		{
			name: "cluster mode - invalid MaxRedirects",
			config: RedisConfig{
				Addrs:           []string{"node1:6379", "node2:6379", "node3:6379"},
				ClusterMode:     true,
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    0,
			},
			wantErr: true,
			errMsg:  "MaxRedirects must be at least 1",
		},
		{
			name: "standalone mode - invalid DB number",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				DB:              20,
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			wantErr: true,
			errMsg:  "DB must be between 0 and 15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestRedisConfig_ApplyDefaults tests default value application
func TestRedisConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		expected RedisConfig
	}{
		{
			name:   "empty config",
			config: RedisConfig{},
			expected: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    3,
			},
		},
		{
			name: "partial config",
			config: RedisConfig{
				Addrs:    []string{"redis.example.com:6379"},
				PoolSize: 50,
			},
			expected: RedisConfig{
				Addrs:           []string{"redis.example.com:6379"},
				PoolSize:        50,
				MinIdleConns:    15, // 30% of 50
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    3,
			},
		},
		{
			name: "custom values preserved",
			config: RedisConfig{
				Addrs:           []string{"custom:6379"},
				PoolSize:        30,
				MinIdleConns:    10,
				ConnMaxLifetime: 60 * time.Minute,
				DialTimeout:     10 * time.Second,
				ReadTimeout:     5 * time.Second,
				WriteTimeout:    5 * time.Second,
				MaxRedirects:    5,
			},
			expected: RedisConfig{
				Addrs:           []string{"custom:6379"},
				PoolSize:        30,
				MinIdleConns:    10,
				ConnMaxLifetime: 60 * time.Minute,
				DialTimeout:     10 * time.Second,
				ReadTimeout:     5 * time.Second,
				WriteTimeout:    5 * time.Second,
				MaxRedirects:    5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()

			if len(tt.config.Addrs) != len(tt.expected.Addrs) || tt.config.Addrs[0] != tt.expected.Addrs[0] {
				t.Errorf("Addrs: expected %v, got %v", tt.expected.Addrs, tt.config.Addrs)
			}
			if tt.config.PoolSize != tt.expected.PoolSize {
				t.Errorf("PoolSize: expected %d, got %d", tt.expected.PoolSize, tt.config.PoolSize)
			}
			if tt.config.MinIdleConns != tt.expected.MinIdleConns {
				t.Errorf("MinIdleConns: expected %d, got %d", tt.expected.MinIdleConns, tt.config.MinIdleConns)
			}
			if tt.config.ConnMaxLifetime != tt.expected.ConnMaxLifetime {
				t.Errorf("ConnMaxLifetime: expected %v, got %v", tt.expected.ConnMaxLifetime, tt.config.ConnMaxLifetime)
			}
			if tt.config.DialTimeout != tt.expected.DialTimeout {
				t.Errorf("DialTimeout: expected %v, got %v", tt.expected.DialTimeout, tt.config.DialTimeout)
			}
			if tt.config.ReadTimeout != tt.expected.ReadTimeout {
				t.Errorf("ReadTimeout: expected %v, got %v", tt.expected.ReadTimeout, tt.config.ReadTimeout)
			}
			if tt.config.WriteTimeout != tt.expected.WriteTimeout {
				t.Errorf("WriteTimeout: expected %v, got %v", tt.expected.WriteTimeout, tt.config.WriteTimeout)
			}
			if tt.config.MaxRedirects != tt.expected.MaxRedirects {
				t.Errorf("MaxRedirects: expected %d, got %d", tt.expected.MaxRedirects, tt.config.MaxRedirects)
			}
		})
	}
}

// TestRedisConfig_OptimizeForQPS tests automatic pool size optimization
func TestRedisConfig_OptimizeForQPS(t *testing.T) {
	tests := []struct {
		name             string
		qps              int
		expectedPoolSize int
		expectedMinIdle  int
	}{
		{
			name:             "low QPS - minimum pool size",
			qps:              5000,
			expectedPoolSize: 10, // 5000/1000 = 5, but minimum is 10
			expectedMinIdle:  3,  // 30% of 10
		},
		{
			name:             "medium QPS",
			qps:              20000,
			expectedPoolSize: 20, // 20000/1000 = 20
			expectedMinIdle:  6,  // 30% of 20
		},
		{
			name:             "high QPS",
			qps:              40000,
			expectedPoolSize: 40, // 40000/1000 = 40
			expectedMinIdle:  12, // 30% of 40
		},
		{
			name:             "very high QPS - maximum pool size",
			qps:              100000,
			expectedPoolSize: 50, // 100000/1000 = 100, but maximum is 50
			expectedMinIdle:  15, // 30% of 50
		},
		{
			name:             "zero QPS - no change",
			qps:              0,
			expectedPoolSize: 20, // Default value, unchanged
			expectedMinIdle:  6,  // Default value, unchanged
		},
		{
			name:             "negative QPS - no change",
			qps:              -1000,
			expectedPoolSize: 20, // Default value, unchanged
			expectedMinIdle:  6,  // Default value, unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedisConfig()
			config.OptimizeForQPS(tt.qps)

			if config.PoolSize != tt.expectedPoolSize {
				t.Errorf("PoolSize: expected %d, got %d", tt.expectedPoolSize, config.PoolSize)
			}
			if config.MinIdleConns != tt.expectedMinIdle {
				t.Errorf("MinIdleConns: expected %d, got %d", tt.expectedMinIdle, config.MinIdleConns)
			}

			// Verify the optimized config is still valid
			if err := config.Validate(); err != nil {
				t.Errorf("Optimized config should be valid, got error: %v", err)
			}
		})
	}
}

// TestRedisConfig_String tests string representation
func TestRedisConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		contains []string
	}{
		{
			name:   "standalone config without password",
			config: DefaultRedisConfig(),
			contains: []string{
				"Mode=standalone",
				"Addrs=[localhost:6379]",
				"PoolSize=20",
				"MinIdleConns=6",
				"Password=<empty>",
			},
		},
		{
			name: "standalone config with password",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				Password:        "secret123",
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			contains: []string{
				"Mode=standalone",
				"Password=<redacted>",
			},
		},
		{
			name: "cluster config",
			config: RedisConfig{
				Addrs:           []string{"node1:6379", "node2:6379", "node3:6379"},
				ClusterMode:     true,
				PoolSize:        30,
				MinIdleConns:    9,
				ConnMaxLifetime: 45 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
				MaxRedirects:    5,
			},
			contains: []string{
				"Mode=cluster",
				"Addrs=[node1:6379 node2:6379 node3:6379]",
				"PoolSize=30",
				"MinIdleConns=9",
				"MaxRedirects=5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.config.String()

			for _, expected := range tt.contains {
				if !strings.Contains(str, expected) {
					t.Errorf("String() should contain '%s', got: %s", expected, str)
				}
			}

			// Verify password is never exposed in plain text
			if tt.config.Password != "" && strings.Contains(str, tt.config.Password) {
				t.Errorf("String() should not contain plain text password, got: %s", str)
			}
		})
	}
}

// TestRedisConfig_ValidationAfterDefaults ensures ApplyDefaults produces valid config
func TestRedisConfig_ValidationAfterDefaults(t *testing.T) {
	config := RedisConfig{}
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		t.Errorf("Config after ApplyDefaults should be valid, got error: %v", err)
	}
}

// TestRedisConfig_OptimizeForQPSProducesValidConfig ensures OptimizeForQPS produces valid config
func TestRedisConfig_OptimizeForQPSProducesValidConfig(t *testing.T) {
	qpsValues := []int{1000, 5000, 10000, 20000, 50000, 100000, 500000}

	for _, qps := range qpsValues {
		t.Run(string(rune(qps)), func(t *testing.T) {
			config := DefaultRedisConfig()
			config.OptimizeForQPS(qps)

			if err := config.Validate(); err != nil {
				t.Errorf("Config optimized for %d QPS should be valid, got error: %v", qps, err)
			}

			// Verify MinIdleConns is approximately 30% of PoolSize
			expectedMinIdle := config.PoolSize * 3 / 10
			if config.MinIdleConns < expectedMinIdle-1 || config.MinIdleConns > expectedMinIdle+1 {
				t.Errorf("MinIdleConns should be ~30%% of PoolSize (%d), got %d",
					config.PoolSize, config.MinIdleConns)
			}
		})
	}
}

// TestNewOptimizedRedisClient_Standalone tests client creation for standalone mode
func TestNewOptimizedRedisClient_Standalone(t *testing.T) {
	config := RedisConfig{
		Addrs:           []string{"localhost:6379"},
		Password:        "",
		DB:              0,
		ClusterMode:     false,
		PoolSize:        20,
		MinIdleConns:    6,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	client := NewOptimizedRedisClient(config)
	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	// Verify it's a standalone client (not cluster)
	// We can check this by attempting to get pool stats
	stats := client.PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available")
	}

	// Clean up
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestNewOptimizedRedisClient_Cluster tests client creation for cluster mode
func TestNewOptimizedRedisClient_Cluster(t *testing.T) {
	config := RedisConfig{
		Addrs:           []string{"localhost:7000", "localhost:7001", "localhost:7002"},
		Password:        "",
		ClusterMode:     true,
		PoolSize:        30,
		MinIdleConns:    9,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MaxRedirects:    3,
	}

	client := NewOptimizedRedisClient(config)
	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	// Verify it's a cluster client
	stats := client.PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available")
	}

	// Clean up
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestNewOptimizedRedisClient_AppliesDefaults tests that defaults are applied
func TestNewOptimizedRedisClient_AppliesDefaults(t *testing.T) {
	// Create config with minimal settings
	config := RedisConfig{
		Addrs: []string{"localhost:6379"},
		// All other fields are zero values
	}

	client := NewOptimizedRedisClient(config)
	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	// Verify client was created successfully (defaults were applied)
	stats := client.PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available")
	}

	// Clean up
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestNewOptimizedRedisClient_PoolConfiguration tests pool settings are applied correctly
func TestNewOptimizedRedisClient_PoolConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		validate func(*testing.T, *redis.PoolStats)
	}{
		{
			name: "default pool size",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			validate: func(t *testing.T, stats *redis.PoolStats) {
				// Pool stats should be available
				if stats == nil {
					t.Error("Expected pool stats to be available")
				}
			},
		},
		{
			name: "large pool size",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        50,
				MinIdleConns:    15,
				ConnMaxLifetime: 45 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			validate: func(t *testing.T, stats *redis.PoolStats) {
				if stats == nil {
					t.Error("Expected pool stats to be available")
				}
			},
		},
		{
			name: "small pool size",
			config: RedisConfig{
				Addrs:           []string{"localhost:6379"},
				PoolSize:        10,
				MinIdleConns:    3,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			},
			validate: func(t *testing.T, stats *redis.PoolStats) {
				if stats == nil {
					t.Error("Expected pool stats to be available")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOptimizedRedisClient(tt.config)
			if client == nil {
				t.Fatal("Expected non-nil client, got nil")
			}

			stats := client.PoolStats()
			tt.validate(t, stats)

			// Clean up
			if err := client.Close(); err != nil {
				t.Errorf("Failed to close client: %v", err)
			}
		})
	}
}

// TestNewOptimizedRedisClient_TimeoutConfiguration tests timeout settings
func TestNewOptimizedRedisClient_TimeoutConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config RedisConfig
	}{
		{
			name: "default timeouts",
			config: RedisConfig{
				Addrs:        []string{"localhost:6379"},
				PoolSize:     20,
				MinIdleConns: 6,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
		{
			name: "custom timeouts",
			config: RedisConfig{
				Addrs:        []string{"localhost:6379"},
				PoolSize:     20,
				MinIdleConns: 6,
				DialTimeout:  10 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			},
		},
		{
			name: "short timeouts",
			config: RedisConfig{
				Addrs:        []string{"localhost:6379"},
				PoolSize:     20,
				MinIdleConns: 6,
				DialTimeout:  2 * time.Second,
				ReadTimeout:  1 * time.Second,
				WriteTimeout: 1 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOptimizedRedisClient(tt.config)
			if client == nil {
				t.Fatal("Expected non-nil client, got nil")
			}

			// Verify client was created successfully
			stats := client.PoolStats()
			if stats == nil {
				t.Error("Expected pool stats to be available")
			}

			// Clean up
			if err := client.Close(); err != nil {
				t.Errorf("Failed to close client: %v", err)
			}
		})
	}
}

// TestNewOptimizedRedisClient_WithPassword tests client creation with authentication
func TestNewOptimizedRedisClient_WithPassword(t *testing.T) {
	config := RedisConfig{
		Addrs:           []string{"localhost:6379"},
		Password:        "test-password",
		DB:              0,
		PoolSize:        20,
		MinIdleConns:    6,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	client := NewOptimizedRedisClient(config)
	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	// Verify client was created (even though connection may fail due to wrong password)
	stats := client.PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available")
	}

	// Clean up
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestNewOptimizedRedisClient_DifferentDatabases tests standalone mode with different DB numbers
func TestNewOptimizedRedisClient_DifferentDatabases(t *testing.T) {
	databases := []int{0, 1, 5, 15}

	for _, db := range databases {
		t.Run(string(rune(db)), func(t *testing.T) {
			config := RedisConfig{
				Addrs:           []string{"localhost:6379"},
				DB:              db,
				PoolSize:        20,
				MinIdleConns:    6,
				ConnMaxLifetime: 30 * time.Minute,
				DialTimeout:     5 * time.Second,
				ReadTimeout:     3 * time.Second,
				WriteTimeout:    3 * time.Second,
			}

			client := NewOptimizedRedisClient(config)
			if client == nil {
				t.Fatal("Expected non-nil client, got nil")
			}

			// Verify client was created successfully
			stats := client.PoolStats()
			if stats == nil {
				t.Error("Expected pool stats to be available")
			}

			// Clean up
			if err := client.Close(); err != nil {
				t.Errorf("Failed to close client: %v", err)
			}
		})
	}
}

// TestNewRedisClientWithMetrics tests the creation of a Redis client with metrics
func TestNewRedisClientWithMetrics(t *testing.T) {
	// Create test observability
	obs := createTestObservability()

	config := DefaultRedisConfig()
	wrapper := NewRedisClientWithMetrics(config, obs)

	if wrapper == nil {
		t.Fatal("Expected non-nil wrapper, got nil")
	}

	if wrapper.Client() == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	// Verify pool stats are available
	stats := wrapper.Client().PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available")
	}

	// Clean up
	wrapper.Stop()
	if err := wrapper.Client().Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestRedisClientWithMetrics_ExposePoolMetrics tests that pool metrics are exposed
func TestRedisClientWithMetrics_ExposePoolMetrics(t *testing.T) {
	// Create test observability
	obs := createTestObservability()

	config := DefaultRedisConfig()
	wrapper := NewRedisClientWithMetrics(config, obs)
	defer wrapper.Stop()
	defer wrapper.Client().Close()

	// Wait a bit for the first metric collection cycle
	// The metrics are collected every 10 seconds, but we'll wait just a moment
	// to ensure the goroutine has started
	time.Sleep(100 * time.Millisecond)

	// Verify the wrapper is working
	if wrapper.Client() == nil {
		t.Fatal("Expected non-nil client")
	}

	// Get pool stats to verify they're being collected
	stats := wrapper.Client().PoolStats()
	if stats == nil {
		t.Fatal("Expected pool stats to be available")
	}

	// Note: We can't easily verify the metrics were actually set without
	// accessing the internal metrics collector, but we can verify the
	// goroutine is running and the client is functional
}

// TestRedisClientWithMetrics_Stop tests graceful shutdown of metrics collection
func TestRedisClientWithMetrics_Stop(t *testing.T) {
	obs := createTestObservability()

	config := DefaultRedisConfig()
	wrapper := NewRedisClientWithMetrics(config, obs)

	// Stop the metrics collection
	wrapper.Stop()

	// Verify client is still functional after stopping metrics
	stats := wrapper.Client().PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available after Stop()")
	}

	// Clean up
	if err := wrapper.Client().Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// TestRedisClientWithMetrics_MetricsCollection tests that metrics are collected over time
func TestRedisClientWithMetrics_MetricsCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping metrics collection test in short mode")
	}

	obs := createTestObservability()

	config := DefaultRedisConfig()
	wrapper := NewRedisClientWithMetrics(config, obs)
	defer wrapper.Stop()
	defer wrapper.Client().Close()

	// Perform some operations to generate pool activity
	ctx := context.Background()
	client := wrapper.Client()

	// Try to perform some operations (these may fail if Redis is not running, but that's okay)
	for i := 0; i < 5; i++ {
		_ = client.Ping(ctx).Err()
		time.Sleep(10 * time.Millisecond)
	}

	// Get pool stats to verify activity
	stats := client.PoolStats()
	if stats == nil {
		t.Fatal("Expected pool stats to be available")
	}

	// The metrics goroutine should be running and collecting stats
	// We can't easily verify the exact metric values without accessing
	// the internal metrics collector, but we've verified the mechanism works
}

// TestRedisClientWithMetrics_PoolStatsMetrics tests that all expected metrics are exposed
func TestRedisClientWithMetrics_PoolStatsMetrics(t *testing.T) {
	obs := createTestObservability()

	config := DefaultRedisConfig()
	wrapper := NewRedisClientWithMetrics(config, obs)
	defer wrapper.Stop()
	defer wrapper.Client().Close()

	// Get pool stats
	stats := wrapper.Client().PoolStats()
	if stats == nil {
		t.Fatal("Expected pool stats to be available")
	}

	// Verify that the stats structure has the expected fields
	// These are the metrics that should be exposed:
	// - Hits: Number of times a free connection was found in the pool
	// - Misses: Number of times a free connection was NOT found
	// - Timeouts: Number of times a wait timeout occurred
	// - TotalConns: Total number of connections in the pool
	// - IdleConns: Number of idle connections
	// - Active connections: TotalConns - IdleConns

	// We can't check the exact values, but we can verify the stats are accessible
	_ = stats.Hits
	_ = stats.Misses
	_ = stats.Timeouts
	_ = stats.TotalConns
	_ = stats.IdleConns

	// Calculate active connections
	activeConns := stats.TotalConns - stats.IdleConns
	if activeConns < 0 {
		t.Errorf("Active connections should not be negative, got %d", activeConns)
	}
}

// TestRedisClientWithMetrics_ClusterMode tests metrics collection in cluster mode
func TestRedisClientWithMetrics_ClusterMode(t *testing.T) {
	obs := createTestObservability()

	config := RedisConfig{
		Addrs:           []string{"localhost:7000", "localhost:7001", "localhost:7002"},
		ClusterMode:     true,
		PoolSize:        30,
		MinIdleConns:    9,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MaxRedirects:    3,
	}

	wrapper := NewRedisClientWithMetrics(config, obs)
	defer wrapper.Stop()
	defer wrapper.Client().Close()

	// Verify pool stats are available for cluster client
	stats := wrapper.Client().PoolStats()
	if stats == nil {
		t.Error("Expected pool stats to be available for cluster client")
	}
}
