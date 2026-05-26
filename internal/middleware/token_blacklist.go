package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBlacklistPrefix = "blacklist:jti:"

// TokenBlacklist 基于 Redis 的 Token 黑名单
type TokenBlacklist struct {
	client *redis.Client
}

// NewTokenBlacklist 创建 Token 黑名单
func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// Add 将 token 加入黑名单
func (b *TokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	if b.client == nil {
		return nil
	}
	key := tokenBlacklistPrefix + jti
	return b.client.Set(ctx, key, "1", ttl).Err()
}

// IsBlacklisted 检查 token 是否在黑名单中
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) bool {
	if b.client == nil {
		return false
	}
	key := tokenBlacklistPrefix + jti
	exists, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// KeyForJTI 生成黑名单 Redis key
func KeyForJTI(jti string) string {
	return tokenBlacklistPrefix + jti
}

// TTL 计算剩余有效期
func TTL(expiresAt time.Time) time.Duration {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return time.Minute
	}
	return remaining
}

// BlacklistError token 已被吊销的错误
var ErrTokenBlacklisted = fmt.Errorf("token has been revoked")
