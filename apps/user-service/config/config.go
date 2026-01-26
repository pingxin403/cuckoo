package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for user-service
type Config struct {
	// Server configuration
	Server config.ServerConfig `mapstructure:"server"`

	// Database configuration
	Database config.DatabaseConfig `mapstructure:"database"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	loader, err := config.Load(config.Options{
		ServiceName: "user-service",
		ConfigType:  "yaml",
		ConfigPaths: []string{".", "./config"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	// Set defaults
	config.SetCommonDefaults(loader)
	setUserServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setUserServiceDefaults sets user-service specific defaults
func setUserServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.port", 9096)

	// Database defaults
	loader.SetDefault("database.host", "localhost")
	loader.SetDefault("database.port", 3306)
	loader.SetDefault("database.user", "im_service")
	loader.SetDefault("database.password", "im_password")
	loader.SetDefault("database.database", "im_chat")

	// Observability defaults
	loader.SetDefault("observability.service_name", "user-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}

// GetDSN returns the MySQL DSN string
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
	)
}
