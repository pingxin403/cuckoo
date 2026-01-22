# Kubernetes Deployment Configuration

This directory contains Kubernetes manifests for deploying all services and infrastructure. You can use either **Kustomize** or **Helm** for deployment.

## Deployment Options

### Option 1: Kustomize (Recommended for GitOps)
- Native kubectl integration
- Simple overlay-based configuration
- Good for GitOps workflows (ArgoCD, Flux)

### Option 2: Helm
- Package management with versioning
- Templating and value overrides
- Good for complex deployments with many parameters

Both approaches are supported. Choose based on your team's preference and infrastructure setup.

## Directory Structure

```
deploy/
├── infra/                          # Infrastructure components
│   ├── etcd/                       # etcd cluster for IM Registry
│   ├── mysql/                      # MySQL databases (TODO)
│   ├── redis/                      # Redis instances (TODO)
│   └── kafka/                      # Kafka cluster (TODO)
├── services/                       # Application services
│   ├── hello-service/              # Hello gRPC service
│   ├── todo-service/               # TODO gRPC service
│   ├── auth-service/               # Auth service (TODO)
│   ├── user-service/               # User service (TODO)
│   ├── im-service/                 # IM routing service (TODO)
│   ├── im-gateway-service/         # IM WebSocket gateway (TODO)
│   └── ingress.yaml                # Shared ingress configuration
├── overlays/                       # Environment-specific configurations
│   ├── development/                # Development environment
│   ├── staging/                    # Staging environment
│   └── production/                 # Production environment
└── README.md                       # This file
```

## Design Principles

### Infrastructure vs Services

- **`infra/`**: Stateful infrastructure components (databases, message queues, registries)
  - Typically use StatefulSets
  - Require persistent volumes
  - Shared across multiple services
  
- **`services/`**: Stateless application services
  - Use Deployments
  - Can scale horizontally
  - Depend on infrastructure components

### Kustomize Organization

Each component has its own directory with:
- Resource manifests (YAML files)
- `kustomization.yaml` for composition

Overlays apply environment-specific patches:
- **development**: Minimal resources, single replicas
- **staging**: Medium resources, 2 replicas
- **production**: Full resources, 3+ replicas, HA configuration

## Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl installed and configured
- Kustomize (built into kubectl v1.14+)
- Storage class configured for persistent volumes
- Ingress controller (Higress/Nginx) installed

## Quick Start

### Deploy to Development

```bash
# Deploy all infrastructure and services
kubectl apply -k deploy/overlays/development

# Verify deployment
kubectl get all -n development
```

### Deploy to Production

```bash
# Deploy all infrastructure and services
kubectl apply -k deploy/overlays/production

# Verify deployment
kubectl get all -n production
kubectl get pvc -n production
```

### Deploy Individual Components

```bash
# Deploy only etcd
kubectl apply -k deploy/infra/etcd

# Deploy only hello-service
kubectl apply -k deploy/services/hello-service
```

## Infrastructure Components

We use **Bitnami Helm charts** for deploying infrastructure components (MySQL, Redis, Kafka). This provides:
- Production-ready configurations
- Built-in monitoring and metrics
- Easy version management
- Less maintenance overhead

See [deploy/infra/README.md](./infra/README.md) for detailed deployment instructions.

### Quick Deploy All Infrastructure

```bash
# Deploy all infrastructure components
./deploy/infra/deploy-all.sh

# Or manually with Helm
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Deploy etcd (custom)
kubectl apply -k deploy/infra/etcd

# Deploy MySQL
helm install im-mysql bitnami/mysql -n im-chat-system -f deploy/infra/mysql-values.yaml

# Deploy Redis
helm install im-redis bitnami/redis -n im-chat-system -f deploy/infra/redis-values.yaml

# Deploy Kafka
helm install im-kafka bitnami/kafka -n im-chat-system -f deploy/infra/kafka-values.yaml
```

### Infrastructure Overview

| Component | Type | Purpose | Helm Chart |
|-----------|------|---------|------------|
| etcd | Custom StatefulSet | User-to-gateway registry | Custom |
| MySQL | Bitnami Helm | Persistent storage (offline messages, users, groups) | bitnami/mysql |
| Redis | Bitnami Helm | Deduplication, caching, sequence generation | bitnami/redis |
| Kafka | Bitnami Helm | Message bus (group messages, offline queue) | bitnami/kafka |

## Application Services

### hello-service

**Purpose**: Example gRPC service for greeting

**Endpoints**:
- gRPC: port 9090
- Health: `/health`

### todo-service

**Purpose**: Example gRPC service for TODO management

**Endpoints**:
- gRPC: port 9091
- Health: `/health`

### Future Services (TODO)

- **auth-service** (Task 3): JWT validation and token refresh
- **user-service** (Task 4): User profiles and group membership
- **im-service** (Task 9): Message routing and delivery
- **im-gateway-service** (Task 10): WebSocket gateway for clients

## Environment Overlays

### Development

**Characteristics**:
- Minimal resource allocation
- Single replica for most services
- Fast startup and iteration
- Suitable for local Kubernetes (minikube, kind)

**Deploy**:
```bash
kubectl apply -k deploy/overlays/development
```

### Staging

**Characteristics**:
- Medium resource allocation
- 2 replicas for HA testing
- Production-like configuration
- Used for pre-production testing

**Deploy**:
```bash
kubectl apply -k deploy/overlays/staging
```

### Production

**Characteristics**:
- Full resource allocation
- 3+ replicas for high availability
- Resource limits enforced
- Monitoring and alerting enabled

**Deploy**:
```bash
kubectl apply -k deploy/overlays/production
```

## Common Operations

### View Resources

```bash
# List all resources in namespace
kubectl get all -n <namespace>

# List persistent volumes
kubectl get pvc -n <namespace>

# List ingress
kubectl get ingress -n <namespace>
```

### View Logs

```bash
# View logs for a deployment
kubectl logs -f deployment/<service-name> -n <namespace>

# View logs for a StatefulSet pod
kubectl logs -f <pod-name> -n <namespace>

# View logs for all pods with label
kubectl logs -f -l app=etcd -n <namespace>
```

### Port Forwarding

```bash
# Forward etcd client port
kubectl port-forward svc/etcd-client 2379:2379 -n <namespace>

# Forward service port
kubectl port-forward svc/hello-service 9090:9090 -n <namespace>
```

### Scaling

```bash
# Scale a deployment
kubectl scale deployment hello-service --replicas=5 -n <namespace>

# Scale a StatefulSet
kubectl scale statefulset etcd --replicas=5 -n <namespace>
```

### Debugging

```bash
# Describe a resource
kubectl describe pod <pod-name> -n <namespace>

# Get events
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# Execute command in pod
kubectl exec -it <pod-name> -n <namespace> -- /bin/sh

# Check resource usage
kubectl top pods -n <namespace>
kubectl top nodes
```

## Cleanup

### Delete Specific Environment

```bash
# Delete development
kubectl delete -k deploy/overlays/development

# Delete production
kubectl delete -k deploy/overlays/production
```

### Delete Specific Component

```bash
# Delete etcd
kubectl delete -k deploy/infra/etcd

# Delete hello-service
kubectl delete -k deploy/services/hello-service
```

### Delete Namespace (removes everything)

```bash
kubectl delete namespace development
kubectl delete namespace production
```

## Migration from k8s/ Directory

The old `k8s/` directory structure has been reorganized into `deploy/`:

**Old Structure**:
```
k8s/
├── base/
│   ├── etcd-statefulset.yaml
│   ├── hello-service-*.yaml
│   └── todo-service-*.yaml
└── overlays/
    ├── development/
    └── production/
```

**New Structure**:
```
deploy/
├── infra/
│   └── etcd/
├── services/
│   ├── hello-service/
│   └── todo-service/
└── overlays/
    ├── development/
    ├── staging/
    └── production/
```

**Benefits**:
- Clear separation of infrastructure and services
- Easier to manage individual components
- Better scalability for adding new services
- Consistent with monorepo best practices

## Best Practices

1. **Use Overlays**: Never modify base configurations directly in production
2. **Version Control**: All Kustomize configs are in git
3. **Resource Limits**: Always define CPU and memory limits
4. **Health Checks**: Configure liveness and readiness probes
5. **Secrets Management**: Use Kubernetes secrets or external secret managers
6. **Monitoring**: Enable Prometheus metrics and Grafana dashboards
7. **RBAC**: Use role-based access control
8. **Namespaces**: Isolate environments with namespaces
9. **Labels**: Use consistent labeling for resource organization
10. **Documentation**: Keep README files updated

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n <namespace>

# View pod events
kubectl describe pod <pod-name> -n <namespace>

# Check logs
kubectl logs <pod-name> -n <namespace>

# Check previous container logs (if crashed)
kubectl logs <pod-name> -n <namespace> --previous
```

### Persistent Volume Issues

```bash
# Check PVC status
kubectl get pvc -n <namespace>

# Describe PVC
kubectl describe pvc <pvc-name> -n <namespace>

# Check storage class
kubectl get storageclass
```

### Service Not Accessible

```bash
# Check service endpoints
kubectl get endpoints -n <namespace>

# Test service directly
kubectl port-forward svc/<service-name> <local-port>:<service-port> -n <namespace>

# Check ingress
kubectl describe ingress -n <namespace>
```

### etcd Cluster Issues

```bash
# Check cluster health
kubectl exec etcd-0 -n <namespace> -- etcdctl endpoint health --cluster

# Check member list
kubectl exec etcd-0 -n <namespace> -- etcdctl member list

# View etcd logs
kubectl logs etcd-0 -n <namespace>
```

## Additional Resources

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)

## Related Documentation

- [Docker Compose Setup](../docker-compose.yml) - Local development
- [IM Chat System README](../apps/im-chat-system/README.md) - Infrastructure details
- [Task List](../.kiro/specs/im-chat-system/tasks.md) - Implementation tasks
