# OAuth SSO 设置指南

## 问题诊断

### 问题 1: 访问 `/api/v1/oauth/login` 返回 404

**原因**: OAuth 功能当前处于禁用状态（`oauth.authentik.enabled: false`）

**解决**: 按照下面的步骤启用 OAuth 并配置 Authentik

### 问题 2: 重定向到内部 Docker 主机名

**症状**: 访问 OAuth 登录后，浏览器重定向到 `http://microvibe-authentik-server:9000`（无法访问）

**根本原因**: ⚠️ **Authentik 管理后台的重定向 URL 配置不正确**

**正确解决方案**:
1. 登录 Authentik 管理后台 (http://localhost:9000/if/admin/)
2. 编辑 Provider 配置，确保 Redirect URIs 正确
3. 确保 Authentik Application 的 Launch URL 使用浏览器可访问的地址

**错误做法**: ❌ 不要在后端代码中尝试替换 URL，问题应该在 Authentik 配置中解决

## 快速设置（3 分钟）

### 方法一：使用自动化脚本（推荐）

```bash
./scripts/setup-oauth.sh
```

按照提示完成配置即可。

### 方法二：手动配置

#### 第一步：配置 Authentik 服务器

1. **访问初始化页面**（首次启动）：
   ```bash
   open http://localhost:9000/if/flow/initial-setup/
   ```

2. **创建管理员账号**：
   - Email: `admin@microvibe.com`
   - Password: `设置强密码`（例如：`Admin@123456`）

3. **登录管理后台**：
   ```bash
   open http://localhost:9000/if/admin/
   ```

4. **创建 OAuth2/OIDC Provider**：

   导航到：**Applications** → **Providers** → **Create**

   选择：**OAuth2/OpenID Provider**

   填写表单：
   ```
   Name: MicroVibe Backend
   Authorization flow: default-provider-authorization-implicit-consent
   Client type: Confidential
   Client ID: microvibe-backend
   ```

   在 "Redirect URIs" 部分添加（重要！）：
   ```
   http://localhost:8888/api/v1/oauth/callback
   http://microvibe-app:8080/api/v1/oauth/callback
   ```

   点击 **Finish** 保存

   ⚠️ **重要**：复制显示的 **Client Secret**（只显示一次！）

   示例：`AbCdEfGh1234567890IjKlMnOpQrStUvWxYz`

5. **创建 Application**：

   导航到：**Applications** → **Applications** → **Create**

   填写表单：
   ```
   Name: MicroVibe
   Slug: microvibe
   Provider: MicroVibe Backend（选择刚创建的）
   Launch URL: http://localhost:8888/
   ```

   点击 **Create** 保存

#### 第二步：更新后端配置

1. **编辑配置文件** `configs/config.yaml`：

   ```yaml
   oauth:
     authentik:
       enabled: true  # 改为 true
       issuer_url: "http://localhost:9000/application/o/microvibe/"  # ⚠️ 使用 localhost（需要特殊网络配置，见下文）
       client_id: "microvibe-backend"
       client_secret: "粘贴你的 Client Secret"  # 粘贴步骤 4 中复制的 Secret
       redirect_url: "http://localhost:8888/api/v1/oauth/callback"
       scopes:
         - "openid"
         - "email"
         - "profile"
   ```

   **重要说明 - Docker 网络配置**:

   如果使用 Docker 部署，后端容器需要能访问 Authentik。有两种方案：

   **方案 A（推荐）**: 使用 Docker 内部主机名
   ```yaml
   issuer_url: "http://microvibe-authentik-server:9000/application/o/microvibe/"
   ```
   ⚠️ 但这会导致浏览器无法访问重定向 URL。需要在 Authentik 管理后台配置：
   - **Provider 的 "Issuer mode"**: 设置为 "Per Provider" 或使用自定义域名
   - **Application 的 "Launch URL"**: 使用 `http://localhost:9000/...`

   **方案 B（开发环境）**: 在宿主机运行后端
   ```bash
   # 停止 Docker 中的后端
   docker-compose stop app

   # 在宿主机运行
   make run
   # 或
   go run cmd/server/main.go
   ```
   这样后端可以直接访问 `localhost:9000`，无需特殊配置。

2. **重新构建并启动后端**：

   ```bash
   docker-compose up -d --build app
   ```

3. **等待服务启动**（约 5 秒）：

   ```bash
   docker logs -f microvibe-app
   ```

   看到类似输出表示成功：
   ```
   [GIN-debug] GET    /api/v1/oauth/login     --> ...
   [GIN-debug] GET    /api/v1/oauth/callback  --> ...
   INFO  OAuth handler initialized successfully
   ```

#### 第三步：测试 OAuth 登录

1. **测试重定向**：
   ```bash
   curl -L http://localhost:8888/api/v1/oauth/login
   ```

   应该返回 Authentik 登录页面的 HTML

2. **浏览器测试**：
   ```bash
   open http://localhost:8888/api/v1/oauth/login
   ```

   应该跳转到 Authentik 登录页面

3. **完整流程测试**：
   - 访问：http://localhost:8888/login
   - 点击 SSO 登录按钮（🔐）
   - 登录 Authentik
   - 应该返回到应用并自动登录

## 故障排查

### 问题 1: 404 - OAuth 路由未找到

**原因**：`oauth.authentik.enabled` 为 `false`

**解决**：
```bash
# 检查配置
docker exec microvibe-app cat /root/configs/config.yaml | grep -A 5 "oauth:"

# 确保 enabled: true
# 如果不是，修改配置文件并重启
docker-compose up -d --build app
```

### 问题 2: "Failed to create OIDC provider"

**原因**：后端无法连接到 Authentik 服务器

**解决**：
```bash
# 检查 Authentik 是否运行
docker ps | grep authentik

# 检查网络连接
docker exec microvibe-app ping -c 3 microvibe-authentik-server

# 如果失败，确保两个服务在同一 Docker 网络
docker network inspect microvibe-network
```

### 问题 3: "Invalid redirect URI"

**原因**：Authentik Provider 中配置的 Redirect URI 与实际不匹配

**解决**：
1. 登录 Authentik 管理后台
2. Applications → Providers → MicroVibe Backend → Edit
3. 确保 Redirect URIs 包含：
   ```
   http://localhost:8888/api/v1/oauth/callback
   http://microvibe-app:8080/api/v1/oauth/callback
   ```

### 问题 4: "State mismatch"

**原因**：OAuth state 验证失败（可能是 cookie 问题）

**解决**：
- 清除浏览器 cookies
- 确保浏览器允许 localhost cookies
- 检查 CORS 配置

## 验证配置

### 检查 Authentik 配置

访问：http://localhost:9000/application/o/microvibe/.well-known/openid-configuration

应该返回类似：
```json
{
  "issuer": "http://localhost:9000/application/o/microvibe/",
  "authorization_endpoint": "http://localhost:9000/application/o/authorize/",
  "token_endpoint": "http://localhost:9000/application/o/token/",
  "userinfo_endpoint": "http://localhost:9000/application/o/userinfo/",
  ...
}
```

### 检查后端配置

```bash
# 检查 OAuth 是否启用
docker exec microvibe-app cat /root/configs/config.yaml | grep -A 10 "oauth:"

# 检查日志中的 OAuth 初始化
docker logs microvibe-app 2>&1 | grep -i oauth

# 应该看到：
# INFO  OAuth handler initialized successfully
```

### 检查路由注册

```bash
# 查看所有注册的路由
docker logs microvibe-app 2>&1 | grep "/api/v1/oauth"

# 应该看到：
# [GIN-debug] GET    /api/v1/oauth/login
# [GIN-debug] GET    /api/v1/oauth/callback
```

## 下一步：前端集成

配置完成后，需要在前端添加 SSO 登录按钮。参考：
- `docs/AUTHENTIK_INTEGRATION.md` - 详细集成指南
- `AUTHENTIK_QUICKSTART.md` - 快速开始指南

前端集成代码示例：

```vue
<!-- login.vue -->
<button
  class="w-10 h-10 rounded-full bg-gradient-to-r from-pink-500 to-purple-500"
  @click="handleAuthentikLogin"
>
  <span class="text-xl">🔐</span>
</button>

<script setup lang="ts">
function handleAuthentikLogin() {
  window.location.href = 'http://localhost:8888/api/v1/oauth/login'
}
</script>
```

## 安全注意事项

1. ⚠️ **生产环境**必须使用 HTTPS
2. ⚠️ 修改默认的 `AUTHENTIK_SECRET_KEY`
3. ⚠️ 使用强密码
4. ⚠️ 定期备份 PostgreSQL 数据库
5. ⚠️ 启用 Authentik MFA（多因素认证）

## 获取帮助

- Authentik 文档: https://goauthentik.io/docs/
- 项目文档: `docs/AUTHENTIK_INTEGRATION.md`
- GitHub Issues: https://github.com/goauthentik/authentik/issues
