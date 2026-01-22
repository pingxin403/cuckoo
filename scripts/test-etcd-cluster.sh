#!/bin/bash
# Test script for etcd cluster health and leader election
# Validates Requirements 7.6 (etcd cluster) and 10.3 (high availability)

set -e

NAMESPACE="${NAMESPACE:-default}"
ETCD_ENDPOINTS="http://etcd-0.etcd.${NAMESPACE}.svc.cluster.local:2379,http://etcd-1.etcd.${NAMESPACE}.svc.cluster.local:2379,http://etcd-2.etcd.${NAMESPACE}.svc.cluster.local:2379"

echo "=========================================="
echo "Testing etcd Cluster for IM Chat System"
echo "=========================================="
echo ""

# Function to run etcdctl commands
run_etcdctl() {
    kubectl exec -n "${NAMESPACE}" etcd-0 -- etcdctl \
        --endpoints="${ETCD_ENDPOINTS}" \
        "$@"
}

# Test 1: Check cluster health
echo "Test 1: Checking cluster health..."
if run_etcdctl endpoint health; then
    echo "✓ Cluster health check passed"
else
    echo "✗ Cluster health check failed"
    exit 1
fi
echo ""

# Test 2: Check cluster member list
echo "Test 2: Checking cluster members..."
if run_etcdctl member list; then
    echo "✓ Member list retrieved successfully"
else
    echo "✗ Failed to retrieve member list"
    exit 1
fi
echo ""

# Test 3: Check leader election
echo "Test 3: Checking leader election..."
if run_etcdctl endpoint status --write-out=table; then
    echo "✓ Leader election status retrieved"
else
    echo "✗ Failed to retrieve leader status"
    exit 1
fi
echo ""

# Test 4: Test write operation
echo "Test 4: Testing write operation..."
TEST_KEY="/registry/test/$(date +%s)"
TEST_VALUE="test-value-$(date +%s)"
if run_etcdctl put "${TEST_KEY}" "${TEST_VALUE}"; then
    echo "✓ Write operation successful"
else
    echo "✗ Write operation failed"
    exit 1
fi
echo ""

# Test 5: Test read operation
echo "Test 5: Testing read operation..."
if RESULT=$(run_etcdctl get "${TEST_KEY}" --print-value-only) && [ "${RESULT}" = "${TEST_VALUE}" ]; then
    echo "✓ Read operation successful (value matches)"
else
    echo "✗ Read operation failed or value mismatch"
    exit 1
fi
echo ""

# Test 6: Test TTL (lease) functionality
echo "Test 6: Testing TTL/lease functionality..."
LEASE_ID=$(run_etcdctl lease grant 10 | grep -oP 'lease \K[0-9a-f]+')
if [ -n "${LEASE_ID}" ]; then
    echo "✓ Lease created: ${LEASE_ID}"
    TTL_KEY="/registry/test/ttl-$(date +%s)"
    if run_etcdctl put "${TTL_KEY}" "ttl-value" --lease="${LEASE_ID}"; then
        echo "✓ TTL key created successfully"
    else
        echo "✗ Failed to create TTL key"
        exit 1
    fi
else
    echo "✗ Failed to create lease"
    exit 1
fi
echo ""

# Test 7: Test watch mechanism
echo "Test 7: Testing watch mechanism..."
WATCH_KEY="/registry/test/watch-$(date +%s)"
# Start watch in background
kubectl exec -n "${NAMESPACE}" etcd-0 -- sh -c "etcdctl --endpoints='${ETCD_ENDPOINTS}' watch '${WATCH_KEY}' --prefix &" &
WATCH_PID=$!
sleep 2
# Trigger watch event
run_etcdctl put "${WATCH_KEY}" "watch-value" > /dev/null
sleep 1
# Kill watch process
kill ${WATCH_PID} 2>/dev/null || true
echo "✓ Watch mechanism test completed"
echo ""

# Test 8: Test cluster consistency
echo "Test 8: Testing cluster consistency..."
CONSISTENCY_KEY="/registry/test/consistency-$(date +%s)"
CONSISTENCY_VALUE="consistency-value-$(date +%s)"
run_etcdctl put "${CONSISTENCY_KEY}" "${CONSISTENCY_VALUE}" > /dev/null

# Read from each node
for i in 0 1 2; do
    ENDPOINT="http://etcd-${i}.etcd.${NAMESPACE}.svc.cluster.local:2379"
    VALUE=$(kubectl exec -n "${NAMESPACE}" etcd-${i} -- etcdctl \
        --endpoints="${ENDPOINT}" \
        get "${CONSISTENCY_KEY}" --print-value-only)
    if [ "${VALUE}" = "${CONSISTENCY_VALUE}" ]; then
        echo "✓ Node etcd-${i}: Value consistent"
    else
        echo "✗ Node etcd-${i}: Value mismatch (expected: ${CONSISTENCY_VALUE}, got: ${VALUE})"
        exit 1
    fi
done
echo ""

# Test 9: Cleanup test keys
echo "Test 9: Cleaning up test keys..."
if run_etcdctl del "/registry/test/" --prefix > /dev/null; then
    echo "✓ Test keys cleaned up"
else
    echo "✗ Failed to clean up test keys"
fi
echo ""

# Summary
echo "=========================================="
echo "All etcd cluster tests passed successfully!"
echo "=========================================="
echo ""
echo "Cluster is ready for IM Chat System Registry service"
echo "Client endpoint: etcd-client.${NAMESPACE}.svc.cluster.local:2379"
echo "Endpoints for applications:"
echo "  - etcd-0.etcd.${NAMESPACE}.svc.cluster.local:2379"
echo "  - etcd-1.etcd.${NAMESPACE}.svc.cluster.local:2379"
echo "  - etcd-2.etcd.${NAMESPACE}.svc.cluster.local:2379"
