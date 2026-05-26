#!/bin/bash
# MicroVibe Security Report Generator
# 运行安全测试、gosec扫描，并生成带图表的HTML报告

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "=== MicroVibe 安全测试报告生成 ==="
echo ""

# 1. 运行安全单元测试
echo "[1/3] 运行安全测试..."
go test -json \
    -run "TestJWT|TestPassword|TestAuth|TestCORS|TestSecurity|TestLogin|TestRegister|TestComment|TestUpload" \
    ./pkg/utils/ ./internal/middleware/ ./internal/handler/ \
    > security-test-output.json 2>/dev/null || true
echo "  -> 测试结果写入 security-test-output.json"

# 2. 运行 gosec 静态分析
echo "[2/3] 运行 gosec 安全扫描..."
if command -v gosec &> /dev/null; then
    gosec -fmt=json -quiet ./... > security-gosec-output.json 2>/dev/null || true
    echo "  -> gosec 结果写入 security-gosec-output.json"
else
    echo "  -> gosec 未安装，跳过 (go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)"
    echo '{"Issues":[]}' > security-gosec-output.json
fi

# 3. 生成 HTML 报告
echo "[3/3] 生成 HTML 报告..."
go run cmd/security-report/main.go
echo ""
echo "=== 报告生成完成 ==="
echo "在浏览器中打开: file://${PROJECT_DIR}/security-report.html"
echo "查看实时 Grafana 仪表盘: http://localhost:3000/d/microvibe-security"
