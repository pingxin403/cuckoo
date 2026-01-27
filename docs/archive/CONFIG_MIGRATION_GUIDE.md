# Configuration Migration Guide

> **注意**: 本文档提供配置迁移步骤。完整的配置系统文档请参阅 [配置系统完整指南](./CONFIG_SYSTEM_GUIDE.md)

## Quick Start

All services now use the centralized `libs/config` library for configuration management. This guide helps you understand the new configuration system.

## For Developers

### Using Configuration in Your Service

1. **Import the config package:**
```go
import "github.com/pingxin403/cuckoo/apps/your-service/config"
```

2. **Load configuration:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

3. **Access configuration values:**
```go
port := cfg.Server.Port
dbHost := cfg.Database.Host
```

### Configuration Sources (Priority Order)

1. **Environment Variables** (highest priority)
2. **Configuration File** (`config.yaml`)
3. **Default Values** (lowest priority)

### Environment Variable Naming

Environment variables use the format: `SECTION_KEY`

Examples:
- `SERVER_PORT=9095`
- `DATABASE_HOST=localhost`
- `REDIS_ADDR=localhost:6379`
- `OBSERVABILITY_LOG_LEVEL=debug`

### Configuration File

Create a `config.yaml` file in your service directory:

```yaml
server:
  port: 9095
  
database:
  host: localhost
  port: 3306
  
observability:
  log_level: info
```

## For Operations

### Backward Compatibility

**All existing environment variables continue to work!**

Old and new variable names are both supported:

| Old Variable | New Variable | Status |
|-------------|--------------|--------|
| `PORT` | `SERVER_PORT` | Both work |
| `DB_HOST` | `DATABASE_HOST` | Both work |
| `LOG_LEVEL` | `OBSERVABILITY_LOG_LEVEL` | Both work |

### Docker Compose

No changes needed to existing `docker-compose.yml` files:

```yaml
services:
  auth-service:
    environment:
      - PORT=9095              # Still works
      - JWT_SECRET=secret      # Still works
      - LOG_LEVEL=info         # Still works
```

### Kubernetes

No changes needed to existing ConfigMaps or environment variables:

```yaml
env:
  - name: PORT
    value: "9095"
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: auth-secrets
        key: jwt-secret
```

## Service-Specific Configuration

### auth-service

**Required:**
- `JWT_SECRET` - JWT signing secret

**Optional:**
- `SERVER_PORT` (default: 9095)
- `OBSERVABILITY_LOG_LEVEL` (default: info)

### user-service

**Required:**
- Database credentials (or `MYSQL_DSN`)

**Optional:**
- `SERVER_PORT` (default: 9096)
- `DATABASE_HOST` (default: localhost)
- `DATABASE_PORT` (default: 3306)

### im-service

**Required:**
- Database credentials
- Redis connection

**Optional:**
- `SERVER_GRPC_PORT` (default: 9094)
- `SERVER_HTTP_PORT` (default: 8080)
- `OFFLINE_WORKER_ENABLED` (default: true)
- `KAFKA_BROKERS` (default: localhost:9092)

### im-gateway-service

**Required:**
- Redis connection

**Optional:**
- `SERVER_HTTP_PORT` (default: 8080)
- `SERVER_GRPC_PORT` (default: 9097)
- `ETCD_ENDPOINTS` (default: localhost:2379)

## Example Configurations

### Development (Environment Variables)

```bash
export SERVER_PORT=9095
export DATABASE_HOST=localhost
export DATABASE_PORT=3306
export REDIS_ADDR=localhost:6379
export OBSERVABILITY_LOG_LEVEL=debug
```

### Development (Config File)

Create `config.yaml`:

```yaml
server:
  port: 9095

database:
  host: localhost
  port: 3306
  user: dev_user
  password: dev_password
  database: dev_db

redis:
  addr: localhost:6379

observability:
  log_level: debug
  environment: development
```

### Production (Environment Variables)

```bash
export SERVER_PORT=9095
export DATABASE_HOST=prod-db.example.com
export DATABASE_PORT=3306
export DATABASE_USER=prod_user
export DATABASE_PASSWORD=${DB_PASSWORD}
export REDIS_ADDR=prod-redis.example.com:6379
export OBSERVABILITY_LOG_LEVEL=info
export OBSERVABILITY_ENVIRONMENT=production
```

## Validation

Configuration is validated at startup. Invalid configurations will cause the service to exit with an error message:

```
Failed to load configuration: config validation failed: 
  - Server.Port: must be between 1 and 65535
  - Database.Host: required field is missing
```

## Troubleshooting

### Service won't start

1. Check required environment variables are set
2. Verify configuration file syntax (if using YAML)
3. Check validation errors in logs

### Configuration not taking effect

1. Environment variables override config files
2. Check variable naming (use `SECTION_KEY` format)
3. Restart service after changing configuration

### Finding configuration options

1. Check `config.example.yaml` in service directory
2. Look at `config/config.go` for structure
3. See service-specific documentation

## Migration Checklist

- [x] All services migrated to `libs/config`
- [x] Example configuration files created
- [x] Backward compatibility maintained
- [x] Compilation tests passed
- [ ] Integration tests with new config
- [ ] Documentation updated
- [ ] Deployment guides updated

## Additional Resources

- [libs/config README](../libs/config/README.md)
- [Configuration Library Migration Summary](../CONFIG_LIBRARY_MIGRATION_SUMMARY.md)
- Service-specific `config.example.yaml` files
- Service-specific `DEPLOYMENT.md` files
