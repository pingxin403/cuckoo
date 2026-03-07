package connpool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the health status of a connection pool
type HealthStatus struct {
	Name                 string        `json:"name"`
	Type                 string        `json:"type"` // database, redis, kafka_producer, kafka_consumer
	Healthy              bool          `json:"healthy"`
	LastCheck            time.Time     `json:"last_check"`
	LastSuccess          time.Time     `json:"last_success"`
	LastFailure          time.Time     `json:"last_failure"`
	ConsecutiveFailures  int           `json:"consecutive_failures"`
	ConsecutiveSuccesses int           `json:"consecutive_successes"`
	Message              string        `json:"message,omitempty"`
	Latency              time.Duration `json:"latency"`
}

// HealthChecker performs periodic health checks on connection pools
type HealthChecker struct {
	config HealthCheckConfig

	// Registered pools
	mu             sync.RWMutex
	databases      map[string]*DatabasePool
	redisClients   map[string]*RedisPool
	kafkaProducers map[string]*KafkaProducerPool
	kafkaConsumers map[string]*KafkaConsumerPool

	// Health status
	statusMu sync.RWMutex
	status   map[string]*HealthStatus
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config HealthCheckConfig) *HealthChecker {
	return &HealthChecker{
		config:         config,
		databases:      make(map[string]*DatabasePool),
		redisClients:   make(map[string]*RedisPool),
		kafkaProducers: make(map[string]*KafkaProducerPool),
		kafkaConsumers: make(map[string]*KafkaConsumerPool),
		status:         make(map[string]*HealthStatus),
	}
}

// RegisterDatabase registers a database pool for health checks
func (hc *HealthChecker) RegisterDatabase(name string, pool *DatabasePool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.databases[name] = pool

	hc.statusMu.Lock()
	hc.status[name] = &HealthStatus{
		Name:    name,
		Type:    "database",
		Healthy: true,
	}
	hc.statusMu.Unlock()
}

// RegisterRedis registers a Redis pool for health checks
func (hc *HealthChecker) RegisterRedis(name string, pool *RedisPool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.redisClients[name] = pool

	hc.statusMu.Lock()
	hc.status[name] = &HealthStatus{
		Name:    name,
		Type:    "redis",
		Healthy: true,
	}
	hc.statusMu.Unlock()
}

// RegisterKafkaProducer registers a Kafka producer pool for health checks
func (hc *HealthChecker) RegisterKafkaProducer(name string, pool *KafkaProducerPool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.kafkaProducers[name] = pool

	hc.statusMu.Lock()
	hc.status[name] = &HealthStatus{
		Name:    name,
		Type:    "kafka_producer",
		Healthy: true,
	}
	hc.statusMu.Unlock()
}

// RegisterKafkaConsumer registers a Kafka consumer pool for health checks
func (hc *HealthChecker) RegisterKafkaConsumer(name string, pool *KafkaConsumerPool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.kafkaConsumers[name] = pool

	hc.statusMu.Lock()
	hc.status[name] = &HealthStatus{
		Name:    name,
		Type:    "kafka_consumer",
		Healthy: true,
	}
	hc.statusMu.Unlock()
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	// Perform initial health check
	hc.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAll(ctx)
		}
	}
}

// checkAll performs health checks on all registered pools
func (hc *HealthChecker) checkAll(ctx context.Context) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Check databases
	for name, pool := range hc.databases {
		hc.checkDatabase(ctx, name, pool)
	}

	// Check Redis
	for name, pool := range hc.redisClients {
		hc.checkRedis(ctx, name, pool)
	}

	// Check Kafka producers
	for name, pool := range hc.kafkaProducers {
		hc.checkKafkaProducer(ctx, name, pool)
	}

	// Check Kafka consumers
	for name, pool := range hc.kafkaConsumers {
		hc.checkKafkaConsumer(ctx, name, pool)
	}
}

// checkDatabase performs a health check on a database pool
func (hc *HealthChecker) checkDatabase(ctx context.Context, name string, pool *DatabasePool) {
	checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
	defer cancel()

	start := time.Now()
	err := pool.Ping(checkCtx)
	latency := time.Since(start)

	hc.updateStatus(name, err == nil, latency, err)

	// Perform optimization if healthy
	if err == nil {
		pool.Optimize()
	}
}

// checkRedis performs a health check on a Redis pool
func (hc *HealthChecker) checkRedis(ctx context.Context, name string, pool *RedisPool) {
	checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
	defer cancel()

	start := time.Now()
	err := pool.Ping(checkCtx)
	latency := time.Since(start)

	hc.updateStatus(name, err == nil, latency, err)

	// Perform optimization if healthy
	if err == nil {
		pool.Optimize()
	}
}

// checkKafkaProducer performs a health check on a Kafka producer pool
func (hc *HealthChecker) checkKafkaProducer(ctx context.Context, name string, pool *KafkaProducerPool) {
	start := time.Now()
	healthy := pool.IsHealthy(ctx)
	latency := time.Since(start)

	var err error
	if !healthy {
		metrics := pool.GetMetrics()
		err = metrics.LastError
	}

	hc.updateStatus(name, healthy, latency, err)
}

// checkKafkaConsumer performs a health check on a Kafka consumer pool
func (hc *HealthChecker) checkKafkaConsumer(ctx context.Context, name string, pool *KafkaConsumerPool) {
	start := time.Now()
	healthy := pool.IsHealthy(ctx)
	latency := time.Since(start)

	var err error
	if !healthy {
		metrics := pool.GetMetrics()
		err = metrics.LastError
	}

	hc.updateStatus(name, healthy, latency, err)
}

// updateStatus updates the health status of a pool
func (hc *HealthChecker) updateStatus(name string, healthy bool, latency time.Duration, err error) {
	hc.statusMu.Lock()
	defer hc.statusMu.Unlock()

	status, exists := hc.status[name]
	if !exists {
		status = &HealthStatus{
			Name: name,
		}
		hc.status[name] = status
	}

	status.LastCheck = time.Now()
	status.Latency = latency

	if healthy {
		status.LastSuccess = time.Now()
		status.ConsecutiveFailures = 0
		status.ConsecutiveSuccesses++
		status.Message = ""

		// Mark as healthy if we reach success threshold
		if status.ConsecutiveSuccesses >= hc.config.SuccessThreshold {
			status.Healthy = true
		}
	} else {
		status.LastFailure = time.Now()
		status.ConsecutiveSuccesses = 0
		status.ConsecutiveFailures++

		if err != nil {
			status.Message = err.Error()
		} else {
			status.Message = "Health check failed"
		}

		// Mark as unhealthy if we reach failure threshold
		if status.ConsecutiveFailures >= hc.config.FailureThreshold {
			status.Healthy = false
		}
	}
}

// GetStatus returns the health status of a specific pool
func (hc *HealthChecker) GetStatus(name string) *HealthStatus {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	status, exists := hc.status[name]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	statusCopy := *status
	return &statusCopy
}

// GetAllStatus returns the health status of all pools
func (hc *HealthChecker) GetAllStatus() map[string]HealthStatus {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	result := make(map[string]HealthStatus, len(hc.status))
	for name, status := range hc.status {
		result[name] = *status
	}

	return result
}

// IsAllHealthy returns true if all pools are healthy
func (hc *HealthChecker) IsAllHealthy() bool {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	for _, status := range hc.status {
		if !status.Healthy {
			return false
		}
	}

	return true
}

// GetUnhealthyPools returns a list of unhealthy pool names
func (hc *HealthChecker) GetUnhealthyPools() []string {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	var unhealthy []string
	for name, status := range hc.status {
		if !status.Healthy {
			unhealthy = append(unhealthy, name)
		}
	}

	return unhealthy
}

// GetHealthSummary returns a summary of health status
func (hc *HealthChecker) GetHealthSummary() HealthSummary {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	summary := HealthSummary{
		TotalPools:     len(hc.status),
		HealthyPools:   0,
		UnhealthyPools: 0,
		CheckTime:      time.Now(),
	}

	for _, status := range hc.status {
		if status.Healthy {
			summary.HealthyPools++
		} else {
			summary.UnhealthyPools++
			summary.UnhealthyNames = append(summary.UnhealthyNames, status.Name)
		}
	}

	summary.OverallHealthy = summary.UnhealthyPools == 0

	return summary
}

// HealthSummary provides a summary of health status
type HealthSummary struct {
	TotalPools     int       `json:"total_pools"`
	HealthyPools   int       `json:"healthy_pools"`
	UnhealthyPools int       `json:"unhealthy_pools"`
	UnhealthyNames []string  `json:"unhealthy_names,omitempty"`
	OverallHealthy bool      `json:"overall_healthy"`
	CheckTime      time.Time `json:"check_time"`
}

// String returns a string representation of the health summary
func (hs HealthSummary) String() string {
	if hs.OverallHealthy {
		return fmt.Sprintf("All %d pools are healthy", hs.TotalPools)
	}
	return fmt.Sprintf("%d/%d pools unhealthy: %v", hs.UnhealthyPools, hs.TotalPools, hs.UnhealthyNames)
}
