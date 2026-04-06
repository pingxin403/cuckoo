# Chaos Engineering Tests for URL Shortener Service

## Overview

This document describes chaos engineering tests to verify the URL shortener service's resilience and graceful degradation under various failure scenarios.

## Test Scenarios

### 1. Redis Pod Kill

**Objective**: Verify service continues to function when Redis (L2 cache) becomes unavailable.

**Procedure**:
1. Start shortener service with Redis available
2. Create several short links to warm the cache
3. Stop Redis container/pod
4. Attempt to resolve existing short links
5. Verify:
   - Service returns responses (possibly slower)
   - Error messages are appropriate
   - Service recovers when Redis is restored

**Expected Results**:
- L2 cache misses should fallback to MySQL
- Service should return 302/404 instead of 500
- Recovery should be automatic once Redis is back

### 2. MySQL Connection Loss

**Objective**: Verify service handles database unavailability gracefully.

**Procedure**:
1. Start shortener service with MySQL available
2. Create a short link
3. Stop MySQL container/pod
4. Attempt operations:
   - Create new short link
   - Resolve existing short link (from cache)
   - Get link info
5. Verify appropriate error responses

**Expected Results**:
- Redirect with cached data: Should work
- Create new link: Should return 503 Service Unavailable
- Get link info: Should return 503

### 3. Network Partition

**Objective**: Verify service handles network issues between components.

**Procedure**:
1. Start all services (MySQL, Redis, Shortener)
2. Use firewall rules or iptables to block:
   - Redis connectivity
   - MySQL connectivity
3. Test various operations
4. Remove blocks and verify recovery

**Expected Results**:
- Cached operations work
- Non-cached operations fail gracefully
- Recovery is automatic

### 4. High Latency Injection

**Objective**: Verify service handles slow downstream dependencies.

**Procedure**:
1. Start shortener service
2. Use tools like Toxiproxy or network delay to inject latency:
   - 500ms delay to Redis
   - 1000ms delay to MySQL
3. Measure response times
4. Verify timeouts are handled properly

**Expected Results**:
- Configured timeouts should prevent hanging
- Appropriate error messages on timeout

## Running Chaos Tests

### Prerequisites

```bash
# Install chaos testing tools (optional)
# For local testing, use docker to simulate failures
```

### Manual Test Script

```bash
#!/bin/bash

echo "=== Chaos Engineering Tests ==="

# Test 1: Redis Failure
echo "Test 1: Redis Failure"
docker stop shortener-redis
curl -I http://localhost:8080/abc1234
docker start shortener-redis

# Test 2: MySQL Failure
echo "Test 2: MySQL Failure"
docker stop shortener-mysql
curl -X POST http://localhost:8080/api/v1/shortener \
  -H "Content-Type: application/json" \
  -d '{"long_url":"https://example.com"}'
docker start shortener-mysql

echo "=== Tests Complete ==="
```

## Verification Checklist

- [ ] Service continues with degraded performance
- [ ] Proper error responses returned
- [ ] Recovery automatic after fault resolution
- [ ] No data corruption
- [ ] Logging captures failure events

## Implementation Notes

The service already implements graceful degradation:
- Cache manager falls back to MySQL when Redis unavailable
- Structured errors map to appropriate HTTP status codes
- Health checks verify component availability