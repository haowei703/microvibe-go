#!/bin/bash
# ============================================================
# MicroVibe 安全流量模拟器
# 模拟各类攻击和异常请求，为 Grafana 安全仪表盘生成实时数据
#
# 使用方式:
#   ./scripts/security-traffic-sim.sh
#   ./scripts/security-traffic-sim.sh 60    # 运行 60 秒后停止
#
# Grafana: http://localhost:3000/d/microvibe-security
# ============================================================

BASE_URL="${BASE_URL:-http://localhost:8080}"
DURATION="${1:-0}"   # 0 = 一直运行

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[$(date +%H:%M:%S)]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()  { echo -e "${RED}[ALERT]${NC} $1"; }
ok()   { echo -e "${GREEN}[OK]${NC} $1"; }

# 随机延迟 (0.1 - 2.0 秒)，模拟真实流量间隔
random_sleep() {
    sleep "$(awk "BEGIN {printf \"%.2f\", 0.1 + rand() * 1.9}")"
}

# ---- 攻击场景模拟 ----

# 1. 未登录直接访问需要认证的接口 (触发 auth_failures: missing_header)
sim_unauthorized_access() {
    local endpoints=(
        "/api/v1/users/me"
        "/api/v1/users/me/videos"
        "/api/v1/users/me/stats"
        "/api/v1/videos/follow"
        "/api/v1/notifications"
        "/api/v1/messages/conversations"
    )
    local ep="${endpoints[$((RANDOM % ${#endpoints[@]}))]}"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$ep" 2>/dev/null)
    if [ "$code" = "401" ]; then
        warn "未授权访问 $ep -> $code"
    fi
}

# 2. 伪造/过期 Token (触发 auth_failures: invalid_token + token_validation_failures)
sim_forged_token() {
    local endpoints=(
        "/api/v1/users/me"
        "/api/v1/admin/videos"
        "/api/v1/videos/upload"
        "/api/v1/users/me/privacy"
    )
    local ep="${endpoints[$((RANDOM % ${#endpoints[@]}))]}"
    local fake_tokens=(
        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
        "invalid.token.format"
        "Bearer.invalid.payload.signature"
    )
    local token="${fake_tokens[$((RANDOM % ${#fake_tokens[@]}))]}"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $token" "$BASE_URL$ep" 2>/dev/null)
    if [ "$code" = "401" ]; then
        err "伪造Token攻击 $ep -> $code"
    fi
}

# 3. 暴力破解登录 (触发 rate limit: /api/v1/auth/login)
sim_brute_force() {
    local passwords=("admin123" "password" "123456" "test" "qwerty" "letmein" "admin" "root" "guest")
    local users=("admin" "root" "test" "user1" "superadmin" "manager")
    local count=$((5 + RANDOM % 10))
    log "暴力破解模拟: $count 次快速登录尝试"
    for i in $(seq 1 $count); do
        local u="${users[$((RANDOM % ${#users[@]}))]}"
        local p="${passwords[$((RANDOM % ${#passwords[@]}))]}"
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$u\",\"password\":\"$p\"}" 2>/dev/null)
        if [ "$code" = "429" ]; then
            err "速率限制触发: 登录爆破已拦截 -> $code"
            break
        fi
    done
}

# 4. 跨域攻击 (触发 CORS blocked: 非白名单 Origin)
sim_cors_attack() {
    local malicious_origins=(
        "http://evil.example.com"
        "http://phishing-site.net"
        "http://localhost:9999"
        "http://malware.download"
        "https://fake-microvibe.com"
    )
    local origin="${malicious_origins[$((RANDOM % ${#malicious_origins[@]}))]}"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" -H "Origin: $origin" \
        "$BASE_URL/api/v1/auth/login" 2>/dev/null)
    warn "CORS 攻击尝试: Origin=$origin"
}

# 5. 暴力注册 (触发 rate limit: /api/v1/auth/register)
sim_mass_registration() {
    local count=$((3 + RANDOM % 6))
    for i in $(seq 1 $count); do
        local r=$RANDOM
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/auth/register" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"bot_$r\",\"password\":\"BotPass123\",\"email\":\"bot$r@spam.com\"}" 2>/dev/null)
        if [ "$code" = "429" ]; then
            err "注册攻击已被限流 -> $code"
            break
        fi
    done
}

# 6. 越权访问 Admin 接口 (触发 auth_failures: insufficient_role)
sim_privilege_escalation() {
    # 先正常登录获取普通用户 token
    local token
    token=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"testuser","password":"test123"}' 2>/dev/null | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$token" ]; then
        local admin_eps=(
            "/api/v1/admin/users"
            "/api/v1/admin/videos"
            "/api/v1/admin/reports"
        )
        local ep="${admin_eps[$((RANDOM % ${#admin_eps[@]}))]}"
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $token" "$BASE_URL$ep" 2>/dev/null)
        if [ "$code" = "403" ]; then
            err "越权攻击被拦截: 普通用户访问 $ep -> $code"
        fi
    fi
}

# 7. 正常流量 (保持真实用户行为基线)
sim_normal_traffic() {
    local eps=("/health" "/api/v1/ping" "/api/v1/videos/feed" "/api/v1/videos/hot" "/api/v1/categories")
    local ep="${eps[$((RANDOM % ${#eps[@]}))]}"
    curl -s -o /dev/null "$BASE_URL$ep" 2>/dev/null
}

# 8. 正常用户登录后操作
sim_normal_user() {
    local token
    token=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"testuser","password":"test123"}' 2>/dev/null | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$token" ]; then
        curl -s -o /dev/null -H "Authorization: Bearer $token" \
            "$BASE_URL/api/v1/users/me" 2>/dev/null
        curl -s -o /dev/null -H "Authorization: Bearer $token" \
            "$BASE_URL/api/v1/videos/feed" 2>/dev/null
    fi
}

# ---- 主循环 ----

echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║     MicroVibe 安全流量模拟器                          ║"
echo "║     Grafana: http://localhost:3000/d/microvibe-security ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""

START_TIME=$(date +%s)
CYCLE=0

cleanup() {
    echo ""
    log "模拟器停止。运行了 $CYCLE 轮."
    exit 0
}
trap cleanup SIGINT SIGTERM

while true; do
    CYCLE=$((CYCLE + 1))
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━ 第 $CYCLE 轮 ━━━━━━━━━━━━━━━━━━━"

    # 混合流量: 40% 正常, 60% 攻击/异常
    sim_normal_traffic
    random_sleep

    sim_unauthorized_access
    random_sleep

    sim_normal_traffic
    random_sleep

    sim_forged_token
    random_sleep

    sim_cors_attack
    random_sleep

    sim_normal_traffic
    random_sleep

    sim_unauthorized_access
    random_sleep

    sim_normal_user
    random_sleep

    # 随机选择运行高影响攻击
    case $((RANDOM % 3)) in
        0) sim_brute_force ;;
        1) sim_mass_registration ;;
        2) sim_privilege_escalation ;;
    esac

    ok "第 $CYCLE 轮完成 | 查看 Grafana: http://localhost:3000/d/microvibe-security"

    # 检查运行时长
    if [ "$DURATION" -gt 0 ]; then
        ELAPSED=$(($(date +%s) - START_TIME))
        if [ "$ELAPSED" -ge "$DURATION" ]; then
            log "达到运行时长 ${DURATION}s，停止."
            break
        fi
    fi

    sleep "$(awk "BEGIN {printf \"%.2f\", 2 + rand() * 3}")"
done
