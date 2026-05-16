package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理跨域请求
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			// 在开发模式下，动态允许请求源，以解决携带 Credentials 时的限制
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// 允许的 HTTP 方法
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		// 允许的请求头
		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Platform, X-App-Version, X-OS-Version, X-Device-Model")
		}

		// 允许携带认证信息（cookies/JWT in header/etc.）
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 预检请求的缓存时间（秒）
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// 允许前端访问的响应头
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		// 处理 OPTIONS 预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
