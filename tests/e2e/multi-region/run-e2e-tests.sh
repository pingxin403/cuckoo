#!/bin/bash

# Multi-Region End-to-End Test Runner
# Task 10.1: 端到端多地域功能验证

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
DOCKER_DIR="$PROJECT_ROOT/deploy/docker"
TEST_TIMEOUT="15m"
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=2

# Functions
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

print_header() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
    echo ""
}

check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_success "Docker is installed"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    log_success "Docker Compose is installed"
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    log_success "Go is installed ($(go version))"
    
    # Check if start-multi-region.sh exists
    if [ ! -f "$DOCKER_DIR/start-multi-region.sh" ]; then
        log_error "start-multi-region.sh not found at $DOCKER_DIR"
        exit 1
    fi
    log_success "Multi-region deployment script found"
}

start_infrastructure() {
    print_header "Starting Multi-Region Infrastructure"
    
    cd "$DOCKER_DIR"
    
    log_info "Starting multi-region services..."
    ./start-multi-region.sh start
    
    if [ $? -ne 0 ]; then
        log_error "Failed to start multi-region services"
        exit 1
    fi
    
    log_success "Multi-region services started"
}

wait_for_services() {
    print_header "Waiting for Services to be Ready"
    
    local services=(
        "Region A IM Service:http://localhost:8184/health"
        "Region B IM Service:http://localhost:8284/health"
        "Region A Gateway:http://localhost:8182/health"
        "Region B Gateway:http://localhost:8282/health"
    )
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r service_name service_url <<< "$service_info"
        
        log_info "Checking $service_name..."
        
        local retries=0
        while [ $retries -lt $HEALTH_CHECK_RETRIES ]; do
            if curl -sf "$service_url" > /dev/null 2>&1; then
                log_success "$service_name is ready"
                break
            fi
            
            retries=$((retries + 1))
            if [ $retries -eq $HEALTH_CHECK_RETRIES ]; then
                log_error "$service_name failed to become ready after $((HEALTH_CHECK_RETRIES * HEALTH_CHECK_INTERVAL)) seconds"
                return 1
            fi
            
            sleep $HEALTH_CHECK_INTERVAL
        done
    done
    
    # Additional wait for geo routers to complete initial health checks
    log_info "Waiting for geo routers to complete initial health checks..."
    sleep 5
    
    log_success "All services are ready"
}

run_tests() {
    print_header "Running End-to-End Verification Tests"
    
    cd "$SCRIPT_DIR"
    
    log_info "Test timeout: $TEST_TIMEOUT"
    log_info "Running tests with tag: e2e"
    
    # Set environment variables for tests
    export REGION_A_IM_SERVICE_ADDR="localhost:9194"
    export REGION_A_GATEWAY_ADDR="localhost:8182"
    export REGION_A_REDIS_ADDR="localhost:6379"
    export REGION_A_ETCD_ADDR="localhost:2379"
    
    export REGION_B_IM_SERVICE_ADDR="localhost:9294"
    export REGION_B_GATEWAY_ADDR="localhost:8282"
    export REGION_B_REDIS_ADDR="localhost:6379"
    export REGION_B_ETCD_ADDR="localhost:2379"
    
    export SHARED_ETCD_ADDR="localhost:2379"
    
    # Run tests
    if go test -v -tags=e2e -timeout "$TEST_TIMEOUT" 2>&1 | tee test-output.log; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Some tests failed. Check test-output.log for details."
        return 1
    fi
}

generate_report() {
    print_header "Test Report"
    
    if [ -f "$SCRIPT_DIR/test-output.log" ]; then
        # Count test results
        local total_tests=$(grep -c "=== RUN" "$SCRIPT_DIR/test-output.log" || echo "0")
        local passed_tests=$(grep -c "--- PASS" "$SCRIPT_DIR/test-output.log" || echo "0")
        local failed_tests=$(grep -c "--- FAIL" "$SCRIPT_DIR/test-output.log" || echo "0")
        
        echo "Total Tests: $total_tests"
        echo "Passed: $passed_tests"
        echo "Failed: $failed_tests"
        echo ""
        
        if [ "$failed_tests" -gt 0 ]; then
            log_warning "Failed tests:"
            grep "--- FAIL" "$SCRIPT_DIR/test-output.log" || true
        fi
        
        # Extract test duration
        local duration=$(grep "PASS\|FAIL" "$SCRIPT_DIR/test-output.log" | tail -1 | grep -oP '\d+\.\d+s' || echo "unknown")
        echo "Test Duration: $duration"
    else
        log_warning "Test output log not found"
    fi
}

cleanup() {
    print_header "Cleaning Up"
    
    cd "$DOCKER_DIR"
    
    log_info "Stopping multi-region services..."
    ./start-multi-region.sh stop
    
    if [ $? -eq 0 ]; then
        log_success "Multi-region services stopped"
    else
        log_warning "Failed to stop some services (may already be stopped)"
    fi
}

show_logs() {
    print_header "Service Logs (Last 50 Lines)"
    
    cd "$DOCKER_DIR"
    
    log_info "Region A IM Service logs:"
    docker-compose -f docker-compose.services.yml logs --tail=50 im-service-region-a || true
    
    echo ""
    log_info "Region B IM Service logs:"
    docker-compose -f docker-compose.services.yml logs --tail=50 im-service-region-b || true
    
    echo ""
    log_info "Region A Gateway logs:"
    docker-compose -f docker-compose.services.yml logs --tail=50 im-gateway-service-region-a || true
    
    echo ""
    log_info "Region B Gateway logs:"
    docker-compose -f docker-compose.services.yml logs --tail=50 im-gateway-service-region-b || true
}

usage() {
    cat << EOF
Multi-Region End-to-End Test Runner

Usage: $0 [COMMAND]

Commands:
    run         Run complete test suite (default)
    start       Start infrastructure only
    test        Run tests only (assumes infrastructure is running)
    stop        Stop infrastructure
    logs        Show service logs
    clean       Stop infrastructure and clean up
    help        Show this help message

Examples:
    $0                  # Run complete test suite
    $0 run              # Run complete test suite
    $0 start            # Start infrastructure
    $0 test             # Run tests (infrastructure must be running)
    $0 logs             # Show service logs
    $0 stop             # Stop infrastructure

Environment Variables:
    TEST_TIMEOUT                Default: 15m
    HEALTH_CHECK_RETRIES        Default: 30
    HEALTH_CHECK_INTERVAL       Default: 2

EOF
}

# Main execution
main() {
    local command="${1:-run}"
    
    case "$command" in
        run)
            check_prerequisites
            start_infrastructure
            
            if wait_for_services; then
                if run_tests; then
                    generate_report
                    cleanup
                    exit 0
                else
                    generate_report
                    log_error "Tests failed. Keeping infrastructure running for debugging."
                    log_info "Run '$0 logs' to view service logs"
                    log_info "Run '$0 stop' to stop services when done"
                    exit 1
                fi
            else
                log_error "Services failed to become ready"
                show_logs
                cleanup
                exit 1
            fi
            ;;
        
        start)
            check_prerequisites
            start_infrastructure
            wait_for_services
            log_success "Infrastructure is ready. Run '$0 test' to run tests."
            ;;
        
        test)
            log_info "Running tests (assuming infrastructure is already running)..."
            if run_tests; then
                generate_report
                exit 0
            else
                generate_report
                exit 1
            fi
            ;;
        
        stop)
            cleanup
            ;;
        
        logs)
            show_logs
            ;;
        
        clean)
            cleanup
            log_info "Removing test artifacts..."
            rm -f "$SCRIPT_DIR/test-output.log"
            log_success "Cleanup complete"
            ;;
        
        help|--help|-h)
            usage
            ;;
        
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Trap errors and cleanup
trap 'log_error "Script failed at line $LINENO"' ERR

# Run main function
main "$@"
