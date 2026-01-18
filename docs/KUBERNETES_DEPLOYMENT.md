# Kubernetes Deployment Guide

This guide covers deploying the Monorepo Hello/TODO Services to a Kubernetes cluster using Kustomize.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Deployment Options](#deployment-options)
- [Kustomize Configuration](#kustomize-configuration)
- [Manual Deployment](#manual-deployment)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Rollback](#rollback)

## Prerequisites

### Required Tools

- **kubectl** 1.14+ (with built-in Kustomize support)
- **Docker** (for building images)
- **Access to a Kubernetes cluster** (local or cloud)

### Verify Installation

```bash
# Check kubectl
kubectl version --client

# Check cluster connection
kubectl cluster-info

# Check Kustomize (built into kubectl)
kubectl kustomize --help
```

### Cluster Requirements

- Kubernetes 1.20+
- Sufficient resources:
  - CPU: 2+ cores
  - Memory: 4GB+ RAM
  - Storage: 10GB+

## Quick Start

### 1. Build and Push Images

```bash
# Build Docker images
make docker-build

# Tag for your registry
docker tag hello-service:latest registry.example.com/hello-service:v1.0.0
docker tag todo-service:latest registry.example.com/todo-service:v1.0.0

# Push to registry
docker push registry.example.com/hello-service:v1.0.0
docker push registry.example.com/todo-service:v1.0.0
```

### 2. Update Kustomize Configuration

Edit `k8s/overlays/production/kustomization.yaml`:

```yaml
images:
  - name: hello-service
    newName: registry.example.com/hello-service
    newTag: v1.0.0
  - name: todo-service
    newName: registry.example.com/todo-service
    newTag: v1.0.0
```

### 3. Deploy

```bash
# Deploy using script
./scripts/deploy-k8s.sh

# Or deploy manually
kubectl apply -k k8s/overlays/production
```

### 4. Verify

```bash
# Check pods
kubectl get pods -n production

# Check services
kubectl get services -n production

# Check ingress
kubectl get ingress -n production
```

## Deployment Options

### Development Overlay

For development/testing environments:

```bash
./scripts/deploy-k8s.sh --overlay development --namespace development
```

### Production Overlay

For production environments:

```bash
./scripts/deploy-k8s.sh --overlay production --namespace production
```

### Dry Run

Preview what will be deployed without applying:

```bash
./scripts/deploy-k8s.sh --dry-run
```

### Skip Image Build

If images are already built:

```bash
./scripts/deploy-k8s.sh --skip-build
```

## Kustomize Configuration

### Directory Structure

```
k8s/
├── base/                          # Base configuration
│   └── kustomization.yaml
├── overlays/
│   ├── development/               # Development overlay
│   │   ├── kustomization.yaml
│   │   └── resources-patch.yaml
│   └── production/                # Production overlay
│       ├── kustomization.yaml
│       ├── resources-patch.yaml
│       └── ingress-patch.yaml
```

### Base Configuration

The base configuration includes:
- Deployments for Hello and TODO services
- Services for internal communication
- ConfigMaps for configuration
- Basic Ingress configuration

### Overlays

**Development Overlay**:
- 1 replica per service
- Lower resource limits
- Debug logging enabled

**Production Overlay**:
- 3 replicas per service
- Higher resource limits
- Production logging
- Enhanced monitoring

### Customization

To create a new overlay:

```bash
mkdir -p k8s/overlays/staging
cd k8s/overlays/staging

cat > kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../../base

namespace: staging

replicas:
  - name: hello-service
    count: 2
  - name: todo-service
    count: 2

images:
  - name: hello-service
    newTag: staging-latest
  - name: todo-service
    newTag: staging-latest
EOF
```

## Manual Deployment

### Step-by-Step Deployment

#### 1. Create Namespace

```bash
kubectl create namespace production
```

#### 2. Apply Base Resources

```bash
kubectl apply -k k8s/base
```

#### 3. Apply Overlay

```bash
kubectl apply -k k8s/overlays/production
```

#### 4. Wait for Rollout

```bash
kubectl rollout status deployment/hello-service -n production
kubectl rollout status deployment/todo-service -n production
```

### Individual Resource Deployment

```bash
# Deploy Hello Service only
kubectl apply -f apps/hello-service/k8s/deployment.yaml
kubectl apply -f apps/hello-service/k8s/service.yaml

# Deploy TODO Service only
kubectl apply -f apps/todo-service/k8s/deployment.yaml
kubectl apply -f apps/todo-service/k8s/service.yaml
```

## Verification

### Check Pod Status

```bash
# List all pods
kubectl get pods -n production

# Describe a pod
kubectl describe pod <pod-name> -n production

# Check pod logs
kubectl logs <pod-name> -n production

# Follow logs
kubectl logs -f <pod-name> -n production
```

### Check Service Status

```bash
# List services
kubectl get services -n production

# Describe a service
kubectl describe service hello-service -n production

# Test service connectivity
kubectl run test-pod --rm -it --image=busybox -n production -- sh
# Inside the pod:
# wget -O- http://hello-service:9090
```

### Check Ingress

```bash
# List ingress
kubectl get ingress -n production

# Describe ingress
kubectl describe ingress monorepo-ingress -n production

# Get ingress URL
kubectl get ingress monorepo-ingress -n production -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

### Health Checks

```bash
# Check deployment health
kubectl get deployments -n production

# Check replica sets
kubectl get replicasets -n production

# Check events
kubectl get events -n production --sort-by='.lastTimestamp'
```

### Test Services

#### Port Forward for Testing

```bash
# Forward Hello Service
kubectl port-forward -n production svc/hello-service 9090:9090

# In another terminal, test with grpcurl
grpcurl -plaintext -d '{"name":"Kubernetes"}' localhost:9090 api.v1.HelloService/SayHello
```

```bash
# Forward TODO Service
kubectl port-forward -n production svc/todo-service 9091:9091

# Test with grpcurl
grpcurl -plaintext -d '{"title":"K8s TODO"}' localhost:9091 api.v1.TodoService/CreateTodo
```

## Troubleshooting

### Pod Not Starting

**Check pod status:**
```bash
kubectl get pods -n production
kubectl describe pod <pod-name> -n production
```

**Common issues:**

1. **ImagePullBackOff**: Image not found in registry
   ```bash
   # Check image name and tag
   kubectl describe pod <pod-name> -n production | grep Image
   
   # Verify image exists in registry
   docker pull registry.example.com/hello-service:v1.0.0
   ```

2. **CrashLoopBackOff**: Container crashes on startup
   ```bash
   # Check logs
   kubectl logs <pod-name> -n production
   
   # Check previous container logs
   kubectl logs <pod-name> -n production --previous
   ```

3. **Pending**: Insufficient resources
   ```bash
   # Check node resources
   kubectl describe nodes
   
   # Check pod resource requests
   kubectl describe pod <pod-name> -n production | grep -A 5 Requests
   ```

### Service Not Accessible

**Check service endpoints:**
```bash
kubectl get endpoints -n production
kubectl describe service hello-service -n production
```

**Test service connectivity:**
```bash
# From within the cluster
kubectl run test-pod --rm -it --image=nicolaka/netshoot -n production -- bash
# Inside the pod:
curl -v telnet://hello-service:9090
```

### Ingress Not Working

**Check Ingress controller:**
```bash
# Check if Ingress controller is running
kubectl get pods -n ingress-nginx  # or your ingress namespace

# Check Ingress resource
kubectl describe ingress monorepo-ingress -n production
```

**Common issues:**

1. **No Ingress controller installed**
   ```bash
   # Install NGINX Ingress Controller
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml
   ```

2. **Incorrect annotations**
   ```bash
   # Check Ingress annotations
   kubectl get ingress monorepo-ingress -n production -o yaml
   ```

### High Memory/CPU Usage

**Monitor resource usage:**
```bash
# Check pod resource usage
kubectl top pods -n production

# Check node resource usage
kubectl top nodes
```

**Adjust resource limits:**

Edit `k8s/overlays/production/resources-patch.yaml`:
```yaml
spec:
  template:
    spec:
      containers:
      - name: hello-service
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### Logs and Debugging

**View logs:**
```bash
# All pods with label
kubectl logs -n production -l app=hello-service --tail=100

# Stream logs
kubectl logs -n production -l app=hello-service -f

# Logs from all containers in a pod
kubectl logs -n production <pod-name> --all-containers=true
```

**Execute commands in pod:**
```bash
# Get shell access
kubectl exec -it <pod-name> -n production -- sh

# Run a command
kubectl exec <pod-name> -n production -- env
```

## Rollback

### Rollback Deployment

```bash
# View rollout history
kubectl rollout history deployment/hello-service -n production

# Rollback to previous version
kubectl rollout undo deployment/hello-service -n production

# Rollback to specific revision
kubectl rollout undo deployment/hello-service -n production --to-revision=2
```

### Delete Deployment

```bash
# Delete all resources from overlay
kubectl delete -k k8s/overlays/production

# Delete specific resources
kubectl delete deployment hello-service -n production
kubectl delete service hello-service -n production
```

## Scaling

### Manual Scaling

```bash
# Scale Hello Service
kubectl scale deployment hello-service -n production --replicas=5

# Scale TODO Service
kubectl scale deployment todo-service -n production --replicas=3
```

### Horizontal Pod Autoscaler (HPA)

Create HPA for automatic scaling:

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hello-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

Apply:
```bash
kubectl apply -f hpa.yaml
```

## Monitoring

### Built-in Monitoring

```bash
# Watch pod status
kubectl get pods -n production -w

# Monitor resource usage
watch kubectl top pods -n production
```

### Prometheus Integration

Add Prometheus annotations to deployments:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

## Security

### Network Policies

Create network policies to restrict traffic:

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-hello-to-todo
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: todo-service
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: hello-service
    ports:
    - protocol: TCP
      port: 9091
```

### Pod Security

Both services run as non-root users (configured in Dockerfiles).

### Secrets Management

For sensitive data, use Kubernetes Secrets:

```bash
# Create secret
kubectl create secret generic app-secrets \
  --from-literal=db-password=secret123 \
  -n production

# Use in deployment
env:
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: app-secrets
      key: db-password
```

## CI/CD Integration

### GitHub Actions Example

```yaml
# .github/workflows/deploy.yml
name: Deploy to Kubernetes

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build and push images
        run: |
          make docker-build
          docker push registry.example.com/hello-service:${{ github.sha }}
          docker push registry.example.com/todo-service:${{ github.sha }}
      
      - name: Set up kubectl
        uses: azure/setup-kubectl@v3
      
      - name: Deploy to K8s
        run: |
          ./scripts/deploy-k8s.sh --skip-build
```

## Best Practices

1. **Use Namespaces**: Separate environments (dev, staging, prod)
2. **Resource Limits**: Always set resource requests and limits
3. **Health Checks**: Configure liveness and readiness probes
4. **Rolling Updates**: Use rolling update strategy for zero-downtime
5. **Monitoring**: Set up logging and monitoring from day one
6. **Backup**: Regularly backup configurations and data
7. **Security**: Use RBAC, network policies, and pod security policies

## Next Steps

After successful deployment:

1. **Set up monitoring**: Integrate with Prometheus/Grafana
2. **Configure logging**: Set up centralized logging (ELK, Loki)
3. **Enable autoscaling**: Configure HPA for automatic scaling
4. **Set up alerts**: Configure alerting for critical issues
5. **Implement GitOps**: Use ArgoCD or Flux for automated deployments

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)

