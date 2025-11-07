#!/bin/bash

# Authentik OAuth 配置脚本

echo "🔐 Authentik OAuth2/OIDC 配置向导"
echo "=================================="
echo ""

# 检查 Authentik 是否运行
if ! docker ps | grep -q "microvibe-authentik-server"; then
    echo "❌ Authentik 服务器未运行，请先运行:"
    echo "   ./scripts/start-authentik.sh"
    exit 1
fi

echo "✅ Authentik 服务器正在运行"
echo ""

# 提示用户配置 Authentik
echo "📋 步骤 1: 配置 Authentik 服务器"
echo "================================"
echo ""
echo "1. 访问 Authentik 初始化页面（首次启动）："
echo "   http://localhost:9000/if/flow/initial-setup/"
echo ""
echo "2. 创建管理员账号并登录到管理后台："
echo "   http://localhost:9000/if/admin/"
echo ""
echo "3. 创建 OAuth2/OIDC Provider："
echo "   - Applications → Providers → Create"
echo "   - 选择: OAuth2/OpenID Provider"
echo "   - Name: MicroVibe Backend"
echo "   - Client ID: microvibe-backend"
echo "   - Redirect URIs:"
echo "     - http://localhost:8888/api/v1/oauth/callback"
echo "     - http://microvibe-app:8080/api/v1/oauth/callback"
echo ""
echo "4. 创建 Application："
echo "   - Applications → Applications → Create"
echo "   - Name: MicroVibe"
echo "   - Provider: MicroVibe Backend"
echo ""
echo "⚠️  重要：完成后，请复制 Provider 的 Client Secret！"
echo ""

read -p "是否已完成 Authentik 配置？(y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 请先完成 Authentik 配置"
    exit 1
fi

# 获取 Client Secret
echo ""
echo "📋 步骤 2: 配置后端"
echo "=================="
echo ""
read -p "请粘贴 Client Secret: " client_secret

if [ -z "$client_secret" ]; then
    echo "❌ Client Secret 不能为空"
    exit 1
fi

# 更新配置文件
echo ""
echo "📝 更新配置文件..."
sed -i.bak "s/client_secret: \".*\"/client_secret: \"$client_secret\"/" configs/config.yaml
sed -i.bak "s/enabled: false/enabled: true/" configs/config.yaml

echo "✅ 配置文件已更新"
echo ""

# 重新构建并启动后端
echo "📋 步骤 3: 重启后端服务"
echo "======================"
echo ""
read -p "是否立即重启后端服务？(y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔄 重新构建并启动后端..."
    docker-compose up -d --build app

    echo ""
    echo "⏳ 等待服务启动..."
    sleep 5

    # 检查 OAuth 路由是否注册
    if docker logs microvibe-app 2>&1 | grep -q "oauth"; then
        echo "✅ OAuth 路由已成功注册！"
    else
        echo "⚠️  OAuth 路由可能未注册，请检查日志："
        echo "   docker logs microvibe-app"
    fi
fi

echo ""
echo "=========================================="
echo "✅ OAuth 配置完成！"
echo "=========================================="
echo ""
echo "📝 后续步骤："
echo ""
echo "1. 测试 OAuth 登录："
echo "   curl -L http://localhost:8888/api/v1/oauth/login"
echo ""
echo "2. 在前端添加 SSO 登录按钮"
echo ""
echo "3. 查看详细文档："
echo "   - docs/AUTHENTIK_INTEGRATION.md"
echo "   - AUTHENTIK_QUICKSTART.md"
echo ""
