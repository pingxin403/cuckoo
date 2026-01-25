# Infrastructure and API Gateway Configuration

This document describes the infrastructure setup, API gateway configuration, and CI/CD pipeline for the Monorepo Hello/TODO Services project.

## Overview

The infrastructure consists of:
- **Local Development**: Envoy proxy for local development
- **Kubernetes Deployment**: Higress ingress controller for production
- **CI/CD Pipeline**: GitHub Actions for automated testing and deployment
- **Code Quality**: Linters and formatters for all languages

## Components

### 1. Local Envoy Proxy

**Location**: `deploy/docker/envoy-local-config.yaml`

**Purpose**: Provides a local API gateway for development that:
- Routes `/api/hello` to Hello Service (localhost:9090)
- Routes `/api/todo` to TODO Service (localhost:9091)
- Handles gRPC-Web protocol conversion
- Configures CORS for frontend access

**Usage**:
```bash
# Start Envoy
envoy -c deploy/docker/envoy-local-config.yaml

# Or use the dev script (starts all services)
./scripts/dev.sh
```

**Ports**:
- Envoy proxy: 8080
- Envoy admin: 9901

### 2. Development Startup Script

**Location**: `scripts/dev.sh`

**Purpose**: Orchestrates all services in development mode with:
- Automatic port availability checking
- Service health monitoring
- Graceful shutdown handling
- Centralized logging

**Services Started**:
1. Hello Service (Java/Spring Boot) - Port 9090
2. TODO Service (Go) - Port 9091
3. Envoy Proxy - Port 8080
4. Frontend (React/Vite) - Port 5173

**Usage**:
```bash
./scripts/dev.sh
```

**Logs**: Available in `logs/` directory

### 3. Higress Ingress (Kubernetes)

**Location**: `deploy/k8s/services/higress/higress-routes.yaml`

**Purpose**: Production-grade API gateway for Kubernetes with:
- gRPC and gRPC-Web support
- CORS configuration
- TLS/SSL termination
- Rate limiting
- Security headers

**Features**:
- Path-based routing to backend services
- Health checks
- Load balancing
- Protocol conversion (gRPC-Web to gRPC)

**Deployment**:
```bash
kubectl apply -f deploy/k8s/services/higress/higress-routes.yaml
```

### 4. Kustomize Configuration

**Location**: `k8s/`

**Structure**:
```
k8s/
├── base/                    # Base configuration
│   └── kustomization.yaml
├── overlays/
│   ├── development/         # Development environment
│   └── production/          # Production environment
└── README.md
```

**Environments**:

**Development**:
- Single replica per service
- Lower resource limits
- Debug logging enabled
- No TLS

**Production**:
- 3 replicas per service
- Higher resource limits
- Production logging
- TLS enabled
- Rate limiting
- Security headers

**Usage**:
```bash
# Deploy to development
kubectl apply -k k8s/overlays/development

# Deploy to production
kubectl apply -k k8s/overlays/production
```

### 5. CI/CD Pipeline

**Location**: `.github/workflows/ci.yml`

**Jobs**:

1. **verify-proto**: Verify Protobuf code generation
2. **build-hello-service**: Build and test Java service
3. **build-todo-service**: Build and test Go service
4. **build-frontend**: Build and test React app
5. **push-images**: Push Docker images to registry
6. **deploy-k8s**: Deploy to Kubernetes (production)
7. **security-scan**: Scan Docker images for vulnerabilities

**Triggers**:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`

**Artifacts**:
- Docker images (pushed to GitHub Container Registry)
- Test reports
- Build artifacts

**Deployment**:
- Automatic deployment to production on push to `main`
- Manual approval can be added via GitHub Environments

### 6. Code Quality Tools

**Location**: Various configuration files

**Java (Hello Service)**:
- Checkstyle: `apps/hello-service/config/checkstyle/checkstyle.xml`
- SpotBugs: `apps/hello-service/config/spotbugs/spotbugs-exclude.xml`

**Go (TODO Service)**:
- golangci-lint: `apps/todo-service/.golangci.yml`

**TypeScript (Web)**:
- ESLint: `apps/web/eslint.config.js`
- Prettier: `apps/web/.prettierrc`

**Pre-commit Hooks**:
- Location: `.githooks/pre-commit`
- Install: `./scripts/install-hooks.sh`

**Usage**:
```bash
# Run all linters
make lint

# Run all formatters
make format

# Run specific linters
make lint APP=hello-service
make lint APP=todo-service
make lint APP=web
```

## Architecture Diagrams

### Local Development Architecture

```
┌─────────────┐
│   Browser   │
│  (Frontend) │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────┐
│    Envoy    │
│  (Port 8080)│
└──────┬──────┘
       │
       ├─────────────┐
       │             │
       ▼             ▼
┌─────────────┐ ┌─────────────┐
│   Hello     │ │    TODO     │
│  Service    │◄┤   Service   │
│ (Port 9090) │ │ (Port 9091) │
└─────────────┘ └─────────────┘
```

### Kubernetes Production Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTPS
       ▼
┌─────────────┐
│   Higress   │
│   Ingress   │
└──────┬──────┘
       │
       ├─────────────┐
       │             │
       ▼             ▼
┌─────────────┐ ┌─────────────┐
│   Hello     │ │    TODO     │
│  Service    │◄┤   Service   │
│   (3 pods)  │ │   (3 pods)  │
└─────────────┘ └─────────────┘
```

## Configuration

### Environment Variables

**Hello Service**:
- `SPRING_PROFILES_ACTIVE`: Spring profile (development/production)
- `GRPC_SERVER_PORT`: gRPC server port (default: 9090)

**TODO Service**:
- `PORT`: Server port (default: 9091)
- `HELLO_SERVICE_ADDR`: Hello service address (e.g., hello-service:9090)

**Frontend**:
- Configured via Vite proxy in development
- Uses Envoy/Higress in production

### Resource Limits

**Development**:
- Hello Service: 256Mi memory, 100m CPU
- TODO Service: 128Mi memory, 50m CPU

**Production**:
- Hello Service: 512Mi-1Gi memory, 500m-1000m CPU
- TODO Service: 256Mi-512Mi memory, 200m-400m CPU

## Monitoring and Observability

### Health Checks

All services implement gRPC health checks:
- Liveness probe: Checks if service is running
- Readiness probe: Checks if service is ready to accept traffic

### Logs

**Local Development**:
- Logs available in `logs/` directory
- Tailed automatically by `dev.sh` script

**Kubernetes**:
```bash
# View logs
kubectl logs -f deployment/hello-service -n production
kubectl logs -f deployment/todo-service -n production

# View all logs
kubectl logs -f -l project=monorepo-platform -n production
```

### Metrics

Envoy admin interface provides metrics:
- Local: http://localhost:9901
- Kubernetes: Port-forward to Envoy pod

## Security

### TLS/SSL

Production ingress supports TLS:
- Configured via cert-manager
- Automatic certificate renewal
- HTTPS redirect enabled

### CORS

CORS is configured to allow:
- All origins (can be restricted in production)
- Methods: GET, POST, PUT, DELETE, OPTIONS
- Headers: content-type, x-grpc-web, x-user-agent, grpc-timeout

### Security Headers

Production ingress adds security headers:
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- X-XSS-Protection: 1; mode=block
- Strict-Transport-Security: max-age=31536000

### Rate Limiting

Production ingress includes rate limiting:
- 1000 requests per second
- Burst: 2000 requests

## Troubleshooting

### Local Development Issues

**Port already in use**:
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

**Envoy not found**:
```bash
# Install Envoy (macOS)
brew install envoy

# Install Envoy (Linux)
# See https://www.envoyproxy.io/docs/envoy/latest/start/install
```

**Services not starting**:
- Check logs in `logs/` directory
- Ensure all dependencies are installed
- Verify ports are available

### Kubernetes Issues

**Ingress not working**:
```bash
# Check ingress status
kubectl describe ingress monorepo-ingress -n production

# Check Higress controller
kubectl logs -n higress-system deployment/higress-controller
```

**Pods not starting**:
```bash
# Check pod status
kubectl get pods -n production

# Describe pod
kubectl describe pod <pod-name> -n production

# View logs
kubectl logs <pod-name> -n production
```

**Service not accessible**:
```bash
# Check service endpoints
kubectl get endpoints -n production

# Port-forward for testing
kubectl port-forward svc/hello-service 9090:9090 -n production
```

### CI/CD Issues

**Build failing**:
- Check GitHub Actions logs
- Verify all tests pass locally
- Ensure dependencies are up to date

**Deployment failing**:
- Verify KUBECONFIG secret is set
- Check Kubernetes cluster connectivity
- Verify image registry credentials

## Best Practices

1. **Always use the dev script** for local development
2. **Run linters before committing** code
3. **Test locally** before pushing to CI/CD
4. **Use Kustomize overlays** for environment-specific config
5. **Monitor logs and metrics** in production
6. **Keep dependencies updated** regularly
7. **Review security scan results** from CI/CD

## Additional Resources

- [Envoy Documentation](https://www.envoyproxy.io/docs)
- [Higress Documentation](https://higress.io/)
- [Kustomize Documentation](https://kustomize.io/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
