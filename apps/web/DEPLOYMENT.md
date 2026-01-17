# 前后端通信架构说明

本文档详细说明在不同环境（本地开发、测试、生产）下，前端应用如何与后端服务通信。

## 架构概览

系统采用 **南北向网关 + 东西向直连** 的通信模式：

- **南北向（North-South）**：前端 → 网关 → 后端服务
- **东西向（East-West）**：服务间直连 gRPC 通信

```
┌─────────────────────────────────────────────────────────────┐
│                    浏览器 (Browser)                          │
│               React Application (Port 5173)                  │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP/gRPC-Web
                     │
        ┌────────────┴────────────┐
        │                         │
    本地开发                  测试/生产
        │                         │
        ▼                         ▼
┌──────────────────┐    ┌──────────────────┐
│  Vite Dev Proxy  │    │  Higress Gateway │
│  (localhost:8080)│    │  (K8s Ingress)   │
└────────┬─────────┘    └────────┬─────────┘
         │                       │
         │ gRPC                  │ gRPC
         ▼                       ▼
┌──────────────────────────────────────────┐
│         后端服务 (Backend Services)        │
│  ┌──────────────┐    ┌──────────────┐   │
│  │ Hello Service│◄──►│ TODO Service │   │
│  │  Port: 9090  │gRPC│  Port: 9091  │   │
│  └──────────────┘    └──────────────┘   │
└──────────────────────────────────────────┘
```

## 1. 本地开发环境 (Local Development)

### 1.1 架构

```
浏览器 (localhost:5173)
    ↓ HTTP Request to /api/hello or /api/todo
Vite Dev Server (localhost:5173)
    ↓ Proxy to localhost:8080
Envoy Proxy (localhost:8080)
    ↓ gRPC-Web → gRPC 转换
后端服务
    - Hello Service: localhost:9090
    - TODO Service: localhost:9091
```

### 1.2 配置详情

#### 前端配置 (vite.config.ts)

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/hello': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/todo': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

**工作原理**：
1. 前端代码调用 `/api/hello/api.v1.HelloService/SayHello`
2. Vite 开发服务器拦截请求，代理到 `http://localhost:8080`
3. Envoy 接收 gRPC-Web 请求，转换为 gRPC
4. Envoy 路由到对应的后端服务

#### Envoy 本地配置 (tools/envoy/envoy-local.yaml)

```yaml
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
          route_config:
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
                    regex: ".*"
                allow_methods: "GET, POST, PUT, DELETE, OPTIONS"
                allow_headers: "content-type,x-grpc-web,x-user-agent"
                expose_headers: "grpc-status,grpc-message"
          http_filters:
          - name: envoy.filters.http.grpc_web
          - name: envoy.filters.http.cors
          - name: envoy.filters.http.router

  clusters:
  - name: hello_service
    connect_timeout: 0.25s
    type: STRICT_DNS
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

### 1.3 启动步骤

```bash
# 终端 1: 启动 Envoy 代理
docker run -d --name envoy-local --network host \
  -v $(pwd)/tools/envoy:/config \
  envoyproxy/envoy:v1.30 -c /config/envoy-local.yaml

# 终端 2: 启动 Hello Service
cd apps/hello-service
./gradlew bootRun
# 监听端口: 9090

# 终端 3: 启动 TODO Service
cd apps/todo-service
go run .
# 监听端口: 9091

# 终端 4: 启动前端
cd apps/web
npm run dev
# 访问: http://localhost:5173
```

或使用统一脚本：

```bash
./scripts/dev.sh
```

### 1.4 请求流程示例

**前端调用 Hello Service**：

```typescript
// 1. 前端代码
const request: HelloRequest = { name: "Alice" };
const response = await helloClient.sayHello(request);

// 2. 实际 HTTP 请求
POST http://localhost:5173/api/hello/api.v1.HelloService/SayHello
Content-Type: application/grpc-web+proto

// 3. Vite 代理转发
POST http://localhost:8080/api/hello/api.v1.HelloService/SayHello

// 4. Envoy 转换并路由
gRPC call to localhost:9090 (Hello Service)

// 5. Hello Service 处理并返回
gRPC response → Envoy → Vite → Browser
```

## 2. 测试环境 (Testing/Staging)

### 2.1 架构

```
浏览器
    ↓ HTTPS
CDN / Load Balancer (test.example.com)
    ↓
Kubernetes Cluster
    ↓
Higress Ingress Gateway
    ↓ gRPC
Backend Services (K8s Pods)
    - Hello Service (ClusterIP: hello-service:9090)
    - TODO Service (ClusterIP: todo-service:9091)
```

### 2.2 配置详情

#### 前端构建配置

测试环境使用**静态构建**，不需要 Vite 代理：

```bash
# 构建前端
cd apps/web
npm run build

# 输出到 dist/ 目录
# 部署到 Nginx/CDN
```

#### 前端服务配置

前端代码中的 API 路径保持不变（`/api/hello`, `/api/todo`），由 Higress 网关处理路由。

**Nginx 配置示例** (如果使用 Nginx 托管前端)：

```nginx
server {
    listen 80;
    server_name test.example.com;

    # 前端静态文件
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    # API 请求代理到 Higress
    location /api/ {
        proxy_pass http://higress-gateway.higress-system.svc.cluster.local;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # gRPC-Web headers
        proxy_set_header Content-Type application/grpc-web+proto;
        proxy_set_header X-Grpc-Web 1;
    }
}
```

#### Higress Ingress 配置

```yaml
# k8s/overlays/testing/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monorepo-ingress
  namespace: testing
  annotations:
    # Higress 特定注解
    higress.io/backend-protocol: "GRPC"
    higress.io/cors-allow-origin: "https://test.example.com"
    higress.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
    higress.io/cors-allow-headers: "content-type,x-grpc-web,x-user-agent"
    higress.io/cors-expose-headers: "grpc-status,grpc-message"
    
    # TLS 配置
    cert-manager.io/cluster-issuer: "letsencrypt-staging"
spec:
  ingressClassName: higress
  tls:
  - hosts:
    - test.example.com
    secretName: test-tls-cert
  rules:
  - host: test.example.com
    http:
      paths:
      # Hello Service 路由
      - path: /api/hello
        pathType: Prefix
        backend:
          service:
            name: hello-service
            port:
              number: 9090
      
      # TODO Service 路由
      - path: /api/todo
        pathType: Prefix
        backend:
          service:
            name: todo-service
            port:
              number: 9091
```

#### Kubernetes Service 配置

```yaml
# Hello Service
apiVersion: v1
kind: Service
metadata:
  name: hello-service
  namespace: testing
spec:
  type: ClusterIP
  ports:
  - port: 9090
    targetPort: 9090
    protocol: TCP
    name: grpc
  selector:
    app: hello-service

---
# TODO Service
apiVersion: v1
kind: Service
metadata:
  name: todo-service
  namespace: testing
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

### 2.3 部署步骤

```bash
# 1. 构建 Docker 镜像
make docker-build

# 2. 推送到镜像仓库
docker tag hello-service:latest registry.example.com/hello-service:test-v1.0.0
docker tag todo-service:latest registry.example.com/todo-service:test-v1.0.0
docker push registry.example.com/hello-service:test-v1.0.0
docker push registry.example.com/todo-service:test-v1.0.0

# 3. 部署到 K8s 测试环境
kubectl apply -k k8s/overlays/testing

# 4. 验证部署
kubectl get pods -n testing
kubectl get ingress -n testing

# 5. 构建并部署前端
cd apps/web
npm run build
# 将 dist/ 目录内容部署到 CDN 或 Nginx
```

### 2.4 请求流程示例

```
浏览器
    ↓ HTTPS GET https://test.example.com/
CDN/Nginx
    ↓ 返回 index.html 和静态资源

浏览器执行 JavaScript
    ↓ POST https://test.example.com/api/hello/api.v1.HelloService/SayHello
Higress Ingress
    ↓ 路由到 hello-service:9090
Hello Service Pod
    ↓ 处理 gRPC 请求
    ↓ 返回响应
Higress → Browser
```

## 3. 生产环境 (Production)

### 3.1 架构

生产环境与测试环境类似，但增加了高可用和安全配置：

```
用户浏览器
    ↓ HTTPS
全球 CDN (CloudFlare/Akamai)
    ↓
负载均衡器 (ALB/NLB)
    ↓
Kubernetes 集群 (多可用区)
    ↓
Higress Ingress (多副本)
    ↓ gRPC (mTLS)
后端服务 (多副本 + HPA)
    - Hello Service (3+ replicas)
    - TODO Service (3+ replicas)
```

### 3.2 关键差异

#### 高可用配置

```yaml
# k8s/overlays/production/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monorepo-ingress
  namespace: production
  annotations:
    higress.io/backend-protocol: "GRPC"
    
    # 生产环境 CORS 配置（限制来源）
    higress.io/cors-allow-origin: "https://app.example.com"
    
    # 速率限制
    higress.io/rate-limit: "1000r/s"
    
    # 超时配置
    higress.io/proxy-connect-timeout: "5s"
    higress.io/proxy-send-timeout: "30s"
    higress.io/proxy-read-timeout: "30s"
    
    # TLS 配置
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: higress
  tls:
  - hosts:
    - app.example.com
    - api.example.com
    secretName: prod-tls-cert
  rules:
  - host: app.example.com
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

#### 服务副本和自动扩缩容

```yaml
# Hello Service Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-service
  namespace: production
spec:
  replicas: 3  # 最少 3 个副本
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: hello-service
        image: registry.example.com/hello-service:v1.0.0
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"

---
# HPA (Horizontal Pod Autoscaler)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hello-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 3.3 CDN 配置

前端静态资源部署到 CDN，API 请求通过 CDN 回源到 Higress：

```
# CloudFlare Page Rules 示例
app.example.com/*
  - Cache Level: Standard
  - Browser Cache TTL: 4 hours

app.example.com/api/*
  - Cache Level: Bypass
  - Origin: https://api.example.com
```

### 3.4 监控和日志

```yaml
# ServiceMonitor for Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hello-service
  namespace: production
spec:
  selector:
    matchLabels:
      app: hello-service
  endpoints:
  - port: metrics
    interval: 30s
```

## 4. 环境对比总结

| 特性 | 本地开发 | 测试环境 | 生产环境 |
|------|---------|---------|---------|
| **前端服务器** | Vite Dev Server (5173) | Nginx/CDN | 全球 CDN |
| **API 网关** | Envoy (localhost:8080) | Higress Ingress | Higress Ingress (多副本) |
| **后端服务** | 本地进程 | K8s Pods (1-2 副本) | K8s Pods (3+ 副本 + HPA) |
| **协议** | HTTP → gRPC-Web → gRPC | HTTPS → gRPC-Web → gRPC | HTTPS → gRPC-Web → gRPC (mTLS) |
| **域名** | localhost | test.example.com | app.example.com |
| **TLS** | 无 | Let's Encrypt Staging | Let's Encrypt Production |
| **CORS** | 允许所有来源 | 限制测试域名 | 严格限制生产域名 |
| **速率限制** | 无 | 宽松 | 严格 (1000 req/s) |
| **监控** | 本地日志 | 基础监控 | 完整监控 + 告警 |
| **日志** | Console | ELK/Loki | ELK/Loki + 长期存储 |

## 5. 故障排查

### 5.1 本地开发常见问题

**问题：前端无法连接后端**

```bash
# 检查 Envoy 是否运行
docker ps | grep envoy

# 检查后端服务是否启动
curl http://localhost:9090  # Hello Service
curl http://localhost:9091  # TODO Service

# 检查 Vite 代理配置
# 查看浏览器 Network 面板，确认请求被代理到 localhost:8080
```

**问题：CORS 错误**

检查 Envoy 配置中的 CORS 设置，确保包含必要的 headers。

### 5.2 测试/生产环境常见问题

**问题：502 Bad Gateway**

```bash
# 检查 Ingress 状态
kubectl get ingress -n testing
kubectl describe ingress monorepo-ingress -n testing

# 检查后端服务
kubectl get pods -n testing
kubectl logs -f <pod-name> -n testing

# 检查 Service
kubectl get svc -n testing
kubectl describe svc hello-service -n testing
```

**问题：gRPC 请求失败**

```bash
# 检查 Higress 注解
kubectl get ingress monorepo-ingress -n testing -o yaml | grep annotations -A 10

# 确认 backend-protocol 设置为 GRPC
# 确认 CORS headers 正确配置
```

## 6. 最佳实践

### 6.1 开发阶段

1. 使用 Envoy 本地代理，模拟生产环境路由
2. 保持前端 API 路径与生产环境一致
3. 使用 Docker Compose 统一管理本地服务

### 6.2 部署阶段

1. 使用 Kustomize 管理多环境配置
2. 通过 CI/CD 自动化部署流程
3. 金丝雀发布：先部署到测试环境验证

### 6.3 运维阶段

1. 配置完善的监控和告警
2. 定期进行压力测试
3. 建立应急响应流程

## 7. 相关文档

- [Higress 官方文档](https://higress.io/docs/)
- [Envoy Proxy 文档](https://www.envoyproxy.io/docs/)
- [gRPC-Web 规范](https://github.com/grpc/grpc-web)
- [Kubernetes Ingress 文档](https://kubernetes.io/docs/concepts/services-networking/ingress/)
