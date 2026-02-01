#!/bin/bash

# Network Testing Script for Multi-Region Setup
# Tests network connectivity, latency, and traffic control functionality

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MVP_DIR="$(dirname "$SCRIPT_DIR")"

echo "🌐 Multi-Region Network Testing Suite"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to test basic connectivity
test_connectivity() {
    log "🔍 Testing basic connectivity between regions..."
    
    echo "Testing Region A to Region B:"
    docker exec im-service-region-a ping -c 3 im-service-region-b || echo "❌ A->B ping failed"
    
    echo "Testing Region B to Region A:"
    docker exec im-service-region-b ping -c 3 im-service-region-a || echo "❌ B->A ping failed"
    
    echo "Testing cross-region network:"
    docker exec im-service-region-a ping -c 3 network-chaos || echo "❌ A->Network-Chaos ping failed"
    docker exec im-service-region-b ping -c 3 network-chaos || echo "❌ B->Network-Chaos ping failed"
}

# Function to measure network latency
measure_latency() {
    log "📏 Measuring network latency..."
    
    echo "Region A to Region B latency:"
    docker exec im-service-region-a sh -c "
        for i in \$(seq 1 5); do
            ping -c 1 -W 1 im-service-region-b | grep 'time=' | awk -F'time=' '{print \$2}' | awk '{print \$1}'
        done | awk '{sum+=\$1; count++} END {if(count>0) printf \"Average: %.2fms\\n\", sum/count}'
    "
    
    echo "Region B to Region A latency:"
    docker exec im-service-region-b sh -c "
        for i in \$(seq 1 5); do
            ping -c 1 -W 1 im-service-region-a | grep 'time=' | awk -F'time=' '{print \$2}' | awk '{print \$1}'
        done | awk '{sum+=\$1; count++} END {if(count>0) printf \"Average: %.2fms\\n\", sum/count}'
    "
}

# Function to test traffic control
test_traffic_control() {
    log "🚦 Testing traffic control functionality..."
    
    echo "Current network configuration:"
    docker exec network-chaos tc qdisc show dev eth0 || echo "❌ Failed to show tc config"
    
    echo "Testing latency injection (100ms):"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 100ms 20ms
        echo 'Latency injected: 100ms ± 20ms'
    "
    
    # Measure latency with injection
    echo "Measuring latency with injection:"
    docker exec im-service-region-a sh -c "
        ping -c 3 im-service-region-b | grep 'time=' | tail -1
    " || echo "❌ Latency test failed"
    
    # Restore normal latency
    echo "Restoring normal latency (40ms):"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms
        echo 'Normal latency restored: 40ms ± 10ms'
    "
}

# Function to test packet loss simulation
test_packet_loss() {
    log "📦 Testing packet loss simulation..."
    
    echo "Injecting 10% packet loss:"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms loss 10%
        echo 'Packet loss injected: 10%'
    "
    
    # Test with packet loss
    echo "Testing connectivity with packet loss:"
    docker exec im-service-region-a ping -c 10 im-service-region-b | grep -E "(transmitted|received)" || echo "❌ Packet loss test failed"
    
    # Restore normal network
    echo "Restoring normal network:"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms
        echo 'Normal network restored'
    "
}

# Function to test bandwidth limiting
test_bandwidth_limit() {
    log "📊 Testing bandwidth limiting..."
    
    echo "Setting bandwidth limit to 1Mbps:"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: htb default 30
        tc class add dev eth0 parent 1: classid 1:1 htb rate 1mbit
        tc class add dev eth0 parent 1:1 classid 1:10 htb rate 1mbit ceil 1mbit
        tc qdisc add dev eth0 parent 1:10 handle 10: netem delay 40ms 10ms
        echo 'Bandwidth limited to 1Mbps'
    "
    
    # Test bandwidth (simple test)
    echo "Testing limited bandwidth (this may take a moment):"
    timeout 10s docker exec im-service-region-a sh -c "
        dd if=/dev/zero bs=1M count=5 2>/dev/null | nc im-service-region-b 12345 &
        sleep 2
        kill %1 2>/dev/null || true
    " || echo "Bandwidth test completed"
    
    # Restore normal network
    echo "Restoring normal network:"
    docker exec network-chaos sh -c "
        tc qdisc del dev eth0 root 2>/dev/null || true
        tc qdisc add dev eth0 root handle 1: prio
        tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms
        echo 'Normal network restored'
    "
}

# Function to test network isolation
test_network_isolation() {
    log "🔒 Testing network isolation between regions..."
    
    echo "Testing Region A internal network:"
    docker exec im-service-region-a ping -c 2 redis-region-a || echo "❌ A->Redis-A failed"
    docker exec im-service-region-a ping -c 2 redis-region-b && echo "⚠️  A can reach Redis-B (should be isolated)" || echo "✅ A->Redis-B properly isolated"
    
    echo "Testing Region B internal network:"
    docker exec im-service-region-b ping -c 2 redis-region-b || echo "❌ B->Redis-B failed"
    docker exec im-service-region-b ping -c 2 redis-region-a && echo "⚠️  B can reach Redis-A (should be isolated)" || echo "✅ B->Redis-A properly isolated"
}

# Function to test DNS resolution
test_dns_resolution() {
    log "🔍 Testing DNS resolution..."
    
    echo "Testing service discovery:"
    docker exec im-service-region-a nslookup im-service-region-b || echo "❌ DNS resolution failed"
    docker exec im-service-region-a nslookup kafka || echo "❌ Kafka DNS resolution failed"
    docker exec im-service-region-a nslookup arbiter-mock || echo "❌ Arbiter DNS resolution failed"
}

# Function to show network topology
show_network_topology() {
    log "🗺️  Network topology overview..."
    
    echo "Docker networks:"
    docker network ls | grep mvp || echo "No MVP networks found"
    
    echo ""
    echo "Network details:"
    for network in region-a-net region-b-net cross-region-net; do
        echo "--- $network ---"
        docker network inspect "mvp_${network}" --format '{{range .Containers}}{{.Name}}: {{.IPv4Address}}{{"\n"}}{{end}}' 2>/dev/null || echo "Network not found"
    done
}

# Function to run comprehensive network test
run_comprehensive_test() {
    log "🚀 Running comprehensive network test suite..."
    
    test_connectivity
    echo ""
    
    measure_latency
    echo ""
    
    test_traffic_control
    echo ""
    
    test_packet_loss
    echo ""
    
    test_network_isolation
    echo ""
    
    test_dns_resolution
    echo ""
    
    show_network_topology
    echo ""
    
    log "✅ Comprehensive network test completed!"
}

# Main execution
main() {
    cd "$MVP_DIR"
    
    # Check if environment is running
    if ! docker-compose ps | grep -q "Up"; then
        log "❌ MVP environment is not running. Please run './scripts/start-mvp.sh' first."
        exit 1
    fi
    
    local test_type=${1:-"comprehensive"}
    
    case $test_type in
        "connectivity")
            test_connectivity
            ;;
        "latency")
            measure_latency
            ;;
        "traffic-control")
            test_traffic_control
            ;;
        "packet-loss")
            test_packet_loss
            ;;
        "bandwidth")
            test_bandwidth_limit
            ;;
        "isolation")
            test_network_isolation
            ;;
        "dns")
            test_dns_resolution
            ;;
        "topology")
            show_network_topology
            ;;
        "comprehensive"|"all")
            run_comprehensive_test
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [test-type]"
            echo ""
            echo "Available test types:"
            echo "  connectivity     - Basic ping tests between regions"
            echo "  latency         - Network latency measurements"
            echo "  traffic-control - Traffic control functionality"
            echo "  packet-loss     - Packet loss simulation"
            echo "  bandwidth       - Bandwidth limiting tests"
            echo "  isolation       - Network isolation verification"
            echo "  dns             - DNS resolution tests"
            echo "  topology        - Network topology overview"
            echo "  comprehensive   - Run all tests (default)"
            echo "  help            - Show this help message"
            ;;
        *)
            log "❌ Unknown test type: $test_type"
            log "Use '$0 help' for available options"
            exit 1
            ;;
    esac
}

# Execute main function with arguments
main "$@"