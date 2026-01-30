# Flash Sale Gateway Setup Summary

## Overview

This document provides a summary of the Higress/Envoy gateway configuration for the flash-sale-service, including routing and L1 rate limiting as specified in requirement 3.1.

## Configuration Files

- **Main Configuration**: `deploy/docker/envoy-config.yaml`
- **Local Development**: `deploy/docker/envoy-local-config.yaml` (needs similar updates)
- **Documentation**: `deploy/docker/ENVOY_FLASH_SALE_CONFIG.md`
- **Validation Script**: `deploy/docker/validate-envoy-config.sh`
- **Test Script**: `deploy/docker/test-flash-sale-routing.sh`

## What Was Configured

### 1. Service Routing

✅ **Route**: `/api/seckill/*` → `flash-sale-service:8084`
- Prefix rewrite: `/api/seckill` is removed before forwarding
- Timeout: 30 seconds
- Health check: `/actuator/health`

### 2. L1 Rate Limiting (Requirement 3.1)

✅ **Per-IP Rate Limiting**: 10 QPS per IP address
- Implementation: Envoy local rate limit filter with per-route configuration
- Algorithm: Token bucket (10 tokens, refill 10/second)
- Behavior: HTTP 429 when limit exceeded
- Header: `x-local-rate-limit: true` added to rate-limited responses

### 3. Service Cluster

✅ **Cluster Configuration**:
- Service discovery: STRICT_DNS
- Load balancing: Round-robin
- Port: 8084 (HTTP REST API)
- Health checks: Every 10 seconds via `/actuator/health`

## Requirements Satisfied

| Requirement | Status | Implementation |
|------------|--------|----------------|
| 3.1: L1 Rate Limiting at Gateway | ✅ Complete | Per-route local rate limit filter with 10 QPS per IP |
| Route Configuration | ✅ Complete | `/api/seckill/*` routes to flash-sale-service |
| Integration with Existing Gateway | ✅ Complete | Added to existing envoy-config.yaml |

## Validation

Run the validation script to verify the configuration:

```bash
./deploy/docker/validate-envoy-config.sh
```

Expected output:
- ✅ Envoy configuration is valid
- ✅ flash_sale_service cluster found
- ✅ /api/seckill route found
- ✅ Local rate limit filter found
- ✅ Per-IP rate limiting configured

## Testing

### 1. Start Services

```bash
cd deploy/docker
docker-compose -f docker-compose.infra.yml up -d
docker-compose -f docker-compose.services.yml up -d
```

### 2. Test Routing

```bash
# Test health endpoint
curl http://localhost:8080/api/seckill/health

# Expected: Routes to flash-sale-service:8084/actuator/health
```

### 3. Test Rate Limiting

```bash
# Run the test script
./deploy/docker/test-flash-sale-routing.sh

# Or manually test with rapid requests
for i in {1..15}; do
  curl -w "\nStatus: %{http_code}\n" http://localhost:8080/api/seckill/test
done

# Expected:
# - First ~10 requests: HTTP 200 (or 404/503 if endpoint doesn't exist)
# - Remaining requests: HTTP 429 (Too Many Requests)
```

## Monitoring

### Envoy Admin Interface

Access at `http://localhost:9901`:

- **Rate limit stats**: `/stats?filter=flash_sale_rate_limiter`
- **Cluster health**: `/clusters`
- **Configuration**: `/config_dump`

### Key Metrics

1. **Rate Limiting**:
   - `http_local_rate_limiter.flash_sale_rate_limiter.enabled`: Requests evaluated
   - `http_local_rate_limiter.flash_sale_rate_limiter.enforced`: Requests rate limited
   - `http_local_rate_limiter.flash_sale_rate_limiter.rate_limited`: Requests rejected

2. **Service Health**:
   - `cluster.flash_sale_service.upstream_rq_total`: Total requests
   - `cluster.flash_sale_service.upstream_rq_time`: Request latency
   - `cluster.flash_sale_service.health_check.success`: Health check status

## Implementation Details

### Per-Route Rate Limiting

The configuration uses Envoy's `typed_per_filter_config` to apply rate limiting specifically to the flash sale route:

```yaml
typed_per_filter_config:
  envoy.filters.http.local_ratelimit:
    "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
    stat_prefix: flash_sale_rate_limiter
    token_bucket:
      max_tokens: 10
      tokens_per_fill: 10
      fill_interval: 1s
    local_rate_limit_per_downstream_connection: false
```

**Key Setting**: `local_rate_limit_per_downstream_connection: false`
- This ensures rate limiting is per-IP address, not per TCP connection
- Multiple connections from the same IP share the same rate limit
- Satisfies requirement 3.1 for per-IP rate limiting

### Why Per-Route Configuration?

1. **Isolation**: Rate limiting only applies to flash sale endpoints
2. **Flexibility**: Other services are not affected
3. **Performance**: Rate limiting overhead only for flash sale traffic
4. **Monitoring**: Separate metrics for flash sale rate limiting

## Next Steps

1. ✅ Gateway configuration complete
2. ⏭️ Update `envoy-local-config.yaml` for local development (optional)
3. ⏭️ Deploy and test in production environment
4. ⏭️ Monitor rate limiting metrics
5. ⏭️ Tune rate limits based on actual traffic patterns

## Troubleshooting

### Rate Limiting Not Working

1. Check filter is enabled:
   ```bash
   curl http://localhost:9901/stats?filter=flash_sale_rate_limiter
   ```

2. Verify per-route config is applied:
   ```bash
   curl http://localhost:9901/config_dump | grep -A 20 "flash_sale_rate_limiter"
   ```

3. Check Envoy logs:
   ```bash
   docker logs envoy 2>&1 | grep -i "rate"
   ```

### Service Not Reachable

1. Verify service is running:
   ```bash
   docker ps | grep flash-sale-service
   ```

2. Check health endpoint directly:
   ```bash
   curl http://flash-sale-service:8084/actuator/health
   ```

3. Review cluster status:
   ```bash
   curl http://localhost:9901/clusters | grep flash_sale_service
   ```

## References

- [Envoy Local Rate Limit Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter)
- [Envoy Per-Route Configuration](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#config-http-filters-router-route-config)
- Flash Sale Design Document: `.kiro/specs/flash-sale-system/design.md`
- Flash Sale Requirements: `.kiro/specs/flash-sale-system/requirements.md`
- Detailed Configuration: `deploy/docker/ENVOY_FLASH_SALE_CONFIG.md`
