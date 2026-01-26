# Bazel 迁移计划

**状态**: 📋 计划中  
**优先级**: 中期（6-12个月）  
**最后更新**: 2026-01-26

## 背景

当前项目使用多种构建工具和依赖管理系统：
- Go: go mod
- Java: Gradle
- Node.js: npm
- Protobuf: 手动管理

虽然我们已经通过增强 Makefile 实现了统一的命令接口，但随着项目规模增长，我们需要考虑更强大的构建系统。

## 为什么考虑 Bazel？

### 优势
1. **统一构建**: 所有语言使用同一套构建规则
2. **增量构建**: 只重新构建变更的部分，大幅提升构建速度
3. **可重现构建**: 确保构建结果一致性
4. **强大的缓存**: 本地和远程缓存支持
5. **依赖管理**: 精确的依赖图和版本管理
6. **大规模支持**: 适合大型 monorepo

### 挑战
1. **学习曲线**: 团队需要学习新工具
2. **迁移成本**: 需要重写构建配置
3. **生态系统**: 相对小众，社区资源较少
4. **调试复杂**: 构建问题排查较困难

## 迁移触发条件

当满足以下任一条件时，应考虑启动 Bazel 迁移：

### 硬性指标
- ✅ 团队规模 > 10 人
- ✅ 服务数量 > 20 个
- ✅ 完整构建时间 > 10 分钟
- ✅ 每日构建次数 > 100 次

### 软性指标
- 频繁的跨语言依赖问题
- 构建不一致性问题频发
- CI/CD 构建时间成为瓶颈
- 需要更精细的依赖管理

## 当前状态评估

### 项目规模（截至 2026-01-26）
- **团队规模**: ~5 人
- **服务数量**: 7 个（auth, user, im, im-gateway, shortener, hello, todo）
- **构建时间**: ~5 分钟（完整构建）
- **每日构建**: ~20 次

### 结论
✅ **当前不需要迁移到 Bazel**

项目规模尚未达到需要 Bazel 的程度。当前的 Makefile + 原生工具链方案足够满足需求。

## 迁移路线图

### Phase 0: 准备阶段（当前）
**时间**: 持续进行  
**目标**: 监控项目规模，评估迁移时机

**行动项**:
- ✅ 实施统一的 Makefile 依赖管理
- ✅ 记录构建时间和性能指标
- ✅ 收集团队反馈
- ⏳ 每季度评估项目规模
- ⏳ 关注 Bazel 生态发展

**完成标准**:
- 统一依赖管理命令已实施
- 构建性能基线已建立
- 团队对当前工具链熟悉

### Phase 1: 学习和评估（3-6 个月前）
**时间**: 2-3 个月  
**目标**: 团队学习 Bazel，评估可行性

**行动项**:
- [ ] 团队 Bazel 培训
- [ ] 创建 Bazel POC 项目
- [ ] 迁移 1-2 个小服务作为试点
- [ ] 性能对比测试
- [ ] 成本收益分析
- [ ] 团队反馈收集

**完成标准**:
- 至少 2 个服务成功迁移到 Bazel
- 构建性能提升 > 30%
- 团队对 Bazel 有基本了解
- 迁移方案得到团队认可

### Phase 2: 试点迁移（3-4 个月）
**时间**: 3-4 个月  
**目标**: 迁移 30% 的服务

**行动项**:
- [ ] 创建 WORKSPACE 和基础配置
- [ ] 迁移 Go 服务（3-4 个）
- [ ] 迁移 Java 服务（1-2 个）
- [ ] 迁移 Protobuf 构建
- [ ] 设置 CI/CD 集成
- [ ] 建立最佳实践文档

**完成标准**:
- 30% 服务使用 Bazel 构建
- CI/CD 流程正常运行
- 构建时间减少 > 40%
- 无重大阻塞问题

### Phase 3: 全面迁移（4-6 个月）
**时间**: 4-6 个月  
**目标**: 迁移所有服务

**行动项**:
- [ ] 迁移剩余 Go 服务
- [ ] 迁移剩余 Java 服务
- [ ] 迁移前端项目
- [ ] 设置远程缓存
- [ ] 优化构建规则
- [ ] 完善文档和培训

**完成标准**:
- 100% 服务使用 Bazel 构建
- 远程缓存正常工作
- 构建时间减少 > 50%
- 团队熟练使用 Bazel

### Phase 4: 优化和维护（持续）
**时间**: 持续  
**目标**: 持续优化构建性能

**行动项**:
- [ ] 监控构建性能
- [ ] 优化缓存策略
- [ ] 更新 Bazel 版本
- [ ] 分享最佳实践
- [ ] 解决团队问题

## 技术方案预览

### WORKSPACE 文件结构
```python
workspace(name = "cuckoo")

# Go 规则
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "...",
    urls = ["..."],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
go_rules_dependencies()
go_register_toolchains(version = "1.21.0")

# Java 规则
http_archive(
    name = "rules_java",
    sha256 = "...",
    urls = ["..."],
)

# Node.js 规则
http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "...",
    urls = ["..."],
)

# Protobuf 规则
http_archive(
    name = "rules_proto",
    sha256 = "...",
    urls = ["..."],
)
```

### BUILD.bazel 示例（Go 服务）
```python
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "auth_service_lib",
    srcs = glob(["service/*.go"]),
    importpath = "github.com/pingxin403/cuckoo/apps/auth-service",
    deps = [
        "//api/gen/authpb:authpb_go_proto",
        "@org_golang_google_grpc//:go_default_library",
    ],
)

go_binary(
    name = "auth-service",
    embed = [":auth_service_lib"],
)

go_test(
    name = "auth_service_test",
    srcs = glob(["service/*_test.go"]),
    embed = [":auth_service_lib"],
)
```

### 统一命令
```bash
# 构建所有服务
bazel build //...

# 测试所有服务
bazel test //...

# 运行特定服务
bazel run //apps/auth-service

# 清理
bazel clean

# 查询依赖
bazel query 'deps(//apps/auth-service)'
```

## 成本估算

### 人力成本
- **学习阶段**: 2 人周/人
- **POC 开发**: 4 人周
- **试点迁移**: 12 人周
- **全面迁移**: 24 人周
- **总计**: ~42 人周（约 2-3 个月全职工作）

### 风险评估
- **高风险**: 团队学习曲线陡峭
- **中风险**: 迁移过程中可能影响开发效率
- **低风险**: Bazel 生态不够成熟

### 收益预期
- **构建时间**: 减少 50-70%
- **CI/CD 成本**: 减少 40-60%
- **开发体验**: 提升 30-40%
- **构建一致性**: 提升 90%+

## 决策标准

### 何时启动迁移？
当满足以下条件时，应启动 Bazel 迁移评估：
1. 团队规模 > 10 人 **或**
2. 服务数量 > 20 个 **或**
3. 构建时间 > 10 分钟 **且** 团队反馈构建速度是主要痛点

### 何时放弃迁移？
如果在 Phase 1 评估后发现：
1. 性能提升 < 20%
2. 团队学习成本过高
3. 迁移风险大于收益
4. 有更好的替代方案

## 监控指标

### 需要持续跟踪的指标
- **构建时间**: 完整构建和增量构建时间
- **CI/CD 时间**: 每次 CI 运行时间
- **服务数量**: 项目中的服务总数
- **团队规模**: 活跃开发者数量
- **构建频率**: 每日构建次数
- **构建失败率**: 构建失败的比例

### 当前基线（2026-01-26）
- 完整构建时间: ~5 分钟
- CI/CD 时间: ~8 分钟
- 服务数量: 7 个
- 团队规模: ~5 人
- 构建频率: ~20 次/天
- 构建失败率: ~5%

## 替代方案

如果 Bazel 不适合，可以考虑：

1. **Pants**: 类似 Bazel 但更简单，对 Python/Go 友好
2. **Nx**: 适合 JavaScript/TypeScript 为主的项目
3. **增强 Makefile**: 继续优化当前方案
4. **Buck2**: Facebook 的新一代构建系统

## 参考资源

### 官方文档
- [Bazel 官方文档](https://bazel.build/)
- [rules_go](https://github.com/bazelbuild/rules_go)
- [rules_java](https://github.com/bazelbuild/rules_java)
- [rules_nodejs](https://github.com/bazelbuild/rules_nodejs)

### 案例研究
- [Google 的 Bazel 使用经验](https://bazel.build/about/intro)
- [Uber 的 Bazel 迁移](https://eng.uber.com/go-monorepo-bazel/)
- [Dropbox 的 Bazel 实践](https://dropbox.tech/infrastructure/continuous-integration-and-deployment-with-bazel)

### 社区资源
- [Bazel Slack](https://slack.bazel.build/)
- [Bazel 中文社区](https://bazel.build/community)

## 下一步行动

### 立即行动（已完成）
- ✅ 实施统一的 Makefile 依赖管理
- ✅ 创建 Bazel 迁移计划文档

### 短期行动（1-3 个月）
- ⏳ 建立构建性能监控
- ⏳ 收集团队对当前工具链的反馈
- ⏳ 关注 Bazel 生态发展

### 中期行动（3-6 个月）
- ⏳ 每季度评估项目规模
- ⏳ 当达到触发条件时，启动 Phase 1 评估

### 长期行动（6-12 个月）
- ⏳ 根据评估结果决定是否迁移
- ⏳ 如果迁移，按照路线图执行

## 总结

Bazel 是一个强大的构建系统，但不是所有项目都需要它。我们采取务实的态度：

1. **当前**: 使用增强的 Makefile 方案，满足当前需求
2. **监控**: 持续跟踪项目规模和构建性能
3. **评估**: 当达到触发条件时，启动正式评估
4. **决策**: 基于数据和团队反馈做出决策

这个计划确保我们在合适的时机做出正确的技术选择，既不过早优化，也不错过最佳时机。

---

**维护者**: 开发团队  
**审核周期**: 每季度  
**下次审核**: 2026-04-26
