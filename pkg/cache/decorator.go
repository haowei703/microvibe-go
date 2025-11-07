package cache

import (
	"context"
	"errors"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
)

// LoggingDecorator 日志装饰器 - 装饰器模式
type LoggingDecorator[T any] struct {
	cache  Cache[T]
	logger bool
}

// NewLoggingDecorator 创建日志装饰器
func NewLoggingDecorator[T any](cache Cache[T]) Cache[T] {
	return &LoggingDecorator[T]{
		cache:  cache,
		logger: true,
	}
}

// Get 获取缓存（记录日志）
func (ld *LoggingDecorator[T]) Get(ctx context.Context, key string) (T, error) {
	start := time.Now()
	value, err := ld.cache.Get(ctx, key)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			logger.Debug("缓存未命中",
				zap.String("key", key),
				zap.Duration("elapsed", elapsed))
		} else {
			logger.Error("缓存获取失败",
				zap.String("key", key),
				zap.Error(err),
				zap.Duration("elapsed", elapsed))
		}
	} else {
		logger.Debug("缓存命中",
			zap.String("key", key),
			zap.Duration("elapsed", elapsed))
	}

	return value, err
}

// Set 设置缓存（记录日志）
func (ld *LoggingDecorator[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	start := time.Now()
	err := ld.cache.Set(ctx, key, value, ttl)
	elapsed := time.Since(start)

	if err != nil {
		logger.Error("缓存设置失败",
			zap.String("key", key),
			zap.Error(err),
			zap.Duration("ttl", ttl),
			zap.Duration("elapsed", elapsed))
	} else {
		logger.Debug("缓存设置成功",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
			zap.Duration("elapsed", elapsed))
	}

	return err
}

// GetOrSet 获取或设置缓存
func (ld *LoggingDecorator[T]) GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error) {
	start := time.Now()
	value, err := ld.cache.GetOrSet(ctx, key, loader, ttl)
	elapsed := time.Since(start)

	if err != nil {
		logger.Error("缓存GetOrSet失败",
			zap.String("key", key),
			zap.Error(err),
			zap.Duration("elapsed", elapsed))
	} else {
		logger.Debug("缓存GetOrSet成功",
			zap.String("key", key),
			zap.Duration("elapsed", elapsed))
	}

	return value, err
}

// Delete 删除缓存
func (ld *LoggingDecorator[T]) Delete(ctx context.Context, key string) error {
	err := ld.cache.Delete(ctx, key)
	if err != nil {
		logger.Error("缓存删除失败", zap.String("key", key), zap.Error(err))
	} else {
		logger.Debug("缓存删除成功", zap.String("key", key))
	}
	return err
}

// Exists 检查缓存是否存在
func (ld *LoggingDecorator[T]) Exists(ctx context.Context, key string) (bool, error) {
	return ld.cache.Exists(ctx, key)
}

// Clear 清空所有缓存
func (ld *LoggingDecorator[T]) Clear(ctx context.Context) error {
	err := ld.cache.Clear(ctx)
	if err != nil {
		logger.Error("缓存清空失败", zap.Error(err))
	} else {
		logger.Info("缓存已清空")
	}
	return err
}

// GetMulti 批量获取
func (ld *LoggingDecorator[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	start := time.Now()
	result, err := ld.cache.GetMulti(ctx, keys)
	elapsed := time.Since(start)

	logger.Debug("批量获取缓存",
		zap.Int("requested", len(keys)),
		zap.Int("found", len(result)),
		zap.Duration("elapsed", elapsed))

	return result, err
}

// SetMulti 批量设置
func (ld *LoggingDecorator[T]) SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error {
	start := time.Now()
	err := ld.cache.SetMulti(ctx, items, ttl)
	elapsed := time.Since(start)

	if err != nil {
		logger.Error("批量设置缓存失败",
			zap.Int("count", len(items)),
			zap.Error(err),
			zap.Duration("elapsed", elapsed))
	} else {
		logger.Debug("批量设置缓存成功",
			zap.Int("count", len(items)),
			zap.Duration("elapsed", elapsed))
	}

	return err
}

// DeleteMulti 批量删除
func (ld *LoggingDecorator[T]) DeleteMulti(ctx context.Context, keys []string) error {
	err := ld.cache.DeleteMulti(ctx, keys)
	if err != nil {
		logger.Error("批量删除缓存失败", zap.Int("count", len(keys)), zap.Error(err))
	} else {
		logger.Debug("批量删除缓存成功", zap.Int("count", len(keys)))
	}
	return err
}

// GetWithTTL 获取缓存及剩余 TTL
func (ld *LoggingDecorator[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	return ld.cache.GetWithTTL(ctx, key)
}

// Refresh 刷新缓存过期时间
func (ld *LoggingDecorator[T]) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return ld.cache.Refresh(ctx, key, ttl)
}

// GetStats 获取统计信息
func (ld *LoggingDecorator[T]) GetStats() *Stats {
	return ld.cache.GetStats()
}

// Close 关闭缓存
func (ld *LoggingDecorator[T]) Close() error {
	logger.Info("关闭缓存")
	return ld.cache.Close()
}

// MetricsDecorator 监控装饰器（用于 Prometheus 等监控系统）
type MetricsDecorator[T any] struct {
	cache     Cache[T]
	namespace string
}

// NewMetricsDecorator 创建监控装饰器
func NewMetricsDecorator[T any](cache Cache[T], namespace string) Cache[T] {
	return &MetricsDecorator[T]{
		cache:     cache,
		namespace: namespace,
	}
}

// Get 获取缓存（记录指标）
func (md *MetricsDecorator[T]) Get(ctx context.Context, key string) (T, error) {
	start := time.Now()
	value, err := md.cache.Get(ctx, key)
	elapsed := time.Since(start)

	// 这里可以集成 Prometheus 等监控系统
	// prometheus.CacheGetDuration.WithLabelValues(md.namespace).Observe(elapsed.Seconds())
	// if err == nil {
	//     prometheus.CacheHits.WithLabelValues(md.namespace).Inc()
	// } else {
	//     prometheus.CacheMisses.WithLabelValues(md.namespace).Inc()
	// }

	_ = elapsed // 避免未使用警告

	return value, err
}

// Set 设置缓存
func (md *MetricsDecorator[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	start := time.Now()
	err := md.cache.Set(ctx, key, value, ttl)
	elapsed := time.Since(start)

	// prometheus.CacheSetDuration.WithLabelValues(md.namespace).Observe(elapsed.Seconds())
	_ = elapsed

	return err
}

// GetOrSet 获取或设置缓存
func (md *MetricsDecorator[T]) GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error) {
	return md.cache.GetOrSet(ctx, key, loader, ttl)
}

// Delete 删除缓存
func (md *MetricsDecorator[T]) Delete(ctx context.Context, key string) error {
	return md.cache.Delete(ctx, key)
}

// Exists 检查缓存是否存在
func (md *MetricsDecorator[T]) Exists(ctx context.Context, key string) (bool, error) {
	return md.cache.Exists(ctx, key)
}

// Clear 清空所有缓存
func (md *MetricsDecorator[T]) Clear(ctx context.Context) error {
	return md.cache.Clear(ctx)
}

// GetMulti 批量获取
func (md *MetricsDecorator[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	return md.cache.GetMulti(ctx, keys)
}

// SetMulti 批量设置
func (md *MetricsDecorator[T]) SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error {
	return md.cache.SetMulti(ctx, items, ttl)
}

// DeleteMulti 批量删除
func (md *MetricsDecorator[T]) DeleteMulti(ctx context.Context, keys []string) error {
	return md.cache.DeleteMulti(ctx, keys)
}

// GetWithTTL 获取缓存及剩余 TTL
func (md *MetricsDecorator[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	return md.cache.GetWithTTL(ctx, key)
}

// Refresh 刷新缓存过期时间
func (md *MetricsDecorator[T]) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return md.cache.Refresh(ctx, key, ttl)
}

// GetStats 获取统计信息
func (md *MetricsDecorator[T]) GetStats() *Stats {
	return md.cache.GetStats()
}

// Close 关闭缓存
func (md *MetricsDecorator[T]) Close() error {
	return md.cache.Close()
}

// ========================================
// 方法级装饰器 - 类似 Spring Cache 注解
// ========================================

// CacheConfig 缓存配置
type CacheConfig struct {
	CacheName    string        // 缓存名称
	KeyPrefix    string        // 缓存键前缀
	TTL          time.Duration // 过期时间
	KeyGenerator KeyGenerator  // 自定义键生成器（可选，默认使用 DefaultKeyGenerator）
}

// WithCache 为单返回值函数添加缓存支持（类似 Spring @Cacheable）
// 这是最简洁的装饰器，自动处理缓存逻辑
//
// 使用示例:
//
//	func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
//	    return cache.WithCache[*model.User](
//	        cache.CacheConfig{
//	            CacheName: "user",
//	            KeyPrefix: "user:id",
//	            TTL:       10 * time.Minute,
//	        },
//	        func() (*model.User, error) {
//	            var user model.User
//	            if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
//	                return nil, err
//	            }
//	            return &user, nil
//	        },
//	    )(ctx, id)
//	}
func WithCache[T any](
	config CacheConfig,
	loader func() (T, error),
) func(ctx context.Context, args ...interface{}) (T, error) {
	return func(ctx context.Context, args ...interface{}) (T, error) {
		// 使用自定义或默认的 KeyGenerator 生成缓存键
		keygen := getKeyGenerator(config.KeyGenerator)
		key := keygen(config.KeyPrefix, args...)

		// 获取缓存实例
		c, err := GetTyped[T](config.CacheName)
		if err != nil {
			// 缓存实例不存在，直接执行原函数
			logger.Warn("缓存实例不存在，降级到直接查询", zap.String("cache", config.CacheName))
			return loader()
		}

		// 使用 GetOrSet 自动处理缓存逻辑
		return c.GetOrSet(ctx, key, loader, config.TTL)
	}
}

// WithCacheEvict 为函数添加缓存清除支持（类似 Spring @CacheEvict）
//
// 使用示例:
//
//	func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
//	    return cache.WithCacheEvict(
//	        cache.CacheConfig{
//	            CacheName: "user",
//	            KeyPrefix: "user:id",
//	        },
//	        func() error {
//	            return r.db.WithContext(ctx).Save(user).Error
//	        },
//	    )(ctx, user.ID)
//	}
func WithCacheEvict(
	config CacheConfig,
	fn func() error,
) func(ctx context.Context, args ...interface{}) error {
	return func(ctx context.Context, args ...interface{}) error {
		// 执行原函数
		if err := fn(); err != nil {
			return err
		}

		// 使用自定义或默认的 KeyGenerator 生成缓存键
		keygen := getKeyGenerator(config.KeyGenerator)
		key := keygen(config.KeyPrefix, args...)

		// 记录异步操作开始
		manager := GetManager()
		manager.AddAsyncOp()

		// 异步清除缓存
		go func() {
			defer manager.DoneAsyncOp() // 操作完成时通知

			evictCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			c, err := manager.Get(config.CacheName)
			if err != nil {
				return
			}

			// 尝试删除缓存
			if deleter, ok := c.(interface {
				Delete(context.Context, string) error
			}); ok {
				deleter.Delete(evictCtx, key)
				logger.Debug("缓存已清除", zap.String("cache", config.CacheName), zap.String("key", key))
			}
		}()

		return nil
	}
}

// WithMultiCacheEvict 清除多个缓存键（类似 Spring @Caching）
//
// 使用示例:
//
//	func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
//	    keys := []string{
//	        fmt.Sprintf("user:id:%d", user.ID),
//	        fmt.Sprintf("user:username:%s", user.Username),
//	        fmt.Sprintf("user:email:%s", user.Email),
//	    }
//	    return cache.WithMultiCacheEvict("user", keys, func() error {
//	        return r.db.WithContext(ctx).Save(user).Error
//	    })(ctx)
//	}
func WithMultiCacheEvict(
	cacheName string,
	keys []string,
	fn func() error,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// 执行原函数
		if err := fn(); err != nil {
			return err
		}

		// 记录异步操作开始
		manager := GetManager()
		manager.AddAsyncOp()

		// 异步清除多个缓存
		go func() {
			defer manager.DoneAsyncOp() // 操作完成时通知

			evictCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			c, err := manager.Get(cacheName)
			if err != nil {
				return
			}

			// 尝试批量删除
			if deleter, ok := c.(interface {
				DeleteMulti(context.Context, []string) error
			}); ok {
				deleter.DeleteMulti(evictCtx, keys)
				logger.Debug("批量缓存已清除", zap.String("cache", cacheName), zap.Int("count", len(keys)))
				return
			}

			// 逐个删除
			for _, key := range keys {
				if deleter, ok := c.(interface {
					Delete(context.Context, string) error
				}); ok {
					deleter.Delete(evictCtx, key)
				}
			}
			logger.Debug("缓存已清除", zap.String("cache", cacheName), zap.Int("count", len(keys)))
		}()

		return nil
	}
}
