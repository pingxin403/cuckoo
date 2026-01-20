#!/bin/bash

# URL Shortener Service - Integration Test Runner
# This script runs end-to-end integration tests with real MySQL and Redis
# Uses the root docker-compose.yml for service orchestration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}URL Shortener - Integration Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to cleanup
cleanup() {
    print_info "Stopping shortener-service..."
    cd "$PROJECT_DIR"
    docker compose stop shortener-service 2>/dev/null || true
    print_info "Cleanup complete"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Change to project root directory
cd "$PROJECT_DIR"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker compose is available
if ! docker compose version &> /dev/null; then
    print_error "docker compose is not available. Please install Docker Desktop or Docker Compose plugin."
    exit 1
fi

# Step 1: Build the service
print_info "Building shortener-service..."
docker compose build shortener-service
echo ""

# Step 2: Start dependencies
print_info "Starting MySQL and Redis..."
docker compose up -d mysql redis

# Wait for MySQL to be healthy
print_warning "Waiting for MySQL to be ready..."
MAX_WAIT=60
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if docker compose exec mysql mysqladmin ping -h localhost -uroot -proot_password --silent 2>/dev/null; then
        print_info "MySQL is ready"
        break
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
    print_error "MySQL failed to start"
    docker compose logs mysql
    exit 1
fi

# Wait for Redis to be healthy
print_warning "Waiting for Redis to be ready..."
WAIT_COUNT=0
while [ $WAIT_COUNT -lt 30 ]; do
    if docker compose exec redis redis-cli ping 2>/dev/null | grep -q PONG; then
        print_info "Redis is ready"
        break
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq 30 ]; then
    print_error "Redis failed to start"
    docker compose logs redis
    exit 1
fi

echo ""

# Step 3: Start shortener service
print_info "Starting shortener-service..."
docker compose up -d shortener-service

# Wait for service to be healthy
print_warning "Waiting for shortener-service to be ready..."
WAIT_COUNT=0
while [ $WAIT_COUNT -lt 60 ]; do
    if curl -sf http://localhost:8081/health > /dev/null 2>&1; then
        print_info "Shortener service is ready"
        break
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq 60 ]; then
    print_error "Shortener service failed to start"
    docker compose logs shortener-service
    exit 1
fi

echo ""

# Step 4: Show service status
print_info "Service status:"
docker compose ps mysql redis shortener-service
echo ""

# Step 5: Run integration tests
print_info "Running integration tests..."
echo ""

# Set environment variables for tests
export GRPC_ADDR="localhost:9092"
export BASE_URL="http://localhost:8081"

# Run tests
cd "$PROJECT_DIR/apps/shortener-service"
go test -v ./integration_test/... -count=1 -timeout 5m

TEST_EXIT_CODE=$?

# Show service logs if tests failed
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo ""
    print_error "Integration tests failed!"
    print_info "Showing service logs:"
    docker compose logs shortener-service
    exit $TEST_EXIT_CODE
fi

echo ""
print_info "✓ All integration tests passed!"
echo ""

exit 0
