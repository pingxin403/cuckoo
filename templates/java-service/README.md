# Java/Spring Boot Service Template

This template provides a standardized structure for creating new Java/Spring Boot gRPC services in the monorepo.

## Features

- Spring Boot 3.x with Java 17
- gRPC server with grpc-spring-boot-starter
- Protobuf code generation
- Kubernetes deployment configurations
- Backstage service catalog integration
- Docker multi-stage build
- Health checks and monitoring

## Quick Start

### 1. Copy Template

```bash
# From the monorepo root
cp -r templates/java-service apps/your-service-name
cd apps/your-service-name
```

### 2. Customize Configuration

Replace the following placeholders throughout the project:

- `{{SERVICE_NAME}}` → Your service name (e.g., `user-service`)
- `{{SERVICE_DESCRIPTION}}` → Brief description of your service
- `{{GRPC_PORT}}` → gRPC port number (e.g., `9092`)
- `{{PACKAGE_NAME}}` → Java package name (e.g., `com.myorg.user`)
- `{{PROTO_FILE}}` → Protobuf file name (e.g., `user.proto`)
- `{{TEAM_NAME}}` → Owning team name (e.g., `backend-team`)

### 3. Update Files

#### build.gradle
```gradle
group = 'com.myorg'
version = '0.0.1-SNAPSHOT'
description = '{{SERVICE_DESCRIPTION}}'

sourceSets {
    main {
        proto {
            srcDir '../../api/v1'
            include '{{PROTO_FILE}}'
        }
    }
}
```

#### settings.gradle
```gradle
rootProject.name = '{{SERVICE_NAME}}'
```

#### application.yml
```yaml
grpc:
  server:
    port: {{GRPC_PORT}}

spring:
  application:
    name: {{SERVICE_NAME}}
```

#### Rename Java Package
```bash
# Rename the package directory
mv src/main/java/com/myorg/template src/main/java/{{PACKAGE_PATH}}

# Update package declarations in all Java files
# Replace: package com.myorg.template
# With: package {{PACKAGE_NAME}}
```

### 4. Define Protobuf API

Create your service's Protobuf definition in `api/v1/{{PROTO_FILE}}`:

```protobuf
syntax = "proto3";

package api.v1;

option go_package = "github.com/myorg/myrepo/api/v1/{{SERVICE_NAME}}pb";
option java_package = "{{PACKAGE_NAME}}.api.v1";
option java_multiple_files = true;

service {{ServiceName}}Service {
  rpc YourMethod(YourRequest) returns (YourResponse);
}

message YourRequest {
  string field = 1;
}

message YourResponse {
  string result = 1;
}
```

### 5. Generate Protobuf Code

```bash
# From monorepo root
make gen-proto-java

# Or from service directory
./gradlew generateProto
```

### 6. Implement Service Logic

Update `src/main/java/{{PACKAGE_PATH}}/service/{{ServiceName}}ServiceImpl.java`:

```java
package {{PACKAGE_NAME}}.service;

import {{PACKAGE_NAME}}.api.v1.*;
import io.grpc.stub.StreamObserver;
import net.devh.boot.grpc.server.service.GrpcService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@GrpcService
public class {{ServiceName}}ServiceImpl extends {{ServiceName}}ServiceGrpc.{{ServiceName}}ServiceImplBase {
    
    private static final Logger logger = LoggerFactory.getLogger({{ServiceName}}ServiceImpl.class);
    
    @Override
    public void yourMethod(YourRequest request, StreamObserver<YourResponse> responseObserver) {
        // Implement your logic here
        logger.info("Processing request: {}", request);
        
        YourResponse response = YourResponse.newBuilder()
                .setResult("Your result")
                .build();
        
        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
```

### 7. Update Kubernetes Resources

Update the following files in `k8s/`:

- `deployment.yaml`: Update service name, port, and resource limits
- `service.yaml`: Update service name and port
- `configmap.yaml`: Update configuration values

### 8. Update Backstage Catalog

Edit `catalog-info.yaml`:

```yaml
metadata:
  name: {{SERVICE_NAME}}
  description: {{SERVICE_DESCRIPTION}}
  tags:
    - java
    - spring-boot
    - grpc
spec:
  owner: {{TEAM_NAME}}
  providesApis:
    - {{SERVICE_NAME}}-api
```

### 9. Build and Test

```bash
# Build the service
./gradlew clean build

# Run tests
./gradlew test

# Run locally
./gradlew bootRun

# Build Docker image
docker build -t {{SERVICE_NAME}}:latest .
```

### 10. Add to Monorepo Build

Update the root `Makefile`:

```makefile
build-{{SERVICE_NAME}}:
	@echo "Building {{SERVICE_NAME}}..."
	cd apps/{{SERVICE_NAME}} && ./gradlew clean build

test-{{SERVICE_NAME}}:
	@echo "Testing {{SERVICE_NAME}}..."
	cd apps/{{SERVICE_NAME}} && ./gradlew test
```

## Project Structure

```
your-service-name/
├── build.gradle              # Gradle build configuration
├── settings.gradle           # Gradle settings
├── gradlew                   # Gradle wrapper script
├── Dockerfile                # Multi-stage Docker build
├── catalog-info.yaml         # Backstage service catalog
├── README.md                 # Service documentation
├── k8s/                      # Kubernetes resources
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
└── src/
    ├── main/
    │   ├── java/
    │   │   └── {{PACKAGE_PATH}}/
    │   │       ├── {{ServiceName}}Application.java
    │   │       └── service/
    │   │           └── {{ServiceName}}ServiceImpl.java
    │   └── resources/
    │       └── application.yml
    └── test/
        └── java/
            └── {{PACKAGE_PATH}}/
                └── {{ServiceName}}ApplicationTests.java
```

## Dependencies

The template includes:

- **Spring Boot 3.5.0**: Application framework
- **grpc-spring-boot-starter 3.1.0**: gRPC server integration
- **gRPC 1.60.0**: gRPC runtime
- **Protobuf 3.25.1**: Protocol Buffers
- **JUnit 5**: Unit testing
- **jqwik 1.8.2**: Property-based testing

## Configuration

### Application Properties

Key configuration in `application.yml`:

```yaml
grpc:
  server:
    port: {{GRPC_PORT}}              # gRPC server port
    
spring:
  application:
    name: {{SERVICE_NAME}}           # Service name

logging:
  level:
    root: INFO                       # Root log level
    {{PACKAGE_NAME}}: DEBUG          # Service log level
```

### Environment Variables

Supported environment variables:

- `SPRING_PROFILES_ACTIVE`: Active Spring profile (default: `default`)
- `GRPC_SERVER_PORT`: Override gRPC port
- `JAVA_OPTS`: JVM options

## Testing

### Unit Tests

Run unit tests with coverage:

```bash
# Run tests
./gradlew test

# Run tests with coverage report
./gradlew test jacocoTestReport

# Verify coverage thresholds (80% overall, 90% service classes)
./gradlew test jacocoTestCoverageVerification
```

Coverage reports are generated at:
- HTML: `build/reports/jacoco/test/html/index.html`
- XML: `build/reports/jacoco/test/jacocoTestReport.xml`

### Coverage Requirements

The template enforces test coverage thresholds:
- **Overall coverage**: 80% minimum
- **Service classes**: 90% minimum

These thresholds are verified in CI and will fail the build if not met.

### Writing Tests

Example unit test structure:

```java
@ExtendWith(MockitoExtension.class)
@DisplayName("Your Service Tests")
class YourServiceImplTest {
    
    @InjectMocks
    private YourServiceImpl yourService;
    
    @Mock
    private StreamObserver<YourResponse> responseObserver;
    
    @Test
    @DisplayName("Should handle valid request successfully")
    void testValidRequest() {
        // Arrange
        YourRequest request = YourRequest.newBuilder()
            .setField("test-value")
            .build();
        
        // Act
        yourService.yourMethod(request, responseObserver);
        
        // Assert
        ArgumentCaptor<YourResponse> captor = ArgumentCaptor.forClass(YourResponse.class);
        verify(responseObserver).onNext(captor.capture());
        verify(responseObserver).onCompleted();
        
        YourResponse response = captor.getValue();
        assertThat(response.getResult()).isNotEmpty();
    }
}
```

See `src/test/java/com/myorg/template/service/TemplateServiceImplTest.java` for a complete example.

### Property-Based Tests

The template includes jqwik for property-based testing. Example:

```java
import net.jqwik.api.*;

class YourServicePropertyTest {
    
    @Property
    void yourProperty(@ForAll String input) {
        // Test property across many inputs
        YourRequest request = YourRequest.newBuilder()
            .setField(input)
            .build();
        
        // Verify property holds for all inputs
        assertThat(processRequest(request)).isNotNull();
    }
}
```

### Integration Tests

```bash
./gradlew integrationTest
```

For more details, see the [Testing Guide](../../docs/TESTING_GUIDE.md).

## Docker

### Build Image

```bash
docker build -t {{SERVICE_NAME}}:latest .
```

### Run Container

```bash
docker run -p {{GRPC_PORT}}:{{GRPC_PORT}} {{SERVICE_NAME}}:latest
```

## Kubernetes Deployment

### Deploy to Cluster

```bash
kubectl apply -f k8s/
```

### Check Status

```bash
kubectl get pods -l app={{SERVICE_NAME}}
kubectl logs -f deployment/{{SERVICE_NAME}}
```

## Best Practices

1. **Keep Services Small**: Focus on a single domain or capability
2. **Use Protobuf**: Define all APIs in Protobuf for type safety
3. **Add Tests**: Write both unit tests and property-based tests
4. **Document APIs**: Add clear comments to Protobuf definitions
5. **Monitor Health**: Implement health checks and metrics
6. **Handle Errors**: Use appropriate gRPC status codes
7. **Log Appropriately**: Use structured logging with context
8. **Version APIs**: Use semantic versioning for breaking changes

## Troubleshooting

### Protobuf Generation Fails

```bash
# Clean and regenerate
./gradlew clean generateProto
```

### Port Already in Use

Change the port in `application.yml` or set environment variable:

```bash
GRPC_SERVER_PORT=9093 ./gradlew bootRun
```

### Build Fails

```bash
# Clean build directory
./gradlew clean

# Update dependencies
./gradlew dependencies --refresh-dependencies
```

## Additional Resources

- [Spring Boot Documentation](https://spring.io/projects/spring-boot)
- [gRPC Java Documentation](https://grpc.io/docs/languages/java/)
- [grpc-spring-boot-starter](https://github.com/grpc-ecosystem/grpc-spring)
- [Protobuf Guide](https://protobuf.dev/)
- [Backstage Service Catalog](https://backstage.io/docs/features/software-catalog/)

## Support

For questions or issues:
- Check the monorepo root README
- Contact the platform team
- Review existing services for examples
