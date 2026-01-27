# Observability Stack Deployment

Complete observability stack for the monorepo, including OpenTelemetry, Jaeger, Prometheus, Grafana, and Loki.

## Components

| Component | Purpose | Ports |
|-----------|---------|-------|
| OpenTelemetry Collector | Central telemetry collection and processing | 4317 (gRPC), 4318 (HTTP), 8888 (metrics) |
| Jaeger | Distributed tracing backend and UI | 16686 (UI), 14250 (collector) |
| Prometheus | Metrics storage and querying | 9090 (UI/API) |
| Grafana | Visualization and dashboards | 3000 (UI) |
| Loki | Log aggregation | 3100 (API) |

## Quick Start

### 1. Start Observability Stack

```bash
# Start observability components
docker compose -f deploy/docker/docker-compose.observability.yml up -d

# Or use Makefile
make observability-up
```

### 2. Verify Services

```bash
# Check all services are running
docker compose -f deploy/docker/docker-compose.observability.yml ps

# Check OpenTelemetry Collector health
curl http://localhost:13133/

# Check Prometheus
curl http://localhost:9090/-/healthy

# Check Jaeger
curl http://localhost:16686/

# Check Grafana
curl http://localhost:3000/api/health

# Check Loki
curl http://localhost:3100/ready
```

### 3. Access UIs

- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686
- **Prometheus**: http://localhost:9090

### 4. Configure Services to Send Telemetry

Update your service configuration to send telemetry to the OpenTelemetry Collector:

```yaml
# Example service configuration
observability:
  service_name: "my-service"
  service_version: "1.0.0"
  environment: "development"
  
  # OpenTelemetry configuration
  otlp_endpoint: "localhost:4317"  # gRPC endpoint
  # or
  otlp_endpoint: "localhost:4318"  # HTTP endpoint
  
  # Enable telemetry signals
  enable_metrics: true
  enable_tracing: true
  enable_logging: true
  
  # Use OpenTelemetry SDKs
  use_otel_metrics: true
  use_otel_logs: true
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Services                           │
│  (shortener-service, im-service, im-gateway-service, etc.)  │
└────────────────────┬────────────────────────────────────────┘
                     │ OTLP (gRPC/HTTP)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              OpenTelemetry Collector                         │
│  - Receives: Traces, Metrics, Logs                          │
│  - Processes: Batching, Filtering, Enrichment               │
│  - Exports: To multiple backends                            │
└─────┬──────────────┬──────────────┬─────────────────────────┘
      │              │              │
      ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│  Jaeger  │  │Prometheus│  │   Loki   │
│ (Traces) │  │(Metrics) │  │  (Logs)  │
└──────────┘  └──────────┘  └──────────┘
      │              │              │
      └──────────────┴──────────────┘
                     │
                     ▼
              ┌──────────┐
              │ Grafana  │
              │(Dashboards)│
              └──────────┘
```

## Configuration Files

### OpenTelemetry Collector

**File**: `otel-collector-config.yaml`

Key configurations:
- **Receivers**: OTLP (gRPC/HTTP), Prometheus scraping
- **Processors**: Batching, memory limiting, resource attributes
- **Exporters**: Jaeger (traces), Prometheus (metrics), Loki (logs)

### Prometheus

**File**: `prometheus.yml`

Scrape targets:
- OpenTelemetry Collector metrics
- Service metrics (if exposed directly)
- Infrastructure metrics (MySQL, Redis, Kafka exporters)

### Grafana

**Directory**: `grafana/`

- **Datasources**: Auto-provisioned (Prometheus, Jaeger, Loki)
- **Dashboards**: Pre-configured service overview dashboard

### Loki

**File**: `loki-config.yaml`

Configuration:
- Log retention: 7 days
- Storage: Local filesystem (for development)
- Compaction: Enabled

## Usage Examples

### Viewing Traces in Jaeger

1. Open Jaeger UI: http://localhost:16686
2. Select service from dropdown
3. Click "Find Traces"
4. Click on a trace to see detailed span information

### Querying Metrics in Prometheus

1. Open Prometheus UI: http://localhost:9090
2. Enter PromQL query, e.g.:
   ```promql
   # Request rate
   rate(http_requests_total[5m])
   
   # Error rate
   rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
   
   # P95 latency
   histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
   ```

### Viewing Logs in Grafana

1. Open Grafana: http://localhost:3000
2. Go to "Explore"
3. Select "Loki" datasource
4. Use LogQL query, e.g.:
   ```logql
   # All logs from a service
   {service_name="my-service"}
   
   # Error logs only
   {service_name="my-service"} |= "error"
   
   # Logs with trace correlation
   {service_name="my-service"} | json | trace_id="abc123"
   ```

### Creating Dashboards in Grafana

1. Open Grafana: http://localhost:3000
2. Click "+" → "Dashboard"
3. Add panels with queries:
   - **Request Rate**: `rate(http_requests_total[5m])`
   - **Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
   - **Latency**: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`

## Integration with Services

### Go Services (using observability library)

```go
import "github.com/pingxin403/cuckoo/libs/observability"

func main() {
    // Configure observability
    config := observability.Config{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:    "development",
        
        // OpenTelemetry configuration
        OTLPEndpoint:   "localhost:4317",
        UseOTelMetrics: true,
        UseOTelLogs:    true,
        EnableTracing:  true,
        
        // Enable all signals
        EnableMetrics: true,
        LogLevel:      "info",
    }
    
    obs, err := observability.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer obs.Shutdown(context.Background())
    
    // Use observability
    obs.Metrics().IncrementCounter("requests_total", nil)
    obs.Logger().Info(ctx, "Service started")
    ctx, span := obs.Tracer().StartSpan(ctx, "operation")
    defer span.End()
}
```

### Java Services (using OpenTelemetry Java Agent)

```bash
# Download OpenTelemetry Java Agent
wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar

# Run with agent
java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.service.name=my-service \
     -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
     -jar app.jar
```

## Troubleshooting

### OpenTelemetry Collector not receiving data

```bash
# Check collector logs
docker logs otel-collector

# Verify collector is listening
netstat -an | grep 4317

# Test with curl (HTTP endpoint)
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[]}'
```

### Jaeger not showing traces

```bash
# Check Jaeger logs
docker logs jaeger

# Verify Jaeger is receiving data from collector
docker logs otel-collector | grep jaeger

# Check Jaeger storage
curl http://localhost:16686/api/services
```

### Prometheus not scraping metrics

```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check Prometheus logs
docker logs prometheus

# Verify service metrics endpoint
curl http://localhost:9090/metrics
```

### Grafana datasource connection issues

```bash
# Check Grafana logs
docker logs grafana

# Test datasource connectivity from Grafana container
docker exec grafana curl http://prometheus:9090/-/healthy
docker exec grafana curl http://jaeger-query:16686/
docker exec grafana curl http://loki:3100/ready
```

## Performance Tuning

### OpenTelemetry Collector

Adjust batch processor settings in `otel-collector-config.yaml`:

```yaml
processors:
  batch:
    timeout: 10s           # Increase for higher throughput
    send_batch_size: 1024  # Increase for higher throughput
    send_batch_max_size: 2048
```

### Prometheus

Adjust retention and scrape intervals in `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s     # Decrease for more frequent scraping
  
command:
  - '--storage.tsdb.retention.time=15d'  # Increase for longer retention
```

### Loki

Adjust ingestion limits in `loki-config.yaml`:

```yaml
limits_config:
  ingestion_rate_mb: 10        # Increase for higher log volume
  ingestion_burst_size_mb: 20
  max_streams_per_user: 10000
```

## Production Considerations

For production deployment:

1. **Use Kubernetes** (see `deploy/k8s/observability/`)
2. **Enable authentication** for all UIs
3. **Use persistent storage** for data retention
4. **Configure resource limits** appropriately
5. **Enable TLS** for all connections
6. **Set up alerting** with Alertmanager
7. **Configure backup** for Prometheus and Loki data
8. **Use distributed tracing** with sampling
9. **Monitor the observability stack** itself
10. **Implement log rotation** and retention policies

## Cleanup

```bash
# Stop observability stack
docker compose -f deploy/docker/docker-compose.observability.yml down

# Remove volumes (WARNING: deletes all data!)
docker compose -f deploy/docker/docker-compose.observability.yml down -v

# Or use Makefile
make observability-down
```

## Related Documentation

- [Observability Library](../../libs/observability/README.md)
- [OpenTelemetry Guide](../../libs/observability/OPENTELEMETRY_GUIDE.md)
- [Kubernetes Deployment](../k8s/observability/README.md)
- [Migration Guide](../../libs/observability/MIGRATION_GUIDE.md)
