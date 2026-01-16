# Implementation Plan: Monorepo Hello/TODO Services

## Overview

本实现计划将 Monorepo Hello/TODO Services 的设计转化为可执行的开发任务。任务按照依赖关系组织，从项目初始化开始，逐步实现 API 契约、服务、前端和基础设施。

## Tasks

- [x] 1. 项目结构初始化
  - 创建 Monorepo 根目录结构
  - 配置构建系统和工具
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.7_

- [x] 1.1 创建根目录结构
  - 创建 `api/`、`apps/`、`libs/`、`tools/`、`scripts/` 目录
  - 创建 `templates/` 目录用于服务模板
  - _Requirements: 1.1_

- [x] 1.2 配置 Git 和代码所有权
  - 初始化 Git 仓库（如果尚未初始化）
  - 创建 `.gitignore` 文件，排除 `target/`、`bin/`、`node_modules/`、`gen/` 等
  - 创建 `.github/CODEOWNERS` 文件定义代码所有权
  - _Requirements: 1.5, 1.7_

- [x] 1.3 创建根级配置文件
  - 创建 `Makefile` 用于统一构建命令
  - 创建 `README.md` 说明项目结构和快速开始
  - 创建 `.github/pull_request_template.md`
  - _Requirements: 1.4, 12.1_

- [x] 2. API 契约定义
  - 定义 Protobuf 接口
  - 配置代码生成
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 2.1 创建 Protobuf 定义目录
  - 创建 `api/v1/` 目录
  - 创建 `api/v1/README.md` 说明 API 版本和使用方式
  - _Requirements: 5.1_

- [x] 2.2 定义 Hello 服务 Protobuf
  - 创建 `api/v1/hello.proto`
  - 定义 `HelloService`、`HelloRequest`、`HelloResponse`
  - 添加清晰的注释说明
  - _Requirements: 5.2, 5.4_

- [x] 2.3 定义 TODO 服务 Protobuf
  - 创建 `api/v1/todo.proto`
  - 定义 `TodoService`、`Todo`、CRUD 请求/响应消息
  - 添加清晰的注释说明
  - _Requirements: 5.3, 5.4_

- [x] 2.4 配置 Protobuf 代码生成
  - 在 `Makefile` 中添加 `gen-proto` 目标
  - 配置 `gen-proto-go`、`gen-proto-java`、`gen-proto-ts` 子目标
  - 添加 `verify-proto` 目标用于 CI 验证
  - 创建 `.git/hooks/pre-commit` hook
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 3. Hello 服务实现（Java/Spring Boot）
  - 初始化 Java 项目
  - 实现 gRPC 服务
  - 配置和测试
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 7.1, 7.2, 7.4, 7.5_

- [ ] 3.1 初始化 Hello 服务项目
  - 创建 `apps/hello-service/` 目录
  - 创建 `pom.xml` 或 `build.gradle`，配置 Spring Boot 和 gRPC 依赖
  - 配置 Protobuf Maven/Gradle 插件
  - 创建 `src/main/resources/application.yml`
  - _Requirements: 2.1, 2.2, 2.4_

- [ ] 3.2 生成 Java Protobuf 代码
  - 运行 `make gen-proto-java`
  - 验证生成的代码在 `apps/hello-service/src/main/java-gen/`
  - 配置 `build.gradle` 引用生成的代码
  - _Requirements: 6.1, 6.5_

- [ ] 3.3 实现 HelloServiceImpl
  - 创建 `com.myorg.hello.service.HelloServiceImpl`
  - 实现 `sayHello` 方法：非空名字返回包含名字的问候，空名字返回默认问候
  - 使用 `@GrpcService` 注解注册服务
  - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [ ] 3.4 创建 Spring Boot 主应用类
  - 创建 `HelloServiceApplication.java`
  - 配置 gRPC 服务器端口（9090）
  - _Requirements: 2.3, 7.5_

- [ ] 3.5 创建 Dockerfile
  - 创建 `apps/hello-service/Dockerfile`
  - 使用多阶段构建（Maven build + JRE runtime）
  - _Requirements: 11.7_

- [ ] 3.6 创建 K8s 资源
  - 创建 `apps/hello-service/k8s/deployment.yaml`
  - 创建 `apps/hello-service/k8s/service.yaml`
  - 创建 `apps/hello-service/k8s/configmap.yaml`
  - 配置环境变量、资源限制、健康检查
  - _Requirements: 2.5, 7.5_

- [ ] 3.7 创建 catalog-info.yaml
  - 创建 `apps/hello-service/catalog-info.yaml`
  - 定义服务元数据、所有权、API 引用
  - _Requirements: 12.6_

- [ ]* 3.8 编写 Hello 服务单元测试
  - 测试 `sayHello` 方法的各种输入情况
  - 测试空名字返回默认消息
  - 使用 JUnit 5 + Mockito
  - _Requirements: 7.1, 7.2_

- [ ]* 3.9 编写 Hello 服务属性测试
  - **Property 1: Hello Service Name Inclusion**
  - 使用 jqwik 生成随机名字字符串
  - 验证响应消息包含输入的名字
  - _Requirements: 7.1_

- [ ] 4. TODO 服务实现（Go）
  - 初始化 Go 项目
  - 实现 gRPC 服务和存储
  - 配置和测试
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 8.1, 8.2, 8.3, 8.4, 8.6, 8.7_

- [ ] 4.1 初始化 TODO 服务项目
  - 创建 `apps/todo-service/` 目录
  - 初始化 `go.mod`，配置模块路径
  - 添加 gRPC 和 Protobuf 依赖
  - _Requirements: 3.1, 3.4_

- [ ] 4.2 生成 Go Protobuf 代码
  - 运行 `make gen-proto-go`
  - 验证生成的代码在 `apps/todo-service/gen/`
  - _Requirements: 6.2, 6.6_

- [ ] 4.3 实现内存存储
  - 创建 `storage/memory_store.go`
  - 实现 `TodoStore` 接口（Create, Get, List, Update, Delete）
  - 使用 `sync.RWMutex` 保证并发安全
  - _Requirements: 8.6_

- [ ] 4.4 实现 TodoServiceServer
  - 创建 `service/todo_service.go`
  - 实现 `CreateTodo`：生成唯一 ID，保存到存储
  - 实现 `ListTodos`：返回所有 TODO 项
  - 实现 `UpdateTodo`：更新指定 TODO 项
  - 实现 `DeleteTodo`：删除指定 TODO 项
  - 添加输入验证（空标题返回 INVALID_ARGUMENT）
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 4.5 实现 Hello 服务客户端（服务间通信）
  - 创建 `client/hello_client.go`
  - 从环境变量读取 `HELLO_SERVICE_ADDR`
  - 实现 `SayHello` 方法调用
  - 添加超时和重试逻辑
  - _Requirements: 8.1_

- [ ] 4.6 创建主服务器
  - 创建 `main.go`
  - 初始化 gRPC 服务器，监听 9091 端口
  - 注册 TodoServiceServer
  - 添加优雅关闭逻辑
  - _Requirements: 3.2, 3.5, 8.7_

- [ ] 4.7 创建 Dockerfile
  - 创建 `apps/todo-service/Dockerfile`
  - 使用多阶段构建（Go build + Alpine runtime）
  - _Requirements: 11.7_

- [ ] 4.8 创建 K8s 资源
  - 创建 `apps/todo-service/k8s/deployment.yaml`
  - 创建 `apps/todo-service/k8s/service.yaml`
  - 配置 `HELLO_SERVICE_ADDR` 环境变量
  - 配置资源限制、健康检查
  - _Requirements: 3.5, 8.7_

- [ ] 4.9 创建 catalog-info.yaml
  - 创建 `apps/todo-service/catalog-info.yaml`
  - 定义服务元数据、依赖关系（consumesApis: hello-api）
  - _Requirements: 12.6_

- [ ]* 4.10 编写 TODO 服务单元测试
  - 测试内存存储的 CRUD 操作
  - 测试并发访问场景
  - 使用 Go testing + testify
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ]* 4.11 编写 TODO 服务属性测试
  - **Property 2: TODO Creation Returns Unique IDs**
  - **Property 3: TODO CRUD Round-Trip Consistency**
  - 使用 gopter 或 rapid 生成随机 TODO 数据
  - 验证 ID 唯一性和 CRUD 一致性
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ] 5. 前端应用实现（React/TypeScript）
  - 初始化 React 项目
  - 实现 UI 组件
  - 集成 gRPC-Web 客户端
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7_

- [ ] 5.1 初始化前端项目
  - 创建 `apps/web/` 目录
  - 使用 Vite 初始化 React + TypeScript 项目
  - 配置 `package.json`，添加 React Query、grpc-web 等依赖
  - 配置 `tsconfig.json`
  - _Requirements: 4.1, 4.2, 4.4_

- [ ] 5.2 生成 TypeScript Protobuf 代码
  - 运行 `make gen-proto-ts`
  - 验证生成的代码在 `apps/web/src/gen/`
  - 配置 TypeScript 路径别名
  - _Requirements: 6.3, 6.7_

- [ ] 5.3 配置 gRPC-Web 客户端
  - 创建 `src/services/helloClient.ts`
  - 创建 `src/services/todoClient.ts`
  - 配置客户端指向 Envoy 代理路径（`/api/hello`, `/api/todo`）
  - _Requirements: 7.6, 8.8_

- [ ] 5.4 实现 Hello 表单组件
  - 创建 `src/components/HelloForm.tsx`
  - 实现输入框和提交按钮
  - 调用 Hello 服务并显示响应
  - 添加加载状态和错误处理
  - _Requirements: 9.1, 9.2, 9.7_

- [ ] 5.5 实现 TODO 列表组件
  - 创建 `src/components/TodoList.tsx`
  - 使用 React Query 获取 TODO 列表
  - 显示所有 TODO 项
  - 为每个 TODO 项添加更新和删除按钮
  - _Requirements: 9.3, 9.5_

- [ ] 5.6 实现 TODO 创建组件
  - 创建 `src/components/TodoForm.tsx`
  - 实现标题和描述输入
  - 调用 CreateTodo 服务
  - 成功后刷新列表
  - _Requirements: 9.4, 9.6_

- [ ] 5.7 实现 TODO 操作 hooks
  - 创建 `src/hooks/useTodos.ts`
  - 使用 React Query 实现 CRUD mutations
  - 实现乐观更新和缓存失效
  - _Requirements: 9.6_

- [ ] 5.8 实现主应用组件
  - 创建 `src/App.tsx`
  - 组合 HelloForm 和 TodoList 组件
  - 添加基本样式和布局
  - _Requirements: 9.1, 9.3_

- [ ] 5.9 配置 Vite 开发代理
  - 在 `vite.config.ts` 中配置代理
  - 将 `/api/*` 代理到本地 Envoy（`http://localhost:8080`）
  - _Requirements: 4.5, 11.4_

- [ ]* 5.10 编写前端组件测试
  - 使用 Vitest + React Testing Library
  - 测试 HelloForm 和 TodoList 组件
  - Mock gRPC 客户端
  - _Requirements: 9.2, 9.6_

- [ ]* 5.11 编写前端错误处理属性测试
  - **Property 4: Frontend Error Handling**
  - 使用 fast-check 生成随机错误场景
  - 验证 UI 显示错误消息而不崩溃
  - _Requirements: 9.7_

- [ ] 6. API 网关和基础设施
  - 配置 Higress/Envoy
  - 设置 K8s 资源
  - 配置 CI/CD
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 11.1, 11.2, 11.5, 11.6, 11.7_

- [ ] 6.1 配置本地 Envoy 代理
  - 创建 `tools/envoy/envoy-local.yaml`
  - 配置路由规则（`/api/hello` → `localhost:9090`, `/api/todo` → `localhost:9091`）
  - 配置 gRPC-Web 过滤器和 CORS
  - _Requirements: 10.2, 10.3, 10.4, 10.5_

- [ ] 6.2 创建开发启动脚本
  - 创建 `scripts/dev.sh`
  - 启动 Envoy、Hello 服务、TODO 服务、前端
  - 添加优雅关闭逻辑
  - _Requirements: 11.5_

- [ ] 6.3 配置 Higress Ingress
  - 创建 `tools/k8s/ingress.yaml`
  - 配置路由规则和 gRPC 后端协议
  - 配置 CORS 注解
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 6.4 配置 Kustomize
  - 创建 `k8s/base/kustomization.yaml`
  - 聚合所有服务的 K8s 资源
  - 创建 `k8s/overlays/production/` 用于生产配置
  - _Requirements: 11.2_

- [ ] 6.5 配置 CI/CD 流水线
  - 创建 `.github/workflows/ci.yml`
  - 配置测试、构建、Docker 镜像推送
  - 配置 K8s 部署步骤
  - _Requirements: 11.1, 11.2, 11.7_

- [ ] 6.6 配置代码格式化和 Lint
  - 配置 Java Checkstyle/SpotBugs
  - 配置 Go golangci-lint
  - 配置 TypeScript ESLint + Prettier
  - 添加到 CI 流水线
  - _Requirements: 11.6_

- [ ] 7. 服务模板和文档
  - 创建可复用的服务模板
  - 编写项目文档
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 13.1, 13.2, 13.3, 13.4, 13.5_

- [ ] 7.1 创建 Java 服务模板
  - 复制 Hello 服务作为模板到 `templates/java-service/`
  - 泛化配置和代码
  - 添加 README 说明如何使用模板
  - _Requirements: 13.1, 13.2_

- [ ] 7.2 创建 Go 服务模板
  - 复制 TODO 服务作为模板到 `templates/go-service/`
  - 泛化配置和代码
  - 添加 README 说明如何使用模板
  - _Requirements: 13.1, 13.2_

- [ ] 7.3 更新根 README
  - 说明项目结构和架构
  - 添加快速开始指南（运行 `scripts/dev.sh`）
  - 添加构建和部署说明
  - 添加如何添加新服务的指南
  - _Requirements: 12.1, 12.2, 12.3, 12.4_

- [ ] 7.4 创建 API 文档
  - 在 `api/v1/README.md` 中说明 Protobuf 定义
  - 添加使用示例和注释
  - _Requirements: 12.5_

- [ ] 7.5 创建架构图
  - 使用 Mermaid 或其他工具创建架构图
  - 说明南北向和东西向流量
  - 添加到 README
  - _Requirements: 12.6_

- [ ] 7.6 创建治理文档
  - 创建 `docs/governance.md`
  - 说明代码所有权、PR 流程、健康度指标
  - _Requirements: 13.4, 13.5, 13.6_

- [ ] 8. 最终验证和部署
  - 端到端测试
  - 部署到 K8s
  - _Requirements: 11.1, 11.2, 11.7_

- [ ] 8.1 本地端到端测试
  - 运行 `scripts/dev.sh` 启动所有服务
  - 在浏览器中测试 Hello 表单
  - 测试 TODO 的创建、列表、更新、删除
  - 验证服务间通信（TODO 调用 Hello）
  - _Requirements: 11.5_

- [ ] 8.2 构建 Docker 镜像
  - 运行 `make docker-build`
  - 验证镜像构建成功
  - _Requirements: 11.7_

- [ ] 8.3 部署到 K8s 集群
  - 应用 Kustomize 配置：`kubectl apply -k k8s/overlays/production`
  - 验证所有 Pod 运行正常
  - 验证 Higress Ingress 配置
  - _Requirements: 11.2_

- [ ] 8.4 验证生产环境
  - 通过 Higress 网关访问服务
  - 测试前端功能
  - 检查日志和监控
  - _Requirements: 10.1, 10.2, 10.3_

## Notes

- 标记 `*` 的任务为可选任务，可以跳过以加快 MVP 开发
- 每个任务都引用了对应的需求编号，便于追溯
- 建议按顺序执行任务，因为存在依赖关系
- 属性测试任务包含了设计文档中定义的正确性属性
- 完成每个主要阶段后，建议进行检查点验证
