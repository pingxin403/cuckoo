#!/bin/bash

# Service Dependency Integration Test Runner for IM Gateway Service
# This script sets up the multi-service test environment, runs integration tests, and cleans up

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo -e "${GREEN}=== IM Gateway Service Dependency Integration Tests ===${NC}"
echo ""

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}Cleaning up test environment...${NC}"
    cd "$SCRIPT_DIR"
    docker-compose -f docker-compose.test.yml down -v
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Trap EXIT to ensure cleanup
trap cleanup EXIT

# Step 1: Start all services
echo -e "${YELLOW}Step 1: Starting all services...${NC}"
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
for i in {1..90}; do
    if docker-compose -f docker-compose.test.yml ps | grep -q "unhealthy"; then
        echo -n "."
        sleep 2
    else
        # Check if all services are up
        RUNNING=$(docker-compose -f docker-compose.test.yml ps --services --filter "status=running" | wc -l)
        TOTAL=$(docker-compose -f docker-compose.test.yml ps --services | wc -l)
        
        if [ "$RUNNING" -eq "$TOTAL" ]; then
            echo ""
            echo -e "${GREEN}All services are healthy and running${NC}"
            break
        fi
        echo -n "."
        sleep 2
    fi
    
    if [ $i -eq 90 ]; then
        echo -e "${RED}Timeout waiting for services to be healthy${NC}"
        docker-compose -f docker-compose.test.yml ps
        docker-compose -f docker-compose.test.yml logs --tail=50
        exit 1
    fi
done

# Step 2: Create Kafka topics
echo -e "${YELLOW}Step 2: Creating Kafka topics...${NC}"
docker-compose -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic group_msg \
    --partitions 3 \
    --replication-factor 1 || true

docker-compose -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic offline_msg \
    --partitions 3 \
    --replication-factor 1 || true

docker-compose -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic membership_change \
    --partitions 3 \
    --replication-factor 1 || true

echo -e "${GREEN}Kafka topics created${NC}"

# Step 3: Wait a bit more for services to fully initialize
echo -e "${YELLOW}Step 3: Waiting for services to fully initialize...${NC}"
sleep 5

# Step 4: Run integration tests
echo -e "${YELLOW}Step 4: Running service dependency integration tests...${NC}"
cd "$PROJECT_ROOT/apps/im-gateway-service"

# Set environment variables for tests
export AUTH_SERVICE_ADDR="localhost:9095"
export USER_SERVICE_ADDR="localhost:9096"
export IM_SERVICE_ADDR="localhost:9094"
export REDIS_ADDR="localhost:6379"
export ETCD_ADDR="localhost:2379"
export KAFKA_ADDR="localhost:9092"

# Create coverage directory
mkdir -p coverage

# Run tests with integration tag and coverage
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test -v -tags=integration ./integration_test/... -timeout 15m \
    -coverprofile=coverage/integration.out \
    -covermode=atomic \
    -coverpkg=./...

TEST_EXIT_CODE=$?

# Generate coverage reports if tests passed
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== All service dependency integration tests passed! ===${NC}"
    
    # Generate HTML coverage report
    echo -e "${YELLOW}Generating HTML coverage report...${NC}"
    go tool cover -html=coverage/integration.out -o coverage/integration.html
    echo -e "${GREEN}Coverage report: coverage/integration.html${NC}"
    
    # Generate coverage summary
    echo -e "${YELLOW}Coverage Summary:${NC}"
    go tool cover -func=coverage/integration.out | tail -n 1
    
    # Generate JSON test results (if go-junit-report is installed)
    if command -v go-junit-report &> /dev/null; then
        echo -e "${YELLOW}Generating JUnit XML report...${NC}"
        go test -v -tags=integration ./integration_test/... -timeout 15m 2>&1 | \
            go-junit-report > coverage/integration-junit.xml
        echo -e "${GREEN}JUnit report: coverage/integration-junit.xml${NC}"
    fi
    
    # Performance metrics
    echo -e "${YELLOW}Performance Metrics:${NC}"
    echo "Test Duration: $(grep -o 'PASS.*' coverage/integration.out | tail -1 || echo 'N/A')"
    echo "Total Tests: $(grep -c '^=== RUN' coverage/integration.out 2>/dev/null || echo 'N/A')"
else
    echo ""
    echo -e "${RED}=== Service dependency integration tests failed ===${NC}"
    echo -e "${YELLOW}Showing service logs:${NC}"
    echo ""
    echo -e "${YELLOW}=== Auth Service Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 auth-service
    echo ""
    echo -e "${YELLOW}=== User Service Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 user-service
    echo ""
    echo -e "${YELLOW}=== IM Service Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 im-service
    echo ""
    echo -e "${YELLOW}=== IM Gateway Service Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 im-gateway-service
fi

exit $TEST_EXIT_CODE
