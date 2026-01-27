# Scripts Directory

This directory contains automation scripts for the monorepo.

## Service Creation

### create-app.sh

Creates a new service from templates with full automation.

**Usage:**
```bash
./scripts/create-app.sh <type> <name> [options]
```

**What it automates:**
1. ✅ Copies service template (go/java/node)
2. ✅ Replaces all placeholders with your values
3. ✅ Creates protobuf definition file
4. ✅ Generates protobuf code (for Go)
5. ✅ Initializes Go modules (for Go)
6. ✅ Creates Kubernetes resources in `deploy/k8s/services/<name>/`
7. ✅ **Adds service to K8s overlays** (development & production)
8. ✅ **Adds service to Docker Compose** (`docker-compose.services.yml`)
9. ✅ Registers service in app-manager.sh

**No manual steps required!** The service is immediately ready to use.

**Example:**
```bash
./scripts/create-app.sh go payment-service \
  --port 9094 \
  --description "Payment processing service" \
  --team payment-team
```

This creates a complete service with:
- Service code in `apps/payment-service/`
- Proto file in `api/v1/payment.proto`
- K8s resources in `deploy/k8s/services/payment-service/`
- Automatic registration in development overlay (1 replica)
- Automatic registration in production overlay (3 replicas)
- Automatic registration in Docker Compose

## App Management

### app-manager.sh

Provides functions for managing apps in the monorepo.

**Functions:**
- `get_app_type()` - Returns app type (go/java/node)
- `get_app_path()` - Returns app directory path
- `get_all_apps()` - Lists all apps
- `get_changed_apps()` - Detects changed apps

## Deployment

### deploy-k8s.sh

Deploys services to Kubernetes using Kustomize overlays.

**Usage:**
```bash
./scripts/deploy-k8s.sh <environment>
```

**Environments:**
- `development` - 1 replica per service
- `production` - 3 replicas per service
- `staging` - Custom configuration

## Testing

### test-services.sh

Runs tests for all or specific services.

### coverage-manager.sh

Manages test coverage reporting and verification.

## Development

### dev.sh

Starts all services for local development.

### proto-generator.sh

Generates protobuf code for all languages.

## Infrastructure

### test-etcd-cluster.sh

Tests etcd cluster connectivity.

### test-im-infrastructure.sh

Tests IM system infrastructure (MySQL, Redis, Kafka).

## Utilities

### check-env.sh

Verifies required tools are installed.

### check-versions.sh

Checks versions of installed tools.

### detect-changed-apps.sh

Detects which apps have changed (for CI/CD).

### verify-auto-detection.sh

Verifies app auto-detection is working.

### verify-docker-build.sh

Verifies Docker builds for all apps.

### verify-production.sh

Verifies production deployment.

## Hooks

### install-hooks.sh

Installs Git hooks for the repository.

### pre-commit-checks.sh

Runs pre-commit checks (linting, formatting, tests).

## Related Documentation

- [Create App Guide](../docs/development/CREATE_APP_GUIDE.md) - Detailed guide for creating apps
- [App Management](../docs/development/APP_MANAGEMENT.md) - Managing apps in the monorepo
- [Deployment Guide](../docs/deployment/DEPLOYMENT_GUIDE.md) - Deploying to Kubernetes
