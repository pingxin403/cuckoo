package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/worker"
)

// EtcdHealthCheck checks the health of the etcd registry client
type EtcdHealthCheck struct {
	name           string
	registryClient *registry.RegistryClient
	timeout        time.Duration
	interval       time.Duration
	critical       bool
}

// NewEtcdHealthCheck creates a new etcd health check
func NewEtcdHealthCheck(name string, registryClient *registry.RegistryClient) *EtcdHealthCheck {
	return &EtcdHealthCheck{
		name:           name,
		registryClient: registryClient,
		timeout:        200 * time.Millisecond,
		interval:       10 * time.Second,
		critical:       true, // etcd is critical for service discovery
	}
}

func (e *EtcdHealthCheck) Name() string {
	return e.name
}

func (e *EtcdHealthCheck) Timeout() time.Duration {
	return e.timeout
}

func (e *EtcdHealthCheck) Interval() time.Duration {
	return e.interval
}

func (e *EtcdHealthCheck) Critical() bool {
	return e.critical
}

func (e *EtcdHealthCheck) Check(ctx context.Context) error {
	// Try to perform a simple operation to verify etcd connectivity
	// We'll use the LookupUser method with a test user ID
	// If etcd is down, this will fail with a connection error
	_, err := e.registryClient.LookupUser(ctx, "health-check-test-user")
	if err != nil {
		// If the error is about user not found, that's OK - it means etcd is responding
		// We only care if there's a connection error
		if err.Error() == "user not found" || err.Error() == "no devices found for user health-check-test-user" {
			return nil
		}
		return fmt.Errorf("etcd health check failed: %w", err)
	}
	return nil
}

// OfflineWorkerHealthCheck checks the health of the offline worker
type OfflineWorkerHealthCheck struct {
	name     string
	worker   *worker.OfflineWorker
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewOfflineWorkerHealthCheck creates a new offline worker health check
func NewOfflineWorkerHealthCheck(name string, w *worker.OfflineWorker) *OfflineWorkerHealthCheck {
	return &OfflineWorkerHealthCheck{
		name:     name,
		worker:   w,
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: false, // Worker is not critical - service can run without it
	}
}

func (o *OfflineWorkerHealthCheck) Name() string {
	return o.name
}

func (o *OfflineWorkerHealthCheck) Timeout() time.Duration {
	return o.timeout
}

func (o *OfflineWorkerHealthCheck) Interval() time.Duration {
	return o.interval
}

func (o *OfflineWorkerHealthCheck) Critical() bool {
	return o.critical
}

func (o *OfflineWorkerHealthCheck) Check(ctx context.Context) error {
	if o.worker == nil {
		return fmt.Errorf("offline worker is not initialized")
	}

	stats := o.worker.GetStats()

	// Check if worker has errors but no messages processed
	// This indicates the worker is stuck or failing
	if stats.Errors > 0 && stats.MessagesProcessed == 0 {
		return fmt.Errorf("worker has %d errors but no messages processed", stats.Errors)
	}

	// Check error rate if we have processed messages
	if stats.MessagesProcessed > 0 {
		errorRate := float64(stats.Errors) / float64(stats.MessagesProcessed)
		if errorRate > 0.1 { // 10% error rate threshold
			return fmt.Errorf("worker error rate too high: %.2f%% (%d errors / %d processed)",
				errorRate*100, stats.Errors, stats.MessagesProcessed)
		}
	}

	return nil
}
