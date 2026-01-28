#!/bin/bash

# Hello Service - Integration Test Runner
# This script runs integration tests with the service running in Docker

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
echo -e "${BLUE}Hello Service - Integration Tests${NC}"
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
    print_info "Stopping hello-service..."
    cd "$PROJECT_DIR"
    docker compose stop hello-service 2>/dev/null || true
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
print_info "Building hello-service..."
docker compose build hello-service
echo ""

# Step 2: Start hello-service
print_info "Starting hello-service..."
docker compose up -d hello-service

# Wait for service to be healthy
print_warning "Waiting for hello-service to be ready..."
MAX_WAIT=60
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if docker compose ps hello-service | grep -q "healthy"; then
        print_info "Hello service is ready"
        break
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
done

if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
    print_error "Hello service failed to start"
    docker compose logs hello-service
    exit 1
fi

echo ""

# Step 3: Show service status
print_info "Service status:"
docker compose ps hello-service
echo ""

# Step 4: Run integration tests
print_info "Running integration tests..."
echo ""

# Set environment variables for tests
export GRPC_HOST="localhost"
export GRPC_PORT="9090"

# Run tests from hello-service directory
cd "$PROJECT_DIR/apps/hello-service"
./gradlew integrationTest --info

TEST_EXIT_CODE=$?

# Show service logs if tests failed
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo ""
    print_error "Integration tests failed!"
    print_info "Showing service logs:"
    docker compose logs hello-service
    exit $TEST_EXIT_CODE
fi

echo ""
print_info "✓ All integration tests passed!"
echo ""

exit 0
