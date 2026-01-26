package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for auth-service
type Config struct {
	// Server configuration
	Server config.ServerConfig `mapstructure:"server"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`

	// JWT configuration
	JWT JWTConfig `mapstructure:"jwt"`
}

// JWTConfig holds JWT-specific configuration
type JWTConfig struct {
	// Secret is the secret key for signing JWT tokens
	Secret string `mapstructure:"secret" validate:"required"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	loader, err := config.Load(config.Options{
		ServiceName: "auth-service",
		ConfigType:  "yaml",
		ConfigPaths: []string{".", "./config"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	// Set defaults
	config.SetCommonDefaults(loader)
	setAuthServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setAuthServiceDefaults sets auth-service specific defaults
func setAuthServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.port", 9095)

	// Observability defaults
	loader.SetDefault("observability.service_name", "auth-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}
