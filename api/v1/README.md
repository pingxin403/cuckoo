# API Contract Layer

本目录包含所有服务的 Protobuf 接口定义，作为前后端通信的统一契约。

## API 版本

当前版本：**v1**

## 包含的服务

### Auth Service (`auth.proto`)

- **服务名称**: `auth.v1.AuthService`
- **功能**: JWT 认证和令牌管理
- **端口**: 9095 (gRPC)
- **实现语言**: Go
- **实现位置**: `apps/auth-service/` (待实现)

#### API 定义

```protobuf
service AuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
}

message ValidateTokenRequest {
  string access_token = 1;  // JWT 访问令牌
}

message ValidateTokenResponse {
  bool valid = 1;                              // 令牌是否有效
  string user_id = 2;                          // 用户 ID
  string device_id = 3;                        // 设备 ID (UUID v4 格式)
  google.protobuf.Timestamp expires_at = 4;    // 过期时间
  AuthErrorCode error_code = 5;                // 错误码
  string error_message = 6;                    // 错误消息
}
```

#### 使用示例

**验证令牌**:
```go
import authpb "github.com/pingxin403/cuckoo/apps/auth-service/gen/authpb"

req := &authpb.ValidateTokenRequest{
    AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
}
resp, _ := client.ValidateToken(context.Background(), req)
if resp.Valid {
    fmt.Printf("User ID: %s, Device ID: %s\n", resp.UserId, resp.DeviceId)
}
```

**刷新令牌**:
```go
req := &authpb.RefreshTokenRequest{
    RefreshToken: "refresh_token_here",
}
resp, _ := client.RefreshToken(context.Background(), req)
fmt.Printf("New access token: %s\n", resp.AccessToken)
```

#### 错误码

- `AUTH_ERROR_CODE_TOKEN_EXPIRED`: 令牌已过期
- `AUTH_ERROR_CODE_INVALID_SIGNATURE`: 签名无效
- `AUTH_ERROR_CODE_MALFORMED_TOKEN`: 令牌格式错误
- `AUTH_ERROR_CODE_MISSING_CLAIMS`: 缺少必需的声明字段
- `AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN`: 刷新令牌无效
- `AUTH_ERROR_CODE_INTERNAL_ERROR`: 内部服务器错误

### User Service (`user.proto`)

- **服务名称**: `user.v1.UserService`
- **功能**: 用户资料和群组成员管理
- **端口**: 9096 (gRPC)
- **实现语言**: Go
- **实现位置**: `apps/user-service/` (待实现)

#### API 定义

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);
  rpc GetGroupMembers(GetGroupMembersRequest) returns (GetGroupMembersResponse);
  rpc ValidateGroupMembership(ValidateGroupMembershipRequest) returns (ValidateGroupMembershipResponse);
}

message UserProfile {
  string user_id = 1;                          // 用户 ID
  string username = 2;                         // 用户名
  string display_name = 3;                     // 显示名称
  string avatar_url = 4;                       // 头像 URL
  UserStatus status = 5;                       // 用户状态
  google.protobuf.Timestamp created_at = 6;    // 创建时间
  google.protobuf.Timestamp updated_at = 7;    // 更新时间
}
```

#### 使用示例

**获取用户资料**:
```go
import userpb "github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"

req := &userpb.GetUserRequest{UserId: "user123"}
resp, _ := client.GetUser(context.Background(), req)
fmt.Printf("User: %s (%s)\n", resp.User.DisplayName, resp.User.Status)
```

**批量获取用户**:
```go
req := &userpb.BatchGetUsersRequest{
    UserIds: []string{"user1", "user2", "user3"},
}
resp, _ := client.BatchGetUsers(context.Background(), req)
for userId, user := range resp.Users {
    fmt.Printf("%s: %s\n", userId, user.DisplayName)
}
```

**获取群组成员**:
```go
req := &userpb.GetGroupMembersRequest{
    GroupId: "group123",
    Cursor:  "",      // 首页为空
    Limit:   100,     // 每页 100 条
}
resp, _ := client.GetGroupMembers(context.Background(), req)
fmt.Printf("Found %d members (total: %d)\n", len(resp.Members), resp.TotalCount)
```

**验证群组成员资格**:
```go
req := &userpb.ValidateGroupMembershipRequest{
    UserId:  "user123",
    GroupId: "group456",
}
resp, _ := client.ValidateGroupMembership(context.Background(), req)
if resp.IsMember {
    fmt.Printf("User is %s in group\n", resp.Member.Role)
}
```

#### 错误码

- `USER_ERROR_CODE_USER_NOT_FOUND`: 用户不存在
- `USER_ERROR_CODE_GROUP_NOT_FOUND`: 群组不存在
- `USER_ERROR_CODE_INVALID_REQUEST`: 请求参数无效
- `USER_ERROR_CODE_DATABASE_ERROR`: 数据库错误
- `USER_ERROR_CODE_TOO_MANY_IDS`: 批量请求 ID 过多（最多 100 个）
- `USER_ERROR_CODE_INTERNAL_ERROR`: 内部服务器错误

### IM Service (`im.proto`)

- **服务名称**: `im.v1.IMService`
- **功能**: 消息路由和投递
- **端口**: 9094 (gRPC)
- **实现语言**: Go
- **实现位置**: `apps/im-service/` (待实现)

#### API 定义

```protobuf
service IMService {
  rpc RoutePrivateMessage(RoutePrivateMessageRequest) returns (RoutePrivateMessageResponse);
  rpc RouteGroupMessage(RouteGroupMessageRequest) returns (RouteGroupMessageResponse);
  rpc GetMessageStatus(GetMessageStatusRequest) returns (GetMessageStatusResponse);
}

message RoutePrivateMessageRequest {
  string msg_id = 1;                           // 消息 ID (UUID)
  string sender_id = 2;                        // 发送者 ID
  string recipient_id = 3;                     // 接收者 ID
  string content = 4;                          // 消息内容（最多 10,000 字符）
  MessageType message_type = 5;                // 消息类型
  map<string, string> metadata = 6;            // 元数据
  google.protobuf.Timestamp client_timestamp = 7;  // 客户端时间戳
}
```

#### 使用示例

**发送私聊消息**:
```go
import impb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"

req := &impb.RoutePrivateMessageRequest{
    MsgId:       "msg-uuid-here",
    SenderId:    "user123",
    RecipientId: "user456",
    Content:     "Hello, how are you?",
    MessageType: impb.MessageType_MESSAGE_TYPE_TEXT,
    ClientTimestamp: timestamppb.Now(),
}
resp, _ := client.RoutePrivateMessage(context.Background(), req)
fmt.Printf("Message sent with sequence: %d, status: %s\n", 
    resp.SequenceNumber, resp.DeliveryStatus)
```

**发送群组消息**:
```go
req := &impb.RouteGroupMessageRequest{
    MsgId:       "msg-uuid-here",
    SenderId:    "user123",
    GroupId:     "group789",
    Content:     "Hello everyone!",
    MessageType: impb.MessageType_MESSAGE_TYPE_TEXT,
    ClientTimestamp: timestamppb.Now(),
}
resp, _ := client.RouteGroupMessage(context.Background(), req)
fmt.Printf("Delivered to %d online, %d offline members\n", 
    resp.OnlineMemberCount, resp.OfflineMemberCount)
```

**查询消息状态**:
```go
req := &impb.GetMessageStatusRequest{
    MsgId:            "msg-uuid-here",
    ConversationType: impb.ConversationType_CONVERSATION_TYPE_PRIVATE,
    ConversationId:   "user456",
}
resp, _ := client.GetMessageStatus(context.Background(), req)
fmt.Printf("Message status: %s\n", resp.DeliveryStatus)
```

#### 消息类型

- `MESSAGE_TYPE_TEXT`: 文本消息
- `MESSAGE_TYPE_IMAGE`: 图片消息
- `MESSAGE_TYPE_FILE`: 文件消息
- `MESSAGE_TYPE_AUDIO`: 音频消息
- `MESSAGE_TYPE_VIDEO`: 视频消息
- `MESSAGE_TYPE_LOCATION`: 位置消息
- `MESSAGE_TYPE_SYSTEM`: 系统通知

#### 投递状态

- `DELIVERY_STATUS_PENDING`: 消息处理中
- `DELIVERY_STATUS_DELIVERED`: 已投递到接收者
- `DELIVERY_STATUS_READ`: 接收者已读
- `DELIVERY_STATUS_FAILED`: 重试后投递失败
- `DELIVERY_STATUS_OFFLINE`: 已路由到离线通道

#### 错误码

- `IM_ERROR_CODE_INVALID_MESSAGE`: 消息格式无效
- `IM_ERROR_CODE_CONTENT_TOO_LONG`: 内容超过 10,000 字符
- `IM_ERROR_CODE_SENDER_NOT_FOUND`: 发送者不存在
- `IM_ERROR_CODE_RECIPIENT_NOT_FOUND`: 接收者不存在
- `IM_ERROR_CODE_GROUP_NOT_FOUND`: 群组不存在
- `IM_ERROR_CODE_NOT_GROUP_MEMBER`: 发送者不是群组成员
- `IM_ERROR_CODE_SENSITIVE_CONTENT`: 消息包含敏感词（已拦截）
- `IM_ERROR_CODE_SEQUENCE_ERROR`: 序列号生成器错误
- `IM_ERROR_CODE_REGISTRY_ERROR`: 注册表查询错误
- `IM_ERROR_CODE_KAFKA_ERROR`: Kafka 发布错误
- `IM_ERROR_CODE_DELIVERY_TIMEOUT`: 重试后投递超时
- `IM_ERROR_CODE_INTERNAL_ERROR`: 内部服务器错误

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
make proto

# 或单独生成（高级用法）
make gen-proto-go      # Go 代码
make gen-proto-java    # Java 代码
make gen-proto-ts      # TypeScript 代码
```

### 生成代码位置

- **Go**: 
  - `apps/todo-service/gen/todopb/` (TODO Service)
  - `apps/auth-service/gen/authpb/` (Auth Service - 待实现)
  - `apps/user-service/gen/userpb/` (User Service - 待实现)
  - `apps/im-service/gen/impb/` (IM Service - 待实现)
- **Java**: `apps/hello-service/src/main/java-gen/`
- **TypeScript**: `apps/web/src/gen/`

### 修改 API 契约的流程

1. 修改对应的 `.proto` 文件
2. 运行 `make proto` 重新生成代码
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

# 调用 Auth Service - 验证令牌
grpcurl -plaintext -d '{"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}' \
  localhost:9095 auth.v1.AuthService/ValidateToken

# 调用 User Service - 获取用户
grpcurl -plaintext -d '{"user_id": "user123"}' \
  localhost:9096 user.v1.UserService/GetUser

# 调用 IM Service - 发送私聊消息
grpcurl -plaintext -d '{"msg_id": "msg-123", "sender_id": "user1", "recipient_id": "user2", "content": "Hello", "message_type": "MESSAGE_TYPE_TEXT"}' \
  localhost:9094 im.v1.IMService/RoutePrivateMessage
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
1. 运行 `make proto` 重新生成代码
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
