#!/bin/bash
# 注入时钟偏移故障（Docker 环境）
# 使用 libfaketime 模拟时钟偏移

set -e

CONTAINER="${CONTAINER:-im-service-region-a}"
OFFSET="${OFFSET:-+5s}"  # 时钟偏移量
DURATION="${DURATION:-120}"

echo "注入时钟偏移故障..."
echo "容器: $CONTAINER"
echo "偏移量: $OFFSET"
echo "持续时间: ${DURATION}s"

# 注意: 这需要容器中安装 libfaketime
# 或者使用 date 命令修改系统时间（需要 SYS_TIME capability）

# 方法 1: 使用 date 命令（需要 --cap-add SYS_TIME）
docker exec "$CONTAINER" sh -c "
    # 保存当前时间
    ORIGINAL_TIME=\$(date +%s)
    
    # 计算偏移后的时间
    if [[ '$OFFSET' == +* ]]; then
        OFFSET_SECONDS=\${OFFSET:1}
        OFFSET_SECONDS=\${OFFSET_SECONDS%s}
        NEW_TIME=\$((ORIGINAL_TIME + OFFSET_SECONDS))
    else
        OFFSET_SECONDS=\${OFFSET:1}
        OFFSET_SECONDS=\${OFFSET_SECONDS%s}
        NEW_TIME=\$((ORIGINAL_TIME - OFFSET_SECONDS))
    fi
    
    # 设置新时间
    date -s @\$NEW_TIME
    
    echo \"时钟已偏移: $OFFSET\"
    echo \"原始时间: \$(date -d @\$ORIGINAL_TIME)\"
    echo \"新时间: \$(date)\"
"

echo "时钟偏移已注入"
echo "等待 ${DURATION} 秒..."
sleep "$DURATION"

# 恢复时钟（使用 NTP 同步）
echo "恢复时钟..."
docker exec "$CONTAINER" sh -c "
    # 如果安装了 ntpdate
    if command -v ntpdate &> /dev/null; then
        ntpdate -u pool.ntp.org
    else
        # 手动恢复到当前时间
        date -s \"\$(date -u)\"
    fi
    
    echo \"时钟已恢复: \$(date)\"
"

echo "时钟已恢复"
