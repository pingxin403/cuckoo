#!/bin/bash

# Redis Benchmark Script for Flash Sale System
# Tests inventory operations performance

set -e

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
ITERATIONS="${ITERATIONS:-10000}"

echo "========================================"
echo "Redis Benchmark for Flash Sale"
echo "========================================"
echo "Host: $REDIS_HOST:$REDIS_PORT"
echo "Iterations: $ITERATIONS"
echo "========================================"

echo -e "\n[1/6] Testing PING latency..."
redis-cli -h $REDIS_HOST -p $REDIS_PORT PING

echo -e "\n[2/6] Testing SET operations..."
redis-benchmark -h $REDIS_HOST -p $REDIS_PORT -n $ITERATIONS -t set -q

echo -e "\n[3/6] Testing GET operations..."
redis-benchmark -h $REDIS_HOST -p $REDIS_PORT -n $ITERATIONS -t get -q

echo -e "\n[4/6] Testing INCR operations (stock decrement simulation)..."
redis-benchmark -h $REDIS_HOST -p $REDIS_PORT -n $ITERATIONS -t incr -q

echo -e "\n[5/6] Testing Lua script (stock_deduct)..."
redis-cli -h $REDIS_HOST -p $REDIS_PORT SCRIPT LOAD "
local stock = redis.call('GET', KEYS[1])
if tonumber(stock) >= tonumber(ARGV[1]) then
    return redis.call('DECRBY', KEYS[1], ARGV[1])
else
    return -1
end
"

echo -e "\n[6/6] Custom Lua script benchmark..."
redis-benchmark -h $REDIS_HOST -p $REDIS_PORT \
    -n $ITERATIONS \
    -s "stock:test-sku" \
    -S 1000 \
    -r 100000 \
    -L 1 \
    -q \
    --eval "local stock = redis.call('GET', KEYS[1]) if tonumber(stock) >= 1 then return redis.call('DECRBY', KEYS[1], 1) else return -1 end" \
    stock:test-sku

echo -e "\n========================================"
echo "Benchmark Complete!"
echo "========================================"
