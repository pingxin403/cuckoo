-- Add read receipt tracking to offline messages
-- This migration adds read_at timestamp to track when messages are read

ALTER TABLE offline_messages 
ADD COLUMN read_at TIMESTAMP NULL DEFAULT NULL AFTER created_at,
ADD KEY idx_user_read_status (user_id, read_at);

-- Create read_receipts table for tracking read status
-- This table stores read receipts for both online and offline scenarios
CREATE TABLE IF NOT EXISTS read_receipts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    msg_id VARCHAR(36) NOT NULL,
    reader_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conversation_type ENUM('private', 'group') NOT NULL,
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    device_id VARCHAR(64),
    
    UNIQUE KEY idx_msg_reader_device (msg_id, reader_id, device_id),
    KEY idx_sender_conversation (sender_id, conversation_id),
    KEY idx_msg_id (msg_id),
    KEY idx_read_at (read_at)
) ENGINE=InnoDB;

-- Add index for efficient querying of unread messages
ALTER TABLE offline_messages
ADD KEY idx_user_unread (user_id, read_at, timestamp);

