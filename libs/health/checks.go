package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// DatabaseCheck implements health checking for SQL databases
// It performs both a ping check and a simple query to verify connectivity
type DatabaseCheck struct {
	name     string
	db       *sql.DB
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewDatabaseCheck creates a new database health check with default settings
// The check is marked as critical by default since database connectivity
// is typically essential for service operation.
//
// Example:
//
//	db, _ := sql.Open("mysql", dsn)
//	check := health.NewDatabaseCheck("database", db)
//	hc.RegisterCheck(check)
func NewDatabaseCheck(name string, db *sql.DB) *DatabaseCheck {
	return &DatabaseCheck{
		name:     name,
		db:       db,
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}
}

// NewDatabaseCheckWithOptions creates a database health check with custom settings
func NewDatabaseCheckWithOptions(name string, db *sql.DB, timeout, interval time.Duration, critical bool) *DatabaseCheck {
	return &DatabaseCheck{
		name:     name,
		db:       db,
		timeout:  timeout,
		interval: interval,
		critical: critical,
	}
}

// Name returns the name of this health check
func (d *DatabaseCheck) Name() string {
	return d.name
}

// Check performs the database health check
// It first pings the database, then executes a simple SELECT 1 query
func (d *DatabaseCheck) Check(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Ping check
	if err := d.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Simple query check to verify we can execute queries
	var result int
	err := d.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// Timeout returns the maximum duration for this check
func (d *DatabaseCheck) Timeout() time.Duration {
	return d.timeout
}

// Interval returns how often this check should run
func (d *DatabaseCheck) Interval() time.Duration {
	return d.interval
}

// Critical returns true if failure of this check marks service as not ready
func (d *DatabaseCheck) Critical() bool {
	return d.critical
}

// RedisCheck implements health checking for Redis
// It performs a ping operation to verify connectivity
type RedisCheck struct {
	name     string
	client   redis.UniversalClient
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewRedisCheck creates a new Redis health check with default settings
// The check is marked as critical by default.
//
// Example:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	check := health.NewRedisCheck("redis", client)
//	hc.RegisterCheck(check)
func NewRedisCheck(name string, client redis.UniversalClient) *RedisCheck {
	return &RedisCheck{
		name:     name,
		client:   client,
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}
}

// NewRedisCheckWithOptions creates a Redis health check with custom settings
func NewRedisCheckWithOptions(name string, client redis.UniversalClient, timeout, interval time.Duration, critical bool) *RedisCheck {
	return &RedisCheck{
		name:     name,
		client:   client,
		timeout:  timeout,
		interval: interval,
		critical: critical,
	}
}

// Name returns the name of this health check
func (r *RedisCheck) Name() string {
	return r.name
}

// Check performs the Redis health check by pinging the server
func (r *RedisCheck) Check(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	// Ping the Redis server
	result, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	// Verify we got the expected PONG response
	if result != "PONG" {
		return fmt.Errorf("redis ping returned unexpected result: %s", result)
	}

	return nil
}

// Timeout returns the maximum duration for this check
func (r *RedisCheck) Timeout() time.Duration {
	return r.timeout
}

// Interval returns how often this check should run
func (r *RedisCheck) Interval() time.Duration {
	return r.interval
}

// Critical returns true if failure of this check marks service as not ready
func (r *RedisCheck) Critical() bool {
	return r.critical
}

// KafkaCheck implements health checking for Kafka brokers
// It verifies connectivity by attempting to list topics
type KafkaCheck struct {
	name     string
	brokers  []string
	config   *sarama.Config
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewKafkaCheck creates a new Kafka health check with default settings
// The check is marked as non-critical by default since Kafka is often
// used for async operations that can tolerate temporary unavailability.
//
// Example:
//
//	brokers := []string{"localhost:9092"}
//	check := health.NewKafkaCheck("kafka", brokers)
//	hc.RegisterCheck(check)
func NewKafkaCheck(name string, brokers []string) *KafkaCheck {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Admin.Timeout = 100 * time.Millisecond

	return &KafkaCheck{
		name:     name,
		brokers:  brokers,
		config:   config,
		timeout:  200 * time.Millisecond,
		interval: 10 * time.Second,
		critical: false,
	}
}

// NewKafkaCheckWithOptions creates a Kafka health check with custom settings
func NewKafkaCheckWithOptions(name string, brokers []string, config *sarama.Config, timeout, interval time.Duration, critical bool) *KafkaCheck {
	if config == nil {
		config = sarama.NewConfig()
		config.Version = sarama.V2_6_0_0
	}
	
	return &KafkaCheck{
		name:     name,
		brokers:  brokers,
		config:   config,
		timeout:  timeout,
		interval: interval,
		critical: critical,
	}
}

// Name returns the name of this health check
func (k *KafkaCheck) Name() string {
	return k.name
}

// Check performs the Kafka health check by attempting to list topics
func (k *KafkaCheck) Check(ctx context.Context) error {
	if len(k.brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	// Create a temporary admin client
	admin, err := sarama.NewClusterAdmin(k.brokers, k.config)
	if err != nil {
		return fmt.Errorf("kafka connection failed: %w", err)
	}
	defer admin.Close()

	// List topics to verify connectivity
	// We use a channel to implement timeout since sarama doesn't support context
	type result struct {
		topics map[string]sarama.TopicDetail
		err    error
	}
	resultCh := make(chan result, 1)

	go func() {
		topics, err := admin.ListTopics()
		resultCh <- result{topics: topics, err: err}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("kafka health check timeout: %w", ctx.Err())
	case res := <-resultCh:
		if res.err != nil {
			return fmt.Errorf("kafka list topics failed: %w", res.err)
		}
		// Successfully listed topics
		return nil
	}
}

// Timeout returns the maximum duration for this check
func (k *KafkaCheck) Timeout() time.Duration {
	return k.timeout
}

// Interval returns how often this check should run
func (k *KafkaCheck) Interval() time.Duration {
	return k.interval
}

// Critical returns true if failure of this check marks service as not ready
func (k *KafkaCheck) Critical() bool {
	return k.critical
}

// HTTPCheck implements health checking for HTTP service endpoints
// It performs a GET request and validates the response status code
type HTTPCheck struct {
	name           string
	url            string
	client         *http.Client
	expectedStatus int
	timeout        time.Duration
	interval       time.Duration
	critical       bool
}

// NewHTTPCheck creates a new HTTP health check with default settings
// It expects a 200 OK response by default.
//
// Example:
//
//	check := health.NewHTTPCheck("auth-service", "http://auth-service:8080/healthz", true)
//	hc.RegisterCheck(check)
func NewHTTPCheck(name, url string, critical bool) *HTTPCheck {
	return &HTTPCheck{
		name: name,
		url:  url,
		client: &http.Client{
			Timeout: 100 * time.Millisecond,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		expectedStatus: http.StatusOK,
		timeout:        100 * time.Millisecond,
		interval:       5 * time.Second,
		critical:       critical,
	}
}

// NewHTTPCheckWithOptions creates an HTTP health check with custom settings
func NewHTTPCheckWithOptions(name, url string, client *http.Client, expectedStatus int, timeout, interval time.Duration, critical bool) *HTTPCheck {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		}
	}

	return &HTTPCheck{
		name:           name,
		url:            url,
		client:         client,
		expectedStatus: expectedStatus,
		timeout:        timeout,
		interval:       interval,
		critical:       critical,
	}
}

// Name returns the name of this health check
func (h *HTTPCheck) Name() string {
	return h.name
}

// Check performs the HTTP health check by making a GET request
func (h *HTTPCheck) Check(ctx context.Context) error {
	if h.url == "" {
		return fmt.Errorf("http check url is empty")
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Validate status code
	if resp.StatusCode != h.expectedStatus {
		return fmt.Errorf("http check failed: expected status %d, got %d", h.expectedStatus, resp.StatusCode)
	}

	return nil
}

// Timeout returns the maximum duration for this check
func (h *HTTPCheck) Timeout() time.Duration {
	return h.timeout
}

// Interval returns how often this check should run
func (h *HTTPCheck) Interval() time.Duration {
	return h.interval
}

// Critical returns true if failure of this check marks service as not ready
func (h *HTTPCheck) Critical() bool {
	return h.critical
}

// GRPCCheck implements health checking for gRPC service endpoints
// It uses the gRPC health checking protocol to verify service status
type GRPCCheck struct {
	name     string
	conn     *grpc.ClientConn
	service  string // Service name to check (empty for overall server health)
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewGRPCCheck creates a new gRPC health check with default settings
// It checks the overall server health (empty service name).
//
// Example:
//
//	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
//	check := health.NewGRPCCheck("grpc-service", conn, true)
//	hc.RegisterCheck(check)
func NewGRPCCheck(name string, conn *grpc.ClientConn, critical bool) *GRPCCheck {
	return &GRPCCheck{
		name:     name,
		conn:     conn,
		service:  "", // Empty string checks overall server health
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: critical,
	}
}

// NewGRPCCheckWithService creates a gRPC health check for a specific service
func NewGRPCCheckWithService(name string, conn *grpc.ClientConn, service string, critical bool) *GRPCCheck {
	return &GRPCCheck{
		name:     name,
		conn:     conn,
		service:  service,
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: critical,
	}
}

// NewGRPCCheckWithOptions creates a gRPC health check with custom settings
func NewGRPCCheckWithOptions(name string, conn *grpc.ClientConn, service string, timeout, interval time.Duration, critical bool) *GRPCCheck {
	return &GRPCCheck{
		name:     name,
		conn:     conn,
		service:  service,
		timeout:  timeout,
		interval: interval,
		critical: critical,
	}
}

// Name returns the name of this health check
func (g *GRPCCheck) Name() string {
	return g.name
}

// Check performs the gRPC health check using the gRPC health checking protocol
func (g *GRPCCheck) Check(ctx context.Context) error {
	if g.conn == nil {
		return fmt.Errorf("grpc connection is nil")
	}

	// Check connection state first
	state := g.conn.GetState()
	if state != connectivity.Ready && state != connectivity.Idle {
		return fmt.Errorf("grpc connection not ready: state=%s", state)
	}

	// Use gRPC health checking protocol
	client := grpc_health_v1.NewHealthClient(g.conn)

	// Perform health check
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: g.service,
	})
	if err != nil {
		return fmt.Errorf("grpc health check failed: %w", err)
	}

	// Verify service is serving
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("grpc service not serving: status=%s", resp.Status)
	}

	return nil
}

// Timeout returns the maximum duration for this check
func (g *GRPCCheck) Timeout() time.Duration {
	return g.timeout
}

// Interval returns how often this check should run
func (g *GRPCCheck) Interval() time.Duration {
	return g.interval
}

// Critical returns true if failure of this check marks service as not ready
func (g *GRPCCheck) Critical() bool {
	return g.critical
}

// NewDatabaseCheckWithCircuitBreaker creates a database health check wrapped with circuit breaker protection.
// This prevents cascading failures when the database is down.
//
// Example:
//
//	db, _ := sql.Open("mysql", dsn)
//	check := health.NewDatabaseCheckWithCircuitBreaker("database", db, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(check)
func NewDatabaseCheckWithCircuitBreaker(name string, db *sql.DB, config CircuitBreakerConfig) *CircuitBreakerCheck {
	baseCheck := NewDatabaseCheck(name, db)
	return NewCircuitBreakerCheck(baseCheck, config)
}

// NewRedisCheckWithCircuitBreaker creates a Redis health check wrapped with circuit breaker protection.
// This prevents cascading failures when Redis is down.
//
// Example:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	check := health.NewRedisCheckWithCircuitBreaker("redis", client, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(check)
func NewRedisCheckWithCircuitBreaker(name string, client redis.UniversalClient, config CircuitBreakerConfig) *CircuitBreakerCheck {
	baseCheck := NewRedisCheck(name, client)
	return NewCircuitBreakerCheck(baseCheck, config)
}

// NewKafkaCheckWithCircuitBreaker creates a Kafka health check wrapped with circuit breaker protection.
// This prevents cascading failures when Kafka is down.
//
// Example:
//
//	brokers := []string{"localhost:9092"}
//	check := health.NewKafkaCheckWithCircuitBreaker("kafka", brokers, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(check)
func NewKafkaCheckWithCircuitBreaker(name string, brokers []string, config CircuitBreakerConfig) *CircuitBreakerCheck {
	baseCheck := NewKafkaCheck(name, brokers)
	return NewCircuitBreakerCheck(baseCheck, config)
}

// NewHTTPCheckWithCircuitBreaker creates an HTTP health check wrapped with circuit breaker protection.
// This prevents cascading failures when the downstream service is down.
//
// Example:
//
//	check := health.NewHTTPCheckWithCircuitBreaker("api", "http://api:8080/health", true, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(check)
func NewHTTPCheckWithCircuitBreaker(name, url string, critical bool, config CircuitBreakerConfig) *CircuitBreakerCheck {
	baseCheck := NewHTTPCheck(name, url, critical)
	return NewCircuitBreakerCheck(baseCheck, config)
}

// NewGRPCCheckWithCircuitBreaker creates a gRPC health check wrapped with circuit breaker protection.
// This prevents cascading failures when the downstream gRPC service is down.
//
// Example:
//
//	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
//	check := health.NewGRPCCheckWithCircuitBreaker("grpc-service", conn, true, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(check)
func NewGRPCCheckWithCircuitBreaker(name string, conn *grpc.ClientConn, critical bool, config CircuitBreakerConfig) *CircuitBreakerCheck {
	baseCheck := NewGRPCCheck(name, conn, critical)
	return NewCircuitBreakerCheck(baseCheck, config)
}
