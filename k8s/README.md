# Kubernetes Configuration

This directory contains Kubernetes manifests and Kustomize configurations for deploying the Monorepo Hello/TODO Services.

## Structure

```
k8s/
├── base/                           # Base configuration
│   └── kustomization.yaml         # Base Kustomize config
├── overlays/                       # Environment-specific overlays
│   ├── development/               # Development environment
│   │   ├── kustomization.yaml
│   │   └── resources-patch.yaml
│   └── production/                # Production environment
│       ├── kustomization.yaml
│       ├── resources-patch.yaml
│       └── ingress-patch.yaml
└── README.md                      # This file
```

## Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl installed and configured
- Kustomize (built into kubectl v1.14+)
- Higress ingress controller installed (for production)

## Usage

### Deploy to Development

```bash
# Apply development configuration
kubectl apply -k k8s/overlays/development

# Verify deployment
kubectl get all -n development
kubectl get ingress -n development
```

### Deploy to Production

```bash
# Apply production configuration
kubectl apply -k k8s/overlays/production

# Verify deployment
kubectl get all -n production
kubectl get ingress -n production
```

### Deploy Base Configuration

```bash
# Apply base configuration (default namespace)
kubectl apply -k k8s/base

# Verify deployment
kubectl get all -n default
```

## Customization

### Modifying Base Configuration

Edit `k8s/base/kustomization.yaml` to change:
- Common labels and annotations
- Default replica counts
- Base image names and tags
- ConfigMap values

### Creating New Overlays

1. Create a new directory under `k8s/overlays/`:
   ```bash
   mkdir -p k8s/overlays/staging
   ```

2. Create `kustomization.yaml`:
   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   
   bases:
     - ../../base
   
   namespace: staging
   
   # Add your customizations here
   ```

3. Apply the overlay:
   ```bash
   kubectl apply -k k8s/overlays/staging
   ```

## Verification

### Check Deployment Status

```bash
# Check pods
kubectl get pods -n <namespace>

# Check services
kubectl get svc -n <namespace>

# Check ingress
kubectl get ingress -n <namespace>

# Describe resources
kubectl describe deployment hello-service -n <namespace>
kubectl describe deployment todo-service -n <namespace>
```

### View Logs

```bash
# Hello Service logs
kubectl logs -f deployment/hello-service -n <namespace>

# TODO Service logs
kubectl logs -f deployment/todo-service -n <namespace>
```

### Test Services

```bash
# Get ingress IP/hostname
INGRESS_IP=$(kubectl get ingress monorepo-ingress -n <namespace> -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test Hello Service (using grpcurl)
grpcurl -plaintext -d '{"name": "World"}' $INGRESS_IP:80 api.v1.HelloService/SayHello

# Test TODO Service
grpcurl -plaintext $INGRESS_IP:80 api.v1.TodoService/ListTodos
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n <namespace>

# View pod events
kubectl describe pod <pod-name> -n <namespace>

# Check logs
kubectl logs <pod-name> -n <namespace>
```

### Service Not Accessible

```bash
# Check service endpoints
kubectl get endpoints -n <namespace>

# Test service directly
kubectl port-forward svc/hello-service 9090:9090 -n <namespace>
```

### Ingress Issues

```bash
# Check ingress status
kubectl describe ingress monorepo-ingress -n <namespace>

# Check Higress controller logs
kubectl logs -n higress-system deployment/higress-controller
```

## Cleanup

### Delete Specific Environment

```bash
# Delete development
kubectl delete -k k8s/overlays/development

# Delete production
kubectl delete -k k8s/overlays/production
```

### Delete All Resources

```bash
# Delete all namespaces
kubectl delete namespace development production

# Or delete base
kubectl delete -k k8s/base
```

## Best Practices

1. **Never modify base directly in production** - Always use overlays
2. **Use version tags** - Avoid using `latest` tag in production
3. **Set resource limits** - Always define CPU and memory limits
4. **Enable monitoring** - Use Prometheus/Grafana for observability
5. **Implement health checks** - Configure liveness and readiness probes
6. **Use secrets** - Store sensitive data in Kubernetes secrets
7. **Enable RBAC** - Use role-based access control
8. **Backup configurations** - Keep Kustomize configs in version control

## Additional Resources

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Higress Documentation](https://higress.io/)
- [gRPC Health Checking](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
