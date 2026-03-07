package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// ClusterClient wraps Redis Cluster client with hash tag support
// Hash tags ensure related keys are stored on the same cluster slot
type ClusterClient struct {
	client redis.UniversalClient
	obs    observability.Observability
}

// NewClusterClient creates a new ClusterClient wrapper
func NewClusterClient(client redis.UniversalClient, obs observability.Observability) *ClusterClient {
	return &ClusterClient{
		client: client,
		obs:    obs,
	}
}

// trackClusterError analyzes Redis errors and tracks cluster-specific metrics
func (c *ClusterClient) trackClusterError(err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	// Track MOVED redirects
	// MOVED errors indicate the key has been moved to a different slot
	if strings.Contains(errMsg, "MOVED") {
		c.obs.Metrics().IncrementCounter("redis_cluster_moved_redirects_total", nil)
	}

	// Track ASK redirects
	// ASK errors indicate temporary redirection during slot migration
	if strings.Contains(errMsg, "ASK") {
		c.obs.Metrics().IncrementCounter("redis_cluster_ask_redirects_total", nil)
	}

	// Track CROSSSLOT errors
	// CROSSSLOT errors occur when trying to operate on keys in different slots
	if strings.Contains(errMsg, "CROSSSLOT") {
		c.obs.Metrics().IncrementCounter("redis_cluster_crossslot_errors_total", nil)
	}

	// Track CLUSTERDOWN errors
	// CLUSTERDOWN errors indicate the cluster is unavailable
	if strings.Contains(errMsg, "CLUSTERDOWN") {
		c.obs.Metrics().IncrementCounter("redis_cluster_down_errors_total", nil)
	}

	// Track TRYAGAIN errors
	// TRYAGAIN errors indicate temporary failures during slot migration
	if strings.Contains(errMsg, "TRYAGAIN") {
		c.obs.Metrics().IncrementCounter("redis_cluster_tryagain_errors_total", nil)
	}
}

// FormatKeyWithHashTag formats a key with hash tag to ensure slot consistency
// Hash tag format: {prefix}:key
// All keys with the same hash tag will be stored on the same cluster slot
//
// Example:
//
//	FormatKeyWithHashTag("user", "123") -> "{user}:123"
//	FormatKeyWithHashTag("url", "abc") -> "{url}:abc"
//
// Keys with the same prefix will be on the same slot:
//
//	{user}:123, {user}:456, {user}:789 -> same slot
func (c *ClusterClient) FormatKeyWithHashTag(prefix, key string) string {
	return fmt.Sprintf("{%s}:%s", prefix, key)
}

// BatchGetWithHashTag retrieves multiple keys using hash tags
// All keys must use the same hash tag prefix to ensure they're on the same slot
// This enables efficient batch operations in Redis Cluster
//
// Example:
//
//	keys := []string{"123", "456", "789"}
//	results := BatchGetWithHashTag(ctx, "user", keys)
//	// Fetches: {user}:123, {user}:456, {user}:789
func (c *ClusterClient) BatchGetWithHashTag(ctx context.Context, prefix string, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	startTime := time.Now()

	// Format all keys with hash tag
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.FormatKeyWithHashTag(prefix, key)
	}

	// Use pipeline for batch GET operations
	// Since all keys have the same hash tag, they're on the same slot
	pipe := c.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(formattedKeys))
	for i, formattedKey := range formattedKeys {
		cmds[i] = pipe.Get(ctx, formattedKey)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Track cluster-specific errors
		c.trackClusterError(err)
		// Track error
		c.obs.Metrics().IncrementCounter("redis_cluster_batch_get_errors_total", map[string]string{
			"prefix": prefix,
		})
		return nil, fmt.Errorf("failed to batch get with hash tag: %w", err)
	}

	// Collect results
	results := make(map[string]string)
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			// Key not found, skip
			continue
		}
		if err != nil {
			// Individual key error, skip
			continue
		}
		// Map original key to value
		results[keys[i]] = val
	}

	// Track metrics
	duration := time.Since(startTime).Seconds()
	c.obs.Metrics().RecordHistogram("redis_cluster_batch_get_duration_seconds", duration, map[string]string{
		"prefix": prefix,
	})
	c.obs.Metrics().IncrementCounter("redis_cluster_batch_get_total", map[string]string{
		"prefix": prefix,
	})

	return results, nil
}

// BatchSetWithHashTag stores multiple key-value pairs using hash tags
// All keys must use the same hash tag prefix to ensure they're on the same slot
func (c *ClusterClient) BatchSetWithHashTag(ctx context.Context, prefix string, kvPairs map[string]string, ttl time.Duration) error {
	if len(kvPairs) == 0 {
		return nil
	}

	startTime := time.Now()

	// Use pipeline for batch SET operations
	pipe := c.client.Pipeline()
	for key, value := range kvPairs {
		formattedKey := c.FormatKeyWithHashTag(prefix, key)
		pipe.Set(ctx, formattedKey, value, ttl)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		// Track cluster-specific errors
		c.trackClusterError(err)
		// Track error
		c.obs.Metrics().IncrementCounter("redis_cluster_batch_set_errors_total", map[string]string{
			"prefix": prefix,
		})
		return fmt.Errorf("failed to batch set with hash tag: %w", err)
	}

	// Track metrics
	duration := time.Since(startTime).Seconds()
	c.obs.Metrics().RecordHistogram("redis_cluster_batch_set_duration_seconds", duration, map[string]string{
		"prefix": prefix,
	})
	c.obs.Metrics().IncrementCounter("redis_cluster_batch_set_total", map[string]string{
		"prefix": prefix,
	})

	return nil
}

// BatchDeleteWithHashTag deletes multiple keys using hash tags
// All keys must use the same hash tag prefix to ensure they're on the same slot
func (c *ClusterClient) BatchDeleteWithHashTag(ctx context.Context, prefix string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	startTime := time.Now()

	// Format all keys with hash tag
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.FormatKeyWithHashTag(prefix, key)
	}

	// Delete all keys
	err := c.client.Del(ctx, formattedKeys...).Err()
	if err != nil {
		// Track cluster-specific errors
		c.trackClusterError(err)
		// Track error
		c.obs.Metrics().IncrementCounter("redis_cluster_batch_delete_errors_total", map[string]string{
			"prefix": prefix,
		})
		return fmt.Errorf("failed to batch delete with hash tag: %w", err)
	}

	// Track metrics
	duration := time.Since(startTime).Seconds()
	c.obs.Metrics().RecordHistogram("redis_cluster_batch_delete_duration_seconds", duration, map[string]string{
		"prefix": prefix,
	})
	c.obs.Metrics().IncrementCounter("redis_cluster_batch_delete_total", map[string]string{
		"prefix": prefix,
	})

	return nil
}

// Client returns the underlying Redis client
func (c *ClusterClient) Client() redis.UniversalClient {
	return c.client
}

// GetClusterInfo retrieves cluster information and exposes slot distribution metrics
// This method should be called periodically to track cluster health
func (c *ClusterClient) GetClusterInfo(ctx context.Context) error {
	// Check if this is a cluster client
	clusterClient, ok := c.client.(*redis.ClusterClient)
	if !ok {
		// Not a cluster client, skip
		return nil
	}

	// Get cluster info
	info, err := clusterClient.ClusterInfo(ctx).Result()
	if err != nil {
		c.trackClusterError(err)
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Parse cluster state
	if strings.Contains(info, "cluster_state:ok") {
		c.obs.Metrics().SetGauge("redis_cluster_state", 1, nil) // 1 = OK
	} else {
		c.obs.Metrics().SetGauge("redis_cluster_state", 0, nil) // 0 = FAIL
	}

	// Get cluster nodes
	nodes, err := clusterClient.ClusterNodes(ctx).Result()
	if err != nil {
		c.trackClusterError(err)
		return fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Count nodes by role
	masterCount := 0
	slaveCount := 0
	lines := strings.Split(nodes, "\n")
	for _, line := range lines {
		if strings.Contains(line, "master") {
			masterCount++
		} else if strings.Contains(line, "slave") {
			slaveCount++
		}
	}

	// Expose node count metrics
	c.obs.Metrics().SetGauge("redis_cluster_nodes", float64(masterCount), map[string]string{"role": "master"})
	c.obs.Metrics().SetGauge("redis_cluster_nodes", float64(slaveCount), map[string]string{"role": "slave"})

	// Get cluster slots
	slots, err := clusterClient.ClusterSlots(ctx).Result()
	if err != nil {
		c.trackClusterError(err)
		return fmt.Errorf("failed to get cluster slots: %w", err)
	}

	// Track slot distribution
	totalSlots := 0
	for _, slot := range slots {
		slotCount := slot.End - slot.Start + 1
		totalSlots += slotCount
	}

	c.obs.Metrics().SetGauge("redis_cluster_slots_total", float64(totalSlots), nil)
	c.obs.Metrics().SetGauge("redis_cluster_slot_ranges", float64(len(slots)), nil)

	return nil
}

// StartClusterMetricsCollection starts a background goroutine that collects cluster metrics
// This should be called once during initialization
func (c *ClusterClient) StartClusterMetricsCollection(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.GetClusterInfo(ctx); err != nil {
					// Log error but continue
					c.obs.Metrics().IncrementCounter("redis_cluster_metrics_collection_errors_total", nil)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
