-- stock_deduct.lua
-- Atomic stock deduction script for flash sale inventory management
--
-- KEYS[1] = stock:sku_{skuId} (remaining stock)
-- KEYS[2] = sold:sku_{skuId} (sold count)
-- ARGV[1] = quantity to deduct
--
-- Returns:
--   > 0 = remaining stock after deduction (success)
--   0   = out of stock (no deduction performed)
--   -1  = invalid input (error)

local stock_key = KEYS[1]
local sold_key = KEYS[2]
local quantity = tonumber(ARGV[1])

-- Validate input
if quantity == nil or quantity <= 0 then
    return -1
end

-- Get current stock
local current_stock = tonumber(redis.call('GET', stock_key) or 0)

-- Check if sufficient stock
if current_stock < quantity then
    return 0  -- Out of stock
end

-- Atomic deduction: decrease stock and increase sold count
local remaining = redis.call('DECRBY', stock_key, quantity)
redis.call('INCRBY', sold_key, quantity)

return remaining
