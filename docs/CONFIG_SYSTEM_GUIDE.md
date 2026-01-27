# 配置系统完整指南

## 目录

- [概述](#概述)
- [配置库 (libs/config)](#配置库-libsconfig)
- [多环境配置](#多环境配置)
- [快速开始](#快速开始)
- [配置迁移](#配置迁移)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

## 概述

本项目使用统一的配置管理系统，支持：
- 📁 基于目录的多环境配置
- 🔧 基于 Viper 的配置加载
- ✅ 配置验证和类型安全
- 🔒 环境变量覆盖
- 🎯 Go 和 Java 服务支持

### 实现历史

- **2025-01-26**: 创建统一配置库 (libs/config)
- **2025-01-26**: 迁移 4 个核心服务到配置库
- **2025-01-26**: 添加多环境配置支持
- **2025-01-26**: 完善所有服务的多环境配置

## 配置库 (libs/config)

### 特性

- 🔧 基于 Viper 的配置管理
- 📁 支持基于目录的多环境配置
- 📝 支持环境变量、配置文件、命令行参数
- ✅ 配置验证和默认值
- 🔒 类型安全的配置访问
- 🎯 统一的配置结构

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

### 预定义配置结构

库提供了常用的配置结构：

- `ServerConfig` - 服务器配置（端口、超时等）
- `DatabaseConfig` - 数据库配置（MySQL）
- `RedisConfig` - Redis 配置
- `KafkaConfig` - Kafka 配置
- `EtcdConfig` - Etcd 配置
- `ObservabilityConfig` - 可观测性配置

## 多环境配置

### 支持的环境

| 环境 | 用途 | 日志级别 | 日志格式 |
|------|------|----------|----------|
| `local` | 本地开发 | debug | text |
| `testing` | 测试环境 | info | json |
| `staging` | 预发布环境 | info | json |
| `production` | 生产环境 | info | json |

### 目录结构

**Go 服务:**
```
apps/{service}/config/
├── local/config.yaml
├── testing/config.yaml      # 可选
├── staging/config.yaml      # 可选
└── production/config.yaml
```

**Java 服务:**
```
apps/hello-service/src/main/resources/
├── application.yml
├── application-local.yml
└── application-production.yml
```

### 配置搜索顺序

配置库会按以下顺序搜索配置文件：
1. `./config/{environment}/config.yaml` - 环境特定配置（优先）
2. `./config/config.yaml` - 基础配置（回退）
3. `/etc/{service-name}/config.yaml` - 系统配置

### 配置优先级

配置值的优先级（从高到低）：
1. 命令行参数
2. 环境变量
3. 环境特定配置目录（`config/{environment}/`）
4. 基础配置目录（`config/`）
5. 默认值

## 快速开始

### Go 服务

```bash
# 本地开发（默认使用 local 环境）
./service

# 切换到生产环境
export APP_ENV=production
./service

# 或通过代码指定
config.Load(config.Options{
    ServiceName: "my-service",
    Environment: "production",
})
```

### Java/Spring Boot 服务

```bash
# 本地开发
./gradlew bootRun

# 切换到生产环境
export SPRING_PROFILES_ACTIVE=production
java -jar service.jar

# 或通过命令行参数
java -jar service.jar --spring.profiles.active=production
```

### 环境变量

#### 环境名称

**Go 服务:**
```bash
export APP_ENV=production  # 生产环境
export APP_ENV=staging     # 预发布环境
export APP_ENV=testing     # 测试环境
export APP_ENV=local       # 本地开发环境（默认）
```

**Java 服务:**
```bash
export SPRING_PROFILES_ACTIVE=production
```

#### 配置变量命名规则

环境变量使用大写字母和下划线，例如：

- `SERVER_PORT` → `server.port`
- `DATABASE_HOST` → `database.host`
- `REDIS_ADDR` → `redis.addr`

### 常用环境变量

**认证服务:**
```bash
JWT_SECRET=your-secret-key
```

**数据库服务:**
```bash
DATABASE_HOST=db.example.com
DATABASE_PORT=3306
DATABASE_USER=service_user
DATABASE_PASSWORD=secure_password
DATABASE_DATABASE=service_db
```

**缓存和消息队列:**
```bash
REDIS_ADDR=redis.example.com:6379
KAFKA_BROKERS=kafka1:9092,kafka2:9092
```

**服务发现:**
```bash
ETCD_ENDPOINTS=etcd1:2379,etcd2:2379
SERVICE_DISCOVERY_AUTH_SERVICE_ADDR=auth:9095
SERVICE_DISCOVERY_IM_SERVICE_ADDR=im:9094
SERVICE_DISCOVERY_USER_SERVICE_ADDR=user:9096
```

## 配置迁移

### 已迁移服务

所有服务已迁移到统一配置库：

| 服务 | 环境覆盖 | 状态 |
|------|---------|------|
| auth-service | local, testing, staging, production | ✅ |
| user-service | local, testing, staging, production | ✅ |
| im-service | local, testing, staging, production | ✅ |
| im-gateway-service | local, testing, staging, production | ✅ |
| shortener-service | local, production | ✅ |
| todo-service | local, production | ✅ |
| hello-service | local, production | ✅ |

### 迁移步骤

如果需要迁移新服务到配置库：

1. **添加配置库依赖**
   ```go
   import "github.com/pingxin403/cuckoo/libs/config"
   ```

2. **创建配置目录结构**
   ```bash
   mkdir -p apps/my-service/config/{local,production}
   ```

3. **创建配置文件**
   - `config/local/config.yaml` - 本地开发配置
   - `config/production/config.yaml` - 生产环境配置

4. **更新服务代码**
   ```go
   cfg, err := config.Load(config.Options{
       ServiceName: "my-service",
   })
   ```

5. **测试配置加载**
   ```bash
   APP_ENV=local go run main.go
   APP_ENV=production go run main.go
   ```

## 最佳实践

### ✅ 推荐做法

1. **本地开发** - 使用默认的 `local` 环境
2. **生产部署** - 通过环境变量设置敏感信息
3. **配置验证** - 部署前验证配置文件语法
4. **版本控制** - 配置文件提交到 Git（不含敏感信息）
5. **环境隔离** - 每个环境使用独立的配置文件
6. **文档化** - 在配置文件中添加清晰的注释

### ❌ 避免做法

1. **硬编码密钥** - 生产环境不要在配置文件中硬编码密钥
2. **混用环境** - 不要在生产环境使用开发配置
3. **忽略日志** - 注意查看配置加载日志
4. **跳过验证** - 部署前必须验证配置
5. **重复配置** - 避免在多个地方定义相同的配置

### 配置原则

**本地开发环境 (local):**
- ✅ 可以硬编码配置（仅用于开发）
- ✅ 使用 debug 日志级别
- ✅ 使用 text 日志格式
- ✅ 较小的资源配置

**测试环境 (testing):**
- ✅ 使用测试专用基础设施
- ✅ 使用 info 日志级别
- ✅ 使用 json 日志格式
- ✅ 中等资源配置

**预发布环境 (staging):**
- ⚠️ 敏感信息通过环境变量设置
- ✅ 使用 info 日志级别
- ✅ 使用 json 日志格式
- ✅ 接近生产的资源配置

**生产环境 (production):**
- ⚠️ 所有敏感信息必须通过环境变量设置
- ✅ 使用 info 日志级别
- ✅ 使用 json 日志格式
- ✅ 最大化资源配置

## 故障排查

### 配置未加载

```bash
# 检查环境变量
echo $APP_ENV

# 检查配置文件是否存在
ls -la apps/service/config/$APP_ENV/

# 查看服务日志
./service 2>&1 | grep -i config
```

### 环境变量未生效

```bash
# 验证环境变量已设置
env | grep -E "APP_ENV|DATABASE|REDIS|KAFKA"

# 检查配置优先级
# 环境变量 > 配置文件
```

### Java Profile 未激活

```bash
# 检查 Spring Profile
echo $SPRING_PROFILES_ACTIVE

# 查看应用日志
./gradlew bootRun | grep -i "active profiles"
```

### 配置文件语法错误

```bash
# 验证 YAML 语法
yamllint apps/service/config/production/config.yaml

# 使用 yq 验证
yq eval '.' apps/service/config/production/config.yaml
```

## 部署示例

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o service

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
COPY --from=builder /app/config ./config

ENV APP_ENV=production
CMD ["./service"]
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service
spec:
  template:
    spec:
      containers:
      - name: service
        image: service:latest
        env:
        - name: APP_ENV
          value: "production"
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: password
```

### Docker Compose 部署

```yaml
version: '3.8'
services:
  auth-service:
    image: auth-service:latest
    environment:
      - APP_ENV=production
      - JWT_SECRET=${JWT_SECRET}
    volumes:
      - ./apps/auth-service/config:/app/config:ro
```

## 相关文档

- [配置库 README](../libs/config/README.md) - 配置库详细文档
- [多环境配置快速参考](./MULTI_ENV_CONFIG_QUICK_REFERENCE.md) - 快速参考指南
- [部署指南](./deployment/DEPLOYMENT_GUIDE.md) - 部署相关文档
- [开发指南](./development/TESTING_GUIDE.md) - 开发和测试指南

## 附录

### 环境资源配置对比

| 环境 | 数据库连接 | Redis 连接池 | 日志级别 | 日志格式 |
|------|-----------|-------------|---------|---------|
| local | 10 | 10 | debug | text |
| testing | 15 | 20 | info | json |
| staging | 20 | 30 | info | json |
| production | 25-50 | 30-50 | info | json |

### 服务特定配置

**im-gateway-service 最大连接数:**
- local: 1,000
- testing: 5,000
- staging: 50,000
- production: 100,000

**shortener-service 限流配置:**
- local: 100 请求/分钟
- production: 1,000 请求/分钟

**im-service 批量处理大小:**
- local/testing: 50
- staging/production: 100

## 更新日志

- **2025-01-26**: 创建统一配置系统指南
- **2025-01-26**: 合并多个配置文档
- **2025-01-26**: 添加完整的故障排查指南
