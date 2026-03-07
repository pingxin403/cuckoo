#!/bin/bash

# Example: Emergency Failover
# Use this script when a region experiences a critical outage

set -e

CLI="./bin/traffic-cli"
REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
OPERATOR="${OPERATOR:-oncall-engineer}"

# Parse arguments
FAILED_REGION="${1:-region-a}"
TARGET_REGION="${2:-region-b}"

if [ "$FAILED_REGION" = "$TARGET_REGION" ]; then
    echo "Error: Failed region and target region cannot be the same"
    exit 1
fi

echo "Emergency Failover Procedure"
echo "============================"
echo ""
echo "Failed Region: $FAILED_REGION"
echo "Target Region: $TARGET_REGION"
echo ""

# Show current status
echo "Current Status:"
$CLI status --redis-addr "$REDIS_ADDR"
echo ""

# Confirm failover
read -p "⚠️  This will switch ALL traffic to $TARGET_REGION. Continue? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Failover cancelled."
    exit 0
fi

echo ""
echo "Executing emergency failover..."

# Perform the failover
$CLI switch full "$TARGET_REGION" \
    --redis-addr "$REDIS_ADDR" \
    --reason "Emergency failover: $FAILED_REGION outage" \
    --operator "$OPERATOR"

echo ""
echo "✓ Failover complete!"
echo ""

# Show new status
echo "New Status:"
$CLI status --redis-addr "$REDIS_ADDR"
echo ""

# Post-failover checklist
echo "Post-Failover Checklist:"
echo "========================"
echo "□ Verify all services are responding"
echo "□ Check error rates and latency metrics"
echo "□ Notify team in incident channel"
echo "□ Update status page"
echo "□ Begin investigation of $FAILED_REGION"
echo "□ Document incident timeline"
echo ""

# Log the event
echo "Event logged. View with:"
echo "  $CLI events --redis-addr $REDIS_ADDR"
echo ""

# Recovery instructions
echo "Recovery Instructions:"
echo "====================="
echo "When $FAILED_REGION is healthy again:"
echo ""
echo "1. Verify $FAILED_REGION health:"
echo "   - Check all services are running"
echo "   - Verify database connectivity"
echo "   - Run health checks"
echo ""
echo "2. Gradually restore traffic:"
echo "   $CLI switch proportional $TARGET_REGION:90 $FAILED_REGION:10 \\"
echo "     --reason 'Recovery phase 1' --operator '$OPERATOR'"
echo ""
echo "   $CLI switch proportional $TARGET_REGION:50 $FAILED_REGION:50 \\"
echo "     --reason 'Recovery complete' --operator '$OPERATOR'"
