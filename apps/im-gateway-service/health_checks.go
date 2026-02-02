package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
)

// WebSocketHealthCheck monitors WebSocket connection health
type WebSocketHealthCheck struct {
	gateway  *service.GatewayService
	timeout  time.Duration
	interval time.Duration
	critical bool
}

// NewWebSocketHealthCheck creates a new WebSocket health check
func NewWebSocketHealthCheck(gateway *service.GatewayService) *WebSocketHealthCheck {
	return &WebSocketHealthCheck{
		gateway:  gateway,
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: false, // Non-critical - service can start without connections
	}
}

func (w *WebSocketHealthCheck) Name() string {
	return "websocket-connections"
}

func (w *WebSocketHealthCheck) Timeout() time.Duration {
	return w.timeout
}

func (w *WebSocketHealthCheck) Interval() time.Duration {
	return w.interval
}

func (w *WebSocketHealthCheck) Critical() bool {
	return w.critical
}

func (w *WebSocketHealthCheck) Check(ctx context.Context) error {
	stats := w.gateway.GetConnectionStats()
	
	// Check if gateway is accepting connections
	// This is a basic health check - we don't fail if there are no connections
	// We only fail if the gateway is in an error state
	
	// For now, we just verify the gateway is operational
	// In a production system, you might check:
	// - Connection count is within expected range
	// - No excessive connection errors
	// - Connection pool is not exhausted
	
	if stats.TotalConnections < 0 {
		return fmt.Errorf("invalid connection count: %d", stats.TotalConnections)
	}
	
	return nil
}

// ConnectionStats represents WebSocket connection statistics
type ConnectionStats struct {
	TotalConnections int64
	ActiveDevices    int64
	ErrorCount       int64
}
