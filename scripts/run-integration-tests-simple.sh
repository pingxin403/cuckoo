#!/bin/bash

# IM Chat System - 简化版集成测试脚本
# 不依赖 Docker Compose，直接使用 Docker 命令

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    log_success "Docker 版本: $(docker --version)"
}

# 启动基础设施（使用 Docker 命令）
start_infrastructure() {
    log_info "启动基础设施服务..."
    
    # 创建网络
    docker network create im-network 2>/dev/null || true
    
    # 启动 etcd
    log_info "启动 etcd..."
    docker run -d --name etcd-1 --network im-network \
        -p 2379:2379 -p 2380:2380 \
        -e ETCD_NAME=etcd-1 \
        -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
        -e ETCD_ADVERTISE_CLIENT_URLS=http://etcd-1:2379 \
        -e ETCD_LISTEN_PEER_URLS=http://0.0.0.0:2380 \
        -e ETCD_INITIAL_ADVERTISE_PEER_URLS=http://etcd-1:2380 \
        -e ETCD_INITIAL_CLUSTER=etcd-1=http://etcd-1:2380 \
        -e ETCD_INITIAL_CLUSTER_STATE=new \
        quay.io/coreos/etcd:v3.5.0 2>/dev/null || docker start etcd-1
    
    # 启动 MySQL
    log_info "启动 MySQL..."
    docker run -d --name mysql --network im-network \
        -p 3306:3306 \
        -e MYSQL_ROOT_PASSWORD=password \
        -e MYSQL_DATABASE=im_chat \
        mysql:8.0 2>/dev/null || docker start mysql
    
    # 启动 Redis
    log_info "启动 Redis..."
    docker run -d --name redis --network im-network \
        -p 6379:6379 \
        redis:7-alpine 2>/dev/null || docker start redis
    
    # 启动 Kafka (需要 Zookeeper)
    log_info "启动 Zookeeper..."
    docker run -d --name zookeeper --network im-network \
        -p 2181:2181 \
        -e ZOOKEEPER_CLIENT_PORT=2181 \
        confluentinc/cp-zookeeper:7.5.0 2>/dev/null || docker start zookeeper
    
    log_info "启动 Kafka..."
    docker run -d --name kafka-1 --network im-network \
        -p 9092:9092 \
        -e KAFKA_BROKER_ID=1 \
        -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
        -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
        -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
        confluentinc/cp-kafka:7.5.0 2>/dev/null || docker start kafka-1
    
    log_info "等待服务就绪..."
    sleep 30
    
    # 验证服务
    log_info "验证服务状态..."
    
    if docker exec etcd-1 etcdctl endpoint health &> /dev/null; then
        log_success "✓ etcd 健康"
    else
        log_warning "✗ etcd 可能未就绪，继续等待..."
        sleep 10
    fi
    
    if docker exec mysql mysqladmin ping -h localhost -uroot -ppassword &> /dev/null; then
        log_success "✓ MySQL 健康"
    else
        log_warning "✗ MySQL 可能未就绪，继续等待..."
        sleep 10
    fi
    
    if docker exec redis redis-cli ping &> /dev/null; then
        log_success "✓ Redis 健康"
    else
        log_error "✗ Redis 不健康"
    fi
    
    log_success "基础设施启动完成"
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    
    # 等待 MySQL 完全就绪
    for i in {1..30}; do
        if docker exec mysql mysqladmin ping -h localhost -uroot -ppassword &> /dev/null; then
            break
        fi
        log_info "等待 MySQL 就绪... ($i/30)"
        sleep 2
    done
    
    # 创建用户和授权
    docker exec mysql mysql -uroot -ppassword -e "
        CREATE USER IF NOT EXISTS 'im_service'@'%' IDENTIFIED BY 'password';
        GRANT ALL PRIVILEGES ON im_chat.* TO 'im_service'@'%';
        FLUSH PRIVILEGES;
    " 2>/dev/null || true
    
    log_success "数据库初始化完成"
}

# 运行测试
run_tests() {
    log_info "运行集成测试..."
    
    cd "${WORKSPACE_ROOT}/apps/im-service/integration_test"
    
    # 设置环境变量
    export IM_SERVICE_ADDR="localhost:9094"
    export MYSQL_ADDR="root:password@tcp(localhost:3306)/im_chat"
    export REDIS_ADDR="localhost:6379"
    export ETCD_ADDR="localhost:2379"
    export KAFKA_ADDR="localhost:9092"
    
    # 运行测试
    if go test -v -tags=integration -timeout=10m; then
        log_success "✓ 集成测试通过"
        return 0
    else
        log_error "✗ 集成测试失败"
        return 1
    fi
}

# 清理
cleanup() {
    log_info "清理环境..."
    
    if [ "${CLEANUP_INFRA}" = "true" ]; then
        log_info "停止并删除容器..."
        docker stop etcd-1 mysql redis zookeeper kafka-1 2>/dev/null || true
        docker rm etcd-1 mysql redis zookeeper kafka-1 2>/dev/null || true
        docker network rm im-network 2>/dev/null || true
        log_success "清理完成"
    else
        log_info "基础设施保持运行 (使用 CLEANUP_INFRA=true 来清理)"
        log_info "手动清理命令:"
        log_info "  docker stop etcd-1 mysql redis zookeeper kafka-1"
        log_info "  docker rm etcd-1 mysql redis zookeeper kafka-1"
        log_info "  docker network rm im-network"
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "IM Chat System - 简化版集成测试"
    log_info "========================================="
    
    check_docker
    start_infrastructure
    init_database
    
    log_info "========================================="
    log_info "基础设施已就绪，请手动启动服务并运行测试"
    log_info "========================================="
    
    log_info "下一步操作:"
    log_info ""
    log_info "1. 生成 Proto 代码:"
    log_info "   cd ${WORKSPACE_ROOT}"
    log_info "   make proto-go"
    log_info ""
    log_info "2. 启动服务 (在不同终端中):"
    log_info "   cd ${WORKSPACE_ROOT}/apps/im-service"
    log_info "   go run main.go"
    log_info ""
    log_info "3. 运行测试:"
    log_info "   cd ${WORKSPACE_ROOT}/apps/im-service/integration_test"
    log_info "   go test -v -tags=integration"
    log_info ""
    log_info "或者运行完整测试 (如果服务已启动):"
    if run_tests; then
        log_success "测试通过!"
    else
        log_error "测试失败"
    fi
    
    cleanup
}

# 捕获 Ctrl+C
trap cleanup EXIT

# 运行主函数
main "$@"
