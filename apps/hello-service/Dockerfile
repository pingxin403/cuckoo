# Multi-stage Dockerfile for Java/Spring Boot Service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/hello-service/Dockerfile -t hello-service:latest .
# Proto code is generated inside Docker to avoid Gradle path remapping issues

# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# Copy Gradle configuration files
COPY apps/hello-service/build.gradle apps/hello-service/settings.gradle apps/hello-service/gradlew ./
COPY apps/hello-service/gradle ./gradle

# Download dependencies (cached layer)
RUN ./gradlew dependencies --no-daemon || true

# Copy source code first
COPY apps/hello-service/ ./

# Copy proto files into project directory (avoid cross-directory references)
RUN mkdir -p src/main/proto
COPY api/v1/hello.proto src/main/proto/

# Temporarily modify build.gradle to use local proto path
RUN sed -i "s|srcDir '../../api/v1'|srcDir 'src/main/proto'|g" build.gradle

# Build application
RUN ./gradlew build -x test --no-daemon --no-configuration-cache

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
