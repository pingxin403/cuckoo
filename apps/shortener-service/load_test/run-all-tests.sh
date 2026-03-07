#!/bin/bash

# Run All Load Tests
# This script runs all k6 load test scenarios and collects results.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
RESULTS_DIR="./results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Redis Optimization - Load Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Base URL: ${GREEN}$BASE_URL${NC}"
echo -e "Results Directory: ${GREEN}$RESULTS_DIR${NC}"
echo -e "Timestamp: ${GREEN}$TIMESTAMP${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: k6 is not installed${NC}"
    echo "Please install k6: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if service is running
echo -e "${YELLOW}Checking if service is running...${NC}"
if ! curl -s -f "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}Error: Service is not running at $BASE_URL${NC}"
    echo "Please start the service first:"
    echo "  docker-compose -f docker-compose.loadtest.yml up -d"
    exit 1
fi
echo -e "${GREEN}✓ Service is running${NC}"
echo ""

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    local duration=$3
    
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running: $test_name${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "Test File: ${GREEN}$test_file${NC}"
    echo -e "Expected Duration: ${GREEN}$duration${NC}"
    echo ""
    
    local output_file="$RESULTS_DIR/${test_name}_${TIMESTAMP}.json"
    local summary_file="$RESULTS_DIR/${test_name}_${TIMESTAMP}_summary.txt"
    
    echo -e "${YELLOW}Starting test...${NC}"
    
    # Run k6 test
    if k6 run \
        -e BASE_URL="$BASE_URL" \
        --out json="$output_file" \
        "$test_file" | tee "$summary_file"; then
        echo -e "${GREEN}✓ Test completed successfully${NC}"
        echo -e "Results saved to: ${GREEN}$output_file${NC}"
        echo -e "Summary saved to: ${GREEN}$summary_file${NC}"
        return 0
    else
        echo -e "${RED}✗ Test failed${NC}"
        return 1
    fi
    
    echo ""
}

# Test 1: Cache Stampede Test (fastest, run first)
echo -e "${YELLOW}Test 1/3: Cache Stampede Test${NC}"
if run_test "cache-stampede" "cache-stampede.js" "~30 seconds"; then
    STAMPEDE_PASSED=true
else
    STAMPEDE_PASSED=false
fi
echo ""
sleep 5

# Test 2: Spike Test (medium duration)
echo -e "${YELLOW}Test 2/3: Spike Test${NC}"
if run_test "spike-test" "spike-test.js" "~4 minutes"; then
    SPIKE_PASSED=true
else
    SPIKE_PASSED=false
fi
echo ""
sleep 10

# Test 3: Sustained Load Test (longest, run last)
echo -e "${YELLOW}Test 3/3: Sustained Load Test${NC}"
echo -e "${YELLOW}Warning: This test will take 10 minutes${NC}"
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if run_test "sustained-load" "sustained-load.js" "~10 minutes"; then
        SUSTAINED_PASSED=true
    else
        SUSTAINED_PASSED=false
    fi
else
    echo -e "${YELLOW}Skipping sustained load test${NC}"
    SUSTAINED_PASSED="skipped"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Suite Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ "$STAMPEDE_PASSED" = true ]; then
    echo -e "Cache Stampede Test: ${GREEN}✓ PASSED${NC}"
else
    echo -e "Cache Stampede Test: ${RED}✗ FAILED${NC}"
fi

if [ "$SPIKE_PASSED" = true ]; then
    echo -e "Spike Test: ${GREEN}✓ PASSED${NC}"
else
    echo -e "Spike Test: ${RED}✗ FAILED${NC}"
fi

if [ "$SUSTAINED_PASSED" = true ]; then
    echo -e "Sustained Load Test: ${GREEN}✓ PASSED${NC}"
elif [ "$SUSTAINED_PASSED" = "skipped" ]; then
    echo -e "Sustained Load Test: ${YELLOW}⊘ SKIPPED${NC}"
else
    echo -e "Sustained Load Test: ${RED}✗ FAILED${NC}"
fi

echo ""
echo -e "Results Directory: ${GREEN}$RESULTS_DIR${NC}"
echo ""

# Check if all tests passed
if [ "$STAMPEDE_PASSED" = true ] && [ "$SPIKE_PASSED" = true ] && ([ "$SUSTAINED_PASSED" = true ] || [ "$SUSTAINED_PASSED" = "skipped" ]); then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}All tests completed successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}Some tests failed${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
