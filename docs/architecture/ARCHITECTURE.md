# 系统架构文档

## 整体架构

本系统采用 Monorepo 架构，包含多语言微服务和前端应用。

### 技术栈

- **前端**: React + TypeScript + Vite + gRPC-Web
- **后端服务**:
  - Hello Service: Java + Spring Boot + gRPC
  - TODO Service: Go + gRPC
- **API 网关**: Higress (生产) / Envoy (本地开发)
- **容器编排**: Kubernetes
- **API 契约**: Protocol Buffers (Protobuf)

## 通信模式

### 南北向流量 (North-South)

前端到后端的通信通过 API 网关：

```
浏览器 (React App)
    ↓ HTTP/HTTPS + gRPC-Web
API 网关 (Higress/Envoy)
    ↓ gRPC
后端服务 (Hello/TODO)
```

**特点**：
- 统一入口，便于安全控制和监控
- gRPC-Web 到 gRPC 的协议转换
- 支持 CORS、速率限制、认证等

### 东西向流量 (East-West)

服务间直接通过 gRPC 通信：

```
TODO Service
    ↓ gRPC (直连)
Hello Service
```

**特点**：
- 低延迟，避免网关成为瓶颈
- 基于 K8s Service 的服务发现
- 类型安全（基于 Protobuf）

## 环境架构

### 本地开发环境

```
┌─────────────────────────────────────────────────────┐
│ 开发者机器 (localhost)                               │
│                                                      │
│  ┌──────────────┐                                   │
│  │   浏览器      │                                   │
│  │ :5173        │                                   │
│  └──────┬───────┘                                   │
│         │ HTTP                                      │
│  ┌──────▼───────┐                                   │
│  │ Vite Dev     │                                   │
│  │ Server       │                                   │
│  │ (代理)       │                                   │
│  └──────┬───────┘                                   │
│         │ Proxy to :8080                            │
│  ┌──────▼───────┐                                   │
│  │ Envoy Proxy  │                                   │
│  │ :8080        │                                   │
│  └──┬───────┬───┘                                   │
│     │ gRPC  │ gRPC                                  │
│  ┌──▼───┐ ┌▼────┐                                  │
│  │Hello │ │TODO │                                   │
│  │:9090 │ │:9091│                                   │
│  └──────┘ └─────┘                                   │
└─────────────────────────────────────────────────────┘
```

**启动命令**：
```bash
# 方式 1: 使用统一脚本
./scripts/dev.sh

# 方式 2: 分别启动
# Terminal 1: Envoy
docker run -d --name envoy-local --network host \
  -v $(pwd)/tools/envoy:/config \
  envoyproxy/envoy:v1.30 -c /config/envoy-local.yaml

# Terminal 2: Hello Service
cd apps/hello-service && ./gradlew bootRun

# Terminal 3: TODO Service
cd apps/todo-service && go run .

# Terminal 4: Frontend
cd apps/web && npm run dev
```

### 测试/生产环境

```
┌─────────────────────────────────────────────────────┐
│ 互联网                                               │
└────────────────────┬────────────────────────────────┘
                     │ HTTPS
┌────────────────────▼────────────────────────────────┐
│ CDN / 负载均衡器                                      │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────┐
│ Kubernetes 集群                                      │
│                                                      │
│  ┌──────────────────────────────────────┐           │
│  │ Higress Ingress Gateway              │           │
│  │ (gRPC-Web → gRPC 转换)               │           │
│  └──────┬───────────────────┬───────────┘           │
│         │ gRPC              │ gRPC                  │
│  ┌──────▼───────┐    ┌──────▼───────┐              │
│  │ Hello Service│◄──►│ TODO Service │              │
│  │ (3+ Pods)    │gRPC│ (3+ Pods)    │              │
│  │ ClusterIP    │    │ ClusterIP    │              │
│  │ :9090        │    │ :9091        │              │
│  └──────────────┘    └──────────────┘              │
│                                                      │
│  ┌──────────────────────────────────────┐           │
│  │ 持久化存储 (未来)                     │           │
│  │ - PostgreSQL                         │           │
│  │ - Redis                              │           │
│  └──────────────────────────────────────┘           │
└─────────────────────────────────────────────────────┘
```

**部署命令**：
```bash
# 构建镜像
make docker-build

# 部署到 K8s
kubectl apply -k k8s/overlays/production

# 或使用 ArgoCD (GitOps)
argocd app sync monorepo-platform
```

## 数据流示例

### 场景 1: 用户调用 Hello Service

```
1. 用户在浏览器输入名字 "Alice"
   ↓
2. React 组件调用 helloClient.sayHello({ name: "Alice" })
   ↓
3. 发送 HTTP POST 请求
   URL: /api/hello/api.v1.HelloService/SayHello
   Body: Protobuf 编码的 HelloRequest
   ↓
4. [本地] Vite 代理到 localhost:8080
   [生产] Higress Ingress 接收请求
   ↓
5. Envoy/Higress 转换 gRPC-Web → gRPC
   ↓
6. 路由到 Hello Service (localhost:9090 或 K8s Service)
   ↓
7. Hello Service 处理请求
   - 检查 name 是否为空
   - 生成问候消息: "Hello, Alice!"
   ↓
8. 返回 gRPC 响应
   ↓
9. Envoy/Higress 转换 gRPC → gRPC-Web
   ↓
1