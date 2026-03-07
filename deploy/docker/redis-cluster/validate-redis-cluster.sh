#!/bin/bash
# Redis Cluster 验收脚本
# 用于验证三主三从部署的正确性和高可用能力

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 计数器
PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

# 打印函数
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASS_COUNT++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAIL_COUNT++))
}

print_warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
    ((WARN_COUNT++))
}

print_info() {
    echo -e "${BLUE}ℹ INFO${NC}: $1"
}

# 检查 Docker 容器状态
check_container_status() {
    print_header "1. 检查容器运行状态"
    
    local containers=("redis-node-1" "redis-node-2" "redis-node-3" "redis-node-4" "redis-node-5" "redis-node-6")
    local all_running=true
    
    for container in "${containers[@]}"; do
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            print_pass "$container 运行中"
        else
            print_fail "$container 未运行"
            all_running=false
        fi
    done
    
    if [ "$all_running" = true ]; then
        return 0
    else
        return 1
    fi
}

# 检查容器健康状态
check_container_health() {
    print_header "2. 检查容器健康状态"
    
    local containers=("redis-node-1" "redis-node-2" "redis-node-3" "redis-node-4" "redis-node-5" "redis-node-6")
    local all_healthy=true
    
    for container in "${containers[@]}"; do
        local health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "no-healthcheck")
        
        case "$health" in
            "healthy")
                print_pass "$container 健康状态：$health"
                ;;
            "starting")
                print_warn "$container 健康状态：$health (正在启动)"
                ;;
            "unhealthy")
                print_fail "$container 健康状态：$health"
                all_healthy=false
                ;;
            *)
                print_info "$container 健康状态：$health"
                ;;
        esac
    done
    
    if [ "$all_healthy" = true ]; then
        return 0
    else
        return 1
    fi
}

# 检查 Cluster 基础信息
check_cluster_info() {
    print_header "3. 检查 Cluster 基础信息"
    
    local cluster_info=$(docker exec redis-node-1 redis-cli -p 6379 cluster info 2>/dev/null)
    
    # 检查 cluster_state
    local cluster_state=$(echo "$cluster_info" | grep "cluster_state" | cut -d: -f2 | tr -d '\r')
    if [ "$cluster_state" = "ok" ]; then
        print_pass "Cluster 状态：$cluster_state"
    else
        print_fail "Cluster 状态：$cluster_state (期望：ok)"
    fi
    
    # 检查 cluster_slots_assigned
    local slots_assigned=$(echo "$cluster_info" | grep "cluster_slots_assigned" | cut -d: -f2 | tr -d '\r')
    if [ "$slots_assigned" = "16384" ]; then
        print_pass "Slot 分配：$slots_assigned/16384"
    else
        print_fail "Slot 分配：$slots_assigned/16384"
    fi
    
    # 检查 cluster_slots_ok
    local slots_ok=$(echo "$cluster_info" | grep "cluster_slots_ok" | cut -d: -f2 | tr -d '\r')
    if [ "$slots_ok" = "16384" ]; then
        print_pass "可用 Slot：$slots_ok/16384"
    else
        print_fail "可用 Slot：$slots_ok/16384"
    fi
    
    # 检查 cluster_known_nodes
    local known_nodes=$(echo "$cluster_info" | grep "cluster_known_nodes" | cut -d: -f2 | tr -d '\r')
    if [ "$known_nodes" = "6" ]; then
        print_pass "已知节点数：$known_nodes/6"
    else
        print_fail "已知节点数：$known_nodes/6 (期望：6)"
    fi
    
    # 检查 cluster_size (master 数量)
    local cluster_size=$(echo "$cluster_info" | grep "cluster_size" | cut -d: -f2 | tr -d '\r')
    if [ "$cluster_size" = "3" ]; then
        print_pass "Master 节点数：$cluster_size/3"
    else
        print_fail "Master 节点数：$cluster_size/3 (期望：3)"
    fi
}

# 检查节点角色
check_node_roles() {
    print_header "4. 检查节点角色分配"
    
    local cluster_nodes=$(docker exec redis-node-1 redis-cli -p 6379 cluster nodes 2>/dev/null)
    
    # 统计 master 和 slave 数量
    local master_count=$(echo "$cluster_nodes" | grep -c "master" || echo 0)
    local slave_count=$(echo "$cluster_nodes" | grep -c "slave" || echo 0)
    
    print_info "节点角色统计:"
    echo "$cluster_nodes" | awk '{print "  " $1 " - " $3}'
    
    if [ "$master_count" -eq 3 ]; then
        print_pass "Master 节点数量：$master_count"
    else
        print_fail "Master 节点数量：$master_count (期望：3)"
    fi
    
    if [ "$slave_count" -eq 3 ]; then
        print_pass "Slave 节点数量：$slave_count"
    else
        print_fail "Slave 节点数量：$slave_count (期望：3)"
    fi
    
    # 检查 slave 是否正确跟随 master
    print_info "Slave 跟随关系:"
    echo "$cluster_nodes" | grep "slave" | while read -r line; do
        local slave_id=$(echo "$line" | awk '{print $1}')
        local master_id=$(echo "$line" | awk '{print $4}' | tr -d '-')
        print_info "  Slave $slave_id 跟随 Master $master_id"
    done
}

# 检查 Slot 分配
check_slot_distribution() {
    print_header "5. 检查 Slot 分配情况"
    
    local cluster_slots=$(docker exec redis-node-1 redis-cli -p 6379 cluster slots 2>/dev/null)
    
    # 统计每个 master 的 slot 数量
    local slot_ranges=$(echo "$cluster_slots" | grep -c "^" || echo 0)
    print_info "Slot 范围数量：$slot_ranges"
    
    # 打印每个 slot 范围
    echo "$cluster_slots" | while IFS= read -r line; do
        if [ -n "$line" ]; then
            local start=$(echo "$line" | awk '{print $1}')
            local end=$(echo "$line" | awk '{print $2}')
            local master_ip=$(echo "$line" | awk '{print $4}' | cut -d'@' -f2)
            if [[ "$start" =~ ^[0-9]+$ ]] && [[ "$end" =~ ^[0-9]+$ ]]; then
                local count=$((end - start + 1))
                print_info "  Slot $start-$end ($count 个) -> $master_ip"
            fi
        fi
    done
    
    # 验证总 slot 数 - 使用 cluster info 而不是解析 slots 输出
    local cluster_info=$(docker exec redis-node-1 redis-cli -p 6379 cluster info 2>/dev/null)
    local slots_assigned=$(echo "$cluster_info" | grep "cluster_slots_assigned" | cut -d: -f2 | tr -d '\r')
    
    if [ "$slots_assigned" = "16384" ]; then
        print_pass "总 Slot 数：$slots_assigned/16384"
    else
        print_fail "总 Slot 数：$slots_assigned/16384"
    fi
}

# 检查读写功能
check_read_write() {
    print_header "6. 检查读写功能"
    
    local test_key="test:$(date +%s)"
    local test_value="redis_cluster_test_value"
    
    # 测试写入
    print_info "测试写入：SET $test_key $test_value"
    local set_result=$(docker exec redis-node-1 redis-cli -p 6379 -c set "$test_key" "$test_value" 2>/dev/null)
    
    if [ "$set_result" = "OK" ]; then
        print_pass "写入成功"
    else
        print_fail "写入失败：$set_result"
        return 1
    fi
    
    # 测试读取
    print_info "测试读取：GET $test_key"
    local get_result=$(docker exec redis-node-1 redis-cli -p 6379 -c get "$test_key" 2>/dev/null)
    
    if [ "$get_result" = "$test_value" ]; then
        print_pass "读取成功：$get_result"
    else
        print_fail "读取失败：$get_result (期望：$test_value)"
        return 1
    fi
    
    # 测试集群命令路由（使用 cluster keyslot 验证）
    print_info "验证集群 Key 路由..."
    local keyslot=$(docker exec redis-node-1 redis-cli -p 6379 cluster keyslot "$test_key" 2>/dev/null)
    if [[ "$keyslot" =~ ^[0-9]+$ ]]; then
        print_pass "集群 Key Slot 查询成功：$keyslot"
    else
        print_fail "集群 Key Slot 查询失败"
        return 1
    fi
    
    # 测试从不同节点执行集群命令
    print_info "测试从不同节点执行集群命令..."
    local node2_info=$(docker exec redis-node-2 redis-cli -p 6379 cluster info 2>/dev/null | grep "cluster_state" | cut -d: -f2 | tr -d '\r')
    if [ "$node2_info" = "ok" ]; then
        print_pass "从 node-2 查询集群状态成功"
    else
        print_fail "从 node-2 查询集群状态失败"
        return 1
    fi
    
    # 清理测试数据
    docker exec redis-node-1 redis-cli -p 6379 -c del "$test_key" > /dev/null 2>&1
    print_info "已清理测试数据"
}

# 检查网络连通性
check_network_connectivity() {
    print_header "7. 检查网络连通性"
    
    local ports=(6379)
    local all_connected=true
    
    for port in "${ports[@]}"; do
        for i in {1..6}; do
            local container="redis-node-$i"
            local result=$(docker exec "$container" redis-cli -p "$port" ping 2>/dev/null)
            
            if [ "$result" = "PONG" ]; then
                print_pass "$container:$port 响应正常"
            else
                print_fail "$container:$port 响应异常：$result"
                all_connected=false
            fi
        done
    done
    
    if [ "$all_connected" = true ]; then
        return 0
    else
        return 1
    fi
}

# 高可用测试（可选，需要用户确认）
check_high_availability() {
    print_header "8. 高可用测试（故障转移）"
    
    read -p "是否执行故障转移测试？(这将停止 redis-node-1 约 15 秒) [y/N]: " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "跳过故障转移测试"
        return 0
    fi
    
    print_info "停止 redis-node-1 (Master 节点)..."
    docker stop redis-node-1
    
    print_info "等待故障转移完成 (15 秒)..."
    sleep 15
    
    # 检查集群状态
    local cluster_state=$(docker exec redis-node-2 redis-cli -p 6379 cluster info 2>/dev/null | grep "cluster_state" | cut -d: -f2 | tr -d '\r')
    
    if [ "$cluster_state" = "ok" ]; then
        print_pass "故障转移后集群状态：$cluster_state"
    else
        print_fail "故障转移后集群状态：$cluster_state"
    fi
    
    # 测试读写
    local test_key="ha_test:$(date +%s)"
    local ha_write=$(docker exec redis-node-2 redis-cli -p 6379 -c set "$test_key" "ha_test_value" 2>/dev/null)
    
    if [ "$ha_write" = "OK" ]; then
        print_pass "故障转移后写入成功"
    else
        print_fail "故障转移后写入失败"
    fi
    
    # 恢复节点
    print_info "恢复 redis-node-1..."
    docker start redis-node-1
    
    print_info "等待节点重新加入集群 (10 秒)..."
    sleep 10
    
    # 验证节点重新加入
    local known_nodes=$(docker exec redis-node-2 redis-cli -p 6379 cluster info 2>/dev/null | grep "cluster_known_nodes" | cut -d: -f2 | tr -d '\r')
    
    if [ "$known_nodes" = "6" ]; then
        print_pass "节点重新加入集群：$known_nodes/6"
    else
        print_warn "节点重新加入集群：$known_nodes/6 (期望：6)"
    fi
    
    # 清理测试数据
    docker exec redis-node-2 redis-cli -p 6379 -c del "$test_key" > /dev/null 2>&1
}

# 检查持久化配置
check_persistence() {
    print_header "9. 检查持久化配置"
    
    # 检查 AOF 文件/目录（Redis 7+ 使用 appendonlydir）
    for i in {1..6}; do
        local container="redis-node-$i"
        # Redis 7+ 使用 appendonlydir 目录，早期版本使用 appendonly.aof 文件
        local aof_exists=$(docker exec "$container" sh -c "ls -d /data/appendonlydir 2>/dev/null || ls /data/appendonly.aof 2>/dev/null || echo ''")
        
        if [ -n "$aof_exists" ]; then
            print_pass "$container AOF 持久化已启用"
        else
            print_warn "$container AOF 文件不存在 (可能是新启动的节点)"
        fi
    done
    
    # 测试数据持久化
    print_info "测试数据持久化..."
    local persist_key="persist:$(date +%s)"
    docker exec redis-node-1 redis-cli -p 6379 -c set "$persist_key" "persist_value" > /dev/null 2>&1
    
    # 强制保存 RDB 快照
    docker exec redis-node-1 redis-cli -p 6379 bgsave > /dev/null 2>&1
    sleep 2
    
    # 强制重写 AOF
    docker exec redis-node-1 redis-cli -p 6379 bgrewriteaof > /dev/null 2>&1
    sleep 2
    
    # 验证写入的数据
    local verify_before=$(docker exec redis-node-1 redis-cli -p 6379 -c get "$persist_key" 2>/dev/null)
    if [ "$verify_before" = "persist_value" ]; then
        print_pass "写入数据验证成功：$verify_before"
    else
        print_fail "写入数据验证失败：$verify_before"
    fi
    
    # 检查持久化文件
    print_info "检查持久化文件..."
    local rdb_size=$(docker exec redis-node-1 ls -la /data/dump.rdb 2>/dev/null | awk '{print $5}')
    if [ -n "$rdb_size" ] && [ "$rdb_size" -gt 0 ]; then
        print_pass "RDB 文件存在 (大小：$rdb_size 字节)"
    else
        print_warn "RDB 文件不存在或为空"
    fi
    
    local aof_dir_size=$(docker exec redis-node-1 du -s /data/appendonlydir 2>/dev/null | awk '{print $1}')
    if [ -n "$aof_dir_size" ] && [ "$aof_dir_size" -gt 0 ]; then
        print_pass "AOF 目录存在 (大小：${aof_dir_size}KB)"
    else
        print_warn "AOF 目录不存在或为空"
    fi
    
    # 清理测试数据
    docker exec redis-node-1 redis-cli -p 6379 -c del "$persist_key" > /dev/null 2>&1
    print_info "已清理测试数据"
    print_info "注意：跳过重启测试（Cluster 模式重启可能导致节点配置丢失）"
}

# 生成验收报告
generate_report() {
    print_header "验收报告"
    
    local total=$((PASS_COUNT + FAIL_COUNT + WARN_COUNT))
    
    echo ""
    echo "测试结果汇总:"
    echo -e "  ${GREEN}通过${NC}: $PASS_COUNT"
    echo -e "  ${RED}失败${NC}: $FAIL_COUNT"
    echo -e "  ${YELLOW}警告${NC}: $WARN_COUNT"
    echo "  总计：$total"
    echo ""
    
    if [ "$FAIL_COUNT" -eq 0 ]; then
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}  验收通过！Redis Cluster 部署成功！${NC}"
        echo -e "${GREEN}========================================${NC}"
        return 0
    else
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}  验收失败！请检查上述错误信息。${NC}"
        echo -e "${RED}========================================${NC}"
        return 1
    fi
}

# 主函数
main() {
    print_header "Redis Cluster 三主三从验收测试"
    echo "开始时间：$(date)"
    echo ""
    
    # 检查是否在项目根目录
    if [ ! -f "deploy/docker/redis-cluster/docker-compose.yml" ]; then
        print_fail "请在项目根目录执行此脚本"
        exit 1
    fi
    
    # 执行各项检查
    check_container_status || true
    check_container_health || true
    check_cluster_info || true
    check_node_roles || true
    check_slot_distribution || true
    check_read_write || true
    check_network_connectivity || true
    check_high_availability || true
    check_persistence || true
    
    # 生成报告
    generate_report
    
    echo ""
    echo "结束时间：$(date)"
}

# 执行主函数
main "$@"
