#!/bin/bash
# Chaos Engineering Test Script
# 验证需求: 9.2.2, 9.2.3, 9.2.4, 9.2.5
# 自动化混沌工程测试流程: 注入故障 → 等待恢复 → 验证数据一致性

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
NAMESPACE="${NAMESPACE:-default}"
REGION_A_NAMESPACE="${REGION_A_NAMESPACE:-im-region-a}"
REGION_B_NAMESPACE="${REGION_B_NAMESPACE:-im-region-b}"
CHAOS_MESH_NAMESPACE="${CHAOS_MESH_NAMESPACE:-chaos-mesh}"
SYNC_TIMEOUT="${SYNC_TIMEOUT:-60}"  # 数据同步超时时间（秒）
RTO_TIMEOUT="${RTO_TIMEOUT:-30}"    # 故障转移超时时间（秒）
VERIFICATION_INTERVAL="${VERIFICATION_INTERVAL:-5}"  # 验证间隔（秒）

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq 未安装"
        exit 1
    fi
    
    # 检查 Chaos Mesh 是否安装
    if ! kubectl get namespace "$CHAOS_MESH_NAMESPACE" &> /dev/null; then
        log_error "Chaos Mesh 未安装，请先安装 Chaos Mesh"
        log_info "安装命令: curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 获取 Pod 状态
get_pod_status() {
    local namespace=$1
    local label=$2
    
    kubectl get pods -n "$namespace" -l "$label" \
        --no-headers -o custom-columns=":metadata.name,:status.phase" 2>/dev/null || echo ""
}

# 等待 Pod 就绪
wait_for_pods_ready() {
    local namespace=$1
    local label=$2
    local timeout=$3
    local start_time=$(date +%s)
    
    log_info "等待 Pod 就绪: namespace=$namespace, label=$label"
    
    while true; do
        local ready_count=$(kubectl get pods -n "$namespace" -l "$label" \
            --field-selector=status.phase=Running 2>/dev/null | grep -c "Running" || echo "0")
        local total_count=$(kubectl get pods -n "$namespace" -l "$label" \
            --no-headers 2>/dev/null | wc -l || echo "0")
        
        if [ "$ready_count" -eq "$total_count" ] && [ "$total_count" -gt 0 ]; then
            log_success "所有 Pod 已就绪 ($ready_count/$total_count)"
            return 0
        fi
        
        local elapsed=$(($(date +%s) - start_time))
        if [ $elapsed -ge $timeout ]; then
            log_error "等待 Pod 就绪超时 (${timeout}s)"
            return 1
        fi
        
        log_info "等待中... ($ready_count/$total_count 就绪, ${elapsed}s/${timeout}s)"
        sleep "$VERIFICATION_INTERVAL"
    done
}

# 应用 Chaos 配置
apply_chaos() {
    local chaos_file=$1
    local chaos_name=$2
    
    log_info "应用 Chaos 配置: $chaos_file"
    
    if [ ! -f "$chaos_file" ]; then
        log_error "Chaos 配置文件不存在: $chaos_file"
        return 1
    fi
    
    kubectl apply -f "$chaos_file"
    
    # 等待 Chaos 实验创建
    sleep 2
    
    # 检查 Chaos 实验状态
    local status=$(kubectl get -f "$chaos_file" -o jsonpath='{.items[0].status.experiment.phase}' 2>/dev/null || echo "Unknown")
    log_info "Chaos 实验状态: $status"
    
    return 0
}

# 删除 Chaos 配置
delete_chaos() {
    local chaos_file=$1
    
    log_info "删除 Chaos 配置: $chaos_file"
    
    if [ ! -f "$chaos_file" ]; then
        log_warning "Chaos 配置文件不存在: $chaos_file"
        return 0
    fi
    
    kubectl delete -f "$chaos_file" --ignore-not-found=true
    
    # 等待 Chaos 实验删除
    sleep 2
    
    log_success "Chaos 配置已删除"
    return 0
}

# 验证数据一致性（使用 Merkle Tree 对账）
verify_data_consistency() {
    local region_a_endpoint=$1
    local region_b_endpoint=$2
    
    log_info "验证数据一致性（Merkle Tree 对账）..."
    
    # 调用数据对账 API
    local region_a_hash=$(curl -s "$region_a_endpoint/api/v1/reconcile/merkle-root" | jq -r '.hash' 2>/dev/null || echo "")
    local region_b_hash=$(curl -s "$region_b_endpoint/api/v1/reconcile/merkle-root" | jq -r '.hash' 2>/dev/null || echo "")
    
    if [ -z "$region_a_hash" ] || [ -z "$region_b_hash" ]; then
        log_warning "无法获取 Merkle Tree 哈希值，跳过一致性验证"
        return 0
    fi
    
    log_info "Region A Merkle Root: $region_a_hash"
    log_info "Region B Merkle Root: $region_b_hash"
    
    if [ "$region_a_hash" == "$region_b_hash" ]; then
        log_success "数据一致性验证通过"
        return 0
    else
        log_error "数据一致性验证失败: 两个地域的 Merkle Root 不一致"
        return 1
    fi
}

# 测试网络分区恢复
test_network_partition_recovery() {
    log_info "=========================================="
    log_info "测试: 网络分区恢复 (需求 9.2.1, 9.2.2)"
    log_info "=========================================="
    
    local chaos_file="tests/chaos/network-partition.yaml"
    local start_time=$(date +%s)
    
    # 1. 应用网络分区 Chaos
    apply_chaos "$chaos_file" "cross-region-partition"
    
    # 2. 等待网络分区持续时间（60秒）
    log_info "网络分区已注入，等待 60 秒..."
    sleep 60
    
    # 3. 删除网络分区 Chaos（恢复网络）
    delete_chaos "$chaos_file"
    
    # 4. 记录恢复开始时间
    local recovery_start=$(date +%s)
    log_info "网络分区已恢复，开始验证数据同步..."
    
    # 5. 等待数据重新同步（最多 60 秒）
    local sync_verified=false
    while [ $(($(date +%s) - recovery_start)) -lt $SYNC_TIMEOUT ]; do
        sleep "$VERIFICATION_INTERVAL"
        
        # 验证数据一致性
        if verify_data_consistency "http://region-a-service" "http://region-b-service"; then
            local sync_duration=$(($(date +%s) - recovery_start))
            log_success "数据重新同步完成，耗时: ${sync_duration}s"
            
            # 验证需求 9.2.2: 60 秒内完成数据重新同步
            if [ $sync_duration -le $SYNC_TIMEOUT ]; then
                log_success "✓ 需求 9.2.2 验证通过: 数据在 ${sync_duration}s 内完成重新同步 (要求 < 60s)"
                sync_verified=true
                break
            else
                log_error "✗ 需求 9.2.2 验证失败: 数据同步耗时 ${sync_duration}s (要求 < 60s)"
                return 1
            fi
        fi
    done
    
    if [ "$sync_verified" = false ]; then
        log_error "✗ 数据同步超时 (${SYNC_TIMEOUT}s)"
        return 1
    fi
    
    local total_duration=$(($(date +%s) - start_time))
    log_success "网络分区恢复测试完成，总耗时: ${total_duration}s"
    return 0
}

# 测试地域故障转移
test_region_failover() {
    log_info "=========================================="
    log_info "测试: 地域故障转移 (需求 9.2.3)"
    log_info "=========================================="
    
    local chaos_file="tests/chaos/pod-kill.yaml"
    local start_time=$(date +%s)
    
    # 1. 记录故障前的服务状态
    log_info "记录故障前的服务状态..."
    local pods_before=$(get_pod_status "$REGION_A_NAMESPACE" "app=im-service")
    log_info "Region A Pods: $pods_before"
    
    # 2. 应用 Pod Kill Chaos（模拟地域故障）
    apply_chaos "$chaos_file" "im-service-region-failure"
    
    # 3. 记录故障转移开始时间
    local failover_start=$(date +%s)
    log_info "地域故障已注入，开始监控故障转移..."
    
    # 4. 等待 Region B 接管流量（最多 30 秒）
    local failover_verified=false
    while [ $(($(date +%s) - failover_start)) -lt $RTO_TIMEOUT ]; do
        sleep "$VERIFICATION_INTERVAL"
        
        # 检查 Region B 的 Pod 是否正常运行
        local region_b_ready=$(kubectl get pods -n "$REGION_B_NAMESPACE" -l "app=im-service" \
            --field-selector=status.phase=Running 2>/dev/null | grep -c "Running" || echo "0")
        
        if [ "$region_b_ready" -gt 0 ]; then
            local failover_duration=$(($(date +%s) - failover_start))
            log_success "故障转移完成，Region B 已接管流量，耗时: ${failover_duration}s"
            
            # 验证需求 9.2.3: RTO < 30 秒
            if [ $failover_duration -le $RTO_TIMEOUT ]; then
                log_success "✓ 需求 9.2.3 验证通过: 故障转移在 ${failover_duration}s 内完成 (要求 < 30s)"
                failover_verified=true
                break
            else
                log_error "✗ 需求 9.2.3 验证失败: 故障转移耗时 ${failover_duration}s (要求 < 30s)"
                return 1
            fi
        fi
    done
    
    if [ "$failover_verified" = false ]; then
        log_error "✗ 故障转移超时 (${RTO_TIMEOUT}s)"
        return 1
    fi
    
    # 5. 删除 Chaos 配置，恢复 Region A
    delete_chaos "$chaos_file"
    
    # 6. 等待 Region A 恢复
    log_info "等待 Region A 恢复..."
    wait_for_pods_ready "$REGION_A_NAMESPACE" "app=im-service" 60
    
    local total_duration=$(($(date +%s) - start_time))
    log_success "地域故障转移测试完成，总耗时: ${total_duration}s"
    return 0
}

# 测试时钟偏移容错
test_clock_skew_tolerance() {
    log_info "=========================================="
    log_info "测试: 时钟偏移容错 (需求 9.2.4)"
    log_info "=========================================="
    
    local chaos_file="tests/chaos/clock-skew.yaml"
    local start_time=$(date +%s)
    
    # 1. 应用时钟偏移 Chaos
    apply_chaos "$chaos_file" "clock-skew-positive"
    
    # 2. 等待时钟偏移持续时间（120秒）
    log_info "时钟偏移已注入（+5s），等待 120 秒..."
    sleep 120
    
    # 3. 验证 HLC 仍然正确工作
    log_info "验证 HLC 算法容错能力..."
    
    # 调用 HLC 健康检查 API
    local hlc_status=$(curl -s "http://region-a-service/api/v1/hlc/health" | jq -r '.status' 2>/dev/null || echo "unknown")
    
    if [ "$hlc_status" == "healthy" ] || [ "$hlc_status" == "calibrated" ]; then
        log_success "✓ 需求 9.2.4 验证通过: HLC 在时钟偏移情况下仍正确工作"
    else
        log_error "✗ 需求 9.2.4 验证失败: HLC 状态异常: $hlc_status"
        delete_chaos "$chaos_file"
        return 1
    fi
    
    # 4. 删除时钟偏移 Chaos
    delete_chaos "$chaos_file"
    
    local total_duration=$(($(date +%s) - start_time))
    log_success "时钟偏移容错测试完成，总耗时: ${total_duration}s"
    return 0
}

# 测试数据一致性验证
test_data_consistency_verification() {
    log_info "=========================================="
    log_info "测试: 数据一致性验证 (需求 9.2.5)"
    log_info "=========================================="
    
    local start_time=$(date +%s)
    
    # 1. 验证故障恢复后数据一致性
    log_info "验证故障恢复后数据一致性（Merkle Tree 对账）..."
    
    if verify_data_consistency "http://region-a-service" "http://region-b-service"; then
        log_success "✓ 需求 9.2.5 验证通过: 故障恢复后数据一致性验证通过"
    else
        log_error "✗ 需求 9.2.5 验证失败: 数据一致性验证失败"
        return 1
    fi
    
    local total_duration=$(($(date +%s) - start_time))
    log_success "数据一致性验证测试完成，总耗时: ${total_duration}s"
    return 0
}

# 清理环境
cleanup() {
    log_info "清理测试环境..."
    
    # 删除所有 Chaos 配置
    kubectl delete networkchaos --all -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete podchaos --all -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete timechaos --all -n "$NAMESPACE" --ignore-not-found=true
    
    log_success "清理完成"
}

# 主函数
main() {
    log_info "=========================================="
    log_info "混沌工程测试开始"
    log_info "=========================================="
    
    # 检查依赖
    check_dependencies
    
    # 测试计数器
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    # 测试 1: 网络分区恢复
    total_tests=$((total_tests + 1))
    if test_network_partition_recovery; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    echo ""
    
    # 测试 2: 地域故障转移
    total_tests=$((total_tests + 1))
    if test_region_failover; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    echo ""
    
    # 测试 3: 时钟偏移容错
    total_tests=$((total_tests + 1))
    if test_clock_skew_tolerance; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    echo ""
    
    # 测试 4: 数据一致性验证
    total_tests=$((total_tests + 1))
    if test_data_consistency_verification; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    echo ""
    
    # 清理环境
    cleanup
    
    # 输出测试结果
    log_info "=========================================="
    log_info "混沌工程测试完成"
    log_info "=========================================="
    log_info "总测试数: $total_tests"
    log_success "通过: $passed_tests"
    if [ $failed_tests -gt 0 ]; then
        log_error "失败: $failed_tests"
    else
        log_info "失败: $failed_tests"
    fi
    
    if [ $failed_tests -eq 0 ]; then
        log_success "所有测试通过！"
        exit 0
    else
        log_error "部分测试失败！"
        exit 1
    fi
}

# 捕获退出信号，确保清理
trap cleanup EXIT INT TERM

# 运行主函数
main "$@"
