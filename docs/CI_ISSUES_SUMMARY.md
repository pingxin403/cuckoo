# CI Issues Summary

**日期**: 2026-01-20  
**状态**: 🔍 分析中

## Issue 1: Hello Service - Zero Test Coverage ❌

**错误信息**:
```
Rule violated for bundle hello-service: instructions covered ratio is 0.00, but expected minimum is 0.30
```

**原因分析**:
- JaCoCo 报告 0% 测试覆盖率（期望最低 30%）
- 集成测试被标记为 `@Tag("integration")` 并从常规测试运行中排除
- 单元测试存在但可能没有正确运行或覆盖不足

**现有测试文件**:
- `HelloServiceApplicationTests.java` - 应用启动测试
- `HelloServiceImplTest.java` - 服务实现单元测试
- `HelloServicePropertyTest.java` - 属性测试
- `HelloServiceIntegrationTest.java` - 集成测试（已排除）

**可能的解决方案**:
1. 检查单元测试是否正确运行
2. 确保单元测试覆盖服务代码
3. 可能需要添加更多单元测试
4. 检查 JaCoCo 配置是否正确排除了生成的代码

## Issue 2: Shortener Service - Missing Generated Protobuf Code ❌

**错误信息**:
```
no required module provides package github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb
```

**原因分析**:
- Go 服务在运行测试前需要生成 protobuf 代码
- CI 工作流中的 Go 服务构建步骤包含 `make proto-go`
- 但是测试覆盖率脚本可能在 proto 生成之前运行

**CI 工作流步骤**:
```yaml
- name: Generate proto for Go
  if: steps.detect-type.outputs.type == 'go'
  run: make proto-go

- name: Build Go service
  if: steps.detect-type.outputs.type == 'go'
  run: |
    cd apps/${{ matrix.app }}
    go mod download
    ./scripts/test-coverage.sh  # ← 这里运行测试
    go build -v -o bin/${{ matrix.app }} .
```

**问题**:
- `make proto-go` 在工作流中运行
- 但是 `test-coverage.sh` 脚本可能期望 proto 代码已经存在
- 或者 `go.mod` 中缺少对生成代码的引用

**解决方案**:
1. 确保 `make proto-go` 在测试前运行（已经在 CI 中）
2. 检查 `go.mod` 是否正确引用生成的代码
3. 可能需要在 `test-coverage.sh` 脚本中添加 proto 生成步骤
4. 或者确保生成的代码被提交到仓库（不推荐）

## 下一步行动

### Hello Service
1. 本地运行 `cd apps/hello-service && ./gradlew test jacocoTestReport`
2. 检查测试报告：`apps/hello-service/build/reports/tests/test/index.html`
3. 检查覆盖率报告：`apps/hello-service/build/reports/jacoco/test/html/index.html`
4. 确定哪些测试没有运行或哪些代码没有被覆盖

### Shortener Service
1. 本地运行 `make proto-go`
2. 检查生成的代码：`apps/shortener-service/gen/`
3. 运行测试：`cd apps/shortener-service && go test ./...`
4. 检查 `go.mod` 中的依赖关系
5. 可能需要在 `test-coverage.sh` 中添加 proto 生成步骤

## 相关文件

- `.github/workflows/ci.yml` - CI 工作流配置
- `apps/hello-service/build.gradle` - Gradle 构建配置
- `apps/hello-service/scripts/test-coverage.sh` - 测试覆盖率脚本
- `apps/shortener-service/scripts/test-coverage.sh` - 测试覆盖率脚本
- `Makefile` - Proto 生成命令
