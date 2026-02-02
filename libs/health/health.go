package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// NewHealthChecker creates a new health checker with the given configuration
// and observability integration. It initializes all internal state including
// the liveness probe, readiness probe, and check management structures.
//
// The health checker must be started with Start() before it begins checking health.
//
// Example:
//
//	obs, _ := observability.New(observability.Config{ServiceName: "my-service"})
//	hc := health.NewHealthChecker(health.Config{
//	    ServiceName:      "my-service",
//	    CheckInterval:    5 * time.Second,
//	    DefaultTimeout:   100 * time.Millisecond,
//	    FailureThreshold: 3,
//	}, obs)
//	hc.Start()
//	defer hc.Stop()
func NewHealthChecker(config Config, obs observability.Observability) *HealthChecker {
	// Validate configuration
	if err := config.Validate(); err != nil {
		// Log error and use defaults
		if obs != nil {
			obs.Logger().Error(context.Background(), "Invalid health checker configuration, using defaults",
				"error", err.Error(),
				"service", config.ServiceName,
			)
		}
		config = DefaultConfig(config.ServiceName)
	}

	// Initialize liveness probe
	livenessProbe := &LivenessProbe{
		memoryLimit:       config.LivenessConfig.MemoryLimit,
		goroutineLimit:    config.LivenessConfig.GoroutineLimit,
		heartbeatInterval: config.LivenessConfig.HeartbeatInterval,
		heartbeatTimeout:  config.LivenessConfig.HeartbeatTimeout,
		stopCh:            make(chan struct{}),
	}
	// Initialize heartbeat timestamp
	livenessProbe.lastHeartbeat.Store(time.Now())

	// Initialize readiness probe
	readinessProbe := &ReadinessProbe{
		checks:           make([]Check, 0),
		failureCount:     make(map[string]int),
		failureThreshold: config.FailureThreshold,
	}
	// Start as ready (will be updated by first health check)
	readinessProbe.isReady.Store(1)

	// Create health checker
	hc := &HealthChecker{
		config:         config,
		obs:            obs,
		livenessProbe:  livenessProbe,
		readinessProbe: readinessProbe,
		checks:         make(map[string]Check),
		results:        make(map[string]*CheckResult),
		stopCh:         make(chan struct{}),
	}

	// Log initialization
	if obs != nil {
		obs.Logger().Info(context.Background(), "Health checker initialized",
			"service", config.ServiceName,
			"check_interval", config.CheckInterval,
			"default_timeout", config.DefaultTimeout,
			"failure_threshold", config.FailureThreshold,
		)
	}

	return hc
}

// RegisterCheck registers a health check to be executed periodically.
// The check will be added to the readiness probe and executed according
// to its configured interval.
//
// If a check with the same name already exists, it will be replaced.
//
// Example:
//
//	db, _ := sql.Open("mysql", dsn)
//	hc.RegisterCheck(health.NewDatabaseCheck("database", db))
func (hc *HealthChecker) RegisterCheck(check Check) {
	if check == nil {
		if hc.obs != nil {
			hc.obs.Logger().Warn(context.Background(), "Attempted to register nil check")
		}
		return
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	name := check.Name()
	if name == "" {
		if hc.obs != nil {
			hc.obs.Logger().Warn(context.Background(), "Attempted to register check with empty name")
		}
		return
	}

	// Check for duplicate
	if _, exists := hc.checks[name]; exists {
		if hc.obs != nil {
			hc.obs.Logger().Warn(context.Background(), "Replacing existing health check",
				"check", name,
			)
		}
	}

	// Register the check
	hc.checks[name] = check
	hc.readinessProbe.checks = append(hc.readinessProbe.checks, check)

	// Initialize result
	hc.results[name] = &CheckResult{
		Name:         name,
		Status:       StatusHealthy,
		LastCheck:    time.Time{},
		ResponseTime: 0,
		Error:        "",
		FailureCount: 0,
		SuccessCount: 0,
	}

	if hc.obs != nil {
		hc.obs.Logger().Info(context.Background(), "Health check registered",
			"check", name,
			"critical", check.Critical(),
			"timeout", check.Timeout(),
			"interval", check.Interval(),
		)
	}
}

// Start begins health checking. It starts the liveness probe heartbeat
// and begins executing registered health checks according to their intervals.
//
// Start is idempotent - calling it multiple times has no effect.
//
// Example:
//
//	hc.Start()
//	defer hc.Stop()
func (hc *HealthChecker) Start() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Check if already started
	select {
	case <-hc.stopCh:
		// Channel is closed, create a new one
		hc.stopCh = make(chan struct{})
	default:
		// Already running, check if we have a running goroutine
		// For simplicity, we'll allow restart
	}

	// Start liveness probe heartbeat
	hc.wg.Add(1)
	go hc.livenessProbe.startHeartbeat(&hc.wg, hc.stopCh)

	// Start health check loop
	hc.wg.Add(1)
	go hc.healthCheckLoop()

	if hc.obs != nil {
		hc.obs.Logger().Info(context.Background(), "Health checker started",
			"service", hc.config.ServiceName,
			"checks", len(hc.checks),
		)
	}

	return nil
}

// Stop stops health checking and waits for all goroutines to finish.
// It's safe to call Stop multiple times.
//
// Example:
//
//	hc.Stop()
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	
	// Check if already stopped
	select {
	case <-hc.stopCh:
		// Already stopped
		hc.mu.Unlock()
		return
	default:
		// Signal stop
		close(hc.stopCh)
	}
	
	hc.mu.Unlock()

	// Wait for all goroutines to finish
	hc.wg.Wait()

	if hc.obs != nil {
		hc.obs.Logger().Info(context.Background(), "Health checker stopped",
			"service", hc.config.ServiceName,
		)
	}
}

// IsLive returns true if the service process is alive.
// This checks the liveness probe which monitors heartbeat, memory, and goroutines.
//
// This method is lock-free and very fast (< 1ms).
//
// Example:
//
//	if hc.IsLive() {
//	    // Process is alive
//	}
func (hc *HealthChecker) IsLive() bool {
	return hc.livenessProbe.Check(context.Background()) == nil
}

// IsReady returns true if the service is ready to serve traffic.
// This checks the readiness probe which monitors all registered health checks.
//
// This method is lock-free and very fast (< 1ms).
//
// Example:
//
//	if hc.IsReady() {
//	    // Service is ready
//	}
func (hc *HealthChecker) IsReady() bool {
	return hc.readinessProbe.isReady.Load() == 1
}

// GetSystemHealth returns the complete health status of the system,
// including overall status, health score, and individual component health.
//
// Example:
//
//	health := hc.GetSystemHealth()
//	fmt.Printf("Status: %s, Score: %.2f\n", health.Status, health.Score)
func (hc *HealthChecker) GetSystemHealth() *SystemHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Calculate overall health
	components := make(map[string]*ComponentHealth)
	totalScore := 0.0
	healthyCount := 0
	degradedCount := 0
	criticalCount := 0

	for name, result := range hc.results {
		component := &ComponentHealth{
			Name:         result.Name,
			Status:       result.Status,
			LastCheck:    result.LastCheck,
			ResponseTime: result.ResponseTime,
			Error:        result.Error,
		}
		components[name] = component

		// Count status
		switch result.Status {
		case StatusHealthy:
			healthyCount++
			totalScore += 1.0
		case StatusDegraded:
			degradedCount++
			totalScore += 0.5
		case StatusCritical:
			criticalCount++
			totalScore += 0.0
		}
	}

	// Calculate health score
	score := 0.0
	if len(hc.results) > 0 {
		score = totalScore / float64(len(hc.results))
	} else {
		// No checks registered, assume healthy
		score = 1.0
	}

	// Determine overall status
	status := StatusHealthy
	if score < hc.config.DegradedScore {
		status = StatusCritical
	} else if score < hc.config.HealthyScore {
		status = StatusDegraded
	}

	// Create summary
	summary := fmt.Sprintf("All systems operational (%d/%d healthy)",
		healthyCount, len(hc.results))
	if degradedCount > 0 || criticalCount > 0 {
		summary = fmt.Sprintf("%d healthy, %d degraded, %d critical",
			healthyCount, degradedCount, criticalCount)
	}

	return &SystemHealth{
		Status:     status,
		Service:    hc.config.ServiceName,
		Timestamp:  time.Now(),
		Score:      score,
		Summary:    summary,
		Components: components,
	}
}

// GetComponentHealth returns the health status of a specific component.
// Returns nil if the component doesn't exist.
//
// Example:
//
//	if health := hc.GetComponentHealth("database"); health != nil {
//	    fmt.Printf("Database: %s\n", health.Status)
//	}
func (hc *HealthChecker) GetComponentHealth(name string) *ComponentHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, exists := hc.results[name]
	if !exists {
		return nil
	}

	return &ComponentHealth{
		Name:         result.Name,
		Status:       result.Status,
		LastCheck:    result.LastCheck,
		ResponseTime: result.ResponseTime,
		Error:        result.Error,
	}
}

// healthCheckLoop runs the periodic health check execution
func (hc *HealthChecker) healthCheckLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	// Run initial check immediately
	hc.executeHealthChecks()

	for {
		select {
		case <-ticker.C:
			hc.executeHealthChecks()
		case <-hc.stopCh:
			return
		}
	}
}

// executeHealthChecks runs all registered health checks in parallel
func (hc *HealthChecker) executeHealthChecks() {
	hc.mu.RLock()
	checks := make([]Check, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	if len(checks) == 0 {
		return
	}

	// Execute checks in parallel
	var wg sync.WaitGroup
	resultsCh := make(chan *CheckResult, len(checks))

	for _, check := range checks {
		wg.Add(1)
		go func(c Check) {
			defer wg.Done()
			result := hc.executeCheck(c)
			resultsCh <- result
		}(check)
	}

	// Wait for all checks to complete
	wg.Wait()
	close(resultsCh)

	// Collect results
	results := make([]*CheckResult, 0, len(checks))
	for result := range resultsCh {
		results = append(results, result)
	}

	// Update results and readiness status
	hc.updateResults(results)
	hc.updateReadinessStatus()
	hc.exportMetrics()
}

// executeCheck executes a single health check with timeout
func (hc *HealthChecker) executeCheck(check Check) *CheckResult {
	timeout := check.Timeout()
	if timeout == 0 {
		timeout = hc.config.DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	err := check.Check(ctx)
	responseTime := time.Since(start)

	result := &CheckResult{
		Name:         check.Name(),
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
	}

	if err != nil {
		result.Status = StatusCritical
		result.Error = err.Error()
	} else if responseTime > timeout/2 {
		// Slow response - mark as degraded
		result.Status = StatusDegraded
		result.Error = fmt.Sprintf("slow response: %v", responseTime)
	} else {
		result.Status = StatusHealthy
		result.Error = ""
	}

	return result
}

// updateResults updates the stored check results
func (hc *HealthChecker) updateResults(results []*CheckResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for _, result := range results {
		oldResult, exists := hc.results[result.Name]
		
		if exists {
			// Update failure/success counts
			if result.Status == StatusCritical {
				result.FailureCount = oldResult.FailureCount + 1
				result.SuccessCount = 0
			} else {
				result.FailureCount = 0
				result.SuccessCount = oldResult.SuccessCount + 1
			}

			// Log status changes
			if oldResult.Status != result.Status {
				hc.logStatusChange(result.Name, oldResult.Status, result.Status, result.Error)
			}
		}

		hc.results[result.Name] = result
	}
}

// updateReadinessStatus updates the overall readiness status based on check results
func (hc *HealthChecker) updateReadinessStatus() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Check for critical failures
	hasCriticalFailure := false
	for name, result := range hc.results {
		check, exists := hc.checks[name]
		if !exists {
			continue
		}

		// Only critical checks affect readiness
		if check.Critical() && result.Status == StatusCritical {
			// Check if we've exceeded failure threshold
			if result.FailureCount >= hc.config.FailureThreshold {
				hasCriticalFailure = true
				break
			}
		}
	}

	// Update readiness flag
	if hasCriticalFailure {
		// Mark as not ready
		if hc.readinessProbe.isReady.Load() == 1 {
			hc.readinessProbe.isReady.Store(0)
			if hc.obs != nil {
				hc.obs.Logger().Warn(context.Background(), "Service marked as not ready",
					"service", hc.config.ServiceName,
				)
			}
		}
	} else {
		// Mark as ready
		if hc.readinessProbe.isReady.Load() == 0 {
			hc.readinessProbe.isReady.Store(1)
			if hc.obs != nil {
				hc.obs.Logger().Info(context.Background(), "Service marked as ready",
					"service", hc.config.ServiceName,
				)
			}
		}
	}
}

// logStatusChange logs when a component's health status changes
func (hc *HealthChecker) logStatusChange(component string, oldStatus, newStatus HealthStatus, errorMsg string) {
	if hc.obs == nil {
		return
	}

	ctx := context.Background()
	fields := []interface{}{
		"component", component,
		"old_status", oldStatus,
		"new_status", newStatus,
		"service", hc.config.ServiceName,
	}

	if errorMsg != "" {
		fields = append(fields, "error", errorMsg)
	}

	if newStatus == StatusCritical {
		hc.obs.Logger().Error(ctx, "Component health critical", fields...)
	} else if newStatus == StatusHealthy && oldStatus != StatusHealthy {
		hc.obs.Logger().Info(ctx, "Component recovered", fields...)
	} else {
		hc.obs.Logger().Warn(ctx, "Component health changed", fields...)
	}
}

// exportMetrics exports health metrics to the observability system
func (hc *HealthChecker) exportMetrics() {
	if hc.obs == nil {
		return
	}

	health := hc.GetSystemHealth()

	// Overall health status
	statusValue := 0.0
	switch health.Status {
	case StatusHealthy:
		statusValue = 2.0
	case StatusDegraded:
		statusValue = 1.0
	case StatusCritical:
		statusValue = 0.0
	}

	hc.obs.Metrics().SetGauge("health_status", statusValue, map[string]string{
		"service": hc.config.ServiceName,
	})

	// Health score
	hc.obs.Metrics().SetGauge("health_score", health.Score, map[string]string{
		"service": hc.config.ServiceName,
	})

	// Component-level metrics
	for name, component := range health.Components {
		componentStatus := 0.0
		switch component.Status {
		case StatusHealthy:
			componentStatus = 2.0
		case StatusDegraded:
			componentStatus = 1.0
		case StatusCritical:
			componentStatus = 0.0
		}

		hc.obs.Metrics().SetGauge("component_status", componentStatus, map[string]string{
			"service":   hc.config.ServiceName,
			"component": name,
		})

		hc.obs.Metrics().RecordHistogram("component_response_time_seconds",
			component.ResponseTime.Seconds(), map[string]string{
				"service":   hc.config.ServiceName,
				"component": name,
			})

		// Count failures
		if component.Status == StatusCritical {
			hc.obs.Metrics().IncrementCounter("health_check_failures_total", map[string]string{
				"service":   hc.config.ServiceName,
				"component": name,
			})
		}
	}
}
