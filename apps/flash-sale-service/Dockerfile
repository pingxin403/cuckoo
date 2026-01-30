# Multi-stage Dockerfile for Java/Spring Boot Service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/flash-sale-service/Dockerfile -t flash-sale-service:latest .

# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /workspace

# Copy centralized proto generated code
COPY api/gen/java ./api/gen/java

# Copy flash-sale-service
COPY apps/flash-sale-service ./apps/flash-sale-service

# Build from the service directory
WORKDIR /workspace/apps/flash-sale-service

# Download dependencies (cached layer)
RUN ./gradlew dependencies --no-daemon || true

# Build application (uses pre-generated proto code from api/gen/java)
RUN ./gradlew build -x test --no-daemon --no-configuration-cache

# Stage 2: Runtime stage
# Use multi-arch base image that works on both ARM64 (local dev) and AMD64 (production)
FROM eclipse-temurin:17-jre-jammy

WORKDIR /app

# Create non-root user
RUN groupadd -r spring && useradd -r -g spring spring
USER spring:spring

# Copy JAR file from build stage
COPY --from=build /workspace/apps/flash-sale-service/build/libs/*.jar app.jar

# Expose gRPC port
EXPOSE 9094

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD pgrep -f "java.*app.jar" || exit 1

# Start application with JVM optimizations
ENTRYPOINT ["java", \
  "-XX:+UseContainerSupport", \
  "-XX:MaxRAMPercentage=75.0", \
  "-XX:+UseG1GC", \
  "-jar", "app.jar"]
