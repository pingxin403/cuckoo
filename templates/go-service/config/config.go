package config

import (
	"fmt"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config 服务配置结构体
type Config struct {
	// Server 服务器配置
	Server config.ServerConfig `mapstructure:"server"`

	// Observability 可观测性配置
	Observability config.ObservabilityConfig `mapstructure:"observability"`

	// 服务特定配置可在此添加
}

// Load 加载配置
func Load() (*Config, error) {
	loader, err := config.Load(config.Options{
		ServiceName: "", // 使用默认 "config" 文件名
		ConfigType:  "yaml",
		ConfigPaths: []string{".", "./config"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	// 设置默认值
	config.SetCommonDefaults(loader)
	setServiceDefaults(loader)

	// 加载配置
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setServiceDefaults 设置服务特定默认值
func setServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.port", {{GRPC_PORT}})

	// Observability defaults
	loader.SetDefault("observability.service_name", "{{SERVICE_NAME}}")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}
