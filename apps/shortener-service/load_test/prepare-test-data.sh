#!/bin/bash

# Prepare Test Data for Load Testing
# Creates short links that will be used in load tests

set -e

GRPC_PORT=${GRPC_PORT:-50051}
BASE_URL=${BASE_URL:-http://localhost:8080}

echo "========================================="
echo "Preparing Test Data"
echo "========================================="
echo ""
echo "gRPC Port: $GRPC_PORT"
echo "Base URL: $BASE_URL"
echo ""

# Check if grpcurl is available
if ! command -v grpcurl &> /dev/null; then
    echo "Error: grpcurl is not installed"
    echo "Install: brew install grpcurl (macOS) or see https://github.com/fullstorydev/grpcurl"
    exit 1
fi

# Check if service is healthy
echo "Checking service health..."
if ! curl -sf $BASE_URL/health > /dev/null; then
    echo "Error: Service is not healthy at $BASE_URL"
    exit 1
fi
echo "✓ Service is healthy"
echo ""

# Create test codes
echo "Creating test short links..."
TEST_CODES=("test001" "test002" "test003" "test004" "test005")

for code in "${TEST_CODES[@]}"; do
    echo -n "Creating $code... "
    result=$(grpcurl -plaintext -d "{\"long_url\": \"https://example.com/page-$code\", \"custom_code\": \"$code\"}" \
        localhost:$GRPC_PORT api.v1.ShortenerService/CreateShortLink 2>&1)
    
    if echo "$result" | grep -q "shortCode"; then
        echo "✓"
    elif echo "$result" | grep -q "AlreadyExists"; then
        echo "✓ (already exists)"
    else
        echo "✗ Failed: $result"
    fi
done

echo ""
echo "Creating additional random short links for cache warming..."
for i in {1..100}; do
    grpcurl -plaintext -d "{\"long_url\": \"https://example.com/page-$i\"}" \
        localhost:$GRPC_PORT api.v1.ShortenerService/CreateShortLink > /dev/null 2>&1 || true
    
    if [ $((i % 20)) -eq 0 ]; then
        echo "  Created $i links..."
    fi
done

echo ""
echo "========================================="
echo "Test Data Ready!"
echo "========================================="
echo ""
echo "Test codes created:"
for code in "${TEST_CODES[@]}"; do
    echo "  - $BASE_URL/$code"
done
echo ""
echo "Total links created: ~100"
echo ""
echo "You can now run load tests:"
echo "  k6 run redirect-qps-test.js"
echo "  k6 run quick-qps-test.js"
echo "  k6 run cache-stampede.js"
echo ""
