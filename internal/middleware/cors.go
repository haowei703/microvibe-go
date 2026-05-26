package middleware

import (
	"slices"

	"microvibe-go/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理跨域请求，基于白名单校验 Origin
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := cfg.CORS.AllowedOrigins

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			if slices.Contains(allowedOrigins, origin) {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Platform, X-App-Version, X-OS-Version, X-Device-Model")
		}

		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
