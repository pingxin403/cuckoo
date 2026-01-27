# Observability Library Proposal

## Problem Statement

Currently, we are implementing Task 16.1 (Prometheus metrics) for the IM Gateway Service. However, we have 4-5 Go services in the monorepo that all need similar observability capabilities:

- im-gateway-service
- im-service
- auth-service
- user-service
- shortener-service

**Current approach issues**:
1. Each service implements its own metrics package
2. Duplicate code across services
3. Inconsistent metric names and formats
4. Difficult to maintain and update
5. No standardization across services

## Proposed Solution

Create a unified **observability library** (`libs/observability`) that provides:

1. **Metrics**: Prometheus-compatible metrics collection
2. **Tracing**: Distributed tracing (OpenTelemetry ready)
3. **Logging**: Structured logging with context propagation
4. **Middleware**: Automatic instrumentation for HTTP/gRPC/WebSocket

## Benefits

### 1. Reduce Duplication
- Single implementation shared by all services
- Estimated **80% reduction** in observability code per service
- One place to fix bugs and add features

### 2. Standardization
- Consistent metric names across services
- Standard labels and formats
- Reusable Grafana dashboards
- Predictable alert rules

### 3. Easy Integration
- Minimal code changes to adopt (< 10 lines)
- Automatic instrumentation via middleware
- Zero-boilerplate for common patterns

### 4. Better Testing
- Mock implementations for unit tests
- No need to mock external dependencies
- Faster test execution

### 5. Future-Proof
- Easy to add new features (tracing, profiling)
- Upgrade once, all services benefit
- OpenTelemetry compatible

## Architecture

```
libs/observability/
├── observability.go          # Main entry point
├── config.go                 # Configuration
├── metrics/
│   ├── metrics.go           # Interface
│   ├── prometheus.go        # Prometheus implementation
│   └── mock.go              # Mock for testing
├── tracing/
│   ├── tracing.go           # Interface
│   └── noop.go              # No-op implementation
├── logging/
│   ├── logging.go           # Interface
│   └── structured.go        # Implementation
└── middleware/
    ├── http.go              # HTTP middleware
    ├── grpc.go              # gRPC interceptors
    └── websocket.go         # WebSocket middleware
```

## Usage Example

### Before (Current Approach)

```go
// Each service implements its own metrics
package metrics

type Metrics struct {
    activeConnections atomic.Int64
    messagesDelivered atomic.Int64
    // ... 50+ lines of metric definitions
}

func (m *Metrics) Handler() http.HandlerFunc {
    // ... 200+ lines of Prometheus format code
}

// In main.go
m := metrics.NewMetrics()
m.IncrementActiveConnections()
http.HandleFunc("/metrics", m.Handler())
```

### After (Unified Library)

```go
// In main.go
obs, _ := observability.New(observability.Config{
    ServiceName:   "im-gateway",
    EnableMetrics: true,
    MetricsPort:   9090,
})
defer obs.Shutdown(context.Background())

// Metrics automatically exposed on :9090/metrics
// Use middleware for automatic instrumentation
handler := middleware.HTTPObservability(obs, mux)
http.ListenAndServe(":8080", handler)

// Custom metrics
obs.Metrics().SetGauge("active_connections", float64(count), nil)
obs.Metrics().RecordDuration("message_delivery", duration, 
    map[string]string{"success": "true"})
```

**Result**: ~90% less code, automatic instrumentation, consistent metrics

## Standard Metrics

All services automatically get:

### HTTP Services
- `http_requests_total{method, path, status}`
- `http_request_duration_seconds{method, path}` (histogram)
- `http_requests_in_flight`

### gRPC Services
- `grpc_requests_total{method, status}`
- `grpc_request_duration_seconds{method}` (histogram)

### System Metrics
- `process_cpu_seconds_total`
- `process_memory_bytes`
- `go_goroutines`

### Service-Specific
Services can add custom metrics as needed

## Implementation Plan

### Phase 1: Core Library (Week 1)
- [x] Create package structure
- [x] Define interfaces
- [x] Implement configuration
- [ ] Implement Prometheus collector
- [ ] Implement structured logger
- [ ] Add unit tests

### Phase 2: Middleware (Week 2)
- [ ] HTTP middleware
- [ ] gRPC interceptors
- [ ] WebSocket middleware
- [ ] Integration tests

### Phase 3: Migration (Week 3-4)
- [ ] Migrate im-gateway-service
- [ ] Migrate im-service
- [ ] Migrate auth-service
- [ ] Migrate user-service
- [ ] Update templates

### Phase 4: Advanced Features (Optional)
- [ ] OpenTelemetry tracing
- [ ] Distributed context propagation
- [ ] Performance optimization

## Migration Strategy

### Step 1: Implement Core Library
Complete Phase 1 implementation with tests

### Step 2: Migrate One Service (Proof of Concept)
Migrate im-gateway-service to validate approach

### Step 3: Update Templates
Update service templates to include observability by default

### Step 4: Migrate Remaining Services
Roll out to all services

### Step 5: Remove Old Code
Delete custom metrics implementations

## Comparison

| Aspect | Current Approach | Unified Library |
|--------|-----------------|-----------------|
| Code per service | ~600 lines | ~10 lines |
| Maintenance | Per service | Centralized |
| Consistency | Variable | Standardized |
| Testing | Complex | Simple (mocks) |
| Upgrades | Manual per service | Automatic |
| Onboarding | Learn per service | Learn once |

## Risks and Mitigation

### Risk 1: Breaking Changes
**Mitigation**: Maintain backward compatibility, gradual migration

### Risk 2: Performance Overhead
**Mitigation**: Benchmark and optimize, use atomic operations

### Risk 3: Service-Specific Requirements
**Mitigation**: Extensible design, allow custom metrics

### Risk 4: Learning Curve
**Mitigation**: Comprehensive documentation, examples

## Decision Points

### Should we proceed with unified library?
**Recommendation**: ✅ **YES**

**Reasons**:
1. Significant reduction in duplicate code
2. Better maintainability
3. Consistent observability across services
4. Industry best practice (similar to how Uber, Google do it)
5. Minimal migration effort

### Should we pause Task 16.1?
**Recommendation**: ✅ **YES, temporarily**

**Reasons**:
1. Avoid implementing metrics twice
2. Current metrics package will be replaced
3. Better to implement once in shared library
4. Can resume Task 16.1 after Phase 1 complete

### Timeline
- **Phase 1**: 3-5 days (core library)
- **Phase 2**: 2-3 days (middleware)
- **Phase 3**: 5-7 days (migration)
- **Total**: ~2 weeks

## Next Steps

1. **Get approval** for unified library approach
2. **Pause Task 16.1** temporarily
3. **Complete Phase 1** (core library implementation)
4. **Migrate im-gateway-service** as proof of concept
5. **Resume Task 16.1** using unified library
6. **Roll out** to remaining services

## Success Criteria

- [ ] All services use unified observability library
- [ ] Zero custom metrics implementations in services
- [ ] Consistent metric names across services
- [ ] Grafana dashboards work for all services
- [ ] Unit tests use mock implementations
- [ ] Documentation complete
- [ ] Templates updated

## Conclusion

The unified observability library is a **strategic investment** that will:
- Save significant development time
- Improve code quality and maintainability
- Provide better observability across all services
- Follow industry best practices

**Recommendation**: Proceed with implementation and migrate all services.

## References

- Prometheus best practices: https://prometheus.io/docs/practices/naming/
- OpenTelemetry: https://opentelemetry.io/
- Similar approaches:
  - Uber's observability library
  - Google's internal monitoring
  - Netflix's Atlas
