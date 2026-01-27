# Design Document: Monorepo Hello/TODO Services

## Overview

本设计文档描述了一个多语言 Monorepo 项目的技术实现方案，包括项目初始化、API 契约定义、代码生成、服务实现和前端集成。该系统采用契约先行的设计理念，使用 Protobuf 作为统一的接口定义语言，支持 Java/Spring Boot（Hello 服务）、Go（TODO 服务）和 React/TypeScript（前端）三种技术栈的协同开发。

### Design Goals

1. 建立清晰的项目结构和代码组织方式
2. 实现基于 Protobuf 的类型安全的跨语言通信
3. 提供高效的开发体验和构建流程
4. 确保服务的独立性和可测试性
5. 支持未来的扩展和新服务的添加

### Key Design Decisions

1. **构建系统选择**: 使用 Makefile + 脚本的混合方案，而非纯 Bazel
   - 理由：降低学习曲线，保持各语言生态的原生工具链优势
   - 权衡：牺牲部分增量构建能力，换取更好的开发体验

2. **API 网关选择**: 使用 Higress 作为 K8s 原生网关
   - 理由：K8s 原生，支持 gRPC 和 HTTP，配置简单，性能优秀
   - 用途：南北向流量（前端到后端）
   - 替代方案：Envoy（更通用但配置复杂）、Nginx（需要额外配置）

3. **服务间通信**: 服务间直连 gRPC
   - 理由：避免网关成为瓶颈，降低延迟，简化架构
   - 原则：南北向走网关，东西向直连
   - 基于共享 Protobuf 契约保证类型安全
   - 实施超时、重试、熔断等弹性模式

4. **代码生成策略**: 各服务独立生成代码到自己的目录
   - 理由：保持服务独立性和自治性，避免跨服务依赖
   - 实现：每个服务在自己的 `gen/` 或 `src/main/java-gen/` 目录生成代码
   - 权衡：需要在多处生成相同的 Protobuf 定义，但保证了服务的完全自包含

5. **项目模板**: 提供标准化的服务模板
   - Java/Spring Boot 服务模板
   - Go 服务模板
   - 包含 catalog-info.yaml 用于服务注册和文档

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│                    (React Application)                       │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP/gRPC-Web
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Higress Gateway                            │
│              (K8s Native API Gateway)                        │
│              南北向流量 (North-South)                         │
└────────┬────────────────────────────────────┬───────────────┘
         │ gRPC                               │ gRPC
         ▼                                    ▼
┌──────────────────────┐          ┌──────────────────────┐
│   Hello Service      │          │   TODO Service       │
│   (Java/Spring Boot) │◄────────►│   (Go)               │
│   Port: 9090         │  gRPC    │   Port: 9091         │
│                      │  直连     │                      │
│                      │  东西向   │                      │
└──────────────────────┘  (E-W)   └──────────────────────┘

说明：
- 南北向（North-South）：前端 → Higress → 后端服务
- 东西向（East-West）：服务间直连 gRPC 通信
- 所有服务基于共享 Protobuf 契约
```

### Communication Patterns

1. **南北向流量（North-South）**
   - 前端通过 Higress 网关访问后端服务
   - Higress 提供 gRPC-Web 到 gRPC 的协议转换
   - 统一的入口点，便于实施安全策略和监控

2. **东西向流量（East-West）**
   - 服务间直接通过 gRPC 通信
   - 避免网关成为性能瓶颈
   - 基于 K8s Service 进行服务发现
   - 使用共享 Protobuf 定义保证类型安全

### Component Layers

1. **API Contract Layer** (`api/`)
   - Protobuf 定义文件
   - 版本化的接口规范
   - 跨语言的类型定义

2. **Service Layer** (`apps/`)
   - 独立的微服务实现
   - 各自的构建配置和依赖管理
   - 独立的生命周期和部署

3. **Frontend Layer** (`apps/web/`)
   - React 单页应用
   - TypeScript 类型安全
   - 通过 Envoy 代理访问后端服务

4. **Infrastructure Layer** (`tools/`, `scripts/`)
   - 构建脚本和工具
   - 开发环境配置
   - CI/CD 支持

## Components and Interfaces

### 1. API Contract Module

**Location**: `api/v1/`

**Protobuf Definitions**:

```protobuf
// api/v1/hello.proto
syntax = "proto3";

package api.v1;

option go_package = "github.com/myorg/myrepo/api/v1/hellopb";
option java_package = "com.myorg.api.v1";
option java_multiple_files = true;

// Hello 服务定义
service HelloService {
  // 发送问候
  rpc SayHello(HelloRequest) returns (HelloResponse);
}

// Hello 请求消息
message HelloRequest {
  string name = 1;  // 用户姓名，可以为空
}

// Hello 响应消息
message HelloResponse {
  string message = 1;  // 问候消息
}
```

```protobuf
// api/v1/todo.proto
syntax = "proto3";

package api.v1;

option go_package = "github.com/myorg/myrepo/api/v1/todopb";
option java_package = "com.myorg.api.v1";
option java_multiple_files = true;

import "google/protobuf/timestamp.proto";

// TODO 服务定义
service TodoService {
  // 创建 TODO 项
  rpc CreateTodo(CreateTodoRequest) returns (CreateTodoResponse);
  
  // 获取所有 TODO 项
  rpc ListTodos(ListTodosRequest) returns (ListTodosResponse);
  
  // 更新 TODO 项
  rpc UpdateTodo(UpdateTodoRequest) returns (UpdateTodoResponse);
  
  // 删除 TODO 项
  rpc DeleteTodo(DeleteTodoRequest) returns (DeleteTodoResponse);
}

// TODO 项数据模型
message Todo {
  string id = 1;                              // 唯一标识符
  string title = 2;                           // 标题
  string description = 3;                     // 描述
  bool completed = 4;                         // 是否完成
  google.protobuf.Timestamp created_at = 5;   // 创建时间
  google.protobuf.Timestamp updated_at = 6;   // 更新时间
}

message CreateTodoRequest {
  string title = 1;
  string description = 2;
}

message CreateTodoResponse {
  Todo todo = 1;
}

message ListTodosRequest {
  // 未来可扩展分页参数
}

message ListTodosResponse {
  repeated Todo todos = 1;
}

message UpdateTodoRequest {
  string id = 1;
  string title = 2;
  string description = 3;
  bool completed = 4;
}

message UpdateTodoResponse {
  Todo todo = 1;
}

message DeleteTodoRequest {
  string id = 1;
}

message DeleteTodoResponse {
  bool success = 1;
}
```

**Code Generation Configuration**:

为了保持服务的独立性和自治性，每个服务在自己的目录中生成 Protobuf 代码。

```makefile
# Makefile for code generation
.PHONY: gen-proto gen-proto-go gen-proto-java gen-proto-ts

gen-proto: gen-proto-go gen-proto-java gen-proto-ts

gen-proto-go:
	@echo "Generating Go code from Protobuf..."
	mkdir -p apps/todo-service/gen
	protoc --go_out=apps/todo-service/gen \
	       --go_opt=paths=source_relative \
	       --go-grpc_out=apps/todo-service/gen \
	       --go-grpc_opt=paths=source_relative \
	       -I api/v1 \
	       api/v1/*.proto

gen-proto-java:
	@echo "Generating Java code from Protobuf..."
	mkdir -p apps/hello-service/src/main/java-gen
	protoc --java_out=apps/hello-service/src/main/java-gen \
	       --grpc-java_out=apps/hello-service/src/main/java-gen \
	       -I api/v1 \
	       api/v1/hello.proto

gen-proto-ts:
	@echo "Generating TypeScript code from Protobuf..."
	mkdir -p apps/web/src/gen
	protoc --plugin=./node_modules/.bin/protoc-gen-ts_proto \
	       --ts_proto_out=apps/web/src/gen \
	       --ts_proto_opt=esModuleInterop=true \
	       --ts_proto_opt=outputServices=grpc-web \
	       -I api/v1 \
	       api/v1/*.proto

# CI 验证：确保生成代码与提交一致
verify-proto:
	@echo "Verifying generated code is up to date..."
	make gen-proto
	git diff --exit-code apps/*/gen apps/*/src/main/java-gen apps/*/src/gen || \
	  (echo "Generated code is out of date. Run 'make gen-proto' and commit changes." && exit 1)
```

**Service Configuration to Use Generated Code**:

**Java (build.gradle)**:
```groovy
// apps/hello-service/build.gradle
sourceSets {
    main {
        java {
            srcDirs += ['src/main/java-gen']
        }
    }
}
```

**Go (import path)**:
```go
// apps/todo-service/main.go
import (
    todopb "github.com/myorg/myrepo/apps/todo-service/gen/api/v1"
)
```

**TypeScript (tsconfig.json)**:
```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@gen/*": ["src/gen/*"]
    }
  }
}
```

**Pre-commit Hook** (`.git/hooks/pre-commit`):
```bash
#!/bin/bash
# 自动检查 Protobuf 生成代码是否最新
make verify-proto
```

**CI Integration**:
```yaml
# .github/workflows/ci.yml
- name: Verify Protobuf code generation
  run: make verify-proto
```

**Benefits of Independent Code Generation**:
- 每个服务完全自包含，可以独立构建和部署
- 避免了跨服务的隐式依赖
- 符合微服务自治原则
- 简化了构建配置

### 2. Hello Service (Java/Spring Boot)

**Location**: `apps/hello-service/`

**Technology Stack**:
- Java 17+
- Spring Boot 3.x
- grpc-spring-boot-starter
- Maven 或 Gradle

**Project Structure**:
```
apps/hello-service/
├── pom.xml (或 build.gradle)
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/myorg/hello/
│   │   │       ├── HelloServiceApplication.java
│   │   │       ├── service/
│   │   │       │   └── HelloServiceImpl.java
│   │   │       └── config/
│   │   │           └── GrpcConfig.java
│   │   ├── proto/  (生成的代码)
│   │   └── resources/
│   │       └── application.yml
│   └── test/
│       └── java/
└── Dockerfile
```

**Key Components**:

1. **HelloServiceImpl** - gRPC 服务实现
```java
@GrpcService
public class HelloServiceImpl extends HelloServiceGrpc.HelloServiceImplBase {
    
    @Override
    public void sayHello(HelloRequest request, 
                        StreamObserver<HelloResponse> responseObserver) {
        String name = request.getName();
        String message;
        
        if (name == null || name.trim().isEmpty()) {
            message = "Hello, World!";
        } else {
            message = "Hello, " + name + "!";
        }
        
        HelloResponse response = HelloResponse.newBuilder()
            .setMessage(message)
            .build();
            
        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
```

2. **Application Configuration** (application.yml)
```yaml
grpc:
  server:
    port: 9090
spring:
  application:
    name: hello-service
```

**Dependencies** (Maven pom.xml):
```xml
<dependencies>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter</artifactId>
    </dependency>
    <dependency>
        <groupId>net.devh</groupId>
        <artifactId>grpc-spring-boot-starter</artifactId>
        <version>2.15.0.RELEASE</version>
    </dependency>
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-protobuf</artifactId>
    </dependency>
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-stub</artifactId>
    </dependency>
</dependencies>
```

### 3. TODO Service (Go)

**Location**: `apps/todo-service/`

**Technology Stack**:
- Go 1.21+
- gRPC Go
- Protocol Buffers

**Project Structure**:
```
apps/todo-service/
├── go.mod
├── go.sum
├── main.go
├── gen/  (生成的 Protobuf 代码)
├── service/
│   └── todo_service.go
├── storage/
│   └── memory_store.go
└── Dockerfile
```

**Key Components**:

1. **TodoService Implementation**
```go
package service

import (
    "context"
    "time"
    
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    todopb "github.com/myorg/myrepo/apps/todo-service/gen/api/v1"
)

type TodoServiceServer struct {
    todopb.UnimplementedTodoServiceServer
    store TodoStore
}

func NewTodoServiceServer(store TodoStore) *TodoServiceServer {
    return &TodoServiceServer{store: store}
}

func (s *TodoServiceServer) CreateTodo(ctx context.Context, 
    req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
    
    if req.Title == "" {
        return nil, status.Error(codes.InvalidArgument, "title is required")
    }
    
    todo := &todopb.Todo{
        Id:          generateID(),
        Title:       req.Title,
        Description: req.Description,
        Completed:   false,
        CreatedAt:   timestamppb.Now(),
        UpdatedAt:   timestamppb.Now(),
    }
    
    if err := s.store.Create(todo); err != nil {
        return nil, status.Error(codes.Internal, "failed to create todo")
    }
    
    return &todopb.CreateTodoResponse{Todo: todo}, nil
}

// 其他方法实现...
```

2. **Memory Store**
```go
package storage

import (
    "sync"
    todopb "github.com/myorg/myrepo/apps/todo-service/gen/api/v1"
)

type MemoryStore struct {
    mu    sync.RWMutex
    todos map[string]*todopb.Todo
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        todos: make(map[string]*todopb.Todo),
    }
}

func (s *MemoryStore) Create(todo *todopb.Todo) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.todos[todo.Id] = todo
    return nil
}

// 其他存储方法...
```

3. **Main Server**
```go
package main

import (
    "log"
    "net"
    
    "google.golang.org/grpc"
    todopb "github.com/myorg/myrepo/apps/todo-service/gen/api/v1"
)

func main() {
    lis, err := net.Listen("tcp", ":9091")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    
    store := storage.NewMemoryStore()
    todoService := service.NewTodoServiceServer(store)
    
    grpcServer := grpc.NewServer()
    todopb.RegisterTodoServiceServer(grpcServer, todoService)
    
    log.Println("TODO Service listening on :9091")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

### 4. Frontend Application (React/TypeScript)

**Location**: `apps/web/`

**Technology Stack**:
- React 18+
- TypeScript 5+
- Vite
- grpc-web
- TanStack Query (React Query)

**Project Structure**:
```
apps/web/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── gen/  (生成的 Protobuf 代码)
│   ├── services/
│   │   ├── helloClient.ts
│   │   └── todoClient.ts
│   ├── components/
│   │   ├── HelloForm.tsx
│   │   └── TodoList.tsx
│   └── hooks/
│       └── useTodos.ts
└── index.html
```

**Key Components**:

1. **gRPC Client Setup**
```typescript
// src/services/helloClient.ts
import { HelloServiceClient } from '../gen/api/v1/hello';

export const helloClient = new HelloServiceClient(
  '/api/hello',  // Envoy 代理路径
  null,
  null
);
```

```typescript
// src/services/todoClient.ts
import { TodoServiceClient } from '../gen/api/v1/todo';

export const todoClient = new TodoServiceClient(
  '/api/todo',
  null,
  null
);
```

2. **Hello Component**
```typescript
// src/components/HelloForm.tsx
import { useState } from 'react';
import { helloClient } from '../services/helloClient';
import { HelloRequest } from '../gen/api/v1/hello';

export function HelloForm() {
  const [name, setName] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    
    try {
      const request = new HelloRequest();
      request.setName(name);
      
      const response = await helloClient.sayHello(request, {});
      setMessage(response.getMessage());
    } catch (error) {
      console.error('Error:', error);
      setMessage('Error calling service');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter your name"
        />
        <button type="submit" disabled={loading}>
          Say Hello
        </button>
      </form>
      {message && <p>{message}</p>}
    </div>
  );
}
```

3. **TODO List Component with React Query**
```typescript
// src/hooks/useTodos.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { todoClient } from '../services/todoClient';
import { 
  ListTodosRequest, 
  CreateTodoRequest,
  UpdateTodoRequest,
  DeleteTodoRequest 
} from '../gen/api/v1/todo';

export function useTodos() {
  const queryClient = useQueryClient();

  const { data: todos, isLoading } = useQuery({
    queryKey: ['todos'],
    queryFn: async () => {
      const request = new ListTodosRequest();
      const response = await todoClient.listTodos(request, {});
      return response.getTodosList();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: { title: string; description: string }) => {
      const request = new CreateTodoRequest();
      request.setTitle(data.title);
      request.setDescription(data.description);
      return todoClient.createTodo(request, {});
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
  });

  // 其他 mutations...

  return { todos, isLoading, createMutation };
}
```

### 5. API Gateway (Higress)

**Location**: `deploy/k8s/services/higress/` 或 K8s 配置

**Higress Configuration**:

Higress 是阿里云开源的云原生 API 网关，基于 Envoy 和 Istio 构建，专为 K8s 环境优化。

**Ingress 配置** (K8s):
```yaml
# deploy/k8s/services/higress/higress-routes.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monorepo-ingress
  annotations:
    higress.io/backend-protocol: "GRPC"
    higress.io/cors-allow-origin: "*"
    higress.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
    higress.io/cors-allow-headers: "content-type,x-grpc-web,x-user-agent"
    higress.io/cors-expose-headers: "grpc-status,grpc-message"
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /api/hello
        pathType: Prefix
        backend:
          service:
            name: hello-service
            port:
              number: 9090
      - path: /api/todo
        pathType: Prefix
        backend:
          service:
            name: todo-service
            port:
              number: 9091
```

**McpBridge 配置** (Higress 插件，用于 gRPC-Web):
```yaml
# deploy/k8s/services/higress/mcpbridge.yaml
apiVersion: extensions.higress.io/v1alpha1
kind: McpBridge
metadata:
  name: grpc-web-bridge
  namespace: higress-system
spec:
  services:
  - name: hello-service
    namespace: default
    port: 9090
    protocol: grpc
  - name: todo-service
    namespace: default
    port: 9091
    protocol: grpc
```

### 6. Service Templates

**Location**: `templates/`

#### Java/Spring Boot Service Template

**Directory Structure**:
```
templates/java-service/
├── catalog-info.yaml
├── pom.xml
├── Dockerfile
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
└── src/
    ├── main/
    │   ├── java/
    │   │   └── com/myorg/template/
    │   │       ├── TemplateApplication.java
    │   │       ├── service/
    │   │       │   └── TemplateServiceImpl.java
    │   │       └── config/
    │   │           └── GrpcConfig.java
    │   └── resources/
    │       └── application.yml
    └── test/
        └── java/
```

**catalog-info.yaml** (Backstage Service Catalog):
```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: hello-service
  description: Hello greeting service
  annotations:
    github.com/project-slug: myorg/myrepo
    backstage.io/techdocs-ref: dir:.
  tags:
    - java
    - spring-boot
    - grpc
  links:
    - url: https://api.example.com/api/hello
      title: API Endpoint
      icon: web
spec:
  type: service
  lifecycle: production
  owner: backend-java-team
  system: monorepo-platform
  providesApis:
    - hello-api
  dependsOn:
    - resource:default/hello-database
---
apiVersion: backstage.io/v1alpha1
kind: API
metadata:
  name: hello-api
  description: Hello Service gRPC API
spec:
  type: grpc
  lifecycle: production
  owner: backend-java-team
  system: monorepo-platform
  definition: |
    # Reference to Protobuf definition
    file: api/v1/hello.proto
```

**pom.xml** (Maven Template):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    
    <groupId>com.myorg</groupId>
    <artifactId>hello-service</artifactId>
    <version>1.0.0-SNAPSHOT</version>
    
    <properties>
        <java.version>17</java.version>
        <grpc.version>1.60.0</grpc.version>
        <protobuf.version>3.25.1</protobuf.version>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter</artifactId>
        </dependency>
        <dependency>
            <groupId>net.devh</groupId>
            <artifactId>grpc-spring-boot-starter</artifactId>
            <version>2.15.0.RELEASE</version>
        </dependency>
        <dependency>
            <groupId>io.grpc</groupId>
            <artifactId>grpc-protobuf</artifactId>
            <version>${grpc.version}</version>
        </dependency>
        <dependency>
            <groupId>io.grpc</groupId>
            <artifactId>grpc-stub</artifactId>
            <version>${grpc.version}</version>
        </dependency>
        
        <!-- Testing -->
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-test</artifactId>
            <scope>test</scope>
        </dependency>
        <dependency>
            <groupId>net.jqwik</groupId>
            <artifactId>jqwik</artifactId>
            <version>1.8.2</version>
            <scope>test</scope>
        </dependency>
    </dependencies>
    
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
            <plugin>
                <groupId>org.xolstice.maven.plugins</groupId>
                <artifactId>protobuf-maven-plugin</artifactId>
                <version>0.6.1</version>
                <configuration>
                    <protocArtifact>
                        com.google.protobuf:protoc:${protobuf.version}:exe:${os.detected.classifier}
                    </protocArtifact>
                    <pluginId>grpc-java</pluginId>
                    <pluginArtifact>
                        io.grpc:protoc-gen-grpc-java:${grpc.version}:exe:${os.detected.classifier}
                    </pluginArtifact>
                    <protoSourceRoot>../../api/v1</protoSourceRoot>
                </configuration>
                <executions>
                    <execution>
                        <goals>
                            <goal>compile</goal>
                            <goal>compile-custom</goal>
                        </goals>
                    </execution>
                </executions>
            </plugin>
        </plugins>
    </build>
</project>
```

#### Go Service Template

**Directory Structure**:
```
templates/go-service/
├── catalog-info.yaml
├── go.mod
├── Dockerfile
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
├── main.go
├── service/
│   └── template_service.go
└── storage/
    └── store.go
```

**catalog-info.yaml**:
```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: todo-service
  description: TODO management service
  annotations:
    github.com/project-slug: myorg/myrepo
    backstage.io/techdocs-ref: dir:.
  tags:
    - go
    - grpc
  links:
    - url: https://api.example.com/api/todo
      title: API Endpoint
      icon: web
spec:
  type: service
  lifecycle: production
  owner: backend-go-team
  system: monorepo-platform
  providesApis:
    - todo-api
  consumesApis:
    - hello-api
---
apiVersion: backstage.io/v1alpha1
kind: API
metadata:
  name: todo-api
  description: TODO Service gRPC API
spec:
  type: grpc
  lifecycle: production
  owner: backend-go-team
  system: monorepo-platform
  definition: |
    file: api/v1/todo.proto
```

**go.mod** (Template):
```go
module github.com/myorg/myrepo/apps/todo-service

go 1.21

require (
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
    github.com/google/uuid v1.5.0
)
```

### 7. Kubernetes Deployment Resources

**Location**: `k8s/` (根目录) 或各服务的 `k8s/` 子目录

#### Hello Service K8s Resources

**Deployment**:
```yaml
# apps/hello-service/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-service
  namespace: default
  labels:
    app: hello-service
    version: v1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: hello-service
  template:
    metadata:
      labels:
        app: hello-service
        version: v1
    spec:
      containers:
      - name: hello-service
        image: hello-service:latest
        imagePullPolicy: IfNotPresent
        ports:
        - name: grpc
          containerPort: 9090
          protocol: TCP
        env:
        - name: SPRING_PROFILES_ACTIVE
          value: "production"
        - name: GRPC_SERVER_PORT
          value: "9090"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          grpc:
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          grpc:
            port: 9090
          initialDelaySeconds: 10
          periodSeconds: 5
```

**Service**:
```yaml
# apps/hello-service/k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: hello-service
  namespace: default
  labels:
    app: hello-service
spec:
  type: ClusterIP
  ports:
  - port: 9090
    targetPort: 9090
    protocol: TCP
    name: grpc
  selector:
    app: hello-service
```

**ConfigMap**:
```yaml
# apps/hello-service/k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hello-service-config
  namespace: default
data:
  application.yml: |
    grpc:
      server:
        port: 9090
    spring:
      application:
        name: hello-service
    logging:
      level:
        root: INFO
        com.myorg: DEBUG
```

#### TODO Service K8s Resources

**Deployment**:
```yaml
# apps/todo-service/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: todo-service
  namespace: default
  labels:
    app: todo-service
    version: v1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: todo-service
  template:
    metadata:
      labels:
        app: todo-service
        version: v1
    spec:
      containers:
      - name: todo-service
        image: todo-service:latest
        imagePullPolicy: IfNotPresent
        ports:
        - name: grpc
          containerPort: 9091
          protocol: TCP
        env:
        - name: PORT
          value: "9091"
        - name: HELLO_SERVICE_ADDR
          value: "hello-service:9090"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          grpc:
            port: 9091
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          grpc:
            port: 9091
          initialDelaySeconds: 5
          periodSeconds: 5
```

**Service**:
```yaml
# apps/todo-service/k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: todo-service
  namespace: default
  labels:
    app: todo-service
spec:
  type: ClusterIP
  ports:
  - port: 9091
    targetPort: 9091
    protocol: TCP
    name: grpc
  selector:
    app: todo-service
```

#### Kustomization

**Base Kustomization**:
```yaml
# k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../apps/hello-service/k8s/deployment.yaml
- ../../apps/hello-service/k8s/service.yaml
- ../../apps/hello-service/k8s/configmap.yaml
- ../../apps/todo-service/k8s/deployment.yaml
- ../../apps/todo-service/k8s/service.yaml
- ../../deploy/k8s/services/higress/higress-routes.yaml

namespace: default

commonLabels:
  project: monorepo-platform
  managed-by: kustomize
```

**Production Overlay**:
```yaml
# k8s/overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
- ../../base

namespace: production

replicas:
- name: hello-service
  count: 3
- name: todo-service
  count: 3

images:
- name: hello-service
  newName: registry.example.com/hello-service
  newTag: v1.0.0
- name: todo-service
  newName: registry.example.com/todo-service
  newTag: v1.0.0

patchesStrategicMerge:
- resources-patch.yaml
```

**Resources Patch**:
```yaml
# k8s/overlays/production/resources-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-service
spec:
  template:
    spec:
      containers:
      - name: hello-service
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: todo-service
spec:
  template:
    spec:
      containers:
      - name: todo-service
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "400m"
```

### Service-to-Service Communication Example

**TODO Service calling Hello Service** (东西向直连):

**配置驱动的服务发现**:

```go
// apps/todo-service/client/hello_client.go
package client

import (
    "context"
    "fmt"
    "os"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    hellopb "github.com/myorg/myrepo/gen/go/api/v1"
)

type HelloClient struct {
    client hellopb.HelloServiceClient
    conn   *grpc.ClientConn
}

func NewHelloClient() (*HelloClient, error) {
    // 从环境变量读取服务地址，支持不同环境
    addr := os.Getenv("HELLO_SERVICE_ADDR")
    if addr == "" {
        addr = "localhost:9090" // 默认本地地址
    }
    
    conn, err := grpc.Dial(addr, 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
    }
    
    client := hellopb.NewHelloServiceClient(conn)
    return &HelloClient{
        client: client,
        conn:   conn,
    }, nil
}

func (c *HelloClient) SayHello(ctx context.Context, name string) (string, error) {
    req := &hellopb.HelloRequest{Name: name}
    resp, err := c.client.SayHello(ctx, req)
    if err != nil {
        return "", err
    }
    return resp.Message, nil
}

func (c *HelloClient) Close() error {
    return c.conn.Close()
}
```

**Environment-specific Configuration**:

**Local Development** (`.env` or shell):
```bash
export HELLO_SERVICE_ADDR="localhost:9090"
```

**Kubernetes Deployment**:
```yaml
# apps/todo-service/k8s/deployment.yaml
env:
- name: HELLO_SERVICE_ADDR
  value: "hello-service:9090"  # K8s Service DNS
- name: PORT
  value: "9091"
```

**Docker Compose** (for integration testing):
```yaml
# docker-compose.yml
services:
  hello-service:
    build: ./apps/hello-service
    ports:
      - "9090:9090"
  
  todo-service:
    build: ./apps/todo-service
    environment:
      - HELLO_SERVICE_ADDR=hello-service:9090
    ports:
      - "9091:9091"
    depends_on:
      - hello-service
```

**Usage in TODO Service**:
```go
// apps/todo-service/service/todo_service.go
func (s *TodoServiceServer) CreateTodoWithGreeting(ctx context.Context, 
    req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
    
    // 调用 Hello Service（东西向直连）
    greeting, err := s.helloClient.SayHello(ctx, "TODO Creator")
    if err != nil {
        log.Printf("Failed to get greeting: %v", err)
        greeting = "Hello!"  // Fallback
    }
    
    // 创建 TODO，标题包含问候语
    todo := &todopb.Todo{
        Id:          generateID(),
        Title:       fmt.Sprintf("%s - %s", greeting, req.Title),
        Description: req.Description,
        Completed:   false,
        CreatedAt:   timestamppb.Now(),
        UpdatedAt:   timestamppb.Now(),
    }
    
    if err := s.store.Create(todo); err != nil {
        return nil, status.Error(codes.Internal, "failed to create todo")
    }
    
    return &todopb.CreateTodoResponse{Todo: todo}, nil
}
```

**Benefits of Configuration-Driven Service Discovery**:
- 环境解耦：本地、测试、生产使用不同配置
- 易于测试：可以轻松 mock 服务地址
- 灵活性：支持服务迁移和多集群部署

### Service Communication Resilience

**弹性模式实施**:

为了提高服务间通信的健壮性，需要实施以下弹性模式：

1. **超时控制 (Timeout)**
```go
// 设置请求超时
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

greeting, err := s.helloClient.SayHello(ctx, "TODO Creator")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Hello service timeout")
    }
    // Fallback logic
}
```

2. **重试机制 (Retry)**
```go
// Go: 使用 grpc-go 的重试策略
import "google.golang.org/grpc"

conn, err := grpc.Dial(addr,
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithDefaultServiceConfig(`{
        "methodConfig": [{
            "name": [{"service": "api.v1.HelloService"}],
            "retryPolicy": {
                "maxAttempts": 3,
                "initialBackoff": "0.1s",
                "maxBackoff": "1s",
                "backoffMultiplier": 2,
                "retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
            }
        }]
    }`),
)
```

```java
// Java: 使用 Resilience4j
@Retry(name = "helloService", fallbackMethod = "helloServiceFallback")
public String callHelloService(String name) {
    return helloClient.sayHello(name);
}

private String helloServiceFallback(String name, Exception ex) {
    log.warn("Hello service call failed, using fallback", ex);
    return "Hello, " + name + "!";
}
```

3. **熔断器 (Circuit Breaker)**
```java
// Java: Resilience4j Circuit Breaker
@CircuitBreaker(name = "helloService", fallbackMethod = "helloServiceFallback")
public String callHelloService(String name) {
    return helloClient.sayHello(name);
}
```

4. **降级策略 (Fallback)**
- 当依赖服务不可用时，返回默认值或缓存数据
- 记录降级事件用于监控和告警

**Resilience4j Configuration** (application.yml):
```yaml
resilience4j:
  retry:
    instances:
      helloService:
        maxAttempts: 3
        waitDuration: 100ms
        exponentialBackoffMultiplier: 2
  circuitbreaker:
    instances:
      helloService:
        failureRateThreshold: 50
        waitDurationInOpenState: 10s
        slidingWindowSize: 10
```

## Data Models

### Hello Service Data Model

Hello 服务使用简单的请求-响应模式，不需要持久化存储。

**HelloRequest**:
- `name` (string): 用户姓名，可选

**HelloResponse**:
- `message` (string): 生成的问候消息

### TODO Service Data Model

TODO 服务使用内存存储，数据模型如下：

**Todo**:
- `id` (string): UUID 格式的唯一标识符
- `title` (string): TODO 项标题，必填
- `description` (string): TODO 项描述，可选
- `completed` (bool): 完成状态，默认 false
- `created_at` (timestamp): 创建时间
- `updated_at` (timestamp): 最后更新时间

**Storage Interface**:
```go
type TodoStore interface {
    Create(todo *Todo) error
    Get(id string) (*Todo, error)
    List() ([]*Todo, error)
    Update(todo *Todo) error
    Delete(id string) error
}
```

**Memory Store Implementation**:
- 使用 `map[string]*Todo` 存储数据
- 使用 `sync.RWMutex` 保证并发安全
- 数据在服务重启后丢失（符合当前需求）

### Persistence Strategy (Future Enhancement)

当前 TODO 服务使用内存存储，适用于演示和测试。生产部署需支持持久化：

**Phase 1: PostgreSQL Support**
- 使用 GORM 或 sqlx 实现 SQL 存储
- 提供 `storage/postgres.go` 实现 `TodoStore` 接口
- 通过环境变量切换存储后端：
  ```go
  storageType := os.Getenv("STORAGE_TYPE") // "memory" or "postgres"
  if storageType == "postgres" {
      store = storage.NewPostgresStore(dbURL)
  } else {
      store = storage.NewMemoryStore()
  }
  ```

**Phase 2: Migration & Backup**
- 集成 golang-migrate 工具管理 schema
- 支持定期备份到对象存储（S3/OSS）
- 实现数据导入导出功能

**Phase 3: Multi-tenancy Support**
- 添加 tenant_id 字段
- 实现租户隔离
- 支持租户级别的数据备份

## Repository Governance

### Code Ownership

通过 `.github/CODEOWNERS` 文件定义代码所有权：

```
# CODEOWNERS
# API 契约层由平台团队负责
/api/ @platform-team

# 各服务由对应团队负责
/apps/hello-service/ @backend-java-team
/apps/todo-service/ @backend-go-team
/apps/web/ @frontend-team

# 基础设施和工具
/deploy/ @platform-team
/k8s/ @platform-team
/scripts/ @platform-team

# 文档
/docs/ @platform-team
README.md @platform-team
```

### Build Health Metrics

**仓库健康度指标**:
- 最大仓库大小：< 2GB
- CI 构建时间：< 10 分钟
- 代码重复率：< 5%（使用 jscpd 或类似工具）
- 所有服务必须通过 `make test`
- 测试覆盖率：> 70%

**监控方式**:
- 在 CI 中收集指标
- 在 Backstage 中展示健康度仪表板
- 定期生成健康度报告

### Adding New Services

**标准流程**:

1. **从模板创建**
   ```bash
   # 复制对应语言的模板
   cp -r templates/go-service apps/new-service
   cd apps/new-service
   # 修改服务名称和配置
   ```

2. **定义 API（如需要）**
   - 在 `api/v1/` 下创建 `.proto` 文件
   - 运行 `make gen-proto` 生成代码

3. **更新构建脚本**
   - 将新服务添加到 `Makefile`
   - 更新 `scripts/dev.sh`

4. **创建 K8s 资源**
   - 在 `apps/new-service/k8s/` 下创建 Deployment、Service 等
   - 更新 `k8s/base/kustomization.yaml`

5. **注册到 Backstage**
   - 创建 `catalog-info.yaml`
   - 提交 PR，Backstage 自动发现

6. **文档和测试**
   - 添加 README.md
   - 编写单元测试和集成测试
   - 更新根目录 README

### Pull Request Guidelines

**PR 要求**:
- 所有 PR 必须通过 CI 检查
- 需要至少一个 CODEOWNERS 成员审批
- 代码必须符合格式规范（通过 lint 检查）
- 新功能需要包含测试
- API 变更需要更新文档

**PR 模板** (`.github/pull_request_template.md`):
```markdown
## Description
<!-- 描述本 PR 的目的和变更内容 -->

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Checklist
- [ ] 代码通过 `make test`
- [ ] 代码通过 `make lint`
- [ ] 更新了相关文档
- [ ] 添加了必要的测试
- [ ] Protobuf 变更已运行 `make gen-proto`

## Related Issues
<!-- 关联的 Issue 编号 -->
```

### Dependency Management

**原则**:
- 定期更新依赖（每月一次）
- 使用 Dependabot 自动检测安全漏洞
- 重大版本升级需要在 staging 环境测试

**工具**:
- Java: Dependabot + Maven Versions Plugin
- Go: Dependabot + go mod tidy
- TypeScript: Dependabot + npm audit

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*


### Property 1: Hello Service Name Inclusion

*For any* non-empty name string, when calling the Hello service with that name, the returned message should contain the provided name.

**Validates: Requirements 7.1**

**Test Strategy**: Use property-based testing to generate random name strings and verify the response contains the name.

### Property 2: TODO Creation Returns Unique IDs

*For any* sequence of TODO creation requests, all returned TODO IDs should be unique (no duplicates).

**Validates: Requirements 8.1**

**Test Strategy**: Create multiple TODOs and verify all IDs are distinct.

### Property 3: TODO CRUD Round-Trip Consistency

*For any* TODO item, the following operations should maintain consistency:
- Create → List: Created TODO appears in list
- Create → Update → Get: Updated fields are persisted
- Create → Delete → List: Deleted TODO does not appear in list

**Validates: Requirements 8.2, 8.3, 8.4**

**Test Strategy**: Use property-based testing to generate random TODO data and verify CRUD operations maintain consistency.

### Property 4: Frontend Error Handling

*For any* service error response, the frontend should display an error message to the user (not crash or show blank screen).

**Validates: Requirements 9.7**

**Test Strategy**: Simulate various error conditions and verify UI displays error messages.

## Error Handling

### Hello Service Error Handling

1. **Empty Name Handling**
   - Input: Empty or whitespace-only name
   - Behavior: Return default message "Hello, World!"
   - No error thrown

2. **Service Unavailable**
   - Scenario: Service is down or unreachable
   - Response: gRPC UNAVAILABLE status
   - Client should display user-friendly error

### TODO Service Error Handling

1. **Invalid Input Validation**
   - Empty title: Return INVALID_ARGUMENT status
   - Missing required fields: Return INVALID_ARGUMENT status

2. **Not Found Errors**
   - Get/Update/Delete non-existent TODO: Return NOT_FOUND status
   - Include TODO ID in error message

3. **Concurrent Access**
   - Use mutex locks to prevent race conditions
   - Ensure atomic operations on shared data

4. **Storage Errors**
   - Internal errors: Return INTERNAL status
   - Log detailed error information for debugging

### Frontend Error Handling

1. **Network Errors**
   - Display: "Unable to connect to server. Please try again."
   - Retry mechanism with exponential backoff

2. **Service Errors**
   - Parse gRPC status codes
   - Display user-friendly messages based on error type
   - Log technical details to console

3. **Validation Errors**
   - Client-side validation before sending requests
   - Display inline validation messages

## Testing Strategy

### Unit Testing

**Hello Service (Java)**:
- Test `sayHello` with various name inputs
- Test empty name returns default message
- Test service initialization and configuration
- Framework: JUnit 5 + Mockito

**TODO Service (Go)**:
- Test each CRUD operation independently
- Test memory store operations
- Test concurrent access scenarios
- Framework: Go testing package + testify

**Frontend (TypeScript)**:
- Test React components with React Testing Library
- Test gRPC client setup and error handling
- Test custom hooks (useTodos)
- Framework: Vitest + React Testing Library

### Property-Based Testing

**Property Test Configuration**:
- Minimum 100 iterations per test
- Use appropriate PBT libraries:
  - Java: jqwik or QuickTheories
  - Go: gopter or rapid
  - TypeScript: fast-check

**Property Tests to Implement**:

1. **Hello Service Property Test**
   ```java
   @Property
   void helloServiceIncludesNameInResponse(@ForAll String name) {
       // Feature: monorepo-hello-todo, Property 1: Hello Service Name Inclusion
       if (name != null && !name.trim().isEmpty()) {
           HelloResponse response = helloService.sayHello(
               HelloRequest.newBuilder().setName(name).build()
           );
           assertThat(response.getMessage()).contains(name);
       }
   }
   ```

2. **TODO ID Uniqueness Property Test**
   ```go
   // Feature: monorepo-hello-todo, Property 2: TODO Creation Returns Unique IDs
   func TestTodoIdUniqueness(t *testing.T) {
       rapid.Check(t, func(t *rapid.T) {
           count := rapid.IntRange(2, 20).Draw(t, "count")
           ids := make(map[string]bool)
           
           for i := 0; i < count; i++ {
               title := rapid.String().Draw(t, "title")
               todo := createTodo(title, "")
               
               if ids[todo.Id] {
                   t.Fatalf("Duplicate ID found: %s", todo.Id)
               }
               ids[todo.Id] = true
           }
       })
   }
   ```

3. **TODO CRUD Round-Trip Property Test**
   ```go
   // Feature: monorepo-hello-todo, Property 3: TODO CRUD Round-Trip Consistency
   func TestTodoCrudRoundTrip(t *testing.T) {
       rapid.Check(t, func(t *rapid.T) {
           title := rapid.String().Draw(t, "title")
           desc := rapid.String().Draw(t, "description")
           
           // Create
           created := createTodo(title, desc)
           
           // List - should contain created TODO
           list := listTodos()
           assert.Contains(t, list, created)
           
           // Update
           newTitle := rapid.String().Draw(t, "newTitle")
           updated := updateTodo(created.Id, newTitle, desc, true)
           assert.Equal(t, newTitle, updated.Title)
           assert.True(t, updated.Completed)
           
           // Delete
           deleteTodo(created.Id)
           listAfterDelete := listTodos()
           assert.NotContains(t, listAfterDelete, created)
       })
   }
   ```

4. **Frontend Error Handling Property Test**
   ```typescript
   // Feature: monorepo-hello-todo, Property 4: Frontend Error Handling
   it('displays error message for any service error', () => {
     fc.assert(
       fc.property(
         fc.integer({ min: 1, max: 16 }), // gRPC status codes
         fc.string(), // error message
         (statusCode, errorMessage) => {
           const error = new Error(errorMessage);
           error.code = statusCode;
           
           // Simulate error
           mockTodoClient.listTodos.mockRejectedValue(error);
           
           render(<TodoList />);
           
           // Should display error, not crash
           expect(screen.queryByText(/error/i)).toBeInTheDocument();
         }
       )
     );
   });
   ```

### Integration Testing

**End-to-End Tests**:
- Test complete user flows through the UI
- Test service-to-service communication through API gateway
- Test Protobuf serialization/deserialization
- Framework: Playwright or Cypress

**API Gateway Tests**:
- Test routing to correct services
- Test CORS configuration
- Test gRPC-Web protocol conversion
- Framework: curl + shell scripts or Postman

### Test Organization

```
apps/hello-service/
└── src/test/java/
    ├── unit/
    │   └── HelloServiceTest.java
    └── property/
        └── HelloServicePropertyTest.java

apps/todo-service/
└── service/
    ├── todo_service_test.go
    └── todo_service_property_test.go

apps/web/
└── src/
    ├── components/
    │   ├── HelloForm.test.tsx
    │   └── TodoList.test.tsx
    └── __tests__/
        └── properties/
            └── errorHandling.property.test.ts
```

## Build and Deployment

### Development Workflow

**Local Development Script** (`scripts/dev.sh`):
```bash
#!/bin/bash

# Start all services in development mode

echo "Starting Envoy Proxy (for local gRPC-Web)..."
docker run -d --name envoy-local --network host \
  -v $(pwd)/deploy/docker:/config \
  envoyproxy/envoy:v1.30 -c /config/envoy-local.yaml
ENVOY_CONTAINER="envoy-local"

echo "Starting Hello Service (Java)..."
cd apps/hello-service
./mvnw spring-boot:run &
HELLO_PID=$!

echo "Starting TODO Service (Go)..."
cd ../todo-service
HELLO_SERVICE_ADDR="localhost:9090" go run . &
TODO_PID=$!

echo "Starting Frontend (React)..."
cd ../web
npm run dev &
WEB_PID=$!

echo "All services started!"
echo "Frontend: http://localhost:5173"
echo "Envoy Proxy: http://localhost:8080 (gRPC-Web gateway)"
echo "Hello Service: localhost:9090"
echo "TODO Service: localhost:9091"
echo ""
echo "Note: Frontend calls services through Envoy at http://localhost:8080"
echo "This matches the production Higress routing pattern"
echo ""
echo "Press Ctrl+C to stop all services"

# Cleanup on exit
cleanup() {
  echo "Stopping services..."
  kill $HELLO_PID $TODO_PID $WEB_PID 2>/dev/null
  docker stop $ENVOY_CONTAINER 2>/dev/null
  docker rm $ENVOY_CONTAINER 2>/dev/null
}

trap cleanup EXIT
wait
```

**Local Envoy Configuration** (`deploy/docker/envoy-local-config.yaml`):
```yaml
# 本地开发用的 Envoy 配置，模拟 Higress 的路由行为
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 8080
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          route_config:
            name: local_route
            virtual_hosts:
            - name: backend
              domains: ["*"]
              routes:
              - match:
                  prefix: "/api/hello"
                route:
                  cluster: hello_service
                  prefix_rewrite: "/"
              - match:
                  prefix: "/api/todo"
                route:
                  cluster: todo_service
                  prefix_rewrite: "/"
              cors:
                allow_origin_string_match:
                - safe_regex:
                    google_re2: {}
                    regex: ".*"
                allow_methods: "GET, POST, PUT, DELETE, OPTIONS"
                allow_headers: "content-type,x-grpc-web,x-user-agent"
                expose_headers: "grpc-status,grpc-message"
          http_filters:
          - name: envoy.filters.http.grpc_web
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb
          - name: envoy.filters.http.cors
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router

  clusters:
  - name: hello_service
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    load_assignment:
      cluster_name: hello_service
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: localhost
                port_value: 9090

  - name: todo_service
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    load_assignment:
      cluster_name: todo_service
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: localhost
                port_value: 9091
```

**Benefits of Local Envoy**:
- 本地开发流量路径与生产一致（Frontend → Proxy → Services）
- 可验证 CORS、路径路由、Header 转换
- 无缝切换到 Higress（仅改 DNS）

### Build Commands

**Makefile**:
```makefile
.PHONY: all build test clean gen-proto

all: gen-proto build test

# Code generation
gen-proto: gen-proto-go gen-proto-java gen-proto-ts

gen-proto-go:
	@echo "Generating Go code from Protobuf..."
	protoc --go_out=apps/todo-service/gen \
	       --go_opt=paths=source_relative \
	       --go-grpc_out=apps/todo-service/gen \
	       --go-grpc_opt=paths=source_relative \
	       -I api/v1 \
	       api/v1/*.proto

gen-proto-java:
	@echo "Generating Java code from Protobuf..."
	cd apps/hello-service && ./mvnw protobuf:compile protobuf:compile-custom

gen-proto-ts:
	@echo "Generating TypeScript code from Protobuf..."
	cd apps/web && npm run gen-proto

# Build
build: build-hello build-todo build-web

build-hello:
	@echo "Building Hello Service..."
	cd apps/hello-service && ./mvnw clean package -DskipTests

build-todo:
	@echo "Building TODO Service..."
	cd apps/todo-service && go build -o bin/todo-service .

build-web:
	@echo "Building Frontend..."
	cd apps/web && npm run build

# Test
test: test-hello test-todo test-web

test-hello:
	@echo "Testing Hello Service..."
	cd apps/hello-service && ./mvnw test

test-todo:
	@echo "Testing TODO Service..."
	cd apps/todo-service && go test ./...

test-web:
	@echo "Testing Frontend..."
	cd apps/web && npm test

# Clean
clean:
	@echo "Cleaning build artifacts..."
	cd apps/hello-service && ./mvnw clean
	cd apps/todo-service && rm -rf bin/
	cd apps/web && rm -rf dist/

# Docker
docker-build: docker-build-hello docker-build-todo

docker-build-hello:
	docker build -t hello-service:latest apps/hello-service

docker-build-todo:
	docker build -t todo-service:latest apps/todo-service
```

### Docker Configuration

**Hello Service Dockerfile**:
```dockerfile
FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /app
COPY pom.xml .
COPY src ./src
RUN mvn clean package -DskipTests

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=build /app/target/*.jar app.jar
EXPOSE 9090
ENTRYPOINT ["java", "-jar", "app.jar"]
```

**TODO Service Dockerfile**:
```dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o todo-service .

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/todo-service .
EXPOSE 9091
ENTRYPOINT ["./todo-service"]
```

### CI/CD Pipeline

**GitHub Actions Example** (`.github/workflows/ci.yml`):
```yaml
name: CI/CD

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Java
        uses: actions/setup-java@v3
        with:
          java-version: '17'
          distribution: 'temurin'
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Set up Node
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install Protobuf
        run: |
          sudo apt-get update
          sudo apt-get install -y protobuf-compiler
      
      - name: Generate Protobuf code
        run: make gen-proto
      
      - name: Run tests
        run: make test
      
      - name: Build all services
        run: make build

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Login to Container Registry
        uses: docker/login-action@v2
        with:
          registry: registry.example.com
          username: ${{ secrets.REGISTRY_USERNAME }}
          password: ${{ secrets.REGISTRY_PASSWORD }}
      
      - name: Build and push Hello Service
        uses: docker/build-push-action@v4
        with:
          context: apps/hello-service
          push: true
          tags: |
            registry.example.com/hello-service:latest
            registry.example.com/hello-service:${{ github.sha }}
      
      - name: Build and push TODO Service
        uses: docker/build-push-action@v4
        with:
          context: apps/todo-service
          push: true
          tags: |
            registry.example.com/todo-service:latest
            registry.example.com/todo-service:${{ github.sha }}

  deploy:
    needs: build-and-push
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up kubectl
        uses: azure/setup-kubectl@v3
      
      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBECONFIG }}" | base64 -d > kubeconfig
          export KUBECONFIG=./kubeconfig
      
      - name: Deploy to K8s
        run: |
          kubectl apply -k k8s/overlays/production
          kubectl rollout status deployment/hello-service -n production
          kubectl rollout status deployment/todo-service -n production
```

**ArgoCD Application** (GitOps 方式):
```yaml
# k8s/argocd/application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: monorepo-platform
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/myorg/myrepo.git
    targetRevision: main
    path: k8s/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## Security Considerations

1. **API Gateway Security**
   - Configure TLS/SSL for production
   - Implement rate limiting
   - Add authentication/authorization layer (future enhancement)

2. **Input Validation**
   - Validate all inputs on server side
   - Sanitize user inputs to prevent injection attacks
   - Enforce length limits on strings

3. **Dependency Management**
   - Regularly update dependencies
   - Use dependency scanning tools (Dependabot, Snyk)
   - Pin dependency versions for reproducibility

4. **Secrets Management**
   - Never commit secrets to repository
   - Use environment variables for configuration
   - Use secret management tools in production (Vault, AWS Secrets Manager)

## Performance Considerations

1. **gRPC Performance**
   - Use HTTP/2 multiplexing
   - Enable gRPC compression for large payloads
   - Connection pooling for client connections

2. **Frontend Performance**
   - Code splitting for React application
   - Lazy loading of components
   - Caching of API responses with React Query

3. **Memory Management**
   - TODO service uses in-memory storage (limited scalability)
   - Consider adding pagination for large TODO lists
   - Monitor memory usage in production

## Future Enhancements

### 1. Backstage Integration (Internal Developer Platform)

**Goal**: 将 Backstage 作为统一的开发者门户，提供服务发现、文档、CI/CD 可视化等能力。

**Integration Points**:

1. **Service Catalog**
   - 每个服务已包含 catalog-info.yaml
   - Backstage 自动扫描 Monorepo 发现所有服务
   - 展示服务拓扑、依赖关系、责任人

2. **Software Templates (Scaffolder)**
   - 创建新服务模板（Java/Go）
   - 通过 GitHub API 向 Monorepo 提交 PR
   - 自动生成标准化的项目结构和配置

3. **TechDocs**
   - 从各服务的 docs/ 目录自动构建文档
   - 从 Protobuf 生成 API 文档
   - 统一的文档中心

4. **CI/CD Visualization**
   - 集成 GitHub Actions 状态
   - 展示构建日志和部署状态
   - 一键触发部署

5. **API Registry**
   - 从 Protobuf 生成 OpenAPI 规范
   - 展示所有服务的 API 接口
   - 支持 API 版本管理

**Implementation Phases**:
- Phase 1: 服务目录集成（已完成 catalog-info.yaml）
- Phase 2: 文档集成（TechDocs）
- Phase 3: CI/CD 可视化
- Phase 4: 自助服务创建（Scaffolder）
- Phase 5: 全链路开发体验优化

**Key Principles**:
- Backstage 是"仪表盘"，Monorepo 是"引擎"
- 南北向走网关，东西向直连
- 元数据驱动（catalog-info.yaml）
- 自动化优先

### 2. Persistence Layer
   - Add database support for TODO service (PostgreSQL, MongoDB)
   - Implement data migration scripts

### 2. Persistence Layer
   - Add database support for TODO service (PostgreSQL, MongoDB)
   - Implement data migration scripts

### 3. Authentication & Authorization
   - Add JWT-based authentication
   - Implement role-based access control

### 3. Authentication & Authorization
   - Add JWT-based authentication
   - Implement role-based access control

### 4. Observability
   - Add distributed tracing (OpenTelemetry)
   - Implement structured logging
   - Add metrics collection (Prometheus)

### 4. Observability
   - Add distributed tracing (OpenTelemetry)
   - Implement structured logging
   - Add metrics collection (Prometheus)

### 5. Advanced Features
   - Real-time updates with gRPC streaming
   - Offline support with service workers
   - Multi-user support with user accounts

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://protobuf.dev/)
- [Spring Boot gRPC Starter](https://github.com/grpc-ecosystem/grpc-spring)
- [Higress Gateway Documentation](https://higress.io/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Backstage Service Catalog](https://backstage.io/docs/features/software-catalog/)
- [Kustomize Documentation](https://kustomize.io/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [React Query Documentation](https://tanstack.com/query/latest)
