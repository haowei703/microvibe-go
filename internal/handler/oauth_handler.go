package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"microvibe-go/internal/config"
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
	// 1. 验证 state
	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState == "" {
		logger.Warn("Missing oauth_state cookie")
		response.Error(c, response.CodeUnauthorized, "无效的登录请求")
		return
	}

	if c.Query("state") != savedState {
		logger.Warn("State mismatch",
			zap.String("expected", savedState),
			zap.String("got", c.Query("state")))
		response.Error(c, response.CodeUnauthorized, "无效的 state 参数")
		return
	}

	// 清除 state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// 2. 交换授权码获取 Token
	code := c.Query("code")
	if code == "" {
		response.Error(c, response.CodeInvalidParam, "缺少授权码")
		return
	}

	oauth2Token, err := h.oauth2Config.Exchange(c.Request.Context(), code)
	if err != nil {
		logger.Error("Failed to exchange token", zap.Error(err))
		response.Error(c, response.CodeError, "Token 交换失败")
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
	token, err := utils.GenerateToken(user.ID, user.Username, h.config.JWT.Secret, h.config.JWT.Expire)
	if err != nil {
		logger.Error("Failed to generate JWT token", zap.Error(err))
		response.Error(c, response.CodeError, "生成 Token 失败")
		return
	}

	// 7. 返回结果（前端将保存到 localStorage）
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
