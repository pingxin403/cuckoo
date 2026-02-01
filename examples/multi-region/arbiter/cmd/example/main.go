package arbiter

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// ExampleUsage demonstrates how to use the ArbiterClient in a real application
func ExampleUsage() {
	// Configuration for the arbiter client
	config := Config{
		ZookeeperHosts: []string{"zookeeper:2181"},
		RegionID:       "region-a",
		SessionTimeout: 10 * time.Second,
		ElectionTTL:    30 * time.Second,
		Logger:         log.New(os.Stdout, "[ARBITER] ", log.LstdFlags),
	}

	// Create arbiter client
	client, err := NewArbiterClient(config)
	if err != nil {
		log.Fatalf("Failed to create arbiter client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Report health status
	healthStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	err = client.ReportHealth(healthStatus)
	if err != nil {
		log.Printf("Failed to report health: %v", err)
	}

	// Example 2: Perform leader election
	result, err := client.ElectPrimary(ctx, healthStatus)
	if err != nil {
		log.Printf("Failed to elect primary: %v", err)
		return
	}

	fmt.Printf("Election Result:\n")
	fmt.Printf("  Leader: %s\n", result.Leader)
	fmt.Printf("  Is Primary: %v\n", result.IsPrimary)
	fmt.Printf("  Reason: %s\n", result.Reason)
	fmt.Printf("  TTL: %d seconds\n", result.TTL)

	// Example 3: Watch for leader changes
	go func() {
		err := client.WatchLeaderChanges(ctx, func(leader string) {
			log.Printf("Leader changed to: %s", leader)
		})
		if err != nil {
			log.Printf("Error watching leader changes: %v", err)
		}
	}()

	// Example 4: Periodic health reporting and election
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check health of local services
			currentHealth := checkLocalServices()

			// Report health
			err := client.ReportHealth(currentHealth)
			if err != nil {
				log.Printf("Failed to report health: %v", err)
				continue
			}

			// Perform election if needed
			result, err := client.ElectPrimary(ctx, currentHealth)
			if err != nil {
				log.Printf("Failed to elect primary: %v", err)
				continue
			}

			if result.IsPrimary {
				log.Printf("This region is PRIMARY - handling write traffic")
				// Handle primary region responsibilities
			} else {
				log.Printf("This region is SECONDARY (leader: %s) - read-only mode", result.Leader)
				// Handle secondary region responsibilities
			}

		case <-ctx.Done():
			return
		}
	}
}

// checkLocalServices simulates checking the health of local services
func checkLocalServices() map[string]bool {
	// In a real implementation, this would check:
	// - Database connectivity
	// - Redis connectivity
	// - IM service health
	// - Network connectivity to peer region

	return map[string]bool{
		"im-service": checkIMService(),
		"redis":      checkRedis(),
		"database":   checkDatabase(),
	}
}

func checkIMService() bool {
	// Simulate IM service health check
	// In reality, this would make an HTTP request to /health endpoint
	return true
}

func checkRedis() bool {
	// Simulate Redis health check
	// In reality, this would ping Redis
	return true
}

func checkDatabase() bool {
	// Simulate database health check
	// In reality, this would execute a simple query
	return true
}

// ExampleFailoverScenario demonstrates how the arbiter handles failover
func ExampleFailoverScenario() {
	config := Config{
		ZookeeperHosts: []string{"zookeeper:2181"},
		RegionID:       "region-a",
		SessionTimeout: 10 * time.Second,
		ElectionTTL:    30 * time.Second,
		Logger:         log.New(os.Stdout, "[FAILOVER] ", log.LstdFlags),
	}

	client, err := NewArbiterClient(config)
	if err != nil {
		log.Fatalf("Failed to create arbiter client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Scenario 1: Normal operation - all services healthy
	fmt.Println("=== Scenario 1: Normal Operation ===")
	healthyStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}

	result, err := client.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		log.Printf("Election failed: %v", err)
		return
	}
	fmt.Printf("Normal operation: Leader=%s, IsPrimary=%v\n", result.Leader, result.IsPrimary)

	// Scenario 2: Database failure
	fmt.Println("\n=== Scenario 2: Database Failure ===")
	dbFailureStatus := map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   false, // Database failed
	}

	result, err = client.ElectPrimary(ctx, dbFailureStatus)
	if err != nil {
		log.Printf("Election failed: %v", err)
		return
	}
	fmt.Printf("Database failure: Leader=%s, IsPrimary=%v, Reason=%s\n",
		result.Leader, result.IsPrimary, result.Reason)

	// Scenario 3: Complete region failure
	fmt.Println("\n=== Scenario 3: Complete Region Failure ===")
	completeFailureStatus := map[string]bool{
		"im-service": false,
		"redis":      false,
		"database":   false,
	}

	result, err = client.ElectPrimary(ctx, completeFailureStatus)
	if err != nil {
		log.Printf("Election failed: %v", err)
		return
	}
	fmt.Printf("Complete failure: Leader=%s, IsPrimary=%v, Reason=%s\n",
		result.Leader, result.IsPrimary, result.Reason)

	// Scenario 4: Recovery
	fmt.Println("\n=== Scenario 4: Recovery ===")
	result, err = client.ElectPrimary(ctx, healthyStatus)
	if err != nil {
		log.Printf("Election failed: %v", err)
		return
	}
	fmt.Printf("Recovery: Leader=%s, IsPrimary=%v, Reason=%s\n",
		result.Leader, result.IsPrimary, result.Reason)
}

// ExampleSplitBrainPrevention demonstrates how the arbiter prevents split-brain
func ExampleSplitBrainPrevention() {
	fmt.Println("=== Split-Brain Prevention Example ===")
	fmt.Println("This example shows how the arbiter prevents split-brain scenarios")
	fmt.Println("by using Zookeeper as a distributed consensus mechanism.")
	fmt.Println()
	fmt.Println("Key mechanisms:")
	fmt.Println("1. Distributed locks ensure only one region can be primary")
	fmt.Println("2. Health reporting provides visibility into region status")
	fmt.Println("3. Deterministic election rules prevent conflicts")
	fmt.Println("4. TTL-based leadership prevents stale leaders")
	fmt.Println()
	fmt.Println("Network partition scenarios:")
	fmt.Println("- If region-a loses connection to Zookeeper: becomes read-only")
	fmt.Println("- If region-b loses connection to Zookeeper: becomes read-only")
	fmt.Println("- Only the region that can reach Zookeeper can be primary")
	fmt.Println("- When partition heals, normal election resumes")
}

// ExampleMonitoring shows how to monitor arbiter health and elections
func ExampleMonitoring() {
	config := Config{
		ZookeeperHosts: []string{"zookeeper:2181"},
		RegionID:       "region-a",
		SessionTimeout: 10 * time.Second,
		ElectionTTL:    30 * time.Second,
		Logger:         log.New(os.Stdout, "[MONITOR] ", log.LstdFlags),
	}

	client, err := NewArbiterClient(config)
	if err != nil {
		log.Fatalf("Failed to create arbiter client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get election history
	history, err := client.GetElectionHistory(ctx, 10)
	if err != nil {
		log.Printf("Failed to get election history: %v", err)
		return
	}

	fmt.Println("=== Election History ===")
	for i, event := range history {
		fmt.Printf("%d. Leader: %s, Timestamp: %v, Reason: %s\n",
			i+1, event["leader"], event["timestamp"], event["reason"])
	}

	// Get current leader
	leader, err := client.GetCurrentLeader(ctx)
	if err != nil {
		log.Printf("Failed to get current leader: %v", err)
		return
	}

	fmt.Printf("\nCurrent Leader: %s\n", leader)

	// Get health status
	health := client.GetHealthStatus()
	fmt.Printf("Health Status: %v\n", health)
	fmt.Printf("Overall Health: %v\n", client.IsHealthy())
}
