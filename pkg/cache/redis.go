package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCache 泛型 Redis 缓存实现
type redisCache[T any] struct {
	client  *redis.Client
	options *RedisOptions
	opts    *Options
	stats   *Stats
	closed  atomic.Bool
}

// NewRedisCache 创建 Redis 缓存实例
func NewRedisCache[T any](redisOpts *RedisOptions, opts *Options) (Cache[T], error) {
	if redisOpts == nil {
		redisOpts = DefaultRedisOptions()
	}
	if opts == nil {
		opts = DefaultOptions()
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         redisOpts.Addr,
		Password:     redisOpts.Password,
		DB:           redisOpts.DB,
		PoolSize:     redisOpts.PoolSize,
		MinIdleConns: redisOpts.MinIdleConns,
		DialTimeout:  redisOpts.DialTimeout,
		ReadTimeout:  redisOpts.ReadTimeout,
		WriteTimeout: redisOpts.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, &CacheError{
			Op:  "connect",
			Err: err,
		}
	}

	rc := &redisCache[T]{
		client:  client,
		options: redisOpts,
		opts:    opts,
		stats:   &Stats{},
	}

	return rc, nil
}

// Get 获取缓存
func (rc *redisCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	if rc.closed.Load() {
		return zero, ErrCacheClosed
	}

	key = rc.buildKey(key)

	// 从 Redis 获取
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			if rc.opts.EnableStats {
				atomic.AddInt64(&rc.stats.Misses, 1)
			}
			return zero, ErrCacheMiss
		}
		return zero, &CacheError{
			Op:  "get",
			Key: key,
			Err: err,
		}
	}

	// 反序列化
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, &CacheError{
			Op:  "unmarshal",
			Key: key,
			Err: err,
		}
	}

	if rc.opts.EnableStats {
		atomic.AddInt64(&rc.stats.Hits, 1)
	}

	return value, nil
}

// Set 设置缓存
func (rc *redisCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	if ttl == 0 {
		ttl = rc.opts.DefaultTTL
	}

	key = rc.buildKey(key)

	// 序列化
	data, err := json.Marshal(value)
	if err != nil {
		return &CacheError{
			Op:  "marshal",
			Key: key,
			Err: err,
		}
	}

	// 设置到 Redis
	if err := rc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return &CacheError{
			Op:  "set",
			Key: key,
			Err: err,
		}
	}

	if rc.opts.EnableStats {
		atomic.AddInt64(&rc.stats.Sets, 1)
	}

	return nil
}

// GetOrSet 获取或设置缓存（缓存穿透保护）
func (rc *redisCache[T]) GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error) {
	// 先尝试获取
	value, err := rc.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	// 缓存未命中，调用加载器
	value, err = loader()
	if err != nil {
		var zero T
		return zero, err
	}

	// 设置缓存（异步，不影响返回）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		rc.Set(ctx, key, value, ttl)
	}()

	return value, nil
}

// Delete 删除缓存
func (rc *redisCache[T]) Delete(ctx context.Context, key string) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	key = rc.buildKey(key)

	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return &CacheError{
			Op:  "delete",
			Key: key,
			Err: err,
		}
	}

	if rc.opts.EnableStats {
		atomic.AddInt64(&rc.stats.Deletes, 1)
	}

	return nil
}

// Exists 检查缓存是否存在
func (rc *redisCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	if rc.closed.Load() {
		return false, ErrCacheClosed
	}

	key = rc.buildKey(key)

	count, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, &CacheError{
			Op:  "exists",
			Key: key,
			Err: err,
		}
	}

	return count > 0, nil
}

// Clear 清空所有缓存（谨慎使用）
func (rc *redisCache[T]) Clear(ctx context.Context) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	// 如果有键前缀，只删除带前缀的键
	if rc.opts.KeyPrefix != "" {
		pattern := rc.opts.KeyPrefix + ":*"
		iter := rc.client.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			if err := rc.client.Del(ctx, iter.Val()).Err(); err != nil {
				return &CacheError{
					Op:  "clear",
					Err: err,
				}
			}
		}
		if err := iter.Err(); err != nil {
			return &CacheError{
				Op:  "clear",
				Err: err,
			}
		}
	} else {
		// 没有前缀，清空当前数据库（危险操作）
		if err := rc.client.FlushDB(ctx).Err(); err != nil {
			return &CacheError{
				Op:  "clear",
				Err: err,
			}
		}
	}

	return nil
}

// GetMulti 批量获取
func (rc *redisCache[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	if rc.closed.Load() {
		return nil, ErrCacheClosed
	}

	if len(keys) == 0 {
		return make(map[string]T), nil
	}

	// 构建完整的键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rc.buildKey(key)
	}

	// 批量获取
	values, err := rc.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, &CacheError{
			Op:  "mget",
			Err: err,
		}
	}

	result := make(map[string]T, len(keys))
	for i, val := range values {
		if val == nil {
			continue
		}

		// 反序列化
		var value T
		if strVal, ok := val.(string); ok {
			if err := json.Unmarshal([]byte(strVal), &value); err == nil {
				result[keys[i]] = value
			}
		}
	}

	return result, nil
}

// SetMulti 批量设置
func (rc *redisCache[T]) SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	if len(items) == 0 {
		return nil
	}

	if ttl == 0 {
		ttl = rc.opts.DefaultTTL
	}

	// 使用 Pipeline 提高性能
	pipe := rc.client.Pipeline()

	for key, value := range items {
		fullKey := rc.buildKey(key)

		// 序列化
		data, err := json.Marshal(value)
		if err != nil {
			return &CacheError{
				Op:  "marshal",
				Key: key,
				Err: err,
			}
		}

		pipe.Set(ctx, fullKey, data, ttl)
	}

	// 执行 Pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return &CacheError{
			Op:  "mset",
			Err: err,
		}
	}

	if rc.opts.EnableStats {
		atomic.AddInt64(&rc.stats.Sets, int64(len(items)))
	}

	return nil
}

// DeleteMulti 批量删除
func (rc *redisCache[T]) DeleteMulti(ctx context.Context, keys []string) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	if len(keys) == 0 {
		return nil
	}

	// 构建完整的键
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rc.buildKey(key)
	}

	// 批量删除
	if err := rc.client.Del(ctx, fullKeys...).Err(); err != nil {
		return &CacheError{
			Op:  "mdel",
			Err: err,
		}
	}

	if rc.opts.EnableStats {
		atomic.AddInt64(&rc.stats.Deletes, int64(len(keys)))
	}

	return nil
}

// GetWithTTL 获取缓存及剩余 TTL
func (rc *redisCache[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	var zero T

	if rc.closed.Load() {
		return zero, 0, ErrCacheClosed
	}

	key = rc.buildKey(key)

	// 使用 Pipeline 同时获取值和 TTL
	pipe := rc.client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, 0, ErrCacheMiss
		}
		return zero, 0, &CacheError{
			Op:  "get",
			Key: key,
			Err: err,
		}
	}

	// 获取值
	data, err := getCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, 0, ErrCacheMiss
		}
		return zero, 0, &CacheError{
			Op:  "get",
			Key: key,
			Err: err,
		}
	}

	// 反序列化
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, 0, &CacheError{
			Op:  "unmarshal",
			Key: key,
			Err: err,
		}
	}

	// 获取 TTL
	ttl, err := ttlCmd.Result()
	if err != nil {
		ttl = 0
	}

	return value, ttl, nil
}

// Refresh 刷新缓存过期时间
func (rc *redisCache[T]) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	if rc.closed.Load() {
		return ErrCacheClosed
	}

	key = rc.buildKey(key)

	if err := rc.client.Expire(ctx, key, ttl).Err(); err != nil {
		return &CacheError{
			Op:  "expire",
			Key: key,
			Err: err,
		}
	}

	return nil
}

// GetStats 获取统计信息
func (rc *redisCache[T]) GetStats() *Stats {
	stats := &Stats{
		Hits:    atomic.LoadInt64(&rc.stats.Hits),
		Misses:  atomic.LoadInt64(&rc.stats.Misses),
		Sets:    atomic.LoadInt64(&rc.stats.Sets),
		Deletes: atomic.LoadInt64(&rc.stats.Deletes),
	}
	stats.CalculateHitRate()
	return stats
}

// Close 关闭缓存
func (rc *redisCache[T]) Close() error {
	if !rc.closed.CompareAndSwap(false, true) {
		return ErrCacheClosed
	}

	return rc.client.Close()
}

// buildKey 构建完整的缓存键
func (rc *redisCache[T]) buildKey(key string) string {
	if rc.opts.KeyPrefix != "" {
		return rc.opts.KeyPrefix + ":" + key
	}
	return key
}
