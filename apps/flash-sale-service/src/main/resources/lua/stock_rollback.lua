-- stock_rollback.lua
-- Atomic stock rollback script for order cancellation/timeout
--
-- KEYS[1] = stock:sku_{skuId} (remaining stock)
-- KEYS[2] = sold:sku_{skuId} (sold count)
-- ARGV[1] = quantity to rollback
--
-- Returns:
--   > 0 = new stock count after rollback (success)
--   -1  = invalid input (error)

local stock_key = KEYS[1]
local sold_key = KEYS[2]
local quantity = tonumber(ARGV[1])

-- Validate input
if quantity == nil or quantity <= 0 then
    return -1
end

-- Atomic rollback: increase stock and decrease sold count
local new_stock = redis.call('INCRBY', stock_key, quantity)
local new_sold = redis.call('DECRBY', sold_key, quantity)

-- Ensure sold count doesn't go negative
if new_sold < 0 then
    redis.call('SET', sold_key, 0)
end

return new_stock
