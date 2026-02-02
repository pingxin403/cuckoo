// Package health provides a standardized health check library for Go services.
//
// This package implements production-ready health checking with proper liveness
// and readiness semantics following Kubernetes best practices. It includes:
//
//   - Liveness probes: Verify process health (heartbeat, memory, goroutines)
//   - Readiness probes: Verify dependency health (database, Redis, Kafka, etc.)
//   - Built-in health checks for common dependencies
//   - Circuit breaker pattern for graceful degradation
//   - Auto-recovery mechanisms with exponential backoff
//   - HTTP middleware for traffic control
//   - Full observability integration (metrics, logging, tracing)
//
// # Basic Usage
//
// Create a health checker and register checks:
//
//	import (
//	    "github.com/pingxin403/cuckoo/libs/health"
//	    "github.com/pingxin403/cuckoo/libs/observability"
//	)
//
//	func main() {
//	    // Initialize observability
//	    obs, _ := observability.New(observability.Config{
//	        ServiceName: "my-service",
//	    })
//	    defer obs.Shutdown(context.Background())
//
//	    // Create health checker
//	    hc := health.NewHealthChecker(health.Config{
//	        ServiceName:      "my-service",
//	        CheckInterval:    5 * time.Second,
//	        DefaultTimeout:   100 * time.Millisecond,
//	        FailureThreshold: 3,
//	    }, obs)
//
//	    // Register health checks
//	    db, _ := sql.Open("mysql", dsn)
//	    hc.RegisterCheck(health.NewDatabaseCheck("database", db))
//
//	    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	    hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))
//
//	    // Start health checking
//	    hc.Start()
//	    defer hc.Stop()
//
//	    // Setup HTTP endpoints
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/healthz", health.HealthzHandler(hc))
//	    mux.HandleFunc("/readyz", health.ReadyzHandler(hc))
//	    mux.HandleFunc("/health", health.HealthHandler(hc))
//
//	    // Wrap with readiness middleware
//	    handler := health.ReadinessMiddleware(hc)(mux)
//
//	    http.ListenAndServe(":8080", handler)
//	}
//
// # Liveness vs Readiness
//
// Liveness checks verify that the process is alive and not deadlocked:
//   - Heartbeat mechanism (detects goroutine deadlocks)
//   - Memory usage monitoring
//   - Goroutine count monitoring
//
// Readiness checks verify that the service can handle traffic:
//   - Database connectivity
//   - Redis connectivity
//   - Kafka connectivity
//   - Downstream service health
//   - Custom application-specific checks
//
// # Health Check Interface
//
// Implement the Check interface for custom health checks:
//
//	type MyCheck struct{}
//
//	func (c *MyCheck) Name() string { return "my-check" }
//	func (c *MyCheck) Check(ctx context.Context) error {
//	    // Perform health check
//	    return nil
//	}
//	func (c *MyCheck) Timeout() time.Duration { return 100 * time.Millisecond }
//	func (c *MyCheck) Interval() time.Duration { return 5 * time.Second }
//	func (c *MyCheck) Critical() bool { return true }
//
// # Circuit Breaker
//
// The library includes circuit breaker support to prevent cascading failures:
//
//	check := health.NewHTTPCheckWithCircuitBreaker(
//	    "auth-service",
//	    "http://auth-service:8080/healthz",
//	    false, // non-critical
//	)
//	hc.RegisterCheck(check)
//
// # Auto-Recovery
//
// Register recoverers to automatically reconnect to failed dependencies:
//
//	db, _ := sql.Open("mysql", dsn)
//	dbCheck := health.NewDatabaseCheck("database", db)
//	dbRecoverer := health.NewDatabaseRecoverer(dsn, &db)
//
//	hc.RegisterCheck(dbCheck)
//	hc.RegisterRecoverer("database", dbRecoverer)
//
// # Kubernetes Integration
//
// Configure Kubernetes probes to use the health endpoints:
//
//	livenessProbe:
//	  httpGet:
//	    path: /healthz
//	    port: 8080
//	  initialDelaySeconds: 10
//	  periodSeconds: 15
//	  timeoutSeconds: 5
//	  failureThreshold: 3
//
//	readinessProbe:
//	  httpGet:
//	    path: /readyz
//	    port: 8080
//	  initialDelaySeconds: 5
//	  periodSeconds: 5
//	  timeoutSeconds: 1
//	  failureThreshold: 1
//
// # Performance
//
// The library is designed for minimal overhead:
//   - Health check execution: < 200ms (all checks combined)
//   - Health status retrieval: < 1ms (lock-free atomic operations)
//   - Middleware overhead: < 100μs per request
//   - Memory overhead: < 10MB per service
//
// # Observability
//
// The library exports comprehensive metrics:
//   - health_status: Overall health status (0=critical, 1=degraded, 2=healthy)
//   - health_score: Numerical health score (0.0 to 1.0)
//   - component_status: Status per component
//   - component_response_time_seconds: Response time histogram per component
//   - health_check_failures_total: Failure counter per component
//
// All health status changes are logged with full context.
//
// # Anti-Flapping
//
// The library implements anti-flapping to prevent rapid state changes:
//   - Requires 3 consecutive failures before marking as not ready (configurable)
//   - Immediately marks as ready on first success
//   - Prevents unnecessary pod restarts in Kubernetes
//
// # Thread Safety
//
// All operations are thread-safe and can be called concurrently from multiple
// goroutines. The library uses atomic operations and minimal locking for
// high-performance concurrent access.
package health
