#!/bin/bash
# 注入网络延迟故障（Docker 环境）
# 使用 tc (traffic control) 添加网络延迟

set -e

CONTAINER="${CONTAINER:-im-service-region-a}"
LATENCY="${LATENCY:-500ms}"
JITTER="${JITTER:-100ms}"
DURATION="${DURATION:-120}"

echo "注入网络延迟故障..."
echo "容器: $CONTAINER"
echo "延迟: $LATENCY"
echo "抖动: $JITTER"
echo "持续时间: ${DURATION}s"

# 添加网络延迟
docker exec "$CONTAINER" sh -c "
    tc qdisc add dev eth0 root netem delay $LATENCY $JITTER
"

echo "网络延迟已注入"
echo "等待 ${DURATION} 秒..."
sleep "$DURATION"

# 删除网络延迟
echo "删除网络延迟..."
docker exec "$CONTAINER" sh -c "
    tc qdisc del dev eth0 root
"

echo "网络延迟已删除"
