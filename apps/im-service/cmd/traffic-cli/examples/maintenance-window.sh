#!/bin/bash

# Example: Maintenance Window
# Drain traffic from a region before performing maintenance

set -e

CLI="./bin/traffic-cli"
REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
OPERATOR="${OPERATOR:-ops-team}"

# Parse arguments
MAINTENANCE_REGION="${1:-region-b}"
ACTIVE_REGION="${2:-region-a}"

if [ "$MAINTENANCE_REGION" = "$ACTIVE_REGION" ]; then
    echo "Error: Maintenance region and active region cannot be the same"
    exit 1
fi

echo "Maintenance Window Procedure"
echo "============================"
echo ""
echo "Maintenance Region: $MAINTENANCE_REGION"
echo "Active Region: $ACTIVE_REGION"
echo ""

# Show current status
echo "Current Status:"
$CLI status --redis-addr "$REDIS_ADDR"
echo ""

# Step 1: Drain traffic
echo "Step 1: Draining traffic from $MAINTENANCE_REGION"
echo "=================================================="
read -p "Continue? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Maintenance cancelled."
    exit 0
fi

$CLI switch full "$ACTIVE_REGION" \
    --redis-addr "$REDIS_ADDR" \
    --reason "Maintenance window: Draining $MAINTENANCE_REGION" \
    --operator "$OPERATOR"

echo ""
echo "✓ Traffic drained from $MAINTENANCE_REGION"
echo ""

# Show new status
$CLI status --redis-addr "$REDIS_ADDR"
echo ""

# Wait for connections to drain
echo "Waiting for existing connections to drain..."
echo "Recommended wait time: 2-5 minutes"
echo ""
read -p "Press Enter when ready to proceed with maintenance..."

# Step 2: Perform maintenance
echo ""
echo "Step 2: Perform Maintenance"
echo "==========================="
echo ""
echo "You can now safely perform maintenance on $MAINTENANCE_REGION:"
echo ""
echo "Maintenance Tasks:"
echo "□ Database upgrades"
echo "□ OS patches"
echo "□ Configuration changes"
echo "□ Hardware maintenance"
echo "□ Network changes"
echo ""
read -p "Press Enter when maintenance is complete..."

# Step 3: Verify health
echo ""
echo "Step 3: Verify Health"
echo "===================="
echo ""
echo "Before restoring traffic, verify $MAINTENANCE_REGION health:"
echo ""
echo "Health Checks:"
echo "□ All services are running"
echo "□ Database is accessible"
echo "□ Health endpoints return 200 OK"
echo "□ No errors in logs"
echo "□ Metrics look normal"
echo ""
read -p "Is $MAINTENANCE_REGION healthy? (yes/no): " healthy
if [ "$healthy" != "yes" ]; then
    echo ""
    echo "⚠️  $MAINTENANCE_REGION is not healthy. Traffic will remain on $ACTIVE_REGION."
    echo "Fix the issues and run this script again to restore traffic."
    exit 0
fi

# Step 4: Gradually restore traffic
echo ""
echo "Step 4: Restore Traffic"
echo "======================"
echo ""
echo "Gradually restoring traffic to $MAINTENANCE_REGION..."

# Phase 1: 10% traffic
echo ""
echo "Phase 1: Sending 10% traffic to $MAINTENANCE_REGION"
if [ "$MAINTENANCE_REGION" = "region-a" ]; then
    $CLI switch proportional region-a:10 region-b:90 \
        --redis-addr "$REDIS_ADDR" \
        --reason "Post-maintenance: Testing with 10% traffic" \
        --operator "$OPERATOR"
else
    $CLI switch proportional region-a:90 region-b:10 \
        --redis-addr "$REDIS_ADDR" \
        --reason "Post-maintenance: Testing with 10% traffic" \
        --operator "$OPERATOR"
fi

echo ""
echo "✓ 10% traffic restored. Monitor for 2-3 minutes..."
read -p "Press Enter to continue..."

# Phase 2: 50% traffic
echo ""
echo "Phase 2: Balancing traffic 50:50"
$CLI switch proportional region-a:50 region-b:50 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Post-maintenance: Balanced load" \
    --operator "$OPERATOR"

echo ""
echo "✓ Traffic balanced!"
echo ""

# Final status
echo "Final Status:"
$CLI status --redis-addr "$REDIS_ADDR"
echo ""

echo "Maintenance Complete!"
echo "===================="
echo ""
echo "Post-Maintenance Tasks:"
echo "□ Continue monitoring for 24 hours"
echo "□ Document maintenance in runbook"
echo "□ Update change log"
echo "□ Notify team of completion"
echo ""
echo "View event history:"
echo "  $CLI events --redis-addr $REDIS_ADDR"
