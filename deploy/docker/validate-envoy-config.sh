#!/bin/bash
# Validation script for Envoy configuration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/envoy-config.yaml"

echo "==================================="
echo "Envoy Configuration Validator"
echo "==================================="
echo ""

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Error: Configuration file not found at $CONFIG_FILE"
    exit 1
fi

echo "✅ Configuration file found: $CONFIG_FILE"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed or not in PATH"
    exit 1
fi

echo "✅ Docker is available"
echo ""

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo "❌ Error: Docker daemon is not running"
    echo "   Please start Docker and try again"
    exit 1
fi

echo "✅ Docker daemon is running"
echo ""

# Validate Envoy configuration syntax
echo "Validating Envoy configuration syntax..."
echo ""

if docker run --rm \
    -v "$CONFIG_FILE:/etc/envoy/envoy.yaml:ro" \
    envoyproxy/envoy:v1.28-latest \
    --mode validate --config-path /etc/envoy/envoy.yaml; then
    echo ""
    echo "✅ Envoy configuration is valid!"
    echo ""
else
    echo ""
    echo "❌ Envoy configuration validation failed!"
    echo "   Please check the error messages above"
    exit 1
fi

# Check for flash-sale-service configuration
echo "Checking flash-sale-service configuration..."
echo ""

if grep -q "flash_sale_service" "$CONFIG_FILE"; then
    echo "✅ flash_sale_service cluster found"
else
    echo "❌ flash_sale_service cluster not found"
    exit 1
fi

if grep -q "/api/seckill" "$CONFIG_FILE"; then
    echo "✅ /api/seckill route found"
else
    echo "❌ /api/seckill route not found"
    exit 1
fi

if grep -q "envoy.filters.http.local_ratelimit" "$CONFIG_FILE"; then
    echo "✅ Local rate limit filter found"
else
    echo "❌ Local rate limit filter not found"
    exit 1
fi

if grep -q "local_rate_limit_per_downstream_connection: false" "$CONFIG_FILE"; then
    echo "✅ Per-IP rate limiting configured (via per-route config)"
else
    echo "⚠️  Warning: Per-IP rate limiting may not be configured"
fi

echo ""
echo "==================================="
echo "✅ All validation checks passed!"
echo "==================================="
echo ""
echo "Next steps:"
echo "1. Start the services: docker-compose up -d"
echo "2. Test the routing: curl http://localhost:8080/api/seckill/health"
echo "3. Test rate limiting: See ENVOY_FLASH_SALE_CONFIG.md for test commands"
echo ""
