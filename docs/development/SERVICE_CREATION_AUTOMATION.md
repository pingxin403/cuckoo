# Service Creation Automation

This document describes the automated service creation process in the monorepo.

## Overview

The `scripts/create-app.sh` script provides **fully automated** service creation with zero manual steps required. When you create a new service, everything is configured automatically.

## What Gets Automated

### 1. Service Template Setup ✅

- Copies appropriate template (go-service, java-service, or node-service)
- Renames files to match service name
- Replaces all placeholders with actual values
- Creates proper directory structure

### 2. Protobuf Configuration ✅

- Creates protobuf definition file in `api/v1/<service>.proto`
- Generates protobuf code for Go services
- Sets up proper package paths and imports

### 3. Module Initialization ✅

- Initializes Go modules with `go mod tidy` (for Go services)
- Sets up proper module paths
- Resolves dependencies

### 4. Kubernetes Resources ✅

**Creates K8s resources in `deploy/k8s/services/<service>/`:**
- `<service>-deployment.yaml` - Deployment configuration
- `<service>-service.yaml` - Service definition
- `kustomization.yaml` - Kustomize configuration

**Automatically adds to overlays:**
- `deploy/k8s/overlays/development/kustomization.yaml`
  - Adds to `resources:` section
  - Adds to `replicas:` section with count: 1
- `deploy/k8s/overlays/production/kustomization.yaml`
  - Adds to `resources:` section
  - Adds to `replicas:` section with count: 3

### 5. Docker Compose Integration ✅

**Automatically adds to `deploy/docker/docker-compose.services.yml`:**
- Service definition with proper build context
- Port mappings
- Environment variables
- Health checks
- Network configuration
- Restart policy

### 6. App Registration ✅

- Registers in `scripts/app-manager.sh`
- Creates `.apptype` file for auto-detection
- Creates `metadata.yaml` for service catalog

## Before vs After

### Before (Manual Process)

When creating a new service, you had to:

1. ✋ Copy template manually
2. ✋ Replace placeholders in all files
3. ✋ Create protobuf file
4. ✋ Generate protobuf code
5. ✋ Create K8s resources
6. ✋ **Manually edit `deploy/k8s/overlays/development/kustomization.yaml`**
7. ✋ **Manually edit `deploy/k8s/overlays/production/kustomization.yaml`**
8. ✋ **Manually edit `deploy/docker/docker-compose.services.yml`**
9. ✋ Register in app-manager.sh
10. ✋ Test everything works

**Total time: ~30-45 minutes**
**Error-prone: High** (easy to forget steps or make typos)

### After (Automated Process)

Now you just run:

```bash
./scripts/create-app.sh go payment-service \
  --port 9094 \
  --description "Payment processing service"
```

**Total time: ~30 seconds**
**Error-prone: None** (everything is automated)

## Implementation Details

### K8s Overlay Automation

The script uses `awk` to intelligently insert service entries into kustomization.yaml files:

```bash
# Adds to resources section
awk -v service="  - ../../services/$APP_NAME" '
    /^resources:/ { in_resources=1; print; next }
    in_resources && /^[^ ]/ { print service; in_resources=0 }
    { print }
' kustomization.yaml

# Adds to replicas section
awk -v name="$APP_NAME" -v count="$replica_count" '
    /^replicas:/ { in_replicas=1; print; next }
    in_replicas && /^[^ ]/ { 
        print "  - name: " name "\n    count: " count
        in_replicas=0 
    }
    { print }
' kustomization.yaml
```

**Benefits:**
- Preserves file formatting
- Maintains proper YAML indentation
- Handles edge cases correctly
- No manual editing required

### Docker Compose Automation

The script creates a temporary service block and inserts it before the `networks:` section:

```bash
# Create service block
cat > /tmp/new_service.yml << EOF
  $APP_NAME:
    build:
      context: ../..
      dockerfile: apps/$APP_NAME/Dockerfile
    # ... rest of configuration
EOF

# Insert into docker-compose.services.yml
awk '
    /^networks:/ { 
        while ((getline line < "/tmp/new_service.yml") > 0) {
            print line
        }
    }
    { print }
' docker-compose.services.yml
```

**Benefits:**
- Maintains proper YAML structure
- Adds appropriate health checks per service type
- Configures ports automatically
- No manual editing required

## Service Type Configurations

### Go Services

**Ports:** Auto-assigned from 9090+
**Health Check:** `pgrep -f <service-name>`
**Build:** Multi-stage Docker build
**Tests:** Unit tests + property-based tests
**Coverage:** 80% minimum

### Java Services

**Ports:** Auto-assigned from 9090+
**Health Check:** `pgrep -f 'java.*app.jar'`
**Build:** Gradle with Spring Boot
**Tests:** JUnit + jqwik property tests
**Coverage:** 80% minimum (90% for service classes)

### Node Services

**Ports:** 3000 (default)
**Health Check:** `wget --spider -q http://localhost:3000/health`
**Build:** Vite + TypeScript
**Tests:** Vitest
**Coverage:** 80% minimum

## Verification

After running the script, verify automation worked:

### Check K8s Overlays

```bash
# Development overlay should have your service
grep "your-service" deploy/k8s/overlays/development/kustomization.yaml

# Production overlay should have your service
grep "your-service" deploy/k8s/overlays/production/kustomization.yaml
```

### Check Docker Compose

```bash
# Docker Compose should have your service
grep "your-service:" deploy/docker/docker-compose.services.yml
```

### Test Deployment

```bash
# Test K8s deployment
kubectl apply -k deploy/k8s/overlays/development --dry-run=client

# Test Docker Compose
docker compose -f deploy/docker/docker-compose.services.yml config
```

## Troubleshooting

### Service Not in Overlays

If the service wasn't added to overlays:

1. Check if the overlay files exist
2. Verify the script had write permissions
3. Check for error messages in script output
4. Manually add if needed (but report the issue)

### Service Not in Docker Compose

If the service wasn't added to Docker Compose:

1. Check if `deploy/docker/docker-compose.services.yml` exists
2. Verify the script had write permissions
3. Check for error messages in script output
4. Manually add if needed (but report the issue)

### Duplicate Entries

If you run the script twice for the same service:

- The script detects duplicates and skips adding them
- You'll see warning messages
- No harm done - files remain valid

## Future Enhancements

Potential future automation:

- [ ] Auto-generate API documentation
- [ ] Auto-create monitoring dashboards
- [ ] Auto-configure service mesh
- [ ] Auto-setup CI/CD pipelines
- [ ] Auto-generate client libraries
- [ ] Auto-create integration tests

## Related Documentation

- [Create App Guide](CREATE_APP_GUIDE.md) - Complete guide for creating apps
- [App Management](APP_MANAGEMENT.md) - Managing apps in the monorepo
- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md) - Deploying services
- [Scripts README](../../scripts/README.md) - All available scripts

## Summary

The service creation automation eliminates manual steps and reduces errors. When you create a new service, it's immediately ready to:

- ✅ Build with `make build APP=your-service`
- ✅ Test with `make test APP=your-service`
- ✅ Deploy to K8s with `kubectl apply -k deploy/k8s/overlays/development`
- ✅ Run locally with `docker compose up your-service`

**No manual configuration required!**
