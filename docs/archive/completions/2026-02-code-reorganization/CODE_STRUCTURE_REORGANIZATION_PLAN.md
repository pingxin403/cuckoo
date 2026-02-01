# Code Structure Reorganization Plan - Monorepo Edition

## 背景分析

这是一个**多语言 Monorepo 项目**（Go、Java、TypeScript），已有清晰的目录结构：
- `apps/` - 应用服务
- `libs/` - 共享库（hlc, config, observability）
- `api/` - Protobuf API 契约
- `deploy/` - 部署配置
- `tests/` - E2E 测试
- `tools/` - 开发工具

## 当前问题

根目录下有 8 个多地域相关的包，破坏了 monorepo 的清晰结构：

```
项目根目录/
├── arbiter/          # 仲裁服务（分布式协调）
├── failover/         # 故障转移管理
├── health/           # 健康检查库
├── monitoring/       # 监控面板
├── queue/            # 本地队列（MVP 替代 Kafka）
├── routing/          # 地理路由
├── storage/          # 本地存储（MVP 替代 MySQL）
├── sync/             # 跨地域同步
└── tools/arbiter-mock/  # Mock 仲裁服务
```

### 核心问题

1. **根目录混乱**: 8 个包在根目录，不符合 monorepo 规范
2. **重复实现**: `routing/` 和 `apps/im-gateway-service/routing/` 都存在
3. **重复实现**: `sync/` 和 `apps/im-service/sync/` 都存在
4. **职责不清**: 这些包既不是 apps，也不是 libs，也不是 tools
5. **难以发现**: 新开发者不知道这些包的用途和关系

## 深入分析：重复实现的真相

### routing 包分析

**根目录 `routing/`**:
- 完整的独立实现，包含 HTTP 服务器
- 有完整的 README、示例和集成测试
- 可以独立运行：`go run routing/example_integration.go`
- 用途：**演示和测试**

**`apps/im-gateway-service/routing/`**:
- 集成到 im-gateway-service 的版本
- 代码结构相同，但作为服务的一部分
- 用途：**生产使用**

**结论**: 根目录版本是**独立演示/测试版本**，app 版本是**生产集成版本**

### sync 包分析

**根目录 `sync/`**:
- 完整的独立实现，包含 ConflictResolver 和 MessageSyncer
- 有完整的 README、属性测试和集成测试
- 可以独立测试和演示
- 用途：**演示和测试**

**`apps/im-service/sync/`**:
- 集成到 im-service 的版本
- 只包含 ConflictResolver（MessageSyncer 在 im-service 主代码中）
- 用途：**生产使用**

**结论**: 根目录版本是**独立演示/测试版本**，app 版本是**生产集成版本**

## 组件分类（基于实际用途）

### 类型 1: 演示/测试组件 (Demo & Testing Components)
**特征**: 独立的演示和测试实现，可独立运行

- ✅ **routing/** - 地理路由演示（有独立 HTTP 服务器和示例）
- ✅ **sync/** - 跨地域同步演示（有完整的属性测试）
- ✅ **arbiter/** - 仲裁服务演示（有集成测试）
- ✅ **failover/** - 故障转移演示（有集成测试）
- ✅ **health/** - 健康检查演示（有集成测试）

**用途**: 
- 演示多地域功能如何工作
- 独立测试和验证
- 文档和示例代码
- 不依赖完整的服务部署

**建议**: 移到 `examples/multi-region/` 或保留在根目录但添加清晰的文档说明

### 类型 2: MVP 简化组件 (MVP Simplified Components)
**特征**: 为 MVP 阶段创建的简化实现，替代生产级组件

- ✅ **queue/** - 本地队列（替代 Kafka，基于 Go channels）
- ✅ **storage/** - 本地存储（替代 MySQL，基于 SQLite）

**用途**:
- 快速原型开发
- 本地测试环境
- 不需要外部依赖
- **不应在生产环境使用**

**建议**: 移到 `internal/mvp/` 或 `examples/mvp/`

### 类型 3: 开发工具 (Development Tools)
**特征**: 辅助开发和调试的工具

- ✅ **monitoring/** - Web 监控面板（用于演示和调试）
- ✅ **tools/arbiter-mock/** - Mock 仲裁服务（用于测试）

**用途**:
- 开发时的可视化监控
- 测试时的 mock 服务
- 调试和故障排查

**建议**: 
- `monitoring/` → `tools/monitoring/`
- `tools/arbiter-mock/` → 保持不变

### 类型 4: E2E 测试 (End-to-End Tests)
**特征**: 端到端测试代码

- ✅ **tests/e2e/multi-region/** - 多地域 E2E 测试

**用途**: 验证多地域功能的端到端流程

**建议**: 保持在当前位置 ✅

## 推荐方案：保持简洁的 Monorepo 结构

### 方案 A: 创建 examples/multi-region/ 目录（推荐）⭐

这是最符合 monorepo 最佳实践的方案：

```
项目根目录/
├── apps/                           # 应用服务（已存在）
│   ├── im-service/
│   │   └── sync/                   # 生产版本的 sync（集成版）
│   └── im-gateway-service/
│       └── routing/                # 生产版本的 routing（集成版）
│
├── libs/                           # 共享库（已存在）
│   ├── hlc/                        # HLC 时钟
│   ├── config/                     # 配置库
│   └── observability/              # 可观测性
│
├── examples/                       # 示例和演示代码 ⬅️ 新增
│   ├── multi-region/               # 多地域示例 ⬅️ 新增
│   │   ├── README.md               # 多地域功能总览
│   │   ├── arbiter/                # 仲裁服务演示
│   │   ├── failover/               # 故障转移演示
│   │   ├── health/                 # 健康检查演示
│   │   ├── routing/                # 地理路由演示
│   │   ├── sync/                   # 跨地域同步演示
│   │   └── monitoring/             # 监控面板演示
│   └── mvp/                        # MVP 简化组件 ⬅️ 新增
│       ├── README.md               # MVP 组件说明
│       ├── queue/                  # 本地队列（替代 Kafka）
│       └── storage/                # 本地存储（替代 MySQL）
│
├── tools/                          # 工具（已存在）
│   └── arbiter-mock/               # Mock 仲裁（已存在）
│
└── tests/                          # 测试（已存在）
    └── e2e/
        └── multi-region/           # E2E 测试（已存在）
```

**优势**:
- ✅ 符合 monorepo 最佳实践（examples/ 是常见目录）
- ✅ 清晰区分演示代码和生产代码
- ✅ 根目录保持简洁
- ✅ 易于发现和理解
- ✅ 不影响现有的 apps/ 和 libs/ 结构
- ✅ MVP 组件单独分组，明确标识为非生产代码

**劣势**:
- 需要更新 import 路径
- 需要更新文档引用

### 方案 B: 保留在根目录，添加清晰文档（保守）

```
项目根目录/
├── arbiter/                        # 保持不变，添加 "DEMO" 标识
├── failover/                       # 保持不变，添加 "DEMO" 标识
├── health/                         # 保持不变，添加 "DEMO" 标识
├── monitoring/                     # 保持不变，添加 "DEMO" 标识
├── queue/                          # 保持不变，添加 "MVP" 标识
├── routing/                        # 保持不变，添加 "DEMO" 标识
├── storage/                        # 保持不变，添加 "MVP" 标识
├── sync/                           # 保持不变，添加 "DEMO" 标识
├── MULTI_REGION_COMPONENTS.md      # 新增：说明这些组件的用途
└── apps/                           # 已存在
```

**优势**:
- ✅ 最小改动
- ✅ 不需要更新 import 路径
- ✅ 快速实施

**劣势**:
- ❌ 根目录仍然混乱
- ❌ 不符合 monorepo 最佳实践
- ❌ 新开发者仍然困惑

### 方案 C: 混合方案（折中）

```
项目根目录/
├── examples/                       # 示例和演示代码 ⬅️ 新增
│   └── multi-region/               # 多地域示例 ⬅️ 新增
│       ├── arbiter/                # 仲裁服务演示
│       ├── failover/               # 故障转移演示
│       ├── health/                 # 健康检查演示
│       ├── routing/                # 地理路由演示
│       ├── sync/                   # 跨地域同步演示
│       ├── monitoring/             # 监控面板演示
│       ├── queue/                  # 本地队列（MVP）
│       └── storage/                # 本地存储（MVP）
│
├── apps/                           # 应用服务（已存在）
├── libs/                           # 共享库（已存在）
├── tools/                          # 工具（已存在）
└── tests/                          # 测试（已存在）
```

**优势**:
- ✅ 根目录完全清理
- ✅ 所有演示代码集中管理
- ✅ 符合 monorepo 最佳实践

**劣势**:
- 需要更新所有 import 路径
- 需要更新所有文档引用
- 实施工作量较大

## 推荐实施方案：方案 A（examples/multi-region/）

### 为什么选择方案 A？

1. **符合行业最佳实践**: 
   - Kubernetes 有 `examples/` 目录
   - Istio 有 `samples/` 目录
   - 许多大型 monorepo 都有 `examples/` 或 `samples/` 目录

2. **清晰的职责划分**:
   - `apps/` = 生产应用
   - `libs/` = 共享库
   - `examples/` = 演示和示例
   - `tools/` = 开发工具
   - `tests/` = 测试

3. **易于理解**:
   - 新开发者一眼就能看出这些是示例代码
   - 明确区分演示代码和生产代码
   - MVP 组件单独分组，避免误用

4. **保持根目录简洁**:
   - 只有 5 个顶级目录：apps, libs, examples, tools, tests
   - 符合 monorepo 的简洁原则

### 实施步骤

#### 阶段 1: 创建新目录结构（立即执行）

```bash
# 1. 创建 examples 目录结构
mkdir -p examples/multi-region
mkdir -p examples/mvp

# 2. 移动多地域演示组件
mv arbiter examples/multi-region/
mv failover examples/multi-region/
mv health examples/multi-region/
mv routing examples/multi-region/
mv sync examples/multi-region/
mv monitoring examples/multi-region/

# 3. 移动 MVP 组件
mv queue examples/mvp/
mv storage examples/mvp/

# 4. 创建 README 文件
cat > examples/multi-region/README.md << 'EOF'
# Multi-Region Active-Active Examples

This directory contains demonstration and testing implementations of multi-region active-active components.

## ⚠️ Important Note

These are **example implementations** for demonstration, testing, and learning purposes. 

**Production implementations** are integrated into the services:
- `apps/im-service/sync/` - Production sync implementation
- `apps/im-gateway-service/routing/` - Production routing implementation

## Components

- **arbiter/** - Distributed coordination and split-brain prevention
- **failover/** - Automatic failover management
- **health/** - Multi-dimensional health checking
- **routing/** - Geographic routing with health-aware failover
- **sync/** - Cross-region message synchronization
- **monitoring/** - Web-based monitoring dashboard

## Usage

Each component can be run independently for testing and demonstration:

\`\`\`bash
# Run routing example
go run examples/multi-region/routing/example_integration.go

# Run monitoring dashboard
go run examples/multi-region/monitoring/example_dashboard.go
\`\`\`

## Testing

Run tests for all components:

\`\`\`bash
# Test all multi-region components
go test ./examples/multi-region/...

# Test specific component
go test ./examples/multi-region/sync/...
\`\`\`

## Documentation

See the README in each component directory for detailed documentation.
EOF

cat > examples/mvp/README.md << 'EOF'
# MVP Simplified Components

This directory contains simplified implementations of infrastructure components for MVP and testing purposes.

## ⚠️ Warning

**These components are NOT suitable for production use.** They are simplified implementations designed for:
- Local development
- Testing
- Prototyping
- Learning

For production deployments, use proper infrastructure:
- Use **Kafka** instead of `queue/`
- Use **MySQL/PostgreSQL** instead of `storage/`

## Components

- **queue/** - Go channel-based message queue (replaces Kafka for MVP)
- **storage/** - SQLite-based local storage (replaces MySQL for MVP)

## Usage

These components are used by the multi-region examples and tests:

\`\`\`go
import "github.com/cuckoo-org/cuckoo/examples/mvp/queue"
import "github.com/cuckoo-org/cuckoo/examples/mvp/storage"
\`\`\`

## Migration to Production

When moving to production:

1. Replace `queue` with Kafka client
2. Replace `storage` with MySQL/PostgreSQL client
3. Update configuration and connection strings
4. Test thoroughly in staging environment
EOF
```

#### 阶段 2: 更新 Import 路径（短期执行）

需要更新以下文件中的 import 路径：

**多地域组件的 import 路径更新**:
```go
// 旧路径 → 新路径

// 仲裁服务
"github.com/cuckoo-org/cuckoo/arbiter"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/arbiter"

// 故障转移
"github.com/cuckoo-org/cuckoo/failover"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/failover"

// 健康检查
"github.com/cuckoo-org/cuckoo/health"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/health"

// 地理路由
"github.com/cuckoo-org/cuckoo/routing"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/routing"

// 跨地域同步
"github.com/cuckoo-org/cuckoo/sync"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/sync"

// 监控
"github.com/cuckoo-org/cuckoo/monitoring"
→ "github.com/cuckoo-org/cuckoo/examples/multi-region/monitoring"
```

**MVP 组件的 import 路径更新**:
```go
// 本地队列
"github.com/cuckoo-org/cuckoo/queue"
→ "github.com/cuckoo-org/cuckoo/examples/mvp/queue"

// 本地存储
"github.com/cuckoo-org/cuckoo/storage"
→ "github.com/cuckoo-org/cuckoo/examples/mvp/storage"
```

**需要更新的文件**:
1. 所有 `examples/multi-region/` 下的 Go 文件
2. 所有 `tests/e2e/multi-region/` 下的测试文件
3. 所有 `go.mod` 文件中的 replace 指令（如果有）
4. 所有文档中的代码示例

#### 阶段 3: 更新文档引用（短期执行）

需要更新以下文档：

1. **根目录 README.md**:
   - 添加 `examples/` 目录说明
   - 更新多地域功能的链接

2. **多地域 spec 文档**:
   - `.kiro/specs/multi-region-active-active/README.md`
   - `.kiro/specs/multi-region-active-active/design.md`
   - 更新组件路径引用

3. **部署文档**:
   - `deploy/docker/README.md`
   - `deploy/mvp/README.md`
   - 更新示例代码路径

4. **架构文档**:
   - `docs/architecture/ARCHITECTURE.md`
   - 更新组件位置说明

#### 阶段 4: 验证和测试（短期执行）

```bash
# 1. 验证 Go 模块
go mod tidy
go mod verify

# 2. 运行所有测试
go test ./...

# 3. 运行多地域测试
go test ./examples/multi-region/...
go test ./tests/e2e/multi-region/...

# 4. 验证示例可以运行
go run examples/multi-region/routing/example_integration.go
go run examples/multi-region/monitoring/example_dashboard.go

# 5. 构建所有应用
make build

# 6. 运行 lint 检查
make lint
```

### 迁移脚本

创建自动化迁移脚本 `scripts/reorganize-code-structure.sh`:

```bash
#!/bin/bash
# scripts/reorganize-code-structure.sh
# 
# 重组代码结构：将多地域组件移到 examples/ 目录

set -e

echo "=== Code Structure Reorganization ==="
echo ""
echo "This script will:"
echo "1. Create examples/multi-region/ and examples/mvp/ directories"
echo "2. Move multi-region components to examples/multi-region/"
echo "3. Move MVP components to examples/mvp/"
echo "4. Create README files"
echo "5. Update import paths in all Go files"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# 1. Create directory structure
echo "Creating directory structure..."
mkdir -p examples/multi-region
mkdir -p examples/mvp

# 2. Move multi-region components
echo "Moving multi-region components..."
mv arbiter examples/multi-region/
mv failover examples/multi-region/
mv health examples/multi-region/
mv routing examples/multi-region/
mv sync examples/multi-region/
mv monitoring examples/multi-region/

# 3. Move MVP components
echo "Moving MVP components..."
mv queue examples/mvp/
mv storage examples/mvp/

# 4. Create README files
echo "Creating README files..."

cat > examples/multi-region/README.md << 'EOF'
# Multi-Region Active-Active Examples

This directory contains demonstration and testing implementations of multi-region active-active components.

## ⚠️ Important Note

These are **example implementations** for demonstration, testing, and learning purposes. 

**Production implementations** are integrated into the services:
- \`apps/im-service/sync/\` - Production sync implementation
- \`apps/im-gateway-service/routing/\` - Production routing implementation

## Components

- **arbiter/** - Distributed coordination and split-brain prevention
- **failover/** - Automatic failover management
- **health/** - Multi-dimensional health checking
- **routing/** - Geographic routing with health-aware failover
- **sync/** - Cross-region message synchronization
- **monitoring/** - Web-based monitoring dashboard

## Usage

Each component can be run independently for testing and demonstration:

\`\`\`bash
# Run routing example
go run examples/multi-region/routing/example_integration.go

# Run monitoring dashboard
go run examples/multi-region/monitoring/example_dashboard.go
\`\`\`

## Testing

Run tests for all components:

\`\`\`bash
# Test all multi-region components
go test ./examples/multi-region/...

# Test specific component
go test ./examples/multi-region/sync/...
\`\`\`

## Documentation

See the README in each component directory for detailed documentation.
EOF

cat > examples/mvp/README.md << 'EOF'
# MVP Simplified Components

This directory contains simplified implementations of infrastructure components for MVP and testing purposes.

## ⚠️ Warning

**These components are NOT suitable for production use.** They are simplified implementations designed for:
- Local development
- Testing
- Prototyping
- Learning

For production deployments, use proper infrastructure:
- Use **Kafka** instead of \`queue/\`
- Use **MySQL/PostgreSQL** instead of \`storage/\`

## Components

- **queue/** - Go channel-based message queue (replaces Kafka for MVP)
- **storage/** - SQLite-based local storage (replaces MySQL for MVP)

## Usage

These components are used by the multi-region examples and tests:

\`\`\`go
import "github.com/cuckoo-org/cuckoo/examples/mvp/queue"
import "github.com/cuckoo-org/cuckoo/examples/mvp/storage"
\`\`\`

## Migration to Production

When moving to production:

1. Replace \`queue\` with Kafka client
2. Replace \`storage\` with MySQL/PostgreSQL client
3. Update configuration and connection strings
4. Test thoroughly in staging environment
EOF

cat > examples/README.md << 'EOF'
# Examples and Demonstrations

This directory contains example implementations and demonstrations for various features.

## Directories

- **multi-region/** - Multi-region active-active architecture examples
- **mvp/** - Simplified MVP components for local development and testing

## Purpose

These examples serve multiple purposes:

1. **Learning**: Understand how features work through standalone examples
2. **Testing**: Test components independently without full service deployment
3. **Prototyping**: Quickly prototype new features
4. **Documentation**: Provide working code examples for documentation

## Important Notes

- Examples are **not production-ready** code
- Production implementations are in \`apps/\` directory
- MVP components should not be used in production

## Usage

Each example directory has its own README with specific usage instructions.
EOF

# 5. Update import paths
echo "Updating import paths..."

# Function to update imports in a file
update_imports() {
    local file=$1
    
    # Multi-region components
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/arbiter"|"github.com/cuckoo-org/cuckoo/examples/multi-region/arbiter"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/failover"|"github.com/cuckoo-org/cuckoo/examples/multi-region/failover"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/health"|"github.com/cuckoo-org/cuckoo/examples/multi-region/health"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/routing"|"github.com/cuckoo-org/cuckoo/examples/multi-region/routing"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/sync"|"github.com/cuckoo-org/cuckoo/examples/multi-region/sync"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/monitoring"|"github.com/cuckoo-org/cuckoo/examples/multi-region/monitoring"|g' "$file"
    
    # MVP components
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/queue"|"github.com/cuckoo-org/cuckoo/examples/mvp/queue"|g' "$file"
    sed -i.bak 's|"github.com/cuckoo-org/cuckoo/storage"|"github.com/cuckoo-org/cuckoo/examples/mvp/storage"|g' "$file"
    
    # Remove backup file
    rm -f "$file.bak"
}

# Update all Go files
echo "Updating Go files..."
find examples/multi-region -name "*.go" -type f | while read file; do
    update_imports "$file"
done

find examples/mvp -name "*.go" -type f | while read file; do
    update_imports "$file"
done

find tests/e2e/multi-region -name "*.go" -type f | while read file; do
    update_imports "$file"
done

# 6. Update go.mod
echo "Updating go.mod..."
go mod tidy

echo ""
echo "✓ Code structure reorganization complete!"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Run tests: go test ./..."
echo "3. Verify examples work: go run examples/multi-region/routing/example_integration.go"
echo "4. Update documentation references"
echo "5. Commit changes: git add . && git commit -m 'Reorganize code structure'"
```

使脚本可执行：
```bash
chmod +x scripts/reorganize-code-structure.sh
```

## 详细分析（保留用于参考）

**性质**: 多地域核心组件，提供分布式协调和脑裂防护

**当前位置**: 根目录
**推荐位置**: `libs/multi-region/arbiter/` 或 `libs/arbiter/`

**理由**:
- 是可复用的库，不是独立服务
- 专门为多地域设计
- 有清晰的 API 和文档

**依赖关系**:
- 依赖: Zookeeper, health checker
- 被依赖: im-service, im-gateway

### 2. failover/ - 故障转移管理

**性质**: 多地域核心组件，管理故障转移逻辑

**当前位置**: 根目录
**推荐位置**: `libs/multi-region/failover/` 或 `libs/failover/`

**理由**:
- 是可复用的库
- 与 arbiter 紧密配合
- 可被多个服务使用

**依赖关系**:
- 依赖: arbiter, health checker
- 被依赖: im-service, im-gateway

### 3. health/ - 健康检查

**性质**: 通用共享库，可用于任何服务

**当前位置**: 根目录
**推荐位置**: `libs/health/`

**理由**:
- 通用功能，不限于多地域
- 与 `libs/hlc/`, `libs/config/` 并列
- 可被所有服务使用

**依赖关系**:
- 依赖: storage, queue（可选）
- 被依赖: arbiter, failover, 所有服务

### 4. monitoring/ - 监控面板

**性质**: 开发工具，Web 监控界面

**当前位置**: 根目录
**推荐位置**: `tools/monitoring/`

**理由**:
- 是工具而非库
- 主要用于开发和演示
- 与 `tools/arbiter-mock/` 并列

**依赖关系**:
- 依赖: hlc, sync, conflict resolver
- 被依赖: 无（独立运行）

### 5. queue/ - 本地队列

**性质**: MVP 简化实现，替代 Kafka

**当前位置**: 根目录
**推荐位置**: `internal/mvp/queue/` 或 `libs/queue/`

**理由**:
- 仅用于 MVP 和测试
- 不应在生产环境使用
- 放在 `internal/` 表明内部使用

**依赖关系**:
- 依赖: 无
- 被依赖: sync, storage, 测试代码

### 6. routing/ - 地理路由

**性质**: 多地域核心组件，智能路由决策

**当前位置**: 根目录
**推荐位置**: `libs/multi-region/routing/` 或 `apps/im-gateway-service/routing/`

**理由**:
- 专门为多地域设计
- 可作为库使用，也可集成到 gateway
- 已经在 `apps/im-gateway-service/routing/` 有集成版本

**特殊考虑**:
- 根目录的 `routing/` 是独立版本（可独立运行）
- `apps/im-gateway-service/routing/` 是集成版本
- 可能需要保留两个版本，或者合并

**依赖关系**:
- 依赖: health checker
- 被依赖: im-gateway

### 7. storage/ - 本地存储

**性质**: MVP 简化实现，替代 MySQL

**当前位置**: 根目录
**推荐位置**: `internal/mvp/storage/` 或 `libs/storage/`

**理由**:
- 仅用于 MVP 和测试
- 不应在生产环境使用
- 放在 `internal/` 表明内部使用

**依赖关系**:
- 依赖: 无
- 被依赖: sync, health, 测试代码

### 8. sync/ - 跨地域同步

**性质**: 多地域核心组件，消息同步逻辑

**当前位置**: 根目录
**推荐位置**: `libs/multi-region/sync/` 或 `apps/im-service/sync/`

**理由**:
- 专门为多地域设计
- 可作为库使用，也可集成到 im-service
- 已经在 `apps/im-service/sync/` 有集成版本

**特殊考虑**:
- 根目录的 `sync/` 是独立版本（可独立测试）
- `apps/im-service/sync/` 是集成版本
- 可能需要保留两个版本，或者合并

**依赖关系**:
- 依赖: hlc, queue, storage
- 被依赖: im-service

### 9. tools/arbiter-mock/ - Mock 仲裁服务

**性质**: 测试工具

**当前位置**: `tools/arbiter-mock/`
**推荐位置**: 保持不变 ✅

**理由**:
- 已经在正确的位置
- 与其他工具并列

### 10. tests/e2e/multi-region/ - E2E 测试

**性质**: 端到端测试

**当前位置**: `tests/e2e/multi-region/`
**推荐位置**: 保持不变 ✅

**理由**:
- 已经在正确的位置
- 符合测试目录结构

## 推荐实施方案

### 阶段 1: 立即执行（高优先级）

**目标**: 清理根目录，提高可维护性

1. **移动 MVP 组件到 internal/**
   ```bash
   mkdir -p internal/mvp
   mv queue internal/mvp/
   mv storage internal/mvp/
   ```

2. **移动监控工具到 tools/**
   ```bash
   mv monitoring tools/
   ```

3. **移动健康检查到 libs/**
   ```bash
   mv health libs/
   ```

**影响**: 
- 需要更新 import 路径
- 需要更新文档引用
- 相对低风险

### 阶段 2: 短期执行（中优先级）

**目标**: 整合多地域核心组件

4. **创建 libs/multi-region/ 目录**
   ```bash
   mkdir -p libs/multi-region
   mv arbiter libs/multi-region/
   mv failover libs/multi-region/
   ```

5. **处理 routing 和 sync 的重复**
   - 评估根目录版本和 apps/ 下版本的差异
   - 决定保留哪个版本或如何合并
   - 可能需要重构为库 + 集成两部分

**影响**:
- 需要仔细处理重复代码
- 需要更新所有依赖
- 中等风险

### 阶段 3: 长期优化（低优先级）

**目标**: 进一步优化结构

6. **统一 routing 实现**
   - 将通用逻辑提取到 `libs/multi-region/routing/`
   - 在 `apps/im-gateway-service/` 中只保留集成代码

7. **统一 sync 实现**
   - 将通用逻辑提取到 `libs/multi-region/sync/`
   - 在 `apps/im-service/` 中只保留集成代码

**影响**:
- 需要大量重构
- 需要全面测试
- 较高风险

## 迁移脚本

### 阶段 1 迁移脚本

```bash
#!/bin/bash
# migrate-phase1.sh

set -e

echo "=== Phase 1: Code Structure Reorganization ==="

# 1. Create directories
echo "Creating directories..."
mkdir -p internal/mvp
mkdir -p libs/health

# 2. Move MVP components
echo "Moving MVP components..."
mv queue internal/mvp/
mv storage internal/mvp/

# 3. Move monitoring tool
echo "Moving monitoring tool..."
mv monitoring tools/

# 4. Move health library
echo "Moving health library..."
mv health libs/

echo "✓ Phase 1 complete!"
echo ""
echo "Next steps:"
echo "1. Update import paths in all files"
echo "2. Update documentation references"
echo "3. Run tests to verify"
echo "4. Commit changes"
```

### Import 路径更新

需要更新的 import 路径：

```go
// 旧路径 → 新路径

// MVP 组件
"github.com/cuckoo-org/cuckoo/queue"
→ "github.com/cuckoo-org/cuckoo/internal/mvp/queue"

"github.com/cuckoo-org/cuckoo/storage"
→ "github.com/cuckoo-org/cuckoo/internal/mvp/storage"

// 健康检查
"github.com/cuckoo-org/cuckoo/health"
→ "github.com/cuckoo-org/cuckoo/libs/health"

// 监控（如果有 import）
"github.com/cuckoo-org/cuckoo/monitoring"
→ "github.com/cuckoo-org/cuckoo/tools/monitoring"
```

## 验证清单

重组完成后，验证以下内容：

- [ ] 所有文件已移动到新位置
- [ ] Import 路径已更新
- [ ] 所有测试通过：`go test ./...`
- [ ] 示例可以运行：
  - [ ] `go run examples/multi-region/routing/example_integration.go`
  - [ ] `go run examples/multi-region/monitoring/example_dashboard.go`
- [ ] 文档已更新：
  - [ ] 根目录 README.md
  - [ ] `.kiro/specs/multi-region-active-active/README.md`
  - [ ] `deploy/docker/README.md`
- [ ] Go mod 文件正确：`go mod verify`
- [ ] CI/CD 配置已更新（如果需要）
- [ ] Docker 构建正常：`make docker-build`
- [ ] 所有应用可以构建：`make build`

## 风险评估

### 低风险 ✅
- 创建新目录结构
- 移动文件到新位置
- 创建 README 文件

### 中风险 ⚠️
- 更新 import 路径（可能遗漏某些文件）
- 更新文档引用（需要仔细检查）

### 高风险 ❌
- 无（这是纯粹的代码重组，不改变功能）

## 回滚计划

如果重组后出现问题，可以快速回滚：

```bash
# 1. 回滚 Git 更改
git reset --hard HEAD

# 2. 或者手动移回
mv examples/multi-region/* .
mv examples/mvp/* .
rmdir examples/multi-region examples/mvp examples

# 3. 恢复 import 路径
# 使用相反的 sed 命令
```

## 时间估算

- **阶段 1** (创建目录结构): 10 分钟
- **阶段 2** (更新 import 路径): 30 分钟
- **阶段 3** (更新文档): 30 分钟
- **阶段 4** (验证和测试): 30 分钟

**总计**: 约 1.5-2 小时

## 后续优化（可选）

完成基本重组后，可以考虑以下优化：

### 1. 统一 routing 实现
- 将 `examples/multi-region/routing/` 的通用逻辑提取到 `libs/routing/`
- 在 `apps/im-gateway-service/routing/` 中只保留集成代码

### 2. 统一 sync 实现
- 将 `examples/multi-region/sync/` 的通用逻辑提取到 `libs/sync/`
- 在 `apps/im-service/sync/` 中只保留集成代码

### 3. 创建共享的 health 库
- 将 `examples/multi-region/health/` 提升为 `libs/health/`
- 所有服务都可以使用

### 4. 文档改进
- 创建多地域架构的完整文档
- 添加从示例到生产的迁移指南
- 创建视频教程或演示

## 总结

### 推荐方案
**方案 A: 创建 examples/multi-region/ 目录** ⭐

### 核心原则
1. **清晰的职责划分**: 演示代码 vs 生产代码
2. **符合最佳实践**: 使用 examples/ 目录是行业标准
3. **保持简洁**: 根目录只有 5 个顶级目录
4. **易于理解**: 新开发者一眼就能看出组件用途

### 实施建议
1. **立即执行**: 阶段 1（创建目录结构）
2. **短期执行**: 阶段 2-4（更新路径、文档、验证）
3. **长期优化**: 考虑统一实现和提取共享库

### 预期效果
- ✅ 根目录清晰简洁
- ✅ 代码组织符合 monorepo 最佳实践
- ✅ 新开发者容易理解项目结构
- ✅ 演示代码和生产代码明确分离
- ✅ MVP 组件明确标识，避免误用

---

**创建日期**: 2024  
**状态**: 待审查和执行  
**优先级**: 高  
**预计工作量**: 1.5-2 小时
