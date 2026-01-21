#!/bin/bash

# URL Shortener Service - API Gateway Verification Script
# This script verifies Envoy routing rules for the shortener service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ENVOY_PORT=8080
ENVOY_ADMIN_PORT=9901
GRPC_PORT=9092
HTTP_PORT=8081  # Using 8081 for test environment
METRICS_PORT=9091

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}URL Shortener - Gateway Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=0
    
    echo -e "${YELLOW}Waiting for $name to be ready...${NC}"
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ $name is ready${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    echo -e "${RED}✗ $name failed to start${NC}"
    return 1
}

# Step 1: Check if services are running
echo -e "${BLUE}Step 1: Checking service status...${NC}"
echo ""

SERVICES_RUNNING=true

if check_port $GRPC_PORT; then
    echo -e "${GREEN}✓ Shortener gRPC service is running on port $GRPC_PORT${NC}"
else
    echo -e "${RED}✗ Shortener gRPC service is NOT running on port $GRPC_PORT${NC}"
    SERVICES_RUNNING=false
fi

if check_port $HTTP_PORT; then
    echo -e "${GREEN}✓ Shortener HTTP service is running on port $HTTP_PORT${NC}"
else
    echo -e "${RED}✗ Shortener HTTP service is NOT running on port $HTTP_PORT${NC}"
    SERVICES_RUNNING=false
fi

if ! $SERVICES_RUNNING; then
    echo ""
    echo -e "${YELLOW}Services are not running. Starting with Docker Compose...${NC}"
    echo ""
    
    # Start services
    docker compose -f docker-compose.test.yml up -d
    
    # Wait for services to be ready
    wait_for_service "http://localhost:$HTTP_PORT/health" "Shortener HTTP service"
    
    echo ""
fi

# Step 2: Check if Envoy is running
echo -e "${BLUE}Step 2: Checking Envoy status...${NC}"
echo ""

if check_port $ENVOY_PORT; then
    echo -e "${GREEN}✓ Envoy is running on port $ENVOY_PORT${NC}"
    ENVOY_RUNNING=true
else
    echo -e "${YELLOW}⚠ Envoy is NOT running on port $ENVOY_PORT${NC}"
    echo -e "${YELLOW}To start Envoy, run:${NC}"
    echo -e "${YELLOW}  docker run -d --name envoy-gateway --network host -v \$(pwd)/tools/envoy/envoy-local.yaml:/etc/envoy/envoy.yaml envoyproxy/envoy:v1.28-latest${NC}"
    echo ""
    ENVOY_RUNNING=false
fi

if $ENVOY_RUNNING; then
    # Check Envoy admin interface
    if check_port $ENVOY_ADMIN_PORT; then
        echo -e "${GREEN}✓ Envoy admin interface is accessible on port $ENVOY_ADMIN_PORT${NC}"
        echo -e "${BLUE}  Admin URL: http://localhost:$ENVOY_ADMIN_PORT${NC}"
    fi
    echo ""
fi

# Step 3: Test direct service access (without Envoy)
echo -e "${BLUE}Step 3: Testing direct service access...${NC}"
echo ""

# Test health endpoint
echo -e "${YELLOW}Testing health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:$HTTP_PORT/health)
HEALTH_CODE=$(echo "$HEALTH_RESPONSE" | tail -n1)
if [ "$HEALTH_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Health check passed (HTTP $HEALTH_CODE)${NC}"
else
    echo -e "${RED}✗ Health check failed (HTTP $HEALTH_CODE)${NC}"
fi

# Test readiness endpoint
echo -e "${YELLOW}Testing readiness endpoint...${NC}"
READY_RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:$HTTP_PORT/ready)
READY_CODE=$(echo "$READY_RESPONSE" | tail -n1)
if [ "$READY_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Readiness check passed (HTTP $READY_CODE)${NC}"
else
    echo -e "${RED}✗ Readiness check failed (HTTP $READY_CODE)${NC}"
fi

# Test gRPC service with grpcurl (if available)
if command -v grpcurl &> /dev/null; then
    echo -e "${YELLOW}Testing gRPC service...${NC}"
    
    # Create a short link
    CREATE_RESPONSE=$(grpcurl -plaintext -d '{"long_url": "https://example.com/test"}' \
        localhost:$GRPC_PORT \
        api.v1.ShortenerService/CreateShortLink 2>&1)
    
    if echo "$CREATE_RESPONSE" | grep -q "shortUrl"; then
        echo -e "${GREEN}✓ gRPC CreateShortLink works${NC}"
        
        # Extract short code from response
        SHORT_CODE=$(echo "$CREATE_RESPONSE" | grep -o '"shortCode": "[^"]*"' | cut -d'"' -f4)
        echo -e "${BLUE}  Created short code: $SHORT_CODE${NC}"
        
        # Test HTTP redirect
        if [ -n "$SHORT_CODE" ]; then
            echo -e "${YELLOW}Testing HTTP redirect for /$SHORT_CODE...${NC}"
            REDIRECT_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -o /dev/null -D - http://localhost:$HTTP_PORT/$SHORT_CODE)
            if echo "$REDIRECT_RESPONSE" | grep -q "HTTP/1.1 302"; then
                echo -e "${GREEN}✓ HTTP redirect works (302 Found)${NC}"
                LOCATION=$(echo "$REDIRECT_RESPONSE" | grep -i "Location:" | cut -d' ' -f2 | tr -d '\r')
                echo -e "${BLUE}  Redirects to: $LOCATION${NC}"
            else
                echo -e "${RED}✗ HTTP redirect failed${NC}"
                echo -e "${RED}  Response: $REDIRECT_RESPONSE${NC}"
            fi
        fi
    else
        echo -e "${RED}✗ gRPC CreateShortLink failed${NC}"
        echo -e "${RED}  Response: $CREATE_RESPONSE${NC}"
    fi
else
    echo -e "${YELLOW}⚠ grpcurl not installed, skipping gRPC tests${NC}"
    echo -e "${YELLOW}  Install with: brew install grpcurl (macOS) or go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest${NC}"
fi

echo ""

# Step 4: Test Envoy routing (if Envoy is running)
if $ENVOY_RUNNING; then
    echo -e "${BLUE}Step 4: Testing Envoy routing...${NC}"
    echo ""
    
    # Test health check through Envoy
    echo -e "${YELLOW}Testing health endpoint through Envoy...${NC}"
    ENVOY_HEALTH=$(curl -s -w "\n%{http_code}" http://localhost:$ENVOY_PORT/health)
    ENVOY_HEALTH_CODE=$(echo "$ENVOY_HEALTH" | tail -n1)
    if [ "$ENVOY_HEALTH_CODE" = "200" ]; then
        echo -e "${GREEN}✓ Health check through Envoy passed${NC}"
    else
        echo -e "${RED}✗ Health check through Envoy failed (HTTP $ENVOY_HEALTH_CODE)${NC}"
    fi
    
    # Test gRPC API routing through Envoy (if grpcurl is available)
    if command -v grpcurl &> /dev/null; then
        echo -e "${YELLOW}Testing gRPC API through Envoy (/api/shortener)...${NC}"
        
        # Note: Envoy's gRPC-Web filter requires special headers
        # For now, we'll test if the route exists by checking Envoy config
        ENVOY_CONFIG=$(curl -s http://localhost:$ENVOY_ADMIN_PORT/config_dump)
        if echo "$ENVOY_CONFIG" | grep -q "shortener_service_grpc"; then
            echo -e "${GREEN}✓ Envoy has shortener_service_grpc cluster configured${NC}"
        else
            echo -e "${RED}✗ Envoy missing shortener_service_grpc cluster${NC}"
        fi
        
        if echo "$ENVOY_CONFIG" | grep -q "/api/shortener"; then
            echo -e "${GREEN}✓ Envoy has /api/shortener route configured${NC}"
        else
            echo -e "${RED}✗ Envoy missing /api/shortener route${NC}"
        fi
    fi
    
    # Test HTTP redirect routing through Envoy
    if [ -n "$SHORT_CODE" ]; then
        echo -e "${YELLOW}Testing HTTP redirect through Envoy (/$SHORT_CODE)...${NC}"
        ENVOY_REDIRECT=$(curl -s -w "\nHTTP_CODE:%{http_code}" -o /dev/null -D - http://localhost:$ENVOY_PORT/$SHORT_CODE)
        if echo "$ENVOY_REDIRECT" | grep -q "HTTP/1.1 302"; then
            echo -e "${GREEN}✓ HTTP redirect through Envoy works${NC}"
            ENVOY_LOCATION=$(echo "$ENVOY_REDIRECT" | grep -i "Location:" | cut -d' ' -f2 | tr -d '\r')
            echo -e "${BLUE}  Redirects to: $ENVOY_LOCATION${NC}"
        else
            echo -e "${RED}✗ HTTP redirect through Envoy failed${NC}"
            echo -e "${RED}  Response: $ENVOY_REDIRECT${NC}"
        fi
    fi
    
    # Test that non-matching routes don't interfere
    echo -e "${YELLOW}Testing non-matching route (should 404)...${NC}"
    NOT_FOUND=$(curl -s -w "\n%{http_code}" http://localhost:$ENVOY_PORT/invalid-route-12345)
    NOT_FOUND_CODE=$(echo "$NOT_FOUND" | tail -n1)
    if [ "$NOT_FOUND_CODE" = "404" ]; then
        echo -e "${GREEN}✓ Non-matching routes return 404${NC}"
    else
        echo -e "${YELLOW}⚠ Non-matching route returned HTTP $NOT_FOUND_CODE (expected 404)${NC}"
    fi
    
    echo ""
    
    # Display Envoy cluster status
    echo -e "${BLUE}Envoy Cluster Status:${NC}"
    CLUSTER_STATUS=$(curl -s http://localhost:$ENVOY_ADMIN_PORT/clusters | grep shortener)
    if [ -n "$CLUSTER_STATUS" ]; then
        echo "$CLUSTER_STATUS" | while read -r line; do
            if echo "$line" | grep -q "health_flags::healthy"; then
                echo -e "${GREEN}✓ $line${NC}"
            else
                echo -e "${YELLOW}  $line${NC}"
            fi
        done
    else
        echo -e "${YELLOW}⚠ No shortener clusters found${NC}"
    fi
else
    echo -e "${BLUE}Step 4: Skipped (Envoy not running)${NC}"
    echo ""
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Verification Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if $SERVICES_RUNNING; then
    echo -e "${GREEN}✓ Shortener service is running${NC}"
else
    echo -e "${RED}✗ Shortener service needs to be started${NC}"
fi

if $ENVOY_RUNNING; then
    echo -e "${GREEN}✓ Envoy gateway is running${NC}"
    echo -e "${BLUE}  Test URLs:${NC}"
    echo -e "${BLUE}    - Health: http://localhost:$ENVOY_PORT/health${NC}"
    echo -e "${BLUE}    - Redirect: http://localhost:$ENVOY_PORT/{code}${NC}"
    echo -e "${BLUE}    - Admin: http://localhost:$ENVOY_ADMIN_PORT${NC}"
else
    echo -e "${YELLOW}⚠ Envoy gateway is not running${NC}"
    echo -e "${YELLOW}  To start Envoy:${NC}"
    echo -e "${YELLOW}    docker run -d --name envoy-gateway --network host \\${NC}"
    echo -e "${YELLOW}      -v \$(pwd)/tools/envoy/envoy-local.yaml:/etc/envoy/envoy.yaml \\${NC}"
    echo -e "${YELLOW}      envoyproxy/envoy:v1.28-latest${NC}"
fi

echo ""
echo -e "${BLUE}Useful Commands:${NC}"
echo -e "${BLUE}  - View Envoy config: curl http://localhost:$ENVOY_ADMIN_PORT/config_dump${NC}"
echo -e "${BLUE}  - View Envoy stats: curl http://localhost:$ENVOY_ADMIN_PORT/stats${NC}"
echo -e "${BLUE}  - View Envoy clusters: curl http://localhost:$ENVOY_ADMIN_PORT/clusters${NC}"
echo -e "${BLUE}  - Stop Envoy: docker stop envoy-gateway && docker rm envoy-gateway${NC}"
echo -e "${BLUE}  - Stop services: docker compose -f docker-compose.test.yml down${NC}"
echo ""
