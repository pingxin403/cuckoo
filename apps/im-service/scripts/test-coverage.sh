#!/bin/bash

# Test coverage script for im-service
# Uses Go build tags to control test execution

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== IM Service Test Coverage ===${NC}"

# Parse arguments
RUN_PROPERTY_TESTS=false
TIMEOUT="5m"

while [[ $# -gt 0 ]]; do
    case $1 in
        --with-property-tests)
            RUN_PROPERTY_TESTS=true
            TIMEOUT="30m"
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--with-property-tests] [--timeout <duration>]"
            exit 1
            ;;
    esac
done

# Determine test tags
TEST_TAGS=""
if [ "$RUN_PROPERTY_TESTS" = true ]; then
    TEST_TAGS="-tags=property"
    echo -e "${YELLOW}Running ALL tests (unit + property-based)...${NC}"
else
    echo -e "${BLUE}Running unit tests only (use --with-property-tests for full suite)${NC}"
fi

# Run tests with coverage
go test ./... $TEST_TAGS -coverprofile=coverage.out -timeout="$TIMEOUT"

# Calculate coverage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}Test coverage: ${COVERAGE}${NC}"

# Run linter
echo -e "${BLUE}Running linter...${NC}"
golangci-lint run ./...

echo -e "${GREEN}=== All checks passed! ===${NC}"
