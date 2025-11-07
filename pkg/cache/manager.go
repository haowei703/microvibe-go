package cache

import (
	"fmt"
	"sync"
)

// CacheType 缓存类型
type CacheType string

const (
	TypeMemory     CacheType = "memory"
	TypeRedis      CacheType = "redis"
	TypeMultiLevel CacheType = "multilevel"
)

// Manager 缓存管理器 - 单例模式 + 工厂模式
type Manager struct {
	caches   map[string]interface{} // 存储不同类型的缓存实例
	mu       sync.RWMutex
	asyncOps sync.WaitGroup // 跟踪异步缓存操作（清除等）
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager 获取缓存管理器单例
func GetManager() *Manager {
	managerOnce.Do(func() {
		manager = &Manager{
			caches: make(map[string]interface{}),
		}
	})
	return manager
}

// NewCache 创建缓存实例 - 工厂方法模式
func NewCache[T any](cacheType CacheType, memOpts *MemoryOptions, redisOpts *RedisOptions, multiOpts *MultiLevelOptions, opts *Options) (Cache[T], error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	switch cacheType {
	case TypeMemory:
		return NewMemoryCache[T](memOpts, opts), nil

	case TypeRedis:
		return NewRedisCache[T](redisOpts, opts)

	case TypeMultiLevel:
		return NewMultiLevelCache[T](multiOpts, opts)

	default:
		return nil, &CacheError{
			Op:  "new",
			Err: fmt.Errorf("unknown cache type: %s", cacheType),
		}
	}
}

// Register 注册缓存实例到管理器
func (m *Manager) Register(name string, cache interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.caches[name]; exists {
		return &CacheError{
			Op:  "register",
			Key: name,
			Err: fmt.Errorf("cache already registered"),
		}
	}

	m.caches[name] = cache
	return nil
}

// Get 获取缓存实例
func (m *Manager) Get(name string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cache, exists := m.caches[name]
	if !exists {
		return nil, &CacheError{
			Op:  "get",
			Key: name,
			Err: fmt.Errorf("cache not found"),
		}
	}

	return cache, nil
}

// GetTyped 获取类型化的缓存实例
func GetTyped[T any](name string) (Cache[T], error) {
	m := GetManager()
	cache, err := m.Get(name)
	if err != nil {
		return nil, err
	}

	typedCache, ok := cache.(Cache[T])
	if !ok {
		return nil, &CacheError{
			Op:  "get",
			Key: name,
			Err: fmt.Errorf("cache type mismatch"),
		}
	}

	return typedCache, nil
}

// AddAsyncOp 记录一个异步操作开始
func (m *Manager) AddAsyncOp() {
	m.asyncOps.Add(1)
}

// DoneAsyncOp 标记一个异步操作完成
func (m *Manager) DoneAsyncOp() {
	m.asyncOps.Done()
}

// WaitAsyncOps 等待所有异步操作完成
func (m *Manager) WaitAsyncOps() {
	m.asyncOps.Wait()
}

// Unregister 注销缓存实例
func (m *Manager) Unregister(name string) error {
	// 等待所有异步操作完成，避免竞态条件
	m.asyncOps.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	cache, exists := m.caches[name]
	if !exists {
		return &CacheError{
			Op:  "unregister",
			Key: name,
			Err: fmt.Errorf("cache not found"),
		}
	}

	// 尝试关闭缓存
	if closer, ok := cache.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return &CacheError{
				Op:  "unregister",
				Key: name,
				Err: err,
			}
		}
	}

	delete(m.caches, name)
	return nil
}

// CloseAll 关闭所有缓存实例
func (m *Manager) CloseAll() error {
	// 等待所有异步操作完成，避免竞态条件
	m.asyncOps.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, cache := range m.caches {
		if closer, ok := cache.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				lastErr = &CacheError{
					Op:  "close",
					Key: name,
					Err: err,
				}
			}
		}
	}

	m.caches = make(map[string]interface{})
	return lastErr
}

// List 列出所有已注册的缓存名称
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.caches))
	for name := range m.caches {
		names = append(names, name)
	}
	return names
}

// Builder 缓存构建器 - 构建者模式
type Builder[T any] struct {
	cacheType  CacheType
	memOpts    *MemoryOptions
	redisOpts  *RedisOptions
	multiOpts  *MultiLevelOptions
	opts       *Options
	name       string
	register   bool
	autoCreate bool
	decorators []func(Cache[T]) Cache[T] // 装饰器链
}

// NewBuilder 创建缓存构建器
func NewBuilder[T any]() *Builder[T] {
	return &Builder[T]{
		cacheType:  TypeMemory,
		opts:       DefaultOptions(),
		autoCreate: true,
	}
}

// WithType 设置缓存类型
func (b *Builder[T]) WithType(t CacheType) *Builder[T] {
	b.cacheType = t
	return b
}

// WithMemoryOptions 设置内存缓存选项
func (b *Builder[T]) WithMemoryOptions(opts *MemoryOptions) *Builder[T] {
	b.memOpts = opts
	return b
}

// WithRedisOptions 设置 Redis 缓存选项
func (b *Builder[T]) WithRedisOptions(opts *RedisOptions) *Builder[T] {
	b.redisOpts = opts
	return b
}

// WithMultiLevelOptions 设置多级缓存选项
func (b *Builder[T]) WithMultiLevelOptions(opts *MultiLevelOptions) *Builder[T] {
	b.multiOpts = opts
	return b
}

// WithOptions 设置通用选项
func (b *Builder[T]) WithOptions(opts *Options) *Builder[T] {
	b.opts = opts
	return b
}

// WithName 设置缓存名称（用于注册）
func (b *Builder[T]) WithName(name string) *Builder[T] {
	b.name = name
	b.register = true
	return b
}

// WithDecorator 添加装饰器到装饰器链
// 装饰器按添加顺序应用（从内到外）
func (b *Builder[T]) WithDecorator(decorator func(Cache[T]) Cache[T]) *Builder[T] {
	b.decorators = append(b.decorators, decorator)
	return b
}

// WithLogging 添加日志装饰器（快捷方法）
func (b *Builder[T]) WithLogging() *Builder[T] {
	return b.WithDecorator(NewLoggingDecorator[T])
}

// WithMetrics 添加监控装饰器（快捷方法）
func (b *Builder[T]) WithMetrics(namespace string) *Builder[T] {
	return b.WithDecorator(func(cache Cache[T]) Cache[T] {
		return NewMetricsDecorator[T](cache, namespace)
	})
}

// Build 构建缓存实例
func (b *Builder[T]) Build() (Cache[T], error) {
	cache, err := NewCache[T](b.cacheType, b.memOpts, b.redisOpts, b.multiOpts, b.opts)
	if err != nil {
		return nil, err
	}

	// 应用装饰器链（从内到外）
	for _, decorator := range b.decorators {
		cache = decorator(cache)
	}

	// 如果指定了名称，注册到管理器
	if b.register && b.name != "" {
		if err := GetManager().Register(b.name, cache); err != nil {
			// 注册失败，关闭缓存
			cache.Close()
			return nil, err
		}
	}

	return cache, nil
}

// MustBuild 构建缓存实例（失败则 panic）
func (b *Builder[T]) MustBuild() Cache[T] {
	cache, err := b.Build()
	if err != nil {
		panic(err)
	}
	return cache
}
