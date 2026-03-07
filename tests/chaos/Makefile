# Makefile for Chaos Engineering Tests
# 混沌工程测试 Makefile

.PHONY: help install-chaos-mesh run-all-tests test-network test-failover test-clock test-consistency clean apply-network apply-clock apply-pod delete-all

# 默认目标
help:
	@echo "Chaos Engineering Tests - 混沌工程测试"
	@echo ""
	@echo "可用命令:"
	@echo "  make install-chaos-mesh  - 安装 Chaos Mesh"
	@echo "  make run-all-tests       - 运行所有混沌工程测试"
	@echo "  make test-network        - 运行网络分区恢复测试"
	@echo "  make test-failover       - 运行地域故障转移测试"
	@echo "  make test-clock          - 运行时钟偏移容错测试"
	@echo "  make test-consistency    - 运行数据一致性验证测试"
	@echo "  make apply-network       - 应用网络故障配置"
	@echo "  make apply-clock         - 应用时钟偏移配置"
	@echo "  make apply-pod           - 应用 Pod 故障配置"
	@echo "  make delete-all          - 删除所有 Chaos 配置"
	@echo "  make clean               - 清理测试环境"
	@echo ""
	@echo "环境变量:"
	@echo "  NAMESPACE              - Chaos 实验命名空间 (默认: default)"
	@echo "  REGION_A_NAMESPACE     - Region A 命名空间 (默认: im-region-a)"
	@echo "  REGION_B_NAMESPACE     - Region B 命名空间 (默认: im-region-b)"
	@echo "  SYNC_TIMEOUT           - 数据同步超时时间/秒 (默认: 60)"
	@echo "  RTO_TIMEOUT            - 故障转移超时时间/秒 (默认: 30)"

# 安装 Chaos Mesh
install-chaos-mesh:
	@echo "安装 Chaos Mesh..."
	@curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash
	@echo "等待 Chaos Mesh 就绪..."
	@kubectl wait --for=condition=Ready pods --all -n chaos-mesh --timeout=300s
	@echo "Chaos Mesh 安装完成"

# 运行所有测试
run-all-tests:
	@echo "运行所有混沌工程测试..."
	@./run-chaos-tests.sh

# 运行网络分区恢复测试
test-network:
	@echo "运行网络分区恢复测试..."
	@kubectl apply -f network-partition.yaml
	@echo "网络分区已注入，等待 60 秒..."
	@sleep 60
	@kubectl delete -f network-partition.yaml
	@echo "网络分区已恢复"

# 运行地域故障转移测试
test-failover:
	@echo "运行地域故障转移测试..."
	@kubectl apply -f pod-kill.yaml
	@echo "Pod 故障已注入，监控故障转移..."
	@sleep 30
	@kubectl delete -f pod-kill.yaml
	@echo "故障转移测试完成"

# 运行时钟偏移容错测试
test-clock:
	@echo "运行时钟偏移容错测试..."
	@kubectl apply -f clock-skew.yaml
	@echo "时钟偏移已注入，等待 120 秒..."
	@sleep 120
	@kubectl delete -f clock-skew.yaml
	@echo "时钟偏移测试完成"

# 运行数据一致性验证测试
test-consistency:
	@echo "运行数据一致性验证测试..."
	@echo "验证 Region A 和 Region B 的数据一致性..."
	@curl -s http://region-a-service/api/v1/reconcile/merkle-root | jq
	@curl -s http://region-b-service/api/v1/reconcile/merkle-root | jq

# 应用网络故障配置
apply-network:
	@echo "应用网络故障配置..."
	@kubectl apply -f network-partition.yaml
	@echo "网络故障配置已应用"
	@kubectl get networkchaos

# 应用时钟偏移配置
apply-clock:
	@echo "应用时钟偏移配置..."
	@kubectl apply -f clock-skew.yaml
	@echo "时钟偏移配置已应用"
	@kubectl get timechaos

# 应用 Pod 故障配置
apply-pod:
	@echo "应用 Pod 故障配置..."
	@kubectl apply -f pod-kill.yaml
	@echo "Pod 故障配置已应用"
	@kubectl get podchaos

# 删除所有 Chaos 配置
delete-all:
	@echo "删除所有 Chaos 配置..."
	@kubectl delete networkchaos --all --ignore-not-found=true
	@kubectl delete timechaos --all --ignore-not-found=true
	@kubectl delete podchaos --all --ignore-not-found=true
	@echo "所有 Chaos 配置已删除"

# 清理测试环境
clean: delete-all
	@echo "清理测试环境..."
	@rm -f *.log
	@echo "清理完成"

# 查看 Chaos 实验状态
status:
	@echo "Chaos 实验状态:"
	@echo ""
	@echo "NetworkChaos:"
	@kubectl get networkchaos -o wide || echo "无"
	@echo ""
	@echo "TimeChaos:"
	@kubectl get timechaos -o wide || echo "无"
	@echo ""
	@echo "PodChaos:"
	@kubectl get podchaos -o wide || echo "无"

# 查看 Pod 状态
pods:
	@echo "Region A Pods:"
	@kubectl get pods -n $(REGION_A_NAMESPACE) -l app=im-service
	@echo ""
	@echo "Region B Pods:"
	@kubectl get pods -n $(REGION_B_NAMESPACE) -l app=im-service

# 查看日志
logs-region-a:
	@kubectl logs -n $(REGION_A_NAMESPACE) -l app=im-service --tail=100

logs-region-b:
	@kubectl logs -n $(REGION_B_NAMESPACE) -l app=im-service --tail=100

# 监控 Pod
watch-pods:
	@watch -n 2 'kubectl get pods -n $(REGION_A_NAMESPACE) -l app=im-service && echo "" && kubectl get pods -n $(REGION_B_NAMESPACE) -l app=im-service'

# 监控 Chaos 实验
watch-chaos:
	@watch -n 2 'kubectl get networkchaos,timechaos,podchaos'
