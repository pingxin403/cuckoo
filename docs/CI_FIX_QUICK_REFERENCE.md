# CI Fix Quick Reference

## What Was Fixed

### 1. Security Scan ✅
- **Issue**: Permission denied for SARIF upload
- **Fix**: Added `security-events: write` permission
- **File**: `.github/workflows/ci.yml`

### 2. Kustomize Build ✅
- **Issues**: 
  - Deprecated fields warnings
  - Path outside directory errors
- **Fixes**:
  - Updated field names: `bases`→`resources`, `commonLabels`→`labels`, `patchesStrategicMerge`→`patches`
  - Created resource preparation script
  - Copy resources to base directory before build
- **Files**: 
  - `k8s/base/kustomization.yaml`
  - `k8s/overlays/production/kustomization.yaml`
  - `scripts/prepare-k8s-resources.sh` (new)

### 3. K8s Deployment ✅
- **Issue**: Fails when no cluster configured
- **Fix**: Conditional deployment based on KUBECONFIG
- **Behavior**:
  - Always generate manifests
  - Always upload artifacts
  - Deploy only if KUBECONFIG exists
  - Show instructions if skipped
- **File**: `.github/workflows/ci.yml`

## Quick Commands

```bash
# Prepare K8s resources for Kustomize
make prepare-k8s-resources

# Build manifests locally
kustomize build k8s/overlays/production > manifests.yaml

# Validate manifests
kubectl apply --dry-run=client -f manifests.yaml

# Deploy manually
kubectl apply -f manifests.yaml
```

## CI Pipeline Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Build & Test Apps                                        │
│    ├─ Detect changes                                        │
│    ├─ Build Docker images                                   │
│    └─ Run security scan ✅ (with proper permissions)        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Push Images (if main/develop)                            │
│    └─ Push to container registry                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Deploy to Kubernetes (if main)                           │
│    ├─ Check KUBECONFIG                                      │
│    ├─ Prepare K8s resources ✅ (new step)                   │
│    ├─ Generate manifests ✅ (with error handling)           │
│    ├─ Upload artifacts (always)                             │
│    ├─ Deploy (if KUBECONFIG exists) ✅ (conditional)        │
│    └─ Show instructions (if no KUBECONFIG) ✅ (helpful)     │
└─────────────────────────────────────────────────────────────┘
```

## Kustomize Field Updates

| Old (Deprecated) | New (Current) |
|-----------------|---------------|
| `bases` | `resources` |
| `commonLabels` | `labels` |
| `patchesStrategicMerge` | `patches` |

## Files Modified

- ✅ `.github/workflows/ci.yml` - Security permissions, resource prep, conditional deploy
- ✅ `k8s/base/kustomization.yaml` - Field updates, local resource paths
- ✅ `k8s/overlays/production/kustomization.yaml` - Field updates
- ✅ `scripts/prepare-k8s-resources.sh` - New resource preparation script
- ✅ `Makefile` - Added `prepare-k8s-resources` target
- ✅ `.gitignore` - Ignore generated k8s/base/*.yaml files

## Testing

```bash
# Local test
make prepare-k8s-resources
kustomize build k8s/overlays/production > test.yaml
kubectl apply --dry-run=client -f test.yaml

# Expected: No errors, 407 lines generated
```

## Enable Auto-Deployment

To enable automatic deployment to Kubernetes:

1. Get your kubeconfig:
   ```bash
   cat ~/.kube/config | base64
   ```

2. Add to GitHub repository secrets:
   - Name: `KUBECONFIG`
   - Value: `<base64 output from step 1>`

3. Push to main branch - deployment will happen automatically

## Manual Deployment

If KUBECONFIG is not configured:

1. Go to GitHub Actions run
2. Download `k8s-manifests` artifact
3. Extract and apply:
   ```bash
   kubectl apply -f k8s-manifests.yaml
   ```

## Troubleshooting

### Kustomize build fails
```bash
# Run preparation script first
make prepare-k8s-resources

# Then try build again
kustomize build k8s/overlays/production
```

### Deprecated field warnings
- All deprecated fields have been updated
- If you see warnings, check you're using the latest kustomization.yaml files

### Path errors
- Resources must be in or below the kustomization directory
- Use `prepare-k8s-resources` to copy files to correct location

## Documentation

- [Complete Fix Details](./CI_SECURITY_K8S_FIX.md)
- [Complete Summary](./CI_FIX_COMPLETE_SUMMARY.md)
- [Kubernetes Deployment Guide](./KUBERNETES_DEPLOYMENT.md)

## Status

✅ All issues resolved
✅ CI pipeline working
✅ Local development workflow matches CI
✅ Manual deployment supported
✅ Auto-deployment ready (when KUBECONFIG configured)
