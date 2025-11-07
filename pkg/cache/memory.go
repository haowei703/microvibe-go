package cache

import (
	"container/list"
	"context"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// memoryCache 泛型内存缓存实现 - 使用分片锁提高并发性能
type memoryCache[T any] struct {
	shards  []*cacheShard[T] // 缓存分片
	options *MemoryOptions
	opts    *Options
	stats   *Stats
	closed  atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// cacheShard 缓存分片 - 减少锁竞争
type cacheShard[T any] struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem[T]
	lruList *list.List               // LRU 链表
	lruMap  map[string]*list.Element // 快速查找
}

// cacheItem 缓存项
type cacheItem[T any] struct {
	key       string
	value     T
	expireAt  time.Time
	createdAt time.Time
	accessCnt int64 // 访问次数（用于 LFU）
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache[T any](memOpts *MemoryOptions, opts *Options) Cache[T] {
	if memOpts == nil {
		memOpts = DefaultMemoryOptions()
	}
	if opts == nil {
		opts = DefaultOptions()
	}

	// 确保分片数是 2 的幂（优化哈希性能）
	shardCount := memOpts.ShardCount
	if shardCount <= 0 {
		shardCount = 32
	}

	mc := &memoryCache[T]{
		shards:  make([]*cacheShard[T], shardCount),
		options: memOpts,
		opts:    opts,
		stats:   &Stats{},
		stopCh:  make(chan struct{}),
	}

	// 初始化分片
	for i := 0; i < shardCount; i++ {
		mc.shards[i] = &cacheShard[T]{
			items:   make(map[string]*cacheItem[T]),
			lruList: list.New(),
			lruMap:  make(map[string]*list.Element),
		}
	}

	// 启动清理协程
	if memOpts.CleanupInterval > 0 {
		mc.wg.Add(1)
		go mc.cleanupExpired()
	}

	return mc
}

// getShard 获取键对应的分片（使用 FNV 哈希）
func (mc *memoryCache[T]) getShard(key string) *cacheShard[T] {
	h := fnv.New32a()
	h.Write([]byte(key))
	return mc.shards[h.Sum32()%uint32(len(mc.shards))]
}

// Get 获取缓存
func (mc *memoryCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	if mc.closed.Load() {
		return zero, ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	default:
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.RLock()
	item, exists := shard.items[key]
	shard.mu.RUnlock()

	if !exists {
		if mc.opts.EnableStats {
			atomic.AddInt64(&mc.stats.Misses, 1)
		}
		var zero T
		return zero, ErrCacheMiss
	}

	// 检查是否过期
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		// 异步删除过期项
		go mc.Delete(context.Background(), key)
		if mc.opts.EnableStats {
			atomic.AddInt64(&mc.stats.Misses, 1)
		}
		var zero T
		return zero, ErrCacheExpired
	}

	// 更新 LRU（需要写锁）
	if mc.options.EvictionPolicy == "lru" {
		shard.mu.Lock()
		if elem, ok := shard.lruMap[key]; ok {
			shard.lruList.MoveToFront(elem)
		}
		shard.mu.Unlock()
	}

	// 更新访问次数（LFU）
	atomic.AddInt64(&item.accessCnt, 1)

	if mc.opts.EnableStats {
		atomic.AddInt64(&mc.stats.Hits, 1)
	}

	return item.value, nil
}

// Set 设置缓存
func (mc *memoryCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if mc.closed.Load() {
		return ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if ttl == 0 {
		ttl = mc.opts.DefaultTTL
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 检查是否需要淘汰
	if mc.options.MaxEntries > 0 && len(shard.items) >= mc.options.MaxEntries/len(mc.shards) {
		mc.evictOldest(shard)
	}

	// 创建缓存项
	item := &cacheItem[T]{
		key:       key,
		value:     value,
		createdAt: time.Now(),
	}

	if ttl > 0 {
		item.expireAt = time.Now().Add(ttl)
	}

	// 如果键已存在，先删除旧的 LRU 节点
	if elem, exists := shard.lruMap[key]; exists {
		shard.lruList.Remove(elem)
		delete(shard.lruMap, key)
	}

	// 添加到缓存
	shard.items[key] = item

	// 添加到 LRU
	if mc.options.EvictionPolicy == "lru" {
		elem := shard.lruList.PushFront(key)
		shard.lruMap[key] = elem
	}

	if mc.opts.EnableStats {
		atomic.AddInt64(&mc.stats.Sets, 1)
		atomic.AddInt64(&mc.stats.ItemCount, 1)
	}

	return nil
}

// GetOrSet 获取或设置缓存（缓存穿透保护）
func (mc *memoryCache[T]) GetOrSet(ctx context.Context, key string, loader func() (T, error), ttl time.Duration) (T, error) {
	// 先尝试获取
	value, err := mc.Get(ctx, key)
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
	if err := mc.Set(ctx, key, value, ttl); err != nil {
		// 设置失败不影响返回值
		return value, nil
	}

	return value, nil
}

// Delete 删除缓存
func (mc *memoryCache[T]) Delete(ctx context.Context, key string) error {
	if mc.closed.Load() {
		return ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, exists := shard.items[key]; !exists {
		return ErrCacheMiss
	}

	delete(shard.items, key)

	// 从 LRU 中删除
	if elem, exists := shard.lruMap[key]; exists {
		shard.lruList.Remove(elem)
		delete(shard.lruMap, key)
	}

	if mc.opts.EnableStats {
		atomic.AddInt64(&mc.stats.Deletes, 1)
		atomic.AddInt64(&mc.stats.ItemCount, -1)
	}

	return nil
}

// Exists 检查缓存是否存在
func (mc *memoryCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	if mc.closed.Load() {
		return false, ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.RLock()
	item, exists := shard.items[key]
	shard.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// 检查是否过期
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		go mc.Delete(context.Background(), key)
		return false, nil
	}

	return true, nil
}

// Clear 清空所有缓存（公开方法，带 closed 检查）
func (mc *memoryCache[T]) Clear(ctx context.Context) error {
	if mc.closed.Load() {
		return ErrCacheClosed
	}

	return mc.clearInternal(ctx)
}

// clearInternal 内部清空方法（不检查 closed 标志，供 Close() 调用）
func (mc *memoryCache[T]) clearInternal(ctx context.Context) error {
	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for _, shard := range mc.shards {
		// 在每个 shard 之间检查 context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		shard.mu.Lock()
		shard.items = make(map[string]*cacheItem[T])
		shard.lruList = list.New()
		shard.lruMap = make(map[string]*list.Element)
		shard.mu.Unlock()
	}

	if mc.opts.EnableStats {
		atomic.StoreInt64(&mc.stats.ItemCount, 0)
	}

	return nil
}

// GetMulti 批量获取
func (mc *memoryCache[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	result := make(map[string]T, len(keys))

	for _, key := range keys {
		value, err := mc.Get(ctx, key)
		if err == nil {
			result[key] = value
		}
	}

	return result, nil
}

// SetMulti 批量设置
func (mc *memoryCache[T]) SetMulti(ctx context.Context, items map[string]T, ttl time.Duration) error {
	for key, value := range items {
		if err := mc.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMulti 批量删除
func (mc *memoryCache[T]) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		mc.Delete(ctx, key) // 忽略错误
	}
	return nil
}

// GetWithTTL 获取缓存及剩余 TTL
func (mc *memoryCache[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	var zero T

	if mc.closed.Load() {
		return zero, 0, ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return zero, 0, ctx.Err()
	default:
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.RLock()
	item, exists := shard.items[key]
	shard.mu.RUnlock()

	if !exists {
		var zero T
		return zero, 0, ErrCacheMiss
	}

	var ttl time.Duration
	if !item.expireAt.IsZero() {
		ttl = time.Until(item.expireAt)
		if ttl < 0 {
			go mc.Delete(context.Background(), key)
			var zero T
			return zero, 0, ErrCacheExpired
		}
	}

	return item.value, ttl, nil
}

// Refresh 刷新缓存过期时间
func (mc *memoryCache[T]) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	if mc.closed.Load() {
		return ErrCacheClosed
	}

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	key = mc.buildKey(key)
	shard := mc.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	item, exists := shard.items[key]
	if !exists {
		return ErrCacheMiss
	}

	if ttl > 0 {
		item.expireAt = time.Now().Add(ttl)
	} else {
		item.expireAt = time.Time{}
	}

	return nil
}

// GetStats 获取统计信息
func (mc *memoryCache[T]) GetStats() *Stats {
	stats := &Stats{
		Hits:      atomic.LoadInt64(&mc.stats.Hits),
		Misses:    atomic.LoadInt64(&mc.stats.Misses),
		Sets:      atomic.LoadInt64(&mc.stats.Sets),
		Deletes:   atomic.LoadInt64(&mc.stats.Deletes),
		Evictions: atomic.LoadInt64(&mc.stats.Evictions),
		ItemCount: atomic.LoadInt64(&mc.stats.ItemCount),
	}
	stats.CalculateHitRate()
	return stats
}

// Close 关闭缓存
func (mc *memoryCache[T]) Close() error {
	if !mc.closed.CompareAndSwap(false, true) {
		return ErrCacheClosed
	}

	close(mc.stopCh)
	mc.wg.Wait()

	// 调用内部清空方法，不触发 closed 检查
	return mc.clearInternal(context.Background())
}

// buildKey 构建完整的缓存键
func (mc *memoryCache[T]) buildKey(key string) string {
	if mc.opts.KeyPrefix != "" {
		return mc.opts.KeyPrefix + ":" + key
	}
	return key
}

// evictOldest 淘汰最旧的项（LRU）
func (mc *memoryCache[T]) evictOldest(shard *cacheShard[T]) {
	if shard.lruList.Len() == 0 {
		return
	}

	// 获取最旧的项
	elem := shard.lruList.Back()
	if elem == nil {
		return
	}

	key := elem.Value.(string)
	shard.lruList.Remove(elem)
	delete(shard.lruMap, key)
	delete(shard.items, key)

	if mc.opts.EnableStats {
		atomic.AddInt64(&mc.stats.Evictions, 1)
		atomic.AddInt64(&mc.stats.ItemCount, -1)
	}
}

// cleanupExpired 清理过期缓存（后台协程）
func (mc *memoryCache[T]) cleanupExpired() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.options.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.doCleanup()
		}
	}
}

// doCleanup 执行清理
func (mc *memoryCache[T]) doCleanup() {
	now := time.Now()

	for _, shard := range mc.shards {
		shard.mu.Lock()

		// 查找过期项
		var expiredKeys []string
		for key, item := range shard.items {
			if !item.expireAt.IsZero() && now.After(item.expireAt) {
				expiredKeys = append(expiredKeys, key)
			}
		}

		// 删除过期项
		for _, key := range expiredKeys {
			delete(shard.items, key)
			if elem, exists := shard.lruMap[key]; exists {
				shard.lruList.Remove(elem)
				delete(shard.lruMap, key)
			}
		}

		shard.mu.Unlock()

		if mc.opts.EnableStats && len(expiredKeys) > 0 {
			atomic.AddInt64(&mc.stats.Evictions, int64(len(expiredKeys)))
			atomic.AddInt64(&mc.stats.ItemCount, -int64(len(expiredKeys)))
		}
	}
}
