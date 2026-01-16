# Monorepo Hello/TODO Services

A multi-language monorepo project demonstrating microservices architecture with Java/Spring Boot, Go, and React/TypeScript.

## 项目概述

本项目是一个多语言 Monorepo，包含以下服务：

- **Hello Service** (Java/Spring Boot) - 提供问候功能的 gRPC 服务
- **TODO Service** (Go) - 提供任务管理功能的 gRPC 服务
- **Web Application** (React/TypeScript) - 前端单页应用

所有服务通过 Protobuf 定义统一的 API 契约，使用 gRPC 进行通信。

## 项目结构

```
.
├── api/                    # API 契约层 (Protobuf 定义)
│   └── v1/
│       ├── hello.proto
│       └── todo.proto
├── apps/                   # 应用服务
│   ├── hello-service/      # Java/Spring Boot 服务
│   ├── todo-service/       # Go 服务
│   └── web/                # React 前端应用
├── libs/                   # 共享库
├── tools/                  # 构建工具和配置
│   ├── envoy/              # Envoy 代理配置
│   ├── higress/            # Higress 网关配置
│   └── k8s/                # Kubernetes 资源
├── scripts/                # 构建和开发脚本
│   └── dev.sh              # 开发模式启动脚本
├── templates/              # 服务模板
│   ├── java-service/       # Java 服务模板
│   └── go-service/         # Go 服务模板
├── Makefile                # 统一构建命令
└── README.md
```

## 快速开始

### 前置要求

- **Java**: JDK 17+
- **Go**: Go 1.21+
- **Node.js**: Node 18+
- **Protocol Buffers**: protoc 3.x
- **Docker**: (可选) 用于容器化部署
- **Kubernetes**: (可选) 用于生产部署

### 安装依赖

```bash
# 安装 Protobuf 编译器 (macOS)
brew install protobuf

# 安装 gRPC 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 生成代码

从 Protobuf 定义生成各语言的代码：

```bash
# 生成所有语言的代码
make gen-proto

# 或者单独生成
make gen-proto-go      # Go
make gen-proto-java    # Java
make gen-proto-ts      # TypeScript
```

### 本地开发

#### 方式 1: 使用开发脚本（推荐）

```bash
# 启动所有服务
./scripts/dev.sh
```

这将同时启动：
- Hello Service (端口 9090)
- TODO Service (端口 9091)
- Web Application (端口 5173)
- Envoy Proxy (端口 8080)

#### 方式 2: 手动启动各服务

```bash
# 终端 1: 启动 Hello Service
cd apps/hello-service
./mvnw spring-boot:run

# 终端 2: 启动 TODO Service
cd apps/todo-service
go run .

# 终端 3: 启动 Web Application
cd apps/web
npm install
npm run dev
```

访问 http://localhost:5173 查看前端应用。

### 构建

```bash
# 构建所有服务
make build

# 或者单独构建
make build-hello    # Hello Service
make build-todo     # TODO Service
make build-web      # Web Application
```

### 测试

```bash
# 运行所有测试
make test

# 或者单独测试
make test-hello     # Hello Service 测试
make test-todo      # TODO Service 测试
make test-web       # Web Application 测试
```

### Docker 构建

```bash
# 构建所有 Docker 镜像
make docker-build

# 或者单独构建
make docker-build-hello    # Hello Service 镜像
make docker-build-todo     # TODO Service 镜像
```

## 架构说明

### 通信模式

- **南北向流量** (North-South): 前端 → Higress 网关 → 后端服务
- **东西向流量** (East-West): 服务间直连 gRPC 通信

### API 契约

所有服务接口使用 Protobuf 定义在 `api/v1/` 目录：

- `hello.proto` - Hello 服务接口
- `todo.proto` - TODO 服务接口

### 服务端口

- Hello Service: 9090 (gRPC)
- TODO Service: 9091 (gRPC)
- Web Application: 5173 (开发模式)
- Envoy Proxy: 8080 (HTTP/gRPC-Web)

## 添加新服务

### 使用 Java 模板

```bash
# 复制模板
cp -r templates/java-service apps/my-new-service

# 修改配置
cd apps/my-new-service
# 编辑 pom.xml, application.yml 等
```

### 使用 Go 模板

```bash
# 复制模板
cp -r templates/go-service apps/my-new-service

# 修改配置
cd apps/my-new-service
# 编辑 go.mod, main.go 等
```

### 添加新 API

1. 在 `api/v1/` 目录创建新的 `.proto` 文件
2. 运行 `make gen-proto` 生成代码
3. 在服务中实现接口

## 部署

### Kubernetes 部署

```bash
# 使用 Kustomize 部署
kubectl apply -k k8s/overlays/production

# 验证部署
kubectl get pods
kubectl get services
kubectl get ingress
```

### 配置说明

- **Base**: `k8s/base/` - 基础配置
- **Overlays**: `k8s/overlays/production/` - 生产环境配置

## CI/CD

项目使用 GitHub Actions 进行持续集成：

- 代码提交时自动运行测试
- 验证 Protobuf 生成代码是否最新
- 构建 Docker 镜像并推送到镜像仓库
- 自动部署到 Kubernetes 集群

## 代码所有权

代码所有权定义在 `.github/CODEOWNERS` 文件中：

- API 契约层: @platform-team
- 前端应用: @frontend-team
- Java 服务: @backend-java-team
- Go 服务: @backend-go-team

## 开发规范

### 提交前检查

项目配置了 pre-commit hook，会自动检查：

- Protobuf 生成代码是否最新
- 代码格式是否符合规范

### Pull Request 流程

1. 创建功能分支
2. 提交代码并推送
3. 创建 Pull Request
4. 等待 CI 通过和代码审查
5. 合并到主分支

## 故障排查

### Protobuf 生成失败

```bash
# 确保 protoc 已安装
protoc --version

# 确保插件已安装
which protoc-gen-go
which protoc-gen-go-grpc
```

### 服务启动失败

```bash
# 检查端口是否被占用
lsof -i :9090
lsof -i :9091

# 查看服务日志
cd apps/hello-service && ./mvnw spring-boot:run
cd apps/todo-service && go run .
```

### 前端无法连接后端

确保 Envoy 代理正在运行，或者在 `vite.config.ts` 中配置了正确的代理设置。

## 更多信息

- [API 文档](api/v1/README.md)
- [架构设计](docs/architecture.md)
- [开发指南](docs/development.md)
- [部署指南](docs/deployment.md)

## 许可证

[MIT License](LICENSE)
