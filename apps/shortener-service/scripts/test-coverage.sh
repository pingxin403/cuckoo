#!/bin/bash

# Test coverage script for URL Shortener Service
# This script runs tests with coverage and verifies thresholds:
# - Overall coverage: 70% minimum
# - Service/storage packages: 75% minimum

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

echo ""
echo "Checking coverage thresholds..."

# Check overall coverage (70%)
OVERALL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Overall coverage: ${OVERALL_COVERAGE}%"

if (( $(echo "$OVERALL_COVERAGE < 70" | bc -l) )); then
    echo "❌ FAIL: Overall coverage ${OVERALL_COVERAGE}% is below 70% threshold"
    exit 1
fi

echo "✅ PASS: Overall coverage meets 70% threshold"

# Check service and storage package coverage (75%)
SERVICE_LINES=$(go tool cover -func=coverage.out | grep -E '(service|storage)' || true)

if [ -n "$SERVICE_LINES" ]; then
    # Calculate average coverage for service/storage packages
    SERVICE_COVERAGE=$(echo "$SERVICE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) print sum/count; else print 0}' | sed 's/%//')
    echo "Service/storage coverage: ${SERVICE_COVERAGE}%"
    
    if (( $(echo "$SERVICE_COVERAGE < 75" | bc -l) )); then
        echo "❌ FAIL: Service/storage coverage ${SERVICE_COVERAGE}% is below 75% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Service/storage coverage meets 75% threshold"
else
    echo "⚠️  WARNING: No service or storage packages found"
fi

echo ""
echo "✅ All coverage thresholds met!"
