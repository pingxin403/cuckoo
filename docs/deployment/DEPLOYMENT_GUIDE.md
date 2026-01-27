# Deployment Guide

Complete guide for deploying the monorepo services to different environments.

## Directory Structure

```
deploy/
├── docker/                          # Docker Compose for local development
│   ├── docker-compose.infra.yml     # Infrastructure components
│   ├── docker-compose.services.yml  # Application services
│   └── README.md
├── k8s/                             # Kubernetes for production
│   ├── infra/                       # Infrastructure (Helm charts)
│   ├── services/                    # Application services (Kustomize)
│   ├── overlays/                    # Environment-specific configs
│   └── README.md
└── DEPLOYMENT_GUIDE.md              # This file
```

## Deployment Options

### 1. Local Development (Docker Compose)

**Use Case**: Local development and testing

**Pros**:
- Fast startup
- Easy to debug
- No Kubernetes required
- Matches production architecture

**Cons**:
- Not suitable for production
- Limited scalability
- No HA features

**Quick Start**:
```bash
# Start everything (recommended - use Makefile)
make dev-up

# Or use docker compose directly
docker compose -f deploy/docker/docker-compose.infra.yml \
               -f deploy/docker/docker-compose.services.yml up -d

# Or start infrastructure only
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# Or start services only
docker compose -f deploy/docker/docker-compose.services.yml up -d
```

See [deploy/docker/README.md](./docker/README.md) for details.

### 2. Kubernetes Production (Helm + Kustomize)

**Use Case**: Production, staging, and testing environments

**Pros**:
- Production-ready
- High availability
- Auto-scaling
- Rolling updates
- Health checks and self-healing

**Cons**:
- Requires Kubernetes cluster
- More complex setup
- Higher resource requirements

**Quick Start**:
```bash
# Deploy infrastructure (Helm)
./deploy/k8s/infra/deploy-all.sh

# Deploy services (Kustomize)
kubectl apply -k deploy/k8s/overlays/production
```

See [deploy/k8s/README.md](./k8s/README.md) for details.

## Environment Comparison

| Feature | Docker Compose | Kubernetes Dev | Kubernetes Prod |
|---------|----------------|----------------|-----------------|
| Infrastructure | Single node | Bitnami Helm charts | Bitnami Helm charts |
| Services | Docker containers | Kubernetes Pods (1 replica) | Kubernetes Pods (3+ replicas) |
| Load Balancing | Docker network | ClusterIP | LoadBalancer/Ingress |
| Persistence | Docker volumes | PVC (default storage) | PVC (production storage class) |
| Monitoring | Logs only | Optional | Prometheus + Grafana |
| Secrets | Environment variables | Kubernetes Secrets | External secrets manager |
| Resource Limits | None | Minimal | Production-grade |
| High Availability | No | No | Yes |

## Deployment Workflows

### Development Workflow

```bash
# 1. Start local infrastructure
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# 2. Develop and test locally
# ... make code changes ...

# 3. Rebuild and restart service
docker compose -f deploy/docker/docker-compose.services.yml build hello-service
docker compose -f deploy/docker/docker-compose.services.yml up -d hello-service

# 4. Run tests
make test

# 5. Commit changes
git add .
git commit -m "feat: add new feature"
git push
```

### Staging Deployment

```bash
# 1. Deploy infrastructure (one-time setup)
kubectl create namespace staging
./deploy/k8s/infra/deploy-all.sh

# 2. Deploy services
kubectl apply -k deploy/k8s/overlays/staging

# 3. Verify deployment
kubectl get pods -n staging
kubectl logs -f deployment/hello-service -n staging

# 4. Run smoke tests
./scripts/test-services.sh staging
```

### Production Deployment

```bash
# 1. Deploy infrastructure (one-time setup)
kubectl create namespace production
./deploy/k8s/infra/deploy-all.sh

# 2. Deploy services with rolling update
kubectl apply -k deploy/k8s/overlays/production

# 3. Monitor rollout
kubectl rollout status deployment/hello-service -n production

# 4. Verify health
kubectl get pods -n production
curl https://api.example.com/health

# 5. Rollback if needed
kubectl rollout undo deployment/hello-service -n production
```

## Infrastructure Components

### Shortener Service Stack

- **MySQL**: Persistent storage for URL mappings
- **Redis**: Cache for frequently accessed URLs

### IM Chat System Stack

- **etcd**: Distributed registry for user-to-gateway mappings
- **MySQL**: Persistent storage for offline messages, users, groups
- **Redis**: Deduplication, caching, sequence generation
- **Kafka**: Message bus for group messages and offline queue

## Service Dependencies

```
hello-service (no dependencies)
  ↓
todo-service (depends on hello-service)

shortener-service (depends on mysql, redis)
  ↓
envoy-gateway (depends on shortener-service)

im-gateway-service (depends on etcd, redis, kafka)
  ↓
im-service (depends on mysql, redis, kafka)
  ↓
offline-worker (depends on mysql, redis, kafka)
```

## Monitoring and Observability

### Docker Compose

```bash
# View logs
docker compose logs -f [service-name]

# Check health
docker compose ps

# Resource usage
docker stats
```

### Kubernetes

```bash
# View logs
kubectl logs -f deployment/[service-name] -n [namespace]

# Check health
kubectl get pods -n [namespace]
kubectl describe pod [pod-name] -n [namespace]

# Resource usage
kubectl top pods -n [namespace]
kubectl top nodes

# Metrics (if Prometheus is installed)
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

```bash
# Find process using port
lsof -i :9090

# Kill process
kill -9 [PID]
```

#### 2. Database Connection Failed

```bash
# Docker Compose
docker exec shortener-mysql mysql -u shortener_user -pshortener_password -e "SELECT 1"

# Kubernetes
kubectl exec -it mysql-0 -n [namespace] -- mysql -u shortener_user -pshortener_password -e "SELECT 1"
```

#### 3. Service Not Starting

```bash
# Docker Compose
docker compose logs [service-name]

# Kubernetes
kubectl logs [pod-name] -n [namespace]
kubectl describe pod [pod-name] -n [namespace]
```

#### 4. Kafka Topics Not Created

```bash
# Docker Compose
docker compose -f deploy/docker/docker-compose.infra.yml restart kafka-init

# Kubernetes
kubectl delete job kafka-topic-init -n [namespace]
kubectl apply -f deploy/k8s/infra/kafka/topic-init-job.yaml
```

## Security Best Practices

### Development

- Use default passwords (already in config)
- No TLS required
- Local network only

### Production

- **Never use default passwords**
- Use Kubernetes Secrets or external secrets manager (Vault, AWS Secrets Manager)
- Enable TLS for all connections
- Use network policies to restrict traffic
- Enable RBAC
- Regular security audits
- Keep images updated

### Secrets Management

```bash
# Create secrets in Kubernetes
kubectl create secret generic mysql-secret \
  --from-literal=root-password=<strong-password> \
  --from-literal=user-password=<strong-password> \
  -n production

# Use external secrets (recommended)
# - AWS Secrets Manager
# - HashiCorp Vault
# - Google Secret Manager
```

## Backup and Disaster Recovery

### Docker Compose

```bash
# Backup volumes
docker run --rm -v shortener-mysql-data:/data -v $(pwd):/backup alpine tar czf /backup/mysql-backup.tar.gz /data

# Restore volumes
docker run --rm -v shortener-mysql-data:/data -v $(pwd):/backup alpine tar xzf /backup/mysql-backup.tar.gz -C /
```

### Kubernetes

```bash
# Backup using Velero (recommended)
velero backup create my-backup --include-namespaces production

# Restore
velero restore create --from-backup my-backup

# Manual database backup
kubectl exec mysql-0 -n production -- mysqldump -u root -p<password> --all-databases > backup.sql
```

## Performance Tuning

### Docker Compose

- Increase Docker resources (CPU, Memory)
- Use volume mounts for faster I/O
- Enable BuildKit for faster builds

### Kubernetes

- Set appropriate resource requests and limits
- Use HPA (Horizontal Pod Autoscaler)
- Use node affinity for optimal placement
- Enable cluster autoscaling
- Use PodDisruptionBudgets for availability

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Deploy to Kubernetes
        run: |
          kubectl apply -k deploy/k8s/overlays/production
```

See [.github/workflows/](../.github/workflows/) for complete CI/CD setup.

## Cost Optimization

### Development

- Use Docker Compose (free)
- Use local Kubernetes (minikube, kind)

### Production

- Use managed services (RDS, ElastiCache, MSK)
- Enable autoscaling
- Use spot instances for non-critical workloads
- Set up cost monitoring and alerts
- Use resource quotas and limits

## Migration Guide

### From Docker Compose to Kubernetes

1. Export environment variables to ConfigMaps/Secrets
2. Convert volumes to PersistentVolumeClaims
3. Update service discovery (DNS names)
4. Add health checks and readiness probes
5. Configure resource limits
6. Set up monitoring and logging
7. Test thoroughly in staging

## Support and Resources

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [Bitnami Charts](https://github.com/bitnami/charts)

## Related Documentation

- [Docker Deployment](./docker/README.md)
- [Kubernetes Deployment](./k8s/README.md)
- [IM Service](../apps/im-service/README.md)
- [Shortener Service](../apps/shortener-service/README.md)
