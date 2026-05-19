#!/bin/bash
# 视频推荐流 QPS 极限压测
# 验证目标: QPS ≥ 2000, avg ≤ 150ms, P99 ≤ 500ms, 成功率 ≥ 99.9%
#
# 使用: ./scripts/benchmark/benchmark-video.sh [target_url]

set -e

TARGET="${1:-http://host.docker.internal:8080}"
WRK="docker run --rm williamyeh/wrk"

echo "============================================"
echo "  视频推荐流 — 极限 QPS 压测"
echo "  目标: 峰值 QPS ≥ 2000"
echo "  目标: P99 ≤ 500ms, 成功率 ≥ 99.9%"
echo "============================================"
echo ""
echo "=== 负载阶梯测试 ==="
echo ""

# 阶段 1: 低负载 (50 连接)
echo ">>> 阶段 1/4: 50 并发连接"
$WRK -t4 -c50 -d30s --latency "$TARGET/api/v1/videos/feed" 2>&1
echo ""

# 阶段 2: 中等负载 (200 连接)
echo ">>> 阶段 2/4: 200 并发连接"
$WRK -t8 -c200 -d30s --latency "$TARGET/api/v1/videos/feed" 2>&1
echo ""

# 阶段 3: 高负载 (500 连接)
echo ">>> 阶段 3/4: 500 并发连接"
$WRK -t8 -c500 -d60s --latency "$TARGET/api/v1/videos/feed" 2>&1
echo ""

# 阶段 4: 极限负载 (1000 连接)
echo ">>> 阶段 4/4: 1000 并发连接"
$WRK -t8 -c1000 -d60s --latency "$TARGET/api/v1/videos/feed" 2>&1
echo ""

echo "============================================"
echo "  压测完成，请对照 SLA 检查以下指标:"
echo ""
echo "  ✅ QPS ≥ 2000      | 实际值见上"
echo "  ✅ avg ≤ 150ms     | Latency Avg 列"
echo "  ✅ P99 ≤ 500ms     | Latency P99 列"
echo "  ✅ 成功率 ≥ 99.9%  | Non-2xx or 3xx 应接近 0"
echo "============================================"
