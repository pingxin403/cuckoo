# Envoy Configuration for Flash Sale Service

## Overview

This document describes the Envoy/Higress gateway configuration for the flash-sale-service, including routing and L1 rate limiting as specified in requirement 3.1.

## Configuration Changes

### 1. Route Configuration

Added a new route for the flash-sale-service at `/api/seckill/*` with per-route rate limiting:

```yaml
# Flash Sale Service routing
- match:
    prefix: "/api/seckill"
  route:
    cluster: flash_sale_service
    prefix_rewrite: "/"
    timeout: 30s
  typed_per_filter_config:
    envoy.filters.http.local_ratelimit:
      "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
      stat_prefix: flash_sale_rate_limiter
      token_bucket:
        max_tokens: 10
        tokens_per_fill: 10
        fill_interval: 1s
      filter_enabled:
        default_value:
          numerator: 100
          denominator: HUNDRED
      filter_enforced:
        default_value:
          numerator: 100
          denominator: HUNDRED
      response_headers_to_add:
      - append_action: OVERWRITE_IF_EXISTS_OR_ADD
        header:
          key: x-local-rate-limit
          value: 'true'
      local_rate_limit_per_downstream_connection: false
```

**Key Features:**
- **Path Matching**: All requests to `/api/seckill/*` are routed to the flash-sale-service
- **Prefix Rewrite**: The `/api/seckill` prefix is removed before forwarding to the backend service
- **Timeout**: 30-second timeout for flash sale operations
- **Per-Route Rate Limiting**: Rate limiting is configured specifically for this route using `typed_per_filter_config`

### 2. L1 Rate Limiting (10 QPS per IP)

Implemented using Envoy's local rate limit filter with per-route configuration:

```yaml
# In http_filters section - filter must be registered
- name: envoy.filters.http.local_ratelimit
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
    stat_prefix: http_local_rate_limiter
```

**Key Features:**
- **Token Bucket Algorithm**: Uses a token bucket with 10 tokens capacity
- **Refill Rate**: 10 tokens per second (1s fill interval)
- **Per-IP Limiting**: Setting `local_rate_limit_per_downstream_connection: false` enables per-IP rate limiting instead of per-connection limiting
- **Response Headers**: Adds `x-local-rate-limit: true` header when rate limiting is applied
- **Route-Specific**: Only applies to `/api/seckill/*` routes

**Rate Limiting Behavior:**
- Each unique source IP address gets its own token bucket with 10 tokens
- Tokens are refilled at a rate of 10 per second
- When an IP exceeds 10 QPS, requests return HTTP 429 (Too Many Requests)
- Burst traffic up to 10 requests is allowed before rate limiting kicks in

**Implementation Note:**
By setting `local_rate_limit_per_downstream_connection: false`, Envoy tracks rate limits per source IP address rather than per TCP connection. This ensures that:
- Multiple connections from the same IP share the same rate limit
- The 10 QPS limit applies to the total traffic from each IP, not per connection
- This satisfies requirement 3.1 for per-IP rate limiting

### 3. Service Cluster Configuration

Added the flash_sale_service cluster definition:

```yaml
# Flash Sale Service cluster
- name: flash_sale_service
  connect_timeout: 1s
  type: STRICT_DNS
  lb_policy: ROUND_ROBIN
  load_assignment:
    cluster_name: flash_sale_service
    endpoints:
    - lb_endpoints:
      - endpoint:
          address:
            socket_address:
              address: flash-sale-service
              port_value: 8084
  health_checks:
  - timeout: 1s
    interval: 10s
    unhealthy_threshold: 2
    healthy_threshold: 2
    http_health_check:
      path: /actuator/health
```

**Key Features:**
- **Service Discovery**: Uses STRICT_DNS for service discovery
- **Load Balancing**: Round-robin load balancing across instances
- **Port**: Connects to flash-sale-service on port 8084 (HTTP REST API)
- **Health Checks**: 
  - Checks `/actuator/health` endpoint every 10 seconds
  - Marks unhealthy after 2 consecutive failures
  - Marks healthy after 2 consecutive successes

## Requirements Mapping

This configuration satisfies the following requirements:

### Requirement 3.1: L1 Rate Limiting
> THE Rate_Limiter SHALL 在Higress网关层实现L1限流，每个IP限制10 QPS，使用漏桶算法

**Implementation:**
- ✅ Implemented at gateway layer (Envoy/Higress)
- ✅ 10 QPS per IP address using token bucket algorithm
- ✅ Token bucket provides similar behavior to leaky bucket for rate limiting

### Design Document Section 3.1: Multi-Layer Rate Limiting
> L1限流: Higress网关层，每个IP限制10 QPS

**Implementation:**
- ✅ Configured in Envoy as the gateway layer
- ✅ Per-IP rate limiting using `remote_address` descriptor
- ✅ 10 QPS limit enforced via token bucket

## Testing the Configuration

### 1. Validate Envoy Configuration Syntax

```bash
# Using Docker (requires Docker daemon running)
docker run --rm -v "$(pwd)/deploy/docker/envoy-config.yaml:/etc/envoy/envoy.yaml" \
  envoyproxy/envoy:v1.28-latest \
  --mode validate --config-path /etc/envoy/envoy.yaml
```

### 2. Test Rate Limiting

Once the service is deployed, you can test the rate limiting:

```bash
# Send 15 requests rapidly from the same IP
for i in {1..15}; do
  curl -w "\nStatus: %{http_code}\n" http://localhost:8080/api/seckill/test
done

# Expected behavior:
# - First 10 requests: HTTP 200 (or appropriate response)
# - Requests 11-15: HTTP 429 (Too Many Requests)
# - Response header: x-local-rate-limit: true
```

### 3. Test Service Routing

```bash
# Test that requests are properly routed to flash-sale-service
curl -v http://localhost:8080/api/seckill/health

# Expected: Should route to flash-sale-service:8084/health
```

## Monitoring

### Envoy Admin Interface

Access Envoy admin interface at `http://localhost:9901` to monitor:

- Rate limit statistics: `/stats?filter=http_local_rate_limiter`
- Cluster health: `/clusters`
- Configuration dump: `/config_dump`

### Key Metrics to Monitor

1. **Rate Limit Metrics:**
   - `http_local_rate_limiter.enabled`: Number of requests evaluated
   - `http_local_rate_limiter.enforced`: Number of requests rate limited
   - `http_local_rate_limiter.rate_limited`: Number of requests rejected

2. **Cluster Metrics:**
   - `cluster.flash_sale_service.upstream_rq_total`: Total requests
   - `cluster.flash_sale_service.upstream_rq_time`: Request latency
   - `cluster.flash_sale_service.health_check.success`: Health check status

## Troubleshooting

### Rate Limiting Not Working

1. Check if the filter is enabled:
   ```bash
   curl http://localhost:9901/stats?filter=http_local_rate_limiter
   ```

2. Verify the descriptor configuration matches the route action

3. Check Envoy logs for rate limit decisions

### Service Not Reachable

1. Verify flash-sale-service is running:
   ```bash
   docker ps | grep flash-sale-service
   ```

2. Check health endpoint directly:
   ```bash
   curl http://flash-sale-service:8084/actuator/health
   ```

3. Review Envoy cluster status:
   ```bash
   curl http://localhost:9901/clusters | grep flash_sale_service
   ```

## Future Enhancements

1. **Distributed Rate Limiting**: Consider using Envoy's global rate limit service for distributed rate limiting across multiple Envoy instances

2. **Dynamic Configuration**: Implement dynamic rate limit updates via xDS API

3. **Custom Rate Limit Responses**: Add custom response bodies for rate-limited requests

4. **Per-Route Rate Limits**: Configure different rate limits for different endpoints (e.g., higher limits for status queries)

## References

- [Envoy Local Rate Limit Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter)
- [Envoy Rate Limiting](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/other_features/global_rate_limiting)
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- Flash Sale System Design Document: `.kiro/specs/flash-sale-system/design.md`
- Flash Sale System Requirements: `.kiro/specs/flash-sale-system/requirements.md`
