# Metadata Configuration Migration Guide

## Overview

This document describes the migration from `.apptype` files to `metadata.yaml` as the primary configuration source for service metadata in the monorepo.

## What Changed

### Before (Legacy)
- Service type stored in `.apptype` file
- Short names hardcoded in `app-manager.sh`
- Limited metadata support

### After (Current)
- All metadata centralized in `metadata.yaml`
- Short names configured per service
- Extensible metadata structure
- `.apptype` still supported for backward compatibility

## Metadata.yaml Format

```yaml
spec:
  name: service-name              # Full service name
  short_name: shortname           # Short name for CLI (e.g., 'make test APP=shortname')
  description: Service description
  type: go|java|node             # Service type
  port: 9092                      # Service port
  cd: true                        # Enable continuous deployment
  codeowners:
    - "@team-name"                # Code owners
test:
  coverage: 70                    # Overall coverage threshold
  service_coverage: 75            # Service layer coverage threshold (optional)
```

## Short Name Support

Short names allow convenient CLI usage:

```bash
# Instead of:
make test APP=shortener-service
make lint APP=hello-service
make build APP=todo-service

# You can now use:
make test APP=shortener
make lint APP=hello
make build APP=todo
```

### How Short Names Work

1. Short names are defined in `metadata.yaml` under `spec.short_name`
2. The `app-manager.sh` script automatically resolves short names to full names
3. Short names are dynamically discovered (no hardcoding required)

### Naming Convention

- Short name should be the service name without the `-service` suffix
- Example: `shortener-service` → `shortener`
- Example: `hello-service` → `hello`

## Detection Priority

The system detects service type in this order:

1. **metadata.yaml** (preferred) - `spec.type` field
2. **.apptype** (legacy) - file content
3. **File detection** (fallback) - based on build files (go.mod, build.gradle, package.json)

## Migration Steps

### For Existing Services

1. Add `short_name` field to `metadata.yaml`:
   ```yaml
   spec:
     name: my-service
     short_name: myservice  # Add this line
     type: go
     # ... rest of config
   ```

2. (Optional) Remove `.apptype` file:
   ```bash
   rm apps/my-service/.apptype
   ```

3. Test the configuration:
   ```bash
   make test APP=myservice
   ```

### For New Services

When creating a new service with `create-app.sh`, the short name is automatically generated:

```bash
./scripts/create-app.sh go my-awesome-service --port 9093
# Creates service with short_name: my-awesome
```

## Updated Scripts

The following scripts now support `metadata.yaml`:

- `scripts/app-manager.sh` - Main app management script
- `scripts/verify-auto-detection.sh` - Verification script
- `scripts/create-app.sh` - Service creation script

## Backward Compatibility

- `.apptype` files are still supported for legacy services
- Services without `short_name` in `metadata.yaml` will still work (just can't use short names)
- No breaking changes to existing workflows

## Benefits

1. **Centralized Configuration**: All service metadata in one place
2. **Dynamic Short Names**: No hardcoding in scripts
3. **Extensible**: Easy to add new metadata fields
4. **Better Documentation**: Self-documenting service configuration
5. **Improved DX**: Shorter, more convenient CLI commands

## Examples

### Current Services

| Service | Full Name | Short Name | Type |
|---------|-----------|------------|------|
| Hello Service | hello-service | hello | java |
| Todo Service | todo-service | todo | go |
| Shortener Service | shortener-service | shortener | go |
| Web Frontend | web | web | node |

### Usage Examples

```bash
# Run tests
make test APP=shortener
make test APP=hello
make test APP=todo

# Build services
make build APP=shortener
make build APP=hello

# Lint code
make lint APP=shortener
make lint-fix APP=hello

# Docker build
make docker-build APP=shortener
```

## Troubleshooting

### Short name not working

1. Check `metadata.yaml` has `short_name` field:
   ```bash
   grep "short_name" apps/my-service/metadata.yaml
   ```

2. Verify the short name is unique:
   ```bash
   grep -r "short_name:" apps/*/metadata.yaml
   ```

3. Test detection:
   ```bash
   ./scripts/app-manager.sh list
   ```

### Service type not detected

1. Check `metadata.yaml` has `type` field:
   ```bash
   grep "type:" apps/my-service/metadata.yaml
   ```

2. Verify detection:
   ```bash
   make verify-auto-detection
   ```

## Future Enhancements

Potential future additions to `metadata.yaml`:

- Dependencies between services
- Resource requirements (CPU, memory)
- Environment-specific configurations
- Health check endpoints
- Monitoring/alerting configuration

## References

- [App Management Guide](./APP_MANAGEMENT.md)
- [Create App Guide](./CREATE_APP_GUIDE.md)
- [Dynamic CI Strategy](./DYNAMIC_CI_STRATEGY.md)
