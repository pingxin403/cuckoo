# 文档清理计划

## 当前状态分析

### 根目录文档（需要清理）

根目录当前有以下临时文档：

1. ✅ **CODE_REORGANIZATION_COMPLETE.md** - 代码重组完成总结
2. ✅ **CODE_REORGANIZATION_SUMMARY.md** - 代码重组分析总结
3. ✅ **CODE_STRUCTURE_REORGANIZATION_PLAN.md** - 代码结构重组计划
4. ✅ **MULTI_REGION_DOCS_QUICK_START.md** - 多地域文档快速开始
5. ✅ **DOCKERFILE_FIX_SUMMARY.md** - Dockerfile 修复总结
6. ✅ **SHORTENER_PERFORMANCE_REVIEW_SUMMARY.md** - 短链接性能审查总结

### 保留的根目录文档

这些文档应该保留在根目录：

1. ✅ **README.md** - 项目主文档
2. ✅ **TESTING.md** - 测试指南
3. ✅ **AGENTS.md** - AI 助手指令
4. ✅ **Makefile** - 构建脚本

## 清理方案

### 方案 A: 归档到 docs/archive/completions/（推荐）⭐

将所有完成的任务总结文档移到归档目录：

```
docs/archive/completions/
├── 2026-02-code-reorganization/
│   ├── CODE_REORGANIZATION_COMPLETE.md
│   ├── CODE_REORGANIZATION_SUMMARY.md
│   └── CODE_STRUCTURE_REORGANIZATION_PLAN.md
├── 2026-02-multi-region-docs/
│   └── MULTI_REGION_DOCS_QUICK_START.md
├── dockerfile-fixes/
│   └── DOCKERFILE_FIX_SUMMARY.md
└── performance-reviews/
    └── SHORTENER_PERFORMANCE_REVIEW_SUMMARY.md
```

**优势**：
- ✅ 保留历史记录
- ✅ 根目录保持简洁
- ✅ 易于查找历史文档
- ✅ 按时间和主题组织

### 方案 B: 直接删除（不推荐）

删除所有临时文档，因为信息已经整合到正式文档中。

**劣势**：
- ❌ 丢失详细的实施记录
- ❌ 无法追溯决策过程

## 推荐实施步骤

### 步骤 1: 创建归档目录结构

```bash
mkdir -p docs/archive/completions/2026-02-code-reorganization
mkdir -p docs/archive/completions/2026-02-multi-region-docs
mkdir -p docs/archive/completions/dockerfile-fixes
mkdir -p docs/archive/completions/performance-reviews
```

### 步骤 2: 移动文档

```bash
# 代码重组相关
mv CODE_REORGANIZATION_COMPLETE.md docs/archive/completions/2026-02-code-reorganization/
mv CODE_REORGANIZATION_SUMMARY.md docs/archive/completions/2026-02-code-reorganization/
mv CODE_STRUCTURE_REORGANIZATION_PLAN.md docs/archive/completions/2026-02-code-reorganization/

# 多地域文档重组
mv MULTI_REGION_DOCS_QUICK_START.md docs/archive/completions/2026-02-multi-region-docs/

# Dockerfile 修复
mv DOCKERFILE_FIX_SUMMARY.md docs/archive/completions/dockerfile-fixes/

# 性能审查
mv SHORTENER_PERFORMANCE_REVIEW_SUMMARY.md docs/archive/completions/performance-reviews/
```

### 步骤 3: 创建归档索引

创建 `docs/archive/completions/README.md` 索引文件。

### 步骤 4: 更新主 README

如果主 README 中有引用这些文档的链接，需要更新路径。

## 其他需要清理的区域

### 1. docs/multi-region-demo/ 目录

这个目录应该已经被重组了，检查是否还存在：

```bash
# 检查是否还存在
ls -la docs/multi-region-demo/

# 如果存在，应该移动到正确位置或删除
```

根据之前的计划，这些文档应该已经移到：
- `docs/architecture/MULTI_REGION_ACTIVE_ACTIVE.md`
- `docs/operations/multi-region/`
- `.kiro/specs/multi-region-active-active/blog/`

### 2. apps/ 目录下的临时文档

检查 apps/ 目录下是否有临时的总结文档：

```bash
find apps -maxdepth 2 -name "*SUMMARY*.md" -o -name "*COMPLETE*.md"
```

可能需要归档的文档：
- `apps/INTEGRATION_SUMMARY.md`
- `apps/MULTI_REGION_INTEGRATION_COMPLETE.md`
- `apps/MULTI_REGION_COMPONENTS.md`

### 3. deploy/ 目录下的临时文档

检查 deploy/ 目录下的文档：

```bash
find deploy -name "*SUMMARY*.md" -o -name "*COMPLETE*.md" -o -name "*PLAN*.md"
```

可能需要归档的文档：
- `deploy/docker/IMPLEMENTATION_SUMMARY.md`
- `deploy/docker/DEPLOYMENT_COMPLETE.md`
- `deploy/docker/DEPLOYMENT_EXECUTION_PLAN.md`

### 4. tests/ 目录下的临时文档

检查 tests/ 目录下的文档：

```bash
find tests -name "*SUMMARY*.md" -o -name "*COMPLETE*.md"
```

可能需要归档的文档：
- `tests/e2e/multi-region/IMPLEMENTATION_COMPLETE.md`
- `tests/e2e/multi-region/TASK_10.1_SUMMARY.md`
- `tests/e2e/multi-region/TASK_10.2_SUMMARY.md`

## 文档分类标准

### 应该保留在原位置的文档

1. **README.md** - 每个目录的主文档
2. **QUICKSTART.md** - 快速开始指南
3. **API.md** - API 文档
4. **DEPLOYMENT.md** - 部署文档
5. **TESTING.md** - 测试文档
6. **CHANGELOG.md** - 变更日志

### 应该归档的文档

1. **\*SUMMARY.md** - 任务总结
2. **\*COMPLETE.md** - 完成报告
3. **\*PLAN.md** - 实施计划（已完成的）
4. **\*EXECUTION\*.md** - 执行记录

### 应该删除的文档

1. **\*.bak** - 备份文件
2. **\*.tmp** - 临时文件
3. **重复的文档** - 内容已整合到其他文档

## 归档目录结构建议

```
docs/archive/
├── README.md                           # 归档索引
├── completions/                        # 已完成任务的总结
│   ├── 2026-02-code-reorganization/
│   ├── 2026-02-multi-region-docs/
│   ├── dockerfile-fixes/
│   └── performance-reviews/
├── app-specific/                       # 应用特定的历史文档
│   ├── im-service/
│   ├── auth-service/
│   └── shortener-service/
├── migrations/                         # 迁移记录
├── proposals/                          # 提案（已实施或拒绝）
└── fixes/                             # 修复记录
```

## 执行清单

- [ ] 创建归档目录结构
- [ ] 移动根目录临时文档
- [ ] 创建归档索引文件
- [ ] 检查并清理 docs/multi-region-demo/
- [ ] 检查并归档 apps/ 下的临时文档
- [ ] 检查并归档 deploy/ 下的临时文档
- [ ] 检查并归档 tests/ 下的临时文档
- [ ] 删除所有 .bak 文件
- [ ] 更新主 README 中的文档链接
- [ ] 验证所有重要信息已保留

## 预期效果

清理后的项目结构：

```
项目根目录/
├── README.md                          # 项目主文档
├── TESTING.md                         # 测试指南
├── AGENTS.md                          # AI 助手指令
├── Makefile                           # 构建脚本
├── go.mod / package.json              # 依赖管理
├── apps/                              # 应用服务（只保留 README）
├── libs/                              # 共享库（只保留 README）
├── examples/                          # 示例代码（只保留 README）
├── docs/                              # 文档目录
│   ├── README.md
│   ├── architecture/                  # 架构文档
│   ├── operations/                    # 运维文档
│   ├── deployment/                    # 部署文档
│   └── archive/                       # 归档文档
│       ├── README.md                  # 归档索引
│       └── completions/               # 完成的任务总结
└── tests/                             # 测试（只保留 README 和 QUICKSTART）
```

## 时间估算

- **步骤 1-3**（根目录清理）: 10 分钟
- **步骤 4**（其他目录清理）: 20 分钟
- **验证和更新链接**: 10 分钟

**总计**: 约 40 分钟

---

**创建日期**: 2026-02-01  
**状态**: 待执行  
**优先级**: 中  
**预计工作量**: 40 分钟
