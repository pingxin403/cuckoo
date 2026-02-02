package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthzHandler returns an HTTP handler for the liveness probe endpoint.
// This endpoint checks if the service process is alive (heartbeat, memory, goroutines).
//
// Returns:
//   - 200 OK with "OK" body if the service is alive
//   - 503 Service Unavailable with "NOT ALIVE" body if the service is not alive
//
// This endpoint should be used for Kubernetes livenessProbe configuration.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/healthz", health.HealthzHandler(hc))
func HealthzHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hc.IsLive() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT ALIVE"))
			
			// Log liveness failure
			if hc.obs != nil {
				hc.obs.Logger().Error(r.Context(), "Liveness check failed",
					"service", hc.config.ServiceName,
				)
			}
		}
	}
}

// ReadyzHandler returns an HTTP handler for the readiness probe endpoint.
// This endpoint checks if the service is ready to serve traffic (all dependencies healthy).
//
// Returns:
//   - 200 OK with "READY" body if the service is ready
//   - 503 Service Unavailable with "NOT READY" body if the service is not ready
//
// This endpoint should be used for Kubernetes readinessProbe configuration.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/readyz", health.ReadyzHandler(hc))
func ReadyzHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hc.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("READY"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			
			// Log readiness failure (at debug level to avoid spam)
			if hc.obs != nil {
				hc.obs.Logger().Debug(r.Context(), "Readiness check failed",
					"service", hc.config.ServiceName,
				)
			}
		}
	}
}

// HealthHandler returns an HTTP handler that provides detailed health status information.
// This endpoint returns a JSON response with overall health status, health score,
// and individual component health details.
//
// Returns:
//   - 200 OK if status is Healthy or Degraded (service is still serving traffic)
//   - 503 Service Unavailable if status is Critical
//
// Response format:
//
//	{
//	  "status": "healthy",
//	  "service": "my-service",
//	  "timestamp": "2024-01-15T10:30:00Z",
//	  "score": 0.85,
//	  "summary": "All systems operational (4/4 healthy)",
//	  "components": {
//	    "database": {
//	      "name": "database",
//	      "status": "healthy",
//	      "last_check": "2024-01-15T10:29:58Z",
//	      "response_time_ms": 15,
//	      "error": ""
//	    },
//	    "redis": {
//	      "name": "redis",
//	      "status": "healthy",
//	      "last_check": "2024-01-15T10:29:59Z",
//	      "response_time_ms": 5,
//	      "error": ""
//	    }
//	  }
//	}
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/health", health.HealthHandler(hc))
func HealthHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hc.GetSystemHealth()
		
		// Format response
		response := formatHealthResponse(health)
		
		// Set content type
		w.Header().Set("Content-Type", "application/json")
		
		// Set status code based on health
		switch health.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // Still serving traffic
		case StatusCritical:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		// Encode and write response
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Fallback to error response
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed to encode health response"}`))
			
			if hc.obs != nil {
				hc.obs.Logger().Error(r.Context(), "Failed to encode health response",
					"service", hc.config.ServiceName,
					"error", err.Error(),
				)
			}
		}
	}
}

// HealthResponse represents the JSON response format for the health endpoint
type HealthResponse struct {
	Status     string                        `json:"status"`
	Service    string                        `json:"service"`
	Timestamp  string                        `json:"timestamp"`
	Score      float64                       `json:"score"`
	Summary    string                        `json:"summary"`
	Components map[string]*ComponentResponse `json:"components"`
}

// ComponentResponse represents the JSON response format for a component's health
type ComponentResponse struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	LastCheck      string  `json:"last_check"`
	ResponseTimeMs float64 `json:"response_time_ms"`
	Error          string  `json:"error,omitempty"`
}

// formatHealthResponse converts SystemHealth to HealthResponse for JSON serialization
func formatHealthResponse(health *SystemHealth) *HealthResponse {
	response := &HealthResponse{
		Status:     string(health.Status),
		Service:    health.Service,
		Timestamp:  health.Timestamp.Format(time.RFC3339),
		Score:      health.Score,
		Summary:    health.Summary,
		Components: make(map[string]*ComponentResponse),
	}
	
	// Format component health
	for name, component := range health.Components {
		response.Components[name] = &ComponentResponse{
			Name:           component.Name,
			Status:         string(component.Status),
			LastCheck:      component.LastCheck.Format(time.RFC3339),
			ResponseTimeMs: float64(component.ResponseTime.Milliseconds()),
			Error:          component.Error,
		}
	}
	
	return response
}

// RegisterHealthEndpoints is a convenience function that registers all health check endpoints
// on the provided ServeMux. This is the recommended way to add health endpoints to your service.
//
// Registered endpoints:
//   - GET /healthz - Liveness probe
//   - GET /readyz - Readiness probe
//   - GET /health - Detailed health status
//
// Example:
//
//	mux := http.NewServeMux()
//	health.RegisterHealthEndpoints(mux, hc)
//	
//	// Add your application endpoints
//	mux.HandleFunc("/api/users", usersHandler)
//	
//	// Start server
//	http.ListenAndServe(":8080", mux)
func RegisterHealthEndpoints(mux *http.ServeMux, hc *HealthChecker) {
	mux.HandleFunc("/healthz", HealthzHandler(hc))
	mux.HandleFunc("/readyz", ReadyzHandler(hc))
	mux.HandleFunc("/health", HealthHandler(hc))
}
