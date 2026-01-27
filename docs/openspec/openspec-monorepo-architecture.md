# Monorepo Architecture

**Status**: Implemented  
**Owner**: Platform Team  
**Last Updated**: 2026-01-18

## Overview

Multi-language monorepo supporting Java, Go, and Node.js services with unified build system, dynamic CI/CD, and convention-based service management.

## Architecture Components

### 1. Service Layer

**Hello Service** (Java/Spring Boot):
- Port: 9090
- gRPC service providing greeting functionality
- Location: `apps/hello-service/`
- Type: `java`

**TODO Service** (Go):
- Port: 9091
- gRPC service for task management
- Location: `apps/todo-service/`
- Type: `go`
- Features: CRUD operations, in-memory storage

**Web App** (React/TypeScript):
- Port: 5173 (dev), 80 (prod)
- Frontend application
- Location: `apps/web/`
- Type: `node`

### 2. API Contract Layer

**Location**: `api/v1/`

**Protobuf Definitions**:
- `hello.proto` - Hello service API
- `todo.proto` - TODO service API

**Code Generation**:
- Go: `apps/todo-service/gen/`
- Java: `apps/hello-service/src/main/java-gen/`
- TypeScript: `apps/web/src/gen/`

**Strategy**: Hybrid generation
- Go: Generate inside Docker
- Java: Generate in CI, copy to Docker
- TypeScript: Generate in CI before tests

### 3. API Gateway

**Technology**: Higress (K8s native) / Envoy (local dev)

**Routing**:
- `/api/hello` → hello-service:9090
- `/api/todo` → todo-service:9091

**Features**:
- gRPC-Web protocol conversion
- CORS handling
- TLS termination (production)

### 4. Communication Patterns

**North-South** (Frontend → Backend):
- Through API gateway
- gRPC-Web protocol
- Unified entry point

**East-West** (Service → Service):
- Direct gRPC communication
- Kubernetes DNS for service discovery
- Configuration-driven addresses

## Service Metadata

Each service has:

**`.apptype` file**:
```
java  # or go, or node
```

**`metadata.yaml` file**:
```yaml
spec:
  name: hello-service
  description: Greeting service
  type: java
  cd: true
  codeowners:
    - "@backend-java-team"
test:
  coverage: 30
```

## Scalability Features

### Convention-Based Detection

**Service Type Detection** (priority order):
1. `.apptype` file
2. `metadata.yaml` file
3. File characteristics (build.gradle, go.mod, package.json)

**Benefits**:
- Zero configuration for new services
- Automatic CI/CD integration
- Supports unlimited services (app1-100, web1-50)

### Dynamic CI/CD

**Change Detection**:
- Scans `apps/*/` directory dynamically
- Detects changes via git diff
- Builds only changed services

**Matrix Strategy**:
- Parallel builds for changed services
- Dynamic job generation
- 60-80% CI time savings

## Directory Structure

```
monorepo/
├── api/v1/                    # API contracts (Protobuf)
├── apps/                      # Services
│   ├── hello-service/         # Java service
│   ├── todo-service/          # Go service
│   └── web/                   # React app
├── templates/                 # Service templates
│   ├── java-service/
│   └── go-service/
├── k8s/                       # Kubernetes configs
│   ├── base/
│   └── overlays/
├── scripts/                   # Build scripts
│   ├── app-manager.sh
│   ├── detect-changed-apps.sh
│   ├── create-app.sh
│   └── pre-commit-checks.sh
├── tools/                     # Infrastructure tools
│   ├── envoy/
│   └── k8s/
└── docs/                      # Documentation
```

## Build System

**Primary Tool**: Makefile + Shell Scripts

**Key Commands**:
- `make proto` - Generate protobuf code
- `make test [APP=name]` - Run tests
- `make build [APP=name]` - Build services
- `make docker-build [APP=name]` - Build Docker images
- `make create` - Create new service

**App Manager**:
- Unified interface for all services
- Auto-detection of changed apps
- Support for short names (hello, todo, web)

## Deployment

**Local Development**:
```bash
./scripts/dev.sh  # Starts all services + Envoy
```

**Kubernetes**:
```bash
kubectl apply -k k8s/overlays/production
```

**CI/CD**:
- GitHub Actions
- Dynamic matrix builds
- Selective deployment

## Quality Practices

**Shift-Left**:
- Pre-commit hooks (`make pre-commit`)
- Tool version verification
- Protobuf sync verification
- Linting and testing
- Security scanning

**Test Coverage**:
- Go: 70% overall, 75% service layer
- Java: 30% overall, 50% service layer
- Excludes generated code

## References

- [Architecture Scalability Analysis](../archive/ARCHITECTURE_SCALABILITY_ANALYSIS.md)
- [Dynamic CI Strategy](../ci-cd/DYNAMIC_CI_STRATEGY.md)
- [Proto Hybrid Strategy](../archive/PROTO_HYBRID_STRATEGY.md)
- [Shift-Left Practices](../process/SHIFT_LEFT.md)
