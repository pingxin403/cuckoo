# Kubernetes Observability Stack

Production-ready observability stack for Kubernetes deployment.

## Components

| Component | Type | Replicas | Storage |
|-----------|------|----------|---------|
| OpenTelemetry Collector | Deployment | 2 | - |
| Jaeger | Deployment | 1 | - |
| Prometheus | StatefulSet | 1 | 50Gi |
| Grafana | Deployment | 1 | 10Gi |
| Loki | StatefulSet | 1 | 50Gi |

## Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- StorageClass available (default: `standard`)
- LoadBalancer support (for external access)

## Quick Start

### 1. Create Namespace

```bash
kubectl apply -f namespace.yaml
```

### 2. Deploy Observability Stack

```bash
# Deploy all components
kubectl apply -f otel-collector.yaml
kubectl apply -f jaeger.yaml
kubectl apply -f prometheus.yaml
kubectl apply -f loki.yaml
kubectl apply -f grafana.yaml

# Or deploy all at once
kubectl apply -f .
```

### 3. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n observability

# Check services
kubectl get svc -n observability

# Check persistent volumes
kubectl get pvc -n observability
```

### 4. Access Services

```bash
# Get LoadBalancer IPs
kubectl get svc -n observability

# Port-forward for local access
kubectl port-forward -n observability svc/grafana 3000:80
kubectl port-forward -n observability svc/jaeger-ui 16686:80
kubectl port-forward -n observability svc/prometheus 9090:9090
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Application Namespace                      │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │ │
│  │  │ Service1 │  │ Service2 │  │ Service3 │            │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘            │ │
│  └───────┼─────────────┼─────────────┼──────────────────┘ │
│          │             │             │                      │
│          │ OTLP        │ OTLP        │ OTLP                │
│          ▼             ▼             ▼                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │           Observability Namespace                       │ │
│  │                                                          │ │
│  │  ┌──────────────────────────────────────────────────┐  │ │
│  │  │      OpenTelemetry Collector (2 replicas)        │  │ │
│  │  │  - Service: otel-collector.observability         │  │ │
│  │  │  - Port: 4317 (gRPC), 4318 (HTTP)                │  │ │
│  │  └────┬──────────────┬──────────────┬────────────────┘  │ │
│  │       │              │              │                    │ │
│  │       ▼              ▼              ▼                    │ │
│  │  ┌────────┐    ┌──────────┐   ┌────────┐              │ │
│  │  │ Jaeger │    │Prometheus│   │  Loki  │              │ │
│  │  │(Traces)│    │(Metrics) │   │ (Logs) │              │ │
│  │  └────────┘    └──────────┘   └────────┘              │ │
│  │       │              │              │                    │ │
│  │       └──────────────┴──────────────┘                   │ │
│  │                      │                                   │ │
│  │                      ▼                                   │ │
│  │               ┌──────────┐                              │ │
│  │               │ Grafana  │                              │ │
│  │               │   (UI)   │                              │ │
│  │               └──────────┘                              │ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

### Service Discovery

Prometheus is configured with Kubernetes service discovery to automatically discover and scrape metrics from:

1. **Pods with annotations**:
   ```yaml
   annotations:
     prometheus.io/scrape: "true"
     prometheus.io/port: "9090"
     prometheus.io/path: "/metrics"
   ```

2. **OpenTelemetry Collector**: Automatically scraped
3. **Jaeger**: Automatically scraped

### Application Configuration

Configure your applications to send telemetry to the OpenTelemetry Collector:

```yaml
# Example Kubernetes deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        - name: my-service
          env:
            # OpenTelemetry configuration
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://otel-collector.observability.svc.cluster.local:4317"
            - name: OTEL_SERVICE_NAME
              value: "my-service"
            - name: OTEL_RESOURCE_ATTRIBUTES
              value: "deployment.environment=production,service.namespace=default"
```

## Storage Configuration

### Prometheus Storage

Default: 50Gi PersistentVolumeClaim

To change:
```yaml
# In prometheus.yaml
spec:
  resources:
    requests:
      storage: 100Gi  # Increase as needed
```

### Loki Storage

Default: 50Gi PersistentVolumeClaim

To change:
```yaml
# In loki.yaml
spec:
  resources:
    requests:
      storage: 100Gi  # Increase as needed
```

### Grafana Storage

Default: 10Gi PersistentVolumeClaim

To change:
```yaml
# In grafana.yaml
spec:
  resources:
    requests:
      storage: 20Gi  # Increase as needed
```

## Resource Limits

### OpenTelemetry Collector

```yaml
resources:
  requests:
    cpu: 200m
    memory: 512Mi
  limits:
    cpu: 1000m
    memory: 2Gi
```

### Prometheus

```yaml
resources:
  requests:
    cpu: 500m
    memory: 2Gi
  limits:
    cpu: 2000m
    memory: 8Gi
```

### Jaeger

```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 1Gi
```

### Grafana

```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 1Gi
```

### Loki

```yaml
resources:
  requests:
    cpu: 200m
    memory: 512Mi
  limits:
    cpu: 1000m
    memory: 2Gi
```

## High Availability

### OpenTelemetry Collector

Already configured with 2 replicas for HA.

To increase:
```yaml
# In otel-collector.yaml
spec:
  replicas: 3  # Increase as needed
```

### Prometheus

For HA Prometheus, consider using:
- Thanos (for long-term storage and global view)
- Prometheus Operator (for easier management)
- VictoriaMetrics (as Prometheus alternative)

### Jaeger

For production, use Jaeger with external storage:
- Elasticsearch
- Cassandra
- Kafka

See [Jaeger documentation](https://www.jaegertracing.io/docs/latest/deployment/) for details.

## Security

### Change Default Passwords

```bash
# Update Grafana admin password
kubectl create secret generic grafana-admin \
  --from-literal=username=admin \
  --from-literal=password=YOUR_SECURE_PASSWORD \
  --namespace=observability \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Enable TLS

For production, enable TLS for all services:

1. **Create TLS certificates**:
   ```bash
   # Using cert-manager
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

2. **Configure Ingress with TLS**:
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: grafana
     namespace: observability
   spec:
     tls:
       - hosts:
           - grafana.example.com
         secretName: grafana-tls
     rules:
       - host: grafana.example.com
         http:
           paths:
             - path: /
               pathType: Prefix
               backend:
                 service:
                   name: grafana
                   port:
                     number: 80
   ```

## Monitoring the Observability Stack

The observability stack should monitor itself:

1. **Prometheus scrapes its own metrics**
2. **Grafana has health checks**
3. **OpenTelemetry Collector exports its own metrics**

Create alerts for:
- High memory usage
- Disk space running low
- Service unavailability
- High error rates in collectors

## Backup and Restore

### Prometheus Backup

```bash
# Create snapshot
kubectl exec -n observability prometheus-0 -- \
  curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot

# Copy snapshot
kubectl cp observability/prometheus-0:/prometheus/snapshots/SNAPSHOT_NAME ./backup/
```

### Grafana Backup

```bash
# Backup dashboards and datasources
kubectl exec -n observability deployment/grafana -- \
  tar czf - /var/lib/grafana > grafana-backup.tar.gz
```

### Loki Backup

```bash
# Backup Loki data
kubectl exec -n observability loki-0 -- \
  tar czf - /loki > loki-backup.tar.gz
```

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl get pods -n observability

# Check pod logs
kubectl logs -n observability <pod-name>

# Describe pod for events
kubectl describe pod -n observability <pod-name>
```

### Storage issues

```bash
# Check PVC status
kubectl get pvc -n observability

# Check PV status
kubectl get pv

# Describe PVC for events
kubectl describe pvc -n observability <pvc-name>
```

### Service connectivity issues

```bash
# Test service DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- \
  nslookup otel-collector.observability.svc.cluster.local

# Test service connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://otel-collector.observability.svc.cluster.local:13133/
```

### High resource usage

```bash
# Check resource usage
kubectl top pods -n observability

# Check resource limits
kubectl describe pod -n observability <pod-name> | grep -A 5 Limits
```

## Cleanup

```bash
# Delete all observability resources
kubectl delete -f .

# Delete namespace (WARNING: deletes all data!)
kubectl delete namespace observability
```

## Production Checklist

- [ ] Change default passwords
- [ ] Enable TLS for all services
- [ ] Configure persistent storage with appropriate size
- [ ] Set up backup and restore procedures
- [ ] Configure resource limits based on load
- [ ] Enable authentication for all UIs
- [ ] Set up alerting with Alertmanager
- [ ] Configure log retention policies
- [ ] Enable RBAC for service accounts
- [ ] Set up monitoring for the observability stack itself
- [ ] Document runbooks for common issues
- [ ] Test disaster recovery procedures

## Related Documentation

- [Docker Compose Deployment](../../docker/OBSERVABILITY.md)
- [Observability Library](../../../libs/observability/README.md)
- [OpenTelemetry Guide](../../../libs/observability/OPENTELEMETRY_GUIDE.md)
- [Migration Guide](../../../libs/observability/MIGRATION_GUIDE.md)
