-- Migration: Create sequence_snapshots table for sequence number backup
-- This table stores periodic snapshots of sequence numbers from Redis
-- Used for recovery when Redis fails

CREATE TABLE IF NOT EXISTS sequence_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_type VARCHAR(20) NOT NULL COMMENT 'Type of conversation: private or group',
    conversation_id VARCHAR(255) NOT NULL COMMENT 'Unique conversation identifier',
    sequence_number BIGINT NOT NULL COMMENT 'Last known sequence number',
    snapshot_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'When the snapshot was taken',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Unique constraint to ensure one snapshot per conversation
    UNIQUE KEY uk_conversation (conversation_type, conversation_id),
    
    -- Index for querying by conversation
    INDEX idx_conversation_type (conversation_type),
    INDEX idx_snapshot_time (snapshot_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Stores periodic snapshots of sequence numbers for disaster recovery';
