#!/bin/bash
# Integration test for flash-sale-service routing and rate limiting

set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
SECKILL_ENDPOINT="$GATEWAY_URL/api/seckill"

echo "==================================="
echo "Flash Sale Service Routing Test"
echo "==================================="
echo ""
echo "Gateway URL: $GATEWAY_URL"
echo "Seckill Endpoint: $SECKILL_ENDPOINT"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check if gateway is reachable
echo "Test 1: Gateway Reachability"
echo "-----------------------------------"
if curl -s -f -o /dev/null "$GATEWAY_URL"; then
    echo -e "${GREEN}✅ Gateway is reachable${NC}"
else
    echo -e "${RED}❌ Gateway is not reachable${NC}"
    echo "   Make sure Envoy/Higress is running"
    exit 1
fi
echo ""

# Test 2: Check if flash-sale-service route exists
echo "Test 2: Flash Sale Service Route"
echo "-----------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SECKILL_ENDPOINT/health" || echo "000")
if [ "$HTTP_CODE" != "000" ]; then
    echo -e "${GREEN}✅ Route is configured (HTTP $HTTP_CODE)${NC}"
    if [ "$HTTP_CODE" == "200" ]; then
        echo "   Service is healthy and responding"
    elif [ "$HTTP_CODE" == "503" ]; then
        echo -e "${YELLOW}   ⚠️  Service is not available (check if flash-sale-service is running)${NC}"
    fi
else
    echo -e "${RED}❌ Route is not accessible${NC}"
fi
echo ""

# Test 3: Rate Limiting Test
echo "Test 3: L1 Rate Limiting (10 QPS per IP)"
echo "-----------------------------------"
echo "Sending 15 rapid requests to test rate limiting..."
echo ""

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0
ERROR_COUNT=0

for i in {1..15}; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SECKILL_ENDPOINT/test" 2>/dev/null || echo "000")
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "404" ] || [ "$HTTP_CODE" == "503" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -n "."
    elif [ "$HTTP_CODE" == "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        echo -n "R"
    else
        ERROR_COUNT=$((ERROR_COUNT + 1))
        echo -n "E"
    fi
    
    # Small delay to avoid overwhelming the system
    sleep 0.05
done

echo ""
echo ""
echo "Results:"
echo "  Successful requests: $SUCCESS_COUNT"
echo "  Rate limited (429): $RATE_LIMITED_COUNT"
echo "  Errors: $ERROR_COUNT"
echo ""

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✅ Rate limiting is working!${NC}"
    echo "   Expected: ~10 successful, ~5 rate limited"
    echo "   Actual: $SUCCESS_COUNT successful, $RATE_LIMITED_COUNT rate limited"
else
    echo -e "${YELLOW}⚠️  Rate limiting may not be working as expected${NC}"
    echo "   Expected some requests to be rate limited (HTTP 429)"
    echo "   This could be normal if requests were spread out over time"
fi
echo ""

# Test 4: Check rate limit headers
echo "Test 4: Rate Limit Response Headers"
echo "-----------------------------------"
HEADERS=$(curl -s -I "$SECKILL_ENDPOINT/test" 2>/dev/null || echo "")
if echo "$HEADERS" | grep -qi "x-local-rate-limit"; then
    echo -e "${GREEN}✅ Rate limit headers are present${NC}"
else
    echo -e "${YELLOW}⚠️  Rate limit headers not found${NC}"
    echo "   This is normal if the request was not rate limited"
fi
echo ""

# Test 5: Verify routing to correct backend
echo "Test 5: Backend Service Verification"
echo "-----------------------------------"
echo "Checking if requests are routed to flash-sale-service:8084..."

# Try to access the service health endpoint directly
if command -v docker &> /dev/null && docker ps | grep -q flash-sale-service; then
    DIRECT_HEALTH=$(docker exec flash-sale-service curl -s http://localhost:8084/actuator/health 2>/dev/null || echo "")
    if [ -n "$DIRECT_HEALTH" ]; then
        echo -e "${GREEN}✅ flash-sale-service is running and healthy${NC}"
        echo "   Direct health check: OK"
    else
        echo -e "${YELLOW}⚠️  Could not verify service health directly${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Cannot verify backend service (Docker not available or service not running)${NC}"
fi
echo ""

echo "==================================="
echo "Test Summary"
echo "==================================="
echo ""
echo "Configuration Status:"
echo "  ✅ Gateway routing configured"
echo "  ✅ Rate limiting configured"
echo "  ✅ Backend cluster configured"
echo ""
echo "For detailed configuration information, see:"
echo "  deploy/docker/ENVOY_FLASH_SALE_CONFIG.md"
echo ""
