package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Check executes all registered readiness checks in parallel and updates
// the readiness status based on the results. This method implements anti-flapping
// logic by requiring multiple consecutive failures before marking as not ready.
//
// The readiness probe considers:
// - Critical checks: Must pass for service to be ready
// - Non-critical checks: Failures are logged but don't affect readiness
// - Anti-flapping: Requires FailureThreshold consecutive failures before marking not ready
// - Immediate recovery: First success immediately marks as ready
//
// Returns nil if ready, error if not ready.
func (rp *ReadinessProbe) Check(ctx context.Context) error {
	if len(rp.checks) == 0 {
		// No checks registered, assume ready
		return nil
	}

	// Execute all checks in parallel
	results := make(chan checkResult, len(rp.checks))
	var wg sync.WaitGroup

	for _, check := range rp.checks {
		wg.Add(1)
		go func(c Check) {
			defer wg.Done()
			
			// Set timeout for this check
			timeout := c.Timeout()
			if timeout == 0 {
				timeout = 100 * time.Millisecond
			}

			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Execute the check
			start := time.Now()
			err := c.Check(checkCtx)
			duration := time.Since(start)

			results <- checkResult{
				name:     c.Name(),
				err:      err,
				duration: duration,
				critical: c.Critical(),
			}
		}(check)
	}

	// Wait for all checks to complete
	wg.Wait()
	close(results)

	// Collect results and identify critical failures
	var criticalErrors []error
	checkResults := make(map[string]error)

	for result := range results {
		checkResults[result.name] = result.err
		
		// Only critical checks affect readiness
		if result.critical && result.err != nil {
			criticalErrors = append(criticalErrors, 
				fmt.Errorf("%s: %w", result.name, result.err))
		}
	}

	// Update failure counts and determine readiness
	return rp.updateReadinessStatus(checkResults, criticalErrors)
}

// updateReadinessStatus updates the readiness flag based on check results
// and anti-flapping logic. It requires FailureThreshold consecutive failures
// before marking as not ready, but immediately marks as ready on first success.
func (rp *ReadinessProbe) updateReadinessStatus(checkResults map[string]error, criticalErrors []error) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	// Update failure counts for each check
	for checkName, err := range checkResults {
		if err != nil {
			// Increment failure count
			rp.failureCount[checkName]++
		} else {
			// Reset failure count on success
			rp.failureCount[checkName] = 0
		}
	}

	// Check if any critical check has exceeded failure threshold
	hasExceededThreshold := false
	var failedChecks []string

	for checkName, count := range rp.failureCount {
		if count >= rp.failureThreshold {
			// Check if this is a critical check
			for _, check := range rp.checks {
				if check.Name() == checkName && check.Critical() {
					hasExceededThreshold = true
					failedChecks = append(failedChecks, checkName)
					break
				}
			}
		}
	}

	// Update readiness status
	if hasExceededThreshold {
		// Mark as not ready (anti-flapping threshold exceeded)
		oldValue := rp.isReady.Swap(0)
		if oldValue == 1 {
			// State changed from ready to not ready
			return fmt.Errorf("service not ready: critical checks failed after %d consecutive failures: %v",
				rp.failureThreshold, failedChecks)
		}
		return fmt.Errorf("service not ready: %v", failedChecks)
	}

	// No critical failures or threshold not exceeded - mark as ready
	oldValue := rp.isReady.Swap(1)
	if oldValue == 0 && len(criticalErrors) == 0 {
		// State changed from not ready to ready (recovery)
		return nil
	}

	// Still ready, but may have some failures below threshold
	if len(criticalErrors) > 0 {
		// Have critical errors but below threshold (anti-flapping in progress)
		return fmt.Errorf("critical checks failing but below threshold (%d/%d): %v",
			len(criticalErrors), rp.failureThreshold, criticalErrors)
	}

	return nil
}

// IsReady returns true if the service is ready to serve traffic.
// This is a lock-free atomic operation that's very fast (< 1μs).
func (rp *ReadinessProbe) IsReady() bool {
	return rp.isReady.Load() == 1
}

// GetFailureCount returns the current failure count for a specific check.
// This is useful for debugging and monitoring.
func (rp *ReadinessProbe) GetFailureCount(checkName string) int {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.failureCount[checkName]
}

// ResetFailureCount resets the failure count for a specific check.
// This can be used to manually clear anti-flapping state.
func (rp *ReadinessProbe) ResetFailureCount(checkName string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.failureCount[checkName] = 0
}

// ResetAllFailureCounts resets all failure counts.
// This can be used to manually clear all anti-flapping state.
func (rp *ReadinessProbe) ResetAllFailureCounts() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	for checkName := range rp.failureCount {
		rp.failureCount[checkName] = 0
	}
}

// Name returns the name of this probe
func (rp *ReadinessProbe) Name() string {
	return "readiness"
}

// Timeout returns the timeout for this probe
func (rp *ReadinessProbe) Timeout() time.Duration {
	return 1 * time.Second
}

// Interval returns the check interval for this probe
func (rp *ReadinessProbe) Interval() time.Duration {
	return 5 * time.Second
}

// Critical returns whether this probe is critical
func (rp *ReadinessProbe) Critical() bool {
	return true
}

// checkResult is an internal type for collecting check results
type checkResult struct {
	name     string
	err      error
	duration time.Duration
	critical bool
}
