#!/bin/bash
# Validation script for capacity monitoring alerts
# Verifies Prometheus alert rules syntax and configuration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ALERTS_FILE="${SCRIPT_DIR}/prometheus-alerts.yml"

echo "=== Capacity Monitoring Alerts Validation ==="
echo ""

# Check if promtool is available
if ! command -v promtool &> /dev/null; then
    echo "⚠️  Warning: promtool not found. Skipping syntax validation."
    echo "   Install Prometheus to get promtool for validation."
    echo ""
else
    echo "✓ promtool found"
    echo ""
    
    # Validate alert rules syntax
    echo "Validating Prometheus alert rules syntax..."
    if promtool check rules "$ALERTS_FILE"; then
        echo "✓ Alert rules syntax is valid"
    else
        echo "✗ Alert rules syntax validation failed"
        exit 1
    fi
    echo ""
fi

# Check if capacity alerts group exists
echo "Checking capacity_alerts group..."
if grep -q "name: capacity_alerts" "$ALERTS_FILE"; then
    echo "✓ capacity_alerts group found"
else
    echo "✗ capacity_alerts group not found"
    exit 1
fi
echo ""

# Check for required capacity alerts
echo "Checking required capacity alerts..."
REQUIRED_ALERTS=(
    "HighResourceUsage"
    "CriticalResourceUsage"
    "CapacityFullSoon"
    "CapacityFullImminently"
    "HighCapacityCollectionErrorRate"
    "HighMySQLStorageGrowth"
    "HighKafkaStorageGrowth"
    "HighNetworkBandwidthUsage"
)

for alert in "${REQUIRED_ALERTS[@]}"; do
    if grep -q "alert: $alert" "$ALERTS_FILE"; then
        echo "  ✓ $alert"
    else
        echo "  ✗ $alert not found"
        exit 1
    fi
done
echo ""

# Check alert thresholds
echo "Checking alert thresholds..."
echo "  - HighResourceUsage: >= 80%"
if grep -A 1 "alert: HighResourceUsage" "$ALERTS_FILE" | grep -q "capacity_resource_usage_percent >= 80"; then
    echo "    ✓ Threshold configured correctly"
else
    echo "    ✗ Threshold not configured correctly"
    exit 1
fi

echo "  - CriticalResourceUsage: >= 90%"
if grep -A 1 "alert: CriticalResourceUsage" "$ALERTS_FILE" | grep -q "capacity_resource_usage_percent >= 90"; then
    echo "    ✓ Threshold configured correctly"
else
    echo "    ✗ Threshold not configured correctly"
    exit 1
fi

echo "  - CapacityFullSoon: <= 7 days"
if grep -A 1 "alert: CapacityFullSoon" "$ALERTS_FILE" | grep -q "capacity_forecast_days_until_full <= 7"; then
    echo "    ✓ Threshold configured correctly"
else
    echo "    ✗ Threshold not configured correctly"
    exit 1
fi

echo "  - CapacityFullImminently: <= 3 days"
if grep -A 1 "alert: CapacityFullImminently" "$ALERTS_FILE" | grep -q "capacity_forecast_days_until_full <= 3"; then
    echo "    ✓ Threshold configured correctly"
else
    echo "    ✗ Threshold not configured correctly"
    exit 1
fi
echo ""

# Check alert labels
echo "Checking alert labels..."
if grep -A 5 "alert: HighResourceUsage" "$ALERTS_FILE" | grep -q "severity: warning"; then
    echo "  ✓ Severity labels configured"
else
    echo "  ✗ Severity labels missing"
    exit 1
fi

if grep -A 5 "alert: HighResourceUsage" "$ALERTS_FILE" | grep -q "service: capacity-monitor"; then
    echo "  ✓ Service labels configured"
else
    echo "  ✗ Service labels missing"
    exit 1
fi
echo ""

# Check alert annotations
echo "Checking alert annotations..."
REQUIRED_ANNOTATIONS=(
    "summary"
    "description"
    "runbook_url"
    "dashboard_url"
    "action"
)

for annotation in "${REQUIRED_ANNOTATIONS[@]}"; do
    if grep -A 15 "alert: HighResourceUsage" "$ALERTS_FILE" | grep -q "$annotation:"; then
        echo "  ✓ $annotation annotation present"
    else
        echo "  ✗ $annotation annotation missing"
        exit 1
    fi
done
echo ""

# Summary
echo "=== Validation Summary ==="
echo "✓ All capacity monitoring alerts are properly configured"
echo "✓ Alert thresholds match requirements (80% warning, 90% critical)"
echo "✓ Forecast alerts configured (7 days warning, 3 days critical)"
echo "✓ Resource-specific alerts configured (MySQL, Kafka, Network)"
echo ""
echo "Next steps:"
echo "1. Ensure Prometheus is configured to load these alert rules"
echo "2. Configure Alertmanager to route capacity alerts appropriately"
echo "3. Test alerts by simulating high resource usage"
echo "4. Review Grafana dashboards: capacity-usage-trends and capacity-forecast"
echo ""
