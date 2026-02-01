#!/bin/bash

# Multi-Region Deployment Automation Script
# This script automates the deployment of the multi-region active-active architecture

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check available memory
    available_memory=$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo "0")
    required_memory=$((16 * 1024 * 1024 * 1024))  # 16GB
    
    if [ "$available_memory" -lt "$required_memory" ]; then
        log_warning "Available Docker memory ($((available_memory / 1024 / 1024 / 1024))GB) is less than recommended (16GB)"
        log_warning "Deployment may be slow or fail. Consider increasing Docker memory limit."
    fi
    
    log_success "Prerequisites check passed"
}

# Phase 1: Deploy Infrastructure
deploy_infrastructure() {
    log_info "Phase 1: Deploying infrastructure services..."
    
    cd "$SCRIPT_DIR"
    
    # Start infrastructure
    log_info "Starting MySQL, Redis, Kafka, etcd..."
    docker compose -f docker-compose.infra.yml up -d
    
    # Wait for services to be ready
    log_info "Waiting for infrastructure to be ready (this may take 2-3 minutes)..."
    sleep 30
    
    # Check MySQL
    log_info "Checking MySQL..."
    for i in {1..30}; do
        if docker exec mysql mysqladmin ping -h localhost -uroot -proot_password &> /dev/null; then
            log_success "MySQL is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "MySQL failed to start"
            return 1
        fi
        sleep 2
    done
    
    # Check Redis
    log_info "Checking Redis..."
    for i in {1..30}; do
        if docker exec redis redis-cli ping &> /dev/null; then
            log_success "Redis is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "Redis failed to start"
            return 1
        fi
        sleep 2
    done
    
    # Check Kafka
    log_info "Checking Kafka..."
    sleep 30  # Kafka needs more time to start
    if docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null; then
        log_success "Kafka is ready"
    else
        log_warning "Kafka may not be fully ready, but continuing..."
    fi
    
    # Check etcd
    log_info "Checking etcd..."
    if docker exec etcd etcdctl endpoint health &> /dev/null; then
        log_success "etcd is ready"
    else
        log_error "etcd failed to start"
        return 1
    fi
    
    log_success "Phase 1 completed: Infrastructure is running"
}

# Phase 2: Deploy Multi-Region Services
deploy_services() {
    log_info "Phase 2: Deploying multi-region services..."
    
    cd "$SCRIPT_DIR"
    
    # Start multi-region services
    log_info "Starting Region A and Region B services..."
    ./start-multi-region.sh start
    
    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    sleep 20
    
    # Verify services
    log_info "Verifying service health..."
    ./start-multi-region.sh test
    
    log_success "Phase 2 completed: Multi-region services are running"
}

# Phase 3: Verify Cross-Region Communication
verify_communication() {
    log_info "Phase 3: Verifying cross-region communication..."
    
    # Test Region A to Region B
    log_info "Testing Region A -> Region B connectivity..."
    if docker exec im-service-region-a ping -c 3 im-service-region-b &> /dev/null; then
        log_success "Region A can reach Region B"
    else
        log_error "Region A cannot reach Region B"
        return 1
    fi
    
    # Test Region B to Region A
    log_info "Testing Region B -> Region A connectivity..."
    if docker exec im-service-region-b ping -c 3 im-service-region-a &> /dev/null; then
        log_success "Region B can reach Region A"
    else
        log_error "Region B cannot reach Region A"
        return 1
    fi
    
    # Check service discovery
    log_info "Checking service discovery in etcd..."
    service_count=$(docker exec etcd etcdctl get /im/services/ --prefix --keys-only | wc -l)
    if [ "$service_count" -gt 0 ]; then
        log_success "Services registered in etcd: $service_count"
    else
        log_warning "No services found in etcd"
    fi
    
    log_success "Phase 3 completed: Cross-region communication verified"
}

# Phase 4: Deploy Observability Stack
deploy_observability() {
    log_info "Phase 4: Deploying observability stack..."
    
    cd "$SCRIPT_DIR"
    
    # Check if observability compose file exists
    if [ ! -f "docker-compose.observability.yml" ]; then
        log_warning "docker-compose.observability.yml not found, skipping observability deployment"
        return 0
    fi
    
    # Start observability services
    log_info "Starting Prometheus, Grafana, Alertmanager..."
    docker compose -f docker-compose.observability.yml up -d
    
    # Wait for services
    sleep 15
    
    # Check Prometheus
    log_info "Checking Prometheus..."
    if curl -s http://localhost:9090/-/healthy &> /dev/null; then
        log_success "Prometheus is ready at http://localhost:9090"
    else
        log_warning "Prometheus may not be ready"
    fi
    
    # Check Grafana
    log_info "Checking Grafana..."
    if curl -s http://localhost:3000/api/health &> /dev/null; then
        log_success "Grafana is ready at http://localhost:3000 (admin/admin)"
    else
        log_warning "Grafana may not be ready"
    fi
    
    log_success "Phase 4 completed: Observability stack deployed"
}

# Phase 5: Run Basic Tests
run_basic_tests() {
    log_info "Phase 5: Running basic functionality tests..."
    
    # Test Region A health
    log_info "Testing Region A health endpoint..."
    if curl -s http://localhost:8184/health | grep -q "healthy"; then
        log_success "Region A is healthy"
    else
        log_error "Region A health check failed"
        return 1
    fi
    
    # Test Region B health
    log_info "Testing Region B health endpoint..."
    if curl -s http://localhost:8284/health | grep -q "healthy"; then
        log_success "Region B is healthy"
    else
        log_error "Region B health check failed"
        return 1
    fi
    
    # Test metrics endpoints
    log_info "Testing metrics endpoints..."
    if curl -s http://localhost:8184/metrics | grep -q "hlc_physical_time"; then
        log_success "Region A metrics are available"
    else
        log_warning "Region A metrics may not be fully available"
    fi
    
    if curl -s http://localhost:8284/metrics | grep -q "hlc_physical_time"; then
        log_success "Region B metrics are available"
    else
        log_warning "Region B metrics may not be fully available"
    fi
    
    log_success "Phase 5 completed: Basic tests passed"
}

# Display deployment summary
show_summary() {
    echo ""
    echo "=========================================="
    echo "  Multi-Region Deployment Complete!"
    echo "=========================================="
    echo ""
    echo "Services:"
    echo "  Region A IM Service:     http://localhost:8184"
    echo "  Region A Gateway:        http://localhost:8182"
    echo "  Region B IM Service:     http://localhost:8284"
    echo "  Region B Gateway:        http://localhost:8282"
    echo ""
    echo "Infrastructure:"
    echo "  MySQL:                   localhost:3307"
    echo "  Redis:                   localhost:6380"
    echo "  Kafka:                   localhost:9093"
    echo "  etcd:                    localhost:2379"
    echo ""
    echo "Monitoring:"
    echo "  Prometheus:              http://localhost:9090"
    echo "  Grafana:                 http://localhost:3000 (admin/admin)"
    echo "  Alertmanager:            http://localhost:9093"
    echo ""
    echo "Next Steps:"
    echo "  1. Import Grafana dashboards from docs/multi-region-demo/"
    echo "  2. Run chaos tests: cd deploy/mvp && ./scripts/chaos-test.sh"
    echo "  3. Run E2E tests: cd tests/e2e/multi-region && ./run-e2e-tests.sh"
    echo "  4. Monitor metrics in Grafana"
    echo "  5. Review logs: ./start-multi-region.sh logs"
    echo ""
    echo "Documentation:"
    echo "  - Deployment Guide:      deploy/docker/DEPLOYMENT_EXECUTION_PLAN.md"
    echo "  - Infrastructure Setup:  deploy/docker/INFRASTRUCTURE_SETUP_GUIDE.md"
    echo "  - Troubleshooting:       docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md"
    echo ""
}

# Cleanup function
cleanup() {
    log_info "Cleaning up multi-region deployment..."
    
    cd "$SCRIPT_DIR"
    
    # Stop services
    ./start-multi-region.sh stop
    
    # Stop observability
    if [ -f "docker-compose.observability.yml" ]; then
        docker compose -f docker-compose.observability.yml down
    fi
    
    # Stop infrastructure
    docker compose -f docker-compose.infra.yml down
    
    log_success "Cleanup completed"
}

# Main deployment function
deploy() {
    log_info "Starting multi-region deployment..."
    echo ""
    
    # Run deployment phases
    check_prerequisites
    echo ""
    
    deploy_infrastructure
    echo ""
    
    deploy_services
    echo ""
    
    verify_communication
    echo ""
    
    deploy_observability
    echo ""
    
    run_basic_tests
    echo ""
    
    show_summary
}

# Command line interface
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    cleanup)
        cleanup
        ;;
    verify)
        verify_communication
        run_basic_tests
        ;;
    summary)
        show_summary
        ;;
    *)
        echo "Usage: $0 {deploy|cleanup|verify|summary}"
        echo ""
        echo "Commands:"
        echo "  deploy   - Deploy complete multi-region environment (default)"
        echo "  cleanup  - Stop and remove all services"
        echo "  verify   - Verify deployment health"
        echo "  summary  - Show deployment summary"
        exit 1
        ;;
esac
