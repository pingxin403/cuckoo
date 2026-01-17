# TODO Service

TODO 管理服务，使用 Go 和 gRPC 实现。

## 功能

- 创建 TODO 项
- 列出所有 TODO 项
- 更新 TODO 项
- 删除 TODO 项
- 服务间通信（调用 Hello Service）

## 技术栈

- Go 1.21+
- gRPC
- Protocol Buffers
- 内存存储（可扩展为持久化存储）

## 本地开发

### 前置条件

- Go 1.21 或更高版本
- Protocol Buffers 编译器（protoc）

### 运行服务

```bash
# 从项目根目录
cd apps/todo-service

# 安装依赖
go mod download

# 运行服务
go run .
```

服务将在端口 9091 上监听。

### 环境变量

- `PORT`: gRPC 服务器端口（默认: 9091）
- `HELLO_SERVICE_ADDR`: Hello 服务地址（默认: localhost:9090）

### 构建

```bash
# 构建二进制文件
go build -o bin/todo-service .

# 运行
./bin/todo-service
```

### Docker

```bash
# 从项目根目录构建镜像
docker build -t todo-service:latest apps/todo-service

# 运行容器
docker run -p 9091:9091 \
  -e HELLO_SERVICE_ADDR=host.docker.internal:9090 \
  todo-service:latest
```

## API

服务实现了以下 gRPC 方法：

- `CreateTodo`: 创建新的 TODO 项
- `ListTodos`: 获取所有 TODO 项
- `UpdateTodo`: 更新现有 TODO 项
- `DeleteTodo`: 删除 TODO 项

详细的 API 定义请参考 `api/v1/todo.proto`。

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./service
go test ./storage
```

## 项目结构

```
apps/todo-service/
├── main.go              # 主入口
├── service/             # gRPC 服务实现
│   └── todo_service.go
├── storage/             # 存储层
│   └── memory_store.go
├── client/              # Hello 服务客户端
│   └── hello_client.go
├── gen/                 # 生成的 Protobuf 代码
│   ├── hellopb/
│   └── todopb/
├── k8s/                 # Kubernetes 资源
│   ├── deployment.yaml
│   └── service.yaml
├── Dockerfile           # Docker 镜像构建
└── README.md
```

## 部署

### Kubernetes

```bash
# 应用 Kubernetes 资源
kubectl apply -f k8s/

# 检查部署状态
kubectl get pods -l app=todo-service
kubectl get svc todo-service
```

## 开发指南

### 添加新功能

1. 更新 `api/v1/todo.proto` 定义新的消息或方法
2. 运行 `make gen-proto-go` 重新生成代码
3. 在 `service/todo_service.go` 中实现新方法
4. 添加相应的测试

### 代码规范

- 遵循 Go 标准代码风格
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查

## 故障排除

### 服务无法启动

- 检查端口 9091 是否被占用
- 确认 Hello Service 在 9090 端口运行（如果需要服务间通信）

### 连接 Hello Service 失败

- 检查 `HELLO_SERVICE_ADDR` 环境变量设置
- 确认 Hello Service 正在运行并可访问

## 许可证

[添加许可证信息]
