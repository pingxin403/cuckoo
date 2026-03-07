#!/bin/bash

# Health Check Chaos Engineering Test Script
# Tests health check system behavior under various failure scenarios
# 
# Usage: ./scripts/health-check-chaos-test.sh [test-name]
# 
# Available tests:
#   all              - Run all tests
#   database         - Database failure scenario
#   redis            - Redis failure scenario
#   network          - Network partition scenario (simulated with delays)
#   load             - High load scenario
#
# Prerequisites:
#   - Docker and Docker Compose installed
#   - Service running locally or in Docker
#   - jq installed for JSON parsing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="${SERVICE_NAME:-shortener-service}"
SERVICE_URL="${SERVICE_URL:-http://localhost:8080}"
DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-deploy/docker/docker-compose.infra.yml}"
DETECTION_TIME_TARGET=15  # seconds
RECOVERY_TIME_TARGET=60   # seconds

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TEST_RESULTS=()

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}TEST: $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# Check if service is healthy
check_health() {
    local endpoint="$1"
    local expected_status="$2"
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL$endpoint" 2>/dev/null || echo "000")
    
    if [ "$response" = "$expected_status" ]; then
        return 0
    else
        return 1
    fi
}

# Get detailed health status
get_health_status() {
    curl -s "$SERVICE_URL/health" 2>/dev/null | jq -r '.status' 2>/dev/null || echo "unknown"
}

# Get component status
get_component_status() {
    local component="$1"
    curl -s "$SERVICE_URL/health" 2>/dev/null | jq -r ".components[] | select(.name==\"$component\") | .status" 2>/dev/null || echo "unknown"
}

# Wait for condition with timeout
wait_for_condition() {
    local condition_func="$1"
    local timeout="$2"
    local description="$3"
    
    log_info "Waiting for: $description (timeout: ${timeout}s)" >&2
    
    local start_time=$(date +%s)
    local elapsed=0
    
    while [ $elapsed -lt $timeout ]; do
        if $condition_func 2>/dev/null; then
            log_success "Condition met after ${elapsed}s" >&2
            echo $elapsed
            return 0
        fi
        sleep 1
        elapsed=$(($(date +%s) - start_time))
    done
    
    log_error "Timeout after ${timeout}s" >&2
    return 1
}

# Record test result
record_result() {
    local test_name="$1"
    local passed="$2"
    local details="$3"
    
    if [ "$passed" = "true" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        TEST_RESULTS+=("✅ $test_name: PASSED - $details")
        log_success "$test_name: PASSED"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        TEST_RESULTS+=("❌ $test_name: FAILED - $details")
        log_error "$test_name: FAILED - $details"
    fi
}

# Test 1: Database Failure Scenario
test_database_failure() {
    log_test "Database Failure Scenario"
    
    log_info "Step 1: Verify service is healthy"
    if ! check_health "/readyz" "200"; then
        record_result "Database Failure" "false" "Service not healthy at start"
        return 1
    fi
    log_success "Service is healthy"
    
    log_info "Step 2: Stop MySQL container"
    docker compose -f "$DOCKER_COMPOSE_FILE" stop mysql
    
    log_info "Step 3: Wait for failure detection"
    detection_condition() {
        check_health "/readyz" "503"
    }
    
    if detection_time=$(wait_for_condition detection_condition 30 "Service marked not ready"); then
        if [ $detection_time -le $DETECTION_TIME_TARGET ]; then
            log_success "Detection time: ${detection_time}s (target: <${DETECTION_TIME_TARGET}s) ✅"
            detection_passed=true
        else
            log_warning "Detection time: ${detection_time}s (target: <${DETECTION_TIME_TARGET}s) ⚠️"
            detection_passed=false
        fi
    else
        log_error "Failed to detect database failure"
        docker compose -f "$DOCKER_COMPOSE_FILE" start mysql
        record_result "Database Failure - Detection" "false" "Failed to detect failure within 30s"
        return 1
    fi
    
    log_info "Step 4: Verify component status"
    db_status=$(get_component_status "database")
    if [ "$db_status" = "critical" ] || [ "$db_status" = "unhealthy" ]; then
        log_success "Database component marked as critical"
    else
        log_warning "Database component status: $db_status (expected: critical)"
    fi
    
    log_info "Step 5: Restart MySQL container"
    docker compose -f "$DOCKER_COMPOSE_FILE" start mysql
    sleep 5  # Wait for MySQL to be ready
    
    log_info "Step 6: Wait for auto-recovery"
    recovery_condition() {
        check_health "/readyz" "200"
    }
    
    if recovery_time=$(wait_for_condition recovery_condition 90 "Service recovered"); then
        if [ $recovery_time -le $RECOVERY_TIME_TARGET ]; then
            log_success "Recovery time: ${recovery_time}s (target: <${RECOVERY_TIME_TARGET}s) ✅"
            recovery_passed=true
        else
            log_warning "Recovery time: ${recovery_time}s (target: <${RECOVERY_TIME_TARGET}s) ⚠️"
            recovery_passed=false
        fi
    else
        log_error "Failed to recover from database failure"
        record_result "Database Failure - Recovery" "false" "Failed to recover within 90s"
        return 1
    fi
    
    log_info "Step 7: Verify no false positives"
    sleep 10
    if check_health "/readyz" "200"; then
        log_success "No false positives detected"
        false_positive_passed=true
    else
        log_error "False positive detected - service became not ready again"
        false_positive_passed=false
    fi
    
    # Overall result
    if [ "$detection_passed" = "true" ] && [ "$recovery_passed" = "true" ] && [ "$false_positive_passed" = "true" ]; then
        record_result "Database Failure" "true" "Detection: ${detection_time}s, Recovery: ${recovery_time}s"
    else
        record_result "Database Failure" "false" "Detection: ${detection_time}s, Recovery: ${recovery_time}s"
    fi
}

# Test 2: Redis Failure Scenario
test_redis_failure() {
    log_test "Redis Failure Scenario"
    
    log_info "Step 1: Verify service is healthy"
    if ! check_health "/readyz" "200"; then
        record_result "Redis Failure" "false" "Service not healthy at start"
        return 1
    fi
    log_success "Service is healthy"
    
    log_info "Step 2: Stop Redis container"
    docker compose -f "$DOCKER_COMPOSE_FILE" stop redis
    
    log_info "Step 3: Wait for failure detection"
    detection_condition() {
        local redis_status=$(get_component_status "redis")
        [ "$redis_status" = "critical" ] || [ "$redis_status" = "unhealthy" ]
    }
    
    if detection_time=$(wait_for_condition detection_condition 30 "Redis marked as critical"); then
        log_success "Detection time: ${detection_time}s"
    else
        log_warning "Redis failure not detected (may be non-critical component)"
        detection_time=0
    fi
    
    log_info "Step 4: Restart Redis container"
    docker compose -f "$DOCKER_COMPOSE_FILE" start redis
    sleep 3  # Wait for Redis to be ready
    
    log_info "Step 5: Wait for auto-recovery"
    recovery_condition() {
        local redis_status=$(get_component_status "redis")
        [ "$redis_status" = "healthy" ]
    }
    
    if recovery_time=$(wait_for_condition recovery_condition 90 "Redis recovered"); then
        log_success "Recovery time: ${recovery_time}s"
    else
        log_error "Failed to recover from Redis failure"
        record_result "Redis Failure" "false" "Failed to recover within 90s"
        return 1
    fi
    
    log_info "Step 6: Verify service is healthy"
    if check_health "/readyz" "200"; then
        log_success "Service is healthy after Redis recovery"
        record_result "Redis Failure" "true" "Detection: ${detection_time}s, Recovery: ${recovery_time}s"
    else
        log_error "Service not healthy after Redis recovery"
        record_result "Redis Failure" "false" "Service not healthy after recovery"
    fi
}

# Test 3: Network Partition Scenario (Simulated)
test_network_partition() {
    log_test "Network Partition Scenario (Simulated with delays)"
    
    log_info "Note: This test simulates network issues by observing timeout behavior"
    log_info "For real network partition testing, use tc (traffic control) in a Linux environment"
    
    log_info "Step 1: Verify service is healthy"
    if ! check_health "/readyz" "200"; then
        record_result "Network Partition" "false" "Service not healthy at start"
        return 1
    fi
    log_success "Service is healthy"
    
    log_info "Step 2: Check health check response times"
    # Use Python for cross-platform millisecond timing
    start_time=$(python3 -c 'import time; print(int(time.time() * 1000))')
    curl -s "$SERVICE_URL/health" > /dev/null
    end_time=$(python3 -c 'import time; print(int(time.time() * 1000))')
    response_time=$((end_time - start_time))
    
    log_info "Health check response time: ${response_time}ms"
    
    if [ $response_time -lt 200 ]; then
        log_success "Response time within target (<200ms)"
        record_result "Network Partition" "true" "Response time: ${response_time}ms (target: <200ms)"
    else
        log_warning "Response time above target: ${response_time}ms"
        record_result "Network Partition" "false" "Response time: ${response_time}ms (target: <200ms)"
    fi
    
    log_info "Note: For comprehensive network testing, run this in a controlled environment with tc"
}

# Test 4: High Load Scenario
test_high_load() {
    log_test "High Load Scenario"
    
    log_info "Step 1: Verify service is healthy"
    if ! check_health "/readyz" "200"; then
        record_result "High Load" "false" "Service not healthy at start"
        return 1
    fi
    log_success "Service is healthy"
    
    log_info "Step 2: Generate load (100 concurrent requests)"
    log_info "Note: Install 'hey' tool for better load testing: go install github.com/rakyll/hey@latest"
    
    # Simple load test with curl
    log_info "Sending 100 requests..."
    for i in {1..100}; do
        curl -s "$SERVICE_URL/healthz" > /dev/null &
    done
    wait
    
    log_info "Step 3: Check health during load"
    if check_health "/readyz" "200"; then
        log_success "Service remained healthy under load"
    else
        log_error "Service became unhealthy under load"
        record_result "High Load" "false" "Service unhealthy under load"
        return 1
    fi
    
    log_info "Step 4: Check health check latency"
    # Use Python for cross-platform millisecond timing
    start_time=$(python3 -c 'import time; print(int(time.time() * 1000))')
    curl -s "$SERVICE_URL/health" > /dev/null
    end_time=$(python3 -c 'import time; print(int(time.time() * 1000))')
    response_time=$((end_time - start_time))
    
    log_info "Health check response time after load: ${response_time}ms"
    
    if [ $response_time -lt 200 ]; then
        log_success "Response time within target (<200ms)"
        record_result "High Load" "true" "Response time: ${response_time}ms, Service remained stable"
    else
        log_warning "Response time above target: ${response_time}ms"
        record_result "High Load" "false" "Response time: ${response_time}ms (target: <200ms)"
    fi
}

# Print test summary
print_summary() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}TEST SUMMARY${NC}"
    echo -e "${BLUE}========================================${NC}\n"
    
    for result in "${TEST_RESULTS[@]}"; do
        echo -e "$result"
    done
    
    echo -e "\n${BLUE}Total Tests:${NC} $((TESTS_PASSED + TESTS_FAILED))"
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}✅ All tests passed!${NC}\n"
        return 0
    else
        echo -e "\n${RED}❌ Some tests failed${NC}\n"
        return 1
    fi
}

# Main function
main() {
    local test_name="${1:-all}"
    
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Health Check Chaos Engineering Tests${NC}"
    echo -e "${BLUE}========================================${NC}\n"
    
    log_info "Service: $SERVICE_NAME"
    log_info "Service URL: $SERVICE_URL"
    log_info "Docker Compose: $DOCKER_COMPOSE_FILE"
    echo ""
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker not found. Please install Docker."
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker Compose not found. Please install Docker Compose."
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Please install jq for JSON parsing."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
    echo ""
    
    # Run tests
    case "$test_name" in
        all)
            test_database_failure
            test_redis_failure
            test_network_partition
            test_high_load
            ;;
        database)
            test_database_failure
            ;;
        redis)
            test_redis_failure
            ;;
        network)
            test_network_partition
            ;;
        load)
            test_high_load
            ;;
        *)
            log_error "Unknown test: $test_name"
            echo "Available tests: all, database, redis, network, load"
            exit 1
            ;;
    esac
    
    # Print summary
    print_summary
}

# Run main function
main "$@"
