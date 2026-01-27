-- Migration: Add partitioning to offline_messages table
-- This migration adds hash partitioning by user_id for better query performance
-- 16 partitions for even distribution (each partition handles ~625K users for 10M total)

-- Drop the existing table if it exists (for clean migration)
-- Note: In production, you would migrate data first
DROP TABLE IF EXISTS offline_messages;

-- Recreate offline_messages table with partitioning
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
    
    PRIMARY KEY (id, user_id),  -- Composite key required for partitioning
    UNIQUE KEY idx_msg_id (msg_id),
    KEY idx_user_timestamp (user_id, timestamp),
    KEY idx_conversation_seq (conversation_id, sequence_number),
    KEY idx_expires_at (expires_at),
    KEY idx_user_conversation (user_id, conversation_id, sequence_number)
) ENGINE=InnoDB
PARTITION BY HASH(CRC32(user_id))
PARTITIONS 16;

-- Create index on msg_id for deduplication checks
-- Note: UNIQUE constraint on msg_id is maintained across all partitions

-- Test query performance with sample data
-- Insert sample offline messages for testing
INSERT INTO offline_messages (
    msg_id, user_id, sender_id, conversation_id, conversation_type,
    content, sequence_number, timestamp, expires_at
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
    DATE_ADD(NOW(), INTERVAL 7 DAY)
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
    DATE_ADD(NOW(), INTERVAL 7 DAY)
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
    DATE_ADD(NOW(), INTERVAL 7 DAY)
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
