# Docker Deployment Guide

This guide covers building, testing, and deploying the Monorepo Hello/TODO Services using Docker.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Building Docker Images](#building-docker-images)
- [Running with Docker Compose](#running-with-docker-compose)
- [Manual Docker Commands](#manual-docker-commands)
- [Image Registry](#image-registry)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- **Docker** 20.10+ installed and running
- **Docker Compose** 2.0+ (included with Docker Desktop)
- Sufficient disk space (~2GB for images)

### Verify Docker Installation

```bash
docker --version
docker-compose --version
docker info
```

## Building Docker Images

### Quick Build (All Services)

```bash
# Build all images using Makefile
make docker-build
```

This will build:
- `hello-service:latest` (~300MB)
- `todo-service:latest` (~20MB)

### Verify Build

```bash
# Run verification script
./scripts/verify-docker-build.sh
```

This script will:
1. Build both images
2. Test that containers can start
3. Display image sizes
4. Provide next steps

### Individual Service Builds

```bash
# Build Hello Service only
make docker-build APP=hello-service

# Build TODO Service only
make docker-build APP=todo-service
```

### Manual Build Commands

```bash
# Hello Service
docker build -t hello-service:latest -f apps/hello-service/Dockerfile apps/hello-service

# TODO Service
docker build -t todo-service:latest -f apps/todo-service/Dockerfile apps/todo-service
```

## Running with Docker Compose

Docker Compose provides the easiest way to run all services together.

### Start All Services

```bash
docker-compose up -d
```

This starts:
- Hello Service (port 9090)
- TODO Service (port 9091)
- Envoy Gateway (port 8080, 9901)

### Check Service Status

```bash
# View running containers
docker-compose ps

# View logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f hello-service
docker-compose logs -f todo-service
docker-compose logs -f envoy
```

### Test Services

```bash
# Test Hello Service
grpcurl -plaintext -d '{"name":"Docker"}' localhost:9090 api.v1.HelloService/SayHello

# Test TODO Service
grpcurl -plaintext -d '{"title":"Docker TODO"}' localhost:9091 api.v1.TodoService/CreateTodo

# Test through Envoy (requires gRPC-Web client)
curl http://localhost:8080
```

### Stop Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# Stop and remove images
docker-compose down --rmi all
```

## Manual Docker Commands

### Run Hello Service

```bash
docker run -d \
  --name hello-service \
  -p 9090:9090 \
  -e SPRING_PROFILES_ACTIVE=docker \
  hello-service:latest
```

### Run TODO Service

```bash
docker run -d \
  --name todo-service \
  -p 9091:9091 \
  -e HELLO_SERVICE_ADDR=host.docker.internal:9090 \
  todo-service:latest
```

### Run Envoy Gateway

```bash
docker run -d \
  --name envoy-gateway \
  -p 8080:8080 \
  -p 9901:9901 \
  -v $(pwd)/tools/envoy/envoy-docker.yaml:/etc/envoy/envoy.yaml:ro \
  envoyproxy/envoy:v1.30-latest \
  -c /etc/envoy/envoy.yaml
```

### Container Management

```bash
# View running containers
docker ps

# View all containers (including stopped)
docker ps -a

# View container logs
docker logs hello-service
docker logs -f todo-service  # Follow logs

# Execute command in container
docker exec -it hello-service sh

# Stop container
docker stop hello-service

# Remove container
docker rm hello-service

# Remove all stopped containers
docker container prune
```

## Image Registry

### Tag Images for Registry

```bash
# Tag for Docker Hub
docker tag hello-service:latest username/hello-service:v1.0.0
docker tag todo-service:latest username/todo-service:v1.0.0

# Tag for private registry
docker tag hello-service:latest registry.example.com/hello-service:v1.0.0
docker tag todo-service:latest registry.example.com/todo-service:v1.0.0
```

### Push to Registry

```bash
# Login to registry
docker login registry.example.com

# Push images
docker push registry.example.com/hello-service:v1.0.0
docker push registry.example.com/todo-service:v1.0.0
```

### Pull from Registry

```bash
# Pull images
docker pull registry.example.com/hello-service:v1.0.0
docker pull registry.example.com/todo-service:v1.0.0
```

### Update Kustomize Configuration

After pushing to registry, update the Kustomize configuration:

```yaml
# k8s/overlays/production/kustomization.yaml
images:
  - name: hello-service
    newName: registry.example.com/hello-service
    newTag: v1.0.0
  - name: todo-service
    newName: registry.example.com/todo-service
    newTag: v1.0.0
```

## Image Optimization

### Multi-Stage Builds

Both Dockerfiles use multi-stage builds to minimize image size:

**Hello Service**:
- Build stage: Uses `gradle:8.14.3-jdk17` (~800MB)
- Runtime stage: Uses `eclipse-temurin:17-jre-alpine` (~200MB)
- Final image: ~300MB

**TODO Service**:
- Build stage: Uses `golang:1.21-alpine` (~300MB)
- Runtime stage: Uses `alpine:latest` (~5MB)
- Final image: ~20MB

### Best Practices

1. **Layer Caching**: Order Dockerfile commands from least to most frequently changing
2. **Minimize Layers**: Combine RUN commands where possible
3. **Use .dockerignore**: Exclude unnecessary files from build context
4. **Security**: Run as non-root user (already implemented)
5. **Health Checks**: Include health check commands (already implemented)

### Create .dockerignore Files

```bash
# apps/hello-service/.dockerignore
target/
.gradle/
build/
*.log
.git/
.idea/
*.iml

# apps/todo-service/.dockerignore
bin/
*.log
.git/
.idea/
```

## Troubleshooting

### Build Failures

**Problem**: Protobuf code not generated

```bash
# Solution: Generate Protobuf code first
make gen-proto
make docker-build
```

**Problem**: Out of disk space

```bash
# Solution: Clean up Docker resources
docker system prune -a
docker volume prune
```

**Problem**: Build context too large

```bash
# Solution: Add .dockerignore files (see above)
```

### Runtime Issues

**Problem**: Container exits immediately

```bash
# Check logs
docker logs hello-service

# Common causes:
# - Port already in use
# - Missing environment variables
# - Configuration errors
```

**Problem**: Cannot connect to service

```bash
# Check if container is running
docker ps

# Check port mapping
docker port hello-service

# Check network connectivity
docker network inspect bridge
```

**Problem**: Service-to-service communication fails

```bash
# Use Docker network
docker network create monorepo-network
docker run --network monorepo-network ...

# Or use Docker Compose (handles networking automatically)
docker-compose up
```

### Performance Issues

**Problem**: Slow startup

```bash
# Check resource allocation
docker stats

# Increase Docker resources in Docker Desktop settings:
# - Memory: 4GB minimum
# - CPUs: 2 minimum
```

**Problem**: High memory usage

```bash
# Monitor container resources
docker stats hello-service todo-service

# Adjust JVM settings for Hello Service
docker run -e JAVA_OPTS="-Xmx512m" hello-service:latest
```

## Health Checks

### Container Health Status

```bash
# View health status
docker ps

# Inspect health check details
docker inspect hello-service | grep -A 10 Health
```

### Custom Health Checks

**Hello Service** (gRPC health check):
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD pgrep -f "java.*app.jar" || exit 1
```

**TODO Service** (process check):
```dockerfile
HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
  CMD pgrep -f "todo-service" || exit 1
```

## Security Considerations

### Image Scanning

```bash
# Scan for vulnerabilities (requires Docker Scout or Trivy)
docker scout cves hello-service:latest
docker scout cves todo-service:latest

# Or use Trivy
trivy image hello-service:latest
trivy image todo-service:latest
```

### Best Practices

1. **Use Official Base Images**: We use `eclipse-temurin` and `alpine`
2. **Run as Non-Root**: Both services run as non-root users
3. **Minimal Base Images**: Use Alpine for smaller attack surface
4. **Regular Updates**: Keep base images and dependencies updated
5. **Secrets Management**: Never hardcode secrets in Dockerfiles

## Next Steps

After successful Docker deployment:

1. **Test locally**: Use Docker Compose to verify all services work together
2. **Push to registry**: Tag and push images to your container registry
3. **Deploy to Kubernetes**: Use Kustomize to deploy to K8s cluster
4. **Set up CI/CD**: Automate builds and deployments

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Multi-Stage Builds](https://docs.docker.com/build/building/multi-stage/)

