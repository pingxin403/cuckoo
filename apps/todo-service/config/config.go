package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for todo-service
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`
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
	setTodoServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setTodoServiceDefaults sets todo-service specific defaults
func setTodoServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.grpc_port", 9091)
	loader.SetDefault("server.http_port", 8080)
	loader.SetDefault("server.host", "0.0.0.0")

	// Observability defaults
	loader.SetDefault("observability.service_name", "todo-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}
