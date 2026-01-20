#!/bin/bash

# Script to run integration tests for the URL Shortener Service
# This script starts the test environment with Docker Compose and runs integration tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=========================================="
echo "URL Shortener Service - Integration Tests"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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
    print_info "Cleaning up test environment..."
    cd "$PROJECT_DIR"
    docker compose -f docker-compose.test.yml down -v
    print_info "Cleanup complete"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Change to project directory
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

# Stop any existing test containers
print_info "Stopping any existing test containers..."
docker compose -f docker-compose.test.yml down -v 2>/dev/null || true

# Build the service image
print_info "Building service image..."
docker compose -f docker-compose.test.yml build shortener-service-test

# Start the test environment
print_info "Starting test environment (MySQL, Redis, Service)..."
docker compose -f docker-compose.test.yml up -d

# Wait for services to be healthy
print_info "Waiting for services to be healthy..."
MAX_WAIT=60
WAIT_COUNT=0

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    # Check MySQL health
    MYSQL_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' shortener-mysql-test 2>/dev/null || echo "not_found")
    
    # Check Redis health
    REDIS_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' shortener-redis-test 2>/dev/null || echo "not_found")
    
    # Check Service health
    SERVICE_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' shortener-service-test 2>/dev/null || echo "not_found")
    
    if [ "$MYSQL_HEALTH" = "healthy" ] && [ "$REDIS_HEALTH" = "healthy" ] && [ "$SERVICE_HEALTH" = "healthy" ]; then
        print_info "All services are healthy!"
        break
    fi
    
    echo "Waiting for services... MySQL: $MYSQL_HEALTH, Redis: $REDIS_HEALTH, Service: $SERVICE_HEALTH"
    sleep 2
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
    print_error "Services did not become healthy in time"
    print_info "Showing service logs:"
    docker-compose -f docker-compose.test.yml logs
    exit 1
fi

# Show service status
print_info "Service status:"
docker compose -f docker-compose.test.yml ps

# Run integration tests
print_info "Running integration tests..."
echo ""

# Set environment variables for tests
export GRPC_ADDR="localhost:9092"
export BASE_URL="http://localhost:8081"

# Run tests with integration tag
cd "$PROJECT_DIR"
go test -v -tags=integration ./integration_test/... -timeout 5m

TEST_EXIT_CODE=$?

# Show service logs if tests failed
if [ $TEST_EXIT_CODE -ne 0 ]; then
    print_error "Integration tests failed!"
    print_info "Showing service logs:"
    docker compose -f docker-compose.test.yml logs shortener-service-test
    exit $TEST_EXIT_CODE
fi

print_info "Integration tests passed!"
echo ""
print_info "Test environment will be cleaned up automatically"

exit 0
