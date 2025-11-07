# Authentik SSO 集成指南

本文档描述如何将 Authentik 开源认证框架集成到 MicroVibe 项目中，实现单点登录（SSO）、OAuth2/OIDC 等高级认证功能。

## 目录

- [1. Authentik 简介](#1-authentik-简介)
- [2. 快速开始](#2-快速开始)
- [3. Authentik 配置](#3-authentik-配置)
- [4. 后端集成](#4-后端集成)
- [5. 前端集成](#5-前端集成)
- [6. 高级功能](#6-高级功能)

## 1. Authentik 简介

**Authentik** 是一个开源的身份提供商（IdP），支持：

- ✅ **单点登录（SSO）**：一次登录，多处使用
- ✅ **OAuth2/OIDC**：标准化的认证协议
- ✅ **SAML 2.0**：企业级认证
- ✅ **LDAP**：目录服务
- ✅ **多因素认证（MFA）**：增强安全性
- ✅ **社交登录**：Google、GitHub、微信等
- ✅ **用户管理**：完整的用户生命周期管理
- ✅ **策略引擎**：灵活的访问控制

## 2. 快速开始

### 2.1 启动 Authentik 服务器

```bash
cd /Users/ai6677/dev/coding/golang/microvibe-go

# 启动 Authentik
./scripts/start-authentik.sh
```

### 2.2 初始化管理员账号

1. 访问：http://localhost:9000/if/flow/initial-setup/
2. 设置管理员邮箱和密码（建议使用强密码）
3. 登录管理后台：http://localhost:9000/if/admin/

### 2.3 验证服务状态

```bash
# 查看容器状态
docker-compose -f docker-compose.authentik.yml ps

# 查看日志
docker-compose -f docker-compose.authentik.yml logs -f authentik-server
```

## 3. Authentik 配置

### 3.1 创建 OAuth2/OIDC 提供者

1. 登录 Authentik 管理后台
2. 导航到 **Applications** → **Providers** → **Create**
3. 选择 **OAuth2/OpenID Connect**
4. 配置如下：

   ```
   Name: MicroVibe Backend
   Authorization flow: default-provider-authorization-implicit-consent
   Client type: Confidential
   Client ID: microvibe-backend
   Redirect URIs:
     - http://localhost:8080/auth/callback
     - http://localhost:8888/auth/callback
   Signing Key: (选择默认)
   ```

5. 保存后，记录 **Client ID** 和 **Client Secret**

### 3.2 创建应用

1. 导航到 **Applications** → **Applications** → **Create**
2. 配置如下：

   ```
   Name: MicroVibe
   Slug: microvibe
   Provider: MicroVibe Backend (选择刚创建的)
   Launch URL: http://localhost:8888/
   ```

3. 保存

### 3.3 配置用户属性映射

1. 导航到 **Customization** → **Property Mappings**
2. 确保以下映射存在（默认已创建）：
   - `openid` scope mappings
   - `email` scope mappings
   - `profile` scope mappings

## 4. 后端集成

### 4.1 安装依赖

```bash
cd /Users/ai6677/dev/coding/golang/microvibe-go

# 安装 OAuth2 库
go get golang.org/x/oauth2
go get github.com/coreos/go-oidc/v3/oidc
```

### 4.2 配置文件

更新 `configs/config.yaml`：

```yaml
# OAuth2/OIDC 配置
oauth:
  authentik:
    enabled: true
    issuer_url: "http://localhost:9000/application/o/microvibe/"
    client_id: "microvibe-backend"
    client_secret: "your-client-secret-here"
    redirect_url: "http://localhost:8888/auth/callback"
    scopes:
      - openid
      - email
      - profile
```

### 4.3 创建 OAuth 配置结构

创建 `internal/config/oauth.go`：

```go
package config

type OAuthConfig struct {
    Authentik AuthentikConfig `yaml:"authentik" mapstructure:"authentik"`
}

type AuthentikConfig struct {
    Enabled      bool     `yaml:"enabled" mapstructure:"enabled"`
    IssuerURL    string   `yaml:"issuer_url" mapstructure:"issuer_url"`
    ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
    ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret"`
    RedirectURL  string   `yaml:"redirect_url" mapstructure:"redirect_url"`
    Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
}
```

### 4.4 实现 OAuth Handler

创建 `internal/handler/oauth_handler.go`：

```go
package handler

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/coreos/go-oidc/v3/oidc"
    "github.com/gin-gonic/gin"
    "golang.org/x/oauth2"

    "microvibe-go/internal/config"
    "microvibe-go/pkg/logger"
    "microvibe-go/pkg/response"
)

type OAuthHandler struct {
    config       *config.Config
    oauth2Config *oauth2.Config
    verifier     *oidc.IDTokenVerifier
}

func NewOAuthHandler(cfg *config.Config) (*OAuthHandler, error) {
    if !cfg.OAuth.Authentik.Enabled {
        return nil, nil
    }

    ctx := context.Background()

    provider, err := oidc.NewProvider(ctx, cfg.OAuth.Authentik.IssuerURL)
    if err != nil {
        return nil, err
    }

    oauth2Config := &oauth2.Config{
        ClientID:     cfg.OAuth.Authentik.ClientID,
        ClientSecret: cfg.OAuth.Authentik.ClientSecret,
        RedirectURL:  cfg.OAuth.Authentik.RedirectURL,
        Endpoint:     provider.Endpoint(),
        Scopes:       cfg.OAuth.Authentik.Scopes,
    }

    verifier := provider.Verifier(&oidc.Config{
        ClientID: cfg.OAuth.Authentik.ClientID,
    })

    return &OAuthHandler{
        config:       cfg,
        oauth2Config: oauth2Config,
        verifier:     verifier,
    }, nil
}

// Login 发起 OAuth 登录
func (h *OAuthHandler) Login(c *gin.Context) {
    state := generateRandomState() // 实现随机 state 生成

    // 保存 state 到 session 或 Redis
    c.SetCookie("oauth_state", state, 3600, "/", "", false, true)

    url := h.oauth2Config.AuthCodeURL(state)
    c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback 处理 OAuth 回调
func (h *OAuthHandler) Callback(c *gin.Context) {
    // 验证 state
    savedState, _ := c.Cookie("oauth_state")
    if c.Query("state") != savedState {
        response.Unauthorized(c, "Invalid state")
        return
    }

    // 交换授权码
    oauth2Token, err := h.oauth2Config.Exchange(c.Request.Context(), c.Query("code"))
    if err != nil {
        logger.Error("Failed to exchange token", zap.Error(err))
        response.Error(c, response.CodeError, "认证失败")
        return
    }

    // 验证 ID Token
    rawIDToken, ok := oauth2Token.Extra("id_token").(string)
    if !ok {
        response.Error(c, response.CodeError, "No id_token")
        return
    }

    idToken, err := h.verifier.Verify(c.Request.Context(), rawIDToken)
    if err != nil {
        logger.Error("Failed to verify ID token", zap.Error(err))
        response.Error(c, response.CodeError, "Token 验证失败")
        return
    }

    // 提取用户信息
    var claims struct {
        Email         string `json:"email"`
        EmailVerified bool   `json:"email_verified"`
        Name          string `json:"name"`
        Sub           string `json:"sub"`
    }

    if err := idToken.Claims(&claims); err != nil {
        response.Error(c, response.CodeError, "解析用户信息失败")
        return
    }

    // 创建或更新用户（调用 UserService）
    // ...

    // 生成 JWT Token
    // ...

    response.Success(c, gin.H{
        "token": "generated-jwt-token",
        "user":  claims,
    })
}
```

### 4.5 注册路由

在 `internal/router/router.go` 中添加：

```go
// OAuth 认证路由
if oauthHandler, err := handler.NewOAuthHandler(cfg); err == nil && oauthHandler != nil {
    oauth := v1.Group("/oauth")
    {
        oauth.GET("/login", oauthHandler.Login)
        oauth.GET("/callback", oauthHandler.Callback)
    }
}
```

## 5. 前端集成

### 5.1 添加 Authentik 登录按钮

更新 `packages/web/pages/login.vue`：

```vue
<template>
  <!-- ... 现有代码 ... -->

  <!-- 其他登录方式 -->
  <div class="mt-8 pt-6 border-t border-gray-100">
    <p class="text-center text-xs text-gray-400 mb-4">其他登录方式</p>
    <div class="flex justify-center gap-4">
      <!-- Authentik SSO 登录 -->
      <button
        class="w-10 h-10 rounded-full bg-gradient-to-r from-pink-500 to-purple-500 hover:from-pink-600 hover:to-purple-600 transition-colors flex items-center justify-center text-white shadow-lg"
        @click="handleAuthentikLogin"
        title="Authentik SSO 登录"
      >
        <span class="text-xl">🔐</span>
      </button>

      <!-- 其他登录方式 -->
      <button class="w-10 h-10 rounded-full bg-gray-100 hover:bg-gray-200 transition-colors flex items-center justify-center">
        <span class="text-xl">📱</span>
      </button>
      <button class="w-10 h-10 rounded-full bg-gray-100 hover:bg-gray-200 transition-colors flex items-center justify-center">
        <span class="text-xl">💬</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
// ... 现有代码 ...

function handleAuthentikLogin() {
  // 跳转到后端 OAuth 登录端点
  window.location.href = 'http://localhost:8888/api/v1/oauth/login'
}
</script>
```

### 5.2 处理 OAuth 回调

创建回调页面 `packages/web/pages/auth/callback.vue`：

```vue
<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-pink-50 via-blue-50 to-purple-50">
    <div class="text-center">
      <div class="animate-spin rounded-full h-16 w-16 border-b-2 border-pink-500 mx-auto mb-4"></div>
      <p class="text-gray-600">正在登录...</p>
    </div>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const authStore = useAuthStore()

onMounted(async () => {
  const { code, state } = route.query

  if (code) {
    try {
      // 后端已经处理了回调，直接从 localStorage 读取 token
      const token = localStorage.getItem('auth_token')
      if (token) {
        // 获取用户信息
        await authStore.fetchCurrentUser()
        // 跳转到首页
        navigateTo('/')
      } else {
        throw new Error('未获取到 token')
      }
    } catch (error) {
      console.error('登录失败:', error)
      navigateTo('/login?error=auth_failed')
    }
  } else {
    navigateTo('/login?error=invalid_callback')
  }
})
</script>
```

## 6. 高级功能

### 6.1 社交登录集成

在 Authentik 中配置社交登录：

1. 导航到 **Directory** → **Federation & Social login** → **Create**
2. 选择提供商（Google、GitHub、微信等）
3. 配置相应的 Client ID 和 Secret

### 6.2 多因素认证（MFA）

1. 导航到 **Flows & Stages** → **Stages** → **Create**
2. 选择 **Authenticator Validation Stage**
3. 配置 TOTP、WebAuthn 等方式

### 6.3 用户自助注册

1. 导航到 **Flows & Stages** → **Flows**
2. 编辑 **default-enrollment-flow**
3. 添加注册字段和验证规则

### 6.4 访问策略

1. 导航到 **Applications** → 选择应用
2. 编辑 **Policy Bindings**
3. 添加访问控制策略（IP 限制、用户组等）

## 7. 故障排查

### 7.1 查看日志

```bash
# Authentik 服务器日志
docker logs -f microvibe-authentik-server

# Authentik Worker 日志
docker logs -f microvibe-authentik-worker
```

### 7.2 常见问题

**Q: 无法访问 Authentik 管理界面**
A: 确保端口 9000 没有被占用，检查防火墙设置

**Q: OAuth 回调失败**
A: 确认 Redirect URI 配置正确，必须完全匹配（包括协议、端口）

**Q: Token 验证失败**
A: 检查 `issuer_url` 是否正确，确保可以访问 `/.well-known/openid-configuration`

## 8. 生产环境部署

### 8.1 安全建议

1. 修改默认的 `AUTHENTIK_SECRET_KEY`
2. 使用 HTTPS（配置反向代理）
3. 定期备份 PostgreSQL 数据库
4. 启用速率限制
5. 配置 SMTP 邮件服务

### 8.2 性能优化

1. 增加 Redis 内存限制
2. 配置数据库连接池
3. 启用 CDN 加速静态资源
4. 使用专用数据库服务器

## 9. 参考资料

- [Authentik 官方文档](https://goauthentik.io/docs/)
- [OAuth 2.0 规范](https://oauth.net/2/)
- [OpenID Connect 规范](https://openid.net/connect/)
- [Go OAuth2 库文档](https://pkg.go.dev/golang.org/x/oauth2)
