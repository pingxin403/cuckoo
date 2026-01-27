# Hello Service

A greeting service based on Java/Spring Boot and gRPC, providing personalized greeting message functionality.

## Overview

This service is part of the monorepo and provides greeting capabilities via gRPC.

- **Port**: 9090
- **Protocol**: gRPC
- **Language**: Java 17
- **Framework**: Spring Boot 3.5.0
- **Team**: backend-java-team

## Technology Stack

- **Java**: 17
- **Spring Boot**: 3.5.0
- **gRPC**: 1.60.0
- **Protobuf**: 3.25.1
- **Build Tool**: Gradle 8.14.3

## Project Structure

```
hello-service/
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/pingxin403/cuckoo/hello/
│   │   │       ├── HelloServiceApplication.java    # Main application class
│   │   │       └── service/
│   │   │           └── HelloServiceImpl.java       # gRPC service implementation
│   │   └── resources/
│   │       └── application.yml                     # Application configuration
│   └── test/
├── k8s/                                            # Kubernetes resources
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
├── build.gradle                                    # Gradle build configuration
├── Dockerfile                                      # Docker image build
└── catalog-info.yaml                               # Backstage service catalog
```

## API Definition

The service is based on Protobuf definition located at `../../api/v1/hello.proto`:

```protobuf
service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
```

### Functionality

- **Input**: User name (optional)
- **Output**: Personalized greeting message
- **Rules**:
  - If name is provided: Returns "Hello, {name}!"
  - If name is not provided or empty: Returns "Hello, World!"

## Quick Start

### Prerequisites

- Java 17+
- Gradle (using gradlew wrapper)

### Generate Protobuf Code

```bash
./gradlew generateProto
```

### Build Project

```bash
# Build (skip tests)
./gradlew build -x test

# Build (with tests)
./gradlew build
```

### Run Service

```bash
./gradlew bootRun
```

The service will start the gRPC server on port **9090**.

### Test Service

Test the service using grpcurl:

```bash
# Install grpcurl
brew install grpcurl

# Call SayHello method
grpcurl -plaintext -d '{"name": "Alice"}' localhost:9090 api.v1.HelloService/SayHello

# Expected output
{
  "message": "Hello, Alice!"
}

# Test with empty name
grpcurl -plaintext -d '{"name": ""}' localhost:9090 api.v1.HelloService/SayHello

# Expected output
{
  "message": "Hello, World!"
}
```

## Docker Deployment

### Build Docker Image

```bash
docker build -t hello-service:latest .
```

### Run Docker Container

```bash
docker run -p 9090:9090 hello-service:latest
```

## Kubernetes Deployment

### Deploy to K8s Cluster

```bash
# Apply all K8s resources
kubectl apply -k deploy/k8s/overlays/development

# Check deployment status
kubectl get pods -l app=hello-service
kubectl get svc hello-service

# View logs
kubectl logs -l app=hello-service -f
```

### Access Service

Within the K8s cluster, the service can be accessed at:

```
hello-service:9090
```

## Configuration

### application.yml

Main configuration items:

```yaml
grpc:
  server:
    port: 9090              # gRPC server port

spring:
  application:
    name: hello-service     # Application name

logging:
  level:
    root: INFO
    com.pingxin403.cuckoo: DEBUG
```

### Environment Variables

- `SPRING_PROFILES_ACTIVE`: Spring profile (e.g., `production`)
- `GRPC_SERVER_PORT`: gRPC server port (default 9090)
- `JAVA_OPTS`: JVM parameters

## Testing

### Run Tests

```bash
# Run all tests
./gradlew test

# Run tests with coverage report
./gradlew test jacocoTestReport

# Verify coverage thresholds (30% overall)
./gradlew test jacocoTestCoverageVerification
```

Coverage reports are generated at:
- HTML: `build/reports/jacoco/test/html/index.html`
- XML: `build/reports/jacoco/test/jacocoTestReport.xml`

### Coverage Requirements

- **Overall coverage**: 30% minimum
- **Service classes**: 50% minimum

### Property-Based Tests

The service includes jqwik for property-based testing:

```java
import net.jqwik.api.*;

class HelloServicePropertyTest {
    
    @Property
    void sayHelloNeverReturnsNull(@ForAll String name) {
        HelloServiceImpl service = new HelloServiceImpl();
        HelloRequest request = HelloRequest.newBuilder()
            .setName(name)
            .build();
        
        HelloResponse response = service.sayHello(request);
        assertThat(response.getMessage()).isNotNull();
    }
}
```

## Monitoring and Health Checks

### Health Checks

K8s configuration includes the following probes:

- **Liveness Probe**: gRPC health check on port 9090
- **Readiness Probe**: gRPC readiness check on port 9090
- **Startup Probe**: gRPC startup check on port 9090

## Development Guide

### Adding New RPC Methods

1. Update `api/v1/hello.proto` file
2. Run `./gradlew generateProto` to regenerate code
3. Implement new method in `HelloServiceImpl`
4. Add corresponding unit tests

### Code Standards

- Follow Java code conventions
- Use meaningful variable and method names
- Add appropriate comments and documentation
- Write unit tests to cover core logic

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Find process using the port
   lsof -i :9090
   
   # Terminate process
   kill -9 <PID>
   ```

2. **Protobuf Generation Fails**
   ```bash
   # Clean and regenerate
   ./gradlew clean generateProto
   ```

3. **Build Fails**
   ```bash
   # View detailed error information
   ./gradlew build --stacktrace
   ```

## Resources

- [API Definition](../../api/v1/hello.proto)
- [Design Document](../../.kiro/specs/monorepo-hello-todo/design.md)
- [Requirements Document](../../.kiro/specs/monorepo-hello-todo/requirements.md)
- [Monorepo Documentation](../../docs/README.md)
- [gRPC Documentation](https://grpc.io/docs/)
- [Spring Boot gRPC Starter](https://github.com/grpc-ecosystem/grpc-spring)

## Support

For questions or issues:
- Check the monorepo documentation
- Contact backend-java-team
- Review existing services for examples
