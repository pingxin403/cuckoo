# 多环境配置快速参考

> **完整文档**: 详细信息请参阅 [配置系统完整指南](./CONFIG_SYSTEM_GUIDE.md)

## 快速开始

### Go 服务

```bash
# 本地开发（默认）
./service

# 切换环境
export APP_ENV=production  # 或 testing, staging
./service
```

### Java 服务

```bash
# 本地开发
./gradlew bootRun

# 切换环境
export SPRING_PROFILES_ACTIVE=production
java -jar service.jar
```

## 环境列表

| 环境 | 用途 | 日志级别 |
|------|------|----------|
| `local` | 本地开发（默认） | debug |
| `testing` | 测试环境 | info |
| `staging` | 预发布环境 | info |
| `production` | 生产环境 | info |

## 常用环境变量

```bash
# 环境选择
APP_ENV=production                    # Go 服务
SPRING_PROFILES_ACTIVE=production     # Java 服务

# 服务配置
SERVER_PORT=9095
OBSERVABILITY_LOG_LEVEL=info

# 数据库
DATABASE_HOST=db.example.com
DATABASE_PORT=3306
DATABASE_USER=service_user
DATABASE_PASSWORD=secure_password

# 缓存和消息队列
REDIS_ADDR=redis:6379
KAFKA_BROKERS=kafka1:9092,kafka2:9092

# 服务发现
ETCD_ENDPOINTS=etcd1:2379,etcd2:2379
```

## 配置优先级

1. 命令行参数（最高）
2. 环境变量
3. 环境特定配置（`config/{environment}/`）
4. 基础配置（`config/`）
5. 默认值（最低）

## 故障排查

```bash
# 检查环境
echo $APP_ENV
ls -la apps/service/config/$APP_ENV/

# 验证环境变量
env | grep -E "APP_ENV|DATABASE|REDIS"

# 查看日志
./service 2>&1 | grep -i config
```

## 相关文档

- [配置系统完整指南](./CONFIG_SYSTEM_GUIDE.md) - 完整文档
- [配置库 README](../libs/config/README.md) - API 文档
