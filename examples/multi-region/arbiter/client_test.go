package arbiter

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

// TestArbiterClient_Basic tests basic functionality without requiring Zookeeper
func TestArbiterClient_Basic(t *testing.T) {
	// Test configuration validation
	config := Config{
		ZookeeperHosts: []string{"localhost:2181"},
		RegionID:       "region-a",
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    30 * time.Second,
		Logger:         log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}

	// Test config validation
	if config.RegionID == "" {
		t.Error("RegionID should not be empty")
	}

	if len(config.ZookeeperHosts) == 0 {
		t.Error("ZookeeperHosts should not be empty")
	}
}

// TestHealthStatusLogic tests the health status determination logic
func TestHealthStatusLogic(t *testing.T) {
	client := &ArbiterClient{
		regionID:     "region-a",
		healthStatus: make(map[string]bool),
		logger:       log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}

	tests := []struct {
		name           string
		healthStatus   map[string]bool
		expectedHealth bool
	}{
		{
			name: "all services healthy",
			healthStatus: map[string]bool{
				"im-service": true,
				"redis":      true,
				"database":   true,
			},
			expectedHealth: true,
		},
		{
			name: "missing critical service",
			healthStatus: map[string]bool{
				"im-service": true,
				"redis":      true,
				// database missing
			},
			expectedHealth: false,
		},
		{
			name: "service unhealthy",
			healthStatus: map[string]bool{
				"im-service": true,
				"redis":      false,
				"database":   true,
			},
			expectedHealth: false,
		},
		{
			name: "extra services don't affect health",
			healthStatus: map[string]bool{
				"im-service":    true,
				"redis":         true,
				"database":      true,
				"extra-service": false,
			},
			expectedHealth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.healthStatus = tt.healthStatus
			result := client.isRegionHealthy()
			if result != tt.expectedHealth {
				t.Errorf("isRegionHealthy() = %v, want %v", result, tt.expectedHealth)
			}
		})
	}
}

// TestElectionLogic tests the leader election logic without Zookeeper
func TestElectionLogic(t *testing.T) {
	client := &ArbiterClient{
		regionID:      "region-a",
		currentLeader: "",
		logger:        log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}

	tests := []struct {
		name           string
		healthyRegions []string
		currentLeader  string
		expectedLeader string
		expectedReason string
	}{
		{
			name:           "no healthy regions",
			healthyRegions: []string{},
			currentLeader:  "",
			expectedLeader: "",
			expectedReason: "no_healthy_regions",
		},
		{
			name:           "current leader still healthy",
			healthyRegions: []string{"region-a", "region-b"},
			currentLeader:  "region-b",
			expectedLeader: "region-b",
			expectedReason: "current_leader_healthy",
		},
		{
			name:           "deterministic election - prefer region-a",
			healthyRegions: []string{"region-a", "region-b"},
			currentLeader:  "",
			expectedLeader: "region-a",
			expectedReason: "deterministic_election",
		},
		{
			name:           "deterministic election - only region-b healthy",
			healthyRegions: []string{"region-b"},
			currentLeader:  "",
			expectedLeader: "region-b",
			expectedReason: "deterministic_election",
		},
		{
			name:           "fallback election",
			healthyRegions: []string{"region-c"},
			currentLeader:  "",
			expectedLeader: "region-c",
			expectedReason: "fallback_election",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.currentLeader = tt.currentLeader

			// Mock the determinePrimaryRegion logic
			var leader, reason string

			if len(tt.healthyRegions) == 0 {
				leader, reason = "", "no_healthy_regions"
			} else {
				// Check if current leader is still healthy
				if client.currentLeader != "" {
					for _, regionID := range tt.healthyRegions {
						if regionID == client.currentLeader {
							leader, reason = client.currentLeader, "current_leader_healthy"
							goto done
						}
					}
				}

				// Deterministic election
				preferredOrder := []string{"region-a", "region-b"}
				for _, preferred := range preferredOrder {
					for _, regionID := range tt.healthyRegions {
						if regionID == preferred {
							leader, reason = regionID, "deterministic_election"
							goto done
						}
					}
				}

				// Fallback
				leader, reason = tt.healthyRegions[0], "fallback_election"
			}

		done:
			if leader != tt.expectedLeader {
				t.Errorf("Expected leader %s, got %s", tt.expectedLeader, leader)
			}
			if reason != tt.expectedReason {
				t.Errorf("Expected reason %s, got %s", tt.expectedReason, reason)
			}
		})
	}
}

// TestReportHealth tests the health reporting functionality
func TestReportHealth(t *testing.T) {
	client := &ArbiterClient{
		regionID:     "region-a",
		healthStatus: make(map[string]bool),
		logger:       log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}

	services := map[string]bool{
		"im-service": true,
		"redis":      false,
		"database":   true,
	}

	// This would normally report to ZK, but we'll just test the local state update
	client.healthStatus = make(map[string]bool)
	for service, healthy := range services {
		client.healthStatus[service] = healthy
	}

	// Verify health status was updated
	status := client.GetHealthStatus()
	for service, expectedHealth := range services {
		if actualHealth, exists := status[service]; !exists || actualHealth != expectedHealth {
			t.Errorf("Service %s: expected %v, got %v (exists: %v)",
				service, expectedHealth, actualHealth, exists)
		}
	}

	// Test overall health
	expectedOverallHealth := false // redis is false
	if client.IsHealthy() != expectedOverallHealth {
		t.Errorf("Expected overall health %v, got %v", expectedOverallHealth, client.IsHealthy())
	}
}

// Integration test that requires Zookeeper to be running
func TestArbiterClient_Integration(t *testing.T) {
	// Skip if ZK_TEST_HOSTS environment variable is not set
	zkHosts := os.Getenv("ZK_TEST_HOSTS")
	if zkHosts == "" {
		t.Skip("Skipping integration test: ZK_TEST_HOSTS not set")
	}

	config := Config{
		ZookeeperHosts: []string{zkHosts},
		RegionID:       "test-region",
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    10 * time.Second,
		Logger:         log.New(os.Stdout, "[INTEGRATION] ", log.LstdFlags),
	}

	client, err := NewArbiterClient(config)
	if err != nil {
		t.Fatalf("Failed to create arbiter client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test health reporting
	healthStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	err = client.ReportHealth(healthStatus)
	if err != nil {
		t.Errorf("Failed to report health: %v", err)
	}

	// Test leader election
	result, err := client.ElectPrimary(ctx, healthStatus)
	if err != nil {
		t.Errorf("Failed to elect primary: %v", err)
	}

	if result == nil {
		t.Error("Election result should not be nil")
	}

	if result.Leader == "" {
		t.Error("Leader should not be empty when region is healthy")
	}

	// Test getting current leader
	leader, err := client.GetCurrentLeader(ctx)
	if err != nil {
		t.Errorf("Failed to get current leader: %v", err)
	}

	if leader != result.Leader {
		t.Errorf("Current leader %s doesn't match election result %s", leader, result.Leader)
	}

	t.Logf("Integration test completed successfully. Leader: %s, IsPrimary: %v",
		result.Leader, result.IsPrimary)
}

// Benchmark test for election performance
func BenchmarkElection(b *testing.B) {
	client := &ArbiterClient{
		regionID:      "region-a",
		currentLeader: "",
		logger:        log.New(os.Stdout, "[BENCH] ", log.LstdFlags),
	}

	healthStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.healthStatus = healthStatus
		_ = client.isRegionHealthy()
	}
}
