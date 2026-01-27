package config

import (
	"os"
	"testing"
	"time"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	if loader == nil {
		t.Fatal("Expected loader to be created")
	}

	if loader.v == nil {
		t.Fatal("Expected viper instance to be created")
	}

	if loader.validate == nil {
		t.Fatal("Expected validator to be created")
	}
}

func TestLoad(t *testing.T) {
	loader, err := Load(Options{
		ServiceName: "test-service",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if loader == nil {
		t.Fatal("Expected loader to be created")
	}
}

func TestLoaderGetters(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	// 设置测试值
	loader.Set("test.string", "hello")
	loader.Set("test.int", 42)
	loader.Set("test.bool", true)
	loader.Set("test.duration", "5s")
	loader.Set("test.slice", []string{"a", "b", "c"})

	// 测试 GetString
	if got := loader.GetString("test.string"); got != "hello" {
		t.Errorf("GetString() = %v, want %v", got, "hello")
	}

	// 测试 GetInt
	if got := loader.GetInt("test.int"); got != 42 {
		t.Errorf("GetInt() = %v, want %v", got, 42)
	}

	// 测试 GetBool
	if got := loader.GetBool("test.bool"); got != true {
		t.Errorf("GetBool() = %v, want %v", got, true)
	}

	// 测试 GetDuration
	if got := loader.GetDuration("test.duration"); got != 5*time.Second {
		t.Errorf("GetDuration() = %v, want %v", got, 5*time.Second)
	}

	// 测试 GetStringSlice
	got := loader.GetStringSlice("test.slice")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Errorf("GetStringSlice() length = %v, want %v", len(got), len(want))
	}
}

func TestLoaderEnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("TEST_PORT", "8080")
	os.Setenv("TEST_HOST", "localhost")
	defer func() {
		os.Unsetenv("TEST_PORT")
		os.Unsetenv("TEST_HOST")
	}()

	loader := NewLoader(Options{
		ServiceName: "test-service",
		EnvPrefix:   "TEST",
	})

	// 测试环境变量读取
	if got := loader.GetInt("port"); got != 8080 {
		t.Errorf("GetInt(port) = %v, want %v", got, 8080)
	}

	if got := loader.GetString("host"); got != "localhost" {
		t.Errorf("GetString(host) = %v, want %v", got, "localhost")
	}
}

func TestLoaderSetDefault(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	loader.SetDefault("default.value", "test")

	if got := loader.GetString("default.value"); got != "test" {
		t.Errorf("GetString(default.value) = %v, want %v", got, "test")
	}
}

func TestLoaderIsSet(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	loader.Set("test.key", "value")

	if !loader.IsSet("test.key") {
		t.Error("IsSet(test.key) = false, want true")
	}

	if loader.IsSet("nonexistent.key") {
		t.Error("IsSet(nonexistent.key) = true, want false")
	}
}

func TestLoadInto(t *testing.T) {
	type TestConfig struct {
		Server struct {
			Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
			Host string `mapstructure:"host"`
		} `mapstructure:"server"`
	}

	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	// 设置测试配置
	loader.Set("server.port", 8080)
	loader.Set("server.host", "localhost")

	var cfg TestConfig
	if err := loader.LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("cfg.Server.Port = %v, want %v", cfg.Server.Port, 8080)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("cfg.Server.Host = %v, want %v", cfg.Server.Host, "localhost")
	}
}

func TestLoadIntoValidation(t *testing.T) {
	type TestConfig struct {
		Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
	}

	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	// 设置无效的端口
	loader.Set("port", 99999)

	var cfg TestConfig
	err := loader.LoadInto(&cfg)
	if err == nil {
		t.Error("LoadInto() expected validation error, got nil")
	}
}

func TestSetCommonDefaults(t *testing.T) {
	loader := NewLoader(Options{
		ServiceName: "test-service",
	})

	SetCommonDefaults(loader)

	// 测试服务器默认值
	if got := loader.GetInt("server.port"); got != 8080 {
		t.Errorf("server.port = %v, want %v", got, 8080)
	}

	// 测试数据库默认值
	if got := loader.GetInt("database.port"); got != 3306 {
		t.Errorf("database.port = %v, want %v", got, 3306)
	}

	// 测试 Redis 默认值
	if got := loader.GetString("redis.addr"); got != "localhost:6379" {
		t.Errorf("redis.addr = %v, want %v", got, "localhost:6379")
	}
}
