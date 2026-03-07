# Geographic Routing for IM Gateway Service

This package provides intelligent geographic routing capabilities integrated into the IM Gateway Service for multi-region active-active architecture.

## Overview

The routing package enables the IM Gateway to intelligently route user connections to the most appropriate region based on:
- Geographic location
- User preferences
- System health
- Custom routing rules

## Integration with IM Gateway

### Gateway Service Integration

```go
// In apps/im-gateway-service/service/gateway.go
type Gateway struct {
    // ... existing fields
    geoRouter *routing.GeoRouter
}

func (g *Gateway) handleConnection(conn *websocket.Conn) {
    // Determine target region for this connection
    decision := g.geoRouter.RouteRequest(conn.Request())
    
    // Route to appropriate region
    g.routeToRegion(decision.TargetRegion, conn)
}
```

### Configuration

```yaml
# In apps/im-gateway-service/config/config.yaml
routing:
  enabled: true
  port: 8080
  health_check_interval: 30s
  default_region: "region-a"
  enable_geo_ip: true
```

## Features

- **Header-based Routing**: Direct region targeting via HTTP headers
- **Geographic Routing**: Automatic routing based on client IP
- **User-based Routing**: Consistent routing based on user ID
- **Health-aware Routing**: Automatic failover to healthy regions
- **WebSocket Support**: Routing for WebSocket connections

## API Endpoints

### POST /route
Get routing decision for a connection request.

### GET /regions
List all configured regions and their status.

### GET /health
Health check endpoint.

## Requirements Satisfied

- ✅ Requirement 3.1: Geographic routing
- ✅ Requirement 3.2: WebSocket session management
- ✅ Requirement 4.1: Health-aware routing
- ✅ Requirement 4.2: Automatic failover

## Next Steps

1. Integrate with gateway service WebSocket handler
2. Add region-aware session management
3. Implement cross-region failover logic
4. Add routing metrics to observability
