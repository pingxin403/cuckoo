#!/bin/bash

# Development startup script for Monorepo Hello/TODO Services
# This script starts all services in development mode with proper cleanup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# PID tracking
PIDS=()
LOG_DIR="logs"

# Create logs directory
mkdir -p "$LOG_DIR"

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"
    
    # Kill all tracked processes
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "${BLUE}Stopping process $pid${NC}"
            kill "$pid" 2>/dev/null || true
        fi
    done
    
    # Wait for processes to terminate
    sleep 2
    
    # Force kill if still running
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "${RED}Force killing process $pid${NC}"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
    
    echo -e "${GREEN}All services stopped${NC}"
    exit 0
}

# Set up trap for cleanup
trap cleanup SIGINT SIGTERM EXIT

# Function to check if a port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${RED}Error: Port $port is already in use${NC}"
        exit 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local name=$1
    local port=$2
    local max_attempts=30
    local attempt=0
    
    echo -e "${BLUE}Waiting for $name to be ready on port $port...${NC}"
    
    while [ $attempt -lt $max_attempts ]; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo -e "${GREEN}$name is ready!${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    echo -e "${RED}$name failed to start within timeout${NC}"
    return 1
}

echo -e "${GREEN}=== Starting Monorepo Development Environment ===${NC}\n"

# Check if required ports are available
echo -e "${BLUE}Checking port availability...${NC}"
check_port 9090  # Hello Service
check_port 9091  # TODO Service
check_port 8080  # Envoy
check_port 5173  # Vite dev server

# Check if Envoy is installed
if ! command -v envoy &> /dev/null; then
    echo -e "${YELLOW}Warning: Envoy not found. Please install Envoy to use the API gateway.${NC}"
    echo -e "${YELLOW}You can install it via: brew install envoy (macOS) or see https://www.envoyproxy.io/docs/envoy/latest/start/install${NC}"
    echo -e "${YELLOW}Continuing without Envoy...${NC}\n"
    SKIP_ENVOY=true
else
    SKIP_ENVOY=false
fi

# Start Hello Service (Java/Spring Boot)
echo -e "${BLUE}Starting Hello Service (Java/Spring Boot)...${NC}"
cd apps/hello-service
./gradlew bootRun > "../../$LOG_DIR/hello-service.log" 2>&1 &
HELLO_PID=$!
PIDS+=($HELLO_PID)
cd ../..
echo -e "${GREEN}Hello Service started (PID: $HELLO_PID)${NC}"

# Wait for Hello Service to be ready
wait_for_service "Hello Service" 9090

# Start TODO Service (Go)
echo -e "${BLUE}Starting TODO Service (Go)...${NC}"
cd apps/todo-service
export HELLO_SERVICE_ADDR="localhost:9090"
go run . > "../../$LOG_DIR/todo-service.log" 2>&1 &
TODO_PID=$!
PIDS+=($TODO_PID)
cd ../..
echo -e "${GREEN}TODO Service started (PID: $TODO_PID)${NC}"

# Wait for TODO Service to be ready
wait_for_service "TODO Service" 9091

# Start Envoy (if available)
if [ "$SKIP_ENVOY" = false ]; then
    echo -e "${BLUE}Starting Envoy proxy...${NC}"
    envoy -c tools/envoy/envoy-local.yaml > "$LOG_DIR/envoy.log" 2>&1 &
    ENVOY_PID=$!
    PIDS+=($ENVOY_PID)
    echo -e "${GREEN}Envoy started (PID: $ENVOY_PID)${NC}"
    
    # Wait for Envoy to be ready
    wait_for_service "Envoy" 8080
fi

# Start Frontend (React/Vite)
echo -e "${BLUE}Starting Frontend (React/Vite)...${NC}"
cd apps/web
npm run dev > "../../$LOG_DIR/web.log" 2>&1 &
WEB_PID=$!
PIDS+=($WEB_PID)
cd ../..
echo -e "${GREEN}Frontend started (PID: $WEB_PID)${NC}"

# Wait for frontend to be ready
wait_for_service "Frontend" 5173

echo -e "\n${GREEN}=== All services are running ===${NC}\n"
echo -e "${BLUE}Service URLs:${NC}"
echo -e "  - Frontend:      ${GREEN}http://localhost:5173${NC}"
if [ "$SKIP_ENVOY" = false ]; then
    echo -e "  - API Gateway:   ${GREEN}http://localhost:8080${NC}"
    echo -e "  - Envoy Admin:   ${GREEN}http://localhost:9901${NC}"
fi
echo -e "  - Hello Service: ${GREEN}localhost:9090${NC} (gRPC)"
echo -e "  - TODO Service:  ${GREEN}localhost:9091${NC} (gRPC)"
echo -e "\n${BLUE}Logs are available in:${NC} $LOG_DIR/"
echo -e "\n${YELLOW}Press Ctrl+C to stop all services${NC}\n"

# Keep script running and tail logs
tail -f "$LOG_DIR"/*.log &
TAIL_PID=$!
PIDS+=($TAIL_PID)

# Wait indefinitely
wait
