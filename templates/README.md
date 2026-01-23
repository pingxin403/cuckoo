# Service Templates

This directory contains templates for creating new services in the monorepo.

## Directory Structure

```
templates/
├── go-service/          # Go service template
│   ├── Dockerfile       # Multi-stage Docker build
│   ├── main.go          # Application entry point
│   ├── go.mod           # Go module definition
│   ├── metadata.yaml    # Service metadata
│   ├── catalog-info.yaml # Backstage integration
│   ├── service/         # Service implementation
│   └── storage/         # Storage layer (optional)
│
├── java-service/        # Java/Spring Boot service template
│   ├── Dockerfile       # Multi-stage Docker build
│   ├── build.gradle     # Gradle build configuration
│   ├── settings.gradle  # Gradle settings
│   ├── metadata.yaml    # Service metadata
│   ├── catalog-info.yaml # Backstage integration
│   └── src/             # Java source code
│
└── k8s/                 # Kubernetes resource templates
    ├── deployment.yaml  # Deployment configuration
    ├── service.yaml     # Service configuration
    └── kustomization.yaml # Kustomize configuration
```

## Usage

### Creating a New Service

Use the `create-app.sh` script to create a new service from templates:

```bash
# Go service
./scripts/create-app.sh go my-service --port 9097 --description "My service"

# Java service
./scripts/create-app.sh java my-service --port 9098 --description "My service"
```

### What Gets Created

When you create a new service, the script:

1. **Copies service template** from `templates/{type}-service/` to `apps/{service-name}/`
2. **Copies K8s templates** from `templates/k8s/` to `deploy/k8s/services/{service-name}/`
3. **Replaces placeholders** with actual values:
   - `{{SERVICE_NAME}}` → your service name
   - `{{GRPC_PORT}}` → your port number
   - `{{SERVICE_DESCRIPTION}}` → your description
   - And more...
4. **Creates protobuf file** in `api/v1/{service-name}.proto`
5. **Generates protobuf code** (for Go services)

### Result

After running the create command, you'll have:

```
apps/my-service/              # Service code
├── Dockerfile
├── main.go
├── service/
└── ...

deploy/k8s/services/my-service/  # K8s resources
├── my-service-deployment.yaml
├── my-service-service.yaml
└── kustomization.yaml

api/v1/my_service.proto       # API definition
```

## Customizing Templates

### Service Templates

To customize the default service structure:

1. Edit files in `templates/go-service/` or `templates/java-service/`
2. Use `{{PLACEHOLDER}}` syntax for values that should be replaced
3. New services will automatically use the updated templates

Available placeholders:
- `{{SERVICE_NAME}}` - Service name (e.g., `user-service`)
- `{{SHORT_NAME}}` - Short name without `-service` suffix
- `{{SERVICE_NAME_UPPER}}` - Uppercase with underscores (e.g., `USER_SERVICE`)
- `{{SERVICE_NAME_CAMEL}}` - CamelCase (e.g., `UserService`)
- `{{SERVICE_NAME_SNAKE}}` - Snake case (e.g., `user_service`)
- `{{SERVICE_DESCRIPTION}}` - Service description
- `{{GRPC_PORT}}` - gRPC port number
- `{{PACKAGE_NAME}}` - Java package name
- `{{MODULE_PATH}}` - Go module path
- `{{PROTO_FILE}}` - Protobuf file name
- `{{PROTO_PACKAGE}}` - Protobuf package name
- `{{TEAM_NAME}}` - Team name

### Kubernetes Templates

To customize the default K8s configuration:

1. Edit files in `templates/k8s/`
2. Use `{{SERVICE_NAME}}` and `{{GRPC_PORT}}` placeholders
3. New services will automatically use the updated templates

## Template Maintenance

### Adding New Files to Templates

1. Add the file to the appropriate template directory
2. Use placeholders for values that should be customized
3. The `create-app.sh` script will automatically copy and process it

### Removing Files from Templates

1. Delete the file from the template directory
2. Update this README if necessary
3. Existing services are not affected

### Testing Templates

After modifying templates, test by creating a new service:

```bash
# Create a test service
./scripts/create-app.sh go test-service --port 9999

# Verify the generated files
ls -la apps/test-service/
ls -la deploy/k8s/services/test-service/

# Clean up
rm -rf apps/test-service
rm -rf deploy/k8s/services/test-service
```

## Best Practices

1. **Keep templates minimal** - Only include essential files
2. **Use placeholders** - Make templates reusable with `{{PLACEHOLDER}}` syntax
3. **Document changes** - Update this README when modifying templates
4. **Test thoroughly** - Create a test service after template changes
5. **Version control** - Commit template changes with clear messages

## Related Documentation

- [Create App Guide](../docs/development/CREATE_APP_GUIDE.md) - Detailed guide for creating apps
- [App Management](../docs/development/APP_MANAGEMENT.md) - Managing apps in the monorepo
- [K8s Templates README](k8s/README.md) - Kubernetes template details
