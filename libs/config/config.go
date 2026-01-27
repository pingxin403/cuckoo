package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Loader 配置加载器
type Loader struct {
	v        *viper.Viper
	validate *validator.Validate
	options  Options
}

// Options 配置加载选项
type Options struct {
	// ServiceName 服务名称，用于环境变量前缀
	ServiceName string
	// ConfigFile 配置文件路径（可选）
	ConfigFile string
	// ConfigType 配置文件类型（yaml, json, toml 等）
	ConfigType string
	// ConfigPaths 配置文件搜索路径
	ConfigPaths []string
	// EnvPrefix 环境变量前缀（默认为空）
	EnvPrefix string
	// Environment 环境名称（development, staging, production）
	// 用于加载环境特定的配置文件，如 config.development.yaml
	Environment string
}

// NewLoader 创建新的配置加载器
func NewLoader(opts Options) *Loader {
	v := viper.New()

	// 如果未指定环境，从环境变量读取
	if opts.Environment == "" {
		if env := os.Getenv("APP_ENV"); env != "" {
			opts.Environment = strings.TrimSpace(strings.ToLower(env))
		} else {
			opts.Environment = "local"
		}
	}

	// 设置配置文件类型
	configType := opts.ConfigType
	if configType == "" {
		configType = "yaml"
	}
	v.SetConfigType(configType)

	// 设置配置文件
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
	} else {
		// 添加配置文件搜索路径
		searchPaths := opts.ConfigPaths
		if len(searchPaths) == 0 {
			searchPaths = []string{
				".",
				"./config",
				"/etc/" + opts.ServiceName,
			}
		}

		// 添加环境特定的配置目录到搜索路径
		// 例如: ./config/production, ./config/staging, ./config/local
		envSearchPaths := make([]string, 0, len(searchPaths)*2)
		for _, path := range searchPaths {
			// 先添加环境特定目录
			envSearchPaths = append(envSearchPaths, fmt.Sprintf("%s/%s", path, opts.Environment))
			// 再添加基础目录
			envSearchPaths = append(envSearchPaths, path)
		}

		// 设置配置文件名
		configName := opts.ServiceName
		if configName == "" {
			configName = "config"
		}
		v.SetConfigName(configName)

		// 添加所有搜索路径
		for _, path := range envSearchPaths {
			v.AddConfigPath(path)
		}
	}

	// 设置环境变量
	v.AutomaticEnv()
	if opts.EnvPrefix != "" {
		v.SetEnvPrefix(opts.EnvPrefix)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	return &Loader{
		v:        v,
		validate: validator.New(),
		options:  opts,
	}
}

// Load 加载配置（简化版本）
func Load(opts Options) (*Loader, error) {
	loader := NewLoader(opts)

	// 尝试读取配置文件（如果存在）
	if err := loader.v.ReadInConfig(); err != nil {
		// 配置文件不存在不是错误，可以只使用环境变量
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return loader, nil
}

// LoadInto 加载配置到指定的结构体
func (l *Loader) LoadInto(cfg interface{}) error {
	// 尝试读取配置文件
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置到结构体
	if err := l.v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := l.validate.Struct(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// Get 获取配置值
func (l *Loader) Get(key string) interface{} {
	return l.v.Get(key)
}

// GetString 获取字符串配置
func (l *Loader) GetString(key string) string {
	return l.v.GetString(key)
}

// GetInt 获取整数配置
func (l *Loader) GetInt(key string) int {
	return l.v.GetInt(key)
}

// GetBool 获取布尔配置
func (l *Loader) GetBool(key string) bool {
	return l.v.GetBool(key)
}

// GetDuration 获取时间间隔配置
func (l *Loader) GetDuration(key string) time.Duration {
	return l.v.GetDuration(key)
}

// GetStringSlice 获取字符串切片配置
func (l *Loader) GetStringSlice(key string) []string {
	return l.v.GetStringSlice(key)
}

// GetStringMap 获取字符串映射配置
func (l *Loader) GetStringMap(key string) map[string]interface{} {
	return l.v.GetStringMap(key)
}

// Set 设置配置值（用于测试）
func (l *Loader) Set(key string, value interface{}) {
	l.v.Set(key, value)
}

// SetDefault 设置默认值
func (l *Loader) SetDefault(key string, value interface{}) {
	l.v.SetDefault(key, value)
}

// IsSet 检查配置是否已设置
func (l *Loader) IsSet(key string) bool {
	return l.v.IsSet(key)
}

// AllSettings 获取所有配置
func (l *Loader) AllSettings() map[string]interface{} {
	return l.v.AllSettings()
}

// Viper 获取底层的 Viper 实例（高级用法）
func (l *Loader) Viper() *viper.Viper {
	return l.v
}
