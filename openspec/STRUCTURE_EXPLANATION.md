# OpenSpec Directory Structure Explanation

## Question: Why does `openspec list --specs` return "No specs found" when there are .md files in openspec/specs/?

## Answer

OpenSpec 工具期望特定的目录结构，而不是直接在 `openspec/specs/` 下放置 `.md` 文件。

### 错误的结构 ❌

```
openspec/specs/
├── hello-todo-services.md
├── url-shortener-service.md
├── app-management-system.md
└── monorepo-architecture.md
```

在这种结构下，`openspec list --specs` 会返回 "No specs found"。

### 正确的结构 ✅

```
openspec/specs/
├── hello-todo-services/
│   └── spec.md              # 必需
├── url-shortener-service/
│   └── spec.md              # 必需
├── app-management-system/
│   └── spec.md              # 必需
└── monorepo-architecture/
    ├── spec.md              # 必需
    └── design.md            # 可选
```

### 为什么需要这种结构？

1. **Capability 隔离**: 每个 capability 有自己的目录，可以包含多个相关文件
2. **扩展性**: 可以在同一个 capability 目录下添加 `design.md`、`tasks.md` 等文件
3. **工具识别**: OpenSpec CLI 工具通过扫描子目录和查找 `spec.md` 文件来识别规范
4. **变更管理**: 在 `openspec/changes/` 中创建变更提案时，可以引用特定的 capability

### 验证结构

重新组织后，可以使用以下命令验证：

```bash
# 列出所有规范
openspec list --specs

# 查看特定规范
openspec show hello-todo-services --type spec
openspec show url-shortener-service --type spec

# 验证规范格式
openspec validate --specs
```

### 输出示例

```
$ openspec list --specs
Specs:
  app-management-system     requirements 0
  hello-todo-services       requirements 0
  integration-testing       requirements 0
  monorepo-architecture     requirements 0
  quality-practices         requirements 0
  url-shortener-service     requirements 0
```

## 已完成的重组

OpenSpec 规范（符合 OpenSpec 格式，包含 Purpose 和 Requirements）：

- ✅ `openspec/specs/hello-todo-services/spec.md` - 10 requirements
- ✅ `openspec/specs/url-shortener-service/spec.md` - 16 requirements

架构文档（已移至 docs/ 目录）：

- 📄 `docs/openspec-app-management-system.md`
- 📄 `docs/openspec-monorepo-architecture.md`
- 📄 `docs/openspec-integration-testing.md`
- 📄 `docs/openspec-quality-practices.md`

## 验证结果

```bash
$ openspec validate --specs
✓ spec/hello-todo-services
✓ spec/url-shortener-service
Totals: 2 passed, 0 failed (2 items)
```

## 参考

- `openspec/AGENTS.md` - OpenSpec 完整使用指南
- `openspec/SYNC_SUMMARY.md` - 规范同步总结
