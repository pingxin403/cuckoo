#!/bin/bash
# 注入网络丢包故障（Docker 环境）
# 使用 tc (traffic control) 添加丢包

set -e

CONTAINER="${CONTAINER:-im-service-region-a}"
LOSS_RATE="${LOSS_RATE:-10}"  # 丢包率 (%)
DURATION="${DURATION:-90}"

echo "注入网络丢包故障..."
echo "容器: $CONTAINER"
echo "丢包率: ${LOSS_RATE}%"
echo "持续时间: ${DURATION}s"

# 添加网络丢包
docker exec "$CONTAINER" sh -c "
    tc qdisc add dev eth0 root netem loss ${LOSS_RATE}%
"

echo "网络丢包已注入"
echo "等待 ${DURATION} 秒..."
sleep "$DURATION"

# 删除网络丢包
echo "删除网络丢包..."
docker exec "$CONTAINER" sh -c "
    tc qdisc del dev eth0 root
"

echo "网络丢包已删除"
