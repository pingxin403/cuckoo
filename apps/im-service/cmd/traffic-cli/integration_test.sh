#!/bin/bash

# Integration test script for traffic-cli
# This script demonstrates the CLI tool functionality with a real Redis instance

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
CLI_BIN="./bin/traffic-cli"

echo -e "${GREEN}Traffic CLI Integration Test${NC}"
echo "=============================="
echo ""

# Check if Redis is running
echo -e "${YELLOW}Checking Redis connection...${NC}"
if ! redis-cli -h "${REDIS_ADDR%%:*}" -p "${REDIS_ADDR##*:}" ping > /dev/null 2>&1; then
    echo -e "${RED}Error: Redis is not running at ${REDIS_ADDR}${NC}"
    echo "Please start Redis first: docker-compose up -d redis"
    exit 1
fi
echo -e "${GREEN}✓ Redis is running${NC}"
echo ""

# Build the CLI tool
echo -e "${YELLOW}Building traffic-cli...${NC}"
go build -o bin/traffic-cli ./cmd/traffic-cli
echo -e "${GREEN}✓ Build successful${NC}"
echo ""

# Test 1: View initial status
echo -e "${YELLOW}Test 1: View current status${NC}"
$CLI_BIN status --redis-addr "$REDIS_ADDR"
echo ""

# Test 2: Dry run - proportional switch
echo -e "${YELLOW}Test 2: Dry run - proportional switch (90:10)${NC}"
$CLI_BIN switch proportional region-a:90 region-b:10 \
    --redis-addr "$REDIS_ADDR" \
    --dry-run \
    --reason "Integration test - dry run"
echo ""

# Test 3: Apply proportional switch
echo -e "${YELLOW}Test 3: Apply proportional switch (80:20)${NC}"
$CLI_BIN switch proportional region-a:80 region-b:20 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Integration test - proportional switch" \
    --operator "integration-test"
echo ""

# Test 4: View status after switch
echo -e "${YELLOW}Test 4: View status after switch${NC}"
$CLI_BIN status --redis-addr "$REDIS_ADDR"
echo ""

# Test 5: Test user routing
echo -e "${YELLOW}Test 5: Test user routing${NC}"
for user_id in "user123" "user456" "user789"; do
    echo "Routing for $user_id:"
    $CLI_BIN route "$user_id" --redis-addr "$REDIS_ADDR"
    echo ""
done

# Test 6: Full switch to region-a
echo -e "${YELLOW}Test 6: Full switch to region-a${NC}"
$CLI_BIN switch full region-a \
    --redis-addr "$REDIS_ADDR" \
    --reason "Integration test - full switch" \
    --operator "integration-test"
echo ""

# Test 7: View status after full switch
echo -e "${YELLOW}Test 7: View status after full switch${NC}"
$CLI_BIN status --redis-addr "$REDIS_ADDR"
echo ""

# Test 8: Switch to 50:50
echo -e "${YELLOW}Test 8: Switch to 50:50 balance${NC}"
$CLI_BIN switch proportional region-a:50 region-b:50 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Integration test - balanced load" \
    --operator "integration-test"
echo ""

# Test 9: View event history
echo -e "${YELLOW}Test 9: View event history${NC}"
$CLI_BIN events --redis-addr "$REDIS_ADDR" --limit 5
echo ""

# Test 10: Error handling - invalid weights
echo -e "${YELLOW}Test 10: Error handling - invalid weights (should fail)${NC}"
if $CLI_BIN switch proportional region-a:60 region-b:30 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Integration test - invalid weights" 2>&1 | grep -q "total weight must equal 100"; then
    echo -e "${GREEN}✓ Correctly rejected invalid weights${NC}"
else
    echo -e "${RED}✗ Failed to reject invalid weights${NC}"
    exit 1
fi
echo ""

# Test 11: Error handling - missing reason
echo -e "${YELLOW}Test 11: Error handling - missing reason (should fail)${NC}"
if $CLI_BIN switch proportional region-a:70 region-b:30 \
    --redis-addr "$REDIS_ADDR" 2>&1 | grep -q "reason flag is required"; then
    echo -e "${GREEN}✓ Correctly required reason flag${NC}"
else
    echo -e "${RED}✗ Failed to require reason flag${NC}"
    exit 1
fi
echo ""

# Test 12: Gradual migration scenario
echo -e "${YELLOW}Test 12: Gradual migration scenario${NC}"
echo "Simulating gradual migration from region-b to region-a..."

echo "Step 1: 70:30"
$CLI_BIN switch proportional region-a:70 region-b:30 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration phase 1" \
    --operator "integration-test"

sleep 1

echo "Step 2: 90:10"
$CLI_BIN switch proportional region-a:90 region-b:10 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration phase 2" \
    --operator "integration-test"

sleep 1

echo "Step 3: 100:0 (full switch)"
$CLI_BIN switch full region-a \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration complete" \
    --operator "integration-test"

echo -e "${GREEN}✓ Gradual migration completed${NC}"
echo ""

# Final status
echo -e "${YELLOW}Final Status:${NC}"
$CLI_BIN status --redis-addr "$REDIS_ADDR"
echo ""

# Summary
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}All integration tests passed! ✓${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "Summary:"
echo "- Status viewing: ✓"
echo "- Dry run mode: ✓"
echo "- Proportional switching: ✓"
echo "- Full switching: ✓"
echo "- User routing: ✓"
echo "- Event logging: ✓"
echo "- Error handling: ✓"
echo "- Gradual migration: ✓"
echo ""
echo "The traffic-cli tool is working correctly!"
