package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/pingxin403/cuckoo/apps/im-service/traffic"
)

var (
	redisAddr     string
	redisPassword string
	redisDB       int
	dryRun        bool
	operator      string
	reason        string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "traffic-cli",
		Short: "CLI tool for managing multi-region traffic switching",
		Long: `Traffic CLI is a command-line tool for managing traffic distribution 
between regions in a multi-region active-active architecture.

It supports:
- Proportional traffic distribution (e.g., 90:10)
- Full traffic switch to a single region
- Dry-run mode for testing
- Event logging and history`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis server address")
	rootCmd.PersistentFlags().StringVar(&redisPassword, "redis-password", "", "Redis password")
	rootCmd.PersistentFlags().IntVar(&redisDB, "redis-db", 0, "Redis database number")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without applying changes")
	rootCmd.PersistentFlags().StringVar(&operator, "operator", getDefaultOperator(), "Operator performing the switch")
	rootCmd.PersistentFlags().StringVar(&reason, "reason", "", "Reason for the traffic switch")

	// Add subcommands
	rootCmd.AddCommand(newSwitchCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newEventsCmd())
	rootCmd.AddCommand(newRouteCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch traffic between regions",
		Long:  "Switch traffic distribution between regions using proportional or full switch modes",
	}

	cmd.AddCommand(newSwitchProportionalCmd())
	cmd.AddCommand(newSwitchFullCmd())

	return cmd
}

func newSwitchProportionalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proportional [region-a:weight] [region-b:weight]",
		Short: "Switch traffic with specified proportions",
		Long: `Switch traffic with specified proportions between regions.

Examples:
  # Switch 90% to region-a, 10% to region-b
  traffic-cli switch proportional region-a:90 region-b:10

  # Switch 50% to each region
  traffic-cli switch proportional region-a:50 region-b:50

  # Dry run mode
  traffic-cli switch proportional region-a:80 region-b:20 --dry-run

  # With reason and operator
  traffic-cli switch proportional region-a:100 region-b:0 \
    --reason "Maintenance on region-b" \
    --operator "ops-team"`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSwitchProportional,
	}

	return cmd
}

func newSwitchFullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "full [target-region]",
		Short: "Switch all traffic to a single region",
		Long: `Switch 100% of traffic to the specified target region.

Examples:
  # Switch all traffic to region-a
  traffic-cli switch full region-a

  # Switch all traffic to region-b with reason
  traffic-cli switch full region-b --reason "Failover due to region-a outage"

  # Dry run mode
  traffic-cli switch full region-a --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: runSwitchFull,
	}

	return cmd
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current traffic configuration",
		Long:  "Display the current traffic distribution configuration across regions",
		RunE:  runStatus,
	}

	return cmd
}

func newEventsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show traffic switching event history",
		Long:  "Display recent traffic switching events with details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvents(limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of recent events to show")

	return cmd
}

func newRouteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [user-id]",
		Short: "Test routing for a specific user",
		Long:  "Determine which region a specific user would be routed to based on current configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  runRoute,
	}

	return cmd
}

func runSwitchProportional(cmd *cobra.Command, args []string) error {
	if reason == "" {
		return fmt.Errorf("--reason flag is required for traffic switching")
	}

	// Parse region weights from arguments
	regionWeights, err := parseRegionWeights(args)
	if err != nil {
		return fmt.Errorf("failed to parse region weights: %w", err)
	}

	// Create traffic switcher
	switcher, err := createTrafficSwitcher()
	if err != nil {
		return err
	}

	// Perform the switch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := switcher.SwitchTrafficProportional(ctx, regionWeights, reason, operator, dryRun)
	if err != nil {
		return fmt.Errorf("traffic switch failed: %w", err)
	}

	// Display results
	printSwitchResponse(response)

	return nil
}

func runSwitchFull(cmd *cobra.Command, args []string) error {
	if reason == "" {
		return fmt.Errorf("--reason flag is required for traffic switching")
	}

	targetRegion := args[0]

	// Create traffic switcher
	switcher, err := createTrafficSwitcher()
	if err != nil {
		return err
	}

	// Perform the switch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := switcher.SwitchTrafficFull(ctx, targetRegion, reason, operator, dryRun)
	if err != nil {
		return fmt.Errorf("traffic switch failed: %w", err)
	}

	// Display results
	printSwitchResponse(response)

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	switcher, err := createTrafficSwitcher()
	if err != nil {
		return err
	}

	config := switcher.GetCurrentConfig()

	fmt.Println("Current Traffic Configuration")
	fmt.Println("=============================")
	fmt.Printf("Version:        %d\n", config.Version)
	fmt.Printf("Last Updated:   %s\n", config.LastUpdated.Format(time.RFC3339))
	fmt.Printf("Updated By:     %s\n", config.UpdatedBy)
	fmt.Printf("Default Region: %s\n", config.DefaultRegion)
	fmt.Println("\nRegion Weights:")

	for region, weight := range config.RegionWeights {
		bar := strings.Repeat("█", weight/2)
		fmt.Printf("  %-10s %3d%% %s\n", region, weight, bar)
	}

	return nil
}

func runEvents(limit int) error {
	switcher, err := createTrafficSwitcher()
	if err != nil {
		return err
	}

	events := switcher.GetTrafficEvents(limit)

	if len(events) == 0 {
		fmt.Println("No traffic switching events found")
		return nil
	}

	fmt.Printf("Recent Traffic Switching Events (showing %d)\n", len(events))
	fmt.Println("===========================================")

	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		fmt.Printf("\nEvent ID:   %s\n", event.ID)
		fmt.Printf("Type:       %s\n", event.Type)
		fmt.Printf("Status:     %s\n", event.Status)
		fmt.Printf("Timestamp:  %s\n", event.Timestamp.Format(time.RFC3339))
		fmt.Printf("Operator:   %s\n", event.Operator)
		fmt.Printf("Reason:     %s\n", event.Reason)

		if event.Duration > 0 {
			fmt.Printf("Duration:   %s\n", event.Duration)
		}

		if event.FromConfig != nil && event.ToConfig != nil {
			fmt.Println("Changes:")
			for region := range event.ToConfig.RegionWeights {
				oldWeight := event.FromConfig.RegionWeights[region]
				newWeight := event.ToConfig.RegionWeights[region]
				if oldWeight != newWeight {
					fmt.Printf("  %s: %d%% → %d%%\n", region, oldWeight, newWeight)
				}
			}
		}

		if len(event.Metadata) > 0 {
			fmt.Println("Metadata:")
			for key, value := range event.Metadata {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		fmt.Println(strings.Repeat("-", 50))
	}

	return nil
}

func runRoute(cmd *cobra.Command, args []string) error {
	userID := args[0]

	switcher, err := createTrafficSwitcher()
	if err != nil {
		return err
	}

	targetRegion := switcher.RouteRequest(userID)
	config := switcher.GetCurrentConfig()

	fmt.Printf("Routing Information for User: %s\n", userID)
	fmt.Println("=====================================")
	fmt.Printf("Target Region: %s\n", targetRegion)
	fmt.Println("\nCurrent Configuration:")

	for region, weight := range config.RegionWeights {
		marker := " "
		if region == targetRegion {
			marker = "→"
		}
		fmt.Printf("%s %-10s %3d%%\n", marker, region, weight)
	}

	return nil
}

// Helper functions

func createTrafficSwitcher() (*traffic.TrafficSwitcher, error) {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", redisAddr, err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[TrafficCLI] ", log.LstdFlags)

	// Create traffic switcher
	switcher := traffic.NewTrafficSwitcher(redisClient, logger)

	return switcher, nil
}

func parseRegionWeights(args []string) (map[string]int, error) {
	weights := make(map[string]int)

	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format '%s', expected 'region:weight'", arg)
		}

		region := strings.TrimSpace(parts[0])
		weightStr := strings.TrimSpace(parts[1])

		weight, err := strconv.Atoi(weightStr)
		if err != nil {
			return nil, fmt.Errorf("invalid weight '%s' for region '%s': %w", weightStr, region, err)
		}

		if weight < 0 || weight > 100 {
			return nil, fmt.Errorf("weight for region '%s' must be between 0 and 100, got %d", region, weight)
		}

		weights[region] = weight
	}

	// Validate total weight
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	if totalWeight != 100 {
		return nil, fmt.Errorf("total weight must equal 100, got %d", totalWeight)
	}

	return weights, nil
}

func printSwitchResponse(response *traffic.TrafficSwitchResponse) {
	if dryRun {
		fmt.Println("DRY RUN MODE - No changes applied")
		fmt.Println("=================================")
	} else {
		fmt.Println("Traffic Switch Result")
		fmt.Println("====================")
	}

	if response.Success {
		fmt.Println("✓ Success")
	} else {
		fmt.Println("✗ Failed")
	}

	fmt.Printf("\nEvent ID: %s\n", response.EventID)
	fmt.Printf("Message:  %s\n", response.Message)

	if response.OldConfig != nil {
		fmt.Println("\nOld Configuration:")
		for region, weight := range response.OldConfig.RegionWeights {
			fmt.Printf("  %-10s %3d%%\n", region, weight)
		}
	}

	if response.NewConfig != nil {
		fmt.Println("\nNew Configuration:")
		for region, weight := range response.NewConfig.RegionWeights {
			bar := strings.Repeat("█", weight/2)
			fmt.Printf("  %-10s %3d%% %s\n", region, weight, bar)
		}
	}

	if response.EstimatedDuration > 0 {
		fmt.Printf("\nEstimated Duration: %s\n", response.EstimatedDuration)
	}

	if !dryRun && response.Success {
		fmt.Println("\n✓ Traffic configuration has been updated successfully")
		fmt.Println("  New connections will be routed according to the new weights")
		fmt.Println("  Existing connections will remain on their current regions")
	}
}

func getDefaultOperator() string {
	// Try to get username from environment
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}
