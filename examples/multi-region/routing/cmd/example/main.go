package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/multi-region/routing"
)

func main() {
	// Create logger
	logger := log.New(os.Stdout, "[GeoRouter] ", log.LstdFlags|log.Lshortfile)

	// Create geo router configuration
	config := routing.DefaultGeoRouterConfig()
	config.Port = 8081
	config.LogRequests = true

	// Create geo router
	router := routing.NewGeoRouter("region-a", config, logger)

	// Start router in a goroutine
	go func() {
		logger.Println("Starting geo router on http://localhost:8081")
		if err := router.Start(); err != nil {
			logger.Printf("Router server error: %v", err)
		}
	}()

	// Start demo client to test routing
	go runDemoClient(logger)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("Geo router is running. Available endpoints:")
	logger.Println("  - http://localhost:8081/health")
	logger.Println("  - http://localhost:8081/route")
	logger.Println("  - http://localhost:8081/regions")
	logger.Println("  - http://localhost:8081/rules")
	logger.Println("  - http://localhost:8081/status")
	logger.Println("Press Ctrl+C to stop...")

	<-sigChan
	logger.Println("Shutting down...")

	// Stop router
	if err := router.Stop(); err != nil {
		logger.Printf("Error stopping router: %v", err)
	}

	logger.Println("Geo router stopped")
}

// runDemoClient demonstrates various routing scenarios
func runDemoClient(logger *log.Logger) {
	// Wait for server to start
	time.Sleep(2 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://localhost:8081"

	scenarios := []struct {
		name    string
		headers map[string]string
		query   string
	}{
		{
			name: "Header-based routing to region-b",
			headers: map[string]string{
				"X-Target-Region": "region-b",
			},
		},
		{
			name: "Geographic routing - North China",
			headers: map[string]string{
				"X-Forwarded-For": "10.1.1.1",
			},
		},
		{
			name: "Geographic routing - South China",
			headers: map[string]string{
				"X-Forwarded-For": "10.2.1.1",
			},
		},
		{
			name: "User ID hash routing",
			headers: map[string]string{
				"X-User-ID": "user12345",
			},
		},
		{
			name: "Combined routing (geo + user)",
			headers: map[string]string{
				"X-Forwarded-For": "10.1.1.1",
				"X-User-ID":       "user67890",
			},
		},
		{
			name:    "Default fallback routing",
			headers: map[string]string{},
		},
	}

	logger.Println("\n=== Running Geo Router Demo ===")

	for i, scenario := range scenarios {
		logger.Printf("\n--- Scenario %d: %s ---", i+1, scenario.name)

		// Create request
		req, err := http.NewRequest("GET", baseURL+"/route", nil)
		if err != nil {
			logger.Printf("Error creating request: %v", err)
			continue
		}

		// Add headers
		for key, value := range scenario.headers {
			req.Header.Set(key, value)
		}

		// Make request
		resp, err := client.Do(req)
		if err != nil {
			logger.Printf("Error making request: %v", err)
			continue
		}

		// Read response
		if resp.StatusCode == http.StatusOK {
			logger.Printf("✓ Routing successful (status: %d)", resp.StatusCode)
		} else {
			logger.Printf("✗ Routing failed (status: %d)", resp.StatusCode)
		}

		resp.Body.Close()

		// Small delay between requests
		time.Sleep(500 * time.Millisecond)
	}

	// Test other endpoints
	logger.Println("\n=== Testing Other Endpoints ===")

	endpoints := []string{
		"/health",
		"/regions",
		"/rules",
		"/status",
		"/regions/region-a",
		"/regions/region-b",
	}

	for _, endpoint := range endpoints {
		logger.Printf("\nTesting %s", endpoint)

		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			logger.Printf("✗ Error: %v", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			logger.Printf("✓ Success (status: %d)", resp.StatusCode)
		} else {
			logger.Printf("✗ Failed (status: %d)", resp.StatusCode)
		}

		resp.Body.Close()
	}

	logger.Println("\n=== Demo Complete ===")
}

// Example of how to integrate geo router with an existing HTTP service
func integrateWithExistingService() {
	// This shows how you might integrate the geo router with an existing service

	logger := log.New(os.Stdout, "[Integration] ", log.LstdFlags)
	config := routing.DefaultGeoRouterConfig()
	router := routing.NewGeoRouter("region-a", config, logger)

	// Create HTTP handler that uses geo routing
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// Get routing decision
		decision := router.RouteRequest(r)

		// Add routing information to response headers
		w.Header().Set("X-Routed-To", decision.TargetRegion)
		w.Header().Set("X-Routing-Reason", decision.Reason)
		w.Header().Set("X-Routing-Confidence", fmt.Sprintf("%.2f", decision.Confidence))

		// Log routing decision
		logger.Printf("Request routed to %s (reason: %s, confidence: %.2f)",
			decision.TargetRegion, decision.Reason, decision.Confidence)

		// In a real implementation, you would:
		// 1. Proxy the request to the target region
		// 2. Or redirect the client to the appropriate endpoint
		// 3. Or handle the request locally if this is the target region

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Request routed to: %s\n", decision.TargetRegion)
		fmt.Fprintf(w, "Reason: %s\n", decision.Reason)
		fmt.Fprintf(w, "Confidence: %.2f\n", decision.Confidence)
	})

	logger.Println("Integrated service would be available on /api/")
}

// Example of custom routing rules
func exampleCustomRules() {
	logger := log.New(os.Stdout, "[CustomRules] ", log.LstdFlags)
	config := routing.DefaultGeoRouterConfig()
	router := routing.NewGeoRouter("region-a", config, logger)

	// In a real implementation, you might add custom rules like:
	// - Route premium users to specific regions
	// - Route based on API version
	// - Route based on request size or type
	// - Route based on time of day
	// - Route based on load balancing algorithms

	logger.Println("Custom routing rules would be configured here")

	// Example of how you might check routing for different scenarios
	scenarios := []struct {
		description string
		headers     map[string]string
	}{
		{
			description: "Premium user routing",
			headers: map[string]string{
				"X-User-Tier": "premium",
				"X-User-ID":   "premium_user_123",
			},
		},
		{
			description: "API version routing",
			headers: map[string]string{
				"X-API-Version": "v2",
			},
		},
		{
			description: "Load balancing routing",
			headers: map[string]string{
				"X-Request-Type": "heavy_computation",
			},
		},
	}

	for _, scenario := range scenarios {
		logger.Printf("Scenario: %s", scenario.description)
		// In practice, you would create a request and test routing
		logger.Printf("Headers: %+v", scenario.headers)
	}
}
