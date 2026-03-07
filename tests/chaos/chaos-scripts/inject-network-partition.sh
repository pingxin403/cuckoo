#!/bin/bash
# 注入网络分区故障（Docker 环境）
# 使用 iptables 阻断两个地域之间的网络通信

set -e

REGION_A_CONTAINER="${REGION_A_CONTAINER:-im-service-region-a}"
REGION_B_CONTAINER="${REGION_B_CONTAINER:-im-service-region-b}"
DURATION="${DURATION:-60}"

echo "注入网络分区故障..."
echo "Region A: $REGION_A_CONTAINER"
echo "Region B: $REGION_B_CONTAINER"
echo "持续时间: ${DURATION}s"

# 获取容器 IP
REGION_A_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$REGION_A_CONTAINER")
REGION_B_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$REGION_B_CONTAINER")

echo "Region A IP: $REGION_A_IP"
echo "Region B IP: $REGION_B_IP"

# 在 Region A 容器中阻断到 Region B 的流量
docker exec "$REGION_A_CONTAINER" sh -c "
    iptables -A OUTPUT -d $REGION_B_IP -j DROP
    iptables -A INPUT -s $REGION_B_IP -j DROP
"

# 在 Region B 容器中阻断到 Region A 的流量
docker exec "$REGION_B_CONTAINER" sh -c "
    iptables -A OUTPUT -d $REGION_A_IP -j DROP
    iptables -A INPUT -s $REGION_A_IP -j DROP
"

echo "网络分区已注入"
echo "等待 ${DURATION} 秒..."
sleep "$DURATION"

# 恢复网络
echo "恢复网络连接..."
docker exec "$REGION_A_CONTAINER" sh -c "
    iptables -D OUTPUT -d $REGION_B_IP -j DROP
    iptables -D INPUT -s $REGION_B_IP -j DROP
"

docker exec "$REGION_B_CONTAINER" sh -c "
    iptables -D OUTPUT -d $REGION_A_IP -j DROP
    iptables -D INPUT -s $REGION_A_IP -j DROP
"

echo "网络连接已恢复"
