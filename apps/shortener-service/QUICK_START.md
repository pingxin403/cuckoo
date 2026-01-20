# URL Shortener Service - Quick Start Guide

## 🚀 Quick Start (5 Minutes)

### Prerequisites
- Docker and Docker Compose installed
- Go 1.21+ installed (for local development)

### Option 1: Run with Docker Compose (Recommended)

```bash
# From repository root
docker compose up -d mysql redis shortener-service

# Check service status
docker compose ps

# View logs
docker compose logs -f shortener-service

# Test the service
curl http://localhost:8081/health

# Stop services
docker compose down
```

### Option 2: Run Integration Tests

```bash
cd apps/shortener-service

# Run automated integration tests
./scripts/run-integration-tests.sh
```

This will:
1. Build the service
2. Start MySQL and Redis
3. Run all integration tests
4. Clean up automatically

## 📝 Quick API Examples

### Create a Short Link (gRPC)

```bash
# Using grpcurl
grpcurl -plaintext \
  -d '{"long_url": "https://example.com/very/long/path"}' \
  localhost:9092 \
  api.v1.ShortenerService/CreateShortLink
```

Response:
```json
{
  "shortUrl": "http://localhost:8081/abc123x",
  "shortCode": "abc123x",
  "createdAt": "2026-01-20T10:00:00Z"
}
```

### Create with Custom Code

```bash
grpcurl -plaintext \
  -d '{"long_url": "https://example.com", "custom_code": "promo2024"}' \
  localhost:9092 \
  api.v1.ShortenerService/CreateShortLink
```

### Get Link Info

```bash
grpcurl -plaintext \
  -d '{"short_code": "abc123x"}' \
  localhost:9092 \
  api.v1.ShortenerService/GetLinkInfo
```

### Test Redirect (HTTP)

```bash
# This will return a 302 redirect
curl -I http://localhost:8081/abc123x

# Follow the redirect
curl -L http://localhost:8081/abc123x
```

### Delete a Link

```bash
grpcurl -plaintext \
  -d '{"short_code": "abc123x"}' \
  localhost:9092 \
  api.v1.ShortenerService/DeleteShortLink
```

## 🔍 Monitoring

### Health Checks

```bash
# Liveness probe
curl http://localhost:8081/health

# Readiness probe
curl http://localhost:8081/ready
```

### Metrics

```bash
# Prometheus metrics
curl http://localhost:9091/metrics
```

Key metrics:
- `shortener_requests_total` - Total requests
- `shortener_request_duration_seconds` - Request latency
- `shortener_cache_hits_total` - Cache hits (L1/L2)
- `shortener_cache_misses_total` - Cache misses
- `shortener_errors_total` - Errors by type

## 🧪 Testing

### Run Unit Tests

```bash
cd apps/shortener-service
make test APP=shortener
```

### Run Integration Tests

```bash
cd apps/shortener-service
./scripts/run-integration-tests.sh
```

### Check Test Coverage

```bash
cd apps/shortener-service
./scripts/test-coverage.sh
```

## 🛠️ Development

### Build Locally

```bash
cd apps/shortener-service
go build -o shortener-service .
```

### Run Locally (requires MySQL and Redis)

```bash
# Set environment variables
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
export MYSQL_DATABASE=shortener
export MYSQL_USER=root
export MYSQL_PASSWORD=password
export REDIS_ADDR=localhost:6379
export BASE_URL=http://localhost:8080

# Run migrations
./scripts/run-migrations.sh

# Start service
./shortener-service
```

### Run with Monorepo Commands

```bash
# From repository root
make test APP=shortener
make build APP=shortener
make lint APP=shortener
make docker-build APP=shortener
```

## 📊 Service Ports

| Service | Port | Description |
|---------|------|-------------|
| gRPC API | 9092 | Main gRPC service |
| HTTP Redirect | 8080 (8081 in test) | Short link redirects |
| Metrics | 9090 (9091 in test) | Prometheus metrics |
| MySQL | 3306 (3307 in test) | Database |
| Redis | 6379 (6380 in test) | Cache |

## 🔐 Security Features

- ✅ HTTP/HTTPS protocol enforcement
- ✅ URL length validation (max 2048 chars)
- ✅ Malicious pattern detection
- ✅ Security headers on redirects
- ✅ Input sanitization
- ✅ Audit logging with source IP

## 📚 Documentation

- [Full README](./README.md) - Complete documentation
- [API Documentation](./docs/API.md) - Detailed API reference
- [Integration Test Summary](./INTEGRATION_TEST_SUMMARY.md) - Test results
- [MVP Completion Summary](./MVP_COMPLETION_SUMMARY.md) - Feature overview

## 🐛 Troubleshooting

### Service won't start

```bash
# Check if ports are in use
lsof -i :9092  # gRPC
lsof -i :8081  # HTTP
lsof -i :3306  # MySQL
lsof -i :6379  # Redis

# Check Docker logs
docker compose logs shortener-service
```

### Tests failing

```bash
# Ensure services are healthy
docker compose ps

# Check service logs
docker compose logs shortener-service

# Restart services
docker compose restart shortener-service
```

### Database connection issues

```bash
# Check MySQL is running
docker compose ps mysql

# Test MySQL connection
docker exec -it shortener-mysql mysql -uroot -proot_password -e "SHOW DATABASES;"

# Check migrations
docker exec -it shortener-mysql mysql -ushortener_user -pshortener_password shortener -e "SHOW TABLES;"
```

### Redis connection issues

```bash
# Check Redis is running
docker compose ps redis

# Test Redis connection
docker exec -it shortener-redis redis-cli ping
```

## 🎯 Next Steps

1. **Explore the API**: Try creating and using short links
2. **Check Metrics**: View Prometheus metrics at http://localhost:9091/metrics
3. **Run Tests**: Execute integration tests to see everything working
4. **Read Documentation**: Check out the full [README](./README.md) and [API docs](./docs/API.md)
5. **Deploy**: Use Kubernetes manifests in `k8s/` directory

## 💡 Tips

- Use custom codes for memorable links (e.g., "promo2024")
- Set expiration times for temporary campaigns
- Monitor cache hit rates for performance optimization
- Check logs for audit trail of link creation
- Use health checks for load balancer configuration

---

**Need Help?** Check the [full documentation](./README.md) or [API reference](./docs/API.md).
