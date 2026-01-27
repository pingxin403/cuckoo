#!/bin/bash

# Test IM Chat System Infrastructure
# This script verifies all infrastructure components are running correctly

echo "=========================================="
echo "Testing IM Chat System Infrastructure"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to print test results
test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

# ===== Test etcd Cluster =====
echo "Testing etcd cluster..."
echo "----------------------------------------"

# Test etcd-1
if docker exec im-etcd-1 etcdctl endpoint health 2>/dev/null | grep -q "is healthy"; then
    test_result 0 "etcd-1 is healthy"
else
    test_result 1 "etcd-1 is not healthy"
fi

# Test etcd-2
if docker exec im-etcd-2 etcdctl endpoint health 2>/dev/null | grep -q "is healthy"; then
    test_result 0 "etcd-2 is healthy"
else
    test_result 1 "etcd-2 is not healthy"
fi

# Test etcd-3
if docker exec im-etcd-3 etcdctl endpoint health 2>/dev/null | grep -q "is healthy"; then
    test_result 0 "etcd-3 is healthy"
else
    test_result 1 "etcd-3 is not healthy"
fi

# Test etcd cluster member list
MEMBER_COUNT=$(docker exec im-etcd-1 etcdctl member list 2>/dev/null | wc -l)
if [ "$MEMBER_COUNT" -eq 3 ]; then
    test_result 0 "etcd cluster has 3 members"
else
    test_result 1 "etcd cluster member count is $MEMBER_COUNT (expected 3)"
fi

# Test etcd write/read
docker exec im-etcd-1 etcdctl put /test/key "test-value" >/dev/null 2>&1
ETCD_VALUE=$(docker exec im-etcd-1 etcdctl get /test/key --print-value-only 2>/dev/null)
if [ "$ETCD_VALUE" = "test-value" ]; then
    test_result 0 "etcd write/read operations work"
else
    test_result 1 "etcd write/read operations failed"
fi
docker exec im-etcd-1 etcdctl del /test/key >/dev/null 2>&1

echo ""

# ===== Test MySQL =====
echo "Testing IM MySQL database..."
echo "----------------------------------------"

# Test MySQL connection
if docker exec im-mysql mysqladmin ping -h localhost -u root -pim_root_password 2>/dev/null | grep -q "mysqld is alive"; then
    test_result 0 "MySQL is running"
else
    test_result 1 "MySQL is not running"
fi

# Test database exists
if docker exec im-mysql mysql -u root -pim_root_password -e "SHOW DATABASES LIKE 'im_chat';" 2>/dev/null | grep -q "im_chat"; then
    test_result 0 "im_chat database exists"
else
    test_result 1 "im_chat database does not exist"
fi

# Test tables exist
TABLES=$(docker exec im-mysql mysql -u root -pim_root_password -D im_chat -e "SHOW TABLES;" 2>/dev/null | tail -n +2 | wc -l)
if [ "$TABLES" -ge 5 ]; then
    test_result 0 "Database tables created ($TABLES tables)"
else
    test_result 1 "Database tables not created (found $TABLES tables, expected at least 5)"
fi

# Test user permissions
if docker exec im-mysql mysql -u im_service -pim_service_password -D im_chat -e "SELECT 1;" >/dev/null 2>&1; then
    test_result 0 "im_service user has database access"
else
    test_result 1 "im_service user cannot access database"
fi

echo ""

# ===== Test Redis =====
echo "Testing IM Redis..."
echo "----------------------------------------"

# Test Redis connection
if docker exec im-redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
    test_result 0 "Redis is running"
else
    test_result 1 "Redis is not running"
fi

# Test Redis write/read
docker exec im-redis redis-cli SET test:key "test-value" >/dev/null 2>&1
REDIS_VALUE=$(docker exec im-redis redis-cli GET test:key 2>/dev/null)
if [ "$REDIS_VALUE" = "test-value" ]; then
    test_result 0 "Redis write/read operations work"
else
    test_result 1 "Redis write/read operations failed"
fi
docker exec im-redis redis-cli DEL test:key >/dev/null 2>&1

# Test Redis persistence (AOF)
if docker exec im-redis redis-cli CONFIG GET appendonly 2>/dev/null | grep -q "yes"; then
    test_result 0 "Redis AOF persistence is enabled"
else
    test_result 1 "Redis AOF persistence is not enabled"
fi

echo ""

# ===== Test Kafka Cluster (KRaft mode) =====
echo "Testing Kafka cluster (KRaft mode)..."
echo "----------------------------------------"

# Test Kafka brokers
for i in 1 2 3; do
    if docker exec im-kafka-$i kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1; then
        test_result 0 "Kafka broker $i is running"
    else
        test_result 1 "Kafka broker $i is not running"
    fi
done

# Test Kafka cluster metadata (use kafka-cluster command instead)
CLUSTER_INFO=$(docker exec im-kafka-1 kafka-cluster cluster-id --bootstrap-server localhost:9092 2>/dev/null)
if [ -n "$CLUSTER_INFO" ]; then
    test_result 0 "Kafka cluster metadata is accessible"
else
    test_result 1 "Kafka cluster metadata is not accessible (non-critical)"
fi

# Test Kafka topics
TOPICS=$(docker exec im-kafka-1 kafka-topics --list --bootstrap-server localhost:9092 2>/dev/null)

if echo "$TOPICS" | grep -q "group_msg"; then
    test_result 0 "Kafka topic 'group_msg' exists"
else
    test_result 1 "Kafka topic 'group_msg' does not exist"
fi

if echo "$TOPICS" | grep -q "offline_msg"; then
    test_result 0 "Kafka topic 'offline_msg' exists"
else
    test_result 1 "Kafka topic 'offline_msg' does not exist"
fi

if echo "$TOPICS" | grep -q "membership_change"; then
    test_result 0 "Kafka topic 'membership_change' exists"
else
    test_result 1 "Kafka topic 'membership_change' does not exist"
fi

# Test Kafka produce/consume
TEST_MESSAGE="test-message-$(date +%s)"
# Produce message
if echo "$TEST_MESSAGE" | docker exec -i im-kafka-1 kafka-console-producer --bootstrap-server localhost:9092 --topic group_msg 2>/dev/null; then
    # Give Kafka time to replicate
    sleep 3
    # Consume message (use gtimeout on macOS, timeout on Linux)
    TIMEOUT_CMD="timeout"
    if command -v gtimeout >/dev/null 2>&1; then
        TIMEOUT_CMD="gtimeout"
    fi
    CONSUMED=$(${TIMEOUT_CMD} 10 docker exec im-kafka-1 kafka-console-consumer --bootstrap-server localhost:9092 --topic group_msg --from-beginning --max-messages 1 --timeout-ms 8000 2>/dev/null | grep -F "$TEST_MESSAGE" 2>/dev/null || true)
    if [ -n "$CONSUMED" ]; then
        test_result 0 "Kafka produce/consume operations work"
    else
        # Try without timeout command as fallback
        CONSUMED=$(docker exec im-kafka-1 sh -c "kafka-console-consumer --bootstrap-server localhost:9092 --topic group_msg --from-beginning --max-messages 1 --timeout-ms 5000 2>/dev/null | grep -F '$TEST_MESSAGE' || true")
        if [ -n "$CONSUMED" ]; then
            test_result 0 "Kafka produce/consume operations work"
        else
            test_result 1 "Kafka produce/consume operations failed (message not consumed - this may be a timing issue)"
        fi
    fi
else
    test_result 1 "Kafka produce/consume operations failed (produce failed)"
fi

echo ""

# ===== Summary =====
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All infrastructure tests passed!${NC}"
    echo ""
    echo "Infrastructure endpoints:"
    echo "  - etcd:   localhost:2379, localhost:2381, localhost:2383"
    echo "  - MySQL:  localhost:3307 (user: im_service, password: im_service_password, database: im_chat)"
    echo "  - Redis:  localhost:6380"
    echo "  - Kafka (KRaft):  localhost:9093, localhost:9094, localhost:9095"
    exit 0
else
    echo -e "${RED}Some infrastructure tests failed. Please check the logs.${NC}"
    exit 1
fi
