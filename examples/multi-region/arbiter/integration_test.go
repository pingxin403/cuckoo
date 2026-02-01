package arbiter

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

// TestArbiterIntegration_FullScenario tests a complete failover scenario
func TestArbiterIntegration_FullScenario(t *testing.T) {
	// Skip if ZK_TEST_HOSTS environment variable is not set
	zkHosts := os.Getenv("ZK_TEST_HOSTS")
	if zkHosts == "" {
		t.Skip("Skipping integration test: ZK_TEST_HOSTS not set")
	}

	logger := log.New(os.Stdout, "[INTEGRATION] ", log.LstdFlags)

	// Create two arbiter clients for region-a and region-b
	configA := Config{
		ZookeeperHosts: []string{zkHosts},
		RegionID:       "region-a",
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    10 * time.Second,
		Logger:         logger,
	}

	configB := Config{
		ZookeeperHosts: []string{zkHosts},
		RegionID:       "region-b",
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    10 * time.Second,
		Logger:         logger,
	}

	clientA, err := NewArbiterClient(configA)
	if err != nil {
		t.Fatalf("Failed to create client A: %v", err)
	}
	defer clientA.Close()

	clientB, err := NewArbiterClient(configB)
	if err != nil {
		t.Fatalf("Failed to create client B: %v", err)
	}
	defer clientB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test Scenario 1: Both regions healthy - region-a should be preferred
	t.Log("=== Scenario 1: Both regions healthy ===")

	healthyStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	// Both regions report healthy
	err = clientA.ReportHealth(healthyStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-a: %v", err)
	}

	err = clientB.ReportHealth(healthyStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-b: %v", err)
	}

	// Wait a bit for health reports to propagate
	time.Sleep(2 * time.Second)

	// Both regions perform election
	resultA, err := clientA.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		t.Errorf("Election failed for region-a: %v", err)
	}

	resultB, err := clientB.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		t.Errorf("Election failed for region-b: %v", err)
	}

	// Verify region-a is preferred when both are healthy
	if resultA.Leader != "region-a" || !resultA.IsPrimary {
		t.Errorf("Expected region-a to be primary, got leader=%s, is_primary=%v",
			resultA.Leader, resultA.IsPrimary)
	}

	if resultB.Leader != "region-a" || resultB.IsPrimary {
		t.Errorf("Expected region-b to see region-a as leader, got leader=%s, is_primary=%v",
			resultB.Leader, resultB.IsPrimary)
	}

	t.Logf("Scenario 1 passed: region-a is primary, region-b is secondary")

	// Test Scenario 2: Region-a fails - region-b should become primary
	t.Log("=== Scenario 2: Region-a failure ===")

	failedStatus := map[string]bool{
		"im-service": false, // Service failed
		"redis":      true,
		"database":   true,
	}

	// Region-a reports failure
	err = clientA.ReportHealth(failedStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-a: %v", err)
	}

	// Region-b still healthy
	err = clientB.ReportHealth(healthyStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-b: %v", err)
	}

	// Wait for health reports to propagate
	time.Sleep(2 * time.Second)

	// Perform elections
	resultA, err = clientA.ElectPrimary(ctx, failedStatus)
	if err != nil {
		t.Errorf("Election failed for region-a: %v", err)
	}

	resultB, err = clientB.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		t.Errorf("Election failed for region-b: %v", err)
	}

	// Verify region-b becomes primary when region-a fails
	if resultB.Leader != "region-b" || !resultB.IsPrimary {
		t.Errorf("Expected region-b to be primary after region-a failure, got leader=%s, is_primary=%v",
			resultB.Leader, resultB.IsPrimary)
	}

	if resultA.Leader != "region-b" || resultA.IsPrimary {
		t.Errorf("Expected region-a to see region-b as leader, got leader=%s, is_primary=%v",
			resultA.Leader, resultA.IsPrimary)
	}

	t.Logf("Scenario 2 passed: region-b became primary after region-a failure")

	// Test Scenario 3: Both regions fail - no primary
	t.Log("=== Scenario 3: Both regions fail ===")

	// Both regions report failure
	err = clientA.ReportHealth(failedStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-a: %v", err)
	}

	err = clientB.ReportHealth(failedStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-b: %v", err)
	}

	// Wait for health reports to propagate
	time.Sleep(2 * time.Second)

	// Perform elections
	resultA, err = clientA.ElectPrimary(ctx, failedStatus)
	if err != nil {
		t.Errorf("Election failed for region-a: %v", err)
	}

	resultB, err = clientB.ElectPrimary(ctx, failedStatus)
	if err != nil {
		t.Errorf("Election failed for region-b: %v", err)
	}

	// Verify no leader when both regions are unhealthy
	if resultA.Leader != "" {
		t.Errorf("Expected no leader when both regions unhealthy, got leader=%s", resultA.Leader)
	}

	if resultB.Leader != "" {
		t.Errorf("Expected no leader when both regions unhealthy, got leader=%s", resultB.Leader)
	}

	t.Logf("Scenario 3 passed: no primary when both regions failed")

	// Test Scenario 4: Recovery - region-a recovers and becomes primary again
	t.Log("=== Scenario 4: Region-a recovery ===")

	// Region-a recovers
	err = clientA.ReportHealth(healthyStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-a: %v", err)
	}

	// Region-b still failed
	err = clientB.ReportHealth(failedStatus)
	if err != nil {
		t.Errorf("Failed to report health for region-b: %v", err)
	}

	// Wait for health reports to propagate
	time.Sleep(2 * time.Second)

	// Perform elections
	resultA, err = clientA.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		t.Errorf("Election failed for region-a: %v", err)
	}

	resultB, err = clientB.ElectPrimary(ctx, failedStatus)
	if err != nil {
		t.Errorf("Election failed for region-b: %v", err)
	}

	// Verify region-a becomes primary again after recovery
	if resultA.Leader != "region-a" || !resultA.IsPrimary {
		t.Errorf("Expected region-a to be primary after recovery, got leader=%s, is_primary=%v",
			resultA.Leader, resultA.IsPrimary)
	}

	t.Logf("Scenario 4 passed: region-a became primary after recovery")

	t.Log("=== All integration scenarios passed ===")
}

// TestArbiterIntegration_LeaderWatch tests the leader change watching functionality
func TestArbiterIntegration_LeaderWatch(t *testing.T) {
	zkHosts := os.Getenv("ZK_TEST_HOSTS")
	if zkHosts == "" {
		t.Skip("Skipping integration test: ZK_TEST_HOSTS not set")
	}

	logger := log.New(os.Stdout, "[WATCH-TEST] ", log.LstdFlags)

	config := Config{
		ZookeeperHosts: []string{zkHosts},
		RegionID:       "region-watch-test",
		SessionTimeout: 5 * time.Second,
		ElectionTTL:    10 * time.Second,
		Logger:         logger,
	}

	client, err := NewArbiterClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channel to receive leader change notifications
	leaderChanges := make(chan string, 10)

	// Start watching for leader changes
	go func() {
		err := client.WatchLeaderChanges(ctx, func(leader string) {
			leaderChanges <- leader
		})
		if err != nil && err != context.Canceled {
			t.Errorf("Leader watch failed: %v", err)
		}
	}()

	// Perform an election to trigger a leader change
	healthStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	_, err = client.ElectPrimary(ctx, healthStatus)
	if err != nil {
		t.Errorf("Election failed: %v", err)
	}

	// Wait for leader change notification
	select {
	case leader := <-leaderChanges:
		t.Logf("Received leader change notification: %s", leader)
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for leader change notification")
	}
}

// Example_arbiterUsage demonstrates basic arbiter usage
func Example_arbiterUsage() {
	config := Config{
		ZookeeperHosts: []string{"localhost:2181"},
		RegionID:       "region-a",
		SessionTimeout: 10 * time.Second,
		ElectionTTL:    30 * time.Second,
	}

	client, err := NewArbiterClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Report health status
	healthStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	err = client.ReportHealth(healthStatus)
	if err != nil {
		log.Printf("Failed to report health: %v", err)
	}

	// Perform leader election
	result, err := client.ElectPrimary(ctx, healthStatus)
	if err != nil {
		log.Printf("Election failed: %v", err)
		return
	}

	if result.IsPrimary {
		fmt.Println("This region is the PRIMARY")
	} else {
		fmt.Printf("This region is SECONDARY (leader: %s)\n", result.Leader)
	}

	// Get current leader
	leader, err := client.GetCurrentLeader(ctx)
	if err != nil {
		log.Printf("Failed to get leader: %v", err)
		return
	}

	fmt.Printf("Current leader: %s\n", leader)
}
