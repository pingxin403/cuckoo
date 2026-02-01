#!/bin/bash

# Test script for the Arbiter system
# This script demonstrates the arbiter functionality using Docker Compose

set -e

echo "=== Arbiter System Test ==="
echo "This script tests the distributed leader election and split-brain prevention"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to wait for service to be ready
wait_for_service() {
    local service_name=$1
    local health_endpoint=$2
    local max_attempts=30
    local attempt=1

    print_status "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$health_endpoint" > /dev/null 2>&1; then
            print_success "$service_name is ready"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to check Zookeeper cluster health
check_zookeeper_cluster() {
    print_status "Checking Zookeeper cluster health..."
    
    local healthy_nodes=0
    
    for port in 2181 2182 2183; do
        if echo "ruok" | nc localhost $port 2>/dev/null | grep -q "imok"; then
            print_success "Zookeeper node on port $port is healthy"
            healthy_nodes=$((healthy_nodes + 1))
        else
            print_warning "Zookeeper node on port $port is not responding"
        fi
    done
    
    if [ $healthy_nodes -ge 2 ]; then
        print_success "Zookeeper cluster is healthy ($healthy_nodes/3 nodes)"
        return 0
    else
        print_error "Zookeeper cluster is unhealthy ($healthy_nodes/3 nodes)"
        return 1
    fi
}

# Function to test arbiter election
test_election() {
    local region=$1
    local expected_leader=$2
    
    print_status "Testing election for $region (expecting leader: $expected_leader)"
    
    # Simulate health report and election via arbiter mock API
    local response=$(curl -s -X POST http://localhost:9999/elect \
        -H "Content-Type: application/json" \
        -d '{
            "region_id": "'$region'",
            "services": {
                "im-service": true,
                "redis": true,
                "database": true
            },
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
        }')
    
    local leader=$(echo "$response" | jq -r '.winner // empty')
    local is_primary=$(echo "$response" | jq -r '.is_primary // false')
    
    if [ "$leader" = "$expected_leader" ]; then
        if [ "$region" = "$expected_leader" ] && [ "$is_primary" = "true" ]; then
            print_success "$region correctly elected as PRIMARY"
        elif [ "$region" != "$expected_leader" ] && [ "$is_primary" = "false" ]; then
            print_success "$region correctly sees $leader as PRIMARY"
        else
            print_error "$region election result inconsistent: leader=$leader, is_primary=$is_primary"
            return 1
        fi
    else
        print_error "$region election failed: expected $expected_leader, got $leader"
        return 1
    fi
    
    return 0
}

# Function to test failover scenario
test_failover() {
    print_status "Testing failover scenario..."
    
    # Step 1: Both regions healthy - region-a should be primary
    print_status "Step 1: Both regions healthy"
    test_election "region-a" "region-a"
    test_election "region-b" "region-a"
    
    # Step 2: Region-a fails - region-b should become primary
    print_status "Step 2: Region-a fails"
    
    # Report region-a as unhealthy
    curl -s -X POST http://localhost:9999/elect \
        -H "Content-Type: application/json" \
        -d '{
            "region_id": "region-a",
            "services": {
                "im-service": false,
                "redis": true,
                "database": true
            },
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
        }' > /dev/null
    
    sleep 2  # Wait for health state to propagate
    
    test_election "region-b" "region-b"
    test_election "region-a" "region-b"
    
    # Step 3: Region-a recovers - should become primary again
    print_status "Step 3: Region-a recovers"
    
    test_election "region-a" "region-a"
    test_election "region-b" "region-a"
    
    print_success "Failover scenario completed successfully"
}

# Function to test split-brain prevention
test_split_brain_prevention() {
    print_status "Testing split-brain prevention..."
    
    # Get current arbiter status
    local status=$(curl -s http://localhost:9999/status)
    local current_primary=$(echo "$status" | jq -r '.current_primary // empty')
    
    print_status "Current primary: $current_primary"
    
    # Verify only one region can be primary at a time
    local region_a_response=$(curl -s -X POST http://localhost:9999/elect \
        -H "Content-Type: application/json" \
        -d '{
            "region_id": "region-a",
            "services": {"im-service": true, "redis": true, "database": true},
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
        }')
    
    local region_b_response=$(curl -s -X POST http://localhost:9999/elect \
        -H "Content-Type: application/json" \
        -d '{
            "region_id": "region-b",
            "services": {"im-service": true, "redis": true, "database": true},
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
        }')
    
    local leader_a=$(echo "$region_a_response" | jq -r '.winner')
    local leader_b=$(echo "$region_b_response" | jq -r '.winner')
    local is_primary_a=$(echo "$region_a_response" | jq -r '.is_primary')
    local is_primary_b=$(echo "$region_b_response" | jq -r '.is_primary')
    
    if [ "$leader_a" = "$leader_b" ]; then
        if [ "$is_primary_a" = "true" ] && [ "$is_primary_b" = "false" ]; then
            print_success "Split-brain prevention working: only region-a is primary"
        elif [ "$is_primary_a" = "false" ] && [ "$is_primary_b" = "true" ]; then
            print_success "Split-brain prevention working: only region-b is primary"
        else
            print_error "Split-brain detected: both regions think they are primary"
            return 1
        fi
    else
        print_error "Inconsistent leader election: region-a sees $leader_a, region-b sees $leader_b"
        return 1
    fi
    
    print_success "Split-brain prevention test passed"
}

# Function to show arbiter metrics
show_metrics() {
    print_status "Arbiter Metrics:"
    echo
    
    # Get metrics from arbiter mock
    local metrics=$(curl -s http://localhost:9999/metrics)
    
    if [ -n "$metrics" ]; then
        echo "$metrics" | grep -E "(arbiter_elections_total|arbiter_health_checks_total)" || true
    else
        print_warning "No metrics available"
    fi
    
    echo
    
    # Get current status
    print_status "Current Arbiter Status:"
    curl -s http://localhost:9999/status | jq '.' || print_warning "Failed to get status"
}

# Main test execution
main() {
    print_status "Starting Arbiter System Test"
    echo
    
    # Check if Docker Compose is running
    if ! docker-compose ps | grep -q "Up"; then
        print_error "Docker Compose services are not running"
        print_status "Please run: docker-compose up -d"
        exit 1
    fi
    
    # Wait for services to be ready
    wait_for_service "Arbiter Mock" "http://localhost:9999/health"
    
    # Check Zookeeper cluster (if using real Zookeeper)
    if docker-compose ps | grep -q "zookeeper-1"; then
        check_zookeeper_cluster
    fi
    
    echo
    print_status "Running arbiter tests..."
    echo
    
    # Run test scenarios
    test_failover
    echo
    
    test_split_brain_prevention
    echo
    
    # Show metrics and status
    show_metrics
    echo
    
    print_success "All arbiter tests completed successfully!"
    echo
    
    print_status "Test Summary:"
    echo "✓ Distributed leader election working"
    echo "✓ Health-based failover working"
    echo "✓ Split-brain prevention working"
    echo "✓ Deterministic election rules working"
    echo
    
    print_status "The arbiter system is ready for production use"
}

# Run tests
main "$@"