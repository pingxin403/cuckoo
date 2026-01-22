# Kubernetes Deployment Migration Guide

## Overview

The Kubernetes deployment configuration has been reorganized from `k8s/` to `deploy/` directory for better organization and scalability.

## Migration Date

2026-01-22

## Changes

### Old Structure (Deprecated)

```
k8s/
├── base/
│   ├── etcd-statefulset.yaml
│   ├── hello-service-configmap.yaml
│   ├── hello-service-deployment.yaml
│   ├── hello-service-service.yaml
│   ├── todo-service-deployment.yaml
│   ├── todo-service-service.yaml
│   ├── ingress.yaml
│   └── kustomization.yaml
└── overlays/
    ├── development/
    └── production/
```

### New Structure (Current)

```
deploy/
├── infra/                          # Infrastructure components
│   ├── etcd/
│   │   ├── etcd-statefulset.yaml
│   │   ├── etcd-README.md
│   │   └── kustomization.yaml
│   ├── mysql/                      # TODO: Task 1.2
│   ├── redis/                      # TODO: Task 1.3
│   └── kafka/                      # TODO: Task 1.4
├── services/                       # Application services
│   ├── hello-service/
│   │   ├── hello-service-configmap.yaml
│   │   ├── hello-service-deployment.yaml
│   │   ├── hello-service-service.yaml
│   │   └── kustomization.yaml
│   ├── todo-service/
│   │   ├── todo-service-deployment.yaml
│   │   ├── todo-service-service.yaml
│   │   └── kustomization.yaml
│   └── ingress.yaml
├── overlays/
│   ├── development/
│   │   └── kustomization.yaml
│   ├── staging/                    # NEW
│   │   └── kustomization.yaml
│   └── production/
│       └── kustomization.yaml
└── README.md
```

## Benefits

### 1. Clear Separation of Concerns

**Infrastructure (`infra/`)**:
- Stateful components (databases, message queues, registries)
- Require persistent volumes
- Shared across multiple services
- Typically use StatefulSets

**Services (`services/`)**:
- Stateless application services
- Can scale horizontally
- Use Deployments
- Depend on infrastructure

### 2. Better Scalability

- Easy to add new infrastructure components
- Easy to add new services
- Each component is self-contained
- Independent versioning and deployment

### 3. Improved Organization

- Logical grouping by function
- Easier to navigate
- Better for monorepo structure
- Follows Kubernetes best practices

### 4. Enhanced Flexibility

- Deploy infrastructure independently
- Deploy services independently
- Mix and match components
- Environment-specific configurations

## Migration Steps

### For Existing Deployments

If you have existing deployments using the old `k8s/` structure:

#### 1. Backup Current State

```bash
# Export current resources
kubectl get all -n development -o yaml > backup-development.yaml
kubectl get all -n production -o yaml > backup-production.yaml
```

#### 2. Update Commands

**Old Commands**:
```bash
kubectl apply -k k8s/overlays/development
kubectl apply -k k8s/overlays/production
```

**New Commands**:
```bash
kubectl apply -k deploy/overlays/development
kubectl apply -k deploy/overlays/production
```

#### 3. Verify Migration

```bash
# Check resources are the same
kubectl get all -n development
kubectl get all -n production

# Verify etcd cluster
kubectl exec etcd-0 -n production -- etcdctl endpoint health --cluster

# Verify services
kubectl get svc -n production
```

#### 4. No Downtime Migration

The resource names and configurations are identical, so you can:

```bash
# Simply apply the new structure
kubectl apply -k deploy/overlays/production

# Kubernetes will update in-place (no downtime)
```

### For New Deployments

Simply use the new `deploy/` directory:

```bash
# Deploy to development
kubectl apply -k deploy/overlays/development

# Deploy to production
kubectl apply -k deploy/overlays/production
```

## Command Reference

### Old vs New Commands

| Operation | Old Command | New Command |
|-----------|-------------|-------------|
| Deploy dev | `kubectl apply -k k8s/overlays/development` | `kubectl apply -k deploy/overlays/development` |
| Deploy prod | `kubectl apply -k k8s/overlays/production` | `kubectl apply -k deploy/overlays/production` |
| Deploy etcd | `kubectl apply -k k8s/base` (partial) | `kubectl apply -k deploy/infra/etcd` |
| Deploy service | `kubectl apply -k k8s/base` (partial) | `kubectl apply -k deploy/services/hello-service` |
| Delete dev | `kubectl delete -k k8s/overlays/development` | `kubectl delete -k deploy/overlays/development` |
| Delete prod | `kubectl delete -k k8s/overlays/production` | `kubectl delete -k deploy/overlays/production` |

## CI/CD Updates

If you have CI/CD pipelines, update the paths:

### GitHub Actions Example

**Before**:
```yaml
- name: Deploy to Production
  run: kubectl apply -k k8s/overlays/production
```

**After**:
```yaml
- name: Deploy to Production
  run: kubectl apply -k deploy/overlays/production
```

### GitLab CI Example

**Before**:
```yaml
deploy:production:
  script:
    - kubectl apply -k k8s/overlays/production
```

**After**:
```yaml
deploy:production:
  script:
    - kubectl apply -k deploy/overlays/production
```

## Deprecation Timeline

- **2026-01-22**: New `deploy/` structure introduced
- **Current**: Both `k8s/` and `deploy/` coexist
- **Recommendation**: Migrate to `deploy/` immediately
- **Future**: `k8s/` directory will be removed in next major version

## Rollback Plan

If you need to rollback to the old structure:

```bash
# The old k8s/ directory is still available
kubectl apply -k k8s/overlays/production

# Or restore from backup
kubectl apply -f backup-production.yaml
```

## FAQ

### Q: Do I need to delete existing resources?

**A**: No. The resource names are identical, so Kubernetes will update in-place.

### Q: Will there be downtime?

**A**: No. Kubernetes performs rolling updates automatically.

### Q: What about persistent volumes?

**A**: PVCs remain unchanged. They are bound to the same StatefulSets.

### Q: Can I use both structures?

**A**: Technically yes, but not recommended. Choose one structure to avoid confusion.

### Q: What happens to the k8s/ directory?

**A**: It's now ignored by git (see `.gitignore`). You can safely delete it locally.

### Q: How do I add new infrastructure?

**A**: Create a new directory under `deploy/infra/` with manifests and `kustomization.yaml`.

### Q: How do I add new services?

**A**: Create a new directory under `deploy/services/` with manifests and `kustomization.yaml`.

## Examples

### Deploy Only Infrastructure

```bash
# Deploy all infrastructure
kubectl apply -k deploy/infra/etcd
kubectl apply -k deploy/infra/mysql  # TODO
kubectl apply -k deploy/infra/redis  # TODO
kubectl apply -k deploy/infra/kafka  # TODO
```

### Deploy Only Services

```bash
# Deploy all services
kubectl apply -k deploy/services/hello-service
kubectl apply -k deploy/services/todo-service
```

### Deploy Everything

```bash
# Deploy to development (all components)
kubectl apply -k deploy/overlays/development

# Deploy to production (all components)
kubectl apply -k deploy/overlays/production
```

### Selective Deployment

```bash
# Deploy only etcd to production
kubectl apply -k deploy/infra/etcd -n production

# Deploy only hello-service to development
kubectl apply -k deploy/services/hello-service -n development
```

## Support

For questions or issues:
1. Check the [deploy/README.md](../deploy/README.md)
2. Review the [IM Chat System README](../apps/im-chat-system/README.md)
3. Consult the [task list](../.kiro/specs/im-chat-system/tasks.md)

## Related Documentation

- [Deploy README](../deploy/README.md) - Complete deployment guide
- [IM Infrastructure](../apps/im-chat-system/README.md) - Infrastructure details
- [Docker Compose](../docker-compose.yml) - Local development setup
