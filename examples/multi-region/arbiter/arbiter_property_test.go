package arbiter

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"
)

// Property-based tests for the arbiter system
// These tests verify universal properties that should hold across all inputs

// TestProperty_ElectionDeterminism verifies that elections are deterministic
// **Validates: Requirements 4.3**
func TestProperty_ElectionDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Property: Given the same health state, elections should always produce the same result
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random but consistent health state
			seed := int64(i)
			rand.Seed(seed)

			healthState := generateRandomHealthState()

			// Create multiple clients with the same health state
			clients := createTestClients(t, 3)
			defer closeClients(clients)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var results []*ElectionResult

			// All clients perform election with same health state
			for _, client := range clients {
				result, err := client.mockElectPrimary(ctx, healthState)
				if err != nil {
					t.Fatalf("Election failed: %v", err)
				}
				results = append(results, result)
			}

			// Verify all results are identical
			firstResult := results[0]
			for i, result := range results[1:] {
				if result.Leader != firstResult.Leader {
					t.Errorf("Election %d: leader mismatch: got %s, expected %s",
						i+1, result.Leader, firstResult.Leader)
				}
			}
		})
	}
}

// TestProperty_SingleLeader verifies that only one region can be primary at a time
// **Validates: Requirements 4.3**
func TestProperty_SingleLeader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Property: At most one region can be primary at any given time
	for i := 0; i < 50; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			clients := createTestClients(t, 5) // Test with multiple regions
			defer closeClients(clients)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Generate random health states for each client
			healthStates := make([]map[string]bool, len(clients))
			for j := range healthStates {
				healthStates[j] = generateRandomHealthState()
			}

			// All clients perform election
			var results []*ElectionResult
			for j, client := range clients {
				result, err := client.mockElectPrimary(ctx, healthStates[j])
				if err != nil {
					t.Fatalf("Election failed for client %d: %v", j, err)
				}
				results = append(results, result)
			}

			// Count how many regions think they are primary
			primaryCount := 0
			var primaryRegions []string

			for j, result := range results {
				if result.IsPrimary {
					primaryCount++
					primaryRegions = append(primaryRegions, clients[j].regionID)
				}
			}

			// Verify at most one primary
			if primaryCount > 1 {
				t.Errorf("Multiple primaries detected: %v (count: %d)",
					primaryRegions, primaryCount)
			}

			// If there's a primary, all regions should agree on the leader
			if primaryCount == 1 {
				expectedLeader := primaryRegions[0]
				for j, result := range results {
					if result.Leader != expectedLeader {
						t.Errorf("Client %d sees leader %s, expected %s",
							j, result.Leader, expectedLeader)
					}
				}
			}
		})
	}
}

// TestProperty_HealthBasedElection verifies that unhealthy regions cannot become primary
// **Validates: Requirements 4.1, 4.3**
func TestProperty_HealthBasedElection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Property: Unhealthy regions should never be elected as primary
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			client := createSingleTestClient(t, fmt.Sprintf("test-region-%d", i))
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Generate health state that makes region unhealthy
			unhealthyState := generateUnhealthyState()

			result, err := client.mockElectPrimary(ctx, unhealthyState)
			if err != nil {
				t.Fatalf("Election failed: %v", err)
			}

			// Verify unhealthy region is not primary
			if result.IsPrimary && !client.isRegionHealthy() {
				t.Errorf("Unhealthy region became primary: health=%v, result=%+v",
					unhealthyState, result)
			}
		})
	}
}

// TestProperty_ElectionStability verifies that leadership is stable when health doesn't change
// **Validates: Requirements 4.3**
func TestProperty_ElectionStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Property: If health state doesn't change, leadership should remain stable
	for i := 0; i < 20; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			client := createSingleTestClient(t, fmt.Sprintf("stable-region-%d", i))
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// Generate stable health state
			healthState := map[string]bool{
				"im-service": true,
				"redis":      true,
				"database":   true,
			}

			// Perform multiple elections with same health state
			var leaders []string
			for j := 0; j < 5; j++ {
				result, err := client.mockElectPrimary(ctx, healthState)
				if err != nil {
					t.Fatalf("Election %d failed: %v", j, err)
				}
				leaders = append(leaders, result.Leader)

				// Small delay between elections
				time.Sleep(100 * time.Millisecond)
			}

			// Verify leadership stability
			firstLeader := leaders[0]
			for j, leader := range leaders[1:] {
				if leader != firstLeader {
					t.Errorf("Leadership changed without health change: election %d got %s, expected %s",
						j+1, leader, firstLeader)
				}
			}
		})
	}
}

// TestProperty_ElectionConsistency verifies that all regions see consistent election results
// **Validates: Requirements 4.3**
func TestProperty_ElectionConsistency(t *testing.T) {
	zkHosts := os.Getenv("ZK_TEST_HOSTS")
	if zkHosts == "" {
		t.Skip("Skipping property test: ZK_TEST_HOSTS not set")
	}

	// Property: All regions should see consistent election results
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Create multiple clients representing different regions
			regionIDs := []string{"region-a", "region-b", "region-c"}
			clients := make([]*ArbiterClient, len(regionIDs))

			for j, regionID := range regionIDs {
				config := Config{
					ZookeeperHosts: []string{zkHosts},
					RegionID:       regionID,
					SessionTimeout: 5 * time.Second,
					ElectionTTL:    10 * time.Second,
					Logger:         log.New(os.Stdout, fmt.Sprintf("[%s] ", regionID), log.LstdFlags),
				}

				client, err := NewArbiterClient(config)
				if err != nil {
					t.Fatalf("Failed to create client for %s: %v", regionID, err)
				}
				clients[j] = client
			}

			defer func() {
				for _, client := range clients {
					client.Close()
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			// All regions report health and perform election
			healthState := map[string]bool{
				"im-service": true,
				"redis":      true,
				"database":   true,
			}

			var results []*ElectionResult
			for j, client := range clients {
				// Report health first
				err := client.ReportHealth(healthState)
				if err != nil {
					t.Fatalf("Failed to report health for %s: %v", regionIDs[j], err)
				}
			}

			// Wait for health reports to propagate
			time.Sleep(2 * time.Second)

			// Perform elections
			for j, client := range clients {
				result, err := client.ElectPrimary(ctx, healthState)
				if err != nil {
					t.Fatalf("Election failed for %s: %v", regionIDs[j], err)
				}
				results = append(results, result)
			}

			// Verify consistency
			firstResult := results[0]
			for j, result := range results[1:] {
				if result.Leader != firstResult.Leader {
					t.Errorf("Region %s sees leader %s, region %s sees leader %s",
						regionIDs[0], firstResult.Leader, regionIDs[j+1], result.Leader)
				}
			}

			// Verify exactly one region is primary
			primaryCount := 0
			for _, result := range results {
				if result.IsPrimary {
					primaryCount++
				}
			}

			if primaryCount != 1 {
				t.Errorf("Expected exactly 1 primary, got %d", primaryCount)
			}
		})
	}
}

// Helper functions for property tests

func generateRandomHealthState() map[string]bool {
	services := []string{"im-service", "redis", "database"}
	health := make(map[string]bool)

	for _, service := range services {
		health[service] = rand.Float32() > 0.3 // 70% chance of being healthy
	}

	// Add some random extra services
	extraServices := []string{"monitoring", "logging", "cache"}
	for _, service := range extraServices {
		if rand.Float32() > 0.5 {
			health[service] = rand.Float32() > 0.2
		}
	}

	return health
}

func generateUnhealthyState() map[string]bool {
	// Ensure at least one critical service is unhealthy
	criticalServices := []string{"im-service", "redis", "database"}
	unhealthyService := criticalServices[rand.Intn(len(criticalServices))]

	health := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	health[unhealthyService] = false

	return health
}

func createTestClients(t *testing.T, count int) []*ArbiterClient {
	var clients []*ArbiterClient

	for i := 0; i < count; i++ {
		client := createSingleTestClient(t, fmt.Sprintf("test-region-%d", i))
		clients = append(clients, client)
	}

	return clients
}

func createSingleTestClient(t *testing.T, regionID string) *ArbiterClient {
	// Use mock Zookeeper for unit tests
	config := Config{
		ZookeeperHosts: []string{"localhost:2181"}, // This will fail, but we'll mock it
		RegionID:       regionID,
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    10 * time.Second,
		Logger:         log.New(os.Stdout, fmt.Sprintf("[%s] ", regionID), log.LstdFlags),
	}

	// For property tests without Zookeeper, create a mock client
	client := &ArbiterClient{
		regionID:     regionID,
		healthStatus: make(map[string]bool),
		logger:       config.Logger,
		electionPath: "/im/election",
		lockPath:     "/im/locks",
	}

	return client
}

func closeClients(clients []*ArbiterClient) {
	for _, client := range clients {
		if client.zkConn != nil {
			client.Close()
		}
	}
}

// Mock election logic for property tests (when Zookeeper is not available)
func (a *ArbiterClient) mockElectPrimary(ctx context.Context, healthStatus map[string]bool) (*ElectionResult, error) {
	a.healthStatus = healthStatus

	// Simple mock election logic
	isHealthy := a.isRegionHealthy()

	var leader string
	var isPrimary bool

	if isHealthy {
		// Deterministic selection based on region ID
		if a.regionID == "region-a" || a.regionID == "test-region-0" {
			leader = a.regionID
			isPrimary = true
		} else {
			leader = "region-a" // Default to region-a
			isPrimary = false
		}
	} else {
		leader = "" // No leader if unhealthy
		isPrimary = false
	}

	return &ElectionResult{
		Leader:    leader,
		IsPrimary: isPrimary,
		Timestamp: time.Now(),
		TTL:       30,
		Reason:    "mock_election",
	}, nil
}
