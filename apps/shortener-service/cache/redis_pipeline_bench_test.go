package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// setupBenchPipeline creates a test PipelineHelper for benchmarking
func setupBenchPipeline(b *testing.B) (*PipelineHelper, redis.UniversalClient, func()) {
	b.Helper()

	// Start miniredis
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "bench-pipeline",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	if err != nil {
		b.Fatalf("Failed to create observability: %v", err)
	}

	// Create PipelineHelper
	pipeline := NewPipelineHelper(client, obs)

	cleanup := func() {
		client.Close()
		mr.Close()
		obs.Shutdown(context.Background())
	}

	return pipeline, client, cleanup
}

// BenchmarkPipelineVsIndividual compares Pipeline batch operations vs individual commands
func BenchmarkPipelineVsIndividual(b *testing.B) {
	pipeline, client, cleanup := setupBenchPipeline(b)
	defer cleanup()

	ctx := context.Background()
	ttl := 1 * time.Hour

	// Prepare test data
	entries := make(map[string]string)
	for i := 0; i < 100; i++ {
		entries[fmt.Sprintf("bench_key_%d", i)] = fmt.Sprintf("bench_value_%d", i)
	}

	b.Run("Individual_Commands", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for key, value := range entries {
				client.Set(ctx, key, value, ttl)
			}
		}
	})

	b.Run("Pipeline_Batch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.BatchSet(ctx, entries, ttl)
		}
	})
}

// BenchmarkBatchSizes tests different batch sizes
func BenchmarkBatchSizes(b *testing.B) {
	pipeline, _, cleanup := setupBenchPipeline(b)
	defer cleanup()

	ctx := context.Background()
	ttl := 1 * time.Hour

	batchSizes := []int{10, 50, 100, 500, 1000}

	for _, size := range batchSizes {
		entries := make(map[string]string)
		for i := 0; i < size; i++ {
			entries[fmt.Sprintf("size_key_%d", i)] = fmt.Sprintf("size_value_%d", i)
		}

		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pipeline.BatchSet(ctx, entries, ttl)
			}
		})
	}
}

// BenchmarkBatchGet tests batch get performance
func BenchmarkBatchGet(b *testing.B) {
	pipeline, client, cleanup := setupBenchPipeline(b)
	defer cleanup()

	ctx := context.Background()

	// Setup test data
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("get_bench_key_%d", i)
		keys[i] = key
		client.Set(ctx, key, fmt.Sprintf("get_bench_value_%d", i), 1*time.Hour)
	}

	b.Run("Individual_Gets", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				client.Get(ctx, key)
			}
		}
	})

	b.Run("Pipeline_BatchGet", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.BatchGet(ctx, keys)
		}
	})
}

// BenchmarkConcurrentBatchSet tests concurrent batch operations
func BenchmarkConcurrentBatchSet(b *testing.B) {
	pipeline, _, cleanup := setupBenchPipeline(b)
	defer cleanup()

	ctx := context.Background()
	ttl := 1 * time.Hour

	entries := make(map[string]string)
	for i := 0; i < 100; i++ {
		entries[fmt.Sprintf("concurrent_key_%d", i)] = fmt.Sprintf("concurrent_value_%d", i)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pipeline.BatchSet(ctx, entries, ttl)
		}
	})
}

// BenchmarkBatchSplitting tests the performance of batch splitting
func BenchmarkBatchSplitting(b *testing.B) {
	pipeline, _, cleanup := setupBenchPipeline(b)
	defer cleanup()

	// Create a large batch that will be split
	largeEntries := make(map[string]string)
	for i := 0; i < 2500; i++ {
		largeEntries[fmt.Sprintf("split_key_%d", i)] = fmt.Sprintf("split_value_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.splitIntoBatches(largeEntries)
	}
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	pipeline, _, cleanup := setupBenchPipeline(b)
	defer cleanup()

	ctx := context.Background()
	ttl := 1 * time.Hour

	entries := make(map[string]string)
	for i := 0; i < 100; i++ {
		entries[fmt.Sprintf("mem_key_%d", i)] = fmt.Sprintf("mem_value_%d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.BatchSet(ctx, entries, ttl)
	}
}
