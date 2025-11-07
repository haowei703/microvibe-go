package cache

import (
	"context"
	"sync/atomic"
	"time"
)

// multiLevelCache 多级缓存实现 - 组合模式
// L1: 内存缓存（快速访问）
// L2: Redis 缓存（持久化和共享）
type multiLevelCache[T any] struct {
	l1      Cache[T] // 一级缓存（内存）
	l2      Cache[T] // 二级缓存（Redis）
	options *MultiLevelOptions
	opts    *Options
	stats   *Stats
	closed  atomic.Bool
}

// NewMultiLevelCache 创建多级缓存实例
func NewMultiLevelCache[T any](multiOpts *MultiLevelOptions, opts *Options) (Cache[T], error) {
	if multiOpts == nil {
		multiOpts = DefaultMultiLevelOptions()
	}
	if opts == nil {
		opts = DefaultOptions()
	}

	// 创建 L1 缓存（内存）
	l1 := NewMemoryCache[T](multiOpts.L1, opts)

	// 创建 L2 缓存（Redis）
	l2, err := NewRedisCache[T](multiOpts.L2, opts)
	if err != nil {
		return nil, &CacheError{
			Op:  "init",
			Err: err,
		}
	}

	mlc := &multiLevelCache[T]{
		l1:      l1,
		l2:      l2,
		options: multiOpts,
		opts:    opts,
		stats:   &Stats{},
	}

	return mlc, nil
}

// Get 获取缓存（先查 L1，未命中查 L2）
func (mlc *multiLevelCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	if mlc.closed.Load() {
		return zero, ErrCacheClosed
	}

	// 1. 先查 L1（内存）
	value, err := mlc.l1.Get(ctx, key)
	if err == nil {
		if mlc.opts.EnableStats {
			atomic.AddInt64(&mlc.stats.Hits, 1)
		}
		return value, nil
	}

	// 2. L1 未命中，查 L2（Redis）
	value, err = mlc.l2.Get(ctx, key)
	if err != nil {
		if mlc.opts.EnableStats {
			atomic.AddInt64(&mlc.stats.Misses, 1)
		}
		return zero, err
	}

	// 3. L2 命中，回填 L1
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		mlc.l1.Set(ctx, key, value, mlc.options.L1TTL)
	}()

	if mlc.opts.EnableStats {
		atomic.AddInt64(&mlc.stats.Hits, 1)
	}

	return value, nil
}

// Set 设置缓存（同时写入 L1 和 L2）
func (mlc *multiLevelCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	// 使用配置的 TTL
	l1TTL := mlc.options.L1TTL
	l2TTL := mlc.options.L2TTL
	if ttl > 0 {
		l2TTL = ttl
		// L1 的 TTL 不应超过 L2
		if l1TTL > l2TTL {
			l1TTL = l2TTL
		}
	}

	// 先写 L2（确保持久化）
	if err := mlc.l2.Set(ctx, key, value, l2TTL); err != nil {
		return err
	}

	// 再写 L1（允许失败）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		mlc.l1.Set(ctx, key, value, l1TTL)
	}()

	if mlc.opts.EnableStats {
		atomic.AddInt64(&mlc.stats.Sets, 1)
	}

	return nil
}

// GetOrSet 获取或设置缓存
func (mlc *multiLevelCache[T]) GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error) {
	// 先尝试获取
	value, err := mlc.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	// 缓存未命中，调用加载器
	value, err = loader()
	if err != nil {
		var zero T
		return zero, err
	}

	// 设置缓存
	if err := mlc.Set(ctx, key, value, ttl); err != nil {
		// 设置失败不影响返回值
		return value, nil
	}

	return value, nil
}

// Delete 删除缓存（同时删除 L1 和 L2）
func (mlc *multiLevelCache[T]) Delete(ctx context.Context, key string) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	// 同时删除 L1 和 L2
	err1 := mlc.l1.Delete(ctx, key)
	err2 := mlc.l2.Delete(ctx, key)

	// 只要有一个成功就认为成功
	if err1 != nil && err2 != nil {
		return err2
	}

	if mlc.opts.EnableStats {
		atomic.AddInt64(&mlc.stats.Deletes, 1)
	}

	return nil
}

// Exists 检查缓存是否存在
func (mlc *multiLevelCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	if mlc.closed.Load() {
		return false, ErrCacheClosed
	}

	// 先查 L1
	exists, err := mlc.l1.Exists(ctx, key)
	if err == nil && exists {
		return true, nil
	}

	// 再查 L2
	return mlc.l2.Exists(ctx, key)
}

// Clear 清空所有缓存
func (mlc *multiLevelCache[T]) Clear(ctx context.Context) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	// 同时清空 L1 和 L2
	err1 := mlc.l1.Clear(ctx)
	err2 := mlc.l2.Clear(ctx)

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

// GetMulti 批量获取
func (mlc *multiLevelCache[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	if mlc.closed.Load() {
		return nil, ErrCacheClosed
	}

	// 先从 L1 获取
	result, _ := mlc.l1.GetMulti(ctx, keys)

	// 找出未命中的键
	missedKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := result[key]; !exists {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果有未命中的键，从 L2 获取
	if len(missedKeys) > 0 {
		l2Result, err := mlc.l2.GetMulti(ctx, missedKeys)
		if err == nil && len(l2Result) > 0 {
			// 合并结果
			for key, value := range l2Result {
				result[key] = value
			}

			// 异步回填 L1
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				mlc.l1.SetMulti(ctx, l2Result, mlc.options.L1TTL)
			}()
		}
	}

	return result, nil
}

// SetMulti 批量设置
func (mlc *multiLevelCache[T]) SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	l1TTL := mlc.options.L1TTL
	l2TTL := mlc.options.L2TTL
	if ttl > 0 {
		l2TTL = ttl
		if l1TTL > l2TTL {
			l1TTL = l2TTL
		}
	}

	// 先写 L2
	if err := mlc.l2.SetMulti(ctx, items, l2TTL); err != nil {
		return err
	}

	// 异步写 L1
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		mlc.l1.SetMulti(ctx, items, l1TTL)
	}()

	if mlc.opts.EnableStats {
		atomic.AddInt64(&mlc.stats.Sets, int64(len(items)))
	}

	return nil
}

// DeleteMulti 批量删除
func (mlc *multiLevelCache[T]) DeleteMulti(ctx context.Context, keys []string) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	// 同时删除 L1 和 L2
	mlc.l1.DeleteMulti(ctx, keys)
	mlc.l2.DeleteMulti(ctx, keys)

	if mlc.opts.EnableStats {
		atomic.AddInt64(&mlc.stats.Deletes, int64(len(keys)))
	}

	return nil
}

// GetWithTTL 获取缓存及剩余 TTL（从 L2 获取）
func (mlc *multiLevelCache[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	if mlc.closed.Load() {
		var zero T
		return zero, 0, ErrCacheClosed
	}

	// 从 L2 获取（因为 L2 的 TTL 更准确）
	return mlc.l2.GetWithTTL(ctx, key)
}

// Refresh 刷新缓存过期时间
func (mlc *multiLevelCache[T]) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	if mlc.closed.Load() {
		return ErrCacheClosed
	}

	l1TTL := mlc.options.L1TTL
	l2TTL := ttl
	if l1TTL > l2TTL {
		l1TTL = l2TTL
	}

	// 刷新 L2
	if err := mlc.l2.Refresh(ctx, key, l2TTL); err != nil {
		return err
	}

	// 异步刷新 L1
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		mlc.l1.Refresh(ctx, key, l1TTL)
	}()

	return nil
}

// GetStats 获取统计信息（合并 L1 和 L2）
func (mlc *multiLevelCache[T]) GetStats() *Stats {
	l1Stats := mlc.l1.GetStats()
	l2Stats := mlc.l2.GetStats()

	stats := &Stats{
		Hits:      atomic.LoadInt64(&mlc.stats.Hits),
		Misses:    atomic.LoadInt64(&mlc.stats.Misses),
		Sets:      atomic.LoadInt64(&mlc.stats.Sets),
		Deletes:   atomic.LoadInt64(&mlc.stats.Deletes),
		Evictions: l1Stats.Evictions + l2Stats.Evictions,
		ItemCount: l1Stats.ItemCount + l2Stats.ItemCount,
	}
	stats.CalculateHitRate()

	return stats
}

// Close 关闭缓存
func (mlc *multiLevelCache[T]) Close() error {
	if !mlc.closed.CompareAndSwap(false, true) {
		return ErrCacheClosed
	}

	// 关闭 L1 和 L2
	err1 := mlc.l1.Close()
	err2 := mlc.l2.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
