-- Migration: Create url_mappings table
-- Description: Creates the main table for storing URL mappings with proper indexes
-- Requirements: 2.2, 5.5

CREATE TABLE IF NOT EXISTS url_mappings (
    -- Primary key: 7-character Base62 short code (can expand to 10 for future growth)
    short_code VARCHAR(10) PRIMARY KEY,
    
    -- Original long URL (max 2048 characters as per requirements)
    long_url TEXT NOT NULL,
    
    -- Timestamps stored as Unix epoch (BIGINT for efficient indexing and comparison)
    created_at BIGINT NOT NULL,
    expires_at BIGINT NULL,
    
    -- Audit and analytics fields
    creator_ip VARCHAR(45) NOT NULL,  -- Supports both IPv4 and IPv6
    click_count BIGINT DEFAULT 0,
    
    -- Soft delete flag
    is_deleted BOOLEAN DEFAULT FALSE,
    
    -- Indexes for efficient queries
    INDEX idx_expires_at (expires_at),
    INDEX idx_created_at (created_at),
    INDEX idx_is_deleted (is_deleted)
    
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add comments for documentation
ALTER TABLE url_mappings 
    COMMENT = 'Stores URL mappings for the shortener service';

-- Note: The PRIMARY KEY on short_code automatically creates a UNIQUE constraint
-- This prevents duplicate short codes as required by Requirements 2.2
