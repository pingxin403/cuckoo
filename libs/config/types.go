package config

import "time"

// ServerConfig 服务器配置
type ServerConfig struct {
	// Port 服务端口
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
	// Host 服务主机
	Host string `mapstructure:"host"`
	// ReadTimeout 读取超时
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout 写入超时
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	// IdleTimeout 空闲超时
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// Host 数据库主机
	Host string `mapstructure:"host" validate:"required"`
	// Port 数据库端口
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
	// User 数据库用户
	User string `mapstructure:"user" validate:"required"`
	// Password 数据库密码
	Password string `mapstructure:"password"`
	// Database 数据库名称
	Database string `mapstructure:"database" validate:"required"`
	// MaxOpenConns 最大打开连接数
	MaxOpenConns int `mapstructure:"max_open_conns"`
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int `mapstructure:"max_idle_conns"`
	// ConnMaxLifetime 连接最大生命周期
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN 构建 MySQL DSN
func (c *DatabaseConfig) DSN() string {
	return c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + string(rune(c.Port)) + ")/" + c.Database + "?parseTime=true&charset=utf8mb4"
}

// RedisConfig Redis 配置
type RedisConfig struct {
	// Addr Redis 地址
	Addr string `mapstructure:"addr" validate:"required"`
	// Password Redis 密码
	Password string `mapstructure:"password"`
	// DB Redis 数据库编号
	DB int `mapstructure:"db"`
	// PoolSize 连接池大小
	PoolSize int `mapstructure:"pool_size"`
	// MinIdleConns 最小空闲连接数
	MinIdleConns int `mapstructure:"min_idle_conns"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	// Brokers Kafka broker 地址列表
	Brokers []string `mapstructure:"brokers" validate:"required,min=1"`
	// ConsumerGroup 消费者组
	ConsumerGroup string `mapstructure:"consumer_group"`
	// Topic 主题
	Topic string `mapstructure:"topic"`
	// Version Kafka 版本
	Version string `mapstructure:"version"`
}

// EtcdConfig Etcd 配置
type EtcdConfig struct {
	// Endpoints Etcd 端点列表
	Endpoints []string `mapstructure:"endpoints" validate:"required,min=1"`
	// DialTimeout 连接超时
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	// Username 用户名
	Username string `mapstructure:"username"`
	// Password 密码
	Password string `mapstructure:"password"`
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	// ServiceName 服务名称
	ServiceName string `mapstructure:"service_name" validate:"required"`
	// ServiceVersion 服务版本
	ServiceVersion string `mapstructure:"service_version"`
	// Environment 环境（development, staging, production）
	Environment string `mapstructure:"environment"`
	// EnableMetrics 是否启用指标
	EnableMetrics bool `mapstructure:"enable_metrics"`
	// MetricsPort 指标端口
	MetricsPort int `mapstructure:"metrics_port"`
	// LogLevel 日志级别
	LogLevel string `mapstructure:"log_level"`
	// LogFormat 日志格式（json, text）
	LogFormat string `mapstructure:"log_format"`
}

// SetDefaults 设置默认值
func SetCommonDefaults(loader *Loader) {
	// 服务器默认值
	loader.SetDefault("server.port", 8080)
	loader.SetDefault("server.host", "0.0.0.0")
	loader.SetDefault("server.read_timeout", 15*time.Second)
	loader.SetDefault("server.write_timeout", 15*time.Second)
	loader.SetDefault("server.idle_timeout", 60*time.Second)

	// 数据库默认值
	loader.SetDefault("database.host", "localhost")
	loader.SetDefault("database.port", 3306)
	loader.SetDefault("database.max_open_conns", 25)
	loader.SetDefault("database.max_idle_conns", 5)
	loader.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Redis 默认值
	loader.SetDefault("redis.addr", "localhost:6379")
	loader.SetDefault("redis.db", 0)
	loader.SetDefault("redis.pool_size", 10)
	loader.SetDefault("redis.min_idle_conns", 5)

	// Kafka 默认值
	loader.SetDefault("kafka.brokers", []string{"localhost:9092"})
	loader.SetDefault("kafka.version", "2.8.0")

	// Etcd 默认值
	loader.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	loader.SetDefault("etcd.dial_timeout", 5*time.Second)

	// 可观测性默认值
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}
