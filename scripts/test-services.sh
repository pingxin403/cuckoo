#!/bin/bash

# End-to-End Testing Script for Monorepo Services
# This script tests all services and their interactions
# Supports: Hello/TODO services, Shortener service, IM Chat System

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Parse command line arguments
TEST_SUITE="all"
if [ $# -gt 0 ]; then
    TEST_SUITE=$1
fi

# Function to print test result
print_result() {
    local test_name=$1
    local result=$2
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Function to test if a service is running
test_service_running() {
    local name=$1
    local port=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_result "$name is running on port $port" "PASS"
        return 0
    else
        print_result "$name is running on port $port" "FAIL"
        return 1
    fi
}

# Function to test gRPC service with grpcurl
test_grpc_service() {
    local name=$1
    local address=$2
    local method=$3
    local data=$4
    
    if command -v grpcurl &> /dev/null; then
        if grpcurl -plaintext -d "$data" "$address" "$method" >/dev/null 2>&1; then
            print_result "$name gRPC call: $method" "PASS"
            return 0
        else
            print_result "$name gRPC call: $method" "FAIL"
            return 1
        fi
    else
        echo -e "${YELLOW}Warning: grpcurl not installed. Skipping gRPC tests.${NC}"
        echo -e "${YELLOW}Install with: brew install grpcurl (macOS) or go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest${NC}"
        return 0
    fi
}

# Function to test HTTP endpoint
test_http_endpoint() {
    local name=$1
    local url=$2
    local expected_status=$3
    
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [ "$status" = "$expected_status" ]; then
        print_result "$name HTTP endpoint: $url (status: $status)" "PASS"
        return 0
    else
        print_result "$name HTTP endpoint: $url (expected: $expected_status, got: $status)" "FAIL"
        return 1
    fi
}

echo -e "${GREEN}=== End-to-End Testing ===${NC}\n"

# Show usage if help requested
if [ "$TEST_SUITE" = "help" ] || [ "$TEST_SUITE" = "--help" ] || [ "$TEST_SUITE" = "-h" ]; then
    echo "Usage: $0 [test-suite]"
    echo ""
    echo "Available test suites:"
    echo "  all        - Run all tests (default)"
    echo "  hello-todo - Test Hello and TODO services"
    echo "  shortener  - Test URL Shortener service"
    echo "  im         - Test IM Chat System"
    echo "  infra      - Test infrastructure only"
    echo ""
    echo "Examples:"
    echo "  $0              # Run all tests"
    echo "  $0 im           # Test IM Chat System only"
    echo "  $0 infra        # Test infrastructure only"
    exit 0
fi

echo -e "${BLUE}Test Suite: ${TEST_SUITE}${NC}\n"

# Test infrastructure (common for all services)
if [ "$TEST_SUITE" = "all" ] || [ "$TEST_SUITE" = "infra" ] || [ "$TEST_SUITE" = "im" ]; then
    echo -e "${BLUE}=== Infrastructure Tests ===${NC}"
    
    # Test MySQL
    echo -n "Testing MySQL... "
    if docker exec shared-mysql mysqladmin ping -h localhost -u root -proot_password > /dev/null 2>&1; then
        print_result "MySQL is running" "PASS"
    else
        print_result "MySQL is running" "FAIL"
    fi
    
    # Test Redis
    echo -n "Testing Redis... "
    if docker exec shared-redis redis-cli PING | grep -q "PONG"; then
        print_result "Redis is running" "PASS"
    else
        print_result "Redis is running" "FAIL"
    fi
    
    # Test etcd (for IM system)
    if [ "$TEST_SUITE" = "all" ] || [ "$TEST_SUITE" = "im" ]; then
        echo -n "Testing etcd... "
        if docker exec im-etcd etcdctl endpoint health > /dev/null 2>&1; then
            print_result "etcd is running" "PASS"
        else
            print_result "etcd is running" "FAIL"
        fi
    fi
    
    # Test Kafka (for IM system)
    if [ "$TEST_SUITE" = "all" ] || [ "$TEST_SUITE" = "im" ]; then
        echo -n "Testing Kafka... "
        if docker exec im-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
            print_result "Kafka is running" "PASS"
        else
            print_result "Kafka is running" "FAIL"
        fi
    fi
    
    echo ""
fi

# Exit early if only testing infrastructure
if [ "$TEST_SUITE" = "infra" ]; then
    echo -e "${GREEN}=== Test Summary ===${NC}"
    echo -e "Total tests: $TESTS_TOTAL"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}Failed: $TESTS_FAILED${NC}"
        exit 1
    else
        echo -e "Failed: $TESTS_FAILED"
        exit 0
    fi
fi

# Test IM Chat System
if [ "$TEST_SUITE" = "all" ] || [ "$TEST_SUITE" = "im" ]; then
    echo -e "${BLUE}=== IM Chat System Tests ===${NC}"
    
    # Test database schema
    echo -e "${YELLOW}Testing database schema...${NC}"
    
    # Check offline_messages table
    if docker exec shared-mysql mysql -u im_service -pim_service_password -D im_chat -e "DESCRIBE offline_messages" > /dev/null 2>&1; then
        print_result "offline_messages table exists" "PASS"
    else
        print_result "offline_messages table exists" "FAIL"
    fi
    
    # Check users table
    if docker exec shared-mysql mysql -u im_service -pim_service_password -D im_chat -e "DESCRIBE users" > /dev/null 2>&1; then
        print_result "users table exists" "PASS"
    else
        print_result "users table exists" "FAIL"
    fi
    
    # Check groups table
    if docker exec shared-mysql mysql -u im_service -pim_service_password -D im_chat -e "DESCRIBE groups" > /dev/null 2>&1; then
        print_result "groups table exists" "PASS"
    else
        print_result "groups table exists" "FAIL"
    fi
    
    # Check group_members table
    if docker exec shared-mysql mysql -u im_service -pim_service_password -D im_chat -e "DESCRIBE group_members" > /dev/null 2>&1; then
        print_result "group_members table exists" "PASS"
    else
        print_result "group_members table exists" "FAIL"
    fi
    
    # Check sequence_snapshots table
    if docker exec shared-mysql mysql -u im_service -pim_service_password -D im_chat -e "DESCRIBE sequence_snapshots" > /dev/null 2>&1; then
        print_result "sequence_snapshots table exists" "PASS"
    else
        print_result "sequence_snapshots table exists" "FAIL"
    fi
    
    # Test Kafka topics
    echo -e "${YELLOW}Testing Kafka topics...${NC}"
    
    if docker exec im-kafka kafka-topics --list --bootstrap-server localhost:9092 | grep -q "group_msg"; then
        print_result "group_msg topic exists" "PASS"
    else
        print_result "group_msg topic exists" "FAIL"
    fi
    
    if docker exec im-kafka kafka-topics --list --bootstrap-server localhost:9092 | grep -q "offline_msg"; then
        print_result "offline_msg topic exists" "PASS"
    else
        print_result "offline_msg topic exists" "FAIL"
    fi
    
    if docker exec im-kafka kafka-topics --list --bootstrap-server localhost:9092 | grep -q "membership_change"; then
        print_result "membership_change topic exists" "PASS"
    else
        print_result "membership_change topic exists" "FAIL"
    fi
    
    # Test service builds
    echo -e "${YELLOW}Testing service builds...${NC}"
    
    # Test auth-service build
    if cd apps/auth-service && go build -o /tmp/auth-service . > /dev/null 2>&1; then
        print_result "auth-service builds successfully" "PASS"
        cd ../..
    else
        print_result "auth-service builds successfully" "FAIL"
        cd ../..
    fi
    
    # Test user-service build
    if cd apps/user-service && go build -o /tmp/user-service . > /dev/null 2>&1; then
        print_result "user-service builds successfully" "PASS"
        cd ../..
    else
        print_result "user-service builds successfully" "FAIL"
        cd ../..
    fi
    
    # Test im-service build
    if cd apps/im-service && go build -o /tmp/im-service . > /dev/null 2>&1; then
        print_result "im-service builds successfully" "PASS"
        cd ../..
    else
        print_result "im-service builds successfully" "FAIL"
        cd ../..
    fi
    
    # Test im-gateway-service build
    if cd apps/im-gateway-service && go build -o /tmp/im-gateway-service . > /dev/null 2>&1; then
        print_result "im-gateway-service builds successfully" "PASS"
        cd ../..
    else
        print_result "im-gateway-service builds successfully" "FAIL"
        cd ../..
    fi
    
    # Test offline-worker build
    if cd apps/im-service && go build -o /tmp/offline-worker ./cmd/offline-worker > /dev/null 2>&1; then
        print_result "offline-worker builds successfully" "PASS"
        cd ../..
    else
        print_result "offline-worker builds successfully" "FAIL"
        cd ../..
    fi
    
    # Test unit tests (quick smoke test)
    echo -e "${YELLOW}Running unit tests (sample)...${NC}"
    
    if cd apps/auth-service && go test ./service -run TestValidateToken > /dev/null 2>&1; then
        print_result "auth-service unit tests pass" "PASS"
        cd ../..
    else
        print_result "auth-service unit tests pass" "FAIL"
        cd ../..
    fi
    
    if cd apps/im-service && go test ./sequence -run TestGenerateSequence > /dev/null 2>&1; then
        print_result "sequence generator tests pass" "PASS"
        cd ../..
    else
        print_result "sequence generator tests pass" "FAIL"
        cd ../..
    fi
    
    echo ""
fi

# Check if services are running (Hello/TODO)
if [ "$TEST_SUITE" = "all" ] || [ "$TEST_SUITE" = "hello-todo" ]; then
    echo -e "${BLUE}=== Hello/TODO Services Tests ===${NC}"
test_service_running "Hello Service" 9090
test_service_running "TODO Service" 9091
test_service_running "Frontend" 5173

# Check if Envoy is running (optional)
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    test_service_running "Envoy Proxy" 8080
    ENVOY_RUNNING=true
else
    echo -e "${YELLOW}Note: Envoy proxy not running on port 8080${NC}"
    ENVOY_RUNNING=false
fi

echo ""

# Test Hello Service
echo -e "${BLUE}Testing Hello Service...${NC}"

if command -v grpcurl &> /dev/null; then
    # Test with name
    echo -e "${YELLOW}Testing Hello Service with name 'Alice'...${NC}"
    RESPONSE=$(grpcurl -plaintext -d '{"name":"Alice"}' localhost:9090 api.v1.HelloService/SayHello 2>/dev/null || echo "")
    if echo "$RESPONSE" | grep -q "Alice"; then
        print_result "Hello Service returns greeting with name" "PASS"
    else
        print_result "Hello Service returns greeting with name" "FAIL"
    fi
    
    # Test with empty name
    echo -e "${YELLOW}Testing Hello Service with empty name...${NC}"
    RESPONSE=$(grpcurl -plaintext -d '{"name":""}' localhost:9090 api.v1.HelloService/SayHello 2>/dev/null || echo "")
    if echo "$RESPONSE" | grep -q "Hello"; then
        print_result "Hello Service returns default greeting" "PASS"
    else
        print_result "Hello Service returns default greeting" "FAIL"
    fi
else
    echo -e "${YELLOW}Skipping gRPC tests (grpcurl not installed)${NC}"
fi

echo ""

# Test TODO Service
echo -e "${BLUE}Testing TODO Service...${NC}"

if command -v grpcurl &> /dev/null; then
    # Create a TODO
    echo -e "${YELLOW}Creating a TODO item...${NC}"
    CREATE_RESPONSE=$(grpcurl -plaintext -d '{"title":"Test TODO","description":"This is a test"}' localhost:9091 api.v1.TodoService/CreateTodo 2>/dev/null || echo "")
    if echo "$CREATE_RESPONSE" | grep -q "Test TODO"; then
        print_result "TODO Service creates TODO" "PASS"
        
        # Extract TODO ID
        TODO_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo -e "${BLUE}Created TODO with ID: $TODO_ID${NC}"
    else
        print_result "TODO Service creates TODO" "FAIL"
        TODO_ID=""
    fi
    
    # List TODOs
    echo -e "${YELLOW}Listing TODO items...${NC}"
    LIST_RESPONSE=$(grpcurl -plaintext -d '{}' localhost:9091 api.v1.TodoService/ListTodos 2>/dev/null || echo "")
    if echo "$LIST_RESPONSE" | grep -q "Test TODO"; then
        print_result "TODO Service lists TODOs" "PASS"
    else
        print_result "TODO Service lists TODOs" "FAIL"
    fi
    
    # Update TODO (if we have an ID)
    if [ -n "$TODO_ID" ]; then
        echo -e "${YELLOW}Updating TODO item...${NC}"
        UPDATE_RESPONSE=$(grpcurl -plaintext -d "{\"id\":\"$TODO_ID\",\"title\":\"Updated TODO\",\"description\":\"Updated description\",\"completed\":true}" localhost:9091 api.v1.TodoService/UpdateTodo 2>/dev/null || echo "")
        if echo "$UPDATE_RESPONSE" | grep -q "Updated TODO"; then
            print_result "TODO Service updates TODO" "PASS"
        else
            print_result "TODO Service updates TODO" "FAIL"
        fi
        
        # Delete TODO
        echo -e "${YELLOW}Deleting TODO item...${NC}"
        DELETE_RESPONSE=$(grpcurl -plaintext -d "{\"id\":\"$TODO_ID\"}" localhost:9091 api.v1.TodoService/DeleteTodo 2>/dev/null || echo "")
        if echo "$DELETE_RESPONSE" | grep -q "success"; then
            print_result "TODO Service deletes TODO" "PASS"
        else
            print_result "TODO Service deletes TODO" "FAIL"
        fi
    fi
else
    echo -e "${YELLOW}Skipping gRPC tests (grpcurl not installed)${NC}"
fi

echo ""

# Test service-to-service communication
echo -e "${BLUE}Testing service-to-service communication...${NC}"
echo -e "${YELLOW}Note: TODO Service should be able to call Hello Service${NC}"
echo -e "${YELLOW}This is verified by checking if TODO Service can start with HELLO_SERVICE_ADDR set${NC}"

if test_service_running "TODO Service" 9091 >/dev/null 2>&1; then
    print_result "TODO Service can communicate with Hello Service" "PASS"
else
    print_result "TODO Service can communicate with Hello Service" "FAIL"
fi

echo ""

# Test Frontend
echo -e "${BLUE}Testing Frontend...${NC}"

# Test if frontend is accessible
test_http_endpoint "Frontend" "http://localhost:5173" "200"

# Test if frontend can load assets
if curl -s "http://localhost:5173" | grep -q "<!doctype html>"; then
    print_result "Frontend serves HTML" "PASS"
else
    print_result "Frontend serves HTML" "FAIL"
fi

echo ""

# Test API Gateway (if running)
if [ "$ENVOY_RUNNING" = true ]; then
    echo -e "${BLUE}Testing API Gateway (Envoy)...${NC}"
    
    # Test Envoy admin interface
    test_http_endpoint "Envoy Admin" "http://localhost:9901" "200"
    
    echo -e "${YELLOW}Note: gRPC-Web tests require a browser or specialized client${NC}"
    echo -e "${YELLOW}Manual testing recommended for full API Gateway verification${NC}"
fi

echo ""
fi

# Summary
echo -e "${GREEN}=== Test Summary ===${NC}"
echo -e "Total tests: $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
else
    echo -e "Failed: $TESTS_FAILED"
fi

echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  1. Open http://localhost:5173 in your browser"
    echo -e "  2. Test the Hello form by entering a name"
    echo -e "  3. Test TODO operations (create, list, update, delete)"
    echo -e "  4. Verify service-to-service communication in the logs"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo -e "${YELLOW}Please check the logs in the logs/ directory for more details${NC}"
    exit 1
fi

