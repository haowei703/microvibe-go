#!/bin/bash
# MicroVibe 性能基准测试脚本
# 先注册/登录获取 Token，再启动 wrk 进行压测
#
# 用法: ./scripts/benchmark/benchmark.sh [wrk_target_url]
# 默认 wrk_target: http://host.docker.internal:8080
# curl 使用 localhost (运行在 Windows 宿主机)

set -e

WRK_TARGET="${1:-http://host.docker.internal:8080}"
CURL_TARGET="http://localhost:8080"
SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"
WRK="docker run --rm -v "$SCRIPTS_DIR:/scripts" williamyeh/wrk"
TIMEOUT="30s"

echo "========================================="
echo "  MicroVibe 后端性能基准测试"
echo "  目标: $WRK_TARGET"
echo "  时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================="
echo ""

# === 1. 健康检查 ===
echo "--- [1/6] 健康检查 ---"
$WRK -t2 -c10 -d10s --latency "$WRK_TARGET/health" 2>&1
echo ""

# === 2. 注册测试用户 ===
echo "--- [2/6] 用户认证 (注册+登录) ---"
# 注册一个新用户
REGISTER_RESP=$(curl -s -X POST "$CURL_TARGET/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"benchmark_main","password":"bench123456","email":"bench_main@test.com"}')
echo "注册结果: $REGISTER_RESP"

# 登录获取 Token
LOGIN_RESP=$(curl -s -X POST "$CURL_TARGET/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"benchmark_main","password":"bench123456"}')
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "⚠️  获取 Token 失败，部分需要认证的接口跳过"
    echo "登录响应: $LOGIN_RESP"
    AUTH=""
else
    AUTH="Authorization: Bearer $TOKEN"
    echo "Token 获取成功，长度: ${#TOKEN}"
fi
echo ""

# === 3. 视频推荐流（核心接口） ===
echo "--- [3/6] 视频推荐流 ---"
if [ -n "$AUTH" ]; then
    $WRK -t4 -c100 -d$TIMEOUT --latency -H "$AUTH" "$WRK_TARGET/api/v1/videos/feed" 2>&1
else
    $WRK -t4 -c100 -d$TIMEOUT --latency "$WRK_TARGET/api/v1/videos/feed" 2>&1
fi
echo ""

# === 4. 视频详情 ===
echo "--- [4/6] 视频详情 ---"
VIDEO_ID="${2:-1}"  # 默认测试视频 ID 1
if [ -n "$AUTH" ]; then
    $WRK -t4 -c100 -d$TIMEOUT --latency -H "$AUTH" "$WRK_TARGET/api/v1/videos/$VIDEO_ID" 2>&1
else
    $WRK -t4 -c100 -d$TIMEOUT --latency "$WRK_TARGET/api/v1/videos/$VIDEO_ID" 2>&1
fi
echo ""

# === 5. 综合搜索 ===
echo "--- [5/6] 综合搜索 ---"
$WRK -t2 -c50 -d$TIMEOUT --latency "$WRK_TARGET/api/v1/search?keyword=test" 2>&1
echo ""

# === 6. 热门视频（高并发） ===
echo "--- [6/6] 热门视频 ---"
$WRK -t4 -c200 -d$TIMEOUT --latency "$WRK_TARGET/api/v1/videos/hot" 2>&1
echo ""

echo "========================================="
echo "  基准测试完成"
echo "  请访问 Grafana 查看详细指标:"
echo "  http://localhost:3000"
echo "  Dashboard: MicroVibe 性能监控"
echo "========================================="
