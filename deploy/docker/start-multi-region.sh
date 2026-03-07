#!/bin/bash

# Multi-Region Deployment Quick Start Script
# This script helps you quickly start and test the multi-region deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a service is healthy
check_service_health() {
    local service_name=$1
    local health_url=$2
    local max_attempts=30
    local attempt=1

    print_info "Checking health of $service_name..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$health_url" > /dev/null 2>&1; then
            print_info "$service_name is healthy!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to become healthy after $max_attempts attempts"
    return 1
}

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [COMMAND]

Commands:
    start       Start all multi-region services
    stop        Stop all multi-region services
    restart     Restart all multi-region services
    status      Show status of all services
    logs        Show logs from all multi-region services
    test        Run basic health checks
    clean       Stop and remove all containers, networks, and volumes
    help        Show this help message

Examples:
    $0 start        # Start all services
    $0 logs         # View logs
    $0 test         # Run health checks
    $0 stop         # Stop all services

EOF
}

# Function to start services
start_services() {
    print_info "Starting infrastructure services..."
    docker compose -f "$SCRIPT_DIR/docker-compose.infra.yml" up -d
    
    print_info "Waiting for infrastructure to be ready..."
    sleep 10
    
    print_info "Starting multi-region services..."
    docker compose -f "$SCRIPT_DIR/docker-compose.infra.yml" \
                   -f "$SCRIPT_DIR/docker-compose.services.yml" up -d \
                   im-service-region-a \
                   im-gateway-service-region-a \
                   im-service-region-b \
                   im-gateway-service-region-b
    
    print_info "Multi-region services started!"
    print_info "Waiting for services to be ready..."
    sleep 5
    
    # Check health of all services
    check_service_health "Region A IM Service" "http://localhost:8184/health"
    check_service_health "Region A Gateway" "http://localhost:8182/health" || true
    check_service_health "Region B IM Service" "http://localhost:8284/health"
    check_service_health "Region B Gateway" "http://localhost:8282/health" || true
    
    print_info ""
    print_info "=========================================="
    print_info "Multi-Region Deployment Started!"
    print_info "=========================================="
    print_info ""
    print_info "Region A Services:"
    print_info "  - IM Service:      http://localhost:8184 (gRPC: 9194)"
    print_info "  - Gateway Service: http://localhost:8182 (gRPC: 9197)"
    print_info ""
    print_info "Region B Services:"
    print_info "  - IM Service:      http://localhost:8284 (gRPC: 9294)"
    print_info "  - Gateway Service: http://localhost:8282 (gRPC: 9297)"
    print_info ""
    print_info "Next steps:"
    print_info "  1. Run health checks: $0 test"
    print_info "  2. View logs: $0 logs"
    print_info "  3. Check status: $0 status"
    print_info ""
}

# Function to stop services
stop_services() {
    print_info "Stopping multi-region services..."
    docker compose -f "$SCRIPT_DIR/docker-compose.services.yml" stop \
                   im-service-region-a \
                   im-gateway-service-region-a \
                   im-service-region-b \
                   im-gateway-service-region-b
    
    print_info "Multi-region services stopped!"
}

# Function to restart services
restart_services() {
    print_info "Restarting multi-region services..."
    stop_services
    sleep 2
    start_services
}

# Function to show status
show_status() {
    print_info "Multi-Region Services Status:"
    print_info ""
    
    docker compose -f "$SCRIPT_DIR/docker-compose.services.yml" ps \
                   im-service-region-a \
                   im-gateway-service-region-a \
                   im-service-region-b \
                   im-gateway-service-region-b
}

# Function to show logs
show_logs() {
    print_info "Showing logs from multi-region services..."
    print_info "Press Ctrl+C to exit"
    print_info ""
    
    docker compose -f "$SCRIPT_DIR/docker-compose.services.yml" logs -f \
                   im-service-region-a \
                   im-gateway-service-region-a \
                   im-service-region-b \
                   im-gateway-service-region-b
}

# Function to run tests
run_tests() {
    print_info "Running health checks..."
    print_info ""
    
    # Test Region A
    print_info "Testing Region A..."
    if curl -sf http://localhost:8184/health > /dev/null; then
        print_info "✓ Region A IM Service is healthy"
    else
        print_error "✗ Region A IM Service is not responding"
    fi
    
    if curl -sf http://localhost:8182/health > /dev/null 2>&1; then
        print_info "✓ Region A Gateway is healthy"
    else
        print_warn "✗ Region A Gateway is not responding (may not have /health endpoint)"
    fi
    
    # Test Region B
    print_info ""
    print_info "Testing Region B..."
    if curl -sf http://localhost:8284/health > /dev/null; then
        print_info "✓ Region B IM Service is healthy"
    else
        print_error "✗ Region B IM Service is not responding"
    fi
    
    if curl -sf http://localhost:8282/health > /dev/null 2>&1; then
        print_info "✓ Region B Gateway is healthy"
    else
        print_warn "✗ Region B Gateway is not responding (may not have /health endpoint)"
    fi
    
    # Test cross-region connectivity
    print_info ""
    print_info "Testing cross-region connectivity..."
    if docker exec im-service-region-a ping -c 1 im-service-region-b > /dev/null 2>&1; then
        print_info "✓ Region A can reach Region B"
    else
        print_error "✗ Region A cannot reach Region B"
    fi
    
    if docker exec im-service-region-b ping -c 1 im-service-region-a > /dev/null 2>&1; then
        print_info "✓ Region B can reach Region A"
    else
        print_error "✗ Region B cannot reach Region A"
    fi
    
    # Test etcd service discovery
    print_info ""
    print_info "Testing service discovery..."
    if docker exec etcd etcdctl get /im/services/ --prefix > /dev/null 2>&1; then
        print_info "✓ etcd is accessible"
        service_count=$(docker exec etcd etcdctl get /im/services/ --prefix --keys-only 2>/dev/null | wc -l)
        print_info "  Found $service_count registered services"
    else
        print_error "✗ etcd is not accessible"
    fi
    
    print_info ""
    print_info "Health check complete!"
}

# Function to clean up
clean_up() {
    print_warn "This will stop and remove all containers, networks, and volumes."
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Cleaning up..."
        docker compose -f "$SCRIPT_DIR/docker-compose.infra.yml" \
                       -f "$SCRIPT_DIR/docker-compose.services.yml" down -v
        print_info "Cleanup complete!"
    else
        print_info "Cleanup cancelled."
    fi
}

# Main script logic
case "${1:-}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    test)
        run_tests
        ;;
    clean)
        clean_up
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        print_error "Unknown command: ${1:-}"
        echo
        usage
        exit 1
        ;;
esac
