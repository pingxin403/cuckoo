package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// PipelineHelper provides batch operations using Redis Pipeline
type PipelineHelper struct {
	client       redis.UniversalClient
	obs          observability.Observability
	maxBatchSize int // Default: 1000
}

// NewPipelineHelper creates a new PipelineHelper instance
func NewPipelineHelper(client redis.UniversalClient, obs observability.Observability) *PipelineHelper {
	return &PipelineHelper{
		client:       client,
		obs:          obs,
		maxBatchSize: 1000,
	}
}

// BatchSet sets multiple keys using Pipeline
func (p *PipelineHelper) BatchSet(ctx context.Context, entries map[string]string, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		p.obs.Metrics().RecordHistogram("redis_pipeline_duration_seconds", duration, map[string]string{"operation": "batch_set"})
	}()

	// Split into batches if needed
	batches := p.splitIntoBatches(entries)

	for _, batch := range batches {
		pipe := p.client.Pipeline()

		for key, value := range batch {
			pipe.Set(ctx, key, value, ttl)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			p.obs.Metrics().IncrementCounter("redis_pipeline_errors_total", map[string]string{"operation": "batch_set"})
			return fmt.Errorf("pipeline exec failed: %w", err)
		}

		p.obs.Metrics().RecordHistogram("redis_pipeline_batch_size", float64(len(batch)), nil)
	}

	return nil
}

// BatchGet retrieves multiple keys using Pipeline
func (p *PipelineHelper) BatchGet(ctx context.Context, keys []string) (map[string]string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		p.obs.Metrics().RecordHistogram("redis_pipeline_duration_seconds", duration, map[string]string{"operation": "batch_get"})
	}()

	pipe := p.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		p.obs.Metrics().IncrementCounter("redis_pipeline_errors_total", map[string]string{"operation": "batch_get"})
		return nil, fmt.Errorf("pipeline exec failed: %w", err)
	}

	results := make(map[string]string)
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err == nil {
			results[key] = val
		}
	}

	p.obs.Metrics().RecordHistogram("redis_pipeline_batch_size", float64(len(keys)), nil)
	return results, nil
}

// splitIntoBatches splits a large map into smaller batches
func (p *PipelineHelper) splitIntoBatches(entries map[string]string) []map[string]string {
	var batches []map[string]string
	currentBatch := make(map[string]string)

	for key, value := range entries {
		currentBatch[key] = value

		if len(currentBatch) >= p.maxBatchSize {
			batches = append(batches, currentBatch)
			currentBatch = make(map[string]string)
		}
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}
