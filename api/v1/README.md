# API Contract Layer

本目录包含所有服务的 Protobuf 接口定义，作为前后端通信的统一契约。

## API 版本

当前版本：**v1**

## 包含的服务

### Hello Service (`hello.proto`)
- **服务名称**: `api.v1.HelloService`
- **功能**: 提供问候功能
- **端口**: 9090 (gRPC)
- **实现语言**: Java/Spring Boot

### TODO Service (`todo.proto`)
- **服务名称**: `api.v1.TodoService`
- **功能**: TODO 任务管理（CRUD 操作）
- **端口**: 9091 (gRPC)
- **实现语言**: Go

## 使用方式

### 代码生成

从 Protobuf 定义生成各语言的代码：

```bash
# 生成所有语言的代码
make gen-proto

# 或单独生成
make gen-proto-go      # Go 代码
make gen-proto-java    # Java 代码
make gen-proto-ts      # TypeScript 代码
```

### 生成代码位置

- **Go**: `apps/todo-service/gen/`
- **Java**: `apps/hello-service/src/main/java-gen/`
- **TypeScript**: `apps/web/src/gen/`

### 修改 API 契约的流程

1. 修改对应的 `.proto` 文件
2. 运行 `make gen-proto` 重新生成代码
3. 更新服务实现以匹配新的接口
4. 提交 PR（包含 `.proto` 文件和生成的代码）

**注意**: 所有生成的代码都应该提交到版本控制，以确保构建的确定性。

## API 设计原则

1. **向后兼容**: 新增字段使用新的字段编号，不要修改已有字段
2. **清晰命名**: 使用描述性的消息和字段名称
3. **文档完整**: 为所有服务、消息和字段添加注释
4. **版本管理**: 重大变更时创建新版本（如 v2）

## gRPC 通信模式

### 南北向流量（North-South）
前端 → Higress Gateway → 后端服务
- 使用 gRPC-Web 协议
- 通过 API 网关统一入口

### 东西向流量（East-West）
服务间直连 gRPC 通信
- 基于 K8s Service DNS
- 避免网关成为瓶颈

## 相关文档

- [gRPC 官方文档](https://grpc.io/docs/)
- [Protocol Buffers 指南](https://protobuf.dev/)
- [项目根目录 README](../../README.md)
