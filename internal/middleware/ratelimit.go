package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const rlPrefix = "ratelimit:"

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local count = redis.call("INCR", key)
if count == 1 then
    redis.call("EXPIRE", key, window)
end

local ttl = redis.call("TTL", key)
if ttl < 0 then ttl = window end

if count > limit then
    return {0, ttl}
end
return {limit - count, ttl}
`)

// RateLimiter 基于 Redis 滑动窗口的速率限制器
type RateLimiter struct {
	client *redis.Client
	enabled bool
}

// RateLimitConfig 速率限制参数
type RateLimitConfig struct {
	RequestsPerSecond int
	Burst             int
	WindowSeconds     int
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(client *redis.Client, enabled bool) *RateLimiter {
	return &RateLimiter{client: client, enabled: enabled}
}

// Middleware 返回 Gin 速率限制中间件
func (rl *RateLimiter) Middleware(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.enabled || rl.client == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s%s:%s", rlPrefix, c.FullPath(), clientIP)

		now := time.Now().Unix()
		window := cfg.WindowSeconds
		if window <= 0 {
			window = 1
		}
		limit := cfg.RequestsPerSecond * window
		burst := cfg.Burst
		if burst > limit {
			limit = burst
		}

		result, err := rateLimitScript.Run(c.Request.Context(), rl.client, []string{key}, limit, window, now).Slice()
		if err != nil {
			c.Next()
			return
		}

		remaining, _ := result[0].(int64)
		ttl, _ := result[1].(int64)

		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(now+ttl, 10))

		if remaining == 0 {
			rateLimitHitsTotal.WithLabelValues(c.FullPath()).Inc()
			c.Header("Retry-After", strconv.FormatInt(ttl, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": fmt.Sprintf("请求过于频繁，请在 %d 秒后重试", ttl),
			})
			return
		}

		c.Next()
	}
}

// AuthRateLimit 认证接口限流配置
func AuthRateLimit() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 5,
		Burst:             10,
		WindowSeconds:     1,
	}
}

// UploadRateLimit 上传接口限流配置
func UploadRateLimit() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 3,
		Burst:             5,
		WindowSeconds:     1,
	}
}

// SearchRateLimit 搜索接口限流配置
func SearchRateLimit() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 20,
		Burst:             50,
		WindowSeconds:     1,
	}
}

// DefaultRateLimit 默认限流配置
func DefaultRateLimit() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             200,
		WindowSeconds:     1,
	}
}
