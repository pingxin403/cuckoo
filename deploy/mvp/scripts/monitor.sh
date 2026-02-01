#!/bin/bash

# Multi-Region Monitoring Script
# Real-time monitoring of multi-region active-active metrics

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MVP_DIR="$(dirname "$SCRIPT_DIR")"

echo "📊 Multi-Region Active-Active Real-Time Monitor"
echo "Press Ctrl+C to stop monitoring"
echo ""

# Function to format timestamp
timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

# Function to get metric value
get_metric() {
    local endpoint=$1
    local metric_name=$2
    
    curl -s "$endpoint/metrics" | grep "^$metric_name" | head -1 | awk '{print $2}' || echo "N/A"
}

# Function to get health status
get_health() {
    local endpoint=$1
    
    if curl -f -s "$endpoint/health" >/dev/null 2>&1; then
        echo "✅ UP"
    else
        echo "❌ DOWN"
    fi
}

# Function to get arbiter status
get_arbiter_info() {
    local status_json=$(curl -s "http://localhost:9999/status" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ -n "$status_json" ]; then
        local primary=$(echo "$status_json" | jq -r '.current_primary // "none"')
        local region_count=$(echo "$status_json" | jq '.region_health | length')
        echo "Primary: $primary | Regions: $region_count"
    else
        echo "❌ Arbiter Unavailable"
    fi
}

# Function to display system overview
display_overview() {
    clear
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                    Multi-Region Active-Active Monitor                        ║"
    echo "║                           $(timestamp)                            ║"
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Service Health Status
    echo "🏥 Service Health Status:"
    echo "┌─────────────────────┬──────────────┬──────────────┐"
    echo "│ Service             │ Region A     │ Region B     │"
    echo "├─────────────────────┼──────────────┼──────────────┤"
    printf "│ %-19s │ %-12s │ %-12s │\n" "IM Gateway" "$(get_health "http://localhost:8080")" "$(get_health "http://localhost:8081")"
    printf "│ %-19s │ %-12s │ %-12s │\n" "IM Service" "$(get_health "http://localhost:8080")" "$(get_health "http://localhost:8081")"
    echo "└─────────────────────┴──────────────┴──────────────┘"
    echo ""
    
    # Arbiter Status
    echo "🏛️  Arbiter Status: $(get_arbiter_info)"
    echo ""
    
    # HLC Clock Status
    echo "🕐 HLC Clock Status:"
    local hlc_a_physical=$(get_metric "http://localhost:8080" "hlc_physical_time_ms")
    local hlc_a_logical=$(get_metric "http://localhost:8080" "hlc_logical_time")
    local hlc_b_physical=$(get_metric "http://localhost:8081" "hlc_physical_time_ms")
    local hlc_b_logical=$(get_metric "http://localhost:8081" "hlc_logical_time")
    
    echo "┌─────────────────────┬──────────────┬──────────────┐"
    echo "│ Clock Component     │ Region A     │ Region B     │"
    echo "├─────────────────────┼──────────────┼──────────────┤"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Physical Time (ms)" "$hlc_a_physical" "$hlc_b_physical"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Logical Counter" "$hlc_a_logical" "$hlc_b_logical"
    echo "└─────────────────────┴──────────────┴──────────────┘"
    echo ""
    
    # Sync Metrics
    echo "📡 Synchronization Metrics:"
    local sync_latency_a=$(get_metric "http://localhost:8080" "sync_latency_seconds")
    local sync_latency_b=$(get_metric "http://localhost:8081" "sync_latency_seconds")
    local messages_synced_a=$(get_metric "http://localhost:8080" "messages_synced_total")
    local messages_synced_b=$(get_metric "http://localhost:8081" "messages_synced_total")
    
    echo "┌─────────────────────┬──────────────┬──────────────┐"
    echo "│ Metric              │ Region A     │ Region B     │"
    echo "├─────────────────────┼──────────────┼──────────────┤"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Sync Latency (s)" "$sync_latency_a" "$sync_latency_b"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Messages Synced" "$messages_synced_a" "$messages_synced_b"
    echo "└─────────────────────┴──────────────┴──────────────┘"
    echo ""
    
    # Conflict Resolution
    echo "⚡ Conflict Resolution:"
    local conflicts_a=$(get_metric "http://localhost:8080" "conflict_total")
    local conflicts_b=$(get_metric "http://localhost:8081" "conflict_total")
    local conflict_rate_a=$(get_metric "http://localhost:8080" "conflict_rate")
    local conflict_rate_b=$(get_metric "http://localhost:8081" "conflict_rate")
    
    echo "┌─────────────────────┬──────────────┬──────────────┐"
    echo "│ Metric              │ Region A     │ Region B     │"
    echo "├─────────────────────┼──────────────┼──────────────┤"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Total Conflicts" "$conflicts_a" "$conflicts_b"
    printf "│ %-19s │ %-12s │ %-12s │\n" "Conflict Rate (/s)" "$conflict_rate_a" "$conflict_rate_b"
    echo "└─────────────────────┴──────────────┴──────────────┘"
    echo ""
    
    # Network Status
    echo "🌐 Network Status:"
    local network_errors=$(get_metric "http://localhost:8080" "cross_region_network_errors_total")
    local network_latency=$(get_metric "http://localhost:8080" "cross_region_latency_ms")
    
    echo "┌─────────────────────┬──────────────────────────────┐"
    echo "│ Metric              │ Value                        │"
    echo "├─────────────────────┼──────────────────────────────┤"
    printf "│ %-19s │ %-28s │\n" "Network Errors" "$network_errors"
    printf "│ %-19s │ %-28s │\n" "Cross-Region Latency" "${network_latency}ms"
    echo "└─────────────────────┴──────────────────────────────┘"
    echo ""
    
    # Container Status
    echo "📦 Container Status:"
    cd "$MVP_DIR"
    docker-compose ps --format "table {{.Name}}\t{{.State}}\t{{.Ports}}" | head -10
    echo ""
    
    # Recent Events (if available)
    echo "📋 Recent Events:"
    local events=$(curl -s "http://localhost:9999/status" 2>/dev/null | jq -r '.election_history[-3:][]? | "\(.timestamp | strftime("%H:%M:%S")) - \(.reason): \(.winner)"' 2>/dev/null || echo "No recent events")
    echo "$events"
    echo ""
    
    echo "💡 Commands: [q]uit, [r]efresh, [c]haos test, [l]ogs"
    echo "Next refresh in 5 seconds..."
}

# Function to show logs
show_logs() {
    echo "📋 Recent logs from all services:"
    cd "$MVP_DIR"
    docker-compose logs --tail=20 --timestamps
    echo ""
    read -p "Press Enter to continue monitoring..."
}

# Function to run quick chaos test
run_quick_chaos() {
    echo "🔥 Running quick chaos test..."
    cd "$MVP_DIR"
    ./scripts/chaos-test.sh basic
    echo ""
    read -p "Press Enter to continue monitoring..."
}

# Main monitoring loop
main() {
    cd "$MVP_DIR"
    
    # Check if environment is running
    if ! docker-compose ps | grep -q "Up"; then
        echo "❌ MVP environment is not running. Please run './scripts/start-mvp.sh' first."
        exit 1
    fi
    
    # Set up signal handling
    trap 'echo ""; echo "👋 Monitoring stopped."; exit 0' INT TERM
    
    while true; do
        display_overview
        
        # Non-blocking input check
        if read -t 5 -n 1 input 2>/dev/null; then
            case $input in
                'q'|'Q')
                    echo ""
                    echo "👋 Monitoring stopped."
                    exit 0
                    ;;
                'r'|'R')
                    continue
                    ;;
                'c'|'C')
                    run_quick_chaos
                    ;;
                'l'|'L')
                    show_logs
                    ;;
                *)
                    ;;
            esac
        fi
    done
}

# Handle script arguments
case "${1:-monitor}" in
    "monitor"|"")
        main
        ;;
    "once")
        display_overview
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [monitor|once|help]"
        echo "  monitor (default) - Continuous monitoring with refresh"
        echo "  once             - Display metrics once and exit"
        echo "  help             - Show this help message"
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac