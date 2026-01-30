# CI 缓存优化总结

## 优化内容

### 1. Proto 代码生成方式改进

**问题：**
- flash-sale-service 使用 protobuf 插件在构建时生成代码
- 导致 Dockerfile 复杂，需要处理 proto sourceSet 配置
- CI 中每次都需要重新生成 proto 代码

**解决方案：**
参考 hello-service 的方式，使用预生成的 proto 代码：

```gradle
// 直接使用预生成的代码
implementation files('../../api/gen/java')

// 在 sourceSets 中包含预生成的代码
sourceSets {
    main {
        java {
            srcDir 'src/main/java'
            srcDir '../../api/gen/java'
            include 'com/pingxin403/cuckoo/flashsale/**/*.java'
        }
    }
}

// 提供 no-op 的 generateProto 任务用于 CI 兼容性
tasks.register('generateProto') {
    description = 'Proto generation handled by monorepo root (make proto)'
    group = 'build'
    doLast {
        logger.lifecycle('Proto generation is handled by monorepo root - using pre-generated code from api/gen/java')
    }
}
```

**优势：**
- ✅ 简化 Dockerfile（不需要复制 proto 代码到特定位置）
- ✅ 构建更快（不需要运行 protobuf 插件）
- ✅ 与 monorepo 其他 Java 服务保持一致
- ✅ 更容易缓存（proto 代码可以被缓存）

### 2. Gradle 版本升级

**更新：**
- gRPC: `1.60.0` → `1.78.0`
- Protobuf: `3.25.1` → `4.33.4`

**原因：**
- 与 hello-service 保持一致
- 支持最新的 protobuf 生成代码
- 添加了 gRPC 依赖版本强制统一配置

### 3. Dockerfile 简化

**之前：**
```dockerfile
# 复杂的 proto 代码复制逻辑
COPY api/gen/java/com/pingxin403/cuckoo/flashsale ./build/generated/source/proto/main/java/com/pingxin403/cuckoo/flashsale

# 需要跳过多个 proto 相关任务
RUN ./gradlew compileJava bootJar -x test -x generateProto -x extractProto -x extractIncludeProto -x jacocoTestCoverageVerification --no-daemon
```

**现在：**
```dockerfile
# 简单的整体复制
COPY api/gen/java ./api/gen/java
COPY apps/flash-sale-service ./apps/flash-sale-service

# 标准构建命令
RUN ./gradlew build -x test --no-daemon --no-configuration-cache
```

### 4. CI 缓存优化

#### 4.1 Gradle 构建缓存
```yaml
- name: Cache Gradle build
  if: steps.detect-type.outputs.type == 'java'
  uses: actions/cache@v4
  with:
    path: |
      apps/${{ matrix.app }}/build
      apps/${{ matrix.app }}/.gradle
    key: ${{ runner.os }}-gradle-build-${{ matrix.app }}-${{ hashFiles('apps/${{ matrix.app }}/build.gradle', 'apps/${{ matrix.app }}/src/**') }}
    restore-keys: |
      ${{ runner.os }}-gradle-build-${{ matrix.app }}-
      ${{ runner.os }}-gradle-build-
```

**效果：**
- 缓存编译后的 class 文件
- 缓存 Gradle 任务输出
- 源代码未改变时可以跳过编译

#### 4.2 Proto 生成代码缓存
```yaml
- name: Cache proto generated code
  uses: actions/cache@v4
  with:
    path: |
      api/gen/java
      api/gen/go
      api/gen/typescript
    key: ${{ runner.os }}-proto-${{ hashFiles('api/v1/**/*.proto') }}
    restore-keys: |
      ${{ runner.os }}-proto-
```

**效果：**
- proto 文件未改变时直接使用缓存
- 避免重复生成 proto 代码
- 所有语言的生成代码都被缓存

#### 4.3 Docker 层缓存
```yaml
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Cache Docker layers
  uses: actions/cache@v4
  with:
    path: /tmp/.buildx-cache
    key: ${{ runner.os }}-buildx-${{ matrix.app }}-${{ github.sha }}
    restore-keys: |
      ${{ runner.os }}-buildx-${{ matrix.app }}-
      ${{ runner.os }}-buildx-

- name: Build Docker image
  uses: docker/build-push-action@v5
  with:
    cache-from: type=local,src=/tmp/.buildx-cache
    cache-to: type=local,dest=/tmp/.buildx-cache-new,mode=max
```

**效果：**
- 缓存 Docker 构建层
- 依赖未改变时可以重用层
- 显著加快 Docker 构建速度

#### 4.4 Go 构建缓存
```yaml
- name: Cache Go build
  if: steps.detect-type.outputs.type == 'go'
  uses: actions/cache@v4
  with:
    path: |
      apps/${{ matrix.app }}/bin
      ~/.cache/go-build
    key: ${{ runner.os }}-go-build-${{ matrix.app }}-${{ hashFiles('apps/${{ matrix.app }}/**/*.go') }}
    restore-keys: |
      ${{ runner.os }}-go-build-${{ matrix.app }}-
      ${{ runner.os }}-go-build-
```

**效果：**
- 缓存 Go 编译输出
- 缓存 Go 构建缓存目录
- 加快 Go 服务构建

## 性能提升预期

### 首次构建（无缓存）
- Proto 生成: ~30s
- Gradle 依赖下载: ~60s
- Java 编译: ~30s
- Docker 构建: ~120s
- **总计: ~240s (4分钟)**

### 后续构建（有缓存，代码未改变）
- Proto 生成: ~2s (缓存命中)
- Gradle 依赖: ~5s (缓存命中)
- Java 编译: ~5s (增量编译)
- Docker 构建: ~30s (层缓存)
- **总计: ~42s**

### 性能提升
- **约 82% 的时间节省**
- **从 4 分钟降至 42 秒**

## 缓存策略说明

### 缓存键设计

1. **Proto 缓存键**: `${{ hashFiles('api/v1/**/*.proto') }}`
   - 只有 proto 文件改变时才失效
   - 跨所有服务共享

2. **Gradle 构建缓存键**: `${{ hashFiles('apps/${{ matrix.app }}/build.gradle', 'apps/${{ matrix.app }}/src/**') }}`
   - build.gradle 或源代码改变时失效
   - 每个服务独立缓存

3. **Docker 缓存键**: `${{ matrix.app }}-${{ github.sha }}`
   - 每次提交都有新的缓存
   - 使用 restore-keys 回退到之前的缓存

### 缓存恢复策略

使用 `restore-keys` 实现渐进式缓存回退：
```yaml
restore-keys: |
  ${{ runner.os }}-gradle-build-${{ matrix.app }}-  # 同服务的旧缓存
  ${{ runner.os }}-gradle-build-                    # 其他服务的缓存
```

## 验证方法

### 本地验证
```bash
# 1. 清理构建
make clean APP=flash-sale-service

# 2. 测试编译
make build APP=flash-sale-service

# 3. 测试单元测试
make test APP=flash-sale-service

# 4. 测试 Docker 构建
docker build -t flash-sale-service:test -f apps/flash-sale-service/Dockerfile .
```

### CI 验证
1. 提交代码触发 CI
2. 查看 "Build flash-sale-service" job
3. 检查缓存命中情况：
   - "Cache Gradle build" - 查看 "Cache hit" 消息
   - "Cache proto generated code" - 查看 "Cache hit" 消息
   - "Cache Docker layers" - 查看 "Cache hit" 消息

## 最佳实践

### 1. Proto 代码管理
- ✅ 在 monorepo 根目录运行 `make proto` 生成所有代码
- ✅ 提交生成的代码到 git（确保一致性）
- ✅ 服务只引用预生成的代码，不自己生成

### 2. 依赖管理
- ✅ 使用 `resolutionStrategy` 强制统一 gRPC 版本
- ✅ 定期更新依赖版本
- ✅ 在 `.tool-versions` 中统一管理工具版本

### 3. 构建优化
- ✅ 使用 `--no-configuration-cache` 避免配置缓存问题
- ✅ 分离单元测试和集成测试
- ✅ 在 CI 中跳过测试（测试在单独的 job 中运行）

### 4. Docker 优化
- ✅ 使用多阶段构建
- ✅ 利用 Docker Buildx 缓存
- ✅ 按依赖变化频率排序 COPY 指令

## 相关文件

- `apps/flash-sale-service/build.gradle` - Gradle 配置
- `apps/flash-sale-service/Dockerfile` - Docker 构建配置
- `.github/workflows/ci.yml` - CI 工作流配置
- `api/v1/flash_sale_service.proto` - Proto 定义
- `api/gen/java/com/pingxin403/cuckoo/flashsale/` - 生成的 Java 代码

## 参考

- [hello-service build.gradle](../hello-service/build.gradle) - 参考实现
- [hello-service Dockerfile](../hello-service/Dockerfile) - 参考实现
- [GitHub Actions Cache](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [Docker Buildx Cache](https://docs.docker.com/build/cache/)
