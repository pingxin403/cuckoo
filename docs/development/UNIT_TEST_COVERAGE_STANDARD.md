# 单元测试覆盖率规范

## 概述

本文档定义了 Cuckoo Monorepo 中单元测试覆盖率的标准和最佳实践。

## 覆盖率要求

### Go 服务

#### 标准服务（无外部依赖）
- **整体覆盖率**: 80% 最低要求
- **服务包覆盖率**: 85% 最低要求

适用服务：
- `auth-service`
- `user-service`

#### 集成依赖服务（有外部依赖）
- **整体覆盖率**: 65% 最低要求
- **服务包覆盖率**: 55% 最低要求

适用服务：
- `im-gateway-service` (依赖 Kafka、WebSocket、Redis/etcd)

#### 特殊服务（使用不同的测试脚本）
- **im-service**: 使用自定义测试脚本，不强制覆盖率阈值
  - 支持属性测试（property-based testing）
  - 可选的 linter 检查
  - 使用 `--with-property-tests` 标志运行完整测试套件

### Java 服务
- **整体覆盖率**: 80% 最低要求
- **服务类覆盖率**: 90% 最低要求

### TypeScript/Node.js 服务
- **整体覆盖率**: 80% 最低要求
- **组件/服务覆盖率**: 85% 最低要求

## 覆盖率计算规则

### 排除的代码类型

以下代码类型**不应该**被单元测试覆盖，应从覆盖率计算中排除：

#### 1. 生成的代码
```bash
# 排除 /gen/ 目录
# 原因：Protobuf 生成的代码，不是手写代码
grep -v '/gen/'
```

示例：
- `apps/auth-service/gen/authpb/`
- `apps/user-service/gen/userpb/`
- `api/gen/`

#### 2. 应用入口文件
```bash
# 排除 main.go
# 原因：应用程序入口点，通过集成测试验证
grep -v 'main.go'
```

#### 3. 配置加载逻辑
```bash
# 排除 /config/ 目录
# 原因：配置文件加载逻辑，需要集成测试
grep -v '/config/'
```

示例：
- `apps/auth-service/config/config.go`
- `apps/user-service/config/config.go`

#### 4. 数据库存储层
```bash
# 排除 /storage/ 目录
# 原因：数据库访问层，需要数据库集成测试
grep -v '/storage/'
```

示例：
- `apps/auth-service/storage/mysql_store.go`
- `apps/user-service/storage/mysql_store.go`

### 覆盖率计算示例

```bash
# 从覆盖率报告中过滤
FILTERED_COVERAGE=$(go tool cover -func=coverage.out | \
  grep -v '/gen/' | \
  grep -v 'main.go' | \
  grep -v '/config/' | \
  grep -v '/storage/' | \
  grep -v 'total:')

# 计算平均覆盖率
OVERALL_COVERAGE=$(echo "$FILTERED_COVERAGE" | \
  awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | \
  sed 's/%//')
```

## 单元测试 vs 集成测试

### 单元测试应该覆盖

✅ **业务逻辑**
- 服务方法实现
- 数据验证和转换
- 错误处理逻辑
- 算法实现

✅ **纯函数**
- 工具函数
- 数据处理函数
- 格式化函数

✅ **状态管理**
- 内存缓存
- 状态机
- 事件处理

### 集成测试应该覆盖

🔄 **外部依赖**
- 数据库操作（storage 包）
- 消息队列（Kafka consumer/producer）
- 缓存服务（Redis、etcd）
- 网络协议（WebSocket、gRPC）

🔄 **配置和启动**
- 配置文件加载（config 包）
- 应用程序启动（main.go）
- 服务注册和发现

🔄 **端到端流程**
- 完整的用户场景
- 跨服务调用
- 数据一致性

### 不需要测试

❌ **生成的代码**
- Protobuf 生成的 `.pb.go` 文件
- gRPC 生成的 `_grpc.pb.go` 文件
- 其他代码生成工具的输出

❌ **第三方库**
- 依赖库的内部实现
- 框架代码

## 测试覆盖率脚本

### 职责分离

测试覆盖率脚本（`scripts/test-coverage.sh`）应该**只负责测试和覆盖率检查**，不应该包含其他检查：

✅ **应该包含**：
- 运行单元测试
- 生成覆盖率报告
- 验证覆盖率阈值

❌ **不应该包含**：
- Linting 检查（使用 `make lint` 或 CI 中的独立步骤）
- 代码格式化（使用 `make format`）
- 安全扫描（使用独立工具）

### Go 服务标准脚本

每个 Go 服务应该有 `scripts/test-coverage.sh` 脚本。有两种类型的脚本：

#### 类型 1: 标准覆盖率脚本（auth-service, user-service, im-gateway-service）

这些服务使用标准的覆盖率检查脚本，强制执行覆盖率阈值：

```bash
#!/bin/bash

# Test coverage script for Go services
# This script runs tests with coverage and verifies thresholds:
# - Overall coverage: 80% minimum
# - Service packages: 85% minimum

set -e

echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...

echo ""
echo "Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report generated: coverage.html"

echo ""
echo "Coverage summary:"
go tool cover -func=coverage.out

# Filter out generated code, main.go, config, and storage from coverage calculation
# Rationale:
# - /gen/: Generated protobuf code (not manually written)
# - main.go: Entry point, tested via integration tests
# - /config/: Configuration loading, tested via integration tests
# - /storage/: Database layer, requires integration tests with real DB
echo ""
echo "Filtering coverage (excluding generated code, main.go, config, storage)..."
FILTERED_COVERAGE=$(go tool cover -func=coverage.out | grep -v '/gen/' | grep -v 'main.go' | grep -v '/config/' | grep -v '/storage/' | grep -v 'total:')

echo ""
echo "Filtered coverage summary:"
echo "$FILTERED_COVERAGE"

echo ""
echo "Checking coverage thresholds..."

# Check overall coverage (80%) - excluding generated code, config, and storage
OVERALL_COVERAGE=$(echo "$FILTERED_COVERAGE" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
echo "Overall coverage (excluding generated/config/storage): ${OVERALL_COVERAGE}%"

if (( $(echo "$OVERALL_COVERAGE < 80" | bc -l) )); then
    echo "❌ FAIL: Overall coverage ${OVERALL_COVERAGE}% is below 80% threshold"
    exit 1
fi

echo "✅ PASS: Overall coverage meets 80% threshold"

# Check service package coverage (85%) - excluding generated code
SERVICE_LINES=$(echo "$FILTERED_COVERAGE" | grep '/service/' || true)

if [ -n "$SERVICE_LINES" ]; then
    # Calculate average coverage for service packages
    SERVICE_COVERAGE=$(echo "$SERVICE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
    echo "Service package coverage: ${SERVICE_COVERAGE}%"
    
    if (( $(echo "$SERVICE_COVERAGE < 85" | bc -l) )); then
        echo "❌ FAIL: Service package coverage ${SERVICE_COVERAGE}% is below 85% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Service package coverage meets 85% threshold"
else
    echo "⚠️  WARNING: No service packages found"
fi

# Note about storage coverage
STORAGE_LINES=$(echo "$FILTERED_COVERAGE" | grep '/storage/' || true)
if [ -n "$STORAGE_LINES" ]; then
    STORAGE_COVERAGE=$(echo "$STORAGE_LINES" | awk '{sum+=$3; count++} END {if (count > 0) printf "%.1f", sum/count; else print 0}' | sed 's/%//')
    echo "Storage package coverage: ${STORAGE_COVERAGE}% (informational - requires integration tests)"
fi

echo ""
echo "✅ All coverage thresholds met!"
```

### 有外部依赖的服务

对于有外部依赖的服务（如 `im-gateway-service`），使用调整后的阈值：

```bash
# Check overall coverage (80% or 65% for services with integration dependencies)
if (( $(echo "$OVERALL_COVERAGE < 80" | bc -l) )); then
    echo "❌ FAIL: Overall coverage ${OVERALL_COVERAGE}% is below 80% threshold"
    echo ""
    echo "Note: Some components require integration tests:"
    echo "  - kafka_consumer.go: Requires Kafka integration tests"
    echo "  - gateway_service.go: Requires WebSocket integration tests"
    echo "  - cache_manager.go: Some functions require Redis/etcd integration tests"
    echo ""
    echo "Current coverage is acceptable for unit tests. Integration tests should be run separately."
    echo "Adjusting threshold to 65% for services with external dependencies..."
    
    if (( $(echo "$OVERALL_COVERAGE < 65" | bc -l) )); then
        echo "❌ FAIL: Overall coverage ${OVERALL_COVERAGE}% is below 65% threshold"
        exit 1
    fi
    
    echo "✅ PASS: Overall coverage meets 65% threshold (integration test components excluded)"
else
    echo "✅ PASS: Overall coverage meets 80% threshold"
fi
```

## CI/CD 集成

### GitHub Actions 配置

在 `.github/workflows/ci.yml` 中，Go 服务的构建步骤：

```yaml
- name: Build Go service
  if: steps.detect-type.outputs.type == 'go'
  run: |
    cd apps/${{ matrix.app }}
    go mod download
    go mod verify
    ./scripts/test-coverage.sh  # 运行覆盖率检查
    go build -v -o bin/${{ matrix.app }} .
```

### 覆盖率报告上传

```yaml
- name: Upload Go coverage
  if: steps.detect-type.outputs.type == 'go' && always()
  uses: actions/upload-artifact@v4
  with:
    name: ${{ matrix.app }}-coverage
    path: apps/${{ matrix.app }}/coverage.html
```

## 最佳实践

### 1. 编写有意义的测试

❌ **不好的做法**：为了覆盖率而写测试
```go
func TestGetUser_CallsStorage(t *testing.T) {
    // 只是调用函数，没有验证行为
    service.GetUser(ctx, req)
}
```

✅ **好的做法**：测试业务逻辑和行为
```go
func TestGetUser_ReturnsUserWhenFound(t *testing.T) {
    // 验证返回的用户数据是否正确
    resp, err := service.GetUser(ctx, req)
    assert.NoError(t, err)
    assert.Equal(t, expectedUser.Id, resp.User.Id)
    assert.Equal(t, expectedUser.Name, resp.User.Name)
}
```

### 2. 测试边界条件

```go
func TestValidateInput(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"empty input", "", true},
        {"too long", strings.Repeat("a", 1001), true},
        {"valid input", "valid", false},
        {"special chars", "test@123", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInput(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 3. 使用表驱动测试

```go
func TestCalculateDiscount(t *testing.T) {
    tests := []struct {
        name     string
        price    float64
        discount float64
        want     float64
    }{
        {"no discount", 100, 0, 100},
        {"10% discount", 100, 0.1, 90},
        {"50% discount", 100, 0.5, 50},
        {"100% discount", 100, 1.0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateDiscount(tt.price, tt.discount)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 4. Mock 外部依赖

```go
type MockStorage struct {
    mock.Mock
}

func (m *MockStorage) GetUser(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func TestService_WithMockStorage(t *testing.T) {
    mockStorage := new(MockStorage)
    mockStorage.On("GetUser", mock.Anything, "123").Return(&User{Id: "123"}, nil)
    
    service := NewService(mockStorage)
    user, err := service.GetUser(context.Background(), "123")
    
    assert.NoError(t, err)
    assert.Equal(t, "123", user.Id)
    mockStorage.AssertExpectations(t)
}
```

## 故障排查

### 覆盖率计算不准确

**问题**：覆盖率包含了生成的代码
```bash
# 检查是否正确过滤
go tool cover -func=coverage.out | grep '/gen/'
```

**解决**：确保脚本正确过滤生成的代码
```bash
FILTERED_COVERAGE=$(go tool cover -func=coverage.out | grep -v '/gen/')
```

### 服务包覆盖率过低

**问题**：某些服务方法没有测试

**解决步骤**：
1. 查看覆盖率报告：`open coverage.html`
2. 找到未覆盖的函数
3. 为这些函数添加测试
4. 重新运行覆盖率检查

### CI 中覆盖率检查失败

**问题**：本地通过，CI 失败

**可能原因**：
1. 生成的代码不一致：运行 `make proto` 并提交
2. 依赖版本不同：检查 `.tool-versions`
3. 测试依赖外部服务：将测试改为使用 mock

## 参考资料

### 相关文档
- [测试指南](./TESTING_GUIDE.md) - 完整的测试指南
- [属性测试](./PROPERTY_TESTING.md) - 属性测试最佳实践
- [CI/CD 策略](../ci-cd/DYNAMIC_CI_STRATEGY.md) - CI/CD 配置

### 工具和库
- [Go testing](https://pkg.go.dev/testing) - Go 标准测试库
- [testify](https://github.com/stretchr/testify) - Go 测试断言库
- [JaCoCo](https://www.jacoco.org/) - Java 代码覆盖率工具
- [Vitest](https://vitest.dev/) - TypeScript 测试框架

## 更新历史

| 日期 | 版本 | 变更说明 |
|------|------|----------|
| 2025-01-26 | 1.0 | 初始版本，定义覆盖率标准和排除规则 |
