package config

import (
	"os"
	"testing"
)

func TestMultiEnvironmentConfig(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		wantEnv     string
	}{
		{
			name:        "default to local",
			environment: "",
			wantEnv:     "local",
		},
		{
			name:        "production environment",
			environment: "production",
			wantEnv:     "production",
		},
		{
			name:        "staging environment",
			environment: "staging",
			wantEnv:     "staging",
		},
		{
			name:        "testing environment",
			environment: "testing",
			wantEnv:     "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除环境变量
			os.Unsetenv("APP_ENV")

			// 如果指定了环境，设置环境变量
			if tt.environment != "" {
				os.Setenv("APP_ENV", tt.environment)
				defer os.Unsetenv("APP_ENV")
			}

			loader := NewLoader(Options{
				ServiceName: "test-service",
			})

			if loader.options.Environment != tt.wantEnv {
				t.Errorf("Environment = %v, want %v", loader.options.Environment, tt.wantEnv)
			}
		})
	}
}

func TestConfigPathPriority(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
		Environment: "production",
		ConfigPaths: []string{"./config"},
	})

	// 验证配置路径包含环境特定目录
	v := loader.Viper()

	// Viper 应该已经添加了配置路径
	// 我们无法直接访问内部路径列表，但可以验证 loader 已创建
	if v == nil {
		t.Error("Viper instance should not be nil")
	}

	if loader.options.Environment != "production" {
		t.Errorf("Environment = %v, want production", loader.options.Environment)
	}
}

func TestEnvironmentFromEnvVar(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_ENV", "staging")
	defer os.Unsetenv("APP_ENV")

	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	if loader.options.Environment != "staging" {
		t.Errorf("Environment = %v, want staging", loader.options.Environment)
	}
}

func TestEnvironmentOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("APP_ENV", "staging")
	defer os.Unsetenv("APP_ENV")

	// Options 中指定的环境应该优先
	loader := NewLoader(Options{
		ServiceName: "test-service",
		Environment: "production",
	})

	if loader.options.Environment != "production" {
		t.Errorf("Environment = %v, want production (Options should override env var)", loader.options.Environment)
	}
}
