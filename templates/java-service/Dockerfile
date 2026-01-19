# Multi-stage Dockerfile for Java/Spring Boot Service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/{{SERVICE_NAME}}/Dockerfile -t {{SERVICE_NAME}}:latest .
# IMPORTANT: Proto code must be generated before building Docker image

# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# Copy Gradle configuration files
COPY apps/{{SERVICE_NAME}}/build.gradle apps/{{SERVICE_NAME}}/settings.gradle apps/{{SERVICE_NAME}}/gradlew ./
COPY apps/{{SERVICE_NAME}}/gradle ./gradle

# Download dependencies (cached layer)
RUN ./gradlew dependencies --no-daemon || true

# Copy source code and pre-generated proto code
COPY apps/{{SERVICE_NAME}}/ ./

# Build application (skip proto generation, use pre-generated code)
RUN ./gradlew build -x test -x generateProto --no-daemon

# Stage 2: Runtime stage
# Use multi-arch base image that works on both ARM64 (local dev) and AMD64 (production)
FROM eclipse-temurin:17-jre-jammy

WORKDIR /app

# Create non-root user
RUN groupadd -r spring && useradd -r -g spring spring
USER spring:spring

# Copy JAR file from build stage
COPY --from=build /app/build/libs/*.jar app.jar

# Expose gRPC port
EXPOSE {{GRPC_PORT}}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD pgrep -f "java.*app.jar" || exit 1

# Start application with JVM optimizations
ENTRYPOINT ["java", \
  "-XX:+UseContainerSupport", \
  "-XX:MaxRAMPercentage=75.0", \
  "-XX:+UseG1GC", \
  "-jar", "app.jar"]
