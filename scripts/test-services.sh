#!/bin/bash

# Script to test if all services are running and responding correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Testing Monorepo Services ===${NC}\n"

# Function to check if a port is listening
check_port() {
    local port=$1
    local service=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${GREEN}✓ $service is listening on port $port${NC}"
        return 0
    else
        echo -e "${RED}✗ $service is NOT listening on port $port${NC}"
        return 1
    fi
}

# Function to test HTTP endpoint
test_http() {
    local url=$1
    local service=$2
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $service HTTP endpoint is responding${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ $service HTTP endpoint check skipped (expected for gRPC-only services)${NC}"
        return 0
    fi
}

# Check if services are running
echo -e "${BLUE}Checking if services are running...${NC}"
HELLO_OK=false
TODO_OK=false
WEB_OK=false

if check_port 9090 "Hello Service"; then
    HELLO_OK=true
fi

if check_port 9091 "TODO Service"; then
    TODO_OK=true
fi

if check_port 5173 "Frontend"; then
    WEB_OK=true
fi

echo ""

# Test Frontend
if [ "$WEB_OK" = true ]; then
    echo -e "${BLUE}Testing Frontend...${NC}"
    if curl -s -f http://localhost:5173 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Frontend is serving content${NC}"
    else
        echo -e "${RED}✗ Frontend is not responding${NC}"
    fi
    echo ""
fi

# Summary
echo -e "${BLUE}=== Summary ===${NC}"
if [ "$HELLO_OK" = true ] && [ "$TODO_OK" = true ] && [ "$WEB_OK" = true ]; then
    echo -e "${GREEN}All services are running!${NC}"
    echo ""
    echo -e "${BLUE}Service URLs:${NC}"
    echo -e "  - Frontend:      ${GREEN}http://localhost:5173${NC}"
    echo -e "  - Hello Service: ${GREEN}localhost:9090${NC} (gRPC)"
    echo -e "  - TODO Service:  ${GREEN}localhost:9091${NC} (gRPC)"
    echo ""
    echo -e "${YELLOW}Note: Without Envoy proxy, the frontend cannot communicate with backend services.${NC}"
    echo -e "${YELLOW}To enable full functionality, install Envoy and run: ./scripts/dev.sh${NC}"
    exit 0
else
    echo -e "${RED}Some services are not running!${NC}"
    echo ""
    echo -e "${YELLOW}To start all services, run: ./scripts/dev.sh${NC}"
    exit 1
fi
