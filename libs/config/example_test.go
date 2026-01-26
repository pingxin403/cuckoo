package config_test

import (
	"fmt"
	"log"

	"github.com/pingxin403/cuckoo/libs/config"
)

// ExampleLoad 演示基本的配置加载
func ExampleLoad() {
	loader, err := config.Load(config.Options{
		ServiceName: "my-service",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 设置默认值
	config.SetCommonDefaults(loader)

	// 访问配置
	port := loader.GetInt("server.port")
	fmt.Printf("Server port: %d\n", port)
	// Output: Server port: 8080
}

// ExampleLoader_LoadInto 演示加载配置到结构体
func ExampleLoader_LoadInto() {
	type MyServiceConfig struct {
		Server   config.ServerConfig   `mapstructure:"server"`
		Database config.DatabaseConfig `mapstructure:"database"`
	}

	loader := config.NewLoader(config.Options{
		ServiceName: "my-service",
	})

	// 设置默认值
	config.SetCommonDefaults(loader)

	// 设置必需的配置
	loader.Set("database.user", "testuser")
	loader.Set("database.database", "testdb")

	var cfg MyServiceConfig
	if err := loader.LoadInto(&cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server port: %d\n", cfg.Server.Port)
	fmt.Printf("Database host: %s\n", cfg.Database.Host)
	// Output:
	// Server port: 8080
	// Database host: localhost
}

// ExampleServerConfig 演示服务器配置
func ExampleServerConfig() {
	loader := config.NewLoader(config.Options{
		ServiceName: "my-service",
	})

	config.SetCommonDefaults(loader)

	var serverCfg config.ServerConfig
	loader.Set("server.port", 9090)
	loader.Set("server.host", "0.0.0.0")

	if err := loader.Viper().UnmarshalKey("server", &serverCfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Port: %d, Host: %s\n", serverCfg.Port, serverCfg.Host)
	// Output: Port: 9090, Host: 0.0.0.0
}
