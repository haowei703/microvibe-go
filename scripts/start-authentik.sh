#!/bin/bash

# 启动 Authentik 认证服务器脚本

set -e

echo "🚀 启动 Authentik 认证服务器..."

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 创建网络（如果不存在）
if ! docker network inspect microvibe-network > /dev/null 2>&1; then
    echo "📡 创建 Docker 网络 microvibe-network..."
    docker network create microvibe-network
fi

# 检查环境变量文件
if [ ! -f .env.authentik ]; then
    echo "⚠️  .env.authentik 文件不存在，使用默认配置"
    echo "⚠️  生产环境请修改 AUTHENTIK_SECRET_KEY"
fi

# 创建必要的目录
mkdir -p authentik-media authentik-certs authentik-custom-templates

# 启动 Authentik
echo "🐳 启动 Authentik 容器..."
docker-compose -f docker-compose.authentik.yml --env-file .env.authentik up -d

echo ""
echo "✅ Authentik 启动成功！"
echo ""
echo "📍 访问地址："
echo "   - Authentik 管理界面: http://localhost:9000/if/admin/"
echo "   - Authentik 用户界面: http://localhost:9000/"
echo ""
echo "🔑 默认管理员账号（首次启动后设置）："
echo "   - 访问 http://localhost:9000/if/flow/initial-setup/"
echo "   - 设置管理员邮箱和密码"
echo ""
echo "📝 查看日志："
echo "   docker-compose -f docker-compose.authentik.yml logs -f"
echo ""
echo "🛑 停止服务："
echo "   docker-compose -f docker-compose.authentik.yml down"
echo ""
