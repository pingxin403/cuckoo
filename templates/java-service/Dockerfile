# Multi-stage Dockerfile for Java/Spring Boot Service
# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# Copy Gradle configuration files
COPY build.gradle settings.gradle gradlew ./
COPY gradle ./gradle

# Copy API definitions (for Protobuf generation)
COPY ../../api/v1 /api/v1

# Copy source code (including generated Protobuf code)
COPY src ./src
COPY build/generated ./build/generated

# Build application (skip tests and protobuf generation for faster builds)
# Use -x generateProto to skip protobuf generation since code is already committed
RUN ./gradlew clean build -x test -x generateProto --no-daemon

# Stage 2: Runtime stage
FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -S spring && adduser -S spring -G spring
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
