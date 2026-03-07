package health

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RecoveryManager manages recoverers for health checks
type RecoveryManager struct {
	// checks maps check names to their Check implementations
	checks map[string]Check
	// recoverers maps check names to their Recoverer implementations
	recoverers map[string]Recoverer
	// mu protects the maps
	mu sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager() *RecoveryManager {
	return &RecoveryManager{
		checks:     make(map[string]Check),
		recoverers: make(map[string]Recoverer),
	}
}

// RegisterCheck registers a health check
func (rm *RecoveryManager) RegisterCheck(check Check) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.checks[check.Name()] = check
}

// RegisterRecoverer registers a recoverer for a specific check
func (rm *RecoveryManager) RegisterRecoverer(checkName string, recoverer Recoverer) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.recoverers[checkName] = recoverer
}

// GetRecoverer returns the recoverer for a specific check, if registered
func (rm *RecoveryManager) GetRecoverer(checkName string) (Recoverer, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	recoverer, exists := rm.recoverers[checkName]
	return recoverer, exists
}

// AttemptRecovery attempts to recover a failed check
func (rm *RecoveryManager) AttemptRecovery(ctx context.Context, checkName string) error {
	recoverer, exists := rm.GetRecoverer(checkName)
	if !exists {
		return fmt.Errorf("no recoverer registered for check: %s", checkName)
	}

	return recoverer.Recover(ctx)
}

// DatabaseRecoverer attempts to reconnect to a database
type DatabaseRecoverer struct {
	// dsn is the database connection string
	dsn string
	// db is a pointer to the database connection to replace on recovery
	db **sql.DB
	// maxRetries is the maximum number of recovery attempts
	maxRetries int
	// backoff is the backoff strategy for retries
	backoff *BackoffStrategy
	// mu protects the recovery process
	mu sync.Mutex
}

// NewDatabaseRecoverer creates a new database recoverer
func NewDatabaseRecoverer(dsn string, db **sql.DB) *DatabaseRecoverer {
	return &DatabaseRecoverer{
		dsn:        dsn,
		db:         db,
		maxRetries: 3,
		backoff:    NewExponentialBackoff(),
	}
}

// NewDatabaseRecovererWithConfig creates a new database recoverer with custom configuration
func NewDatabaseRecovererWithConfig(dsn string, db **sql.DB, maxRetries int, backoff *BackoffStrategy) *DatabaseRecoverer {
	return &DatabaseRecoverer{
		dsn:        dsn,
		db:         db,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

// Recover attempts to reconnect to the database
func (dr *DatabaseRecoverer) Recover(ctx context.Context) error {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < dr.maxRetries; attempt++ {
		// Calculate backoff duration
		if attempt > 0 {
			backoffDuration := dr.backoff.Calculate(attempt - 1)
			select {
			case <-time.After(backoffDuration):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("recovery cancelled: %w", ctx.Err())
			}
		}

		// Attempt to open new database connection
		newDB, err := sql.Open("mysql", dr.dsn)
		if err != nil {
			lastErr = fmt.Errorf("failed to open database connection (attempt %d/%d): %w", attempt+1, dr.maxRetries, err)
			continue
		}

		// Verify connection with ping
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = newDB.PingContext(pingCtx)
		cancel()

		if err != nil {
			newDB.Close()
			lastErr = fmt.Errorf("failed to ping database (attempt %d/%d): %w", attempt+1, dr.maxRetries, err)
			continue
		}

		// Success - replace old connection
		if *dr.db != nil {
			(*dr.db).Close()
		}
		*dr.db = newDB

		return nil
	}

	return fmt.Errorf("failed to recover database connection after %d retries: %w", dr.maxRetries, lastErr)
}

// RedisRecoverer attempts to reconnect to Redis
type RedisRecoverer struct {
	// options are the Redis client options
	options *redis.Options
	// client is a pointer to the Redis client to replace on recovery
	client *redis.UniversalClient
	// maxRetries is the maximum number of recovery attempts
	maxRetries int
	// backoff is the backoff strategy for retries
	backoff *BackoffStrategy
	// mu protects the recovery process
	mu sync.Mutex
}

// NewRedisRecoverer creates a new Redis recoverer
func NewRedisRecoverer(options *redis.Options, client *redis.UniversalClient) *RedisRecoverer {
	return &RedisRecoverer{
		options:    options,
		client:     client,
		maxRetries: 3,
		backoff:    NewExponentialBackoff(),
	}
}

// NewRedisRecovererWithConfig creates a new Redis recoverer with custom configuration
func NewRedisRecovererWithConfig(options *redis.Options, client *redis.UniversalClient, maxRetries int, backoff *BackoffStrategy) *RedisRecoverer {
	return &RedisRecoverer{
		options:    options,
		client:     client,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

// Recover attempts to reconnect to Redis
func (rr *RedisRecoverer) Recover(ctx context.Context) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < rr.maxRetries; attempt++ {
		// Calculate backoff duration
		if attempt > 0 {
			backoffDuration := rr.backoff.Calculate(attempt - 1)
			select {
			case <-time.After(backoffDuration):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("recovery cancelled: %w", ctx.Err())
			}
		}

		// Create new Redis client
		newClient := redis.NewClient(rr.options)

		// Verify connection with ping
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := newClient.Ping(pingCtx).Result()
		cancel()

		if err != nil {
			newClient.Close()
			lastErr = fmt.Errorf("failed to ping Redis (attempt %d/%d): %w", attempt+1, rr.maxRetries, err)
			continue
		}

		// Success - replace old client
		if *rr.client != nil {
			(*rr.client).Close()
		}
		*rr.client = newClient

		return nil
	}

	return fmt.Errorf("failed to recover Redis connection after %d retries: %w", rr.maxRetries, lastErr)
}

// BackoffStrategy implements exponential backoff for retries
type BackoffStrategy struct {
	// initial is the initial backoff duration
	initial time.Duration
	// max is the maximum backoff duration
	max time.Duration
	// factor is the exponential backoff multiplier
	factor float64
}

// NewExponentialBackoff creates a new exponential backoff strategy with default values
func NewExponentialBackoff() *BackoffStrategy {
	return &BackoffStrategy{
		initial: 1 * time.Second,
		max:     30 * time.Second,
		factor:  2.0,
	}
}

// NewBackoffStrategy creates a new backoff strategy with custom values
func NewBackoffStrategy(initial, max time.Duration, factor float64) *BackoffStrategy {
	return &BackoffStrategy{
		initial: initial,
		max:     max,
		factor:  factor,
	}
}

// Calculate calculates the backoff duration for a given attempt number
// attempt is 0-indexed (0 = first retry, 1 = second retry, etc.)
func (bs *BackoffStrategy) Calculate(attempt int) time.Duration {
	// Calculate exponential backoff: initial * factor^attempt
	duration := float64(bs.initial) * math.Pow(bs.factor, float64(attempt))

	// Cap at maximum backoff
	if duration > float64(bs.max) {
		return bs.max
	}

	return time.Duration(duration)
}

// CalculateWithJitter calculates the backoff duration with jitter to avoid thundering herd
func (bs *BackoffStrategy) CalculateWithJitter(attempt int, jitterFactor float64) time.Duration {
	baseDuration := bs.Calculate(attempt)

	// Apply jitter: duration * (1 ± jitterFactor)
	// For example, with jitterFactor=0.1, the result will be between 90% and 110% of baseDuration
	if jitterFactor <= 0 || jitterFactor >= 1 {
		return baseDuration
	}

	jitter := float64(baseDuration) * jitterFactor * (2*float64(time.Now().UnixNano()%1000)/1000 - 1)
	result := float64(baseDuration) + jitter

	if result < 0 {
		return baseDuration
	}

	return time.Duration(result)
}
