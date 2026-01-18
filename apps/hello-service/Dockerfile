# 多阶段构建 Dockerfile for Hello Service
# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# 复制 Gradle 配置文件
COPY apps/hello-service/build.gradle apps/hello-service/settings.gradle apps/hello-service/gradlew ./
COPY apps/hello-service/gradle ./gradle

# 复制 API 定义文件（用于 Protobuf 生成）
COPY api/v1 /api/v1

# 复制源代码
COPY apps/hello-service/src ./src

# 构建应用（跳过测试以加快构建速度）
RUN ./gradlew clean build -x test --no-daemon

# Stage 2: Runtime stage
FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# 创建非 root 用户
RUN addgroup -S spring && adduser -S spring -G spring
USER spring:spring

# 从构建阶段复制 JAR 文件
COPY --from=build /app/build/libs/*.jar app.jar

# 暴露 gRPC 端口
EXPOSE 9090

# 健康检查（gRPC 健康检查需要额外配置，这里使用简单的进程检查）
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD pgrep -f "java.*app.jar" || exit 1

# 启动应用
ENTRYPOINT ["java", "-jar", "app.jar"]

# JVM 优化参数（可选）
# ENTRYPOINT ["java", \
#   "-XX:+UseContainerSupport", \
#   "-XX:MaxRAMPercentage=75.0", \
#   "-XX:+UseG1GC", \
#   "-jar", "app.jar"]
