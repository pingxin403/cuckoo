package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for im-gateway-service
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Redis configuration
	Redis config.RedisConfig `mapstructure:"redis"`

	// Kafka configuration
	Kafka config.KafkaConfig `mapstructure:"kafka"`

	// Etcd configuration
	Etcd config.EtcdConfig `mapstructure:"etcd"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`

	// Service Discovery configuration
	ServiceDiscovery ServiceDiscoveryConfig `mapstructure:"service_discovery"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// GRPCPort is the gRPC server port
	GRPCPort int `mapstructure:"grpc_port" validate:"required,min=1,max=65535"`
	// HTTPPort is the HTTP/WebSocket server port
	HTTPPort int `mapstructure:"http_port" validate:"required,min=1,max=65535"`
}

// ServiceDiscoveryConfig holds service discovery configuration
type ServiceDiscoveryConfig struct {
	// AuthServiceAddr is the address of the auth service
	AuthServiceAddr string `mapstructure:"auth_service_addr"`
	// IMServiceAddr is the address of the IM service
	IMServiceAddr string `mapstructure:"im_service_addr"`
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
	setIMGatewayServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setIMGatewayServiceDefaults sets im-gateway-service specific defaults
func setIMGatewayServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.grpc_port", 9097)
	loader.SetDefault("server.http_port", 8080)

	// Redis defaults
	loader.SetDefault("redis.addr", "localhost:6379")
	loader.SetDefault("redis.password", "")
	loader.SetDefault("redis.db", 0)

	// Kafka defaults
	loader.SetDefault("kafka.brokers", []string{"localhost:9092"})
	loader.SetDefault("kafka.topic", "offline_msg")

	// Etcd defaults
	loader.SetDefault("etcd.endpoints", []string{"localhost:2379"})

	// Service Discovery defaults
	loader.SetDefault("service_discovery.auth_service_addr", "localhost:9095")
	loader.SetDefault("service_discovery.im_service_addr", "localhost:9094")

	// Observability defaults
	loader.SetDefault("observability.service_name", "im-gateway-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}
