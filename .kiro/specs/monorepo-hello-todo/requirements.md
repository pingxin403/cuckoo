# Requirements Document

## Introduction

本文档定义了一个多语言 Monorepo 项目的初始化需求，以及 Hello 服务和 TODO 服务的功能需求。该项目采用 Monorepo 架构模式，支持前后端代码共存，Hello 服务使用 Java/Spring Boot 实现，TODO 服务使用 Go 实现，前端使用 React + TypeScript，使用 Protobuf 作为统一的接口契约，并提供高效的构建和开发体验。

## Goals & Non-Goals

### Goals
- 建立统一的 Monorepo 代码仓库，支持多语言服务共存
- 实现基于 Protobuf 的统一 API 契约层
- 提供高效的增量构建和开发体验
- 支持独立的服务构建、测试和部署
- 建立清晰的代码组织和所有权模型

### Non-Goals
- 本阶段不涉及生产环境的部署和运维
- 不涉及复杂的微服务治理（服务发现、熔断等）
- 不涉及多租户或权限管理系统
- 不涉及性能优化和压力测试

## Technology Stack & Constraints

| 能力 | 选型 | 理由 |
|------|------|------|
| 构建系统 | Bazel 或 Makefile + 脚本 | 支持跨语言增量构建、确定性输出 |
| Java 构建 | Maven 或 Gradle | 成熟的 Java 生态系统支持 |
| Go 构建 | Go modules + go build | Go 官方工具链，简单高效 |
| 前端构建 | Vite + npm/pnpm | 开发体验优，生产构建快速 |
| Protobuf 生成 | ts-proto（TS）、protoc-gen-go（Go）、protoc-gen-grpc-java（Java） | 类型安全、支持 gRPC |
| API 网关 | Envoy 或 Nginx | 支持 gRPC-Web 代理，轻量级 |
| 本地开发协调 | 自研脚本（dev.sh） | 同时启动多个服务，支持热更新 |

## Architectural Principles

1. **单一事实源（Single Source of Truth）**
   - 所有接口契约必须定义在 `api/` 目录，禁止在服务内重复定义

2. **零跨应用源码依赖**
   - `apps/hello-service` 不得直接 import `apps/todo-service` 的源码，只能通过 `api/` 或 `libs/` 公共模块交互

3. **构建隔离性**
   - 每个应用必须能独立构建、测试、部署，不依赖其他应用的存在

4. **开发者体验分层**
   - 开发时：使用原生工具链（Vite、go run、Spring Boot DevTools）获得最佳体验
   - 构建/发布时：使用统一构建系统，确保一致性

5. **契约先行（Contract-First）**
   - 接口变更必须先修改 `api/*.proto`，再生成代码，最后实现逻辑

## Glossary

- **Monorepo**: 单一代码仓库，包含多个相关项目（前端和多个后端服务）
- **Build_System**: 构建系统，负责编译、测试和打包代码（Bazel 或其他工具）
- **API_Contract**: API 契约层，使用 Protobuf 定义的前后端接口规范
- **Hello_Service**: Hello 服务，使用 Java/Spring Boot 实现的问候功能服务
- **TODO_Service**: TODO 服务，使用 Go 实现的任务管理功能服务
- **Web_App**: React 前端应用
- **Java_Server**: Java/Spring Boot 后端服务器，运行 Hello_Service
- **Go_Server**: Go 后端服务器，运行 TODO_Service
- **Protobuf**: Protocol Buffers，用于序列化结构化数据的语言中立机制
- **Code_Generator**: 代码生成器，从 Protobuf 定义自动生成 Java、Go 和 TypeScript 代码
- **API_Gateway**: API 网关或代理层，统一前端对多个后端服务的访问

## Requirements

### Requirement 1: Monorepo 项目结构初始化

**User Story:** 作为开发者，我希望初始化一个标准的 Monorepo 项目结构，以便能够在统一的仓库中管理前后端代码。

#### Acceptance Criteria

1. THE Build_System SHALL 创建根目录结构，包含 `api/`、`apps/`、`libs/`、`tools/` 和 `scripts/` 目录
2. THE Build_System SHALL 在 `api/` 目录下创建 Protobuf 接口定义的存放位置
3. THE Build_System SHALL 在 `apps/` 目录下创建 `web/`（React）、`hello-service/`（Java）和 `todo-service/`（Go）子目录
4. THE Build_System SHALL 配置构建工具的根配置文件（WORKSPACE 或等效配置）
5. THE Build_System SHALL 创建 `.gitignore` 文件，排除构建产物和依赖目录（包括 Java 的 target/、Go 的 bin/、Node 的 node_modules/ 等）
6. THE Build_System SHALL 支持多语言项目（Java、Go、TypeScript）的统一构建和依赖管理
7. THE Build_System SHALL 创建 CODEOWNERS 文件，定义各目录的代码所有权

#### Directory Structure

```
my-monorepo/
├── WORKSPACE                     # Bazel workspace root (或等效配置)
├── .bazelrc                      # Bazel 配置 (可选)
├── .gitignore
├── CODEOWNERS                    # 代码所有权定义
├── README.md
│
├── api/                          # API 契约层 (owned by @platform-team)
│   └── v1/
│       ├── hello.proto
│       ├── todo.proto
│       └── BUILD.bazel          # 代码生成规则
│
├── apps/
│   ├── web/                      # React 应用 (owned by @frontend-team)
│   │   ├── src/
│   │   ├── package.json
│   │   └── BUILD.bazel
│   │
│   ├── hello-service/            # Java 服务 (owned by @backend-java)
│   │   ├── src/main/java/
│   │   ├── pom.xml 或 build.gradle
│   │   └── BUILD.bazel
│   │
│   └── todo-service/             # Go 服务 (owned by @backend-go)
│       ├── main.go
│       ├── go.mod
│       └── BUILD.bazel
│
├── libs/                         # 共享库 (可选)
│   ├── common-java/
│   └── common-ts/
│
├── tools/                        # 构建时工具 (protoc 插件、linters)
└── scripts/
    ├── dev.sh                    # 启动所有服务的开发模式
    └── build-all.sh              # 构建脚本
```

### Requirement 2: Java/Spring Boot Hello 服务初始化

**User Story:** 作为后端开发者，我希望初始化 Java/Spring Boot Hello 服务的基础结构，以便能够开始实现业务逻辑。

#### Acceptance Criteria

1. THE Java_Server SHALL 在 `apps/hello-service/` 目录下初始化 Maven 或 Gradle 项目
2. THE Java_Server SHALL 配置 Spring Boot 依赖和 gRPC 服务器依赖
3. THE Java_Server SHALL 创建主应用类（Application.java）和配置文件（application.yml 或 application.properties）
4. THE Java_Server SHALL 配置 Protobuf 和 gRPC 相关的 Maven/Gradle 插件
5. WHEN Java_Server 启动时 THEN THE Java_Server SHALL 监听指定端口并输出启动日志

### Requirement 3: Go TODO 服务初始化

**User Story:** 作为后端开发者，我希望初始化 Go TODO 服务的基础结构，以便能够开始实现业务逻辑。

#### Acceptance Criteria

1. THE Go_Server SHALL 在 `apps/todo-service/` 目录下初始化 Go 模块（go.mod）
2. THE Go_Server SHALL 创建 `main.go` 入口文件
3. THE Go_Server SHALL 配置 gRPC 服务器的基础框架
4. THE Go_Server SHALL 包含必要的依赖项（gRPC、Protobuf 运行时等）
5. WHEN Go_Server 启动时 THEN THE Go_Server SHALL 监听指定端口并输出启动日志

### Requirement 4: React 前端应用初始化

**User Story:** 作为前端开发者，我希望初始化 React 前端应用的基础结构，以便能够开始开发用户界面。

#### Acceptance Criteria

1. THE Web_App SHALL 在 `apps/web/` 目录下初始化 React + TypeScript 项目
2. THE Web_App SHALL 配置 `package.json` 包含必要的依赖（React、TypeScript、构建工具等）
3. THE Web_App SHALL 创建基础的项目结构（src/、public/ 等）
4. THE Web_App SHALL 配置 TypeScript 编译选项（tsconfig.json）
5. WHEN Web_App 启动开发服务器时 THEN THE Web_App SHALL 在浏览器中显示默认页面

### Requirement 5: Protobuf API 契约定义

**User Story:** 作为系统架构师，我希望定义统一的 API 契约，以便前后端能够基于相同的接口规范进行开发。

#### Acceptance Criteria

1. THE API_Contract SHALL 在 `api/v1/` 目录下创建 Protobuf 定义文件
2. THE API_Contract SHALL 定义 Hello 服务的接口（请求和响应消息）
3. THE API_Contract SHALL 定义 TODO 服务的接口（CRUD 操作的消息类型）
4. THE API_Contract SHALL 使用 gRPC 服务定义声明服务方法
5. THE API_Contract SHALL 包含清晰的注释说明每个消息和服务的用途

### Requirement 6: 代码生成配置

**User Story:** 作为开发者，我希望配置自动代码生成，以便从 Protobuf 定义自动生成 Java、Go 和 TypeScript 代码。

#### Acceptance Criteria

1. THE Code_Generator SHALL 配置从 Protobuf 生成 Java 代码的规则（使用 protoc-gen-grpc-java）
2. THE Code_Generator SHALL 配置从 Protobuf 生成 Go 代码的规则（使用 protoc-gen-go 和 protoc-gen-go-grpc）
3. THE Code_Generator SHALL 配置从 Protobuf 生成 TypeScript 代码的规则（使用 ts-proto 或 grpc-web）
4. WHEN Protobuf 文件被修改时 THEN THE Code_Generator SHALL 能够重新生成对应的代码
5. THE Code_Generator SHALL 将生成的 Java 代码输出到 Maven/Gradle 可识别的路径
6. THE Code_Generator SHALL 将生成的 Go 代码输出到 Go 模块可导入的路径
7. THE Code_Generator SHALL 将生成的 TypeScript 代码输出到前端项目可导入的路径

### Requirement 7: Hello 服务实现（Java/Spring Boot）

**User Story:** 作为用户，我希望调用 Hello 服务获取问候消息，以便验证系统的基本功能。

#### Acceptance Criteria

1. WHEN 用户发送包含姓名的 Hello 请求 THEN THE Hello_Service SHALL 返回包含该姓名的问候消息
2. WHEN 用户发送空姓名的 Hello 请求 THEN THE Hello_Service SHALL 返回默认问候消息
3. THE Hello_Service SHALL 实现 Protobuf 定义的 Hello 服务接口
4. THE Hello_Service SHALL 在 Java_Server 中使用 Spring Boot gRPC 框架注册并可被调用
5. THE Hello_Service SHALL 监听独立的端口（例如 9090）
6. THE Web_App SHALL 能够通过 gRPC-Web 或 REST 代理调用 Hello_Service

### Requirement 8: TODO 服务实现（Go）

**User Story:** 作为用户，我希望管理我的待办事项，以便能够创建、查看、更新和删除任务。

#### Acceptance Criteria

1. WHEN 用户创建新的 TODO 项时 THEN THE TODO_Service SHALL 保存该 TODO 项并返回唯一标识符
2. WHEN 用户查询 TODO 列表时 THEN THE TODO_Service SHALL 返回所有 TODO 项
3. WHEN 用户更新 TODO 项时 THEN THE TODO_Service SHALL 修改对应的 TODO 项并返回更新结果
4. WHEN 用户删除 TODO 项时 THEN THE TODO_Service SHALL 移除对应的 TODO 项并返回删除结果
5. THE TODO_Service SHALL 实现 Protobuf 定义的 TODO 服务接口
6. THE TODO_Service SHALL 使用内存存储或简单的持久化机制保存数据
7. THE TODO_Service SHALL 监听独立的端口（例如 9091）
8. THE Web_App SHALL 能够通过 gRPC-Web 或 REST 代理调用 TODO_Service 的所有方法

### Requirement 9: 前端 UI 实现

**User Story:** 作为用户，我希望通过友好的 Web 界面与 Hello 和 TODO 服务交互，以便方便地使用这些功能。

#### Acceptance Criteria

1. THE Web_App SHALL 提供 Hello 服务的交互界面，包含输入框和提交按钮
2. WHEN 用户在 Hello 界面输入姓名并提交时 THEN THE Web_App SHALL 显示服务返回的问候消息
3. THE Web_App SHALL 提供 TODO 列表的展示界面
4. THE Web_App SHALL 提供创建新 TODO 项的输入界面
5. THE Web_App SHALL 为每个 TODO 项提供更新和删除操作按钮
6. WHEN 用户执行 TODO 操作时 THEN THE Web_App SHALL 实时更新界面显示
7. THE Web_App SHALL 处理服务调用错误并向用户显示友好的错误提示

### Requirement 10: 服务间通信和 API 网关

**User Story:** 作为系统架构师，我希望配置统一的 API 网关或代理层，以便前端能够方便地访问多个后端服务。

#### Acceptance Criteria

1. THE API_Gateway SHALL 提供统一的入口点供前端访问
2. THE API_Gateway SHALL 将 Hello 服务的请求路由到 Java_Server
3. THE API_Gateway SHALL 将 TODO 服务的请求路由到 Go_Server
4. THE API_Gateway SHALL 支持 gRPC-Web 协议或提供 REST 到 gRPC 的转换
5. THE API_Gateway SHALL 处理 CORS 配置，允许前端跨域访问
6. WHERE 使用 Envoy 或 Nginx 作为代理 THEN THE API_Gateway SHALL 配置相应的路由规则

### Requirement 11: 构建和开发工作流

**User Story:** 作为开发者，我希望有高效的构建和开发工作流，以便能够快速迭代和测试代码。

#### Acceptance Criteria

1. THE Build_System SHALL 提供命令用于构建整个项目（包括 Java、Go 和 React）
2. THE Build_System SHALL 提供命令用于单独构建前端、Hello 服务或 TODO 服务
3. THE Build_System SHALL 支持增量构建，仅重新构建变更的部分
4. THE Build_System SHALL 提供开发模式，支持前端热更新
5. THE Build_System SHALL 提供脚本用于同时启动所有服务（Java_Server、Go_Server、Web_App）
6. THE Build_System SHALL 配置代码格式化和 Lint 检查工具（Java、Go、TypeScript）
7. THE Build_System SHALL 支持 Docker 容器化构建和部署

#### Development Workflow

**本地开发（优化开发体验）：**
```bash
# 终端 1: 启动 Java Hello Service
cd apps/hello-service && ./mvnw spring-boot:run
# 或 ./gradlew bootRun

# 终端 2: 启动 Go TODO Service
cd apps/todo-service && go run .

# 终端 3: 启动 React Dev Server (配置代理到后端)
cd apps/web && npm run dev
```

前端使用 Vite 代理将 `/api/hello` 路由到 `localhost:9090`，`/api/todo` 路由到 `localhost:9091`。

**生产构建（使用构建系统）：**
```bash
# 构建所有项目
bazel build //... 或 make build-all

# 仅构建前端
bazel build //apps/web:bundle

# 构建 Docker 镜像
bazel run //apps/todo-service:todo_image
```

**CI 流程：**
1. PR 提交时：运行 `bazel test //...` 或等效测试命令
2. 合并后：构建所有服务并推送镜像

### Requirement 12: 文档和示例

**User Story:** 作为新加入的开发者，我希望有清晰的文档和示例，以便能够快速理解项目结构和开发流程。

#### Acceptance Criteria

1. THE Build_System SHALL 在根目录创建 README.md 文件，说明项目结构和快速开始步骤
2. THE Build_System SHALL 在 README.md 中包含如何运行开发服务器的说明（Java、Go、React）
3. THE Build_System SHALL 在 README.md 中包含如何构建生产版本的说明
4. THE Build_System SHALL 在 README.md 中包含如何添加新服务的指南（支持多语言）
5. THE Build_System SHALL 在 `api/` 目录下提供 Protobuf 定义的示例和注释
6. THE Build_System SHALL 提供多语言服务共存的架构图和说明

### Requirement 13: 扩展性和治理

**User Story:** 作为平台团队成员，我希望建立清晰的扩展和治理机制，以便项目能够健康地扩展。

#### Acceptance Criteria

1. THE Build_System SHALL 提供添加新服务的标准流程文档
2. WHEN 开发者添加新服务时 THEN THE Build_System SHALL 要求在 `apps/` 下创建独立目录
3. WHEN 开发者添加新 API 时 THEN THE Build_System SHALL 要求先在 `api/` 目录定义 Protobuf
4. THE Build_System SHALL 配置 pre-commit hooks 检查代码格式和基本规范
5. THE Build_System SHALL 在 README 中说明代码所有权和审查流程
6. THE Build_System SHALL 提供仓库健康度指标的监控建议（构建时间、代码重复率等）
