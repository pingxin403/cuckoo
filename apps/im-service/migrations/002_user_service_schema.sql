-- User Service Schema for IM Chat System
-- Database: im_chat (shared with offline messages)

-- Users table: stores user profile information
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(256) NOT NULL UNIQUE,
    display_name VARCHAR(256) NOT NULL,
    avatar_url VARCHAR(512),
    status INT NOT NULL DEFAULT 2,  -- 0=UNSPECIFIED, 1=ONLINE, 2=OFFLINE, 3=AWAY, 4=BUSY
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_username (username),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Groups table: stores group metadata
CREATE TABLE IF NOT EXISTS groups (
    group_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    creator_id VARCHAR(64) NOT NULL,
    member_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_creator (creator_id),
    INDEX idx_created_at (created_at),
    INDEX idx_member_count (member_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Group members table: stores group membership information
CREATE TABLE IF NOT EXISTS group_members (
    group_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    role INT NOT NULL DEFAULT 1,  -- 0=UNSPECIFIED, 1=MEMBER, 2=ADMIN, 3=OWNER
    group_display_name VARCHAR(256),  -- Optional: overrides user display_name in group
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    
    PRIMARY KEY (group_id, user_id),
    INDEX idx_user_groups (user_id),
    INDEX idx_group_role (group_id, role),
    INDEX idx_joined_at (joined_at),
    
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert sample data for testing
INSERT INTO users (user_id, username, display_name, avatar_url, status) VALUES
('user001', 'alice', 'Alice Smith', 'https://example.com/avatars/alice.jpg', 1),
('user002', 'bob', 'Bob Johnson', 'https://example.com/avatars/bob.jpg', 2),
('user003', 'charlie', 'Charlie Brown', 'https://example.com/avatars/charlie.jpg', 1),
('user004', 'diana', 'Diana Prince', 'https://example.com/avatars/diana.jpg', 3),
('user005', 'eve', 'Eve Wilson', 'https://example.com/avatars/eve.jpg', 2)
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

INSERT INTO groups (group_id, name, creator_id, member_count) VALUES
('group001', 'Engineering Team', 'user001', 3),
('group002', 'Product Team', 'user002', 2),
('group003', 'Large Group Test', 'user001', 5)
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

INSERT INTO group_members (group_id, user_id, role, group_display_name, is_muted) VALUES
('group001', 'user001', 3, NULL, FALSE),  -- Alice is owner
('group001', 'user002', 2, NULL, FALSE),  -- Bob is admin
('group001', 'user003', 1, NULL, FALSE),  -- Charlie is member
('group002', 'user002', 3, NULL, FALSE),  -- Bob is owner
('group002', 'user004', 1, 'Wonder Woman', FALSE),  -- Diana with custom name
('group003', 'user001', 3, NULL, FALSE),  -- Alice is owner
('group003', 'user002', 1, NULL, FALSE),
('group003', 'user003', 1, NULL, FALSE),
('group003', 'user004', 1, NULL, FALSE),
('group003', 'user005', 1, NULL, TRUE)   -- Eve is muted
ON DUPLICATE KEY UPDATE joined_at = CURRENT_TIMESTAMP;

-- Update member counts
UPDATE groups g SET member_count = (
    SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.group_id
);
