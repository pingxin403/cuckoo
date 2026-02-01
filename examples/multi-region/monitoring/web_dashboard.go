package monitoring

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/multi-region/sync"
)

// WebDashboard provides a simple web interface for monitoring multi-region metrics
type WebDashboard struct {
	port             int
	hlcClock         *hlc.HLC
	conflictResolver *sync.ConflictResolver
	messageSyncer    *sync.MessageSyncer
	server           *http.Server
}

// DashboardMetrics aggregates all metrics for display
type DashboardMetrics struct {
	Timestamp       time.Time            `json:"timestamp"`
	RegionID        string               `json:"region_id"`
	HLCMetrics      HLCMetrics           `json:"hlc_metrics"`
	ConflictMetrics sync.ConflictMetrics `json:"conflict_metrics"`
	SyncMetrics     SyncMetrics          `json:"sync_metrics"`
	SystemHealth    SystemHealth         `json:"system_health"`
}

// HLCMetrics represents HLC clock metrics
type HLCMetrics struct {
	PhysicalTime int64  `json:"physical_time_ms"`
	LogicalTime  int64  `json:"logical_time"`
	RegionID     string `json:"region_id"`
	NodeID       string `json:"node_id"`
	Sequence     int64  `json:"sequence"`
}

// SyncMetrics represents message synchronization metrics
type SyncMetrics struct {
	AsyncSyncCount int64   `json:"async_sync_count"`
	SyncSyncCount  int64   `json:"sync_sync_count"`
	ConflictCount  int64   `json:"conflict_count"`
	ErrorCount     int64   `json:"error_count"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	SyncRate       float64 `json:"sync_rate"` // Messages per second
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status          string    `json:"status"` // "healthy", "degraded", "critical"
	LastHealthCheck time.Time `json:"last_health_check"`
	ConflictRate    float64   `json:"conflict_rate"`
	SyncLatencyP99  float64   `json:"sync_latency_p99_ms"`
	ErrorRate       float64   `json:"error_rate"`
}

// NewWebDashboard creates a new web dashboard
func NewWebDashboard(port int, hlcClock *hlc.HLC, conflictResolver *sync.ConflictResolver, messageSyncer *sync.MessageSyncer) *WebDashboard {
	return &WebDashboard{
		port:             port,
		hlcClock:         hlcClock,
		conflictResolver: conflictResolver,
		messageSyncer:    messageSyncer,
	}
}

// Start starts the web dashboard server
func (wd *WebDashboard) Start() error {
	mux := http.NewServeMux()

	// Dashboard HTML page
	mux.HandleFunc("/", wd.handleDashboard)

	// Metrics API endpoint
	mux.HandleFunc("/api/metrics", wd.handleMetrics)

	// Health check endpoint
	mux.HandleFunc("/health", wd.handleHealth)

	// Static assets
	mux.HandleFunc("/static/", wd.handleStatic)

	wd.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", wd.port),
		Handler: mux,
	}

	fmt.Printf("Starting web dashboard on port %d\n", wd.port)
	return wd.server.ListenAndServe()
}

// Stop stops the web dashboard server
func (wd *WebDashboard) Stop() error {
	if wd.server != nil {
		return wd.server.Close()
	}
	return nil
}

// handleDashboard serves the main dashboard HTML page
func (wd *WebDashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Multi-Region Active-Active Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .header { background-color: #2c3e50; color: white; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .metric-card { background: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .metric-title { font-size: 18px; font-weight: bold; margin-bottom: 15px; color: #2c3e50; }
        .metric-value { font-size: 24px; font-weight: bold; margin: 10px 0; }
        .metric-label { font-size: 14px; color: #666; margin-bottom: 5px; }
        .status-healthy { color: #27ae60; }
        .status-warning { color: #f39c12; }
        .status-critical { color: #e74c3c; }
        .refresh-info { text-align: center; margin: 20px 0; color: #666; }
        .timestamp { font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Multi-Region Active-Active Dashboard</h1>
        <p>Real-time monitoring for cross-region synchronization</p>
    </div>
    
    <div class="refresh-info">
        <p>Auto-refreshing every 5 seconds | Last updated: <span id="timestamp">Loading...</span></p>
    </div>
    
    <div class="metrics-grid" id="metrics-container">
        <div class="metric-card">
            <div class="metric-title">Loading metrics...</div>
        </div>
    </div>

    <script>
        function updateMetrics() {
            fetch('/api/metrics')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('timestamp').textContent = new Date(data.timestamp).toLocaleString();
                    renderMetrics(data);
                })
                .catch(error => {
                    console.error('Error fetching metrics:', error);
                    document.getElementById('metrics-container').innerHTML = 
                        '<div class="metric-card"><div class="metric-title status-critical">Error loading metrics</div></div>';
                });
        }

        function renderMetrics(data) {
            const container = document.getElementById('metrics-container');
            
            const healthStatus = getHealthStatusClass(data.system_health.status);
            
            container.innerHTML = ` + "`" + `
                <div class="metric-card">
                    <div class="metric-title">System Health</div>
                    <div class="metric-value ${healthStatus}">${data.system_health.status.toUpperCase()}</div>
                    <div class="metric-label">Region: ${data.region_id}</div>
                    <div class="metric-label">Conflict Rate: ${(data.system_health.conflict_rate * 100).toFixed(4)}%</div>
                    <div class="metric-label">Error Rate: ${(data.system_health.error_rate * 100).toFixed(2)}%</div>
                </div>
                
                <div class="metric-card">
                    <div class="metric-title">HLC Clock Status</div>
                    <div class="metric-value">${data.hlc_metrics.physical_time_ms}</div>
                    <div class="metric-label">Physical Time (ms)</div>
                    <div class="metric-label">Logical Time: ${data.hlc_metrics.logical_time}</div>
                    <div class="metric-label">Sequence: ${data.hlc_metrics.sequence}</div>
                    <div class="metric-label">Node: ${data.hlc_metrics.node_id}</div>
                </div>
                
                <div class="metric-card">
                    <div class="metric-title">Message Synchronization</div>
                    <div class="metric-value">${data.sync_metrics.async_sync_count}</div>
                    <div class="metric-label">Async Syncs</div>
                    <div class="metric-label">Sync Syncs: ${data.sync_metrics.sync_sync_count}</div>
                    <div class="metric-label">Avg Latency: ${data.sync_metrics.avg_latency_ms.toFixed(2)}ms</div>
                    <div class="metric-label">Sync Rate: ${data.sync_metrics.sync_rate.toFixed(2)}/sec</div>
                </div>
                
                <div class="metric-card">
                    <div class="metric-title">Conflict Resolution</div>
                    <div class="metric-value">${data.conflict_metrics.total_conflicts}</div>
                    <div class="metric-label">Total Conflicts</div>
                    <div class="metric-label">Local Wins: ${data.conflict_metrics.local_wins}</div>
                    <div class="metric-label">Remote Wins: ${data.conflict_metrics.remote_wins}</div>
                    <div class="metric-label">Avg Resolution: ${data.conflict_metrics.avg_resolution_time_us.toFixed(2)}μs</div>
                </div>
                
                <div class="metric-card">
                    <div class="metric-title">Performance Metrics</div>
                    <div class="metric-value">${data.system_health.sync_latency_p99_ms.toFixed(2)}ms</div>
                    <div class="metric-label">Sync Latency P99</div>
                    <div class="metric-label">Error Count: ${data.sync_metrics.error_count}</div>
                    <div class="metric-label">Last Check: ${new Date(data.system_health.last_health_check).toLocaleTimeString()}</div>
                </div>
            ` + "`" + `;
        }

        function getHealthStatusClass(status) {
            switch(status) {
                case 'healthy': return 'status-healthy';
                case 'degraded': return 'status-warning';
                case 'critical': return 'status-critical';
                default: return '';
            }
        }

        // Update metrics immediately and then every 5 seconds
        updateMetrics();
        setInterval(updateMetrics, 5000);
    </script>
</body>
</html>
`

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, nil)
}

// handleMetrics serves the metrics API endpoint
func (wd *WebDashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := wd.collectMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(metrics)
}

// handleHealth serves the health check endpoint
func (wd *WebDashboard) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"region":    wd.hlcClock.GetRegionID(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleStatic serves static assets (placeholder)
func (wd *WebDashboard) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// collectMetrics gathers all metrics from various components
func (wd *WebDashboard) collectMetrics() DashboardMetrics {
	now := time.Now()

	// Collect HLC metrics
	hlcMetrics := HLCMetrics{
		PhysicalTime: wd.hlcClock.GetPhysicalTime(),
		LogicalTime:  wd.hlcClock.GetLogicalTime(),
		RegionID:     wd.hlcClock.GetRegionID(),
		NodeID:       wd.hlcClock.GetNodeID(),
		Sequence:     wd.hlcClock.GetSequence(),
	}

	// Collect conflict metrics
	conflictMetrics := wd.conflictResolver.GetMetrics()

	// Collect sync metrics
	syncMetrics := wd.collectSyncMetrics()

	// Calculate system health
	systemHealth := wd.calculateSystemHealth(conflictMetrics, syncMetrics)

	return DashboardMetrics{
		Timestamp:       now,
		RegionID:        wd.hlcClock.GetRegionID(),
		HLCMetrics:      hlcMetrics,
		ConflictMetrics: conflictMetrics,
		SyncMetrics:     syncMetrics,
		SystemHealth:    systemHealth,
	}
}

// collectSyncMetrics gathers synchronization metrics
func (wd *WebDashboard) collectSyncMetrics() SyncMetrics {
	// Get metrics from message syncer
	asyncCount, syncCount, conflictCount, errorCount := wd.messageSyncer.GetCounts()
	avgLatency := wd.messageSyncer.GetAverageLatency()
	syncRate := wd.messageSyncer.GetSyncRate()

	return SyncMetrics{
		AsyncSyncCount: asyncCount,
		SyncSyncCount:  syncCount,
		ConflictCount:  conflictCount,
		ErrorCount:     errorCount,
		AvgLatencyMs:   avgLatency,
		SyncRate:       syncRate,
	}
}

// calculateSystemHealth determines overall system health based on metrics
func (wd *WebDashboard) calculateSystemHealth(conflictMetrics sync.ConflictMetrics, syncMetrics SyncMetrics) SystemHealth {
	status := "healthy"

	// Check conflict rate (critical if > 0.1%)
	if conflictMetrics.ConflictRate > 0.001 {
		status = "critical"
	} else if conflictMetrics.ConflictRate > 0.0005 {
		status = "degraded"
	}

	// Check sync latency (warning if > 500ms)
	if syncMetrics.AvgLatencyMs > 500 {
		if status == "healthy" {
			status = "degraded"
		}
	}

	// Check error rate (critical if > 1%)
	errorRate := float64(syncMetrics.ErrorCount) / float64(syncMetrics.AsyncSyncCount+syncMetrics.SyncSyncCount+1)
	if errorRate > 0.01 {
		status = "critical"
	} else if errorRate > 0.005 {
		if status == "healthy" {
			status = "degraded"
		}
	}

	return SystemHealth{
		Status:          status,
		LastHealthCheck: time.Now(),
		ConflictRate:    conflictMetrics.ConflictRate,
		SyncLatencyP99:  syncMetrics.AvgLatencyMs * 1.5, // Approximate P99
		ErrorRate:       errorRate,
	}
}
