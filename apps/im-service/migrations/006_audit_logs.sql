-- Migration: Create audit_logs table for security and compliance
-- Validates: Requirements 13.3, 13.4

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL UNIQUE,
    timestamp TIMESTAMP(3) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    event_category VARCHAR(32) NOT NULL,
    severity VARCHAR(16) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    device_id VARCHAR(64),
    ip_address VARCHAR(45),
    user_agent TEXT,
    session_id VARCHAR(64),
    trace_id VARCHAR(64),
    result VARCHAR(16) NOT NULL,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_event_type (event_type),
    INDEX idx_event_category (event_category),
    INDEX idx_severity (severity),
    INDEX idx_trace_id (trace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Note: Partitioning by month should be added in production
-- This requires manual setup based on deployment timeline
-- Example:
-- ALTER TABLE audit_logs PARTITION BY RANGE (UNIX_TIMESTAMP(timestamp)) (
--     PARTITION p202501 VALUES LESS THAN (UNIX_TIMESTAMP('2025-02-01')),
--     PARTITION p202502 VALUES LESS THAN (UNIX_TIMESTAMP('2025-03-01')),
--     PARTITION p_future VALUES LESS THAN MAXVALUE
-- );
