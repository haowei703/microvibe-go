package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"microvibe-go/internal/config"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/model"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/logger"
	"microvibe-go/pkg/response"
	"microvibe-go/pkg/utils"
)

// OAuthHandler OAuth2/OIDC 认证处理器
type OAuthHandler struct {
	config       *config.Config
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	userService  service.UserService
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler(cfg *config.Config, userService service.UserService) (*OAuthHandler, error) {
	if !cfg.OAuth.Authentik.Enabled {
		return nil, nil
	}

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, cfg.OAuth.Authentik.IssuerURL)
	if err != nil {
		logger.Error("Failed to create OIDC provider", zap.Error(err))
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

	logger.Info("OAuth handler initialized successfully",
		zap.String("issuer", cfg.OAuth.Authentik.IssuerURL),
		zap.String("client_id", cfg.OAuth.Authentik.ClientID))

	return &OAuthHandler{
		config:       cfg,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		userService:  userService,
	}, nil
}

// generateRandomState 生成随机 state 字符串
func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Login 发起 OAuth 登录
// @Summary OAuth 登录
// @Description 重定向到 Authentik 登录页面
// @Tags OAuth
// @Produce json
// @Success 302 {string} string "重定向到 Authentik"
// @Router /oauth/login [get]
func (h *OAuthHandler) Login(c *gin.Context) {
	state := generateRandomState()

	// 保存 state 到 cookie (10 分钟有效)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// 获取平台信息，优先从 URL 参数获取 (支持系统浏览器打开时传递平台信息)
	// 其次从 User-Agent 解析 (用于支持系统浏览器唤起应用)
	platform := c.Query("platform")
	if platform == "" {
		ua := c.GetHeader("User-Agent")
		platform = strings.ToLower(ua)
		if strings.Contains(platform, "android") {
			platform = "android"
		} else if strings.Contains(platform, "iphone") || strings.Contains(platform, "ipad") {
			platform = "ios"
		} else if strings.Contains(platform, "windows") {
			platform = "windows"
		} else if strings.Contains(platform, "macintosh") {
			platform = "macos"
		} else if strings.Contains(platform, "linux") {
			platform = "linux"
		} else {
			device := middleware.GetDeviceInfo(c)
			platform = device.Platform
		}
	}
	c.SetCookie("oauth_platform", platform, 600, "/", "", false, true)

	// 保存前端回调地址（动态端口支持）
	if frontendURL := c.Query("frontend_url"); frontendURL != "" {
		c.SetCookie("oauth_frontend_url", frontendURL, 600, "/", "", false, true)
	}

	// 生成授权 URL 并重定向
	url := h.oauth2Config.AuthCodeURL(state)
	logger.Info("Redirecting to OAuth provider", zap.String("url", url))

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback 处理 OAuth 回调
// @Summary OAuth 回调
// @Description 处理 Authentik OAuth 回调并登录用户
// @Tags OAuth
// @Produce json
// @Param code query string true "授权码"
// @Param state query string true "状态码"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /oauth/callback [get]
func (h *OAuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Error(c, response.CodeInvalidParam, "缺少授权码 (code)")
		return
	}

	// 1. 验证 state
	// 尝试从 cookie 获取状态码（原网页/Web 前端跳转流程）
	savedState, _ := c.Cookie("oauth_state")
	if savedState != "" {
		if c.Query("state") != savedState {
			logger.Warn("State mismatch",
				zap.String("expected", savedState),
				zap.String("got", c.Query("state")))
			response.Error(c, response.CodeUnauthorized, "无效的 state 参数")
			return
		}
		// 校验通过后清除 cookie
		c.SetCookie("oauth_state", "", -1, "/", "", false, true)
	}

	// 获取平台信息 (用于决定返回 JSON 还是重定向)
	platform, _ := c.Cookie("oauth_platform")
	if platform == "" {
		// 如果没有 cookie，从 User-Agent 实时分析
		device := middleware.GetDeviceInfo(c)
		platform = device.Platform
	}
	// 清除平台 cookie
	c.SetCookie("oauth_platform", "", -1, "/", "", false, true)

	// 2. 交换 Token
	// 允许客户端通过 query 传入 redirect_uri (必须与获取 code 时使用的一致)
	// 如果不传，则使用配置中的默认值
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		redirectURI = h.config.OAuth.Authentik.RedirectURL
	}

	// 为本次交换创建临时的 config (如果 redirect_uri 不同)
	tc := h.oauth2Config
	if redirectURI != h.oauth2Config.RedirectURL {
		tc = &oauth2.Config{
			ClientID:     h.oauth2Config.ClientID,
			ClientSecret: h.oauth2Config.ClientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     h.oauth2Config.Endpoint,
			Scopes:       h.oauth2Config.Scopes,
		}
	}

	oauth2Token, err := tc.Exchange(c.Request.Context(), code)
	if err != nil {
		logger.Error("Failed to exchange token", zap.Error(err))
		response.Error(c, response.CodeUnauthorized, "授权码无效或已过期")
		return
	}

	// 3. 验证 ID Token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		logger.Error("No id_token in response")
		response.Error(c, response.CodeError, "响应中没有 id_token")
		return
	}

	idToken, err := h.verifier.Verify(c.Request.Context(), rawIDToken)
	if err != nil {
		logger.Error("Failed to verify ID token", zap.Error(err))
		response.Error(c, response.CodeError, "ID Token 验证失败")
		return
	}

	// 4. 提取用户信息
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Sub           string `json:"sub"` // Authentik 用户 UUID
		Username      string `json:"preferred_username"`
	}

	if err := idToken.Claims(&claims); err != nil {
		logger.Error("Failed to parse claims", zap.Error(err))
		response.Error(c, response.CodeError, "解析用户信息失败")
		return
	}

	logger.Info("OAuth login successful",
		zap.String("email", claims.Email),
		zap.String("name", claims.Name),
		zap.String("username", claims.Username))

	// 5. 查找或创建用户
	user, err := h.findOrCreateUser(c.Request.Context(), &claims)
	if err != nil {
		logger.Error("Failed to find or create user", zap.Error(err))
		response.Error(c, response.CodeError, "创建用户失败")
		return
	}

	// 6. 生成 JWT Token
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, h.config.JWT.Secret, h.config.JWT.Expire)
	if err != nil {
		logger.Error("Failed to generate JWT token", zap.Error(err))
		response.Error(c, response.CodeError, "生成 Token 失败")
		return
	}

	// 7. 返回结果
	// 如果是原生平台 (Mobile/Desktop)，重定向到自定义协议以支持客户端 Deep Link 唤回
	// 注意：如果客户端明确通过 AJAX (X-Requested-With) 调用，则跳过重定向返回 JSON
	device := middleware.DeviceInfo{Platform: platform}
	isAjax := c.GetHeader("X-Requested-With") == "XMLHttpRequest"

	if device.IsNative() && !isAjax {
		// 生成深层链接 URL: microvibe://auth?token=xxx&user_id=xxx
		deepLink := fmt.Sprintf("microvibe://auth/callback?token=%s&user_id=%d", token, user.ID)
		logger.Info("Redirecting native user to deep link", zap.String("url", deepLink))
		c.Redirect(http.StatusTemporaryRedirect, deepLink)
		return
	}

	// 如果是 Web 端且配置了前端 URL (且不是 AJAX 请求)，重定向回前端
	// 优先使用 cookie 中的前端地址（支持动态端口），其次使用配置
	frontendURL := h.config.OAuth.Authentik.FrontendURL
	if cookieURL, err := c.Cookie("oauth_frontend_url"); err == nil && cookieURL != "" {
		frontendURL = cookieURL
	}
	if platform == "web" && !isAjax && frontendURL != "" {
		redirectURL := fmt.Sprintf("%s?token=%s&user_id=%d",
			strings.TrimSuffix(frontendURL, "/"),
			token, user.ID)
		logger.Info("Redirecting web user to frontend", zap.String("url", redirectURL))
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	// 否则直接返回 JSON (适用于现代客户端自主换取 Token 的场景)
	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
		},
	})
}

// findOrCreateUser 查找或创建 OAuth 用户
func (h *OAuthHandler) findOrCreateUser(ctx context.Context, claims *struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
	Username      string `json:"preferred_username"`
}) (*model.User, error) {
	// 首先尝试通过 email 查找用户
	user, err := h.userService.GetUserByEmail(ctx, claims.Email)
	if err == nil && user != nil {
		// 用户已存在
		logger.Info("Existing user logged in via OAuth", zap.Uint("user_id", user.ID))
		return user, nil
	}

	// 用户不存在，使用 Register 方法创建
	username := claims.Username
	if username == "" {
		// 使用 email 的本地部分作为 username
		parts := strings.Split(claims.Email, "@")
		if len(parts) > 0 {
			username = parts[0]
		} else {
			username = claims.Email
		}
	}

	nickname := claims.Name
	if nickname == "" {
		nickname = username
	}

	// 使用 Register 方法创建新用户
	registerReq := &service.RegisterRequest{
		Username: username,
		Email:    claims.Email,
		Nickname: nickname,
		Password: utils.GenerateRandomPassword(16), // OAuth 用户生成随机密码（不会被使用）
	}

	newUser, err := h.userService.Register(ctx, registerReq)
	if err != nil {
		return nil, err
	}

	logger.Info("New user created via OAuth",
		zap.Uint("user_id", newUser.ID),
		zap.String("email", newUser.Email))

	return newUser, nil
}
