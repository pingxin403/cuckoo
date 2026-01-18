# API Contract Layer

本目录包含所有服务的 Protobuf 接口定义，作为前后端通信的统一契约。

## API 版本

当前版本：**v1**

## 包含的服务

### Hello Service (`hello.proto`)

- **服务名称**: `api.v1.HelloService`
- **功能**: 提供个性化问候功能
- **端口**: 9090 (gRPC)
- **实现语言**: Java/Spring Boot
- **实现位置**: `apps/hello-service/`

#### API 定义

```protobuf
service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}

message HelloRequest {
  string name = 1;  // 用户姓名，可选
}

message HelloResponse {
  string message = 1;  // 问候消息
}
```

#### 使用示例

**Go 客户端**:
```go
import (
    hellopb "github.com/myorg/myrepo/gen/hellopb"
    "google.golang.org/grpc"
)

conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
client := hellopb.NewHelloServiceClient(conn)

req := &hellopb.HelloRequest{Name: "Alice"}
resp, _ := client.SayHello(context.Background(), req)
fmt.Println(resp.Message) // Output: Hello, Alice!
```

**Java 客户端**:
```java
import com.myorg.api.v1.*;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 9090)
    .usePlaintext()
    .build();

HelloServiceGrpc.HelloServiceBlockingStub stub = 
    HelloServiceGrpc.newBlockingStub(channel);

HelloRequest request = HelloRequest.newBuilder()
    .setName("Alice")
    .build();

HelloResponse response = stub.sayHello(request);
System.out.println(response.getMessage()); // Output: Hello, Alice!
```

**TypeScript/React 客户端**:
```typescript
import { HelloServiceClient } from '@/gen/hello';

const client = new HelloServiceClient('/api/hello');

const request = { name: 'Alice' };
const response = await client.sayHello(request, {});
console.log(response.message); // Output: Hello, Alice!
```

#### 行为说明

- 如果提供了 `name`，返回 "Hello, {name}!"
- 如果 `name` 为空或仅包含空格，返回 "Hello, World!"

### TODO Service (`todo.proto`)

- **服务名称**: `api.v1.TodoService`
- **功能**: TODO 任务管理（CRUD 操作）
- **端口**: 9091 (gRPC)
- **实现语言**: Go
- **实现位置**: `apps/todo-service/`

#### API 定义

```protobuf
service TodoService {
  rpc CreateTodo(CreateTodoRequest) returns (CreateTodoResponse);
  rpc ListTodos(ListTodosRequest) returns (ListTodosResponse);
  rpc UpdateTodo(UpdateTodoRequest) returns (UpdateTodoResponse);
  rpc DeleteTodo(DeleteTodoRequest) returns (DeleteTodoResponse);
}

message Todo {
  string id = 1;                              // 唯一标识符（UUID）
  string title = 2;                           // 标题（必填）
  string description = 3;                     // 描述（可选）
  bool completed = 4;                         // 完成状态
  google.protobuf.Timestamp created_at = 5;   // 创建时间
  google.protobuf.Timestamp updated_at = 6;   // 更新时间
}
```

#### 使用示例

**创建 TODO**:
```go
import todopb "github.com/myorg/myrepo/gen/todopb"

req := &todopb.CreateTodoRequest{
    Title:       "Buy groceries",
    Description: "Milk, eggs, bread",
}
resp, _ := client.CreateTodo(context.Background(), req)
fmt.Printf("Created TODO with ID: %s\n", resp.Todo.Id)
```

**列出所有 TODO**:
```go
req := &todopb.ListTodosRequest{}
resp, _ := client.ListTodos(context.Background(), req)
for _, todo := range resp.Todos {
    fmt.Printf("- [%v] %s\n", todo.Completed, todo.Title)
}
```

**更新 TODO**:
```go
req := &todopb.UpdateTodoRequest{
    Id:          "todo-id-here",
    Title:       "Buy groceries (updated)",
    Description: "Milk, eggs, bread, cheese",
    Completed:   true,
}
resp, _ := client.UpdateTodo(context.Background(), req)
```

**删除 TODO**:
```go
req := &todopb.DeleteTodoRequest{Id: "todo-id-here"}
resp, _ := client.DeleteTodo(context.Background(), req)
fmt.Printf("Deleted: %v\n", resp.Success)
```

#### 错误处理

服务使用标准 gRPC 状态码：

- `INVALID_ARGUMENT`: 输入验证失败（如空标题）
- `NOT_FOUND`: TODO 项不存在
- `INTERNAL`: 内部服务器错误

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
5. **错误处理**: 使用标准 gRPC 状态码
6. **字段验证**: 在服务实现中验证输入，不依赖客户端验证

## 测试 API

### 使用 grpcurl

```bash
# 列出所有服务
grpcurl -plaintext localhost:9090 list

# 列出服务的方法
grpcurl -plaintext localhost:9090 list api.v1.HelloService

# 调用 Hello Service
grpcurl -plaintext -d '{"name": "Alice"}' \
  localhost:9090 api.v1.HelloService/SayHello

# 调用 TODO Service - 创建 TODO
grpcurl -plaintext -d '{"title": "Test", "description": "Test TODO"}' \
  localhost:9091 api.v1.TodoService/CreateTodo

# 列出所有 TODO
grpcurl -plaintext -d '{}' \
  localhost:9091 api.v1.TodoService/ListTodos
```

### 使用 Postman

1. 创建新的 gRPC 请求
2. 输入服务地址：`localhost:9090` 或 `localhost:9091`
3. 导入 `.proto` 文件
4. 选择方法并填写请求数据
5. 发送请求

## 常见问题

### Q: 如何添加新的 RPC 方法？

A: 在 `.proto` 文件中添加新方法，然后：
1. 运行 `make gen-proto` 重新生成代码
2. 在服务实现中添加新方法的实现
3. 更新测试和文档

### Q: 如何处理 API 版本升级？

A: 对于不兼容的变更：
1. 创建新的 API 版本目录（如 `api/v2/`）
2. 复制并修改 `.proto` 文件
3. 更新服务以支持新版本
4. 保持旧版本运行一段时间以便客户端迁移

### Q: 生成的代码应该提交到版本控制吗？

A: 是的。虽然代码是生成的，但提交到版本控制可以：
- 确保构建的确定性
- 方便代码审查
- 避免构建环境差异导致的问题

### Q: 如何在前端使用 gRPC？

A: 前端使用 gRPC-Web 协议：
1. 通过 Envoy/Higress 网关访问后端
2. 网关将 gRPC-Web 转换为 gRPC
3. 使用生成的 TypeScript 客户端代码

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
