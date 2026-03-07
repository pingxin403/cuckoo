#!/bin/bash

# Multi-Region Chaos Engineering Test Script
# This script simulates various failure scenarios to test the resilience of the dual-region setup

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MVP_DIR="$(dirname "$SCRIPT_DIR")"

echo "🔥 Starting Multi-Region Chaos Engineering Tests..."

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to send test message
send_test_message() {
    local region=$1
    local port=$2
    local message=$3
    local conversation_id=${4:-"chaos-test-conv"}
    
    log "📤 Sending message to Region $region: '$message'"
    
    curl -X POST "http://localhost:$port/api/messages" \
        -H "Content-Type: application/json" \
        -d "{
            \"conversation_id\": \"$conversation_id\",
            \"content\": \"$message\",
            \"sender_id\": \"chaos-tester\",
            \"timestamp\": $(date +%s)000
        }" \
        -w "\nHTTP Status: %{http_code}\n" \
        -s || echo "❌ Failed to send message to Region $region"
}

# Function to check message sync
check_message_sync() {
    local conversation_id=${1:-"chaos-test-conv"}
    
    log "🔍 Checking message synchronization..."
    
    echo "Region A messages:"
    curl -s "http://localhost:8080/api/messages/$conversation_id" | jq '.' || echo "❌ Failed to fetch from Region A"
    
    echo "Region B messages:"
    curl -s "http://localhost:8081/api/messages/$conversation_id" | jq '.' || echo "❌ Failed to fetch from Region B"
}

# Function to inject network latency
inject_network_latency() {
    local latency=${1:-"100ms"}
    local jitter=${2:-"20ms"}
    
    log "📡 Injecting network latency: $latency ± $jitter"
    
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay $latency $jitter distribution normal
        echo 'Network latency injected: $latency ± $jitter'
    " || log "❌ Failed to inject network latency"
}

# Function to simulate network partition
simulate_network_partition() {
    local duration=${1:-30}
    
    log "🚫 Simulating network partition for ${duration}s..."
    
    # Block cross-region traffic
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem loss 100%
        echo 'Network partition activated - 100% packet loss'
    " || log "❌ Failed to create network partition"
    
    sleep "$duration"
    
    # Restore network
    log "🔧 Restoring network connectivity..."
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms distribution normal
        echo 'Network connectivity restored'
    " || log "❌ Failed to restore network"
}

# Function to simulate region failure
simulate_region_failure() {
    local region=$1
    local duration=${2:-60}
    
    log "💥 Simulating Region $region failure for ${duration}s..."
    
    if [ "$region" = "A" ]; then
        docker stop im-service-region-a im-gateway-region-a redis-region-a
    elif [ "$region" = "B" ]; then
        docker stop im-service-region-b im-gateway-region-b redis-region-b
    else
        log "❌ Invalid region: $region"
        return 1
    fi
    
    sleep "$duration"
    
    log "🔧 Restoring Region $region..."
    if [ "$region" = "A" ]; then
        docker start redis-region-a im-service-region-a im-gateway-region-a
    elif [ "$region" = "B" ]; then
        docker start redis-region-b im-service-region-b im-gateway-region-b
    fi
    
    # Wait for services to be ready
    sleep 15
    log "✅ Region $region restored"
}

# Function to check arbiter status
check_arbiter_status() {
    log "🏛️  Checking arbiter status..."
    
    curl -s "http://localhost:9999/status" | jq '.' || log "❌ Failed to get arbiter status"
}

# Function to monitor metrics
monitor_metrics() {
    local duration=${1:-30}
    
    log "📊 Monitoring metrics for ${duration}s..."
    
    for i in $(seq 1 $duration); do
        echo "=== Metrics Sample $i/$duration ==="
        
        # HLC metrics
        echo "🕐 HLC Status:"
        curl -s "http://localhost:8080/metrics" | grep -E "hlc_(physical|logical)_time" || true
        curl -s "http://localhost:8081/metrics" | grep -E "hlc_(physical|logical)_time" || true
        
        # Conflict metrics
        echo "⚡ Conflicts:"
        curl -s "http://localhost:8080/metrics" | grep "conflict_total" || true
        
        # Sync latency
        echo "📡 Sync Latency:"
        curl -s "http://localhost:8080/metrics" | grep "sync_latency" || true
        
        echo "---"
        sleep 1
    done
}

# Main test scenarios
run_test_scenario() {
    local scenario=$1
    
    case $scenario in
        "basic")
            log "🧪 Running Basic Functionality Test..."
            send_test_message "A" "8080" "Hello from Region A"
            sleep 2
            send_test_message "B" "8081" "Hello from Region B"
            sleep 2
            check_message_sync
            ;;
            
        "latency")
            log "🧪 Running Network Latency Test..."
            inject_network_latency "200ms" "50ms"
            send_test_message "A" "8080" "High latency test message"
            sleep 5
            check_message_sync
            inject_network_latency "40ms" "10ms"  # Restore normal latency
            ;;
            
        "partition")
            log "🧪 Running Network Partition Test..."
            send_test_message "A" "8080" "Before partition"
            simulate_network_partition 30
            send_test_message "A" "8080" "During partition (should queue)"
            sleep 5
            check_message_sync
            ;;
            
        "failover-a")
            log "🧪 Running Region A Failover Test..."
            send_test_message "A" "8080" "Before Region A failure"
            simulate_region_failure "A" 60
            send_test_message "B" "8081" "During Region A failure"
            sleep 5
            check_message_sync
            ;;
            
        "failover-b")
            log "🧪 Running Region B Failover Test..."
            send_test_message "B" "8081" "Before Region B failure"
            simulate_region_failure "B" 60
            send_test_message "A" "8080" "During Region B failure"
            sleep 5
            check_message_sync
            ;;
            
        "split-brain")
            log "🧪 Running Split-Brain Prevention Test..."
            check_arbiter_status
            simulate_network_partition 45
            check_arbiter_status
            ;;
            
        "monitor")
            log "🧪 Running Metrics Monitoring Test..."
            monitor_metrics 60
            ;;
            
        *)
            log "❌ Unknown test scenario: $scenario"
            return 1
            ;;
    esac
}

# Main execution
main() {
    cd "$MVP_DIR"
    
    # Check if environment is running
    if ! docker-compose ps | grep -q "Up"; then
        log "❌ MVP environment is not running. Please run './scripts/start-mvp.sh' first."
        exit 1
    fi
    
    log "🎯 Available test scenarios:"
    log "  basic      - Basic message sending and sync"
    log "  latency    - Network latency injection"
    log "  partition  - Network partition simulation"
    log "  failover-a - Region A failure simulation"
    log "  failover-b - Region B failure simulation"
    log "  split-brain- Split-brain prevention test"
    log "  monitor    - Metrics monitoring"
    log "  all        - Run all test scenarios"
    
    local scenario=${1:-"all"}
    
    if [ "$scenario" = "all" ]; then
        log "🚀 Running all test scenarios..."
        
        run_test_scenario "basic"
        sleep 5
        
        run_test_scenario "latency"
        sleep 5
        
        run_test_scenario "partition"
        sleep 10
        
        run_test_scenario "failover-a"
        sleep 10
        
        run_test_scenario "split-brain"
        sleep 5
        
        log "✅ All chaos tests completed!"
    else
        run_test_scenario "$scenario"
    fi
    
    log "📊 Final system status:"
    docker-compose ps
    check_arbiter_status
}

# Handle script arguments
if [ $# -eq 0 ]; then
    main "all"
else
    main "$1"
fi