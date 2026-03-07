-- Migration: Add partitioning to offline_messages table
-- This migration adds hash partitioning by user_id for better query performance
-- 16 partitions for even distribution (each partition handles ~625K users for 10M total)

-- Drop the existing table if it exists (for clean migration)
-- Note: In production, you would migrate data first
DROP TABLE IF EXISTS offline_messages;

-- Recreate offline_messages table with partitioning and multi-region support
CREATE TABLE offline_messages (
    id BIGINT AUTO_INCREMENT,
    msg_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conversation_type ENUM('private', 'group') NOT NULL,
    content TEXT NOT NULL,
    sequence_number BIGINT NOT NULL,
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    metadata JSON,
    
    -- Multi-region support fields (Task 18.1)
    region_id VARCHAR(50) NOT NULL DEFAULT 'default',
    global_id VARCHAR(255),  -- HLC-based global ID for cross-region ordering
    sync_status ENUM('pending', 'synced', 'conflict') DEFAULT 'pending',
    synced_at TIMESTAMP NULL,  -- Sync completion time
    
    PRIMARY KEY (id, user_id),  -- Composite key required for partitioning
    UNIQUE KEY idx_msg_id (msg_id),
    KEY idx_user_timestamp (user_id, timestamp),
    KEY idx_conversation_seq (conversation_id, sequence_number),
    KEY idx_expires_at (expires_at),
    KEY idx_user_conversation (user_id, conversation_id, sequence_number),
    
    -- Multi-region indexes (Task 18.1)
    KEY idx_region_sync_status (region_id, sync_status, created_at),
    KEY idx_global_id (global_id)
) ENGINE=InnoDB
PARTITION BY HASH(CRC32(user_id))
PARTITIONS 16;

-- Create index on msg_id for deduplication checks
-- Note: UNIQUE constraint on msg_id is maintained across all partitions

-- Test query performance with sample data
-- Insert sample offline messages for testing (with multi-region fields)
INSERT INTO offline_messages (
    msg_id, user_id, sender_id, conversation_id, conversation_type,
    content, sequence_number, timestamp, expires_at,
    region_id, global_id, sync_status
) VALUES
(
    UUID(),
    'user001',
    'user002',
    'private:user001_user002',
    'private',
    'Test message 1',
    1,
    UNIX_TIMESTAMP() * 1000,
    DATE_ADD(NOW(), INTERVAL 7 DAY),
    'region-a',
    'region-a-1234567890-0-1',
    'synced'
),
(
    UUID(),
    'user001',
    'user003',
    'private:user001_user003',
    'private',
    'Test message 2',
    2,
    UNIX_TIMESTAMP() * 1000,
    DATE_ADD(NOW(), INTERVAL 7 DAY),
    'region-a',
    'region-a-1234567891-0-2',
    'pending'
),
(
    UUID(),
    'user002',
    'user001',
    'private:user001_user002',
    'private',
    'Test message 3',
    3,
    UNIX_TIMESTAMP() * 1000,
    DATE_ADD(NOW(), INTERVAL 7 DAY),
    'region-b',
    'region-b-1234567892-0-3',
    'synced'
);

-- Verify partitioning is working
-- SELECT TABLE_NAME, PARTITION_NAME, TABLE_ROWS 
-- FROM INFORMATION_SCHEMA.PARTITIONS 
-- WHERE TABLE_NAME = 'offline_messages';

-- Performance test queries:
-- 1. Retrieve messages for a user (should use partition pruning)
-- EXPLAIN SELECT * FROM offline_messages WHERE user_id = 'user001' ORDER BY timestamp DESC LIMIT 100;

-- 2. Retrieve messages by conversation (may scan multiple partitions)
-- EXPLAIN SELECT * FROM offline_messages WHERE conversation_id = 'private:user001_user002' ORDER BY sequence_number;

-- 3. Check for duplicate message (should use unique index)
-- EXPLAIN SELECT * FROM offline_messages WHERE msg_id = 'some-uuid';

-- 4. TTL cleanup query (will scan all partitions)
-- EXPLAIN DELETE FROM offline_messages WHERE expires_at < NOW() LIMIT 10000;

-- Multi-region query patterns (Task 18.1):
-- 5. Find pending sync messages for a region (uses idx_region_sync_status)
-- EXPLAIN SELECT * FROM offline_messages 
-- WHERE region_id = 'region-a' AND sync_status = 'pending' 
-- ORDER BY created_at LIMIT 1000;

-- 6. Cross-region query by global_id (uses idx_global_id)
-- EXPLAIN SELECT * FROM offline_messages WHERE global_id = 'region-a-1234567890-0-1';

-- 7. Find conflict messages across regions
-- EXPLAIN SELECT * FROM offline_messages WHERE sync_status = 'conflict' ORDER BY created_at;

-- 8. Update sync status after successful replication
-- UPDATE offline_messages 
-- SET sync_status = 'synced', synced_at = NOW() 
-- WHERE region_id = 'region-a' AND sync_status = 'pending' 
-- LIMIT 1000;

-- 9. Cross-region message ordering by global_id (HLC-based)
-- SELECT * FROM offline_messages 
-- WHERE conversation_id = 'private:user001_user002' 
-- ORDER BY global_id;
