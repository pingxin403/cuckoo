-- token_bucket.lua
-- Token bucket algorithm for rate limiting in flash sale queue management
--
-- KEYS[1] = token_bucket:{skuId} (current token count)
-- KEYS[2] = token_bucket_last:{skuId} (last refill timestamp)
-- ARGV[1] = bucket capacity (max tokens)
-- ARGV[2] = refill rate (tokens per second)
-- ARGV[3] = current timestamp (milliseconds)
--
-- Returns:
--   1 = token acquired successfully
--   0 = no tokens available (need to queue)

local bucket_key = KEYS[1]
local last_key = KEYS[2]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Validate inputs
if capacity == nil or rate == nil or now == nil then
    return 0
end

-- Get current state
local tokens = tonumber(redis.call('GET', bucket_key) or capacity)
local last_time = tonumber(redis.call('GET', last_key) or now)

-- Calculate tokens to add based on elapsed time
local elapsed_ms = now - last_time
local elapsed_seconds = elapsed_ms / 1000.0
local tokens_to_add = elapsed_seconds * rate

-- Refill tokens (capped at capacity)
tokens = math.min(capacity, tokens + tokens_to_add)

-- Try to acquire a token
if tokens >= 1 then
    tokens = tokens - 1
    redis.call('SET', bucket_key, tokens)
    redis.call('SET', last_key, now)
    return 1  -- Token acquired
else
    -- Update last time even if no token acquired (for accurate refill calculation)
    redis.call('SET', last_key, now)
    return 0  -- No tokens available
end
