#!/bin/bash

# Integration Test Runner for IM Service
# This script sets up the test environment, runs integration tests, and cleans up

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo -e "${GREEN}=== IM Service Integration Tests ===${NC}"
echo ""

# Detect docker compose command
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif command -v docker &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    echo -e "${RED}Error: Neither 'docker-compose' nor 'docker compose' found${NC}"
    exit 1
fi

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}Cleaning up test environment...${NC}"
    cd "$SCRIPT_DIR"
    $DOCKER_COMPOSE -f docker-compose.test.yml down -v
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Trap EXIT to ensure cleanup
trap cleanup EXIT

# Step 1: Start infrastructure services
echo -e "${YELLOW}Step 1: Starting infrastructure services...${NC}"
cd "$SCRIPT_DIR"
$DOCKER_COMPOSE -f docker-compose.test.yml up -d mysql redis etcd zookeeper kafka

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
for i in {1..60}; do
    if $DOCKER_COMPOSE -f docker-compose.test.yml ps | grep -q "unhealthy"; then
        echo -n "."
        sleep 2
    else
        echo ""
        echo -e "${GREEN}All infrastructure services are healthy${NC}"
        break
    fi
    
    if [ $i -eq 60 ]; then
        echo -e "${RED}Timeout waiting for services to be healthy${NC}"
        $DOCKER_COMPOSE -f docker-compose.test.yml ps
        $DOCKER_COMPOSE -f docker-compose.test.yml logs
        exit 1
    fi
done

# Step 2: Create Kafka topics
echo -e "${YELLOW}Step 2: Creating Kafka topics...${NC}"
$DOCKER_COMPOSE -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic group_msg \
    --partitions 3 \
    --replication-factor 1

$DOCKER_COMPOSE -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic offline_msg \
    --partitions 3 \
    --replication-factor 1

$DOCKER_COMPOSE -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic membership_change \
    --partitions 3 \
    --replication-factor 1

echo -e "${GREEN}Kafka topics created${NC}"

# Step 3: Build and start IM Service
echo -e "${YELLOW}Step 3: Building and starting IM Service...${NC}"
cd "$PROJECT_ROOT"
$DOCKER_COMPOSE -f apps/im-service/integration_test/docker-compose.test.yml up -d im-service

# Wait for IM Service to be healthy
echo -e "${YELLOW}Waiting for IM Service to be ready...${NC}"
for i in {1..30}; do
    if $DOCKER_COMPOSE -f apps/im-service/integration_test/docker-compose.test.yml ps im-service | grep -q "healthy"; then
        echo -e "${GREEN}IM Service is ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
    
    if [ $i -eq 30 ]; then
        echo -e "${RED}Timeout waiting for IM Service${NC}"
        $DOCKER_COMPOSE -f apps/im-service/integration_test/docker-compose.test.yml logs im-service
        exit 1
    fi
done

# Step 4: Run integration tests
echo -e "${YELLOW}Step 4: Running integration tests...${NC}"
cd "$PROJECT_ROOT/apps/im-service"

# Set environment variables for tests
export IM_SERVICE_ADDR="localhost:9094"
export MYSQL_ADDR="root:password@tcp(localhost:3306)/im_chat"
export REDIS_ADDR="localhost:6379"
export ETCD_ADDR="localhost:2379"
export KAFKA_ADDR="localhost:9092"

# Create coverage directory
mkdir -p coverage

# Run tests with integration tag and coverage
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test -v -tags=integration ./integration_test/... -timeout 10m \
    -coverprofile=coverage/integration.out \
    -covermode=atomic \
    -coverpkg=./...

TEST_EXIT_CODE=$?

# Generate coverage reports if tests passed
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== All integration tests passed! ===${NC}"
    
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
        go test -v -tags=integration ./integration_test/... -timeout 10m 2>&1 | \
            go-junit-report > coverage/integration-junit.xml
        echo -e "${GREEN}JUnit report: coverage/integration-junit.xml${NC}"
    fi
    
    # Performance metrics
    echo -e "${YELLOW}Performance Metrics:${NC}"
    echo "Test Duration: $(grep -o 'PASS.*' coverage/integration.out | tail -1 || echo 'N/A')"
    echo "Total Tests: $(grep -c '^=== RUN' coverage/integration.out 2>/dev/null || echo 'N/A')"
else
    echo ""
    echo -e "${RED}=== Integration tests failed ===${NC}"
    echo -e "${YELLOW}Showing service logs:${NC}"
    $DOCKER_COMPOSE -f integration_test/docker-compose.test.yml logs im-service
fi

exit $TEST_EXIT_CODE

