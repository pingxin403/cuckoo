# Geographic Routing for Multi-Region Active-Active Architecture

This package provides intelligent geographic routing capabilities for the multi-region active-active IM chat system. It enables automatic routing of user requests to the most appropriate region based on various criteria including geographic location, user preferences, system health, and custom rules.

## Features

- **Header-based Routing**: Direct region targeting via HTTP headers
- **Geographic Routing**: Automatic routing based on client IP geolocation
- **User-based Routing**: Consistent routing based on user ID hashing
- **Health-aware Routing**: Automatic failover to healthy regions
- **Custom Routing Rules**: Flexible rule engine for complex routing logic
- **Real-time Health Checking**: Continuous monitoring of region health
- **RESTful API**: Complete HTTP API for routing decisions and management

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client        │    │   Geo Router    │    │   Target        │
│   Request       │───▶│   Decision      │───▶│   Region        │
│                 │    │   Engine        │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Health        │
                       │   Checker       │
                       └─────────────────┘
```

## Components

### GeoRouter

The main routing component that:
- Evaluates routing rules in priority order
- Performs health checks on target regions
- Provides HTTP API for routing decisions
- Maintains routing statistics and metrics

### RegionInfo

Contains information about each region:
- Endpoint URLs and health status
- Geographic location data
- Priority and weight for load balancing
- Performance metrics (latency, etc.)

### RoutingRule

Defines routing logic:
- Conditions for rule matching
- Target region assignment
- Priority ordering
- Enable/disable flags

### HealthChecker

Monitors region health:
- Periodic health check requests
- Latency measurement
- Failure detection and recovery

## Usage

### Basic Setup

```go
package main

import (
    "github.com/cuckoo-org/cuckoo/routing"
    "log"
)

func main() {
    // Create configuration
    config := routing.DefaultGeoRouterConfig()
    config.Port = 8080
    config.HealthCheckInterval = 30 * time.Second
    
    // Create router
    logger := log.New(os.Stdout, "[GeoRouter] ", log.LstdFlags)
    router := routing.NewGeoRouter("region-a", config, logger)
    
    // Start router
    if err := router.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Making Routing Decisions

```go
// Get routing decision for an HTTP request
decision := router.RouteRequest(httpRequest)

fmt.Printf("Route to: %s\n", decision.TargetRegion)
fmt.Printf("Reason: %s\n", decision.Reason)
fmt.Printf("Confidence: %.2f\n", decision.Confidence)
```

### Running the Example

```bash
# Start the geo router example
go run routing/example_integration.go

# The router will be available at:
# http://localhost:8081
```

## API Endpoints

### POST /route
Get routing decision for a request.

**Request Headers:**
- `X-Target-Region`: Force routing to specific region
- `X-Forwarded-For`: Client IP for geo-routing
- `X-User-ID`: User identifier for consistent routing

**Response:**
```json
{
  "target_region": "region-a",
  "reason": "Geographic routing - North China",
  "confidence": 1.0,
  "alternatives": ["region-b"],
  "decision_time": "2024-01-15T10:30:00Z",
  "processing_time": "2.5ms"
}
```

### GET /regions
List all configured regions and their status.

**Response:**
```json
{
  "region-a": {
    "id": "region-a",
    "name": "Region A (Primary)",
    "endpoint": "http://im-service-a:8080",
    "healthy": true,
    "latency": "45ms",
    "geo_location": {
      "country": "CN",
      "region": "North",
      "city": "Beijing"
    }
  }
}
```

### GET /regions/{regionId}
Get detailed information about a specific region.

### GET /rules
List all routing rules.

**Response:**
```json
[
  {
    "id": "header-region-override",
    "name": "Header-based Region Override",
    "priority": 1,
    "conditions": [
      {
        "type": "header",
        "key": "X-Target-Region",
        "operator": "equals",
        "value": "*"
      }
    ],
    "enabled": true
  }
]
```

### GET /status
Get router status and statistics.

### GET /health
Health check endpoint for the router itself.

## Routing Rules

### Default Rules (in priority order)

1. **Header Override** (Priority 1)
   - Condition: `X-Target-Region` header present
   - Action: Route to specified region if healthy

2. **Geographic Routing** (Priority 2)
   - Condition: Client IP indicates North China
   - Action: Route to `region-a`
   - Condition: Client IP indicates South China
   - Action: Route to `region-b`

3. **User ID Hash** (Priority 3)
   - Condition: User ID hash mod 100 < 50
   - Action: Route to `region-a`
   - Condition: User ID hash mod 100 >= 50
   - Action: Route to `region-b`

4. **Default Fallback** (Priority 100)
   - Condition: Always matches
   - Action: Route to default region

### Custom Rules

You can add custom routing rules for:
- Premium user routing
- API version-based routing
- Load balancing algorithms
- Time-based routing
- Request type routing

## Health Checking

The router continuously monitors region health:

- **Check Interval**: 30 seconds (configurable)
- **Timeout**: 5 seconds (configurable)
- **Endpoint**: `{region_endpoint}/health`
- **Success Criteria**: HTTP 2xx response

### Failover Behavior

When a target region is unhealthy:
1. Find healthy alternative regions
2. Route to the highest priority healthy region
3. Include alternatives in the routing decision
4. Update routing reason to indicate failover

## Configuration

### GeoRouterConfig

```go
type GeoRouterConfig struct {
    Port                int           // HTTP server port
    HealthCheckInterval time.Duration // Health check frequency
    HealthCheckTimeout  time.Duration // Health check timeout
    DefaultRegion       string        // Fallback region
    EnableGeoIP         bool          // Enable IP geolocation
    LogRequests         bool          // Log routing decisions
}
```

### Default Configuration

```go
config := routing.DefaultGeoRouterConfig()
// Port: 8080
// HealthCheckInterval: 30s
// HealthCheckTimeout: 5s
// DefaultRegion: "region-a"
// EnableGeoIP: true
// LogRequests: true
```

## Integration Patterns

### 1. API Gateway Integration

```go
// Use geo router in API gateway
func apiHandler(w http.ResponseWriter, r *http.Request) {
    decision := router.RouteRequest(r)
    
    // Proxy to target region
    proxyToRegion(decision.TargetRegion, w, r)
}
```

### 2. Load Balancer Integration

```go
// Use for intelligent load balancing
func selectBackend(r *http.Request) string {
    decision := router.RouteRequest(r)
    return getRegionEndpoint(decision.TargetRegion)
}
```

### 3. Client-side Routing

```go
// Provide routing hints to clients
func routingHint(w http.ResponseWriter, r *http.Request) {
    decision := router.RouteRequest(r)
    
    w.Header().Set("X-Recommended-Region", decision.TargetRegion)
    w.Header().Set("X-Routing-Confidence", fmt.Sprintf("%.2f", decision.Confidence))
}
```

## Performance

### Benchmarks

```
BenchmarkRouteRequest-8           1000000    1.2 μs/op
BenchmarkRouteRequestWithGeo-8     800000    1.5 μs/op
```

### Optimization Tips

1. **Rule Ordering**: Place most specific rules first
2. **Caching**: Cache routing decisions for repeated requests
3. **Health Checks**: Adjust check intervals based on requirements
4. **Logging**: Disable request logging in high-traffic scenarios

## Monitoring

### Metrics

The router exposes metrics for:
- Routing decision latency
- Health check success/failure rates
- Region selection distribution
- Failover events

### Integration with Prometheus

```go
// Example Prometheus metrics integration
var (
    routingDecisions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "geo_router_decisions_total",
            Help: "Total number of routing decisions",
        },
        []string{"target_region", "reason"},
    )
)
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./routing/...

# Run with coverage
go test -cover ./routing/...

# Run benchmarks
go test -bench=. ./routing/...
```

### Integration Tests

The package includes comprehensive tests for:
- Routing rule evaluation
- Health checking logic
- HTTP API endpoints
- Failover scenarios
- Performance benchmarks

## Security Considerations

1. **Input Validation**: All routing inputs are validated
2. **Header Injection**: Prevent malicious header manipulation
3. **Rate Limiting**: Consider rate limiting for routing API
4. **Authentication**: Add authentication for management endpoints
5. **HTTPS**: Use HTTPS in production environments

## Production Deployment

### Recommended Setup

1. **Multiple Instances**: Deploy router in each region
2. **Load Balancing**: Use load balancer in front of routers
3. **Health Monitoring**: Monitor router health and metrics
4. **Logging**: Centralized logging for routing decisions
5. **Alerting**: Alert on routing failures or health issues

### Configuration Management

```yaml
# Example Kubernetes ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: geo-router-config
data:
  config.json: |
    {
      "port": 8080,
      "health_check_interval": "30s",
      "health_check_timeout": "5s",
      "default_region": "region-a",
      "enable_geo_ip": true,
      "log_requests": false
    }
```

## Troubleshooting

### Common Issues

1. **All Regions Unhealthy**
   - Check network connectivity
   - Verify health check endpoints
   - Review timeout settings

2. **Inconsistent Routing**
   - Check rule priorities
   - Verify condition logic
   - Review user ID hashing

3. **High Latency**
   - Optimize rule evaluation
   - Reduce health check frequency
   - Enable request caching

### Debug Mode

Enable detailed logging:
```go
config.LogRequests = true
```

Check routing decision details:
```bash
curl -H "X-User-ID: test123" http://localhost:8080/route
```