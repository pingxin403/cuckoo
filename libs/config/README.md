# Config Library

统一的配置管理库，基于 Viper 封装，提供类型安全的配置加载和验证功能。

## 特性

- 🔧 基于 Viper 的配置管理
- 📁 支持基于目录的多环境配置
- 📝 支持环境变量、配置文件、命令行参数
- ✅ 配置验证和默认值
- 🔒 类型安全的配置访问
- 🎯 统一的配置结构

## 使用方法

### 基本用法

```go
package main

import (
    "log"
    "github.com/pingxin403/cuckoo/libs/config"
)

func main() {
    // 加载配置
    cfg, err := config.Load(config.Options{
        ServiceName: "my-service",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 访问配置
    port := cfg.GetInt("server.port")
    dbHost := cfg.GetString("database.host")
}
```

### 多环境配置

配置库支持基于目录的多环境配置管理：

**目录结构：**
```
apps/my-service/
├── config/
│   ├── local/          # 本地开发环境
│   │   └── config.yaml
│   ├── testing/        # 测试环境
│   │   └── config.yaml
│   ├── staging/        # 预发布环境
│   │   └── config.yaml
│   └── production/     # 生产环境
│       └── config.yaml
└── main.go
```

**使用方式：**

```go
package main

import (
    "log"
    "github.com/pingxin403/cuckoo/libs/config"
)

func main() {
    // 方式 1: 通过 Options 指定环境
    cfg, err := config.Load(config.Options{
        ServiceName: "my-service",
        Environment: "production", // 将从 config/production/ 目录加载
    })
    
    // 方式 2: 通过环境变量指定（推荐）
    // 设置环境变量: export APP_ENV=production
    cfg, err := config.Load(config.Options{
        ServiceName: "my-service",
        // Environment 未指定时，自动从 APP_ENV 环境变量读取
        // 默认值为 "local"
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

**配置搜索顺序：**

配置库会按以下顺序搜索配置文件：
1. `./config/{environment}/config.yaml` - 环境特定配置（优先）
2. `./config/config.yaml` - 基础配置（回退）
3. `/etc/{service-name}/config.yaml` - 系统配置

**环境名称：**
- `local` - 本地开发环境（默认）
- `testing` - 测试环境
- `staging` - 预发布环境
- `production` - 生产环境

### 使用预定义的配置结构

```go
package main

import (
    "log"
    "github.com/pingxin403/cuckoo/libs/config"
)

type MyServiceConfig struct {
    Server   config.ServerConfig
    Database config.DatabaseConfig
    Redis    config.RedisConfig
}

func main() {
    var cfg MyServiceConfig
    
    loader := config.NewLoader(config.Options{
        ServiceName: "my-service",
        Environment: "production",
    })
    
    if err := loader.LoadInto(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // 使用类型安全的配置
    log.Printf("Server port: %d", cfg.Server.Port)
    log.Printf("Database host: %s", cfg.Database.Host)
}
```

## 配置优先级

配置值的优先级（从高到低）：

1. 命令行参数
2. 环境变量
3. 环境特定配置目录（如 `config/production/`）
4. 基础配置目录（`config/`）
5. 默认值

## 环境变量

### 环境名称

通过 `APP_ENV` 环境变量指定当前环境：

```bash
export APP_ENV=production  # 生产环境
export APP_ENV=staging     # 预发布环境
export APP_ENV=testing     # 测试环境
export APP_ENV=local       # 本地开发环境（默认）
```

### 配置变量命名规则

环境变量使用大写字母和下划线，例如：

- `SERVER_PORT` -> `server.port`
- `DATABASE_HOST` -> `database.host`
- `REDIS_ADDR` -> `redis.addr`

## 预定义配置结构

库提供了常用的配置结构：

- `ServerConfig` - 服务器配置（端口、超时等）
- `DatabaseConfig` - 数据库配置（MySQL）
- `RedisConfig` - Redis 配置
- `KafkaConfig` - Kafka 配置
- `EtcdConfig` - Etcd 配置
- `ObservabilityConfig` - 可观测性配置

## 配置验证

配置加载时会自动验证必填字段和值的有效性：

```go
type Config struct {
    Port int `validate:"required,min=1,max=65535"`
    Host string `validate:"required,hostname"`
}
```

## 配置示例

### 目录结构示例

```
apps/auth-service/
├── config/
│   ├── local/
│   │   └── config.yaml      # 本地开发配置
│   ├── testing/
│   │   └── config.yaml      # 测试环境配置
│   ├── staging/
│   │   └── config.yaml      # 预发布环境配置
│   ├── production/
│   │   └── config.yaml      # 生产环境配置
│   └── config.yaml          # 基础配置（可选）
└── main.go
```

### 配置文件示例

**config/local/config.yaml:**
```yaml
server:
  port: 9095
  
observability:
  log_level: debug
  log_format: text
  
jwt:
  secret: "local-dev-secret"
  expiration: 24h
```

**config/production/config.yaml:**
```yaml
server:
  port: 9095
  
observability:
  log_level: info
  log_format: json
  
jwt:
  # 生产环境通过环境变量设置
  # secret: ""
  expiration: 1h
```
