#!/bin/bash

# Go Test Coverage Script
# This script runs tests with coverage and generates reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=70  # TODO: increase to 80% as more tests are added
SERVICE_COVERAGE_THRESHOLD=75  # TODO: increase to 90% as more tests are added
COVERAGE_FILE="coverage.out"
COVERAGE_HTML="coverage.html"

echo -e "${BLUE}Running Go tests with coverage...${NC}\n"

# Run tests with coverage, excluding generated code
go test ./... -coverprofile="$COVERAGE_FILE" -covermode=atomic

# Remove generated code from coverage report
if [ -f "$COVERAGE_FILE" ]; then
    # Exclude gen/ directory and main.go from coverage
    grep -v "/gen/" "$COVERAGE_FILE" | grep -v "main.go" > "${COVERAGE_FILE}.tmp"
    mv "${COVERAGE_FILE}.tmp" "$COVERAGE_FILE"
fi

# Check if coverage file was generated
if [ ! -f "$COVERAGE_FILE" ]; then
    echo -e "${RED}Error: Coverage file not generated${NC}"
    exit 1
fi

echo -e "\n${BLUE}Generating coverage report...${NC}"

# Generate HTML coverage report
go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"

echo -e "${GREEN}✓ HTML coverage report generated: $COVERAGE_HTML${NC}"

# Calculate overall coverage
OVERALL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')

echo -e "\n${BLUE}Coverage Summary:${NC}"
echo -e "Overall coverage: ${OVERALL_COVERAGE}%"

# Check service package coverage
SERVICE_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "service/" | grep -v "_test.go" | awk '{sum+=$3; count++} END {if(count>0) print sum/count; else print 0}')
STORAGE_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "storage/" | grep -v "_test.go" | awk '{sum+=$3; count++} END {if(count>0) print sum/count; else print 0}')

echo -e "Service package coverage: ${SERVICE_COVERAGE}%"
echo -e "Storage package coverage: ${STORAGE_COVERAGE}%"

# Display detailed coverage by package
echo -e "\n${BLUE}Coverage by package:${NC}"
go tool cover -func="$COVERAGE_FILE" | grep -E "^github.com" | awk '{print $1 " " $3}' | sort -u

# Check if coverage meets threshold
echo -e "\n${BLUE}Coverage Verification:${NC}"

PASS=true

# Check overall coverage
if (( $(echo "$OVERALL_COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
    echo -e "${RED}✗ Overall coverage ${OVERALL_COVERAGE}% is below threshold ${COVERAGE_THRESHOLD}%${NC}"
    PASS=false
else
    echo -e "${GREEN}✓ Overall coverage ${OVERALL_COVERAGE}% meets threshold ${COVERAGE_THRESHOLD}%${NC}"
fi

# Check service package coverage
if (( $(echo "$SERVICE_COVERAGE < $SERVICE_COVERAGE_THRESHOLD" | bc -l) )); then
    echo -e "${RED}✗ Service coverage ${SERVICE_COVERAGE}% is below threshold ${SERVICE_COVERAGE_THRESHOLD}%${NC}"
    PASS=false
else
    echo -e "${GREEN}✓ Service coverage ${SERVICE_COVERAGE}% meets threshold ${SERVICE_COVERAGE_THRESHOLD}%${NC}"
fi

# Check storage package coverage
if (( $(echo "$STORAGE_COVERAGE < $SERVICE_COVERAGE_THRESHOLD" | bc -l) )); then
    echo -e "${RED}✗ Storage coverage ${STORAGE_COVERAGE}% is below threshold ${SERVICE_COVERAGE_THRESHOLD}%${NC}"
    PASS=false
else
    echo -e "${GREEN}✓ Storage coverage ${STORAGE_COVERAGE}% meets threshold ${SERVICE_COVERAGE_THRESHOLD}%${NC}"
fi

echo ""

if [ "$PASS" = true ]; then
    echo -e "${GREEN}✓ All coverage thresholds met!${NC}"
    echo -e "${BLUE}View detailed report: open $COVERAGE_HTML${NC}"
    exit 0
else
    echo -e "${RED}✗ Coverage thresholds not met${NC}"
    echo -e "${YELLOW}Run 'go tool cover -html=$COVERAGE_FILE' to see detailed coverage${NC}"
    exit 1
fi

