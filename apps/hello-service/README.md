# Hello Service

Hello Service 是一个基于 Java/Spring Boot 和 gRPC 的问候服务，提供个性化的问候消息功能。

## 技术栈

- **Java**: 17
- **Spring Boot**: 3.5.0
- **gRPC**: 1.60.0
- **Protobuf**: 3.25.1
- **构建工具**: Gradle 8.14.3

## 项目结构

```
hello-service/
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/pingxin/cuckoo/hello/
│   │   │       ├── HelloServiceApplication.java    # 主应用类
│   │   │       └── service/
│   │   │           └── HelloServiceImpl.java       # gRPC 服务实现
│   │   └── resources/
│   │       └── application.yml                     # 应用配置
│   └── test/
├── k8s/                                            # Kubernetes 资源
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
├── build.gradle                                    # Gradle 构建配置
├── Dockerfile                                      # Docker 镜像构建
└── catalog-info.yaml                               # Backstage 服务目录
```

## API 定义

服务基于 Protobuf 定义，位于 `../../api/v1/hello.proto`：

```protobuf
service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
```

### 功能说明

- **输入**: 用户姓名（可选）
- **输出**: 个性化问候消息
- **规则**:
  - 如果提供姓名：返回 "Hello, {name}!"
  - 如果未提供姓名或为空：返回 "Hello, World!"

## 本地开发

### 前置条件

- Java 17+
- Gradle（使用 gradlew 包装器）

### 生成 Protobuf 代码

```bash
./gradlew generateProto
```

### 构建项目

```bash
# 构建（跳过测试）
./gradlew build -x test

# 构建（包含测试）
./gradlew build
```

### 运行服务

```bash
./gradlew bootRun
```

服务将在端口 **9090** 上启动 gRPC 服务器。

### 测试服务

使用 grpcurl 测试服务：

```bash
# 安装 grpcurl
brew install grpcurl

# 调用 SayHello 方法
grpcurl -plaintext -d '{"name": "Alice"}' localhost:9090 api.v1.HelloService/SayHello

# 预期输出
{
  "message": "Hello, Alice!"
}

# 测试空名字
grpcurl -plaintext -d '{"name": ""}' localhost:9090 api.v1.HelloService/SayHello

# 预期输出
{
  "message": "Hello, World!"
}
```

## Docker 部署

### 构建 Docker 镜像

```bash
docker build -t hello-service:latest .
```

### 运行 Docker 容器

```bash
docker run -p 9090:9090 hello-service:latest
```

## Kubernetes 部署

### 部署到 K8s 集群

```bash
# 应用所有 K8s 资源
kubectl apply -f k8s/

# 查看部署状态
kubectl get pods -l app=hello-service
kubectl get svc hello-service

# 查看日志
kubectl logs -l app=hello-service -f
```

### 访问服务

在 K8s 集群内，服务可通过以下地址访问：

```
hello-service:9090
```

## 配置

### application.yml

主要配置项：

```yaml
grpc:
  server:
    port: 9090              # gRPC 服务器端口

spring:
  application:
    name: hello-service     # 应用名称

logging:
  level:
    root: INFO
    com.pingxin.cuckoo: DEBUG
```

### 环境变量

- `SPRING_PROFILES_ACTIVE`: Spring 配置文件（如 `production`）
- `GRPC_SERVER_PORT`: gRPC 服务器端口（默认 9090）
- `JAVA_OPTS`: JVM 参数

## 监控和健康检查

### 健康检查

K8s 配置了以下探针：

- **Liveness Probe**: gRPC 健康检查，端口 9090
- **Readiness Probe**: gRPC 就绪检查，端口 9090
- **Startup Probe**: gRPC 启动检查，端口 9090

## 开发指南

### 添加新的 RPC 方法

1. 更新 `api/v1/hello.proto` 文件
2. 运行 `./gradlew generateProto` 重新生成代码
3. 在 `HelloServiceImpl` 中实现新方法
4. 添加相应的单元测试

### 代码规范

- 遵循 Java 代码规范
- 使用有意义的变量和方法名
- 添加适当的注释和文档
- 编写单元测试覆盖核心逻辑

## 故障排查

### 常见问题

1. **端口已被占用**
   ```bash
   # 查找占用端口的进程
   lsof -i :9090
   
   # 终止进程
   kill -9 <PID>
   ```

2. **Protobuf 生成失败**
   ```bash
   # 清理并重新生成
   ./gradlew clean generateProto
   ```

3. **构建失败**
   ```bash
   # 查看详细错误信息
   ./gradlew build --stacktrace
   ```

## 相关链接

- [API 定义](../../api/v1/hello.proto)
- [设计文档](../../.kiro/specs/monorepo-hello-todo/design.md)
- [需求文档](../../.kiro/specs/monorepo-hello-todo/requirements.md)
- [gRPC 文档](https://grpc.io/docs/)
- [Spring Boot gRPC Starter](https://github.com/grpc-ecosystem/grpc-spring)

## 许可证

[添加许可证信息]
