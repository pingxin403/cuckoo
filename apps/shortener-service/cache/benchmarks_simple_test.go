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

// BenchmarkPipelineEfficiencySimple benchmarks pipeline vs individual commands
func BenchmarkPipelineEfficiencySimple(b *testing.B) {
	mr := miniredis.RunT(b)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	obs, _ := observability.New(observability.Config{
		ServiceName:    "benchmark-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	pipeline := NewPipelineHelper(client, obs)
	ctx := context.Background()

	// Prepare test data
	testData := make(map[string]string)
	for i := 0; i < 100; i++ {
		testData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	b.Run("Individual_Commands", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for key, value := range testData {
				client.Set(ctx, key, value, 0)
			}
		}
	})

	b.Run("Pipeline_Batch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.BatchSet(ctx, testData, 0)
		}
	})
}

// BenchmarkSingleflightEfficiencySimple benchmarks singleflight coalescing
func BenchmarkSingleflightEfficiencySimple(b *testing.B) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "benchmark-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	// Simulate slow operation
	slowOp := func() (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "result", nil
	}

	b.Run("Without_Singleflight", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				slowOp()
			}
		})
	})

	b.Run("With_Singleflight", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sf.Do(ctx, "test-key", slowOp)
			}
		})
	})
}

// BenchmarkConnectionPoolUtilizationSimple benchmarks connection pool performance
func BenchmarkConnectionPoolUtilizationSimple(b *testing.B) {
	mr := miniredis.RunT(b)
	defer mr.Close()

	// Small pool
	smallPoolClient := redis.NewClient(&redis.Options{
		Addr:     mr.Addr(),
		PoolSize: 5,
	})
	defer smallPoolClient.Close()

	// Optimized pool
	optimizedPoolClient := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		PoolSize:     20,
		MinIdleConns: 6,
	})
	defer optimizedPoolClient.Close()

	ctx := context.Background()

	b.Run("Small_Pool_5", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				smallPoolClient.Get(ctx, "test-key")
			}
		})
	})

	b.Run("Optimized_Pool_20", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				optimizedPoolClient.Get(ctx, "test-key")
			}
		})
	})
}

// BenchmarkCircuitBreakerOverheadSimple benchmarks circuit breaker overhead
func BenchmarkCircuitBreakerOverheadSimple(b *testing.B) {
	mr := miniredis.RunT(b)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	obs, _ := observability.New(observability.Config{
		ServiceName:    "benchmark-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	cbConfig := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(cbConfig, obs)
	ctx := context.Background()

	successOp := func() error {
		return client.Get(ctx, "test-key").Err()
	}

	b.Run("Without_CircuitBreaker", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			successOp()
		}
	})

	b.Run("With_CircuitBreaker", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.Execute(ctx, successOp)
		}
	})
}

// BenchmarkLuaScriptPerformanceSimple benchmarks Lua script vs multiple commands
func BenchmarkLuaScriptPerformanceSimple(b *testing.B) {
	mr := miniredis.RunT(b)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	obs, _ := observability.New(observability.Config{
		ServiceName:    "benchmark-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	scriptMgr := NewLuaScriptManager(client, obs)
	scriptMgr.PreloadScripts(context.Background())

	ctx := context.Background()

	b.Run("Multiple_Commands_INCR_EXPIRE", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("counter-%d", i)
			client.Incr(ctx, key)
			client.Expire(ctx, key, 60*time.Second)
		}
	})

	b.Run("Lua_Script_IncrementAndExpire", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("counter-%d", i)
			scriptMgr.ExecuteIncrementAndExpire(ctx, key, 1, 60)
		}
	})
}

// BenchmarkMemoryEfficiencySimple benchmarks memory usage of different approaches
func BenchmarkMemoryEfficiencySimple(b *testing.B) {
	mr := miniredis.RunT(b)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	obs, _ := observability.New(observability.Config{
		ServiceName:    "benchmark-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	pipeline := NewPipelineHelper(client, obs)
	ctx := context.Background()

	testData := make(map[string]string)
	for i := 0; i < 1000; i++ {
		testData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	b.Run("Individual_Commands_1000", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for key, value := range testData {
				client.Set(ctx, key, value, 0)
			}
		}
	})

	b.Run("Pipeline_Batch_1000", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipeline.BatchSet(ctx, testData, 0)
		}
	})
}
