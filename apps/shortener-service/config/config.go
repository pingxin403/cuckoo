package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for shortener-service
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database configuration
	Database config.DatabaseConfig `mapstructure:"database"`

	// Redis configuration (optional)
	Redis *config.RedisConfig `mapstructure:"redis"`

	// Kafka configuration (optional)
	Kafka *config.KafkaConfig `mapstructure:"kafka"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`

	// RateLimiter configuration
	RateLimiter RateLimiterConfig `mapstructure:"rate_limiter"`

	// Cache configuration
	Cache CacheConfig `mapstructure:"cache"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// GRPCPort is the port for gRPC server
	GRPCPort int `mapstructure:"grpc_port" validate:"required,min=1,max=65535"`
	// HTTPPort is the port for HTTP server
	HTTPPort int `mapstructure:"http_port" validate:"required,min=1,max=65535"`
	// Host is the host to bind to
	Host string `mapstructure:"host"`
}

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute
	RequestsPerMinute int `mapstructure:"requests_per_minute" validate:"min=1"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	// L1MaxCost is the maximum cost for L1 cache (in bytes)
	L1MaxCost int64 `mapstructure:"l1_max_cost" validate:"min=1"`
	// L1NumCounters is the number of counters for L1 cache
	L1NumCounters int64 `mapstructure:"l1_num_counters" validate:"min=1"`
	// L2TTL is the TTL for L2 cache (in seconds)
	L2TTL int `mapstructure:"l2_ttl" validate:"min=1"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	loader, err := config.Load(config.Options{
		ServiceName: "", // Empty to use default "config" filename
		ConfigType:  "yaml",
		ConfigPaths: []string{".", "./config"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	// Set defaults
	config.SetCommonDefaults(loader)
	setShortenerServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setShortenerServiceDefaults sets shortener-service specific defaults
func setShortenerServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.grpc_port", 50051)
	loader.SetDefault("server.http_port", 8080)
	loader.SetDefault("server.host", "0.0.0.0")

	// Database defaults
	loader.SetDefault("database.database", "shortener")
	loader.SetDefault("database.user", "shortener_user")
	loader.SetDefault("database.password", "shortener_password")

	// Observability defaults
	loader.SetDefault("observability.service_name", "shortener-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")

	// Rate limiter defaults
	loader.SetDefault("rate_limiter.requests_per_minute", 100)

	// Cache defaults
	loader.SetDefault("cache.l1_max_cost", 10485760) // 10MB
	loader.SetDefault("cache.l1_num_counters", 100000)
	loader.SetDefault("cache.l2_ttl", 3600) // 1 hour
}
