# Observability Deployment Summary

## Overview

Complete observability stack deployment configurations have been created for both Docker Compose (local development) and Kubernetes (production).

## What Was Created

### Docker Compose Configuration

**Location**: `deploy/docker/`

1. **docker-compose.observability.yml** - Main compose file
   - OpenTelemetry Collector (2 replicas)
   - Jaeger (distributed tracing)
   - Prometheus (metrics storage)
   - Grafana (visualization)
   - Loki (log aggregation)

2. **otel-collector-config.yaml** - OpenTelemetry Collector configuration
   - OTLP receivers (gRPC and HTTP)
   - Prometheus scraping
   - Batch processing
   - Memory limiting
   - Exporters for Jaeger, Prometheus, and Loki

3. **prometheus.yml** - Prometheus configuration
   - Service discovery
   - Scrape configurations
   - Target definitions

4. **loki-config.yaml** - Loki configuration
   - Log retention (7 days)
   - Storage configuration
   - Compaction settings

5. **grafana/** - Grafana provisioning
   - `provisioning/datasources/datasources.yml` - Auto-configured datasources
   - `provisioning/dashboards/dashboards.yml` - Dashboard provisioning
   - `dashboards/service-overview.json` - Sample dashboard

6. **OBSERVABILITY.md** - Complete documentation
   - Architecture overview
   - Configuration details
   - Usage examples
   - Troubleshooting guide

7. **QUICK_START_OBSERVABILITY.md** - Quick start guide
   - 5-minute setup
   - Basic usage
   - Common queries

### Kubernetes Configuration

**Location**: `deploy/k8s/observability/`

1. **namespace.yaml** - Observability namespace

2. **otel-collector.yaml** - OpenTelemetry Collector deployment
   - ConfigMap with configuration
   - Service (ClusterIP)
   - Deployment (2 replicas)
   - Resource limits and health checks

3. **jaeger.yaml** - Jaeger deployment
   - Query service (UI)
   - Collector service
   - Deployment (all-in-one)
   - LoadBalancer for external access

4. **prometheus.yaml** - Prometheus deployment
   - ConfigMap with configuration
   - Service (ClusterIP)
   - StatefulSet (1 replica)
   - PersistentVolumeClaim (50Gi)
   - ServiceAccount and RBAC

5. **grafana.yaml** - Grafana deployment
   - ConfigMap for datasources
   - Service (LoadBalancer)
   - Deployment (1 replica)
   - PersistentVolumeClaim (10Gi)
   - Secret for admin credentials

6. **loki.yaml** - Loki deployment
   - ConfigMap with configuration
   - Service (ClusterIP)
   - StatefulSet (1 replica)
   - PersistentVolumeClaim (50Gi)

7. **README.md** - Kubernetes deployment guide
   - Architecture overview
   - Deployment instructions
   - Configuration details
   - High availability setup
   - Security considerations
   - Production checklist

### Makefile Targets

Added to `Makefile`:

```makefile
observability-up      # Start observability stack
observability-down    # Stop observability stack
observability-restart # Restart observability stack
observability-logs    # View observability logs
observability-status  # Check observability status
observability-clean   # Clean observability data
```

### Documentation Updates

Updated `deploy/docker/README.md` to include observability stack information.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Services                           │
│  (shortener-service, im-service, im-gateway-service, etc.)  │
└────────────────────┬────────────────────────────────────────┘
                     │ OTLP (gRPC: 4317, HTTP: 4318)
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
│  :16686  │  │  :9090   │  │  :3100   │
└──────────┘  └──────────┘  └──────────┘
      │              │              │
      └──────────────┴──────────────┘
                     │
                     ▼
              ┌──────────┐
              │ Grafana  │
              │   :3000  │
              └──────────┘
```

## Quick Start

### Docker Compose (Local Development)

```bash
# Start observability stack
make observability-up

# Access UIs
# Grafana:    http://localhost:3000 (admin/admin)
# Jaeger:     http://localhost:16686
# Prometheus: http://localhost:9090

# Configure your service
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_SERVICE_NAME="my-service"

# Stop observability stack
make observability-down
```

### Kubernetes (Production)

```bash
# Deploy observability stack
kubectl apply -f deploy/k8s/observability/

# Check status
kubectl get pods -n observability

# Access services (port-forward)
kubectl port-forward -n observability svc/grafana 3000:80
kubectl port-forward -n observability svc/jaeger-ui 16686:80
kubectl port-forward -n observability svc/prometheus 9090:9090
```

## Service Configuration

### Using Observability Library (Go)

```go
import "github.com/pingxin403/cuckoo/libs/observability"

config := observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "development",
    
    // OpenTelemetry configuration
    OTLPEndpoint:   "localhost:4317",  // or otel-collector.observability.svc.cluster.local:4317 in K8s
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
    EnableMetrics:  true,
}

obs, err := observability.New(config)
if err != nil {
    log.Fatal(err)
}
defer obs.Shutdown(context.Background())
```

### Using Environment Variables

```bash
# Docker Compose
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317

# Kubernetes
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.observability.svc.cluster.local:4317
```

## Components and Ports

| Component | Docker Port | K8s Service | Purpose |
|-----------|-------------|-------------|---------|
| OpenTelemetry Collector | 4317 (gRPC), 4318 (HTTP) | otel-collector:4317 | Telemetry collection |
| Jaeger UI | 16686 | jaeger-ui:80 | Trace visualization |
| Prometheus | 9090 | prometheus:9090 | Metrics storage |
| Grafana | 3000 | grafana:80 | Dashboards |
| Loki | 3100 | loki:3100 | Log aggregation |

## Features

### OpenTelemetry Collector

- ✅ OTLP gRPC and HTTP receivers
- ✅ Prometheus scraping
- ✅ Batch processing
- ✅ Memory limiting
- ✅ Resource attribute enrichment
- ✅ Multiple exporters (Jaeger, Prometheus, Loki)
- ✅ Health checks and profiling

### Jaeger

- ✅ Distributed tracing
- ✅ Service dependency graph
- ✅ Trace search and filtering
- ✅ Span details and logs
- ✅ Metrics integration with Prometheus

### Prometheus

- ✅ Metrics storage and querying
- ✅ Service discovery (Kubernetes)
- ✅ 15-day retention
- ✅ PromQL query language
- ✅ Alerting support (with Alertmanager)

### Grafana

- ✅ Pre-configured datasources (Prometheus, Jaeger, Loki)
- ✅ Sample dashboards
- ✅ Trace-to-logs correlation
- ✅ Logs-to-traces correlation
- ✅ Custom dashboard creation

### Loki

- ✅ Log aggregation
- ✅ 7-day retention
- ✅ LogQL query language
- ✅ Label-based indexing
- ✅ Compaction and retention

## Storage

### Docker Compose

- Prometheus: `prometheus-data` volume
- Grafana: `grafana-data` volume
- Loki: `loki-data` volume

### Kubernetes

- Prometheus: 50Gi PersistentVolumeClaim
- Grafana: 10Gi PersistentVolumeClaim
- Loki: 50Gi PersistentVolumeClaim

## Resource Requirements

### Docker Compose (Minimum)

- CPU: 2 cores
- Memory: 4GB
- Disk: 20GB

### Kubernetes (Production)

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| OTel Collector | 200m | 512Mi | 1000m | 2Gi |
| Jaeger | 100m | 256Mi | 500m | 1Gi |
| Prometheus | 500m | 2Gi | 2000m | 8Gi |
| Grafana | 100m | 256Mi | 500m | 1Gi |
| Loki | 200m | 512Mi | 1000m | 2Gi |

## Security Considerations

### Docker Compose

- Default passwords (change in production!)
- No TLS (local development only)
- No authentication on most services

### Kubernetes

- Change default Grafana password
- Enable TLS for all services
- Configure RBAC
- Use secrets for credentials
- Enable authentication
- Network policies

## Next Steps

1. **Start the stack**: `make observability-up`
2. **Configure services**: Update service configs to send telemetry
3. **Explore UIs**: Access Grafana, Jaeger, and Prometheus
4. **Create dashboards**: Build custom dashboards in Grafana
5. **Set up alerts**: Configure alerting rules in Prometheus
6. **Production deployment**: Deploy to Kubernetes for production

## Documentation

- [Docker Compose Guide](./docker/OBSERVABILITY.md)
- [Quick Start Guide](./docker/QUICK_START_OBSERVABILITY.md)
- [Kubernetes Guide](./k8s/observability/README.md)
- [Observability Library](../libs/observability/README.md)
- [OpenTelemetry Guide](../libs/observability/OPENTELEMETRY_GUIDE.md)
- [Migration Guide](../libs/observability/MIGRATION_GUIDE.md)

## Support

For issues or questions:
1. Check the troubleshooting sections in the documentation
2. Review OpenTelemetry Collector logs: `docker logs otel-collector`
3. Check service logs for OTLP export errors
4. Consult the [OpenTelemetry documentation](https://opentelemetry.io/docs/)

## Production Checklist

- [ ] Change default passwords
- [ ] Enable TLS for all services
- [ ] Configure persistent storage
- [ ] Set up backup and restore
- [ ] Configure resource limits
- [ ] Enable authentication
- [ ] Set up alerting
- [ ] Configure log retention
- [ ] Enable RBAC (Kubernetes)
- [ ] Monitor the observability stack itself
- [ ] Document runbooks
- [ ] Test disaster recovery

## Files Created

```
deploy/
├── docker/
│   ├── docker-compose.observability.yml
│   ├── otel-collector-config.yaml
│   ├── prometheus.yml
│   ├── loki-config.yaml
│   ├── grafana/
│   │   ├── provisioning/
│   │   │   ├── datasources/
│   │   │   │   └── datasources.yml
│   │   │   └── dashboards/
│   │   │       └── dashboards.yml
│   │   └── dashboards/
│   │       └── service-overview.json
│   ├── OBSERVABILITY.md
│   ├── QUICK_START_OBSERVABILITY.md
│   └── README.md (updated)
├── k8s/
│   └── observability/
│       ├── namespace.yaml
│       ├── otel-collector.yaml
│       ├── jaeger.yaml
│       ├── prometheus.yaml
│       ├── grafana.yaml
│       ├── loki.yaml
│       └── README.md
├── OBSERVABILITY_DEPLOYMENT_SUMMARY.md (this file)
└── Makefile (updated)
```

## Conclusion

The observability stack is now fully configured and ready to use for both local development and production deployment. The stack provides comprehensive monitoring, tracing, and logging capabilities using industry-standard open-source tools.
