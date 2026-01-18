# Project Context

## Purpose

This is a multi-language Monorepo platform demonstrating microservices architecture with:
- **Hello Service** (Java/Spring Boot) - Greeting service
- **TODO Service** (Go) - Task management service  
- **Web App** (React/TypeScript) - Frontend application

The project showcases:
- Contract-first API design with Protobuf
- Multi-language service coexistence
- Dynamic CI/CD with change detection
- Scalable architecture supporting unlimited services
- Shift-left quality practices

## Tech Stack

### Backend
- **Java 17+** with Spring Boot 3.x and gRPC
- **Go 1.21+** with gRPC and Protocol Buffers
- **Gradle** for Java builds
- **Go modules** for Go dependency management

### Frontend
- **React 18+** with TypeScript 5+
- **Vite** for build tooling
- **TanStack Query** (React Query) for data fetching
- **grpc-web** for gRPC communication

### Infrastructure
- **Kubernetes** for orchestration
- **Higress/Envoy** for API gateway (gRPC-Web proxy)
- **Docker** for containerization
- **GitHub Actions** for CI/CD
- **Kustomize** for K8s configuration management

### API & Code Generation
- **Protocol Buffers** (protoc 28.3) for API contracts
- **protoc-gen-go** (v1.36.6) for Go code generation
- **protoc-gen-go-grpc** (v1.5.1) for Go gRPC code
- **ts-proto** for TypeScript code generation
- **protoc-gen-grpc-java** for Java gRPC code

## Project Conventions

### Code Style

**Java**:
- Google Java Style Guide
- Checkstyle enforced in CI
- 4-space indentation
- Package naming: `com.pingxin403.*`

**Go**:
- Standard Go formatting (`gofmt`)
- golangci-lint enforced
- Module path: `github.com/pingxin403/*`

**TypeScript**:
- ESLint + Prettier
- 2-space indentation
- Functional components with hooks

### Architecture Patterns

**Microservices Architecture**:
- Each service is independently deployable
- Services communicate via gRPC (east-west traffic)
- Frontend accesses services through API gateway (north-south traffic)

**Contract-First Design**:
- All APIs defined in `api/v1/*.proto`
- Code generated from Protobuf definitions
- No direct service-to-service source code dependencies

**Hybrid Proto Generation Strategy**:
- **Go**: Generate proto inside Docker (self-contained)
- **TypeScript**: Generate proto in CI before tests
- **Java**: Generate proto in CI, copy to Docker

**Convention-Based Service Detection**:
- Services use `.apptype` files (java/go/node)
- Services use `metadata.yaml` for configuration
- CI dynamically detects changed services
- No hardcoded service names in build system

### Testing Strategy

**Test Coverage Requirements**:
- **Go services**: 70% overall, 75% for service layer
- **Java services**: 30% overall, 50% for service layer
- Excludes generated code and non-business logic

**Testing Levels**:
1. **Unit Tests**: Test individual components in isolation
2. **Property-Based Tests**: Verify correctness properties (jqwik for Java, rapid for Go)
3. **Integration Tests**: Test service interactions
4. **E2E Tests**: Test complete user flows

**Test Organization**:
- Unit tests alongside source code
- Property tests in separate files (`*_property_test.go`, `*PropertyTest.java`)
- Coverage reports generated per service

### Git Workflow

**Branching Strategy**:
- `main` branch for production
- Feature branches: `feature/*`
- Bugfix branches: `bugfix/*`

**Commit Conventions**:
- Conventional Commits format
- Pre-commit hooks run quality checks

**Pull Request Requirements**:
- All CI checks must pass
- At least one CODEOWNERS approval
- Code must pass linting and formatting
- New features require tests
- API changes require documentation updates

### Shift-Left Quality Practices

**Pre-Commit Checks** (`make pre-commit`):
1. Tool version consistency
2. Protobuf code synchronization
3. Linting (all languages)
4. Unit tests
5. Common issues (console.log, TODOs, large files)
6. Security (potential secrets scan)

**CI/CD Quality Gates**:
- Test coverage verification
- Security scanning
- Docker image building
- Kubernetes deployment validation

## Domain Context

### Service Communication Patterns

**North-South Traffic** (Frontend → Backend):
- Frontend calls services through Higress/Envoy gateway
- gRPC-Web protocol conversion
- CORS handling at gateway level

**East-West Traffic** (Service → Service):
- Direct gRPC communication between services
- Service discovery via Kubernetes DNS
- Configuration-driven service addresses (environment variables)

### Application Management

**App Manager System**:
- Unified interface: `make <command> APP=<name>`
- Auto-detection of changed apps via git diff
- Support for short names: `hello`, `todo`, `web`
- Commands: test, build, run, docker, lint, clean, format

**App Creation**:
- Template-based: `make create` (interactive)
- Automatic port allocation
- Automatic Protobuf file generation
- Immediate CI/CD integration

### Dynamic CI/CD

**Change Detection**:
- Detects changed apps from git diff
- Checks `apps/*/`, `api/`, `libs/` directories
- Builds only changed services in parallel
- Selective Docker image pushing and K8s deployment

**Matrix Strategy**:
- Parallel builds for changed services
- Dynamic job generation
- 60-80% CI time savings

## Important Constraints

### Tool Versions

All tool versions centralized in `.tool-versions`:
- protoc 28.3
- protoc-gen-go v1.36.6
- protoc-gen-go-grpc v1.5.1
- go 1.21+
- java 17+
- node 18+

**Verification**: `make check-versions`

### Generated Code

**NOT committed to git**:
- `apps/*/gen/` (Go proto code)
- `apps/*/src/main/java-gen/` (Java proto code)
- `apps/*/src/gen/` (TypeScript proto code)

**Generated during**:
- Local development: `make proto`
- CI pipeline: Before build/test steps
- Docker build: Inside container (Go only)

### Port Allocation

- Hello Service: 9090
- TODO Service: 9091
- Web App: 5173 (dev), 80 (prod)
- Envoy Proxy: 8080 (local dev)

### Scalability

Architecture supports:
- Unlimited services of same type (app1-100, web1-50)
- Zero configuration changes for new services
- Convention-based auto-detection

## External Dependencies

### Development Tools
- Docker for containerization
- kubectl for Kubernetes management
- protoc for Protobuf compilation
- golangci-lint for Go linting
- Maven/Gradle for Java builds

### Runtime Dependencies
- Kubernetes cluster for deployment
- Container registry for Docker images
- Higress/Envoy for API gateway

### Optional Tools
- Backstage for developer portal (future)
- ArgoCD for GitOps deployment (future)
- OpenTelemetry for observability (future)

## Key Commands

```bash
# Development
make proto              # Generate protobuf code
make test              # Run all tests
make test APP=hello    # Test specific app
make lint              # Run linters
make lint-fix          # Fix linting issues
make pre-commit        # Run all pre-commit checks

# Building
make build             # Build all apps
make build APP=todo    # Build specific app
make docker-build      # Build Docker images

# Running
make run APP=hello     # Run specific service
./scripts/dev.sh       # Start all services locally

# App Management
make list-apps         # List all applications
make create            # Create new service (interactive)

# Verification
make check-versions    # Verify tool versions
make verify-auto-detection  # Verify service detection
```
