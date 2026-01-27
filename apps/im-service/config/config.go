package config

import (
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/config"
)

// Config holds all configuration for im-service
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database configuration
	Database config.DatabaseConfig `mapstructure:"database"`

	// Redis configuration
	Redis config.RedisConfig `mapstructure:"redis"`

	// Kafka configuration
	Kafka config.KafkaConfig `mapstructure:"kafka"`

	// Etcd configuration
	Etcd EtcdConfig `mapstructure:"etcd"`

	// Sensitive Word Filter configuration
	SensitiveWordFilter SensitiveWordFilterConfig `mapstructure:"sensitive_word_filter"`

	// Observability configuration
	Observability config.ObservabilityConfig `mapstructure:"observability"`

	// Offline Worker configuration
	OfflineWorker OfflineWorkerConfig `mapstructure:"offline_worker"`

	// Read Receipt configuration
	ReadReceipt ReadReceiptConfig `mapstructure:"read_receipt"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// GRPCPort is the gRPC server port
	GRPCPort int `mapstructure:"grpc_port" validate:"required,min=1,max=65535"`
	// HTTPPort is the HTTP server port
	HTTPPort int `mapstructure:"http_port" validate:"required,min=1,max=65535"`
}

// OfflineWorkerConfig holds offline worker configuration
type OfflineWorkerConfig struct {
	// Enabled indicates if the offline worker is enabled
	Enabled bool `mapstructure:"enabled"`
	// BatchSize is the number of messages to batch before writing
	BatchSize int `mapstructure:"batch_size"`
	// BatchTimeout is the maximum time to wait before writing a batch
	BatchTimeout time.Duration `mapstructure:"batch_timeout"`
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int `mapstructure:"max_retries"`
	// RetryBackoff is the backoff durations for retries
	RetryBackoff []time.Duration `mapstructure:"retry_backoff"`
	// MessageTTL is the time-to-live for messages
	MessageTTL time.Duration `mapstructure:"message_ttl"`
}

// ReadReceiptConfig holds read receipt configuration
type ReadReceiptConfig struct {
	// KafkaEnabled indicates if Kafka is enabled for read receipts
	KafkaEnabled bool `mapstructure:"kafka_enabled"`
	// Topic is the Kafka topic for read receipt events
	Topic string `mapstructure:"topic"`
}

// EtcdConfig holds etcd configuration
type EtcdConfig struct {
	// Endpoints is the list of etcd endpoints
	Endpoints []string `mapstructure:"endpoints"`
	// TTL is the time-to-live for registry entries
	TTL time.Duration `mapstructure:"ttl"`
}

// SensitiveWordFilterConfig holds sensitive word filter configuration
type SensitiveWordFilterConfig struct {
	// Enabled indicates if the filter is enabled
	Enabled bool `mapstructure:"enabled"`
	// DefaultAction is the default action to take (block, replace, audit)
	DefaultAction string `mapstructure:"default_action"`
	// WordLists is a map of language to word list file path
	WordLists map[string]string `mapstructure:"word_lists"`
	// CaseSensitive indicates if matching is case-sensitive
	CaseSensitive bool `mapstructure:"case_sensitive"`
	// NormalizeUnicode indicates if Unicode normalization is enabled
	NormalizeUnicode bool `mapstructure:"normalize_unicode"`
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
	setIMServiceDefaults(loader)

	// Load configuration
	cfg := &Config{}
	if err := loader.LoadInto(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// setIMServiceDefaults sets im-service specific defaults
func setIMServiceDefaults(loader *config.Loader) {
	// Server defaults
	loader.SetDefault("server.grpc_port", 9094)
	loader.SetDefault("server.http_port", 8080)

	// Database defaults
	loader.SetDefault("database.host", "localhost")
	loader.SetDefault("database.port", 3306)
	loader.SetDefault("database.user", "im_service")
	loader.SetDefault("database.password", "im_password")
	loader.SetDefault("database.database", "im_chat")
	loader.SetDefault("database.max_open_conns", 25)
	loader.SetDefault("database.max_idle_conns", 5)
	loader.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Redis defaults
	loader.SetDefault("redis.addr", "localhost:6379")
	loader.SetDefault("redis.password", "")
	loader.SetDefault("redis.db", 2)

	// Kafka defaults
	loader.SetDefault("kafka.brokers", []string{"localhost:9092"})
	loader.SetDefault("kafka.consumer_group", "im-service-offline-workers")
	loader.SetDefault("kafka.topic", "offline_msg")

	// Offline Worker defaults
	loader.SetDefault("offline_worker.enabled", true)
	loader.SetDefault("offline_worker.batch_size", 100)
	loader.SetDefault("offline_worker.batch_timeout", 5*time.Second)
	loader.SetDefault("offline_worker.max_retries", 5)
	loader.SetDefault("offline_worker.retry_backoff", []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	})
	loader.SetDefault("offline_worker.message_ttl", 7*24*time.Hour)

	// Read Receipt defaults
	loader.SetDefault("read_receipt.kafka_enabled", true)
	loader.SetDefault("read_receipt.topic", "read_receipt_events")

	// Etcd defaults
	loader.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	loader.SetDefault("etcd.ttl", 90*time.Second)

	// Sensitive Word Filter defaults
	loader.SetDefault("sensitive_word_filter.enabled", false)
	loader.SetDefault("sensitive_word_filter.default_action", "replace")
	loader.SetDefault("sensitive_word_filter.word_lists", map[string]string{})
	loader.SetDefault("sensitive_word_filter.case_sensitive", false)
	loader.SetDefault("sensitive_word_filter.normalize_unicode", true)

	// Observability defaults
	loader.SetDefault("observability.service_name", "im-service")
	loader.SetDefault("observability.service_version", "1.0.0")
	loader.SetDefault("observability.environment", "development")
	loader.SetDefault("observability.enable_metrics", true)
	loader.SetDefault("observability.metrics_port", 9090)
	loader.SetDefault("observability.log_level", "info")
	loader.SetDefault("observability.log_format", "json")
}

// GetDatabaseDSN returns the MySQL DSN string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
	)
}
