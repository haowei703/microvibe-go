#!/bin/bash
# 持续负载监控模式 - 配合 Grafana 实时查看性能指标
# 在 Grafana 打开状态下运行此脚本查看实时数据刷新
#
# 使用: ./scripts/benchmark/benchmark-monitor.sh [target_url] [duration]

set -e

TARGET="${1:-http://host.docker.internal:8080}"
DURATION="${2:-5m}"
WRK="docker run --rm williamyeh/wrk"

echo "============================================"
echo "  MicroVibe 持续负载模式"
echo "  持续时间: $DURATION"
echo "  请在 Grafana 中查看实时指标:"
echo "  http://localhost:3000"
echo ""
echo "  提示: 按 Ctrl+C 随时停止"
echo "============================================"
echo ""

# 混合负载: 模拟真实用户行为
echo "启动混合负载..."

# 后台并发测试: 视频推荐流 (主要负载)
$WRK -t4 -c200 -d$DURATION --latency \
  "$TARGET/api/v1/videos/feed" 2>&1 &

WRK_PID=$!

echo "压测进行中 (PID: $WRK_PID)..."
echo "打开 http://localhost:3000 查看实时指标"
echo ""

# 等待完成
wait $WRK_PID

echo ""
echo "============================================"
echo "  持续负载测试完成"
echo "============================================"
