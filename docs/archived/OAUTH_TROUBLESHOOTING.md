# OAuth SSO 故障排查与解决方案

## 问题总结

在集成 Authentik OAuth2/OIDC 过程中遇到的主要问题及解决方案。

## 问题 1: OAuth 路由返回 404

### 症状
```bash
[GIN] 2025/10/31 - 10:52:38 | 404 | 109.833µs | ::1 | GET "/api/v1/oauth/login"
```

### 原因
配置文件中 `oauth.authentik.enabled: false`，导致 OAuth 处理器未初始化，路由未注册。

### 解决方案
修改 `configs/config.yaml`：
```yaml
oauth:
  authentik:
    enabled: true  # 改为 true
    client_secret: "从 Authentik 管理后台复制的 Secret"
```

## 问题 2: Docker 网络隔离

### 症状
```
ERROR Failed to create OIDC provider {"error": "Get \"http://microvibe-authentik-server:9000/...\": EOF"}
```

### 原因
- `microvibe-app` 容器在 `microvibe-go_microvibe-network` 网络中
- Authentik 容器在 `microvibe-network` 网络中
- 两个网络互相隔离，无法通信

### 解决方案（临时）
```bash
# 将 app 容器连接到 Authentik 所在的网络
docker network connect microvibe-network microvibe-app
docker restart microvibe-app
```

### 问题
每次重新构建容器（`docker-compose up -d --build app`）都会丢失自定义网络连接。

### 长期解决方案
**方案 A**: 在宿主机运行后端（推荐用于开发环境）
```bash
docker-compose stop app
make run
```

**方案 B**: 修改 Docker Compose 配置
在 `docker-compose.yml` 中为 app 服务添加外部网络：
```yaml
services:
  app:
    networks:
      - microvibe-network  # 添加此行
      - default

networks:
  microvibe-network:
    external: true
```

## 问题 3: 浏览器无法访问 Docker 内部主机名

### 症状
OAuth 登录时，浏览器被重定向到：
```
http://microvibe-authentik-server:9000/application/o/authorize/...
```
浏览器无法解析 `microvibe-authentik-server` 这个 Docker 内部主机名。

### 根本原因分析

这是一个**经典的双重解析问题**：

1. **后端容器内**：需要使用 `http://microvibe-authentik-server:9000` 来调用 OIDC Discovery API
2. **浏览器中**：需要使用 `http://localhost:9000` 来访问 Authentik 登录页面

### ❌ 错误的解决方案

**不要在后端代码中尝试 URL 替换！** 这会导致：
- 代码复杂度增加
- 难以维护
- 可能破坏 OAuth2 协议的安全性
- 无法正确处理所有边缘情况

### ✅ 正确的解决方案

**在 Authentik 管理后台配置正确的 Issuer 和 Redirect URLs**

#### 步骤：

1. **登录 Authentik 管理后台**
   ```
   http://localhost:9000/if/admin/
   ```

2. **配置 Provider 的 Issuer Mode**

   导航到：**Applications** → **Providers** → **MicroVibe Backend** → **Edit**

   - **Issuer mode**: 选择 "Each provider has a different issuer, based on the application slug" 或 "Same identifier is used for all providers"
   - 这确保浏览器收到的 URL 是正确的外部地址

3. **验证 Redirect URIs**

   在同一页面确保 Redirect URIs 包含：
   ```
   http://localhost:8888/api/v1/oauth/callback
   ```

4. **配置 Application 的 Launch URL**

   导航到：**Applications** → **Applications** → **MicroVibe** → **Edit**

   - **Launch URL**: `http://localhost:8888/` 或你的前端地址
   - 这确保应用使用正确的外部 URL

#### 后端配置（两种方案）

**开发环境（推荐）**: 在宿主机运行后端
```yaml
# configs/config.yaml
oauth:
  authentik:
    enabled: true
    issuer_url: "http://localhost:9000/application/o/microvibe/"
    client_id: "microvibe-backend"
    client_secret: "你的 Secret"
    redirect_url: "http://localhost:8888/api/v1/oauth/callback"
```

```bash
docker-compose stop app
make run
```

**生产环境**: 使用反向代理（如 Nginx）
- 配置 Nginx 将 `authentik.yourdomain.com` 代理到 Authentik 容器
- 使用统一的外部域名
- 后端和浏览器都使用相同的 URL

## 验证步骤

### 1. 检查 Authentik OIDC Discovery 端点

```bash
curl http://localhost:9000/application/o/microvibe/.well-known/openid-configuration
```

应该返回类似：
```json
{
  "issuer": "http://localhost:9000/application/o/microvibe/",
  "authorization_endpoint": "http://localhost:9000/application/o/authorize/",
  "token_endpoint": "http://localhost:9000/application/o/token/",
  ...
}
```

确保所有 URL 都是 `localhost:9000` 而不是 Docker 内部主机名。

### 2. 检查后端日志

```bash
docker logs microvibe-app 2>&1 | grep -i oauth
```

应该看到：
```
INFO  OAuth handler initialized successfully
[GIN-debug] GET    /api/v1/oauth/login     --> ...
[GIN-debug] GET    /api/v1/oauth/callback  --> ...
```

### 3. 测试 OAuth 重定向

```bash
curl -I http://localhost:8888/api/v1/oauth/login
```

应该返回 307 重定向，并且 Location 头中的 URL 应该是浏览器可访问的：
```
HTTP/1.1 307 Temporary Redirect
Location: http://localhost:9000/application/o/authorize/?client_id=...
```

## 代码修改记录

### 修改前（复杂版本）

尝试在 `oauth_handler.go` 中添加 URL 替换逻辑：
```go
// 错误做法：在代码中替换 URL
publicURL := cfg.OAuth.Authentik.PublicURL
url := h.oauth2Config.AuthCodeURL(state)
url = strings.Replace(url, internalHost, publicHost, -1)
```

### 修改后（简单版本）

回退到简单实现，让 Authentik 配置处理 URL 问题：
```go
// 正确做法：直接使用 OAuth2 库生成的 URL
url := h.oauth2Config.AuthCodeURL(state)
c.Redirect(http.StatusTemporaryRedirect, url)
```

## 最佳实践

1. **开发环境**：在宿主机运行后端，避免 Docker 网络复杂性
2. **生产环境**：使用反向代理和统一的外部域名
3. **配置优先**：在 Authentik 管理后台正确配置，而不是在代码中修复
4. **保持简单**：OAuth2 库已经处理了大部分逻辑，不要过度设计

## 参考文档

- Authentik 官方文档: https://goauthentik.io/docs/
- OAuth2 规范: https://oauth.net/2/
- 项目配置指南: `OAUTH_SETUP_GUIDE.md`
