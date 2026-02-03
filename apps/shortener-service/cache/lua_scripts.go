package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// LuaScriptManager manages Lua scripts for atomic Redis operations
// This provides better performance and atomicity compared to multiple Redis commands
type LuaScriptManager struct {
	client     redis.UniversalClient
	obs        observability.Observability
	scripts    map[string]*redis.Script
	scriptSHAs map[string]string // Cache of script SHAs
}

// Script definitions
const (
	// CacheLoadScript checks if a key exists and sets a lock if it doesn't
	// This combines GET + SETNX into a single atomic operation
	//
	// KEYS[1] = cache key (e.g., "url:abc123")
	// KEYS[2] = lock key (e.g., "lock:abc123")
	// ARGV[1] = lock TTL in seconds (e.g., "5")
	//
	// Returns:
	// - "HIT" if cache key exists (returns cached value)
	// - "LOCKED" if lock was acquired (caller should load from DB)
	// - "CONTENTION" if lock already exists (caller should retry)
	CacheLoadScript = `
		-- Check if cache key exists
		local cache_value = redis.call('HGETALL', KEYS[1])
		if #cache_value > 0 then
			return {'HIT', cache_value}
		end
		
		-- Cache miss - try to acquire lock
		local lock_acquired = redis.call('SETNX', KEYS[2], '1')
		if lock_acquired == 1 then
			-- Set lock TTL to prevent deadlock
			redis.call('EXPIRE', KEYS[2], tonumber(ARGV[1]))
			return {'LOCKED'}
		else
			-- Lock already held by another process
			return {'CONTENTION'}
		end
	`

	// IncrementAndExpireScript atomically increments a counter and sets its expiration
	// This is useful for rate limiting, statistics, etc.
	//
	// KEYS[1] = counter key (e.g., "rate_limit:user:123")
	// ARGV[1] = increment amount (e.g., "1")
	// ARGV[2] = TTL in seconds (e.g., "60")
	//
	// Returns: new counter value after increment
	IncrementAndExpireScript = `
		local current = redis.call('INCRBY', KEYS[1], tonumber(ARGV[1]))
		redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
		return current
	`

	// SetWithTTLJitterScript sets a hash with TTL jitter
	// This combines HSET + EXPIRE with jitter calculation
	//
	// KEYS[1] = cache key
	// ARGV[1] = field1 name
	// ARGV[2] = field1 value
	// ARGV[3] = field2 name
	// ARGV[4] = field2 value
	// ARGV[5] = field3 name
	// ARGV[6] = field3 value
	// ARGV[7] = base TTL in seconds
	// ARGV[8] = jitter range in seconds
	//
	// Returns: "OK"
	SetWithTTLJitterScript = `
		-- Set hash fields
		redis.call('HSET', KEYS[1], ARGV[1], ARGV[2], ARGV[3], ARGV[4], ARGV[5], ARGV[6])
		
		-- Calculate TTL with jitter
		local base_ttl = tonumber(ARGV[7])
		local jitter_range = tonumber(ARGV[8])
		
		-- Generate random jitter: -jitter_range to +jitter_range
		math.randomseed(tonumber(redis.call('TIME')[1]))
		local jitter = math.random(-jitter_range, jitter_range)
		local ttl = base_ttl + jitter
		
		-- Set expiration
		redis.call('EXPIRE', KEYS[1], ttl)
		
		return 'OK'
	`
)

// NewLuaScriptManager creates a new LuaScriptManager instance
func NewLuaScriptManager(client redis.UniversalClient, obs observability.Observability) *LuaScriptManager {
	manager := &LuaScriptManager{
		client:     client,
		obs:        obs,
		scripts:    make(map[string]*redis.Script),
		scriptSHAs: make(map[string]string),
	}

	// Register scripts
	manager.registerScript("cache_load", CacheLoadScript)
	manager.registerScript("increment_expire", IncrementAndExpireScript)
	manager.registerScript("set_ttl_jitter", SetWithTTLJitterScript)

	return manager
}

// registerScript registers a Lua script with the manager
func (m *LuaScriptManager) registerScript(name string, script string) {
	m.scripts[name] = redis.NewScript(script)

	// Calculate SHA1 for the script
	hash := sha1.Sum([]byte(script))
	m.scriptSHAs[name] = hex.EncodeToString(hash[:])
}

// PreloadScripts preloads all scripts to Redis using SCRIPT LOAD
// This improves performance by avoiding script transmission on each call
func (m *LuaScriptManager) PreloadScripts(ctx context.Context) error {
	for name, script := range m.scripts {
		start := time.Now()

		// Load script to Redis
		sha, err := script.Load(ctx, m.client).Result()
		if err != nil {
			m.obs.Metrics().IncrementCounter("redis_lua_script_load_errors_total", map[string]string{
				"script": name,
			})
			return fmt.Errorf("failed to preload script %s: %w", name, err)
		}

		// Verify SHA matches
		if sha != m.scriptSHAs[name] {
			return fmt.Errorf("script %s SHA mismatch: expected %s, got %s", name, m.scriptSHAs[name], sha)
		}

		duration := time.Since(start).Seconds()
		m.obs.Metrics().RecordHistogram("redis_lua_script_load_duration_seconds", duration, map[string]string{
			"script": name,
		})
	}

	return nil
}

// ExecuteCacheLoad executes the cache load script
// Returns: status ("HIT", "LOCKED", "CONTENTION") and optional cache value
func (m *LuaScriptManager) ExecuteCacheLoad(ctx context.Context, cacheKey string, lockKey string, lockTTL int) (string, map[string]string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		m.obs.Metrics().RecordHistogram("redis_lua_script_duration_seconds", duration, map[string]string{
			"script": "cache_load",
		})
	}()

	script := m.scripts["cache_load"]

	// Try EVALSHA first (uses preloaded script)
	result, err := script.EvalSha(ctx, m.client, []string{cacheKey, lockKey}, lockTTL).Result()
	if err != nil {
		// If NOSCRIPT error, fall back to EVAL
		if isNoScriptError(err) {
			m.obs.Metrics().IncrementCounter("redis_lua_script_cache_misses_total", map[string]string{
				"script": "cache_load",
			})

			result, err = script.Eval(ctx, m.client, []string{cacheKey, lockKey}, lockTTL).Result()
			if err != nil {
				m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
					"script": "cache_load",
				})
				return "", nil, fmt.Errorf("cache load script failed: %w", err)
			}
		} else {
			m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
				"script": "cache_load",
			})
			return "", nil, fmt.Errorf("cache load script failed: %w", err)
		}
	} else {
		m.obs.Metrics().IncrementCounter("redis_lua_script_cache_hits_total", map[string]string{
			"script": "cache_load",
		})
	}

	// Parse result
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) == 0 {
		return "", nil, fmt.Errorf("invalid script result format")
	}

	status, ok := resultArray[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid status format")
	}

	// If HIT, parse cache value
	if status == "HIT" && len(resultArray) > 1 {
		cacheValueArray, ok := resultArray[1].([]interface{})
		if !ok {
			return status, nil, fmt.Errorf("invalid cache value format")
		}

		// Convert Redis hash array to map
		cacheValue := make(map[string]string)
		for i := 0; i < len(cacheValueArray); i += 2 {
			if i+1 < len(cacheValueArray) {
				key, _ := cacheValueArray[i].(string)
				value, _ := cacheValueArray[i+1].(string)
				cacheValue[key] = value
			}
		}

		return status, cacheValue, nil
	}

	return status, nil, nil
}

// ExecuteIncrementAndExpire executes the increment and expire script
// Returns: new counter value after increment
func (m *LuaScriptManager) ExecuteIncrementAndExpire(ctx context.Context, key string, increment int64, ttl int) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		m.obs.Metrics().RecordHistogram("redis_lua_script_duration_seconds", duration, map[string]string{
			"script": "increment_expire",
		})
	}()

	script := m.scripts["increment_expire"]

	// Try EVALSHA first
	result, err := script.EvalSha(ctx, m.client, []string{key}, increment, ttl).Result()
	if err != nil {
		// If NOSCRIPT error, fall back to EVAL
		if isNoScriptError(err) {
			m.obs.Metrics().IncrementCounter("redis_lua_script_cache_misses_total", map[string]string{
				"script": "increment_expire",
			})

			result, err = script.Eval(ctx, m.client, []string{key}, increment, ttl).Result()
			if err != nil {
				m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
					"script": "increment_expire",
				})
				return 0, fmt.Errorf("increment and expire script failed: %w", err)
			}
		} else {
			m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
				"script": "increment_expire",
			})
			return 0, fmt.Errorf("increment and expire script failed: %w", err)
		}
	} else {
		m.obs.Metrics().IncrementCounter("redis_lua_script_cache_hits_total", map[string]string{
			"script": "increment_expire",
		})
	}

	// Parse result
	value, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid result format")
	}

	return value, nil
}

// ExecuteSetWithTTLJitter executes the set with TTL jitter script
func (m *LuaScriptManager) ExecuteSetWithTTLJitter(ctx context.Context, key string, fields map[string]string, baseTTL int, jitterRange int) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		m.obs.Metrics().RecordHistogram("redis_lua_script_duration_seconds", duration, map[string]string{
			"script": "set_ttl_jitter",
		})
	}()

	script := m.scripts["set_ttl_jitter"]

	// Prepare arguments (field1, value1, field2, value2, field3, value3, baseTTL, jitterRange)
	args := make([]interface{}, 0, len(fields)*2+2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	args = append(args, baseTTL, jitterRange)

	// Try EVALSHA first
	_, err := script.EvalSha(ctx, m.client, []string{key}, args...).Result()
	if err != nil {
		// If NOSCRIPT error, fall back to EVAL
		if isNoScriptError(err) {
			m.obs.Metrics().IncrementCounter("redis_lua_script_cache_misses_total", map[string]string{
				"script": "set_ttl_jitter",
			})

			_, err = script.Eval(ctx, m.client, []string{key}, args...).Result()
			if err != nil {
				m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
					"script": "set_ttl_jitter",
				})
				return fmt.Errorf("set with TTL jitter script failed: %w", err)
			}
		} else {
			m.obs.Metrics().IncrementCounter("redis_lua_script_errors_total", map[string]string{
				"script": "set_ttl_jitter",
			})
			return fmt.Errorf("set with TTL jitter script failed: %w", err)
		}
	} else {
		m.obs.Metrics().IncrementCounter("redis_lua_script_cache_hits_total", map[string]string{
			"script": "set_ttl_jitter",
		})
	}

	return nil
}

// isNoScriptError checks if the error is a NOSCRIPT error
// This indicates the script is not in Redis cache and needs to be loaded
func isNoScriptError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "NOSCRIPT No matching script. Please use EVAL."
}

// GetScriptSHA returns the SHA1 hash of a script
func (m *LuaScriptManager) GetScriptSHA(name string) (string, bool) {
	sha, ok := m.scriptSHAs[name]
	return sha, ok
}

// ListScripts returns all registered script names
func (m *LuaScriptManager) ListScripts() []string {
	names := make([]string, 0, len(m.scripts))
	for name := range m.scripts {
		names = append(names, name)
	}
	return names
}
