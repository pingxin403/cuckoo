-- TTL Cleanup Job for Offline Messages
-- This script should be run hourly via cron job
-- Deletes messages older than 7 days in batches of 10,000

-- Batch delete expired messages
-- Using LIMIT to prevent long-running transactions
DELETE FROM offline_messages 
WHERE expires_at < NOW() 
LIMIT 10000;

-- Log cleanup statistics (to be captured by the application)
-- SELECT 
--     COUNT(*) as remaining_expired,
--     MIN(expires_at) as oldest_message
-- FROM offline_messages 
-- WHERE expires_at < NOW();
