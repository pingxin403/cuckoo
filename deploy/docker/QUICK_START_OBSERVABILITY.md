# Quick Start: Observability Stack

Get started with the observability stack in 5 minutes.

## Prerequisites

- Docker and Docker Compose installed
- At least 4GB of free RAM
- Ports available: 3000, 4317, 4318, 9090, 16686, 3100

## Step 1: Start the Stack

```bash
# From project root
make observability-up

# Or using docker compose directly
docker compose -f deploy/docker/docker-compose.observability.yml up -d
```

Wait for all services to start (about 30 seconds).

## Step 2: Verify Services

```bash
# Check all services are running
make observability-status

# Or
docker compose -f deploy/docker/docker-compose.observability.yml ps
```

You should see all services in "Up" state.

## Step 3: Access the UIs

Open your browser and visit:

- **Grafana**: http://localhost:3000
  - Username: `admin`
  - Password: `admin`
  - (You'll be prompted to change the password on first login)

- **Jaeger**: http://localhost:16686
  - No authentication required

- **Prometheus**: http://localhost:9090
  - No authentication required

## Step 4: Configure Your Service

Update your service to send telemetry to the OpenTelemetry Collector:

### Go Service (using observability library)

```go
import "github.com/pingxin403/cuckoo/libs/observability"

func main() {
    config := observability.Config{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:    "development",
        
        // Send to OpenTelemetry Collector
        OTLPEndpoint:   "localhost:4317",
        
        // Enable OpenTelemetry
        UseOTelMetrics: true,
        UseOTelLogs:    true,
        EnableTracing:  true,
        EnableMetrics:  true,
        
        LogLevel: "info",
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
    
    // Your service logic here
}
```

### Environment Variables

Alternatively, configure via environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_SERVICE_NAME="my-service"
export OTEL_RESOURCE_ATTRIBUTES="deployment.environment=development,service.version=1.0.0"
```

## Step 5: Generate Some Telemetry

Run your service and perform some operations to generate telemetry data.

## Step 6: View Your Data

### View Traces in Jaeger

1. Go to http://localhost:16686
2. Select your service from the "Service" dropdown
3. Click "Find Traces"
4. Click on a trace to see detailed span information

### View Metrics in Prometheus

1. Go to http://localhost:9090
2. Click "Graph"
3. Enter a query, for example:
   ```promql
   rate(http_requests_total[5m])
   ```
4. Click "Execute"

### View Logs in Grafana

1. Go to http://localhost:3000
2. Click "Explore" (compass icon on the left)
3. Select "Loki" from the datasource dropdown
4. Enter a LogQL query:
   ```logql
   {service_name="my-service"}
   ```
5. Click "Run query"

### Create a Dashboard in Grafana

1. Go to http://localhost:3000
2. Click "+" → "Dashboard"
3. Click "Add visualization"
4. Select "Prometheus" as datasource
5. Enter a query:
   ```promql
   rate(http_requests_total[5m])
   ```
6. Click "Apply"
7. Click "Save dashboard"

## Common Queries

### Prometheus (Metrics)

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Memory usage
process_resident_memory_bytes

# CPU usage
rate(process_cpu_seconds_total[5m])
```

### LogQL (Logs)

```logql
# All logs from a service
{service_name="my-service"}

# Error logs only
{service_name="my-service"} |= "error"

# Logs with specific trace ID
{service_name="my-service"} | json | trace_id="abc123"

# Count errors per minute
sum(count_over_time({service_name="my-service"} |= "error" [1m]))
```

## Troubleshooting

### Services not starting

```bash
# Check logs
make observability-logs

# Or
docker compose -f deploy/docker/docker-compose.observability.yml logs
```

### Can't access UIs

```bash
# Check if ports are in use
lsof -i :3000  # Grafana
lsof -i :16686 # Jaeger
lsof -i :9090  # Prometheus

# Check if services are running
docker ps | grep -E "grafana|jaeger|prometheus|otel-collector|loki"
```

### No data showing up

1. **Check OpenTelemetry Collector is receiving data**:
   ```bash
   # Check collector logs
   docker logs otel-collector
   
   # Should see messages like:
   # "Traces received"
   # "Metrics received"
   # "Logs received"
   ```

2. **Verify your service is sending to correct endpoint**:
   ```bash
   # Test OTLP endpoint
   curl -v http://localhost:4318/v1/traces
   
   # Should return 405 Method Not Allowed (endpoint exists)
   ```

3. **Check service logs for errors**:
   ```bash
   # Look for OTLP export errors in your service logs
   ```

## Next Steps

- Read the [full documentation](./OBSERVABILITY.md)
- Explore [Grafana dashboards](./grafana/dashboards/)
- Learn about [OpenTelemetry](https://opentelemetry.io/docs/)
- Set up [alerting](https://prometheus.io/docs/alerting/latest/overview/)
- Configure [log retention](https://grafana.com/docs/loki/latest/operations/storage/retention/)

## Cleanup

```bash
# Stop observability stack
make observability-down

# Remove all data (WARNING: Deletes all metrics, traces, and logs!)
make observability-clean
```

## Getting Help

- Check the [troubleshooting guide](./OBSERVABILITY.md#troubleshooting)
- Review [OpenTelemetry Collector logs](https://opentelemetry.io/docs/collector/troubleshooting/)
- Ask in the team chat or create an issue

## Production Deployment

For production deployment, see:
- [Kubernetes deployment guide](../k8s/observability/README.md)
- [Production checklist](../k8s/observability/README.md#production-checklist)
