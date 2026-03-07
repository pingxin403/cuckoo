#!/bin/bash

# Cache Protection Test Runner
# Tests cache penetration, breakdown, avalanche, and delayed double delete scenarios

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Cache Protection Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

cd "$SERVICE_DIR"

echo -e "${GREEN}Running cache protection tests...${NC}"
echo ""

# Run cache protection tests
go test -v -run "TestCache.*Protection|TestCache.*Penetration|TestCache.*Breakdown|TestCache.*Avalanche|TestDelayed" ./cache/

TEST_EXIT_CODE=$?

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ All cache protection tests passed!${NC}"
    echo ""
    echo -e "${BLUE}Test Coverage:${NC}"
    echo -e "  ${GREEN}✓${NC} Cache Penetration (non-existent keys)"
    echo -e "  ${GREEN}✓${NC} Cache Breakdown (hotspot expiration)"
    echo -e "  ${GREEN}✓${NC} Cache Avalanche (mass expiration)"
    echo -e "  ${GREEN}✓${NC} Delayed Double Delete (update consistency)"
    echo -e "  ${GREEN}✓${NC} Concurrent Load Testing"
    echo ""
else
    echo -e "${RED}✗ Some cache protection tests failed${NC}"
    exit $TEST_EXIT_CODE
fi

exit 0
