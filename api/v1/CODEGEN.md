# Protobuf Code Generation Guide

本文档说明如何从 Protobuf 定义生成各语言的代码。

## 前置要求

### 必需工具

1. **Protocol Buffers Compiler (protoc)**
   ```bash
   # macOS
   brew install protobuf
   
   # Ubuntu/Debian
   apt-get install protobuf-compiler
   
   # 验证安装
   protoc --version
   ```

2. **Go Protobuf 插件** (用于 Go 代码生成)
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

3. **Java gRPC 插件** (可选，Maven/Gradle 会自动处理)
   - Java 代码生成通常由 Maven 或 Gradle 的 Protobuf 插件处理
   - 手动生成需要下载 protoc-gen-grpc-java

4. **TypeScript 插件** (用于 TypeScript 代码生成)
   ```bash
   # 在 apps/web 目录中安装
   cd apps/web
   npm install ts-proto
   ```

## 代码生成命令

### 生成所有语言的代码

```bash
make gen-proto
```

这会依次执行：
- `make gen-proto-go` - 生成 Go 代码
- `make gen-proto-java` - 生成 Java 代码（如果可用）
- `make gen-proto-ts` - 生成 TypeScript 代码（如果 web 应用存在）

### 单独生成特定语言

```bash
# 仅生成 Go 代码
make gen-proto-go

# 仅生成 Java 代码
make gen-proto-java

# 仅生成 TypeScript 代码
make gen-proto-ts
```

## 生成代码的位置

| 语言 | 输出目录 | 说明 |
|------|----------|------|
| Go | `apps/todo-service/gen/` | 直接在 gen 目录下生成 .pb.go 文件 |
| Java | `apps/hello-service/src/main/java-gen/` | 按 Java 包结构生成 |
| TypeScript | `apps/web/src/gen/` | 生成 .ts 文件 |

## CI/CD 集成

### 验证生成代码是否最新

在 CI 流水线中使用以下命令验证提交的代码是最新的：

```bash
make verify-proto
```

如果生成的代码与 Protobuf 定义不一致，此命令会失败并显示差异。

### Pre-commit Hook

项目配置了 pre-commit hook，在提交包含 `.proto` 文件的更改时自动检查生成代码：

```bash
# Hook 位置
.git/hooks/pre-commit

# 如果生成代码过期，hook 会阻止提交并提示：
# 1. 运行 make gen-proto
# 2. 将生成的文件添加到暂存区
# 3. 重新提交
```

## 修改 API 的工作流程

1. **修改 Protobuf 定义**
   ```bash
   # 编辑 .proto 文件
   vim api/v1/hello.proto
   vim api/v1/todo.proto
   ```

2. **重新生成代码**
   ```bash
   make gen-proto
   ```

3. **检查生成的代码**
   ```bash
   git status
   git diff apps/*/gen apps/*/src/main/java-gen apps/*/src/gen
   ```

4. **更新服务实现**
   - 根据新的接口定义更新服务实现代码
   - 更新测试用例

5. **提交更改**
   ```bash
   # 同时提交 .proto 文件和生成的代码
   git add api/v1/*.proto
   git add apps/*/gen apps/*/src/main/java-gen apps/*/src/gen
   git commit -m "Update API: add new field to Todo"
   ```

## 常见问题

### Q: 为什么要提交生成的代码？

A: 提交生成的代码有以下好处：
- 确保构建的确定性（不依赖本地环境）
- 便于代码审查（可以看到 API 变更的影响）
- 简化 CI/CD（不需要在每次构建时生成）
- 避免版本不一致问题

### Q: protoc-gen-grpc-java 在哪里下载？

A: 通常不需要手动下载。Java 项目使用 Maven/Gradle 插件自动处理：

```xml
<!-- Maven pom.xml -->
<plugin>
    <groupId>org.xolstice.maven.plugins</groupId>
    <artifactId>protobuf-maven-plugin</artifactId>
    <version>0.6.1</version>
</plugin>
```

如需手动下载：https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/

### Q: 生成的代码报错怎么办？

A: 检查以下几点：
1. protoc 版本是否足够新（建议 3.20+）
2. 插件版本是否兼容
3. .proto 文件语法是否正确
4. import 路径是否正确

### Q: 如何添加新的 Protobuf 文件？

A: 
1. 在 `api/v1/` 目录创建新的 `.proto` 文件
2. 更新 Makefile 中的生成命令（如果需要特殊处理）
3. 运行 `make gen-proto`
4. 提交 `.proto` 文件和生成的代码

## 最佳实践

1. **始终使用 make 命令生成代码**，不要手动运行 protoc
2. **修改 .proto 后立即生成代码**，避免忘记
3. **在 PR 中同时审查 .proto 和生成的代码**
4. **保持 protoc 和插件版本更新**
5. **为新的消息和字段添加清晰的注释**

## 参考资料

- [Protocol Buffers 官方文档](https://protobuf.dev/)
- [gRPC 官方文档](https://grpc.io/docs/)
- [protoc-gen-go 文档](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go)
- [ts-proto 文档](https://github.com/stephenh/ts-proto)
