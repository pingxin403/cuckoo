#!/bin/bash

# IM Chat System - 集成测试运行脚本
# 用途: 一键运行所有集成测试并生成报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_DIR="${WORKSPACE_ROOT}/test-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 创建报告目录
mkdir -p "${REPORT_DIR}"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Docker Compose 命令检测
DOCKER_COMPOSE_CMD=""

detect_docker_compose() {
    if command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    elif docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    else
        return 1
    fi
    return 0
}

# 检查前置条件
check_prerequisites() {
    log_info "检查前置条件..."
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    log_success "Go 版本: $(go version)"
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    log_success "Docker 版本: $(docker --version)"
    
    # 检查 Docker Compose
    if ! detect_docker_compose; then
        log_error "Docker Compose 未安装"
        log_error "请安装 Docker Compose 或使用 Docker Desktop (包含 docker compose)"
        exit 1
    fi
    log_success "Docker Compose 命令: ${DOCKER_COMPOSE_CMD}"
    
    # 显示版本
    if [ "$DOCKER_COMPOSE_CMD" = "docker-compose" ]; then
        log_success "Docker Compose 版本: $(docker-compose --version)"
    else
        log_success "Docker Compose 版本: $(docker compose version)"
    fi
}

# 启动基础设施
start_infrastructure() {
    log_info "启动基础设施服务..."
    
    cd "${WORKSPACE_ROOT}/deploy/docker"
    
    # 停止现有服务
    ${DOCKER_COMPOSE_CMD} -f docker-compose.infra.yml down -v 2>/dev/null || true
    
    # 启动服务
    ${DOCKER_COMPOSE_CMD} -f docker-compose.infra.yml up -d
    
    log_info "等待服务就绪..."
    sleep 30
    
    # 验证服务
    local all_healthy=true
    
    # 检查 etcd
    if docker exec etcd etcdctl endpoint health &> /dev/null; then
        log_success "✓ etcd 健康"
    else
        log_error "✗ etcd 不健康"
        all_healthy=false
    fi
    
    # 检查 MySQL
    if docker exec mysql mysql -uroot -proot_password -e "SELECT 1" &> /dev/null; then
        log_success "✓ MySQL 健康"
    else
        log_error "✗ MySQL 不健康"
        all_healthy=false
    fi
    
    # 检查 Redis
    if docker exec redis redis-cli ping &> /dev/null; then
        log_success "✓ Redis 健康"
    else
        log_error "✗ Redis 不健康"
        all_healthy=false
    fi
    
    # 检查 Kafka
    if docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null; then
        log_success "✓ Kafka 健康"
    else
        log_error "✗ Kafka 不健康"
        all_healthy=false
    fi
    
    if [ "$all_healthy" = false ]; then
        log_error "基础设施服务未就绪"
        exit 1
    fi
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    
    # 创建 im_chat 数据库
    docker exec mysql mysql -uroot -proot_password -e "CREATE DATABASE IF NOT EXISTS im_chat;"
    
    # 创建 im_service 用户
    docker exec mysql mysql -uroot -proot_password -e "
        CREATE USER IF NOT EXISTS 'im_service'@'%' IDENTIFIED BY 'im_service_password';
        GRANT ALL PRIVILEGES ON im_chat.* TO 'im_service'@'%';
        FLUSH PRIVILEGES;
    "
    
    # 创建 user_db 数据库
    docker exec mysql mysql -uroot -proot_password -e "CREATE DATABASE IF NOT EXISTS user_db;"
    
    # 运行迁移
    cd "${WORKSPACE_ROOT}/apps/im-service"
    if [ -f "migrations/001_initial_schema.sql" ]; then
        docker exec -i mysql mysql -uim_service -pim_service_password im_chat < migrations/001_initial_schema.sql
        log_success "数据库迁移完成"
    else
        log_warning "未找到迁移文件，跳过"
    fi
}

# 生成 Proto 代码
generate_proto() {
    log_info "生成 Proto 代码..."
    
    cd "${WORKSPACE_ROOT}"
    make proto-go
    
    log_success "Proto 代码生成完成"
}

# 构建服务
build_services() {
    log_info "构建服务..."
    
    local services=("auth-service" "user-service" "im-service" "im-gateway-service")
    
    for service in "${services[@]}"; do
        log_info "构建 ${service}..."
        cd "${WORKSPACE_ROOT}/apps/${service}"
        go build -o "bin/${service}" .
        log_success "✓ ${service} 构建完成"
    done
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    # 停止现有服务
    pkill -f "auth-service" 2>/dev/null || true
    pkill -f "user-service" 2>/dev/null || true
    pkill -f "im-service" 2>/dev/null || true
    pkill -f "im-gateway-service" 2>/dev/null || true
    
    sleep 2
    
    # 设置通用环境变量
    export APP_ENV=local
    
    # 启动 Auth Service
    cd "${WORKSPACE_ROOT}/apps/auth-service"
    APP_ENV=local \
    SERVER_PORT=9095 \
    OBSERVABILITY_METRICS_PORT=9190 \
    JWT_SECRET="local-dev-secret-change-me" \
    ./bin/auth-service > /tmp/auth-service.log 2>&1 &
    log_success "✓ Auth Service 已启动 (PID: $!)"
    
    # 启动 User Service
    cd "${WORKSPACE_ROOT}/apps/user-service"
    APP_ENV=local \
    SERVER_PORT=9096 \
    OBSERVABILITY_METRICS_PORT=9191 \
    DATABASE_HOST=localhost \
    DATABASE_PORT=3306 \
    DATABASE_USER=root \
    DATABASE_PASSWORD=root_password \
    DATABASE_DATABASE=user_db \
    ./bin/user-service > /tmp/user-service.log 2>&1 &
    log_success "✓ User Service 已启动 (PID: $!)"
    
    # 启动 IM Service
    cd "${WORKSPACE_ROOT}/apps/im-service"
    APP_ENV=local \
    SERVER_GRPC_PORT=9094 \
    SERVER_HTTP_PORT=8094 \
    OBSERVABILITY_METRICS_PORT=9192 \
    DATABASE_HOST=localhost \
    DATABASE_PORT=3306 \
    DATABASE_USER=im_service \
    DATABASE_PASSWORD=im_service_password \
    DATABASE_DATABASE=im_chat \
    REDIS_ADDR=localhost:6379 \
    ETCD_ENDPOINTS=localhost:2379 \
    ./bin/im-service > /tmp/im-service.log 2>&1 &
    log_success "✓ IM Service 已启动 (PID: $!)"
    
    # 启动 Gateway Service
    cd "${WORKSPACE_ROOT}/apps/im-gateway-service"
    APP_ENV=local \
    SERVER_PORT=9093 \
    OBSERVABILITY_METRICS_PORT=9193 \
    REDIS_ADDR=localhost:6379 \
    AUTH_SERVICE_ADDR=localhost:9095 \
    USER_SERVICE_ADDR=localhost:9096 \
    IM_SERVICE_ADDR=localhost:9094 \
    ./bin/im-gateway-service > /tmp/im-gateway-service.log 2>&1 &
    log_success "✓ Gateway Service 已启动 (PID: $!)"
    
    log_info "等待服务就绪..."
    sleep 10
    
    # 验证服务
    local ports=(9095 9096 9094 9093)
    local service_names=("Auth" "User" "IM" "Gateway")
    
    for i in "${!ports[@]}"; do
        if nc -z localhost "${ports[$i]}" 2>/dev/null; then
            log_success "✓ ${service_names[$i]} Service (端口 ${ports[$i]}) 运行中"
        else
            log_error "✗ ${service_names[$i]} Service (端口 ${ports[$i]}) 未运行"
            log_error "查看日志: tail -f /tmp/${service_names[$i],,}-service.log"
            exit 1
        fi
    done
}

# 运行 IM Service 集成测试
run_im_service_tests() {
    log_info "运行 IM Service 集成测试..."
    
    cd "${WORKSPACE_ROOT}/apps/im-service/integration_test"
    
    # 设置环境变量
    export IM_SERVICE_ADDR="localhost:9094"
    export MYSQL_ADDR="root:root_password@tcp(localhost:3306)/im_chat"
    export REDIS_ADDR="localhost:6379"
    export ETCD_ADDR="localhost:2379"
    # Use port 9093 for external connections from host machine
    export KAFKA_ADDR="localhost:9093"
    
    # 运行测试
    local test_output="${REPORT_DIR}/im-service-integration-${TIMESTAMP}.log"
    local coverage_file="${REPORT_DIR}/im-service-integration-${TIMESTAMP}.out"
    
    if go test -v -tags=integration -timeout=10m -coverprofile="${coverage_file}" 2>&1 | tee "${test_output}"; then
        log_success "✓ IM Service 集成测试通过"
        
        # 生成覆盖率报告
        go tool cover -html="${coverage_file}" -o "${REPORT_DIR}/im-service-integration-${TIMESTAMP}.html"
        local coverage=$(go tool cover -func="${coverage_file}" | grep total | awk '{print $3}')
        log_success "覆盖率: ${coverage}"
        
        return 0
    else
        log_error "✗ IM Service 集成测试失败"
        return 1
    fi
}

# 运行 Gateway Service 集成测试
run_gateway_service_tests() {
    log_info "运行 Gateway Service 集成测试..."
    
    cd "${WORKSPACE_ROOT}/apps/im-gateway-service/integration_test"
    
    # 设置环境变量
    export AUTH_SERVICE_ADDR="localhost:9095"
    export USER_SERVICE_ADDR="localhost:9096"
    export IM_SERVICE_ADDR="localhost:9094"
    
    # 运行测试
    local test_output="${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.log"
    local coverage_file="${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.out"
    
    if go test -v -tags=integration -timeout=10m -coverprofile="${coverage_file}" 2>&1 | tee "${test_output}"; then
        log_success "✓ Gateway Service 集成测试通过"
        
        # 生成覆盖率报告
        go tool cover -html="${coverage_file}" -o "${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.html"
        local coverage=$(go tool cover -func="${coverage_file}" | grep total | awk '{print $3}')
        log_success "覆盖率: ${coverage}"
        
        return 0
    else
        log_error "✗ Gateway Service 集成测试失败"
        return 1
    fi
}

# 运行基础设施测试
run_infrastructure_tests() {
    log_info "运行基础设施集成测试..."
    
    cd "${WORKSPACE_ROOT}/apps/im-service/integration_test"
    
    # 运行测试
    local test_output="${REPORT_DIR}/infrastructure-${TIMESTAMP}.log"
    
    if go test -v -tags=integration -run "Test.*Failover|Test.*Partition" -timeout=15m 2>&1 | tee "${test_output}"; then
        log_success "✓ 基础设施集成测试通过"
        return 0
    else
        log_error "✗ 基础设施集成测试失败"
        return 1
    fi
}

# 生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    local report_file="${REPORT_DIR}/integration-test-report-${TIMESTAMP}.md"
    
    cat > "${report_file}" << EOF
# IM Chat System - 集成测试报告

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')

## 测试环境

- Go 版本: $(go version)
- Docker 版本: $(docker --version)
- 操作系统: $(uname -s)

## 测试结果

### IM Service 集成测试
EOF
    
    if [ -f "${REPORT_DIR}/im-service-integration-${TIMESTAMP}.log" ]; then
        local im_passed=$(grep -c "PASS:" "${REPORT_DIR}/im-service-integration-${TIMESTAMP}.log" || echo "0")
        local im_failed=$(grep -c "FAIL:" "${REPORT_DIR}/im-service-integration-${TIMESTAMP}.log" || echo "0")
        local im_coverage=$(grep "total:" "${REPORT_DIR}/im-service-integration-${TIMESTAMP}.out" | awk '{print $3}' || echo "N/A")
        
        cat >> "${report_file}" << EOF
- 通过: ${im_passed}
- 失败: ${im_failed}
- 覆盖率: ${im_coverage}
- 详细日志: [im-service-integration-${TIMESTAMP}.log](./im-service-integration-${TIMESTAMP}.log)
- 覆盖率报告: [im-service-integration-${TIMESTAMP}.html](./im-service-integration-${TIMESTAMP}.html)

EOF
    fi
    
    cat >> "${report_file}" << EOF
### Gateway Service 集成测试
EOF
    
    if [ -f "${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.log" ]; then
        local gw_passed=$(grep -c "PASS:" "${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.log" || echo "0")
        local gw_failed=$(grep -c "FAIL:" "${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.log" || echo "0")
        local gw_coverage=$(grep "total:" "${REPORT_DIR}/gateway-service-integration-${TIMESTAMP}.out" | awk '{print $3}' || echo "N/A")
        
        cat >> "${report_file}" << EOF
- 通过: ${gw_passed}
- 失败: ${gw_failed}
- 覆盖率: ${gw_coverage}
- 详细日志: [gateway-service-integration-${TIMESTAMP}.log](./gateway-service-integration-${TIMESTAMP}.log)
- 覆盖率报告: [gateway-service-integration-${TIMESTAMP}.html](./gateway-service-integration-${TIMESTAMP}.html)

EOF
    fi
    
    cat >> "${report_file}" << EOF
### 基础设施集成测试
EOF
    
    if [ -f "${REPORT_DIR}/infrastructure-${TIMESTAMP}.log" ]; then
        local infra_passed=$(grep -c "PASS:" "${REPORT_DIR}/infrastructure-${TIMESTAMP}.log" || echo "0")
        local infra_failed=$(grep -c "FAIL:" "${REPORT_DIR}/infrastructure-${TIMESTAMP}.log" || echo "0")
        
        cat >> "${report_file}" << EOF
- 通过: ${infra_passed}
- 失败: ${infra_failed}
- 详细日志: [infrastructure-${TIMESTAMP}.log](./infrastructure-${TIMESTAMP}.log)

EOF
    fi
    
    cat >> "${report_file}" << EOF
## 服务日志

- Auth Service: /tmp/auth-service.log
- User Service: /tmp/user-service.log
- IM Service: /tmp/im-service.log
- Gateway Service: /tmp/im-gateway-service.log

## 基础设施状态

\`\`\`
$(${DOCKER_COMPOSE_CMD} -f "${WORKSPACE_ROOT}/deploy/docker/docker-compose.infra.yml" ps)
\`\`\`
EOF
    
    log_success "测试报告已生成: ${report_file}"
}

# 清理
cleanup() {
    log_info "清理环境..."
    
    # 停止服务
    pkill -f "auth-service" 2>/dev/null || true
    pkill -f "user-service" 2>/dev/null || true
    pkill -f "im-service" 2>/dev/null || true
    pkill -f "im-gateway-service" 2>/dev/null || true
    
    # 可选: 停止基础设施 (默认不停止，以便查看日志)
    if [ "${CLEANUP_INFRA}" = "true" ]; then
        cd "${WORKSPACE_ROOT}/deploy/docker"
        ${DOCKER_COMPOSE_CMD} -f docker-compose.infra.yml down -v
        log_success "基础设施已停止"
    else
        log_info "基础设施保持运行 (使用 CLEANUP_INFRA=true 来停止)"
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "IM Chat System - 集成测试"
    log_info "========================================="
    
    local start_time=$(date +%s)
    local test_failed=false
    
    # 检查前置条件
    check_prerequisites
    
    # 启动基础设施
    start_infrastructure
    
    # 初始化数据库
    init_database
    
    # 生成 Proto 代码
    generate_proto
    
    # 构建服务
    build_services
    
    # 启动服务
    start_services
    
    # 运行测试
    log_info "========================================="
    log_info "开始运行集成测试"
    log_info "========================================="
    
    if ! run_im_service_tests; then
        test_failed=true
    fi
    
    if ! run_gateway_service_tests; then
        test_failed=true
    fi
    
    if ! run_infrastructure_tests; then
        test_failed=true
    fi
    
    # 生成报告
    generate_report
    
    # 清理
    cleanup
    
    # 计算总时间
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_info "========================================="
    log_info "测试完成"
    log_info "总耗时: ${duration} 秒"
    log_info "报告目录: ${REPORT_DIR}"
    log_info "========================================="
    
    if [ "$test_failed" = true ]; then
        log_error "部分测试失败"
        exit 1
    else
        log_success "所有测试通过!"
        exit 0
    fi
}

# 捕获 Ctrl+C
trap cleanup EXIT

# 运行主函数
main "$@"
