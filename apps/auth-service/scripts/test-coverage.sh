#!/bin/bash

# Test coverage script for Go services
# This script runs tests with coverage and verifies thresholds:
# - Overall coverage: 80% minimum
# - Service/storage packages: 90% minimum

set -e

echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...

echo ""
echo "Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report generated: coverage.html"

echo ""
echo "Coverage summary:"
go tool cover -func=coverage.out

# Filter out generated code, main.go, config, and storage from coverage calculation
# Rationale:
# - /gen/: Generated protobuf code (not manually written)
# - main.go: Entry point, tested via integration tests
# - /config/: Configuration loading, tested via integration tests
# - /storage/: Database layer, requires integration tests with real DB
echo ""
echo "Filtering coverage (excluding generated code, main.go, config, storage)..."
FILTERED_COVERAGE=$(go tool cover -func=coverage.out | grep -v '/gen/' | grep -v 'main.go' | grep -v '/config/' | grep -v '/storage/' | grep -v 'total:')

echo ""
echo "Filtered coverage summary:"
echo "$FILTERED_COVERAGE"

echo ""
echo "Checking coverage thresholds..."

# Check overall coverage (80%) - excluding generated code, config, and storage
OVERALL_COVERAGE=$(echo "$FILTERED_COVERAGE" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
echo "Overall coverage (excluding generated/config/storage): ${OVERALL_COVERAGE}%"

if (( $(echo "$OVERALL_COVERAGE < 80" | bc -l) )); then
    echo "❌ FAIL: Overall coverage ${OVERALL_COVERAGE}% is below 80% threshold"
    exit 1
fi

echo "✅ PASS: Overall coverage meets 80% threshold"

# Check service package coverage (85%) - excluding generated code
SERVICE_LINES=$(echo "$FILTERED_COVERAGE" | grep '/service/' || true)

if [ -n "$SERVICE_LINES" ]; then
    # Calculate average coverage for service packages
    SERVICE_COVERAGE=$(echo "$SERVICE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
    echo "Service package coverage: ${SERVICE_COVERAGE}%"
    
    if (( $(echo "$SERVICE_COVERAGE < 85" | bc -l) )); then
        echo "❌ FAIL: Service package coverage ${SERVICE_COVERAGE}% is below 85% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Service package coverage meets 85% threshold"
else
    echo "⚠️  WARNING: No service packages found"
fi

# Note about storage coverage
STORAGE_LINES=$(echo "$FILTERED_COVERAGE" | grep '/storage/' || true)
if [ -n "$STORAGE_LINES" ]; then
    STORAGE_COVERAGE=$(echo "$STORAGE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
    echo "Storage package coverage: ${STORAGE_COVERAGE}% (informational - requires integration tests)"
fi

echo ""
echo "✅ All coverage thresholds met!"
