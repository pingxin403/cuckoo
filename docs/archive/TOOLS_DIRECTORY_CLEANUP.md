# Tools Directory Cleanup - 2026-01-25

## Summary

Cleaned up and reorganized the `tools/` directory by moving configurations to appropriate locations in the `deploy/` directory structure.

## Changes Made

### 1. Moved Envoy Configurations

**From**: `tools/envoy/`  
**To**: `deploy/docker/`

- `tools/envoy/envoy-docker.yaml` → `deploy/docker/envoy-config.yaml`
- `tools/envoy/envoy-local.yaml` → `deploy/docker/envoy-local-config.yaml`

**Reason**: Envoy configurations are deployment-related and belong with Docker Compose files.

### 2. Removed Duplicate Higress Configurations

**Removed**: `tools/higress/` and `tools/k8s/ingress.yaml`  
**Kept**: `deploy/k8s/services/higress/higress-routes.yaml`

**Reason**: The Higress configuration in `deploy/k8s/services/higress/` is more comprehensive and already covers all functionality from the tools directory:
- Supports all services (Hello, TODO, Shortener, Web)
- Includes advanced features (rate limiting, circuit breaker)
- Uses Higress-specific CRDs (HttpRoute, WasmPlugin)
- Properly integrated with Kustomize overlays

### 3. Deleted Tools Directory

**Removed**: Entire `tools/` directory

**Reason**: All configurations have been moved to appropriate locations in `deploy/`.

### 4. Updated References

Updated all documentation references from `tools/` to `deploy/`:

**Files Updated** (20 files):
- `.kiro/specs/monorepo-hello-todo/design.md`
- `.kiro/specs/monorepo-hello-todo/tasks.md`
- `README.md`
- `apps/im-service/README.md`
- `apps/shortener-service/GATEWAY_VERIFICATION.md`
- `apps/web/DEPLOYMENT.md`
- `deploy/docker/README.md`
- `docs/architecture/ARCHITECTURE.md`
- `docs/architecture/INFRASTRUCTURE.md`
- `docs/archive/CHECKLIST.md`
- `docs/archive/LOCAL_SETUP_VERIFICATION.md`
- `docs/archive/app-specific/GATEWAY_SETUP_SUMMARY.md`
- `docs/archive/app-specific/MVP_COMPLETION_SUMMARY.md`
- `docs/archive/completions/DEPLOYMENT_SUMMARY.md`
- `docs/archive/completions/K8S_CLEANUP_AND_SHORTENER_ADDITION.md`
- `docs/archive/fixes/CI_SECURITY_K8S_FIX.md`
- `docs/archive/migrations/K8S_INFRA_HELM_MIGRATION.md`
- `docs/process/governance.md`

### 5. Updated Docker Compose

**File**: `deploy/docker/docker-compose.services.yml`

**Change**:
```yaml
# Before
volumes:
  - ../../tools/envoy/envoy-docker.yaml:/etc/envoy/envoy.yaml:ro

# After
volumes:
  - ./envoy-config.yaml:/etc/envoy/envoy.yaml:ro
```

## New Directory Structure

```
deploy/
├── docker/
│   ├── docker-compose.infra.yml
│   ├── docker-compose.services.yml
│   ├── docker-compose.observability.yml
│   ├── envoy-config.yaml              # NEW: Envoy for Docker
│   ├── envoy-local-config.yaml        # NEW: Envoy for local dev
│   ├── prometheus.yml
│   ├── loki-config.yaml
│   └── ...
└── k8s/
    ├── infra/
    │   ├── etcd/
    │   ├── kafka/
    │   ├── mysql/
    │   ├── redis/
    │   └── higress-values.yaml
    ├── services/
    │   ├── higress/
    │   │   ├── higress-routes.yaml    # Comprehensive Higress config
    │   │   └── kustomization.yaml
    │   ├── hello-service/
    │   ├── todo-service/
    │   ├── shortener-service/
    │   └── ...
    ├── observability/
    └── overlays/
        ├── development/
        └── production/
```

## Benefits

### 1. Clearer Organization

- All deployment configurations are now in `deploy/`
- Docker-related configs in `deploy/docker/`
- Kubernetes-related configs in `deploy/k8s/`

### 2. Reduced Duplication

- Removed duplicate Higress configurations
- Single source of truth for each configuration type

### 3. Better Maintainability

- Easier to find deployment configurations
- Consistent structure across deployment methods
- Clearer separation of concerns

### 4. Improved CI/CD

- Deployment scripts only need to reference `deploy/` directory
- Simpler path references in automation

## Migration Guide

### For Developers

If you have local scripts or commands referencing `tools/`, update them:

```bash
# Old
envoy -c tools/envoy/envoy-local.yaml

# New
envoy -c deploy/docker/envoy-local-config.yaml
```

```bash
# Old
kubectl apply -f tools/k8s/ingress.yaml

# New
kubectl apply -f deploy/k8s/services/higress/higress-routes.yaml
```

### For CI/CD Pipelines

Update any pipeline scripts that reference `tools/`:

```yaml
# Old
- name: Deploy Higress
  run: kubectl apply -f tools/higress/

# New
- name: Deploy Higress
  run: kubectl apply -k deploy/k8s/services/higress/
```

### For Docker Compose

No changes needed - the docker-compose files have been updated automatically.

## Verification

### 1. Check Envoy Configuration

```bash
# Verify Envoy config is accessible
ls -la deploy/docker/envoy-config.yaml
ls -la deploy/docker/envoy-local-config.yaml

# Test Docker Compose
cd deploy/docker
docker compose -f docker-compose.services.yml config | grep envoy
```

### 2. Check Higress Configuration

```bash
# Verify Higress config exists
ls -la deploy/k8s/services/higress/

# Test Kustomize build
kubectl kustomize deploy/k8s/services/higress/
```

### 3. Check Documentation

```bash
# Verify no references to tools/ remain
grep -r "tools/" --include="*.md" . | grep -v "docs/TOOLS_DIRECTORY_CLEANUP.md"
```

## Rollback (if needed)

If you need to rollback this change:

```bash
# Restore from git
git checkout HEAD~1 -- tools/

# Revert docker-compose changes
git checkout HEAD~1 -- deploy/docker/docker-compose.services.yml

# Remove new files
rm deploy/docker/envoy-config.yaml
rm deploy/docker/envoy-local-config.yaml
```

## Related Documentation

- [Deploy Docker README](../deploy/docker/README.md)
- [Deploy K8s README](../deploy/k8s/README.md)
- [Infrastructure Guide](./architecture/INFRASTRUCTURE.md)
- [Deployment Guide](./deployment/DEPLOYMENT_GUIDE.md)

## Questions?

Contact the Platform Team or open an issue in the repository.

---

**Completed**: 2026-01-25  
**Author**: Platform Team  
**Status**: ✅ Complete
