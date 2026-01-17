# Getting Started Guide

欢迎使用 Monorepo Hello/TODO Services！本指南将帮助你在 5 分钟内完成环境设置并运行项目。

## 📋 前置条件

在开始之前，请确保你的系统满足以下要求：

### 必需工具

| 工具 | 最低版本 | 用途 |
|------|---------|------|
| Java | 17+ | Hello Service (Spring Boot) |
| Go | 1.21+ | TODO Service |
| Node.js | 18+ | Frontend (React) |
| npm | 8+ | 前端包管理 |
| protoc | 3.x | Protobuf 编译器 |

### 可选工具

| 工具 | 用途 | 重要性 |
|------|------|--------|
| Envoy | API 网关（前后端通信） | 强烈推荐 |
| Docker | 容器化部署 | 可选 |
| kubectl | Kubernetes 部署 | 可选 |
| golangci-lint | Go 代码检查 | 可选 |

## 🚀 快速开始

### 方式 1: 一键初始化（推荐）

```bash
# 1. 克隆项目
git clone <repository-url>
cd cuckoo

# 2. 检查环境
make check-env

# 3. 初始化（自动安装依赖）
make init

# 4. 启动所有服务
./scripts/dev.sh

# 5. 访问应用
# 打开浏览器访问 http://localhost:5173
```

就这么简单！🎉

### 方式 2: 手动设置

如果你想了解每一步的细节：

#### 步骤 1: 安装必需工具

**macOS**:
```bash
# 使用 Homebrew
brew install openjdk@17 go node protobuf

# 可选：安装 Envoy
brew install envoy
```

**Linux (Ubuntu/Debian)**:
```bash
# Java
sudo apt-get install openjdk-17-jdk

# Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Node.js (使用 nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18

# Protobuf
sudo apt-get install protobuf-compiler
```

#### 步骤 2: 安装 Go 工具

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 步骤 3: 安装前端依赖

```bash
cd apps/web
npm install
cd ../..
```

#### 步骤 4: 生成 Protobuf 代码

```bash
make gen-proto
```

#### 步骤 5: 安装 Git Hooks

```bash
./scripts/install-hooks.sh
```

#### 步骤 6: 构建所有服务

```bash
make build
```

#### 步骤 7: 启动服务

**选项 A: 使用开发脚本（需要 Envoy）**
```bash
./scripts/dev.sh
```

**选项 B: 手动启动各服务**
```bash
# 终端 1: Hello Service
cd apps/hello-service
./gradlew bootRun

# 终端 2: TODO Service
cd apps/todo-service
HELLO_SERVICE_ADDR=localhost:9090 go run .

# 终端 3: Frontend
cd apps/web
npm run dev
```

## 🔍 验证安装

### 检查环境

```bash
make check-env
```

这将检查所有必需和可选工具是否已安装。

### 测试服务

```bash
# 构建所有服务
make build

# 测试服务状态
./scripts/test-services.sh
```

### 访问应用

- **前端**: http://localhost:5173
- **Hello Service**: localhost:9090 (gRPC)
- **TODO Service**: localhost:9091 (gRPC)
- **Envoy Admin** (如果安装): http://localhost:9901

## 📚 下一步

### 开发工作流

1. **创建功能分支**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **进行开发**
   - 修改代码
   - 运行测试: `make test`
   - 运行 linter: `make lint`

3. **提交代码**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   # Pre-commit hooks 会自动运行
   ```

4. **推送并创建 PR**
   ```bash
   git push origin feature/my-feature
   ```

### 常用命令

```bash
# 生成 Protobuf 代码
make gen-proto

# 构建所有服务
make build

# 运行测试
make test

# 运行 linter
make lint

# 格式化代码
make format

# 构建 Docker 镜像
make docker-build

# 清理构建产物
make clean
```

### 学习资源

- [项目架构](ARCHITECTURE.md) - 了解系统架构
- [基础设施指南](INFRASTRUCTURE.md) - 部署和运维
- [代码质量指南](CODE_QUALITY.md) - 代码规范和工具
- [API 文档](../api/v1/README.md) - API 接口说明

## ❓ 常见问题

### Q: `make init` 失败怎么办？

**A**: 首先运行 `make check-env` 查看哪些工具缺失，然后手动安装缺失的工具。

### Q: 没有 Envoy 可以运行吗？

**A**: 可以！服务可以独立运行和测试。但前端无法通过 API 网关访问后端。你可以：
- 安装 Envoy: `brew install envoy`
- 或者直接测试后端服务（使用 grpcurl 等工具）

### Q: 端口被占用怎么办？

**A**: 检查并释放端口：
```bash
# 查看端口占用
lsof -i :9090
lsof -i :9091
lsof -i :5173

# 杀死进程
kill -9 <PID>
```

### Q: Protobuf 生成失败？

**A**: 确保安装了所有必需的工具：
```bash
# 检查 protoc
protoc --version

# 检查 Go 插件
which protoc-gen-go
which protoc-gen-go-grpc

# 重新安装插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Q: 前端构建失败？

**A**: 清理并重新安装依赖：
```bash
cd apps/web
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Q: Java 服务启动失败？

**A**: 检查 Java 版本和 Gradle：
```bash
java -version  # 应该是 17+
cd apps/hello-service
./gradlew clean build
```

### Q: Go 服务启动失败？

**A**: 检查 Go 版本和依赖：
```bash
go version  # 应该是 1.21+
cd apps/todo-service
go mod download
go build .
```

## 🆘 获取帮助

如果遇到问题：

1. **查看文档**: 检查 `docs/` 目录下的相关文档
2. **查看日志**: 服务日志在 `logs/` 目录
3. **运行诊断**: `make check-env` 和 `./scripts/test-services.sh`
4. **提交 Issue**: 在 GitHub 上创建 issue
5. **联系团队**: 在团队聊天中询问

## 🎯 成功标志

当你看到以下内容时，说明环境已经正确设置：

✅ `make check-env` 显示所有必需工具已安装  
✅ `make build` 成功构建所有服务  
✅ `./scripts/test-services.sh` 显示所有服务正在运行  
✅ 浏览器可以访问 http://localhost:5173  
✅ 前端可以与后端服务通信（如果安装了 Envoy）

恭喜！你已经准备好开始开发了！🚀

## 📖 推荐阅读顺序

1. ✅ **本文档** - 环境设置
2. 📐 [ARCHITECTURE.md](ARCHITECTURE.md) - 理解系统架构
3. 🔧 [INFRASTRUCTURE.md](INFRASTRUCTURE.md) - 了解基础设施
4. 📝 [CODE_QUALITY.md](CODE_QUALITY.md) - 学习代码规范
5. 🚀 开始编码！

---

**祝你编码愉快！** 如有问题，随时查阅文档或寻求帮助。
