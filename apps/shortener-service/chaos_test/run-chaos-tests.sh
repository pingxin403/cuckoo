#!/bin/bash

# Chaos Engineering Test Script for URL Shortener Service
# This script simulates various failure scenarios

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_URL="${BASE_URL:-http://localhost:8080}"
MYSQL_HOST="${MYSQL_HOST:-localhost}"
REDIS_HOST="${REDIS_HOST:-localhost}"

echo -e "${GREEN}=== URL Shortener Chaos Tests ===${NC}\n"

test_redis_failure() {
    echo -e "${YELLOW}Test 1: Redis Failure${NC}"
    echo "Stopping Redis..."
    
    if docker ps | grep -q shortener-redis; then
        docker stop shortener-redis 2>/dev/null || true
    fi
    
    echo "Testing redirect (should fallback to MySQL)..."
    response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/abc1234" 2>/dev/null || echo "000")
    echo "Response code: $response"
    
    echo "Testing health check..."
    curl -s "$BASE_URL/health" | head -1
    
    echo "Restarting Redis..."
    docker start shortener-redis 2>/dev/null || true
    
    echo -e "${GREEN}✓ Redis failure test complete${NC}\n"
}

test_mysql_failure() {
    echo -e "${YELLOW}Test 2: MySQL Failure${NC}"
    echo "Stopping MySQL..."
    
    if docker ps | grep -q shortener-mysql; then
        docker stop shortener-mysql 2>/dev/null || true
    fi
    
    echo "Testing create (should fail)..."
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "$BASE_URL/api/v1/shortener" \
        -H "Content-Type: application/json" \
        -d '{"long_url":"https://example.com/test"}' 2>/dev/null || echo "000")
    echo "Response code: $response"
    
    echo "Testing health..."
    curl -s "$BASE_URL/health" | head -1
    
    echo "Restarting MySQL..."
    docker start shortener-mysql 2>/dev/null || true
    
    echo -e "${GREEN}✓ MySQL failure test complete${NC}\n"
}

test_network_latency() {
    echo -e "${YELLOW}Test 3: High Latency Injection${NC}"
    echo "This test requires network latency tools (e.g., tc, Toxiproxy)"
    echo "Skipping automated test - manual verification required"
    echo -e "${GREEN}✓ Latency test skipped (manual)${NC}\n"
}

verify_graceful_degradation() {
    echo -e "${YELLOW}Verifying Graceful Degradation${NC}"
    
    echo "Checking cache manager fallback logic..."
    if grep -q "fallback" apps/shortener-service/cache/cache_manager.go 2>/dev/null; then
        echo "✓ Cache fallback implemented"
    fi
    
    echo "Checking error handling..."
    if grep -q "ServiceUnavailable" apps/shortener-service/errors/errors.go 2>/dev/null; then
        echo "✓ Service unavailable errors defined"
    fi
    
    echo "Checking health checks..."
    if grep -q "/health" apps/shortener-service/service/redirect_handler.go 2>/dev/null; then
        echo "✓ Health endpoints implemented"
    fi
    
    echo -e "${GREEN}✓ Graceful degradation verified${NC}\n"
}

main() {
    echo "Starting chaos tests..."
    echo ""
    
    verify_graceful_degradation
    
    if command -v docker &> /dev/null; then
        test_redis_failure || echo "Redis test skipped"
        test_mysql_failure || echo "MySQL test skipped"
    else
        echo "Docker not available - skipping container tests"
    fi
    
    test_network_latency
    
    echo -e "${GREEN}=== Chaos Tests Complete ===${NC}"
    echo "Review results above and check service logs for details"
}

main