#!/bin/bash

# Load Test Runner for IM Gateway Service
# Runs various load test scenarios

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}=== IM Gateway Load Tests ===${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: k6 is not installed${NC}"
    echo "Please install k6: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Environment variables
export GATEWAY_HOST="${GATEWAY_HOST:-localhost}"
export GATEWAY_PORT="${GATEWAY_PORT:-8080}"
export AUTH_TOKEN="${AUTH_TOKEN:-test-token}"
export MESSAGE_SIZE="${MESSAGE_SIZE:-1024}"

echo -e "${BLUE}Configuration:${NC}"
echo "  Gateway: ${GATEWAY_HOST}:${GATEWAY_PORT}"
echo "  Auth Token: ${AUTH_TOKEN:0:20}..."
echo "  Message Size: ${MESSAGE_SIZE} bytes"
echo ""

# Create results directory
RESULTS_DIR="$SCRIPT_DIR/results/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo -e "${YELLOW}Results will be saved to: $RESULTS_DIR${NC}"
echo ""

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    local test_args=$3
    
    echo -e "${BLUE}Running: $test_name${NC}"
    echo "----------------------------------------"
    
    if k6 run "$SCRIPT_DIR/$test_file" $test_args \
        --out json="$RESULTS_DIR/${test_name}.json" \
        --summary-export="$RESULTS_DIR/${test_name}-summary.json" 2>&1 | tee "$RESULTS_DIR/${test_name}.log"; then
        echo -e "${GREEN}✓ $test_name completed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        return 1
    fi
    echo ""
}

# Test selection
TEST_SUITE="${1:-all}"

case "$TEST_SUITE" in
    "connection"|"conn")
        echo -e "${YELLOW}Running Connection Load Test${NC}"
        echo ""
        run_test "connection-load-test" "connection-load-test.js" ""
        ;;
    
    "throughput"|"msg")
        echo -e "${YELLOW}Running Message Throughput Test${NC}"
        echo ""
        run_test "message-throughput-test" "message-throughput-test.js" ""
        ;;
    
    "cluster")
        echo -e "${YELLOW}Running Cluster Load Test${NC}"
        echo ""
        NODE_ID="${NODE_ID:-1}"
        CLUSTER_SIZE="${CLUSTER_SIZE:-100}"
        export NODE_ID
        export CLUSTER_SIZE
        run_test "cluster-load-test-node-${NODE_ID}" "cluster-load-test.js" ""
        ;;
    
    "quick")
        echo -e "${YELLOW}Running Quick Load Tests (reduced scale)${NC}"
        echo ""
        
        echo -e "${BLUE}1. Quick Connection Test (1K connections, 2 min)${NC}"
        run_test "quick-connection-test" "connection-load-test.js" \
            "--vus 1000 --duration 2m"
        
        echo -e "${BLUE}2. Quick Throughput Test (100 users, 2 min)${NC}"
        run_test "quick-throughput-test" "message-throughput-test.js" \
            "--vus 100 --duration 2m"
        ;;
    
    "all")
        echo -e "${YELLOW}Running Full Load Test Suite${NC}"
        echo ""
        
        echo -e "${BLUE}Test 1/2: Connection Load Test${NC}"
        run_test "connection-load-test" "connection-load-test.js" ""
        
        echo ""
        echo -e "${BLUE}Test 2/2: Message Throughput Test${NC}"
        run_test "message-throughput-test" "message-throughput-test.js" ""
        ;;
    
    *)
        echo -e "${RED}Unknown test suite: $TEST_SUITE${NC}"
        echo ""
        echo "Usage: $0 [test-suite]"
        echo ""
        echo "Available test suites:"
        echo "  connection, conn    - Connection load test (100K connections)"
        echo "  throughput, msg     - Message throughput test (10K msg/sec)"
        echo "  cluster             - Cluster load test (requires NODE_ID)"
        echo "  quick               - Quick tests (reduced scale)"
        echo "  all                 - Run all tests (default)"
        echo ""
        echo "Environment variables:"
        echo "  GATEWAY_HOST        - Gateway hostname (default: localhost)"
        echo "  GATEWAY_PORT        - Gateway port (default: 8080)"
        echo "  AUTH_TOKEN          - Authentication token"
        echo "  MESSAGE_SIZE        - Message size in bytes (default: 1024)"
        echo "  NODE_ID             - Node ID for cluster test (default: 1)"
        echo "  CLUSTER_SIZE        - Cluster size (default: 100)"
        exit 1
        ;;
esac

# Generate summary report
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo ""

if [ -f "$RESULTS_DIR/connection-load-test-summary.json" ]; then
    echo -e "${GREEN}Connection Load Test:${NC}"
    jq -r '.metrics | 
        "  Success Rate: \(.connection_success.values.rate * 100 | tostring | .[0:5])%\n" +
        "  P95 Connection Time: \(.connection_duration.values["p(95)"] | tostring | .[0:7])ms\n" +
        "  P99 Message Latency: \(.message_latency.values["p(99)"] | tostring | .[0:7])ms"' \
        "$RESULTS_DIR/connection-load-test-summary.json" 2>/dev/null || echo "  (Summary not available)"
    echo ""
fi

if [ -f "$RESULTS_DIR/message-throughput-test-summary.json" ]; then
    echo -e "${GREEN}Message Throughput Test:${NC}"
    jq -r '.metrics | 
        "  Messages Sent: \(.messages_sent.values.count | tostring)\n" +
        "  Throughput: \((.messages_sent.values.count / 600) | tostring | .[0:8]) msg/sec\n" +
        "  P99 Latency: \(.message_latency.values["p(99)"] | tostring | .[0:7])ms\n" +
        "  Success Rate: \(.message_throughput.values.rate * 100 | tostring | .[0:5])%"' \
        "$RESULTS_DIR/message-throughput-test-summary.json" 2>/dev/null || echo "  (Summary not available)"
    echo ""
fi

echo -e "${YELLOW}Results saved to: $RESULTS_DIR${NC}"
echo ""

# Check for failures
if grep -q "✗" "$RESULTS_DIR"/*.log 2>/dev/null; then
    echo -e "${RED}Some tests failed. Check logs for details.${NC}"
    exit 1
else
    echo -e "${GREEN}All tests completed successfully!${NC}"
    exit 0
fi
