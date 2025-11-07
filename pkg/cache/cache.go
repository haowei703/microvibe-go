package cache

import (
	"context"
	"errors"
	"time"
)

// Cache 泛型缓存接口 - 策略模式核心
type Cache[T any] interface {
	// Get 获取缓存（类型安全）
	Get(ctx context.Context, key string) (T, error)

	// Set 设置缓存
	Set(ctx context.Context, key string, value T, ttl time.Duration) error

	// GetOrSet 获取或设置缓存（缓存穿透保护）
	GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error)

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// GetMulti 批量获取
	GetMulti(ctx context.Context, keys []string) (map[string]T, error)

	// SetMulti 批量设置
	SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error

	// DeleteMulti 批量删除
	DeleteMulti(ctx context.Context, keys []string) error

	// GetWithTTL 获取缓存及剩余 TTL
	GetWithTTL(ctx context.Context, key string) (T, time.Duration, error)

	// Refresh 刷新缓存过期时间
	Refresh(ctx context.Context, key string, ttl time.Duration) error

	// GetStats 获取缓存统计信息
	GetStats() *Stats

	// Close 关闭缓存
	Close() error
}

// Stats 缓存统计信息
type Stats struct {
	Hits        int64   `json:"hits"`         // 命中次数
	Misses      int64   `json:"misses"`       // 未命中次数
	Sets        int64   `json:"sets"`         // 设置次数
	Deletes     int64   `json:"deletes"`      // 删除次数
	Evictions   int64   `json:"evictions"`    // 淘汰次数
	HitRate     float64 `json:"hit_rate"`     // 命中率
	ItemCount   int64   `json:"item_count"`   // 当前缓存项数量
	MemoryUsage int64   `json:"memory_usage"` // 内存占用（字节）
}

// CalculateHitRate 计算命中率
func (s *Stats) CalculateHitRate() {
	total := s.Hits + s.Misses
	if total > 0 {
		s.HitRate = float64(s.Hits) / float64(total)
	}
}

// 常见错误
var (
	ErrCacheMiss    = errors.New("cache: key not found")
	ErrCacheExpired = errors.New("cache: key expired")
	ErrInvalidKey   = errors.New("cache: invalid key")
	ErrInvalidValue = errors.New("cache: invalid value")
	ErrCacheClosed  = errors.New("cache: cache is closed")
)

// CacheError 缓存错误包装
type CacheError struct {
	Op  string // 操作
	Key string // 键
	Err error  // 原始错误
}

func (e *CacheError) Error() string {
	if e.Key != "" {
		return "cache " + e.Op + " [" + e.Key + "]: " + e.Err.Error()
	}
	return "cache " + e.Op + ": " + e.Err.Error()
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

func (e *CacheError) IsCacheMiss() bool {
	return errors.Is(e.Err, ErrCacheMiss)
}
