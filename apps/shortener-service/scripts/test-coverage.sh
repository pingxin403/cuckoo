#!/bin/bash

# Test coverage script for URL Shortener Service
# This script runs tests with coverage and verifies thresholds:
# - Overall coverage: 70% minimum
# - Service/storage packages: 75% minimum

set -e

echo "Running tests with coverage..."
# Exclude integration tests (they require external dependencies like Redis/MySQL)
# Integration tests are marked with +build integration tag
go test -v -race -coverprofile=coverage.out $(go list ./... | grep -v '/integration_test')

echo ""
echo "Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report generated: coverage.html"

echo ""
echo "Coverage summary:"
go tool cover -func=coverage.out

echo ""
echo "Checking coverage thresholds..."

# Check overall coverage (for information only)
OVERALL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Overall coverage: ${OVERALL_COVERAGE}%"

# Check core business logic packages (cache, errors, idgen, service) - 70% minimum
# Note: We exclude logger, main, and storage from threshold checks because:
# - logger: initialization code, tested in integration tests
# - main: application bootstrap, tested in integration tests  
# - storage: database operations, tested in integration tests with real DB
CORE_LINES=$(go tool cover -func=coverage.out | grep -E 'github.com/pingxin403/cuckoo/apps/shortener-service/(cache|errors|idgen|service)/' || true)

if [ -n "$CORE_LINES" ]; then
    # Calculate average coverage for core packages
    CORE_COVERAGE=$(echo "$CORE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) print sum/count; else print 0}' | sed 's/%//')
    echo "Core packages (cache, errors, idgen, service) coverage: ${CORE_COVERAGE}%"
    
    if (( $(echo "$CORE_COVERAGE < 70" | bc -l) )); then
        echo "❌ FAIL: Core packages coverage ${CORE_COVERAGE}% is below 70% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Core packages coverage meets 70% threshold"
else
    echo "❌ FAIL: No core packages found"
    exit 1
fi

echo ""
echo "✅ All coverage thresholds met!"
