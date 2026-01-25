#!/bin/bash

# Infrastructure Integration Test Runner for IM Service
# This script runs infrastructure failover and resilience tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo -e "${GREEN}=== IM Service Infrastructure Tests ===${NC}"
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

# Step 1: Start infrastructure services
echo -e "${YELLOW}Step 1: Starting infrastructure services...${NC}"
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.test.yml up -d mysql redis etcd zookeeper kafka

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
for i in {1..60}; do
    if docker-compose -f docker-compose.test.yml ps | grep -q "unhealthy"; then
        echo -n "."
        sleep 2
    else
        echo ""
        echo -e "${GREEN}All infrastructure services are healthy${NC}"
        break
    fi
    
    if [ $i -eq 60 ]; then
        echo -e "${RED}Timeout waiting for services to be healthy${NC}"
        docker-compose -f docker-compose.test.yml ps
        docker-compose -f docker-compose.test.yml logs
        exit 1
    fi
done

# Step 2: Create Kafka topics
echo -e "${YELLOW}Step 2: Creating Kafka topics...${NC}"
docker-compose -f docker-compose.test.yml exec -T kafka kafka-topics \
    --create --if-not-exists \
    --bootstrap-server localhost:9092 \
    --topic test-topic \
    --partitions 3 \
    --replication-factor 1 || true

echo -e "${GREEN}Kafka topics created${NC}"

# Step 3: Set environment variables
echo -e "${YELLOW}Step 3: Setting environment variables...${NC}"
cd "$PROJECT_ROOT/apps/im-service"

export MYSQL_ADDR="root:password@tcp(localhost:3306)/im_chat"
export REDIS_ADDR="localhost:6379"
export ETCD_ADDR="localhost:2379"
export KAFKA_ADDR="localhost:9092"

echo -e "${GREEN}Environment variables set${NC}"

# Step 4: Run infrastructure tests
echo -e "${YELLOW}Step 4: Running infrastructure tests...${NC}"
echo ""

# Create coverage directory
mkdir -p coverage

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a single test
run_test() {
    local test_name=$1
    local test_pattern=$2
    
    echo -e "${BLUE}Running: $test_name${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if go test -v -tags=integration ./integration_test/... -run "$test_pattern" -timeout 5m 2>&1 | tee /tmp/test_output.log; then
        echo -e "${GREEN}✓ $test_name PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ $test_name FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        cat /tmp/test_output.log
    fi
    echo ""
}

# Run individual infrastructure tests
run_test "Etcd Cluster Failover" "TestEtcdClusterFailover"
run_test "Kafka Broker Failover" "TestKafkaBrokerFailover"
run_test "Redis Failover" "TestRedisFailover"
run_test "MySQL Connection Pooling" "TestMySQLConnectionPooling"
run_test "Network Partition Scenario" "TestNetworkPartitionScenario"

# Step 5: Run all infrastructure tests with coverage
echo -e "${YELLOW}Step 5: Running all infrastructure tests with coverage...${NC}"
go test -v -tags=integration ./integration_test/... \
    -run "TestEtcd|TestKafka|TestRedis|TestMySQL|TestNetwork" \
    -timeout 15m \
    -coverprofile=coverage/infrastructure.out \
    -covermode=atomic \
    -coverpkg=./...

TEST_EXIT_CODE=$?

# Step 6: Generate reports
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== All infrastructure tests passed! ===${NC}"
    
    # Generate HTML coverage report
    echo -e "${YELLOW}Generating HTML coverage report...${NC}"
    go tool cover -html=coverage/infrastructure.out -o coverage/infrastructure.html
    echo -e "${GREEN}Coverage report: coverage/infrastructure.html${NC}"
    
    # Generate coverage summary
    echo -e "${YELLOW}Coverage Summary:${NC}"
    go tool cover -func=coverage/infrastructure.out | tail -n 1
    
    # Generate JUnit XML report (if go-junit-report is installed)
    if command -v go-junit-report &> /dev/null; then
        echo -e "${YELLOW}Generating JUnit XML report...${NC}"
        go test -v -tags=integration ./integration_test/... \
            -run "TestEtcd|TestKafka|TestRedis|TestMySQL|TestNetwork" \
            -timeout 15m 2>&1 | \
            go-junit-report > coverage/infrastructure-junit.xml
        echo -e "${GREEN}JUnit report: coverage/infrastructure-junit.xml${NC}"
    fi
else
    echo ""
    echo -e "${RED}=== Infrastructure tests failed ===${NC}"
    echo -e "${YELLOW}Showing service logs:${NC}"
    echo ""
    echo -e "${YELLOW}=== MySQL Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 mysql
    echo ""
    echo -e "${YELLOW}=== Redis Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 redis
    echo ""
    echo -e "${YELLOW}=== Etcd Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 etcd
    echo ""
    echo -e "${YELLOW}=== Kafka Logs ===${NC}"
    docker-compose -f integration_test/docker-compose.test.yml logs --tail=50 kafka
fi

# Step 7: Print test summary
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Total Tests:  $TOTAL_TESTS"
echo -e "${GREEN}Passed:       $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}Failed:       $FAILED_TESTS${NC}"
else
    echo -e "Failed:       $FAILED_TESTS"
fi
echo ""

# Step 8: Infrastructure health check
echo -e "${BLUE}=== Infrastructure Health Check ===${NC}"

# Check MySQL
if docker-compose -f integration_test/docker-compose.test.yml exec -T mysql mysqladmin ping -h localhost -u root -ppassword &> /dev/null; then
    echo -e "${GREEN}✓ MySQL is healthy${NC}"
else
    echo -e "${RED}✗ MySQL is unhealthy${NC}"
fi

# Check Redis
if docker-compose -f integration_test/docker-compose.test.yml exec -T redis redis-cli ping &> /dev/null; then
    echo -e "${GREEN}✓ Redis is healthy${NC}"
else
    echo -e "${RED}✗ Redis is unhealthy${NC}"
fi

# Check Etcd
if docker-compose -f integration_test/docker-compose.test.yml exec -T etcd etcdctl endpoint health &> /dev/null; then
    echo -e "${GREEN}✓ Etcd is healthy${NC}"
else
    echo -e "${RED}✗ Etcd is unhealthy${NC}"
fi

# Check Kafka
if docker-compose -f integration_test/docker-compose.test.yml exec -T kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null; then
    echo -e "${GREEN}✓ Kafka is healthy${NC}"
else
    echo -e "${RED}✗ Kafka is unhealthy${NC}"
fi

echo ""

exit $TEST_EXIT_CODE
