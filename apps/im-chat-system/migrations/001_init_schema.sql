-- IM Chat System Database Schema
-- Database: im_chat

-- ===== Offline Messages Table =====
CREATE TABLE IF NOT EXISTS offline_messages (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    msg_id VARCHAR(36) UNIQUE NOT NULL,
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
    
    KEY idx_msg_id (msg_id),
    KEY idx_user_timestamp (user_id, timestamp),
    KEY idx_conversation_seq (conversation_id, sequence_number),
    KEY idx_expires_at (expires_at),
    KEY idx_user_conversation (user_id, conversation_id, sequence_number)
) ENGINE=InnoDB;

-- ===== Groups Table =====
CREATE TABLE IF NOT EXISTS `groups` (
    group_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    creator_id VARCHAR(64) NOT NULL,
    member_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    KEY idx_creator (creator_id),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB;

-- ===== Group Members Table =====
CREATE TABLE IF NOT EXISTS group_members (
    group_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    role ENUM('owner', 'admin', 'member') NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (group_id, user_id),
    KEY idx_user_groups (user_id),
    FOREIGN KEY (group_id) REFERENCES `groups`(group_id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- ===== Sequence Number Snapshots Table (Redis Failure Recovery) =====
CREATE TABLE IF NOT EXISTS sequence_snapshots (
    conversation_id VARCHAR(128) PRIMARY KEY,
    conversation_type ENUM('private', 'group') NOT NULL,
    sequence_number BIGINT NOT NULL,
    snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    KEY idx_snapshot_at (snapshot_at)
) ENGINE=InnoDB;

-- ===== Users Table (for User Service) =====
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(128) UNIQUE NOT NULL,
    email VARCHAR(256) UNIQUE NOT NULL,
    display_name VARCHAR(256),
    avatar_url VARCHAR(512),
    status ENUM('active', 'inactive', 'banned') NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    KEY idx_username (username),
    KEY idx_email (email),
    KEY idx_status (status)
) ENGINE=InnoDB;

-- ===== Sample Data for Testing =====
-- Insert sample users
INSERT INTO users (user_id, username, email, display_name) VALUES
('user_001', 'alice', 'alice@example.com', 'Alice Smith'),
('user_002', 'bob', 'bob@example.com', 'Bob Johnson'),
('user_003', 'charlie', 'charlie@example.com', 'Charlie Brown')
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- Insert sample group
INSERT INTO `groups` (group_id, name, creator_id, member_count) VALUES
('group_001', 'Test Group', 'user_001', 3)
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- Insert sample group members
INSERT INTO group_members (group_id, user_id, role) VALUES
('group_001', 'user_001', 'owner'),
('group_001', 'user_002', 'member'),
('group_001', 'user_003', 'member')
ON DUPLICATE KEY UPDATE joined_at = CURRENT_TIMESTAMP;
