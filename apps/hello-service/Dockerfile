# Multi-stage Dockerfile for Java/Spring Boot Service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/hello-service/Dockerfile -t hello-service:latest .

# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# Install protoc and plugins for proto generation
ARG PROTOC_VERSION=28.3
ARG PROTOC_GEN_GRPC_JAVA_VERSION=1.60.0

RUN apt-get update && apt-get install -y curl unzip && \
    # Install protoc
    PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP} && \
    unzip -o ${PROTOC_ZIP} -d /usr/local bin/protoc && \
    unzip -o ${PROTOC_ZIP} -d /usr/local 'include/*' && \
    rm -f ${PROTOC_ZIP} && \
    # Verify installation
    protoc --version && \
    # Clean up
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy Gradle configuration files
COPY apps/hello-service/build.gradle apps/hello-service/settings.gradle apps/hello-service/gradlew ./
COPY apps/hello-service/gradle ./gradle

# Copy proto files (Single Source of Truth)
COPY api/v1 /api/v1

# Download dependencies (cached layer)
RUN ./gradlew dependencies --no-daemon || true

# Copy source code
COPY apps/hello-service/src ./src

# Generate proto code and build application
RUN ./gradlew generateProto build -x test --no-daemon

# Stage 2: Runtime stage
FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -S spring && adduser -S spring -G spring
USER spring:spring

# Copy JAR file from build stage
COPY --from=build /app/build/libs/*.jar app.jar

# Expose gRPC port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD pgrep -f "java.*app.jar" || exit 1

# Start application with JVM optimizations
ENTRYPOINT ["java", \
  "-XX:+UseContainerSupport", \
  "-XX:MaxRAMPercentage=75.0", \
  "-XX:+UseG1GC", \
  "-jar", "app.jar"]
