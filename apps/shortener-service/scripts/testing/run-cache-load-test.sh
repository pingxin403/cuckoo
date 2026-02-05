#!/bin/bash

# Cache Effectiveness Load Test Runner
# Runs load tests to verify cache effectiveness

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROJECT_DIR="$(cd "$SERVICE_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Cache Effectiveness Load Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: k6 is not installed${NC}"
    echo "Please install k6: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if service is running
echo -e "${YELLOW}Checking if shortener-service is running...${NC}"
if ! curl -sf http://localhost:8081/health > /dev/null 2>&1; then
    echo -e "${RED}Error: shortener-service is not running${NC}"
    echo "Please start the service first:"
    echo "  cd $PROJECT_DIR"
    echo "  docker compose -f deploy/docker/docker-compose.infra.yml -f deploy/docker/docker-compose.services.yml up -d shortener-service"
    exit 1
fi
echo -e "${GREEN}✓ Service is running${NC}"
echo ""

# Check if Redis is available
echo -e "${YELLOW}Checking Redis availability...${NC}"
if docker compose -f "$PROJECT_DIR/deploy/docker/docker-compose.infra.yml" exec redis redis-cli ping 2>/dev/null | grep -q PONG; then
    echo -e "${GREEN}✓ Redis is available (L2 cache enabled)${NC}"
    REDIS_AVAILABLE=true
else
    echo -e "${YELLOW}⚠ Redis is not available (L1 cache only)${NC}"
    REDIS_AVAILABLE=false
fi
echo ""

# Run cache effectiveness test
echo -e "${GREEN}Running cache effectiveness test...${NC}"
echo -e "${BLUE}This test will run 3 scenarios:${NC}"
echo -e "  1. Cold cache (30s) - First access to URLs"
echo -e "  2. Warm cache (60s) - Repeated access with 80/20 distribution"
echo -e "  3. Hot cache (80s) - High concurrency on popular URLs"
echo ""

cd "$SERVICE_DIR/load_test"

k6 run \
  --out json=cache-effectiveness-results.json \
  --summary-export=cache-effectiveness-summary.json \
  cache-effectiveness-test.js

TEST_EXIT_CODE=$?

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ Cache effectiveness test completed successfully!${NC}"
    echo ""
    
    # Display summary
    echo -e "${BLUE}=== Test Summary ===${NC}"
    if [ -f cache-effectiveness-summary.json ]; then
        echo -e "${YELLOW}Response Time Percentiles:${NC}"
        cat cache-effectiveness-summary.json | jq -r '
          .metrics | to_entries[] | 
          select(.key | contains("http_req_duration")) | 
          "\(.key): p95=\(.value.values.p95)ms, p99=\(.value.values.p99)ms"
        ' 2>/dev/null || echo "Summary file exists but couldn't parse"
        
        echo ""
        echo -e "${YELLOW}Custom Metrics:${NC}"
        cat cache-effectiveness-summary.json | jq -r '
          .metrics | to_entries[] | 
          select(.key | contains("cache") or contains("db") or contains("error")) | 
          "\(.key): \(.value.values.count // .value.values.rate)"
        ' 2>/dev/null || echo "Summary file exists but couldn't parse"
    fi
    
    echo ""
    echo -e "${BLUE}=== Cache Effectiveness Analysis ===${NC}"
    echo -e "Review the metrics above to verify:"
    echo -e "  ${GREEN}✓${NC} Cold cache: Higher response times (p95 < 500ms)"
    echo -e "  ${GREEN}✓${NC} Warm cache: Lower response times (p95 < 100ms)"
    echo -e "  ${GREEN}✓${NC} Hot cache: Very low response times (p95 < 50ms)"
    echo -e "  ${GREEN}✓${NC} Cache hit rate increases across scenarios"
    echo ""
    
    if [ "$REDIS_AVAILABLE" = true ]; then
        echo -e "${GREEN}✓ L1 + L2 cache (Redis) enabled${NC}"
        echo -e "  Expected: Very high cache hit rate in warm/hot scenarios"
    else
        echo -e "${YELLOW}⚠ L1 cache only (no Redis)${NC}"
        echo -e "  Expected: Good cache hit rate but lower than with Redis"
    fi
    echo ""
    
    # Check Redis stats
    if [ "$REDIS_AVAILABLE" = true ]; then
        echo -e "${BLUE}=== Redis Cache Stats ===${NC}"
        docker compose -f "$PROJECT_DIR/deploy/docker/docker-compose.infra.yml" exec redis redis-cli INFO stats | grep -E "keyspace_hits|keyspace_misses" || true
        echo ""
    fi
    
else
    echo -e "${RED}✗ Cache effectiveness test failed${NC}"
    exit $TEST_EXIT_CODE
fi

exit 0
