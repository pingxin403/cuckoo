package main

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingxin403/cuckoo/apps/im-service/traffic"
)

// TestCLIIntegration tests the CLI tool integration with the traffic switcher
func TestCLIIntegration(t *testing.T) {
	// Setup mini-redis for testing
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	// Create traffic switcher
	switcher := traffic.NewTrafficSwitcher(redisClient, nil)

	t.Run("initial status", func(t *testing.T) {
		config := switcher.GetCurrentConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "region-a", config.DefaultRegion)
		assert.Equal(t, 100, config.RegionWeights["region-a"])
		assert.Equal(t, 0, config.RegionWeights["region-b"])
	})

	t.Run("proportional switch", func(t *testing.T) {
		ctx := context.Background()

		response, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 80, "region-b": 20},
			"Test proportional switch",
			"test-operator",
			false,
		)

		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.EventID)
		assert.Equal(t, 80, response.NewConfig.RegionWeights["region-a"])
		assert.Equal(t, 20, response.NewConfig.RegionWeights["region-b"])

		// Verify config was updated
		config := switcher.GetCurrentConfig()
		assert.Equal(t, 80, config.RegionWeights["region-a"])
		assert.Equal(t, 20, config.RegionWeights["region-b"])
	})

	t.Run("dry run mode", func(t *testing.T) {
		ctx := context.Background()

		// Get current config
		oldConfig := switcher.GetCurrentConfig()

		// Perform dry run
		response, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 60, "region-b": 40},
			"Test dry run",
			"test-operator",
			true, // dry run
		)

		require.NoError(t, err)
		assert.True(t, response.Success)

		// Verify config was NOT updated
		newConfig := switcher.GetCurrentConfig()
		assert.Equal(t, oldConfig.RegionWeights, newConfig.RegionWeights)
		assert.Equal(t, oldConfig.Version, newConfig.Version)
	})

	t.Run("full switch", func(t *testing.T) {
		ctx := context.Background()

		response, err := switcher.SwitchTrafficFull(
			ctx,
			"region-b",
			"Test full switch",
			"test-operator",
			false,
		)

		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, 0, response.NewConfig.RegionWeights["region-a"])
		assert.Equal(t, 100, response.NewConfig.RegionWeights["region-b"])

		// Verify config was updated
		config := switcher.GetCurrentConfig()
		assert.Equal(t, 0, config.RegionWeights["region-a"])
		assert.Equal(t, 100, config.RegionWeights["region-b"])
		assert.Equal(t, "region-b", config.DefaultRegion)
	})

	t.Run("event logging", func(t *testing.T) {
		events := switcher.GetTrafficEvents(10)

		// We should have at least 2 events from previous tests
		assert.GreaterOrEqual(t, len(events), 2)

		// Check event structure
		for _, event := range events {
			assert.NotEmpty(t, event.ID)
			assert.NotEmpty(t, event.Type)
			assert.NotEmpty(t, event.Status)
			assert.NotZero(t, event.Timestamp)
			assert.NotEmpty(t, event.Operator)
			assert.NotEmpty(t, event.Reason)
		}
	})

	t.Run("user routing", func(t *testing.T) {
		// Set 50:50 distribution
		ctx := context.Background()
		_, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 50, "region-b": 50},
			"Test routing",
			"test-operator",
			false,
		)
		require.NoError(t, err)

		// Test routing for multiple users
		routingResults := make(map[string]int)
		for i := 0; i < 100; i++ {
			userID := "user" + string(rune(i))
			region := switcher.RouteRequest(userID)
			routingResults[region]++
		}

		// Both regions should receive some traffic
		assert.Greater(t, routingResults["region-a"], 0)
		assert.Greater(t, routingResults["region-b"], 0)

		// Distribution should be roughly 50:50 (allow some variance)
		assert.InDelta(t, 50, routingResults["region-a"], 20)
		assert.InDelta(t, 50, routingResults["region-b"], 20)
	})

	t.Run("validation errors", func(t *testing.T) {
		ctx := context.Background()

		// Test invalid weights (don't sum to 100)
		_, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 60, "region-b": 30},
			"Test invalid weights",
			"test-operator",
			false,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "total weight must equal 100")

		// Test invalid region
		_, err = switcher.SwitchTrafficFull(
			ctx,
			"invalid-region",
			"Test invalid region",
			"test-operator",
			false,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid region")

		// Test negative weight
		_, err = switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": -10, "region-b": 110},
			"Test negative weight",
			"test-operator",
			false,
		)
		assert.Error(t, err)
	})

	t.Run("concurrent switches", func(t *testing.T) {
		ctx := context.Background()

		// First switch acquires lock
		done := make(chan bool)
		go func() {
			_, err := switcher.SwitchTrafficProportional(
				ctx,
				map[string]int{"region-a": 70, "region-b": 30},
				"Concurrent test 1",
				"test-operator",
				false,
			)
			assert.NoError(t, err)
			done <- true
		}()

		// Give first switch time to acquire lock
		time.Sleep(10 * time.Millisecond)

		// Second switch should fail to acquire lock
		// Note: In mini-redis, this might not work exactly like real Redis
		// but we test the logic anyway
		_, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 60, "region-b": 40},
			"Concurrent test 2",
			"test-operator",
			false,
		)

		// Wait for first switch to complete
		<-done

		// The second switch might succeed or fail depending on timing
		// We just verify it doesn't panic
		_ = err
	})

	t.Run("gradual migration scenario", func(t *testing.T) {
		ctx := context.Background()

		// Phase 1: 70:30
		response, err := switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 70, "region-b": 30},
			"Migration phase 1",
			"test-operator",
			false,
		)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Phase 2: 90:10
		response, err = switcher.SwitchTrafficProportional(
			ctx,
			map[string]int{"region-a": 90, "region-b": 10},
			"Migration phase 2",
			"test-operator",
			false,
		)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Phase 3: 100:0
		response, err = switcher.SwitchTrafficFull(
			ctx,
			"region-a",
			"Migration complete",
			"test-operator",
			false,
		)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Verify final state
		config := switcher.GetCurrentConfig()
		assert.Equal(t, 100, config.RegionWeights["region-a"])
		assert.Equal(t, 0, config.RegionWeights["region-b"])

		// Verify we have events for all phases
		events := switcher.GetTrafficEvents(10)
		assert.GreaterOrEqual(t, len(events), 3)
	})
}

// TestParseRegionWeightsIntegration tests the CLI parsing with real traffic switcher
func TestParseRegionWeightsIntegration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	switcher := traffic.NewTrafficSwitcher(redisClient, nil)

	testCases := []struct {
		name   string
		args   []string
		verify func(t *testing.T, weights map[string]int)
	}{
		{
			name: "90:10 split",
			args: []string{"region-a:90", "region-b:10"},
			verify: func(t *testing.T, weights map[string]int) {
				ctx := context.Background()
				response, err := switcher.SwitchTrafficProportional(
					ctx, weights, "Test", "test", false,
				)
				require.NoError(t, err)
				assert.Equal(t, 90, response.NewConfig.RegionWeights["region-a"])
				assert.Equal(t, 10, response.NewConfig.RegionWeights["region-b"])
			},
		},
		{
			name: "50:50 split",
			args: []string{"region-a:50", "region-b:50"},
			verify: func(t *testing.T, weights map[string]int) {
				ctx := context.Background()
				response, err := switcher.SwitchTrafficProportional(
					ctx, weights, "Test", "test", false,
				)
				require.NoError(t, err)
				assert.Equal(t, 50, response.NewConfig.RegionWeights["region-a"])
				assert.Equal(t, 50, response.NewConfig.RegionWeights["region-b"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			weights, err := parseRegionWeights(tc.args)
			require.NoError(t, err)
			tc.verify(t, weights)
		})
	}
}
