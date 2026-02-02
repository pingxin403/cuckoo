package health

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// startHeartbeat starts the heartbeat goroutine that periodically updates
// the last heartbeat timestamp. This is used to detect goroutine deadlocks.
func (lp *LivenessProbe) startHeartbeat(wg *sync.WaitGroup, stopCh chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(lp.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lp.lastHeartbeat.Store(time.Now())
		case <-stopCh:
			return
		}
	}
}

// Check performs the liveness check, verifying:
// 1. Heartbeat is recent (no deadlock)
// 2. Memory usage is within limits
// 3. Goroutine count is within limits
//
// Returns nil if the process is alive, error otherwise.
func (lp *LivenessProbe) Check(ctx context.Context) error {
	// Check heartbeat (detect deadlock)
	lastHeartbeat, ok := lp.lastHeartbeat.Load().(time.Time)
	if !ok || lastHeartbeat.IsZero() {
		// Heartbeat not initialized yet, assume alive
		return nil
	}

	timeSinceHeartbeat := time.Since(lastHeartbeat)
	if timeSinceHeartbeat > lp.heartbeatTimeout {
		return fmt.Errorf("heartbeat timeout: no heartbeat for %v (threshold: %v)",
			timeSinceHeartbeat, lp.heartbeatTimeout)
	}

	// Check memory usage
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if ms.Alloc > lp.memoryLimit {
		return fmt.Errorf("memory limit exceeded: %d bytes > %d bytes (%.2f%%)",
			ms.Alloc, lp.memoryLimit, float64(ms.Alloc)/float64(lp.memoryLimit)*100)
	}

	// Check goroutine count
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > lp.goroutineLimit {
		return fmt.Errorf("goroutine limit exceeded: %d > %d",
			numGoroutines, lp.goroutineLimit)
	}

	return nil
}

// Name returns the name of this check
func (lp *LivenessProbe) Name() string {
	return "liveness"
}

// Timeout returns the timeout for this check
func (lp *LivenessProbe) Timeout() time.Duration {
	return 1 * time.Second
}

// Interval returns the check interval
func (lp *LivenessProbe) Interval() time.Duration {
	return lp.heartbeatInterval
}

// Critical returns whether this check is critical
func (lp *LivenessProbe) Critical() bool {
	return true
}
