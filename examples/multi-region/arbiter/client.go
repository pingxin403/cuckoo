package arbiter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// ArbiterClient provides distributed coordination and split-brain prevention
// using Zookeeper as the consensus mechanism for multi-region active-active setup
type ArbiterClient struct {
	zkConn   *zk.Conn
	regionID string
	logger   *log.Logger

	// Distributed lock for leader election
	electionPath string
	lockPath     string

	// Health check state
	mu            sync.RWMutex
	healthStatus  map[string]bool // service -> healthy
	lastElection  time.Time
	currentLeader string

	// Configuration
	sessionTimeout time.Duration
	electionTTL    time.Duration
}

// Config holds configuration for the arbiter client
type Config struct {
	ZookeeperHosts []string
	RegionID       string
	SessionTimeout time.Duration
	ElectionTTL    time.Duration
	Logger         *log.Logger
}

// ElectionResult represents the result of a leader election
type ElectionResult struct {
	Leader    string    `json:"leader"`
	IsPrimary bool      `json:"is_primary"`
	Timestamp time.Time `json:"timestamp"`
	TTL       int       `json:"ttl_seconds"`
	Reason    string    `json:"reason"`
}

// HealthReport represents health status of a region
type HealthReport struct {
	RegionID  string          `json:"region_id"`
	Services  map[string]bool `json:"services"`
	Timestamp time.Time       `json:"timestamp"`
	IsHealthy bool            `json:"is_healthy"`
}

// NewArbiterClient creates a new arbiter client connected to Zookeeper
func NewArbiterClient(config Config) (*ArbiterClient, error) {
	if config.Logger == nil {
		config.Logger = log.New(log.Writer(), "[ARBITER] ", log.LstdFlags)
	}

	if config.SessionTimeout == 0 {
		config.SessionTimeout = 10 * time.Second
	}

	if config.ElectionTTL == 0 {
		config.ElectionTTL = 30 * time.Second
	}

	// Connect to Zookeeper
	conn, _, err := zk.Connect(config.ZookeeperHosts, config.SessionTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	client := &ArbiterClient{
		zkConn:         conn,
		regionID:       config.RegionID,
		logger:         config.Logger,
		electionPath:   "/im/election",
		lockPath:       "/im/locks",
		healthStatus:   make(map[string]bool),
		sessionTimeout: config.SessionTimeout,
		electionTTL:    config.ElectionTTL,
	}

	// Initialize Zookeeper paths
	if err := client.initializePaths(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize zookeeper paths: %w", err)
	}

	client.logger.Printf("Arbiter client initialized for region %s", config.RegionID)
	return client, nil
}

// initializePaths creates necessary Zookeeper paths if they don't exist
func (a *ArbiterClient) initializePaths() error {
	paths := []string{
		"/im",
		a.electionPath,
		a.lockPath,
		fmt.Sprintf("%s/regions", a.electionPath),
		fmt.Sprintf("%s/health", a.electionPath),
	}

	for _, path := range paths {
		exists, _, err := a.zkConn.Exists(path)
		if err != nil {
			return fmt.Errorf("failed to check path %s: %w", path, err)
		}

		if !exists {
			_, err := a.zkConn.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
			if err != nil && err != zk.ErrNodeExists {
				return fmt.Errorf("failed to create path %s: %w", path, err)
			}
		}
	}

	return nil
}

// ElectPrimary performs distributed leader election using Zookeeper
// Returns the elected leader and whether this region is the primary
func (a *ArbiterClient) ElectPrimary(ctx context.Context, healthStatus map[string]bool) (*ElectionResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update health status
	a.healthStatus = make(map[string]bool)
	for service, healthy := range healthStatus {
		a.healthStatus[service] = healthy
	}

	// Report health to Zookeeper
	if err := a.reportHealthToZK(); err != nil {
		a.logger.Printf("Failed to report health to ZK: %v", err)
		// Continue with election even if health reporting fails
	}

	// Perform leader election using distributed lock
	lockPath := fmt.Sprintf("%s/primary_election", a.lockPath)
	lock := zk.NewLock(a.zkConn, lockPath, zk.WorldACL(zk.PermAll))

	// Try to acquire lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	lockChan := make(chan error, 1)
	go func() {
		lockChan <- lock.Lock()
	}()

	select {
	case err := <-lockChan:
		if err != nil {
			return nil, fmt.Errorf("failed to acquire election lock: %w", err)
		}
		defer lock.Unlock()
	case <-lockCtx.Done():
		return nil, fmt.Errorf("election lock timeout: %w", lockCtx.Err())
	}

	// Now we have the lock, determine the primary based on health
	leader, reason, err := a.determinePrimaryRegion()
	if err != nil {
		return nil, fmt.Errorf("failed to determine primary: %w", err)
	}

	// Record election result
	electionData := map[string]interface{}{
		"leader":    leader,
		"timestamp": time.Now(),
		"reason":    reason,
		"elector":   a.regionID,
	}

	electionBytes, _ := json.Marshal(electionData)
	electionResultPath := fmt.Sprintf("%s/current", a.electionPath)

	// Create or update election result
	exists, stat, err := a.zkConn.Exists(electionResultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check election result path: %w", err)
	}

	if exists {
		_, err = a.zkConn.Set(electionResultPath, electionBytes, stat.Version)
	} else {
		_, err = a.zkConn.Create(electionResultPath, electionBytes, 0, zk.WorldACL(zk.PermAll))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to record election result: %w", err)
	}

	a.currentLeader = leader
	a.lastElection = time.Now()

	result := &ElectionResult{
		Leader:    leader,
		IsPrimary: leader == a.regionID,
		Timestamp: time.Now(),
		TTL:       int(a.electionTTL.Seconds()),
		Reason:    reason,
	}

	a.logger.Printf("Election completed: leader=%s, is_primary=%v, reason=%s",
		leader, result.IsPrimary, reason)

	return result, nil
}

// determinePrimaryRegion implements the election logic based on health status
func (a *ArbiterClient) determinePrimaryRegion() (string, string, error) {
	// Get health reports from all regions
	healthPath := fmt.Sprintf("%s/health", a.electionPath)
	children, _, err := a.zkConn.Children(healthPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get health reports: %w", err)
	}

	healthyRegions := make([]string, 0)
	regionHealth := make(map[string]HealthReport)

	// Parse health reports from all regions
	for _, child := range children {
		childPath := fmt.Sprintf("%s/%s", healthPath, child)
		data, _, err := a.zkConn.Get(childPath)
		if err != nil {
			a.logger.Printf("Failed to get health data for %s: %v", child, err)
			continue
		}

		var report HealthReport
		if err := json.Unmarshal(data, &report); err != nil {
			a.logger.Printf("Failed to parse health data for %s: %v", child, err)
			continue
		}

		regionHealth[report.RegionID] = report

		// Check if region is healthy and recent
		if report.IsHealthy && time.Since(report.Timestamp) < 60*time.Second {
			healthyRegions = append(healthyRegions, report.RegionID)
		}
	}

	// Election rules (as per design document):
	// 1. If current primary is healthy, keep it
	// 2. If no healthy regions, return empty (read-only mode)
	// 3. If multiple healthy regions, prefer region-a (deterministic)
	// 4. If only one healthy region, elect it

	if len(healthyRegions) == 0 {
		return "", "no_healthy_regions", nil
	}

	// Check if current leader is still healthy
	if a.currentLeader != "" {
		for _, regionID := range healthyRegions {
			if regionID == a.currentLeader {
				return a.currentLeader, "current_leader_healthy", nil
			}
		}
	}

	// Deterministic election: prefer region-a, then region-b
	preferredOrder := []string{"region-a", "region-b"}
	for _, preferred := range preferredOrder {
		for _, regionID := range healthyRegions {
			if regionID == preferred {
				return regionID, "deterministic_election", nil
			}
		}
	}

	// Fallback: return first healthy region
	return healthyRegions[0], "fallback_election", nil
}

// reportHealthToZK reports this region's health status to Zookeeper
func (a *ArbiterClient) reportHealthToZK() error {
	report := HealthReport{
		RegionID:  a.regionID,
		Services:  a.healthStatus,
		Timestamp: time.Now(),
		IsHealthy: a.isRegionHealthy(),
	}

	reportBytes, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal health report: %w", err)
	}

	healthPath := fmt.Sprintf("%s/health/%s", a.electionPath, a.regionID)

	// Create or update health report
	exists, stat, err := a.zkConn.Exists(healthPath)
	if err != nil {
		return fmt.Errorf("failed to check health path: %w", err)
	}

	if exists {
		_, err = a.zkConn.Set(healthPath, reportBytes, stat.Version)
	} else {
		_, err = a.zkConn.Create(healthPath, reportBytes, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}

	return err
}

// isRegionHealthy determines if this region is healthy based on critical services
func (a *ArbiterClient) isRegionHealthy() bool {
	criticalServices := []string{"im-service", "redis", "database"}

	for _, service := range criticalServices {
		if healthy, exists := a.healthStatus[service]; !exists || !healthy {
			return false
		}
	}

	return true
}

// GetCurrentLeader returns the current leader from Zookeeper
func (a *ArbiterClient) GetCurrentLeader(ctx context.Context) (string, error) {
	electionResultPath := fmt.Sprintf("%s/current", a.electionPath)

	data, _, err := a.zkConn.Get(electionResultPath)
	if err != nil {
		if err == zk.ErrNoNode {
			return "", nil // No leader elected yet
		}
		return "", fmt.Errorf("failed to get current leader: %w", err)
	}

	var electionData map[string]interface{}
	if err := json.Unmarshal(data, &electionData); err != nil {
		return "", fmt.Errorf("failed to parse election data: %w", err)
	}

	leader, ok := electionData["leader"].(string)
	if !ok {
		return "", fmt.Errorf("invalid leader data in election result")
	}

	return leader, nil
}

// WatchLeaderChanges watches for leader changes and calls the callback
func (a *ArbiterClient) WatchLeaderChanges(ctx context.Context, callback func(leader string)) error {
	electionResultPath := fmt.Sprintf("%s/current", a.electionPath)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Watch for changes to the election result
		_, _, eventCh, err := a.zkConn.GetW(electionResultPath)
		if err != nil {
			if err == zk.ErrNoNode {
				// Node doesn't exist yet, wait and retry
				time.Sleep(1 * time.Second)
				continue
			}
			return fmt.Errorf("failed to watch election result: %w", err)
		}

		// Get current leader
		leader, err := a.GetCurrentLeader(ctx)
		if err != nil {
			a.logger.Printf("Failed to get current leader: %v", err)
		} else {
			callback(leader)
		}

		// Wait for change event
		select {
		case event := <-eventCh:
			a.logger.Printf("Leader change event: %v", event)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ReportHealth updates the health status for this region
func (a *ArbiterClient) ReportHealth(services map[string]bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.healthStatus = make(map[string]bool)
	for service, healthy := range services {
		a.healthStatus[service] = healthy
	}

	return a.reportHealthToZK()
}

// IsHealthy returns whether this region is currently healthy
func (a *ArbiterClient) IsHealthy() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.isRegionHealthy()
}

// GetHealthStatus returns the current health status of all services
func (a *ArbiterClient) GetHealthStatus() map[string]bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := make(map[string]bool)
	for service, healthy := range a.healthStatus {
		status[service] = healthy
	}

	return status
}

// Close closes the connection to Zookeeper
func (a *ArbiterClient) Close() error {
	if a.zkConn != nil {
		a.zkConn.Close()
	}
	return nil
}

// GetElectionHistory returns recent election events (for debugging/monitoring)
func (a *ArbiterClient) GetElectionHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// This would typically be implemented with a separate ZK path for history
	// For now, return current election info
	leader, err := a.GetCurrentLeader(ctx)
	if err != nil {
		return nil, err
	}

	if leader == "" {
		return []map[string]interface{}{}, nil
	}

	history := []map[string]interface{}{
		{
			"leader":    leader,
			"timestamp": a.lastElection,
			"reason":    "current_election",
		},
	}

	return history, nil
}
