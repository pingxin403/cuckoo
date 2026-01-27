# Deployment Quick Reference

Quick reference for deployment commands after the refactoring.

## Docker Compose (Local Development)

### Start Services

```bash
# Start everything (infrastructure + services)
make dev-up
# or
docker compose up -d

# Start infrastructure only
make infra-up
# or
docker compose -f deploy/docker/docker-compose.infra.yml up -d

# Start services only
make services-up
# or
docker compose -f deploy/docker/docker-compose.services.yml up -d
```

### Stop Services

```bash
# Stop everything
make dev-down
# or
docker compose down

# Stop infrastructure
make infra-down
# or
docker compose -f deploy/docker/docker-compose.infra.yml down

# Stop services
make services-down
# or
docker compose -f deploy/docker/docker-compose.services.yml down
```

### Restart Services

```bash
# Restart services (keeps infrastructure running)
make dev-restart
# or
docker compose -f deploy/docker/docker-compose.services.yml restart

# Restart specific service
docker compose -f deploy/docker/docker-compose.services.yml restart hello-service
```

### View Logs

```bash
# View all logs
docker compose logs -f

# View infrastructure logs
docker compose -f deploy/docker/docker-compose.infra.yml logs -f

# View service logs
docker compose -f deploy/docker/docker-compose.services.yml logs -f

# View specific service logs
docker compose logs -f hello-service
```

### Check Status

```bash
# Check all services
docker compose ps

# Check infrastructure
docker compose -f deploy/docker/docker-compose.infra.yml ps

# Check services
docker compose -f deploy/docker/docker-compose.services.yml ps
```

## Kubernetes (Production)

### Validate Manifests

```bash
# Validate all manifests
make k8s-validate

# Validate development
kubectl apply --dry-run=client -k deploy/k8s/overlays/development

# Validate production
kubectl apply --dry-run=client -k deploy/k8s/overlays/production
```

### Deploy Infrastructure

```bash
# Deploy all infrastructure (Helm charts)
make k8s-infra-deploy
# or
./deploy/k8s/infra/deploy-all.sh

# Deploy individual components
helm install mysql bitnami/mysql -f deploy/k8s/infra/mysql-values.yaml
helm install redis bitnami/redis -f deploy/k8s/infra/redis-values.yaml
helm install kafka bitnami/kafka -f deploy/k8s/infra/kafka-values.yaml
kubectl apply -k deploy/k8s/infra/etcd/
```

### Deploy Services

```bash
# Deploy to development
make k8s-deploy-dev
# or
kubectl apply -k deploy/k8s/overlays/development

# Deploy to production
make k8s-deploy-prod
# or
kubectl apply -k deploy/k8s/overlays/production
```

### Check Deployment Status

```bash
# Get all pods
kubectl get pods -n production

# Get all services
kubectl get svc -n production

# Get all deployments
kubectl get deployments -n production

# Check rollout status
kubectl rollout status deployment/hello-service -n production
```

### View Logs

```bash
# View pod logs
kubectl logs -f deployment/hello-service -n production

# View logs from all pods in deployment
kubectl logs -f -l app=hello-service -n production

# View previous logs (if pod crashed)
kubectl logs --previous deployment/hello-service -n production
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/hello-service -n production

# Rollback to specific revision
kubectl rollout undo deployment/hello-service --to-revision=2 -n production

# View rollout history
kubectl rollout history deployment/hello-service -n production
```

### Scale Services

```bash
# Scale manually
kubectl scale deployment/hello-service --replicas=5 -n production

# Enable autoscaling
kubectl autoscale deployment/hello-service --min=3 --max=10 --cpu-percent=80 -n production
```

## Infrastructure Management

### Docker Compose (Local Development)

```bash
# Start all infrastructure (MySQL, Redis, etcd, Kafka)
make infra-up

# Stop infrastructure
make infra-down

# Check infrastructure status
make infra-status

# View infrastructure logs
make infra-logs

# Clean infrastructure data (WARNING: Deletes all data!)
make infra-clean

# Start all services (infrastructure + applications)
make dev-up

# Stop all services
make dev-down

# Restart application services (keep infrastructure running)
make dev-restart
```

### IM Chat System Database Migrations (Liquibase)

```bash
# Apply migrations
docker compose -f deploy/docker/docker-compose.infra.yml up im-liquibase

# Rollback last changeset
docker compose -f deploy/docker/docker-compose.infra.yml run --rm im-liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://mysql:3306/im_chat?useSSL=false \
  --username=im_service \
  --password=im_service_password \
  rollback-count 1

# Show migration status
docker compose -f deploy/docker/docker-compose.infra.yml run --rm im-liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://mysql:3306/im_chat?useSSL=false \
  --username=im_service \
  --password=im_service_password \
  status --verbose
```

## Application Management

### Build

```bash
# Build all changed apps
make build

# Build specific app
make build APP=hello

# Build Docker images
make docker-build APP=shortener
```

### Test

```bash
# Test all changed apps
make test

# Test specific app
make test APP=todo

# Test with coverage
make test-coverage APP=shortener
```

### Lint

```bash
# Lint all changed apps
make lint

# Lint specific app
make lint APP=hello

# Auto-fix lint errors
make lint-fix APP=shortener
```

## CI/CD

### Trigger CI

```bash
# Push to trigger CI
git push origin feature-branch

# CI will:
# 1. Detect changed apps
# 2. Test infrastructure
# 3. Validate Kubernetes manifests
# 4. Build and test apps
# 5. Build Docker images
# 6. Push images (on push to main/develop)
# 7. Deploy to Kubernetes (on push to main)
```

### Manual Deployment

```bash
# Download k8s-manifests.yaml from CI artifacts
# Then apply manually:
kubectl apply -f k8s-manifests.yaml
```

## Troubleshooting

### Port Already in Use

```bash
# Find process using port
lsof -i :9090

# Kill process
kill -9 [PID]
```

### Database Connection Failed

```bash
# Docker Compose
docker exec shortener-mysql mysql -u shortener_user -pshortener_password -e "SELECT 1"

# Kubernetes
kubectl exec -it mysql-0 -n production -- mysql -u shortener_user -pshortener_password -e "SELECT 1"
```

### Service Not Starting

```bash
# Docker Compose
docker compose logs [service-name]

# Kubernetes
kubectl logs [pod-name] -n production
kubectl describe pod [pod-name] -n production
```

### Clean Everything

```bash
# Docker Compose
docker compose down -v
docker system prune -a

# Kubernetes
kubectl delete namespace production
kubectl delete namespace development
```

## Common Workflows

### Daily Development

```bash
# 1. Start infrastructure (once)
make infra-up

# 2. Start services
make services-up

# 3. Make code changes
# ... edit code ...

# 4. Restart service
docker compose -f deploy/docker/docker-compose.services.yml restart hello-service

# 5. View logs
docker compose logs -f hello-service

# 6. Run tests
make test APP=hello

# 7. Stop services (keep infrastructure running)
make services-down
```

### Deploy to Production

```bash
# 1. Validate manifests
make k8s-validate

# 2. Deploy infrastructure (one-time)
make k8s-infra-deploy

# 3. Deploy services
make k8s-deploy-prod

# 4. Check status
kubectl get pods -n production

# 5. View logs
kubectl logs -f deployment/hello-service -n production

# 6. Rollback if needed
kubectl rollout undo deployment/hello-service -n production
```

### Update Infrastructure

```bash
# Docker Compose
# 1. Edit deploy/docker/docker-compose.infra.yml
# 2. Restart infrastructure
make infra-down
make infra-up

# Kubernetes
# 1. Edit deploy/k8s/infra/*-values.yaml
# 2. Upgrade Helm release
helm upgrade mysql bitnami/mysql -f deploy/k8s/infra/mysql-values.yaml
```

## Environment Variables

### Docker Compose

Environment variables are defined in docker-compose files:
- `deploy/docker/docker-compose.infra.yml`
- `deploy/docker/docker-compose.services.yml`

### Kubernetes

Environment variables are defined in:
- ConfigMaps: `deploy/k8s/services/*/configmap.yaml`
- Secrets: `deploy/k8s/services/*/secret.yaml`
- Overlays: `deploy/k8s/overlays/*/`

## File Locations

### Docker Compose
- Infrastructure: `deploy/docker/docker-compose.infra.yml`
- Services: `deploy/docker/docker-compose.services.yml`
- Root: `docker-compose.yml` (includes both)

### Kubernetes
- Infrastructure: `deploy/k8s/infra/`
- Services: `deploy/k8s/services/`
- Overlays: `deploy/k8s/overlays/`

### Documentation
- Deployment Guide: `deploy/DEPLOYMENT_GUIDE.md`
- Docker Guide: `deploy/docker/README.md`
- Kubernetes Guide: `deploy/k8s/README.md`
- Refactoring Summary: `deploy/REFACTORING_SUMMARY.md`

## Related Documentation

- [Deployment Guide](../deploy/DEPLOYMENT_GUIDE.md) - Complete deployment guide
- [Docker README](../deploy/docker/README.md) - Docker Compose usage
- [Kubernetes README](../deploy/k8s/README.md) - Kubernetes deployment
- [Refactoring Summary](../deploy/REFACTORING_SUMMARY.md) - Refactoring details
- [Phase 1 Complete](./DEPLOYMENT_REFACTORING_PHASE1_COMPLETE.md) - Phase 1 summary
