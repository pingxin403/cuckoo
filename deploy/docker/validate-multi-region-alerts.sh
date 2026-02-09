#!/bin/bash

# Validation script for multi-region Prometheus alerting rules
# This script validates the syntax and structure of the alert rules

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ALERTS_FILE="${SCRIPT_DIR}/prometheus-multi-region-alerts.yml"

echo "=================================================="
echo "Multi-Region Alert Rules Validation"
echo "=================================================="
echo ""

# Check if promtool is available
if ! command -v promtool &> /dev/null; then
    echo "❌ promtool not found. Please install Prometheus to use promtool."
    echo ""
    echo "Installation options:"
    echo "  - macOS: brew install prometheus"
    echo "  - Linux: Download from https://prometheus.io/download/"
    echo "  - Docker: docker run --rm -v ${SCRIPT_DIR}:/config prom/prometheus:latest promtool check rules /config/prometheus-multi-region-alerts.yml"
    exit 1
fi

# Check if alerts file exists
if [ ! -f "${ALERTS_FILE}" ]; then
    echo "❌ Alert rules file not found: ${ALERTS_FILE}"
    exit 1
fi

echo "📋 Validating alert rules file: ${ALERTS_FILE}"
echo ""

# Validate alert rules syntax
echo "1. Checking alert rules syntax..."
if promtool check rules "${ALERTS_FILE}"; then
    echo "✅ Alert rules syntax is valid"
else
    echo "❌ Alert rules syntax validation failed"
    exit 1
fi
echo ""

# Count alert groups and rules
echo "2. Analyzing alert rules..."
ALERT_GROUPS=$(grep -c "^  - name:" "${ALERTS_FILE}" || true)
TOTAL_ALERTS=$(grep -c "^      - alert:" "${ALERTS_FILE}" || true)

echo "   Alert Groups: ${ALERT_GROUPS}"
echo "   Total Alerts: ${TOTAL_ALERTS}"
echo ""

# List alert groups
echo "3. Alert Groups:"
grep "^  - name:" "${ALERTS_FILE}" | sed 's/  - name: /   - /' || true
echo ""

# Check for required fields in each alert
echo "4. Checking alert structure..."
MISSING_FIELDS=0

# Check for alerts without severity
if grep -A 10 "^      - alert:" "${ALERTS_FILE}" | grep -L "severity:" > /dev/null 2>&1; then
    echo "   ⚠️  Warning: Some alerts may be missing severity labels"
    MISSING_FIELDS=$((MISSING_FIELDS + 1))
fi

# Check for alerts without annotations
if grep -A 10 "^      - alert:" "${ALERTS_FILE}" | grep -L "annotations:" > /dev/null 2>&1; then
    echo "   ⚠️  Warning: Some alerts may be missing annotations"
    MISSING_FIELDS=$((MISSING_FIELDS + 1))
fi

# Check for alerts without summary
if grep -A 15 "^      - alert:" "${ALERTS_FILE}" | grep -L "summary:" > /dev/null 2>&1; then
    echo "   ⚠️  Warning: Some alerts may be missing summary annotations"
    MISSING_FIELDS=$((MISSING_FIELDS + 1))
fi

# Check for alerts without description
if grep -A 15 "^      - alert:" "${ALERTS_FILE}" | grep -L "description:" > /dev/null 2>&1; then
    echo "   ⚠️  Warning: Some alerts may be missing description annotations"
    MISSING_FIELDS=$((MISSING_FIELDS + 1))
fi

if [ ${MISSING_FIELDS} -eq 0 ]; then
    echo "   ✅ All alerts have required fields"
else
    echo "   ⚠️  ${MISSING_FIELDS} potential issues found (warnings only)"
fi
echo ""

# List all alerts by severity
echo "5. Alerts by Severity:"
echo ""
echo "   Critical Alerts:"
grep -A 5 "severity: critical" "${ALERTS_FILE}" | grep "alert:" | sed 's/.*alert: /      - /' | sort -u || echo "      None"
echo ""
echo "   Warning Alerts:"
grep -A 5 "severity: warning" "${ALERTS_FILE}" | grep "alert:" | sed 's/.*alert: /      - /' | sort -u || echo "      None"
echo ""

# Check requirements coverage
echo "6. Requirements Coverage:"
echo ""
REQUIREMENTS=$(grep "requirement:" "${ALERTS_FILE}" | sed 's/.*requirement: "\(.*\)"/\1/' | sort -u)
for req in ${REQUIREMENTS}; do
    COUNT=$(grep -c "requirement: \"${req}\"" "${ALERTS_FILE}" || true)
    echo "   - Requirement ${req}: ${COUNT} alerts"
done
echo ""

# Summary
echo "=================================================="
echo "Validation Summary"
echo "=================================================="
echo ""
echo "✅ Alert rules file is valid"
echo "   - ${ALERT_GROUPS} alert groups"
echo "   - ${TOTAL_ALERTS} total alerts"
echo "   - All syntax checks passed"
echo ""
echo "Next steps:"
echo "1. Review alert thresholds and durations"
echo "2. Update prometheus.yml to include this file"
echo "3. Reload Prometheus configuration"
echo "4. Test alerts with sample metrics"
echo ""
echo "For more information, see:"
echo "  - MULTI_REGION_ALERTING.md"
echo "  - metrics/README.md"
echo ""
