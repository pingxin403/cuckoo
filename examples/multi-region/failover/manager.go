package failover

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/multi-region/arbiter"
	"github.com/cuckoo-org/cuckoo/examples/multi-region/health"
)

// FailoverManager manages automatic failover between regions
// Integrates with health checker and arbiter to provide RTO < 30 seconds
type FailoverManager struct {
	regionID      string
	healthChecker *health.HealthChecker
	arbiterClient *arbiter.ArbiterClient
	config        Config
	logger        *log.Logger

	// State management
	mu                  sync.RWMutex
	currentState        FailoverState
	isPrimary           bool
	lastFailoverTime    time.Time
	failoverHistory     []FailoverEvent
	consecutiveFailures int

	// Control channels
	stopCh    chan struct{}
	triggerCh chan FailoverTrigger
	wg        sync.WaitGroup

	// Callbacks for external integration
	onFailoverStart    func(event FailoverEvent) error
	onFailoverComplete func(event FailoverEvent) error
	onTrafficSwitch    func(from, to string) error
}

// FailoverState represents the current state of the failover system
type FailoverState string

const (
	StateActive   FailoverState = "active"    // Region is active and serving traffic
	StateStandby  FailoverState = "standby"   // Region is standby, ready for failover
	StateFailover FailoverState = "failover"  // Failover in progress
	StateDegraded FailoverState = "degraded"  // Region is degraded but still serving
	StateReadOnly FailoverState = "read_only" // Region is in read-only mode (split-brain protection)
)

// FailoverTrigger represents a trigger for failover
type FailoverTrigger struct {
	Type      TriggerType            `json:"type"`
	Reason    string                 `json:"reason"`
	Severity  Severity               `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Details   map[string]interface{} `json:"details"`
}

// TriggerType defines the type of failover trigger
type TriggerType string

const (
	TriggerHealthCheck      TriggerType = "health_check"      // Health check failure
	TriggerManual           TriggerType = "manual"            // Manual failover
	TriggerArbiter          TriggerType = "arbiter"           // Arbiter-initiated failover
	TriggerNetworkPartition TriggerType = "network_partition" // Network partition detected
	TriggerServiceFailure   TriggerType = "service_failure"   // Critical service failure
)

// Severity defines the severity of a failover trigger
type Severity string

const (
	SeverityLow      Severity = "low"      // Minor issues, no immediate action
	SeverityMedium   Severity = "medium"   // Degraded performance, monitor closely
	SeverityHigh     Severity = "high"     // Significant issues, prepare for failover
	SeverityCritical Severity = "critical" // Critical failure, immediate failover required
)

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	FromRegion string                 `json:"from_region"`
	ToRegion   string                 `json:"to_region"`
	Trigger    FailoverTrigger        `json:"trigger"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Status     EventStatus            `json:"status"`
	Steps      []FailoverStep         `json:"steps"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// EventStatus represents the status of a failover event
type EventStatus string

const (
	StatusInProgress EventStatus = "in_progress"
	StatusCompleted  EventStatus = "completed"
	StatusFailed     EventStatus = "failed"
	StatusAborted    EventStatus = "aborted"
)

// FailoverStep represents a step in the failover process
type FailoverStep struct {
	Name        string        `json:"name"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Status      EventStatus   `json:"status"`
	Error       string        `json:"error,omitempty"`
	Description string        `json:"description"`
}

// Config holds configuration for the failover manager
type Config struct {
	RegionID                   string        `json:"region_id"`
	HealthCheckInterval        time.Duration `json:"health_check_interval"`
	FailureThreshold           int           `json:"failure_threshold"`    // Consecutive failures before failover
	FailoverTimeout            time.Duration `json:"failover_timeout"`     // Max time for failover completion
	CooldownPeriod             time.Duration `json:"cooldown_period"`      // Min time between failovers
	EnableAutoFailover         bool          `json:"enable_auto_failover"` // Enable automatic failover
	EnableSplitBrainProtection bool          `json:"enable_split_brain_protection"`
	MaxFailoverHistory         int           `json:"max_failover_history"` // Max events to keep in history

	// RTO/RPO targets
	RTOTarget time.Duration `json:"rto_target"` // Recovery Time Objective (< 30s)
	RPOTarget time.Duration `json:"rpo_target"` // Recovery Point Objective (< 1s for messages)
}

// DefaultConfig returns default configuration for failover manager
func DefaultConfig(regionID string) Config {
	return Config{
		RegionID:                   regionID,
		HealthCheckInterval:        5 * time.Second,
		FailureThreshold:           3, // 3 consecutive failures = 15 seconds
		FailoverTimeout:            30 * time.Second,
		CooldownPeriod:             60 * time.Second,
		EnableAutoFailover:         true,
		EnableSplitBrainProtection: true,
		MaxFailoverHistory:         100,
		RTOTarget:                  30 * time.Second,
		RPOTarget:                  1 * time.Second,
	}
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(
	regionID string,
	healthChecker *health.HealthChecker,
	arbiterClient *arbiter.ArbiterClient,
	config Config,
	logger *log.Logger,
) *FailoverManager {
	if logger == nil {
		logger = log.New(log.Writer(), "[FailoverManager] ", log.LstdFlags)
	}

	return &FailoverManager{
		regionID:        regionID,
		healthChecker:   healthChecker,
		arbiterClient:   arbiterClient,
		config:          config,
		logger:          logger,
		currentState:    StateStandby,
		isPrimary:       false,
		failoverHistory: make([]FailoverEvent, 0, config.MaxFailoverHistory),
		stopCh:          make(chan struct{}),
		triggerCh:       make(chan FailoverTrigger, 10),
	}
}

// Start starts the failover manager
func (fm *FailoverManager) Start(ctx context.Context) error {
	fm.logger.Printf("Starting failover manager for region %s", fm.regionID)

	// Start monitoring goroutines
	fm.wg.Add(3)
	go fm.runHealthMonitoring(ctx)
	go fm.runFailoverProcessor(ctx)
	go fm.runArbiterWatcher(ctx)

	// Perform initial leader election
	if err := fm.performInitialElection(ctx); err != nil {
		fm.logger.Printf("Initial election failed: %v", err)
		// Continue running even if initial election fails
	}

	fm.logger.Printf("Failover manager started successfully")
	return nil
}

// Stop stops the failover manager
func (fm *FailoverManager) Stop() error {
	fm.logger.Printf("Stopping failover manager")
	close(fm.stopCh)
	fm.wg.Wait()
	return nil
}

// performInitialElection performs initial leader election
func (fm *FailoverManager) performInitialElection(ctx context.Context) error {
	// Get current health status
	systemHealth := fm.healthChecker.GetSystemHealth()
	healthStatus := make(map[string]bool)

	for name, component := range systemHealth.Components {
		healthStatus[name] = component.Status == health.StatusHealthy
	}

	// Perform election
	result, err := fm.arbiterClient.ElectPrimary(ctx, healthStatus)
	if err != nil {
		return fmt.Errorf("election failed: %w", err)
	}

	// Update state based on election result
	fm.mu.Lock()
	fm.isPrimary = result.IsPrimary
	if fm.isPrimary {
		fm.currentState = StateActive
	} else {
		fm.currentState = StateStandby
	}
	fm.mu.Unlock()

	fm.logger.Printf("Initial election completed: isPrimary=%v, leader=%s, reason=%s",
		result.IsPrimary, result.Leader, result.Reason)

	return nil
}

// runHealthMonitoring monitors health and triggers failover when needed
func (fm *FailoverManager) runHealthMonitoring(ctx context.Context) {
	defer fm.wg.Done()

	ticker := time.NewTicker(fm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.checkHealthAndTriggerFailover(ctx)
		}
	}
}

// checkHealthAndTriggerFailover checks system health and triggers failover if needed
func (fm *FailoverManager) checkHealthAndTriggerFailover(ctx context.Context) {
	systemHealth := fm.healthChecker.GetSystemHealth()

	// Analyze health status
	criticalFailures := 0
	failedServices := make([]string, 0)

	for name, component := range systemHealth.Components {
		if component.Status == health.StatusCritical {
			criticalFailures++
			failedServices = append(failedServices, name)
		}
	}

	// Determine if failover is needed
	if criticalFailures > 0 {
		fm.mu.Lock()
		fm.consecutiveFailures++
		fm.mu.Unlock()

		if fm.consecutiveFailures >= fm.config.FailureThreshold {
			// Trigger failover
			trigger := FailoverTrigger{
				Type:      TriggerHealthCheck,
				Reason:    fmt.Sprintf("Health check failures: %v", failedServices),
				Severity:  SeverityCritical,
				Timestamp: time.Now(),
				Source:    "health_monitor",
				Details: map[string]interface{}{
					"failed_services":      failedServices,
					"consecutive_failures": fm.consecutiveFailures,
					"health_score":         systemHealth.Score,
					"overall_status":       systemHealth.Status,
				},
			}

			select {
			case fm.triggerCh <- trigger:
				fm.logger.Printf("Failover triggered due to health failures: %v", failedServices)
			default:
				fm.logger.Printf("Failover trigger channel full, skipping trigger")
			}
		}
	} else {
		// Reset failure counter on successful health check
		fm.mu.Lock()
		fm.consecutiveFailures = 0
		fm.mu.Unlock()
	}
}

// runFailoverProcessor processes failover triggers
func (fm *FailoverManager) runFailoverProcessor(ctx context.Context) {
	defer fm.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case trigger := <-fm.triggerCh:
			if err := fm.processFailoverTrigger(ctx, trigger); err != nil {
				fm.logger.Printf("Failed to process failover trigger: %v", err)
			}
		}
	}
}

// processFailoverTrigger processes a single failover trigger
func (fm *FailoverManager) processFailoverTrigger(ctx context.Context, trigger FailoverTrigger) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Check if auto-failover is enabled
	if !fm.config.EnableAutoFailover && trigger.Type != TriggerManual {
		fm.logger.Printf("Auto-failover disabled, ignoring trigger: %s", trigger.Reason)
		return nil
	}

	// Check cooldown period
	if time.Since(fm.lastFailoverTime) < fm.config.CooldownPeriod {
		fm.logger.Printf("Failover in cooldown period, ignoring trigger: %s", trigger.Reason)
		return nil
	}

	// Check if we're already in failover
	if fm.currentState == StateFailover {
		fm.logger.Printf("Failover already in progress, ignoring trigger: %s", trigger.Reason)
		return nil
	}

	// Only primary regions can initiate failover
	if !fm.isPrimary {
		fm.logger.Printf("Not primary region, ignoring failover trigger: %s", trigger.Reason)
		return nil
	}

	// Start failover process
	return fm.executeFailover(ctx, trigger)
}

// executeFailover executes the failover process
func (fm *FailoverManager) executeFailover(ctx context.Context, trigger FailoverTrigger) error {
	eventID := fmt.Sprintf("failover-%d", time.Now().Unix())
	startTime := time.Now()

	event := FailoverEvent{
		ID:         eventID,
		Type:       "automatic_failover",
		FromRegion: fm.regionID,
		ToRegion:   "", // Will be determined during failover
		Trigger:    trigger,
		StartTime:  startTime,
		Status:     StatusInProgress,
		Steps:      make([]FailoverStep, 0),
		Metadata: map[string]interface{}{
			"rto_target": fm.config.RTOTarget.String(),
			"rpo_target": fm.config.RPOTarget.String(),
		},
	}

	fm.logger.Printf("Starting failover event %s: %s", eventID, trigger.Reason)

	// Update state
	fm.currentState = StateFailover
	fm.lastFailoverTime = startTime

	// Call external callback
	if fm.onFailoverStart != nil {
		if err := fm.onFailoverStart(event); err != nil {
			fm.logger.Printf("Failover start callback failed: %v", err)
		}
	}

	// Execute failover steps
	steps := []struct {
		name        string
		description string
		fn          func(context.Context, *FailoverEvent) error
	}{
		{"validate_trigger", "Validate failover trigger and conditions", fm.stepValidateTrigger},
		{"elect_new_primary", "Elect new primary region via arbiter", fm.stepElectNewPrimary},
		{"switch_traffic", "Switch traffic to new primary region", fm.stepSwitchTraffic},
		{"update_state", "Update local state and notify systems", fm.stepUpdateState},
		{"verify_failover", "Verify failover completion and health", fm.stepVerifyFailover},
	}

	for _, step := range steps {
		stepResult := fm.executeFailoverStep(ctx, step.name, step.description, step.fn, &event)
		event.Steps = append(event.Steps, stepResult)

		if stepResult.Status == StatusFailed {
			event.Status = StatusFailed
			event.Error = stepResult.Error
			break
		}
	}

	// Complete failover
	event.EndTime = time.Now()
	event.Duration = event.EndTime.Sub(event.StartTime)

	if event.Status != StatusFailed {
		event.Status = StatusCompleted
	}

	// Add to history
	fm.addToHistory(event)

	// Call external callback
	if fm.onFailoverComplete != nil {
		if err := fm.onFailoverComplete(event); err != nil {
			fm.logger.Printf("Failover complete callback failed: %v", err)
		}
	}

	fm.logger.Printf("Failover event %s completed: status=%s, duration=%v",
		eventID, event.Status, event.Duration)

	// Check RTO compliance
	if event.Duration > fm.config.RTOTarget {
		fm.logger.Printf("WARNING: Failover exceeded RTO target: %v > %v",
			event.Duration, fm.config.RTOTarget)
	}

	return nil
}

// executeFailoverStep executes a single failover step
func (fm *FailoverManager) executeFailoverStep(
	ctx context.Context,
	name, description string,
	fn func(context.Context, *FailoverEvent) error,
	event *FailoverEvent,
) FailoverStep {
	step := FailoverStep{
		Name:        name,
		Description: description,
		StartTime:   time.Now(),
		Status:      StatusInProgress,
	}

	fm.logger.Printf("Executing failover step: %s", name)

	// Execute step with timeout
	stepCtx, cancel := context.WithTimeout(ctx, fm.config.FailoverTimeout/5) // Each step gets 1/5 of total timeout
	defer cancel()

	err := fn(stepCtx, event)

	step.EndTime = time.Now()
	step.Duration = step.EndTime.Sub(step.StartTime)

	if err != nil {
		step.Status = StatusFailed
		step.Error = err.Error()
		fm.logger.Printf("Failover step %s failed: %v (took %v)", name, err, step.Duration)
	} else {
		step.Status = StatusCompleted
		fm.logger.Printf("Failover step %s completed (took %v)", name, step.Duration)
	}

	return step
}

// Failover step implementations

func (fm *FailoverManager) stepValidateTrigger(ctx context.Context, event *FailoverEvent) error {
	// Validate that failover is still needed
	systemHealth := fm.healthChecker.GetSystemHealth()

	if systemHealth.Status == health.StatusHealthy {
		return fmt.Errorf("system health recovered, failover no longer needed")
	}

	// Check if we're still the primary
	currentLeader, err := fm.arbiterClient.GetCurrentLeader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current leader: %w", err)
	}

	if currentLeader != fm.regionID {
		return fmt.Errorf("no longer primary region, cannot initiate failover")
	}

	return nil
}

func (fm *FailoverManager) stepElectNewPrimary(ctx context.Context, event *FailoverEvent) error {
	// Report our unhealthy status to arbiter
	healthStatus := make(map[string]bool)
	systemHealth := fm.healthChecker.GetSystemHealth()

	for name, component := range systemHealth.Components {
		healthStatus[name] = component.Status == health.StatusHealthy
	}

	// Force election with our degraded status
	result, err := fm.arbiterClient.ElectPrimary(ctx, healthStatus)
	if err != nil {
		return fmt.Errorf("failed to elect new primary: %w", err)
	}

	if result.Leader == fm.regionID {
		return fmt.Errorf("arbiter still elected us as primary, cannot failover")
	}

	event.ToRegion = result.Leader
	event.Metadata["election_reason"] = result.Reason

	fm.logger.Printf("New primary elected: %s (reason: %s)", result.Leader, result.Reason)
	return nil
}

func (fm *FailoverManager) stepSwitchTraffic(ctx context.Context, event *FailoverEvent) error {
	// Call external traffic switching callback
	if fm.onTrafficSwitch != nil {
		if err := fm.onTrafficSwitch(event.FromRegion, event.ToRegion); err != nil {
			return fmt.Errorf("traffic switch failed: %w", err)
		}
	}

	// In a real implementation, this would:
	// 1. Update DNS records
	// 2. Update load balancer configuration
	// 3. Notify API gateways
	// 4. Update service mesh routing

	fm.logger.Printf("Traffic switched from %s to %s", event.FromRegion, event.ToRegion)
	return nil
}

func (fm *FailoverManager) stepUpdateState(ctx context.Context, event *FailoverEvent) error {
	// Update our local state
	fm.isPrimary = false
	fm.currentState = StateStandby

	// Report updated health status
	healthStatus := make(map[string]bool)
	systemHealth := fm.healthChecker.GetSystemHealth()

	for name, component := range systemHealth.Components {
		healthStatus[name] = component.Status == health.StatusHealthy
	}

	if err := fm.arbiterClient.ReportHealth(healthStatus); err != nil {
		return fmt.Errorf("failed to report health status: %w", err)
	}

	return nil
}

func (fm *FailoverManager) stepVerifyFailover(ctx context.Context, event *FailoverEvent) error {
	// Verify new primary is healthy
	// In a real implementation, this would check:
	// 1. New primary is responding to health checks
	// 2. Traffic is flowing to new primary
	// 3. Data synchronization is working

	// For now, just verify the election result
	currentLeader, err := fm.arbiterClient.GetCurrentLeader(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify new leader: %w", err)
	}

	if currentLeader != event.ToRegion {
		return fmt.Errorf("leader verification failed: expected %s, got %s", event.ToRegion, currentLeader)
	}

	return nil
}

// runArbiterWatcher watches for arbiter events and leader changes
func (fm *FailoverManager) runArbiterWatcher(ctx context.Context) {
	defer fm.wg.Done()

	// Watch for leader changes
	err := fm.arbiterClient.WatchLeaderChanges(ctx, func(leader string) {
		fm.mu.Lock()
		defer fm.mu.Unlock()

		oldIsPrimary := fm.isPrimary
		fm.isPrimary = (leader == fm.regionID)

		if oldIsPrimary != fm.isPrimary {
			if fm.isPrimary {
				fm.currentState = StateActive
				fm.logger.Printf("Became primary region")
			} else {
				fm.currentState = StateStandby
				fm.logger.Printf("Became standby region, new primary: %s", leader)
			}
		}
	})

	if err != nil {
		fm.logger.Printf("Arbiter watcher error: %v", err)
	}
}

// addToHistory adds a failover event to history
func (fm *FailoverManager) addToHistory(event FailoverEvent) {
	if len(fm.failoverHistory) >= fm.config.MaxFailoverHistory {
		// Remove oldest event
		fm.failoverHistory = fm.failoverHistory[1:]
	}
	fm.failoverHistory = append(fm.failoverHistory, event)
}

// Public API methods

// TriggerManualFailover triggers a manual failover
func (fm *FailoverManager) TriggerManualFailover(reason string) error {
	trigger := FailoverTrigger{
		Type:      TriggerManual,
		Reason:    reason,
		Severity:  SeverityCritical,
		Timestamp: time.Now(),
		Source:    "manual",
		Details: map[string]interface{}{
			"manual": true,
		},
	}

	select {
	case fm.triggerCh <- trigger:
		fm.logger.Printf("Manual failover triggered: %s", reason)
		return nil
	default:
		return fmt.Errorf("failover trigger channel full")
	}
}

// GetCurrentState returns the current failover state
func (fm *FailoverManager) GetCurrentState() (FailoverState, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.currentState, fm.isPrimary
}

// GetFailoverHistory returns the failover event history
func (fm *FailoverManager) GetFailoverHistory() []FailoverEvent {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]FailoverEvent, len(fm.failoverHistory))
	copy(history, fm.failoverHistory)
	return history
}

// GetStatus returns the current status of the failover manager
func (fm *FailoverManager) GetStatus() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return map[string]interface{}{
		"region_id":             fm.regionID,
		"current_state":         fm.currentState,
		"is_primary":            fm.isPrimary,
		"last_failover_time":    fm.lastFailoverTime,
		"consecutive_failures":  fm.consecutiveFailures,
		"failover_count":        len(fm.failoverHistory),
		"auto_failover_enabled": fm.config.EnableAutoFailover,
		"rto_target":            fm.config.RTOTarget.String(),
		"rpo_target":            fm.config.RPOTarget.String(),
	}
}

// SetCallbacks sets external callbacks for failover events
func (fm *FailoverManager) SetCallbacks(
	onFailoverStart func(FailoverEvent) error,
	onFailoverComplete func(FailoverEvent) error,
	onTrafficSwitch func(from, to string) error,
) {
	fm.onFailoverStart = onFailoverStart
	fm.onFailoverComplete = onFailoverComplete
	fm.onTrafficSwitch = onTrafficSwitch
}

// EnableAutoFailover enables or disables automatic failover
func (fm *FailoverManager) EnableAutoFailover(enabled bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.config.EnableAutoFailover = enabled
	fm.logger.Printf("Auto-failover %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// GetMetrics returns failover metrics for monitoring
func (fm *FailoverManager) GetMetrics() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	totalFailovers := len(fm.failoverHistory)
	successfulFailovers := 0
	totalFailoverTime := time.Duration(0)
	rtoViolations := 0

	for _, event := range fm.failoverHistory {
		if event.Status == StatusCompleted {
			successfulFailovers++
			totalFailoverTime += event.Duration

			if event.Duration > fm.config.RTOTarget {
				rtoViolations++
			}
		}
	}

	avgFailoverTime := time.Duration(0)
	if successfulFailovers > 0 {
		avgFailoverTime = totalFailoverTime / time.Duration(successfulFailovers)
	}

	return map[string]interface{}{
		"total_failovers":      totalFailovers,
		"successful_failovers": successfulFailovers,
		"failed_failovers":     totalFailovers - successfulFailovers,
		"success_rate":         float64(successfulFailovers) / float64(max(totalFailovers, 1)),
		"avg_failover_time":    avgFailoverTime.String(),
		"rto_violations":       rtoViolations,
		"rto_compliance_rate":  float64(successfulFailovers-rtoViolations) / float64(max(successfulFailovers, 1)),
		"consecutive_failures": fm.consecutiveFailures,
		"last_failover_age":    time.Since(fm.lastFailoverTime).String(),
	}
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
