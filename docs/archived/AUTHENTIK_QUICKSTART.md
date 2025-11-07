# Authentik SSO 快速启动指南

## 🚀 5分钟快速开始

### 第一步：启动 Authentik 服务

```bash
cd /Users/ai6677/dev/coding/golang/microvibe-go

# 启动 Authentik
./scripts/start-authentik.sh
```

等待所有容器启动完成（约30秒）。

### 第二步：初始化管理员账号

1. 打开浏览器访问：http://localhost:9000/if/flow/initial-setup/
2. 填写管理员信息：
   - Email: `admin@microvibe.com`
   - Password: `设置强密码`
3. 点击"Continue"完成设置

### 第三步：登录管理后台

访问：http://localhost:9000/if/admin/

使用刚才设置的账号登录。

### 第四步：创建 OAuth2 提供者

1. 点击左侧菜单 **Applications** → **Providers**
2. 点击右上角 **Create** 按钮
3. 选择 **OAuth2/OpenID Provider**
4. 填写配置：

   | 字段 | 值 |
   |------|-----|
   | Name | `MicroVibe Backend` |
   | Authorization flow | `default-provider-authorization-implicit-consent` |
   | Client type | `Confidential` |
   | Client ID | `microvibe-backend` |
   | Redirect URIs | `http://localhost:8888/auth/callback` |

5. 点击 **Finish** 保存
6. **重要**：记录下 **Client Secret**（只显示一次）

### 第五步：创建应用

1. 点击左侧菜单 **Applications** → **Applications**
2. 点击右上角 **Create** 按钮
3. 填写配置：

   | 字段 | 值 |
   |------|-----|
   | Name | `MicroVibe` |
   | Slug | `microvibe` |
   | Provider | `MicroVibe Backend`（选择刚创建的）|
   | Launch URL | `http://localhost:8888/` |

4. 点击 **Create** 保存

### 第六步：更新后端配置

编辑 `configs/config.yaml`，添加：

```yaml
# OAuth2/OIDC 配置
oauth:
  authentik:
    enabled: true
    issuer_url: "http://localhost:9000/application/o/microvibe/"
    client_id: "microvibe-backend"
    client_secret: "粘贴第四步记录的 Client Secret"
    redirect_url: "http://localhost:8888/auth/callback"
    scopes:
      - openid
      - email
      - profile
```

### 第七步：安装 Go 依赖

```bash
cd /Users/ai6677/dev/coding/golang/microvibe-go

go get golang.org/x/oauth2
go get github.com/coreos/go-oidc/v3/oidc
```

### 第八步：测试 OAuth 登录流程

1. 启动后端服务：`make run`
2. 访问前端：http://localhost:8888/login
3. 点击 SSO 登录按钮（🔐）
4. 将跳转到 Authentik 登录页面
5. 使用管理员账号登录
6. 授权后将返回到应用

## ✅ 验证安装

### 检查服务状态

```bash
# 查看容器状态
docker-compose -f docker-compose.authentik.yml ps

# 应该看到 4 个容器都在运行：
# - microvibe-authentik-db
# - microvibe-authentik-redis
# - microvibe-authentik-server
# - microvibe-authentik-worker
```

### 测试 OIDC 配置

访问：http://localhost:9000/application/o/microvibe/.well-known/openid-configuration

应该返回 JSON 配置信息。

## 📚 下一步

- 阅读完整文档：`docs/AUTHENTIK_INTEGRATION.md`
- 配置社交登录（Google、GitHub 等）
- 启用多因素认证（MFA）
- 配置用户自助注册
- 设置访问策略

## 🛑 停止服务

```bash
docker-compose -f docker-compose.authentik.yml down
```

## 🐛 故障排查

### 无法访问 9000 端口

```bash
# 检查端口是否被占用
lsof -i :9000

# 如果被占用，修改 docker-compose.authentik.yml 中的端口映射
```

### 容器启动失败

```bash
# 查看日志
docker-compose -f docker-compose.authentik.yml logs

# 重新创建容器
docker-compose -f docker-compose.authentik.yml down -v
./scripts/start-authentik.sh
```

### OAuth 回调失败

确认 Redirect URI 完全匹配（包括协议、端口、路径）：
- 配置的：`http://localhost:8888/auth/callback`
- 实际访问：`http://localhost:8888/auth/callback`

## 📞 获取帮助

- Authentik 官方文档：https://goauthentik.io/docs/
- GitHub Issues：https://github.com/goauthentik/authentik/issues
- Discord 社区：https://goauthentik.io/discord
