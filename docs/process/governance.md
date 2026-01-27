# Repository Governance

本文档定义了 Monorepo 项目的治理规则，包括代码所有权、Pull Request 流程、健康度指标等。

## 目录

- [代码所有权](#代码所有权)
- [Pull Request 流程](#pull-request-流程)
- [代码审查指南](#代码审查指南)
- [仓库健康度指标](#仓库健康度指标)
- [依赖管理](#依赖管理)
- [安全策略](#安全策略)
- [发布流程](#发布流程)

## 代码所有权

### CODEOWNERS 配置

代码所有权通过 `.github/CODEOWNERS` 文件定义。每个目录或文件模式都有指定的所有者团队。

```
# API 契约层由平台团队负责
/api/ @platform-team

# 各服务由对应团队负责
/apps/hello-service/ @backend-java-team
/apps/todo-service/ @backend-go-team
/apps/web/ @frontend-team

# 基础设施和工具
/deploy/ @platform-team
/scripts/ @platform-team

# 服务模板
/templates/ @platform-team

# 文档
/docs/ @platform-team
README.md @platform-team
```

### 所有权原则

1. **明确责任**: 每个目录都有明确的所有者团队
2. **审批要求**: PR 必须获得至少一个所有者的审批
3. **跨团队协作**: 涉及多个团队的变更需要所有相关团队审批
4. **平台团队角色**: 负责基础设施、工具和跨服务的协调

### 团队职责

#### Platform Team (@platform-team)
- API 契约定义和版本管理
- 构建系统和工具维护
- 基础设施配置（K8s, Higress）
- 服务模板维护
- 跨服务协调和架构决策
- 文档维护

#### Backend Java Team (@backend-java-team)
- Hello Service 开发和维护
- Java 服务模板改进
- Java 相关工具和依赖管理

#### Backend Go Team (@backend-go-team)
- TODO Service 开发和维护
- Go 服务模板改进
- Go 相关工具和依赖管理

#### Frontend Team (@frontend-team)
- Web 应用开发和维护
- 前端构建配置
- UI/UX 改进

## Pull Request 流程

### PR 创建

1. **分支命名规范**:
   - 功能: `feature/description`
   - 修复: `fix/description`
   - 文档: `docs/description`
   - 重构: `refactor/description`

2. **PR 标题格式**:
   ```
   [类型] 简短描述

   类型: feat, fix, docs, refactor, test, chore
   示例: [feat] Add user authentication service
   ```

3. **PR 描述模板**:
   使用 `.github/pull_request_template.md` 模板，包含：
   - 变更描述
   - 变更类型
   - 测试说明
   - 相关 Issue
   - Checklist

### PR 审批要求

#### 基本要求
- ✅ 所有 CI 检查通过
- ✅ 至少一个 CODEOWNERS 成员审批
- ✅ 代码符合格式规范
- ✅ 包含必要的测试
- ✅ 文档已更新

#### 特殊要求

**API 变更**:
- 需要 @platform-team 审批
- 必须更新 API 文档
- 必须考虑向后兼容性
- 运行 `make gen-proto` 并提交生成的代码

**跨服务变更**:
- 需要所有相关团队审批
- 必须有集成测试
- 必须有回滚计划

**基础设施变更**:
- 需要 @platform-team 审批
- 必须在 staging 环境测试
- 必须有详细的变更说明

**依赖升级**:
- 重大版本升级需要团队讨论
- 必须检查 breaking changes
- 必须更新相关文档

### PR 审查时间

- **普通 PR**: 2 个工作日内
- **紧急修复**: 4 小时内
- **API 变更**: 3 个工作日内
- **基础设施变更**: 5 个工作日内

### PR 合并策略

- **Squash and Merge**: 默认策略，保持主分支历史清晰
- **Rebase and Merge**: 用于保留详细提交历史的场景
- **Merge Commit**: 用于合并长期功能分支

## 代码审查指南

### 审查重点

#### 功能正确性
- ✅ 代码实现是否符合需求
- ✅ 边界条件是否处理正确
- ✅ 错误处理是否完善

#### 代码质量
- ✅ 代码是否清晰易读
- ✅ 是否遵循项目编码规范
- ✅ 是否有适当的注释
- ✅ 是否有代码重复

#### 测试覆盖
- ✅ 是否有单元测试
- ✅ 是否有集成测试
- ✅ 测试是否覆盖主要场景
- ✅ 测试是否可维护

#### 性能和安全
- ✅ 是否有性能问题
- ✅ 是否有安全漏洞
- ✅ 是否有资源泄漏
- ✅ 是否有并发问题

#### 文档和可维护性
- ✅ API 文档是否更新
- ✅ README 是否更新
- ✅ 是否有必要的注释
- ✅ 是否易于理解和维护

### 审查反馈

**建设性反馈**:
- 明确指出问题
- 提供改进建议
- 解释原因
- 保持友好和专业

**示例**:
```
❌ 不好: "这段代码很糟糕"
✅ 好: "这个循环可以用 map 函数简化，提高可读性。例如：..."
```

### 审查响应

**作为 PR 作者**:
- 及时响应审查意见
- 解释设计决策
- 接受建设性批评
- 更新代码或说明原因

**作为审查者**:
- 及时审查 PR
- 提供清晰的反馈
- 区分必须修改和建议改进
- 批准后及时通知

## 仓库健康度指标

### 关键指标

#### 构建健康度
- **目标**: 主分支构建成功率 > 95%
- **监控**: CI/CD 流水线状态
- **行动**: 构建失败时立即修复

#### 测试覆盖率
- **目标**: 
  - 单元测试覆盖率 > 70%
  - 关键路径覆盖率 > 90%
- **监控**: 测试报告
- **行动**: 定期审查和改进测试

#### 代码质量
- **目标**:
  - 代码重复率 < 5%
  - 无严重 lint 错误
  - 无已知安全漏洞
- **监控**: 
  - Java: Checkstyle, SpotBugs
  - Go: golangci-lint
  - TypeScript: ESLint
- **行动**: 定期运行质量检查工具

#### PR 响应时间
- **目标**:
  - 首次响应 < 1 个工作日
  - 审批完成 < 2 个工作日
- **监控**: GitHub PR 统计
- **行动**: 定期提醒和跟进

#### 依赖更新
- **目标**:
  - 无高危安全漏洞
  - 依赖版本不超过 6 个月
- **监控**: Dependabot 报告
- **行动**: 每月审查和更新依赖

#### 仓库大小
- **目标**: 仓库大小 < 2GB
- **监控**: Git 仓库统计
- **行动**: 
  - 避免提交大文件
  - 使用 Git LFS 管理二进制文件
  - 定期清理不需要的文件

### 健康度报告

**每周报告**:
- CI 构建状态
- PR 审查状态
- 测试覆盖率变化

**每月报告**:
- 代码质量趋势
- 依赖更新状态
- 仓库大小变化
- 团队贡献统计

**季度报告**:
- 架构演进
- 技术债务评估
- 改进计划

### 监控工具

- **CI/CD**: GitHub Actions
- **代码质量**: SonarQube (可选)
- **依赖管理**: Dependabot
- **测试覆盖**: Codecov (可选)
- **仓库统计**: GitHub Insights

## 依赖管理

### 依赖更新策略

#### 自动更新
- **安全补丁**: 自动创建 PR（Dependabot）
- **小版本更新**: 每月批量更新
- **大版本更新**: 需要团队评估

#### 更新流程
1. Dependabot 创建 PR
2. 自动运行 CI 测试
3. 团队审查变更日志
4. 在 staging 环境测试
5. 合并到主分支
6. 监控生产环境

### 依赖审批

**无需审批**:
- 安全补丁
- 小版本更新（无 breaking changes）

**需要审批**:
- 大版本更新
- 新增依赖
- 移除依赖

### 依赖原则

1. **最小化依赖**: 只添加必要的依赖
2. **活跃维护**: 选择活跃维护的库
3. **安全第一**: 及时更新安全补丁
4. **版本锁定**: 使用精确版本号
5. **定期审查**: 每季度审查依赖列表

## 安全策略

### 安全扫描

**自动扫描**:
- 依赖漏洞扫描（Dependabot）
- 代码安全扫描（CodeQL）
- 容器镜像扫描

**手动审查**:
- 季度安全审计
- 渗透测试（生产环境）

### 漏洞响应

**严重程度分级**:
- **Critical**: 24 小时内修复
- **High**: 3 天内修复
- **Medium**: 1 周内修复
- **Low**: 下次发布时修复

**响应流程**:
1. 评估影响范围
2. 创建修复 PR
3. 加速审查流程
4. 部署修复
5. 通知相关方
6. 更新文档

### 安全最佳实践

- ✅ 不提交敏感信息（密钥、密码）
- ✅ 使用环境变量配置
- ✅ 定期更新依赖
- ✅ 使用 HTTPS/TLS
- ✅ 实施最小权限原则
- ✅ 启用 2FA
- ✅ 审查第三方代码

## 发布流程

### 版本号规范

使用语义化版本（Semantic Versioning）:
- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

示例: `v1.2.3`

### 发布步骤

1. **准备发布**:
   - 更新版本号
   - 更新 CHANGELOG
   - 运行完整测试套件

2. **创建发布分支**:
   ```bash
   git checkout -b release/v1.2.3
   ```

3. **构建和测试**:
   ```bash
   make build
   make test
   make docker-build
   ```

4. **部署到 Staging**:
   ```bash
   kubectl apply -k k8s/overlays/staging
   ```

5. **验证 Staging**:
   - 运行集成测试
   - 手动验证关键功能
   - 性能测试

6. **创建 Git Tag**:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

7. **部署到 Production**:
   ```bash
   kubectl apply -k k8s/overlays/production
   ```

8. **发布公告**:
   - 创建 GitHub Release
   - 更新文档
   - 通知团队

### 回滚流程

如果发现问题需要回滚：

1. **立即回滚**:
   ```bash
   kubectl rollout undo deployment/service-name
   ```

2. **验证回滚**:
   - 检查服务状态
   - 验证功能正常

3. **分析问题**:
   - 查看日志
   - 分析根本原因
   - 创建修复计划

4. **修复和重新发布**:
   - 修复问题
   - 重新测试
   - 按正常流程发布

## 持续改进

### 定期审查

**每月**:
- 审查 PR 流程效率
- 审查代码质量指标
- 审查依赖更新状态

**每季度**:
- 审查治理流程
- 审查团队职责
- 审查工具和自动化

**每年**:
- 全面架构审查
- 技术栈评估
- 团队结构优化

### 反馈机制

- **团队会议**: 每周同步
- **回顾会议**: 每月回顾
- **改进提案**: 随时提交
- **匿名反馈**: 季度调查

### 文档维护

- 保持文档更新
- 记录重要决策
- 分享最佳实践
- 培训新成员

## 联系方式

### 团队联系

- **Platform Team**: platform-team@example.com
- **Backend Java Team**: backend-java@example.com
- **Backend Go Team**: backend-go@example.com
- **Frontend Team**: frontend@example.com

### 问题报告

- **Bug 报告**: 创建 GitHub Issue
- **功能请求**: 创建 GitHub Issue
- **安全问题**: security@example.com
- **紧急问题**: 联系 on-call 工程师

## 参考资源

- [GitHub CODEOWNERS 文档](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners)
- [语义化版本规范](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Google 工程实践](https://google.github.io/eng-practices/)
