# 多阶段构建 Dockerfile for Hello Service
# Stage 1: Build stage
FROM gradle:8.14.3-jdk17 AS build

WORKDIR /app

# 复制 Gradle 配置文件
COPY build.gradle settings.gradle gradlew ./
COPY gradle ./gradle

# 复制源代码（包括生成的 Protobuf 代码）
COPY src ./src
COPY build/generated ./build/generated

# 构建应用（跳过测试和 protobuf 生成以加快构建速度）
# 使用 -x generateProto 跳过 protobuf 生成，因为代码已经提交到 git
RUN ./gradlew clean build -x test -x generateProto --no-daemon

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
