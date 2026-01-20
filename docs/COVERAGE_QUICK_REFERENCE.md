# 测试覆盖率快速参考

## 基本用法

```bash
# 运行所有应用的覆盖率测试
make test-coverage

# 运行特定应用的覆盖率测试
make test-coverage APP=hello
make test-coverage APP=shortener
make test-coverage APP=todo

# 验证覆盖率阈值（CI 使用）
make verify-coverage
make verify-coverage APP=hello
```

## 直接使用脚本

```bash
# 运行所有应用
./scripts/coverage-manager.sh

# 运行特定应用（支持短名称）
./scripts/coverage-manager.sh hello
./scripts/coverage-manager.sh shortener-service

# 验证覆盖率阈值
./scripts/coverage-manager.sh --verify
./scripts/coverage-manager.sh hello --verify
```

## 覆盖率报告位置

### Java 应用 (hello-service)
- **Gradle**: `apps/hello-service/build/reports/jacoco/test/html/index.html`
- **Maven**: `apps/hello-service/target/site/jacoco/index.html`

### Go 应用 (todo-service, shortener-service)
- **报告**: `apps/*/coverage.html`
- **原始数据**: `apps/*/coverage.out`

### Node.js 应用 (web)
- **报告**: `apps/web/coverage/index.html`

## 覆盖率阈值

### hello-service (Java)
- **整体覆盖率**: 30% 最低
- **服务类覆盖率**: 50% 最低

### todo-service (Go)
- **整体覆盖率**: 70% 最低
- **服务/存储包**: 75% 最低

### shortener-service (Go)
- **整体覆盖率**: 70% 最低
- **服务/存储包**: 75% 最低

## 常见问题

### Q: 如何查看覆盖率报告？
A: 运行 `make test-coverage APP=<name>` 后，打开对应的 HTML 报告文件。

### Q: 覆盖率测试失败怎么办？
A: 
1. 检查测试是否通过: `make test APP=<name>`
2. 查看覆盖率报告，找出未覆盖的代码
3. 添加或改进测试用例

### Q: 如何提高覆盖率？
A:
1. 为核心业务逻辑添加单元测试
2. 为边界条件添加测试用例
3. 使用属性测试（Property-Based Testing）
4. 排除生成的代码和配置类

### Q: CI 中覆盖率验证失败？
A: 
1. 本地运行 `make verify-coverage APP=<name>`
2. 检查是否达到最低阈值
3. 如果阈值过高，考虑调整配置
4. 如果覆盖率不足，添加测试

## 提示

- 使用短名称更方便: `hello` 代替 `hello-service`
- 覆盖率报告会自动生成 HTML 格式
- 验证模式会检查覆盖率阈值并在不达标时失败
- 所有命令都支持彩色输出，便于识别成功/失败
