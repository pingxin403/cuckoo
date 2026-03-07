# Redis Cluster 三主三从部署指南

本目录提供 Redis Cluster 三主三从模式的 Docker Compose 部署配置。

## 目录结构

```
deploy/docker/redis-cluster/
├── docker-compose.yml              # Docker Compose 部署配置
├── redis-cluster.conf              # Redis Cluster 配置文件
├── validate-redis-cluster.sh       # 验收测试脚本
└── README.md                       # 本文档
```

## 快速开始

### 1. 启动 Redis Cluster

```bash
# 从项目根目录执行
docker compose -f deploy/docker/redis-cluster/docker-compose.yml up -d
```

### 2. 查看启动日志

```bash
# 查看初始化日志
docker compose -f deploy/docker/redis-cluster/docker-compose.yml logs -f redis-cluster-init

# 查看所有节点日志
docker compose -f deploy/docker/redis-cluster/docker-compose.yml logs -f
```

### 3. 检查集群状态

```bash
# 查看集群信息
docker exec redis-node-1 redis-cli -p 6379 cluster info

# 查看节点列表
docker exec redis-node-1 redis-cli -p 6379 cluster nodes
```

### 4. 运行验收测试

```bash
# 从项目根目录执行
./deploy/docker/redis-cluster/validate-redis-cluster.sh
```

## 架构说明

### 节点规划

| 节点 | 容器名 | 外部端口 | 内部端口 | 角色 |
|------|--------|----------|----------|------|
| Node 1 | redis-node-1 | 10000 | 6379 | Master |
| Node 2 | redis-node-2 | 10001 | 6379 | Master |
| Node 3 | redis-node-3 | 10002 | 6379 | Master |
| Node 4 | redis-node-4 | 10003 | 6379 | Slave (Node 1) |
| Node 5 | redis-node-5 | 10004 | 6379 | Slave (Node 2) |
| Node 6 | redis-node-6 | 10005 | 6379 | Slave (Node 3) |

### 拓扑结构

```
                    ┌─────────────┐
                    │  Client     │
                    │ (localhost) │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
    │ Node 1  │       │ Node 2  │       │ Node 3  │
    │ Master  │       │ Master  │       │ Master  │
    │ :10000  │       │ :10001  │       │ :10002  │
    └────┬────┘       └────┬────┘       └────┬────┘
         │                 │                 │
    ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
    │ Node 4  │       │ Node 5  │       │ Node 6  │
    │ Slave   │       │ Slave   │       │ Slave   │
    │ :10003  │       │ :10004  │       │ :10005  │
    └─────────┘       └─────────┘       └─────────┘
```

### Slot 分配

- 16384 个 slots 自动均匀分配到 3 个 Master 节点
- 每个 Master 约负责 5461 个 slots
- 每个 Master 配置 1 个 Slave 进行数据复制

## 常用命令

### 连接 Redis Cluster

```bash
# 使用 redis-cli 连接（-c 启用 cluster 模式）
docker exec redis-node-1 redis-cli -p 6379 -c

# 从宿主机连接
redis-cli -p 10000 -c
```

### 集群管理

```bash
# 查看集群信息
docker exec redis-node-1 redis-cli -p 6379 cluster info

# 查看节点列表
docker exec redis-node-1 redis-cli -p 6379 cluster nodes

# 查看 Slot 分配
docker exec redis-node-1 redis-cli -p 6379 cluster slots

# 查看指定 Key 所在的 Slot
docker exec redis-node-1 redis-cli -p 6379 cluster keyslot mykey

# 查看指定 Slot 的信息
docker exec redis-node-1 redis-cli -p 6379 cluster slot 1234
```

### 数据操作

```bash
# 设置 Key（自动路由到正确的节点）
docker exec redis-node-1 redis-cli -p 6379 -c SET foo bar

# 获取 Key
docker exec redis-node-1 redis-cli -p 6379 -c GET foo

# 删除 Key
docker exec redis-node-1 redis-cli -p 6379 -c DEL foo
```

### 停止/重启

```bash
# 停止所有节点
docker compose -f deploy/docker/redis-cluster/docker-compose.yml down

# 停止并删除数据（谨慎使用！）
docker compose -f deploy/docker/redis-cluster/docker-compose.yml down -v

# 重启所有节点
docker compose -f deploy/docker/redis-cluster/docker-compose.yml restart

# 重启单个节点
docker compose -f deploy/docker/redis-cluster/docker-compose.yml restart redis-node-1
```

## 配置说明

### redis-cluster.conf 关键配置

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `cluster-enabled` | yes | 启用 Cluster 模式 |
| `cluster-node-timeout` | 5000 | 节点超时时间 (5 秒) |
| `cluster-require-full-coverage` | no | 不要求全部 slot 覆盖即可写 |
| `appendonly` | yes | 启用 AOF 持久化 |
| `appendfsync` | everysec | 每秒同步 AOF |
| `maxmemory` | 512mb | 每节点最大内存 |
| `maxmemory-policy` | allkeys-lru | LRU 淘汰策略 |

### 持久化

- **RDB**: 定时快照 (900/1, 300/10, 60/10000)
- **AOF**: 每秒同步，自动重写
- **混合持久化**: 启用 (aof-use-rdb-preamble yes)

## 验收测试说明

### 自动化验收脚本

`validate-redis-cluster.sh` 执行以下检查：

1. **容器状态检查**: 验证所有 6 个节点运行正常
2. **健康检查**: 验证 Docker healthcheck 状态
3. **Cluster 信息**: 验证 cluster_state、slots、节点数
4. **节点角色**: 验证 3 主 3 从分配
5. **Slot 分配**: 验证 16384 个 slots 完整分配
6. **读写测试**: 验证数据读写和跨节点访问
7. **网络连通性**: 验证所有节点可访问
8. **高可用测试**: 故障转移测试（可选）
9. **持久化测试**: 重启后数据不丢失

### 手动验收清单

```bash
# 1. 检查集群状态
docker exec redis-node-1 redis-cli -p 6379 cluster info
# 期望：cluster_state:ok, cluster_slots_assigned:16384

# 2. 检查节点角色
docker exec redis-node-1 redis-cli -p 6379 cluster nodes
# 期望：3 个 master, 3 个 slave

# 3. 测试读写
docker exec redis-node-1 redis-cli -p 6379 -c set test_key test_value
docker exec redis-node-1 redis-cli -p 6379 -c get test_key
# 期望：写入 OK, 读取 test_value

# 4. 测试故障转移
docker stop redis-node-1
sleep 15
docker exec redis-node-2 redis-cli -p 6379 cluster info
# 期望：cluster_state:ok
docker start redis-node-1
```

## 故障排查

### 节点无法启动

```bash
# 查看日志
docker compose -f deploy/docker/redis-cluster/docker-compose.yml logs redis-node-1

# 检查端口占用
lsof -i :10000
```

### 集群状态异常

```bash
# 重置集群（删除所有数据）
docker compose -f deploy/docker/redis-cluster/docker-compose.yml down -v
docker compose -f deploy/docker/redis-cluster/docker-compose.yml up -d
```

### 节点无法通信

```bash
# 检查网络
docker network inspect redis-cluster-network

# 测试节点间连通性
docker exec redis-node-1 redis-cli -h redis-node-2 -p 6379 ping
```

### Slot 分配不均

```bash
# 重新平衡 slots（需要 redis-cli 支持）
redis-cli --cluster rebalance redis-node-1:6379
```

## 性能测试

```bash
# 使用 redis-benchmark 进行压力测试
docker exec redis-node-1 redis-benchmark -h redis-node-1 -p 6379 -c 50 -n 10000

# 测试 SET 性能
docker exec redis-node-1 redis-benchmark -h redis-node-1 -p 6379 -t set -n 10000

# 测试 GET 性能
docker exec redis-node-1 redis-benchmark -h redis-node-1 -p 6379 -t get -n 10000
```

## 生产环境建议

当前配置适用于**本地开发和测试**。生产环境请考虑：

1. **安全加固**
   - 启用密码认证 (`requirepass`)
   - 禁用危险命令 (`rename-command`)
   - 开启 protected-mode

2. **资源限制**
   - 根据负载调整内存限制
   - 配置 CPU 限制
   - 调整 maxclients

3. **持久化优化**
   - 根据业务需求调整 AOF 策略
   - 配置 RDB 备份策略
   - 启用磁盘同步

4. **监控告警**
   - 部署 Redis Exporter
   - 配置 Prometheus 监控
   - 设置关键指标告警

5. **高可用增强**
   - 增加 Slave 数量
   - 调整 cluster-node-timeout
   - 配置合理的故障检测时间

## 相关文档

- [Redis Cluster 官方文档](https://redis.io/docs/management/scaling/)
- [Redis 配置参考](https://redis.io/docs/configuration/)
- [项目 Docker Compose 部署](../README.md)
