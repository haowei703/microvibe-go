package middleware

import (
	"microvibe-go/internal/config"
	"microvibe-go/pkg/response"
	"microvibe-go/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		// Token 格式：Bearer {token}
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Token 格式错误")
			c.Abort()
			return
		}

		// 解析 token
		claims, err := utils.ParseToken(parts[1], cfg.JWT.Secret)
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("uid", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求登录）
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := utils.ParseToken(parts[1], cfg.JWT.Secret)
				if err == nil {
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
				}
			}
		}
		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("uid")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}
