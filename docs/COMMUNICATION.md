# 前后端通信说明

## 快速参考

| 环境 | 前端地址 | API 网关 | 后端服务 |
|------|---------|---------|---------|
| **本地开发** | localhost:5173 | Envoy (localhost:8080) | Hello:9090, TODO:9091 |
| **测试环境** | test.example.com | Higress Ingress | K8s ClusterIP Services |
| **生产环境** | app.example.com | Higress Ingress (多副本) | K8s ClusterIP Services (HPA) |

## 通信流程

### 本地开发

```
浏览器 → Vite Dev Server (代理) → Envoy → 后端服务
```

### 测试/生产

```
浏览器 → CDN/Nginx → Higress Ingress → 后端服务 (K8s)
```

## 详细文档

完整的架构和部署说明请参考：
- [apps/web/DEPLOYMENT.md](../apps/web/DEPLOYMENT.md) - 详细的部署和通信架构文档
