#!/bin/bash

# Example: Gradual Migration from Region-B to Region-A
# This script demonstrates a safe, gradual migration strategy

set -e

CLI="./bin/traffic-cli"
REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
OPERATOR="${OPERATOR:-ops-team}"

echo "Gradual Migration: Region-B → Region-A"
echo "======================================"
echo ""

# Phase 1: Start with 70:30
echo "Phase 1: Shifting to 70:30 (region-a:region-b)"
$CLI switch proportional region-a:70 region-b:30 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration phase 1: Initial shift" \
    --operator "$OPERATOR"

echo ""
echo "✓ Phase 1 complete. Monitor metrics for 5 minutes..."
echo "  Check: Error rates, latency, CPU/memory usage"
echo ""
read -p "Press Enter when ready to continue to Phase 2..."

# Phase 2: Increase to 85:15
echo ""
echo "Phase 2: Shifting to 85:15 (region-a:region-b)"
$CLI switch proportional region-a:85 region-b:15 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration phase 2: Increased load" \
    --operator "$OPERATOR"

echo ""
echo "✓ Phase 2 complete. Monitor metrics for 5 minutes..."
echo ""
read -p "Press Enter when ready to continue to Phase 3..."

# Phase 3: Increase to 95:5
echo ""
echo "Phase 3: Shifting to 95:5 (region-a:region-b)"
$CLI switch proportional region-a:95 region-b:5 \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration phase 3: Near completion" \
    --operator "$OPERATOR"

echo ""
echo "✓ Phase 3 complete. Monitor metrics for 5 minutes..."
echo ""
read -p "Press Enter when ready to complete the migration..."

# Phase 4: Complete migration
echo ""
echo "Phase 4: Complete migration to region-a (100:0)"
$CLI switch full region-a \
    --redis-addr "$REDIS_ADDR" \
    --reason "Migration complete: All traffic to region-a" \
    --operator "$OPERATOR"

echo ""
echo "✓ Migration complete!"
echo ""
$CLI status --redis-addr "$REDIS_ADDR"
echo ""
echo "Next steps:"
echo "1. Continue monitoring for 24 hours"
echo "2. Verify all metrics are stable"
echo "3. Document the migration in your runbook"
echo "4. Consider decommissioning region-b if no longer needed"
