package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReadinessProbe_Check_NoChecks(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	err := rp.Check(context.Background())
	if err != nil {
		t.Errorf("Expected no error with no checks, got %v", err)
	}

	if !rp.IsReady() {
		t.Error("Expected ready with no checks")
	}
}

func TestReadinessProbe_Check_AllPassing(t *testing.T) {
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name:     "check1",
				checkFn:  func(ctx context.Context) error { return nil },
				timeout:  100 * time.Millisecond,
				critical: true,
			},
			&MockCheck{
				name:     "check2",
				checkFn:  func(ctx context.Context) error { return nil },
				timeout:  100 * time.Millisecond,
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	err := rp.Check(context.Background())
	if err != nil {
		t.Errorf("Expected no error with all passing checks, got %v", err)
	}

	if !rp.IsReady() {
		t.Error("Expected ready with all passing checks")
	}

	// Verify failure counts are reset
	if rp.GetFailureCount("check1") != 0 {
		t.Errorf("Expected failure count 0 for check1, got %d", rp.GetFailureCount("check1"))
	}
	if rp.GetFailureCount("check2") != 0 {
		t.Errorf("Expected failure count 0 for check2, got %d", rp.GetFailureCount("check2"))
	}
}

func TestReadinessProbe_Check_NonCriticalFailure(t *testing.T) {
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name:     "critical-check",
				checkFn:  func(ctx context.Context) error { return nil },
				timeout:  100 * time.Millisecond,
				critical: true,
			},
			&MockCheck{
				name:     "non-critical-check",
				checkFn:  func(ctx context.Context) error { return errors.New("failed") },
				timeout:  100 * time.Millisecond,
				critical: false, // Non-critical
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	err := rp.Check(context.Background())
	if err != nil {
		t.Errorf("Expected no error with non-critical failure, got %v", err)
	}

	// Should still be ready because only non-critical check failed
	if !rp.IsReady() {
		t.Error("Expected ready with non-critical failure")
	}

	// Failure count should still be tracked
	if rp.GetFailureCount("non-critical-check") != 1 {
		t.Errorf("Expected failure count 1 for non-critical-check, got %d", rp.GetFailureCount("non-critical-check"))
	}
}

func TestReadinessProbe_Check_AntiFlapping(t *testing.T) {
	failCount := 0
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name: "flaky-check",
				checkFn: func(ctx context.Context) error {
					failCount++
					return errors.New("failed")
				},
				timeout:  100 * time.Millisecond,
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	// First failure - should still be ready (anti-flapping)
	err := rp.Check(context.Background())
	if err == nil {
		t.Error("Expected error on first failure")
	}
	if !rp.IsReady() {
		t.Error("Expected ready after first failure (anti-flapping)")
	}
	if rp.GetFailureCount("flaky-check") != 1 {
		t.Errorf("Expected failure count 1, got %d", rp.GetFailureCount("flaky-check"))
	}

	// Second failure - should still be ready
	err = rp.Check(context.Background())
	if err == nil {
		t.Error("Expected error on second failure")
	}
	if !rp.IsReady() {
		t.Error("Expected ready after second failure (anti-flapping)")
	}
	if rp.GetFailureCount("flaky-check") != 2 {
		t.Errorf("Expected failure count 2, got %d", rp.GetFailureCount("flaky-check"))
	}

	// Third failure - should now be not ready (threshold exceeded)
	err = rp.Check(context.Background())
	if err == nil {
		t.Error("Expected error on third failure")
	}
	if rp.IsReady() {
		t.Error("Expected not ready after third failure (threshold exceeded)")
	}
	if rp.GetFailureCount("flaky-check") != 3 {
		t.Errorf("Expected failure count 3, got %d", rp.GetFailureCount("flaky-check"))
	}

	// Fourth failure - should remain not ready
	err = rp.Check(context.Background())
	if err == nil {
		t.Error("Expected error on fourth failure")
	}
	if rp.IsReady() {
		t.Error("Expected not ready after fourth failure")
	}
}

func TestReadinessProbe_Check_ImmediateRecovery(t *testing.T) {
	shouldFail := true
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name: "recovering-check",
				checkFn: func(ctx context.Context) error {
					if shouldFail {
						return errors.New("failed")
					}
					return nil
				},
				timeout:  100 * time.Millisecond,
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	// Fail 3 times to exceed threshold
	for i := 0; i < 3; i++ {
		rp.Check(context.Background())
	}

	// Should be not ready
	if rp.IsReady() {
		t.Error("Expected not ready after 3 failures")
	}

	// Now succeed - should immediately become ready
	shouldFail = false
	err := rp.Check(context.Background())
	if err != nil {
		t.Errorf("Expected no error on recovery, got %v", err)
	}

	if !rp.IsReady() {
		t.Error("Expected immediate recovery to ready state")
	}

	// Failure count should be reset
	if rp.GetFailureCount("recovering-check") != 0 {
		t.Errorf("Expected failure count 0 after recovery, got %d", rp.GetFailureCount("recovering-check"))
	}
}

func TestReadinessProbe_Check_ParallelExecution(t *testing.T) {
	// Create checks with different delays to verify parallel execution
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name: "slow-check",
				checkFn: func(ctx context.Context) error {
					time.Sleep(50 * time.Millisecond)
					return nil
				},
				timeout:  100 * time.Millisecond,
				critical: true,
			},
			&MockCheck{
				name: "fast-check",
				checkFn: func(ctx context.Context) error {
					return nil
				},
				timeout:  100 * time.Millisecond,
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	start := time.Now()
	err := rp.Check(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// If executed serially, would take 50ms + 0ms = 50ms
	// If executed in parallel, should take ~50ms (not 50ms + overhead)
	// Allow some overhead for goroutine scheduling
	if duration > 100*time.Millisecond {
		t.Errorf("Checks appear to be running serially, took %v", duration)
	}
}

func TestReadinessProbe_Check_Timeout(t *testing.T) {
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name: "timeout-check",
				checkFn: func(ctx context.Context) error {
					// Wait for context cancellation
					<-ctx.Done()
					return ctx.Err()
				},
				timeout:  10 * time.Millisecond, // Very short timeout
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	start := time.Now()
	err := rp.Check(context.Background())
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error due to timeout")
	}

	// Should timeout quickly
	if duration > 50*time.Millisecond {
		t.Errorf("Timeout took too long: %v", duration)
	}

	// Should track failure
	if rp.GetFailureCount("timeout-check") != 1 {
		t.Errorf("Expected failure count 1, got %d", rp.GetFailureCount("timeout-check"))
	}
}

func TestReadinessProbe_Check_MixedCriticalAndNonCritical(t *testing.T) {
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name:     "critical-passing",
				checkFn:  func(ctx context.Context) error { return nil },
				timeout:  100 * time.Millisecond,
				critical: true,
			},
			&MockCheck{
				name:     "critical-failing",
				checkFn:  func(ctx context.Context) error { return errors.New("failed") },
				timeout:  100 * time.Millisecond,
				critical: true,
			},
			&MockCheck{
				name:     "non-critical-failing",
				checkFn:  func(ctx context.Context) error { return errors.New("failed") },
				timeout:  100 * time.Millisecond,
				critical: false,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	// First failure - should still be ready
	rp.Check(context.Background())
	if !rp.IsReady() {
		t.Error("Expected ready after first critical failure")
	}

	// Second failure - should still be ready
	rp.Check(context.Background())
	if !rp.IsReady() {
		t.Error("Expected ready after second critical failure")
	}

	// Third failure - should now be not ready
	rp.Check(context.Background())
	if rp.IsReady() {
		t.Error("Expected not ready after third critical failure")
	}

	// Non-critical check should have failures tracked but not affect readiness
	if rp.GetFailureCount("non-critical-failing") != 3 {
		t.Errorf("Expected failure count 3 for non-critical check, got %d", rp.GetFailureCount("non-critical-failing"))
	}
}

func TestReadinessProbe_IsReady(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}

	// Initially ready
	rp.isReady.Store(1)
	if !rp.IsReady() {
		t.Error("Expected IsReady to return true")
	}

	// Mark as not ready
	rp.isReady.Store(0)
	if rp.IsReady() {
		t.Error("Expected IsReady to return false")
	}
}

func TestReadinessProbe_GetFailureCount(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}

	// Initially zero
	if count := rp.GetFailureCount("test-check"); count != 0 {
		t.Errorf("Expected failure count 0, got %d", count)
	}

	// Set failure count
	rp.failureCount["test-check"] = 5
	if count := rp.GetFailureCount("test-check"); count != 5 {
		t.Errorf("Expected failure count 5, got %d", count)
	}
}

func TestReadinessProbe_ResetFailureCount(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}

	// Set failure count
	rp.failureCount["test-check"] = 5

	// Reset
	rp.ResetFailureCount("test-check")

	if count := rp.GetFailureCount("test-check"); count != 0 {
		t.Errorf("Expected failure count 0 after reset, got %d", count)
	}
}

func TestReadinessProbe_ResetAllFailureCounts(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}

	// Set multiple failure counts
	rp.failureCount["check1"] = 3
	rp.failureCount["check2"] = 5
	rp.failureCount["check3"] = 2

	// Reset all
	rp.ResetAllFailureCounts()

	if count := rp.GetFailureCount("check1"); count != 0 {
		t.Errorf("Expected failure count 0 for check1, got %d", count)
	}
	if count := rp.GetFailureCount("check2"); count != 0 {
		t.Errorf("Expected failure count 0 for check2, got %d", count)
	}
	if count := rp.GetFailureCount("check3"); count != 0 {
		t.Errorf("Expected failure count 0 for check3, got %d", count)
	}
}

func TestReadinessProbe_CheckInterface(t *testing.T) {
	rp := &ReadinessProbe{
		checks:           []Check{},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}

	// Verify ReadinessProbe implements Check interface
	var _ Check = rp

	if rp.Name() != "readiness" {
		t.Errorf("Expected name 'readiness', got %s", rp.Name())
	}

	if rp.Timeout() != 1*time.Second {
		t.Errorf("Expected timeout 1s, got %v", rp.Timeout())
	}

	if rp.Interval() != 5*time.Second {
		t.Errorf("Expected interval 5s, got %v", rp.Interval())
	}

	if !rp.Critical() {
		t.Error("Expected readiness probe to be critical")
	}
}

func TestReadinessProbe_ConcurrentAccess(t *testing.T) {
	rp := &ReadinessProbe{
		checks: []Check{
			&MockCheck{
				name:     "test-check",
				checkFn:  func(ctx context.Context) error { return nil },
				timeout:  10 * time.Millisecond,
				critical: true,
			},
		},
		failureCount:     make(map[string]int),
		failureThreshold: 3,
	}
	rp.isReady.Store(1)

	// Run multiple checks concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				rp.Check(context.Background())
				rp.IsReady()
				rp.GetFailureCount("test-check")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or race
}
