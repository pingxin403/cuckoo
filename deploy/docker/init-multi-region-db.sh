#!/bin/bash
# Multi-Region Database Initialization Script
# This script applies schema changes for multi-region support to local MySQL
# Task 18.2: 创建数据库初始化脚本

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MYSQL_HOST="${MYSQL_HOST:-localhost}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-im_service}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-im_service_password}"
MYSQL_DATABASE="${MYSQL_DATABASE:-im_chat}"
MIGRATION_FILE="${MIGRATION_FILE:-../../apps/im-service/migrations/004_offline_messages_partitioning.sql}"

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if MySQL is accessible
check_mysql_connection() {
    print_info "Checking MySQL connection..."
    if mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" -e "SELECT 1;" > /dev/null 2>&1; then
        print_success "MySQL connection successful"
        return 0
    else
        print_error "Cannot connect to MySQL at ${MYSQL_HOST}:${MYSQL_PORT}"
        print_error "Please ensure MySQL is running and credentials are correct"
        return 1
    fi
}

# Function to check if database exists
check_database_exists() {
    print_info "Checking if database '${MYSQL_DATABASE}' exists..."
    if mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" -e "USE ${MYSQL_DATABASE};" > /dev/null 2>&1; then
        print_success "Database '${MYSQL_DATABASE}' exists"
        return 0
    else
        print_error "Database '${MYSQL_DATABASE}' does not exist"
        return 1
    fi
}

# Function to apply schema migration
apply_schema_migration() {
    print_info "Applying schema migration from ${MIGRATION_FILE}..."
    
    if [ ! -f "${MIGRATION_FILE}" ]; then
        print_error "Migration file not found: ${MIGRATION_FILE}"
        return 1
    fi
    
    if mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < "${MIGRATION_FILE}"; then
        print_success "Schema migration applied successfully"
        return 0
    else
        print_error "Failed to apply schema migration"
        return 1
    fi
}

# Function to verify table structure
verify_table_structure() {
    print_info "Verifying offline_messages table structure..."
    
    # Check if table exists
    TABLE_EXISTS=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${MYSQL_DATABASE}' AND table_name='offline_messages';")
    
    if [ "$TABLE_EXISTS" -eq 0 ]; then
        print_error "Table 'offline_messages' does not exist"
        return 1
    fi
    
    print_success "Table 'offline_messages' exists"
    
    # Verify multi-region fields
    print_info "Checking multi-region fields..."
    
    FIELDS=("region_id" "global_id" "sync_status" "synced_at")
    for field in "${FIELDS[@]}"; do
        FIELD_EXISTS=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
            -sN -e "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='${MYSQL_DATABASE}' AND table_name='offline_messages' AND column_name='${field}';")
        
        if [ "$FIELD_EXISTS" -eq 1 ]; then
            print_success "  ✓ Field '${field}' exists"
        else
            print_error "  ✗ Field '${field}' is missing"
            return 1
        fi
    done
    
    return 0
}

# Function to verify indexes
verify_indexes() {
    print_info "Verifying multi-region indexes..."
    
    INDEXES=("idx_region_sync_status" "idx_global_id")
    for index in "${INDEXES[@]}"; do
        INDEX_EXISTS=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
            -sN -e "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema='${MYSQL_DATABASE}' AND table_name='offline_messages' AND index_name='${index}';")
        
        if [ "$INDEX_EXISTS" -gt 0 ]; then
            print_success "  ✓ Index '${index}' exists"
        else
            print_error "  ✗ Index '${index}' is missing"
            return 1
        fi
    done
    
    return 0
}

# Function to verify partitioning
verify_partitioning() {
    print_info "Verifying table partitioning..."
    
    PARTITION_COUNT=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM information_schema.partitions WHERE table_schema='${MYSQL_DATABASE}' AND table_name='offline_messages';")
    
    if [ "$PARTITION_COUNT" -eq 16 ]; then
        print_success "Table has 16 partitions (as expected)"
    else
        print_warning "Table has ${PARTITION_COUNT} partitions (expected 16)"
    fi
    
    return 0
}

# Function to display table structure
display_table_structure() {
    print_info "Table structure:"
    echo ""
    mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -e "DESCRIBE offline_messages;" | sed 's/^/  /'
    echo ""
}

# Function to display indexes
display_indexes() {
    print_info "Table indexes:"
    echo ""
    mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -e "SHOW INDEX FROM offline_messages;" | sed 's/^/  /'
    echo ""
}

# Function to verify test data
verify_test_data() {
    print_info "Verifying test data..."
    
    ROW_COUNT=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM offline_messages;")
    
    if [ "$ROW_COUNT" -gt 0 ]; then
        print_success "Test data inserted: ${ROW_COUNT} rows"
        
        # Display sample data
        print_info "Sample multi-region data:"
        echo ""
        mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
            -e "SELECT msg_id, user_id, region_id, global_id, sync_status FROM offline_messages LIMIT 5;" | sed 's/^/  /'
        echo ""
    else
        print_warning "No test data found in table"
    fi
    
    return 0
}

# Function to test multi-region queries
test_multi_region_queries() {
    print_info "Testing multi-region query patterns..."
    
    # Test 1: Query by region and sync status
    print_info "Test 1: Query pending messages for region-a"
    PENDING_COUNT=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM offline_messages WHERE region_id='region-a' AND sync_status='pending';")
    print_success "  Found ${PENDING_COUNT} pending messages in region-a"
    
    # Test 2: Query by global_id
    print_info "Test 2: Query by global_id"
    GLOBAL_ID_COUNT=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM offline_messages WHERE global_id LIKE 'region-a-%';")
    print_success "  Found ${GLOBAL_ID_COUNT} messages with region-a global_id"
    
    # Test 3: Query conflict messages
    print_info "Test 3: Query conflict messages"
    CONFLICT_COUNT=$(mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -sN -e "SELECT COUNT(*) FROM offline_messages WHERE sync_status='conflict';")
    print_success "  Found ${CONFLICT_COUNT} conflict messages"
    
    # Test 4: Explain query performance
    print_info "Test 4: Query performance analysis"
    echo ""
    mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" \
        -e "EXPLAIN SELECT * FROM offline_messages WHERE region_id='region-a' AND sync_status='pending' ORDER BY created_at LIMIT 1000;" | sed 's/^/  /'
    echo ""
    
    return 0
}

# Function to insert additional test data
insert_additional_test_data() {
    print_info "Inserting additional test data for multi-region validation..."
    
    mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" <<-EOSQL
        -- Insert messages from region-a
        INSERT INTO offline_messages (
            msg_id, user_id, sender_id, conversation_id, conversation_type,
            content, sequence_number, timestamp, expires_at,
            region_id, global_id, sync_status
        ) VALUES
        (
            UUID(),
            'user004',
            'user005',
            'private:user004_user005',
            'private',
            'Message from region-a (pending)',
            1,
            UNIX_TIMESTAMP() * 1000,
            DATE_ADD(NOW(), INTERVAL 7 DAY),
            'region-a',
            CONCAT('region-a-', UNIX_TIMESTAMP(), '-0-', FLOOR(RAND() * 1000)),
            'pending'
        ),
        (
            UUID(),
            'user005',
            'user004',
            'private:user004_user005',
            'private',
            'Message from region-b (synced)',
            2,
            UNIX_TIMESTAMP() * 1000,
            DATE_ADD(NOW(), INTERVAL 7 DAY),
            'region-b',
            CONCAT('region-b-', UNIX_TIMESTAMP(), '-0-', FLOOR(RAND() * 1000)),
            'synced'
        ),
        (
            UUID(),
            'user006',
            'user007',
            'group:team_chat',
            'group',
            'Group message from region-a',
            1,
            UNIX_TIMESTAMP() * 1000,
            DATE_ADD(NOW(), INTERVAL 7 DAY),
            'region-a',
            CONCAT('region-a-', UNIX_TIMESTAMP(), '-0-', FLOOR(RAND() * 1000)),
            'synced'
        );
EOSQL
    
    if [ $? -eq 0 ]; then
        print_success "Additional test data inserted successfully"
        return 0
    else
        print_error "Failed to insert additional test data"
        return 1
    fi
}

# Main execution
main() {
    echo ""
    echo "=========================================="
    echo "  Multi-Region Database Initialization"
    echo "=========================================="
    echo ""
    
    # Step 1: Check MySQL connection
    if ! check_mysql_connection; then
        exit 1
    fi
    
    # Step 2: Check database exists
    if ! check_database_exists; then
        exit 1
    fi
    
    # Step 3: Apply schema migration
    if ! apply_schema_migration; then
        exit 1
    fi
    
    # Step 4: Verify table structure
    if ! verify_table_structure; then
        exit 1
    fi
    
    # Step 5: Verify indexes
    if ! verify_indexes; then
        exit 1
    fi
    
    # Step 6: Verify partitioning
    verify_partitioning
    
    # Step 7: Display table structure
    display_table_structure
    
    # Step 8: Display indexes
    display_indexes
    
    # Step 9: Verify test data
    verify_test_data
    
    # Step 10: Insert additional test data
    insert_additional_test_data
    
    # Step 11: Test multi-region queries
    test_multi_region_queries
    
    echo ""
    echo "=========================================="
    print_success "Multi-Region Database Initialization Complete!"
    echo "=========================================="
    echo ""
    
    print_info "Next steps:"
    echo "  1. Verify the schema changes: mysql -u ${MYSQL_USER} -p ${MYSQL_DATABASE}"
    echo "  2. Run integration tests: cd apps/im-service && go test ./integration_test/..."
    echo "  3. Start multi-region services: docker compose -f deploy/docker/docker-compose.services.yml up -d"
    echo ""
}

# Run main function
main "$@"
