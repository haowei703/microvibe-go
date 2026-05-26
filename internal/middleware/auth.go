package middleware

import (
	"microvibe-go/internal/config"
	"microvibe-go/pkg/response"
	"microvibe-go/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AuthMiddleware JWT 认证中间件
func AuthMiddleware(cfg *config.Config, blacklist *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authFailuresTotal.WithLabelValues(c.FullPath(), "missing_header").Inc()
			response.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			authFailuresTotal.WithLabelValues(c.FullPath(), "malformed_header").Inc()
			response.Unauthorized(c, "Token 格式错误")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1], cfg.JWT.Secret)
		if err != nil {
			tokenValidationFailures.WithLabelValues("parse_error").Inc()
			authFailuresTotal.WithLabelValues(c.FullPath(), "invalid_token").Inc()
			response.Unauthorized(c, "Token 无效或已过期")
			c.Abort()
			return
		}

		if blacklist != nil && blacklist.IsBlacklisted(c.Request.Context(), claims.JTI) {
			authFailuresTotal.WithLabelValues(c.FullPath(), "blacklisted").Inc()
			response.Unauthorized(c, "Token 已失效，请重新登录")
			c.Abort()
			return
		}

		c.Set("uid", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("jti", claims.JTI)
		c.Set("claims", claims)

		c.Next()
	}
}

// AdminMiddleware 管理员认证中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(int8) != 1 {
			authFailuresTotal.WithLabelValues(c.FullPath(), "insufficient_role").Inc()
			response.Forbidden(c, "权限不足，仅限管理员访问")
			c.Abort()
			return
		}
		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求登录）
func OptionalAuthMiddleware(cfg *config.Config, blacklist *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := utils.ParseToken(parts[1], cfg.JWT.Secret)
				if err == nil {
					if blacklist == nil || !blacklist.IsBlacklisted(c.Request.Context(), claims.JTI) {
						c.Set("uid", claims.UserID)
						c.Set("username", claims.Username)
						c.Set("role", claims.Role)
						c.Set("jti", claims.JTI)
						c.Set("claims", claims)
					}
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

// GetTokenBlacklist 从配置和 Redis 客户端创建 Token 黑名单
func GetTokenBlacklist(client *redis.Client) *TokenBlacklist {
	if client == nil {
		return nil
	}
	return NewTokenBlacklist(client)
}
