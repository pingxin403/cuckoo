#!/bin/bash

# Test Observability Stack
# Verifies that all observability components are running and accessible

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0

echo "========================================="
echo "Testing Observability Stack"
echo "========================================="
echo ""

# Function to test HTTP endpoint
test_http() {
    local name=$1
    local url=$2
    local expected_status=${3:-200}
    
    echo -n "Testing $name... "
    
    if response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>&1); then
        if [ "$response" = "$expected_status" ]; then
            echo -e "${GREEN}✓ PASSED${NC} (HTTP $response)"
            ((PASSED++))
        else
            echo -e "${RED}✗ FAILED${NC} (Expected HTTP $expected_status, got $response)"
            ((FAILED++))
        fi
    else
        echo -e "${RED}✗ FAILED${NC} (Connection failed)"
        ((FAILED++))
    fi
}

# Function to test service availability
test_service() {
    local name=$1
    local host=$2
    local port=$3
    
    echo -n "Testing $name connectivity... "
    
    if nc -z -w5 "$host" "$port" 2>/dev/null; then
        echo -e "${GREEN}✓ PASSED${NC} (Port $port is open)"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC} (Port $port is not accessible)"
        ((FAILED++))
    fi
}

# Function to check Docker container
check_container() {
    local name=$1
    
    echo -n "Checking $name container... "
    
    if docker ps --format '{{.Names}}' | grep -q "^${name}$"; then
        status=$(docker inspect --format='{{.State.Status}}' "$name")
        if [ "$status" = "running" ]; then
            echo -e "${GREEN}✓ PASSED${NC} (Running)"
            ((PASSED++))
        else
            echo -e "${RED}✗ FAILED${NC} (Status: $status)"
            ((FAILED++))
        fi
    else
        echo -e "${RED}✗ FAILED${NC} (Container not found)"
        ((FAILED++))
    fi
}

echo "1. Checking Docker Containers"
echo "------------------------------"
check_container "otel-collector"
check_container "jaeger"
check_container "prometheus"
check_container "grafana"
check_container "loki"
echo ""

echo "2. Testing Service Connectivity"
echo "--------------------------------"
test_service "OpenTelemetry Collector (gRPC)" "localhost" "4317"
test_service "OpenTelemetry Collector (HTTP)" "localhost" "4318"
test_service "Jaeger UI" "localhost" "16686"
test_service "Prometheus" "localhost" "9090"
test_service "Grafana" "localhost" "3000"
test_service "Loki" "localhost" "3100"
echo ""

echo "3. Testing HTTP Endpoints"
echo "-------------------------"
test_http "OpenTelemetry Collector Health" "http://localhost:13133/"
test_http "Jaeger UI" "http://localhost:16686/"
test_http "Prometheus" "http://localhost:9090/-/healthy"
test_http "Grafana Health" "http://localhost:3000/api/health"
test_http "Loki Ready" "http://localhost:3100/ready"
echo ""

echo "4. Testing OpenTelemetry Collector Endpoints"
echo "---------------------------------------------"
# Test OTLP HTTP endpoint (should return 405 Method Not Allowed)
test_http "OTLP HTTP Traces Endpoint" "http://localhost:4318/v1/traces" "405"
test_http "OTLP HTTP Metrics Endpoint" "http://localhost:4318/v1/metrics" "405"
test_http "OTLP HTTP Logs Endpoint" "http://localhost:4318/v1/logs" "405"
echo ""

echo "5. Testing Prometheus Targets"
echo "------------------------------"
echo -n "Checking Prometheus targets... "
if targets=$(curl -s "http://localhost:9090/api/v1/targets" 2>&1); then
    if echo "$targets" | grep -q '"status":"success"'; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC} (API returned error)"
        ((FAILED++))
    fi
else
    echo -e "${RED}✗ FAILED${NC} (Connection failed)"
    ((FAILED++))
fi
echo ""

echo "6. Testing Grafana Datasources"
echo "-------------------------------"
echo -n "Checking Grafana datasources... "
if datasources=$(curl -s -u admin:admin "http://localhost:3000/api/datasources" 2>&1); then
    if echo "$datasources" | grep -q "Prometheus"; then
        echo -e "${GREEN}✓ PASSED${NC} (Datasources configured)"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠ WARNING${NC} (Datasources may not be configured)"
        ((PASSED++))
    fi
else
    echo -e "${RED}✗ FAILED${NC} (Connection failed)"
    ((FAILED++))
fi
echo ""

echo "7. Testing Loki Labels"
echo "----------------------"
echo -n "Checking Loki labels... "
if labels=$(curl -s "http://localhost:3100/loki/api/v1/labels" 2>&1); then
    if echo "$labels" | grep -q '"status":"success"'; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠ WARNING${NC} (No labels yet, may be empty)"
        ((PASSED++))
    fi
else
    echo -e "${RED}✗ FAILED${NC} (Connection failed)"
    ((FAILED++))
fi
echo ""

echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Access the UIs:"
    echo "  - Grafana:    http://localhost:3000 (admin/admin)"
    echo "  - Jaeger:     http://localhost:16686"
    echo "  - Prometheus: http://localhost:9090"
    echo ""
    echo "OTLP Endpoints:"
    echo "  - gRPC: localhost:4317"
    echo "  - HTTP: localhost:4318"
    exit 0
else
    echo -e "${RED}✗ Some tests failed!${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check if all containers are running:"
    echo "     docker compose -f deploy/docker/docker-compose.observability.yml ps"
    echo ""
    echo "  2. Check container logs:"
    echo "     docker logs otel-collector"
    echo "     docker logs jaeger"
    echo "     docker logs prometheus"
    echo "     docker logs grafana"
    echo "     docker logs loki"
    echo ""
    echo "  3. Restart the stack:"
    echo "     make observability-restart"
    exit 1
fi
