# Kubernetes Deployment Guide

## Current Status

Kubernetes deployment in the CI/CD pipeline is **conditional**:
- ✅ Always generates Kubernetes manifests
- ✅ Uploads manifests as artifacts
- ✅ Deploys to cluster **only if KUBECONFIG is configured**
- ⏸️ Skips deployment if KUBECONFIG is not set

## How it works

The CI pipeline checks for the `KUBECONFIG` secret:

```yaml
- name: Check if KUBECONFIG is configured
  id: check-kubeconfig
  run: |
    if [ -n "${{ secrets.KUBECONFIG }}" ]; then
      echo "configured=true"
      echo "✅ KUBECONFIG is configured, will deploy to cluster"
    else
      echo "configured=false"
      echo "⚠️  KUBECONFIG is not configured, will skip deployment"
    fi
```

### When KUBECONFIG is configured:
1. ✅ Generate Kubernetes manifests
2. ✅ Upload manifests as artifacts
3. ✅ Deploy to cluster using `kubectl apply`
4. ✅ Wait for rollout completion
5. ✅ Verify deployment

### When KUBECONFIG is NOT configured:
1. ✅ Generate Kubernetes manifests
2. ✅ Upload manifests as artifacts
3. ⏸️ Skip deployment steps
4. ℹ️  Display manual deployment instructions

## How to enable Kubernetes deployment

Kubernetes deployment is **already enabled** in the CI pipeline, but requires configuration.

### Step 1: Configure KUBECONFIG Secret

1. **Get your kubeconfig**:
   ```bash
   cat ~/.kube/config | base64
   ```

2. **Add as GitHub Secret**:
   - Go to your repository settings
   - Navigate to Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `KUBECONFIG`
   - Value: The base64-encoded kubeconfig from step 1
   - Click "Add secret"

3. **That's it!** The next push to `main` will automatically deploy to your cluster.

### Step 2: Verify Deployment

After configuring KUBECONFIG, push to main branch:

```bash
git push origin main
```

Watch the CI pipeline:
- ✅ Check if KUBECONFIG is configured → Yes
- ✅ Generate K8s manifests
- ✅ Deploy to production
- ✅ Wait for rollout
- ✅ Verify deployment

### What happens without KUBECONFIG?

If KUBECONFIG is not configured:
- ✅ Manifests are still generated
- ✅ Manifests are uploaded as artifacts
- ⏸️ Deployment is skipped
- ℹ️  Instructions are displayed for manual deployment

## Kubernetes Resources Structure

```
k8s/
├── base/
│   ├── kustomization.yaml          # Base configuration
│   └── (references app k8s files)
├── overlays/
│   ├── development/
│   │   └── kustomization.yaml      # Dev environment overrides
│   └── production/
│       ├── kustomization.yaml      # Prod environment overrides
│       ├── resources-patch.yaml    # Resource limits/requests
│       └── ingress-patch.yaml      # Ingress configuration
└── README.md

apps/
├── hello-service/
│   └── k8s/
│       ├── deployment.yaml         # Hello service deployment
│       ├── service.yaml            # Hello service service
│       └── configmap.yaml          # Hello service config
└── todo-service/
    └── k8s/
        ├── deployment.yaml         # TODO service deployment
        └── service.yaml            # TODO service service
```

## Deployment Environments

### Development
- Namespace: `development`
- Replicas: 1 per service
- Resources: Minimal
- Image tag: `develop`

### Production
- Namespace: `production`
- Replicas: 3 per service
- Resources: Production-grade
- Image tag: `latest` or specific SHA

## Troubleshooting

### Issue: Kustomize path errors

**Error**: `file is not in or below base directory`

**Solution**: Ensure all referenced files in kustomization.yaml use correct relative paths from the kustomization.yaml location.

### Issue: Image pull errors

**Error**: `ImagePullBackOff`

**Solution**: 
1. Verify images are pushed to registry
2. Check image names and tags in kustomization.yaml
3. Ensure cluster has access to container registry
4. For private registries, create imagePullSecrets

### Issue: RBAC errors

**Error**: `forbidden: User cannot create resource`

**Solution**: Ensure the kubeconfig has sufficient permissions to create/update resources in the target namespace.

## Security Considerations

1. **Never commit kubeconfig to repository**
2. **Use GitHub Secrets for sensitive data**
3. **Rotate kubeconfig regularly**
4. **Use RBAC to limit deployment permissions**
5. **Enable Pod Security Standards**
6. **Use Network Policies to restrict traffic**

## Monitoring Deployment

After deployment, monitor:

```bash
# Watch pod status
kubectl get pods -n production -w

# Check logs
kubectl logs -f deployment/hello-service -n production
kubectl logs -f deployment/todo-service -n production

# Check service endpoints
kubectl get endpoints -n production

# Check ingress
kubectl describe ingress -n production
```

## Rollback

If deployment fails:

```bash
# Rollback to previous version
kubectl rollout undo deployment/hello-service -n production
kubectl rollout undo deployment/todo-service -n production

# Check rollout history
kubectl rollout history deployment/hello-service -n production
```

## Related Documentation

- [CI/CD Pipeline](.github/workflows/ci.yml)
- [Docker Deployment](./DOCKER_DEPLOYMENT.md)
- [Infrastructure Guide](./INFRASTRUCTURE.md)
- [Production Operations](./PRODUCTION_OPERATIONS.md)
