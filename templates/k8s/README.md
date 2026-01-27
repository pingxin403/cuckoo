# Kubernetes Templates

This directory contains Kubernetes resource templates used by the `create-app.sh` script.

## Templates

- **deployment.yaml** - Deployment configuration template
- **service.yaml** - Service configuration template
- **kustomization.yaml** - Kustomize configuration template

## Placeholders

The following placeholders are replaced when creating a new service:

- `{{SERVICE_NAME}}` - The service name (e.g., `user-service`)
- `{{GRPC_PORT}}` - The gRPC port number (e.g., `9095`)

## Usage

These templates are automatically used by `./scripts/create-app.sh` when creating a new service.

When you run:
```bash
./scripts/create-app.sh go my-service --port 9097
```

The script will:
1. Copy these templates to `deploy/k8s/services/my-service/`
2. Replace all placeholders with actual values
3. Rename files to match the service name (e.g., `my-service-deployment.yaml`)

## Customization

To customize the default Kubernetes configuration for all new services:
1. Edit the template files in this directory
2. Use `{{PLACEHOLDER}}` syntax for values that should be replaced
3. New services will automatically use the updated templates

## Example

After running the create command, the templates generate:
```
deploy/k8s/services/my-service/
├── my-service-deployment.yaml
├── my-service-service.yaml
└── kustomization.yaml
```

These files are ready to be deployed with:
```bash
kubectl apply -k deploy/k8s/services/my-service/
```

Or included in Kustomize overlays:
```bash
kubectl apply -k deploy/k8s/overlays/development/
```
