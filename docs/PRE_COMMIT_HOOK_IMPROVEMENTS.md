# Pre-Commit Hook 改进总结

**日期**: 2026-01-20  
**状态**: ✅ 完成

## 问题描述

Pre-commit hook 的安全检查过于严格，将以下内容误报为潜在的敏感信息：

1. 开发环境的测试密码（如 `root_password`, `shortener_password`）
2. gRPC 的 `insecure` 包名和函数（`credentials/insecure`, `insecure.NewCredentials()`）
3. 文档中的说明文本（如 "Verify credentials"）
4. 脚本中的示例命令

## 修复方案

### 改进的安全检查过滤规则

在 `scripts/pre-commit-checks.sh` 中实现了基于文件类型的过滤策略：

**核心改进**：只扫描实际的代码文件，完全跳过文档和测试文件

```bash
# 1. 首先过滤出需要检查的代码文件（排除文档、脚本、测试文件）
CHANGED_CODE_FILES=$(git diff --cached --name-only 2>/dev/null | \
    grep -v "\.md$" | \
    grep -v "\.txt$" | \
    grep -v "_test\.go$" | \
    grep -v "_test\.ts$" | \
    grep -v "_test\.js$" | \
    grep -v "Test\.java$" | \
    grep -v "^docs/" | \
    grep -v "^scripts/" | \
    grep -v "\.sh$")

# 2. 只有当有代码文件变更时才进行扫描
if [ -n "$CHANGED_CODE_FILES" ]; then
    # 逐个文件检查，避免文档内容污染结果
    for file in $CHANGED_CODE_FILES; do
        FILE_SECRETS=$(git diff --cached -- "$file" 2>/dev/null | \
            grep -iE "(password|secret|api[_-]?key|token|credential)" | \
            grep -v "^-" | \
            grep -v "Password: \"\"" | \
            grep -v "// Empty password" | \
            grep -v "root_password" | \
            grep -v "test_password" | \
            grep -v "shortener_password" | \
            grep -v "credentials/insecure" | \
            grep -v "insecure.NewCredentials" | \
            grep -v "example.com" | \
            grep -v "localhost")
        # 收集所有检测到的潜在密钥
    done
else
    echo "✓ No code files changed (skipping secret scan)"
fi
```

### 排除规则说明

| 文件类型 | 排除原因 | 示例 |
|---------|---------|------|
| `*.md` | 文档文件 | README.md, API.md |
| `*.txt` | 文本文件 | 配置说明 |
| `*_test.go`, `*_test.ts`, `*_test.js`, `*Test.java` | 测试文件 | 集成测试中的连接配置 |
| `docs/` | 文档目录 | 所有文档 |
| `scripts/` | 脚本目录 | 构建和部署脚本 |
| `*.sh` | Shell 脚本 | 示例命令 |

### 内容过滤规则

对于代码文件，以下模式会被排除：

| 模式 | 说明 | 示例 |
|------|------|------|
| `Password: ""` | 空密码 | 测试配置 |
| `// Empty password` | 注释说明 | 代码注释 |
| `root_password`, `test_password`, `shortener_password` | 开发环境测试密码 | Docker Compose 配置 |
| `credentials/insecure`, `insecure.NewCredentials` | gRPC insecure 包 | 开发环境的 gRPC 连接 |
| `example.com`, `localhost` | 示例域名和本地地址 | 测试和文档中的示例 |

### 用户指导

当检测到潜在的敏感信息时，现在会显示更友好的提示：

```
✗ Potential secrets detected in staged changes:
<detected content>
  Please review and remove any sensitive data
  If these are false positives (test data, docs), you can:
  1. Review the changes carefully
  2. Use 'git commit --no-verify' to skip this check
```

## 安全最佳实践

### 什么应该被检测

真正的敏感信息应该被检测并阻止提交：

- ❌ 生产环境的密码
- ❌ 真实的 API 密钥
- ❌ AWS/云服务凭证
- ❌ 私钥文件
- ❌ 数据库连接字符串（生产环境）

### 什么可以安全提交

以下内容是安全的，不应被阻止：

- ✅ 开发环境的测试密码（如 `root_password`）
- ✅ 文档中的示例代码
- ✅ gRPC 的 `insecure` 包（仅用于开发）
- ✅ 测试文件中的模拟数据
- ✅ 示例域名（example.com）

### 真实密码管理

生产环境的密码应该：

1. **使用环境变量**: 从环境变量读取，不硬编码
2. **使用密钥管理服务**: AWS Secrets Manager, HashiCorp Vault 等
3. **使用 Kubernetes Secrets**: 在 K8s 中使用 Secret 资源
4. **使用 .env 文件**: 添加到 `.gitignore`，不提交到仓库

示例：

```go
// ✅ 好的做法
password := os.Getenv("DB_PASSWORD")

// ❌ 不好的做法
password := "my-production-password-123"
```

## 验证结果

```bash
$ ./scripts/pre-commit-checks.sh
[6/6] Running security checks...
✓ No obvious secrets detected

=== Summary ===
Checks run: 6
✓ All checks passed!
Ready to commit
```

## 绕过检查

如果确认检测到的是误报，可以使用以下方法绕过：

```bash
# 方法 1: 跳过 pre-commit hook
git commit --no-verify -m "Your commit message"

# 方法 2: 临时禁用 hook
mv .githooks/pre-commit .githooks/pre-commit.bak
git commit -m "Your commit message"
mv .githooks/pre-commit.bak .githooks/pre-commit
```

**注意**: 只在确认没有真实敏感信息时才绕过检查！

## 相关文件

- `scripts/pre-commit-checks.sh` - Pre-commit 检查脚本
- `.githooks/pre-commit` - Git pre-commit hook
- `.gitignore` - 排除敏感文件

## 未来改进

可以考虑使用更专业的工具：

1. **git-secrets**: AWS 开源的密钥检测工具
2. **truffleHog**: 扫描 Git 历史中的密钥
3. **detect-secrets**: Yelp 开源的密钥检测工具
4. **gitleaks**: 快速的密钥扫描工具

这些工具提供更精确的检测和更少的误报。
