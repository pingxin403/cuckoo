package health

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// HealthStatus represents the health state of a component or system
type HealthStatus string

const (
	// StatusHealthy indicates all components are operational (score >= 0.8)
	StatusHealthy HealthStatus = "healthy"
	// StatusDegraded indicates some components are slow/failing (0.5 <= score < 0.8)
	StatusDegraded HealthStatus = "degraded"
	// StatusCritical indicates critical failures (score < 0.5)
	StatusCritical HealthStatus = "critical"
)

// Check defines the interface for health checks
type Check interface {
	// Name returns the unique name of this health check
	Name() string
	// Check performs the health check and returns an error if unhealthy
	Check(ctx context.Context) error
	// Timeout returns the maximum duration for this check
	Timeout() time.Duration
	// Interval returns how often this check should run
	Interval() time.Duration
	// Critical returns true if failure of this check marks service as not ready
	Critical() bool
}

// CheckResult holds the result of a health check execution
type CheckResult struct {
	// Name of the check
	Name string
	// Status of the check
	Status HealthStatus
	// LastCheck is the timestamp of the last check execution
	LastCheck time.Time
	// ResponseTime is how long the check took to execute
	ResponseTime time.Duration
	// Error message if the check failed (empty if successful)
	Error string
	// FailureCount tracks consecutive failures for anti-flapping
	FailureCount int
	// SuccessCount tracks consecutive successes for recovery detection
	SuccessCount int
}

// SystemHealth represents the overall health status of the system
type SystemHealth struct {
	// Status is the overall health status
	Status HealthStatus
	// Service name
	Service string
	// Timestamp when this health status was generated
	Timestamp time.Time
	// Score is a numerical health score (0.0 to 1.0)
	Score float64
	// Summary is a human-readable summary of the health status
	Summary string
	// Components contains the health status of individual components
	Components map[string]*ComponentHealth
}

// ComponentHealth represents the health status of a single component
type ComponentHealth struct {
	// Name of the component
	Name string
	// Status of the component
	Status HealthStatus
	// LastCheck is when this component was last checked
	LastCheck time.Time
	// ResponseTime is how long the last check took
	ResponseTime time.Duration
	// Error message if the component is unhealthy
	Error string
}

// HealthChecker manages health checks for a service
type HealthChecker struct {
	config         Config
	obs            observability.Observability
	livenessProbe  *LivenessProbe
	readinessProbe *ReadinessProbe
	checks         map[string]Check
	results        map[string]*CheckResult
	mu             sync.RWMutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// Config holds health checker configuration
type Config struct {
	// ServiceName identifies the service
	ServiceName string
	// CheckInterval is how often health checks run (default: 5s)
	CheckInterval time.Duration
	// DefaultTimeout is the default timeout for checks (default: 100ms)
	DefaultTimeout time.Duration
	// HealthyScore is the minimum score for healthy status (default: 0.8)
	HealthyScore float64
	// DegradedScore is the minimum score for degraded status (default: 0.5)
	DegradedScore float64
	// FailureThreshold is consecutive failures before marking not ready (default: 3)
	FailureThreshold int
	// LivenessConfig configures the liveness probe
	LivenessConfig LivenessConfig
	// CircuitBreakerConfig configures circuit breakers
	CircuitBreakerConfig CircuitBreakerConfig
	// RecoveryConfig configures auto-recovery
	RecoveryConfig RecoveryConfig
}

// LivenessConfig configures the liveness probe
type LivenessConfig struct {
	// HeartbeatInterval is how often the heartbeat is sent (default: 1s)
	HeartbeatInterval time.Duration
	// HeartbeatTimeout is the maximum time without heartbeat before marking dead (default: 10s)
	HeartbeatTimeout time.Duration
	// MemoryLimit is the maximum memory usage in bytes (default: 4GB)
	MemoryLimit uint64
	// GoroutineLimit is the maximum number of goroutines (default: 10000)
	GoroutineLimit int
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	// MaxFailures is the number of failures before opening circuit (default: 3)
	MaxFailures int
	// Timeout is how long to wait before trying half-open (default: 30s)
	Timeout time.Duration
	// HalfOpenTimeout is how long to test in half-open state (default: 10s)
	HalfOpenTimeout time.Duration
}

// RecoveryConfig configures auto-recovery behavior
type RecoveryConfig struct {
	// Enabled determines if auto-recovery is enabled
	Enabled bool
	// MaxRetries is the maximum number of recovery attempts (default: 3)
	MaxRetries int
	// InitialBackoff is the initial backoff duration (default: 1s)
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration (default: 30s)
	MaxBackoff time.Duration
	// BackoffFactor is the exponential backoff multiplier (default: 2.0)
	BackoffFactor float64
}

// LivenessProbe checks if the service process is alive
type LivenessProbe struct {
	// lastHeartbeat stores the last heartbeat time using atomic.Value
	lastHeartbeat atomic.Value // time.Time
	// memoryLimit is the maximum allowed memory usage
	memoryLimit uint64
	// goroutineLimit is the maximum allowed goroutine count
	goroutineLimit int
	// heartbeatInterval is how often heartbeats are sent
	heartbeatInterval time.Duration
	// heartbeatTimeout is the maximum time without heartbeat
	heartbeatTimeout time.Duration
	// stopCh signals the heartbeat goroutine to stop
	stopCh chan struct{}
}

// ReadinessProbe checks if the service is ready to serve traffic
type ReadinessProbe struct {
	// checks is the list of checks to execute
	checks []Check
	// isReady is an atomic flag (1 = ready, 0 = not ready)
	isReady atomic.Int32
	// failureCount tracks consecutive failures per check for anti-flapping
	failureCount map[string]int
	// mu protects failureCount map
	mu sync.RWMutex
	// failureThreshold is the number of consecutive failures before marking not ready
	failureThreshold int
}

// Recoverer defines the interface for auto-recovery mechanisms
type Recoverer interface {
	// Recover attempts to recover from a failure
	Recover(ctx context.Context) error
}

// State represents circuit breaker state
type State int

const (
	// StateClosed means normal operation
	StateClosed State = iota
	// StateOpen means failing, reject requests
	StateOpen
	// StateHalfOpen means testing recovery
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// name identifies this circuit breaker
	name string
	// maxFailures is the threshold for opening the circuit
	maxFailures int
	// timeout is how long to wait before trying half-open
	timeout time.Duration
	// halfOpenTimeout is how long to test in half-open state
	halfOpenTimeout time.Duration
	// state stores the current state using atomic.Value
	state atomic.Value // State
	// failures counts consecutive failures
	failures atomic.Int32
	// lastFailureTime stores the last failure time using atomic.Value
	lastFailureTime atomic.Value // time.Time
	// mu protects state transitions
	mu sync.RWMutex
}
