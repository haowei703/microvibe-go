package cache

import (
	"time"
)

// Options 缓存配置选项
type Options struct {
	// 默认过期时间
	DefaultTTL time.Duration

	// 键前缀
	KeyPrefix string

	// 是否启用统计
	EnableStats bool

	// 是否启用日志
	EnableLogging bool
}

// MemoryOptions 内存缓存配置
type MemoryOptions struct {
	// 最大缓存项数量
	MaxEntries int

	// 清理间隔
	CleanupInterval time.Duration

	// 淘汰策略: "lru", "lfu"
	EvictionPolicy string

	// 分片数量（减少锁竞争）
	ShardCount int
}

// RedisOptions Redis 缓存配置
type RedisOptions struct {
	// Redis 地址
	Addr string

	// 密码
	Password string

	// 数据库
	DB int

	// 连接池大小
	PoolSize int

	// 最小空闲连接
	MinIdleConns int

	// 连接超时
	DialTimeout time.Duration

	// 读超时
	ReadTimeout time.Duration

	// 写超时
	WriteTimeout time.Duration

	// 连接最大空闲时间（不再使用，保留字段兼容性）
	IdleTimeout time.Duration

	// 连接最大存活时间（不再使用，保留字段兼容性）
	MaxConnAge time.Duration
}

// MultiLevelOptions 多级缓存配置
type MultiLevelOptions struct {
	// L1 缓存（内存）配置
	L1 *MemoryOptions

	// L2 缓存（Redis）配置
	L2 *RedisOptions

	// L1 过期时间（通常小于 L2）
	L1TTL time.Duration

	// L2 过期时间
	L2TTL time.Duration

	// 是否启用 L1 到 L2 的写穿透
	EnableWriteThrough bool
}

// DefaultOptions 返回默认配置
func DefaultOptions() *Options {
	return &Options{
		DefaultTTL:    5 * time.Minute,
		EnableStats:   true,
		EnableLogging: false,
	}
}

// DefaultMemoryOptions 返回默认内存缓存配置
func DefaultMemoryOptions() *MemoryOptions {
	return &MemoryOptions{
		MaxEntries:      10000,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  "lru",
		ShardCount:      32, // 32 个分片减少锁竞争
	}
}

// DefaultRedisOptions 返回默认 Redis 配置
func DefaultRedisOptions() *RedisOptions {
	return &RedisOptions{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  5 * time.Minute,
		MaxConnAge:   0,
	}
}

// DefaultMultiLevelOptions 返回默认多级缓存配置
func DefaultMultiLevelOptions() *MultiLevelOptions {
	return &MultiLevelOptions{
		L1:                 DefaultMemoryOptions(),
		L2:                 DefaultRedisOptions(),
		L1TTL:              1 * time.Minute,  // L1 短期缓存
		L2TTL:              10 * time.Minute, // L2 长期缓存
		EnableWriteThrough: true,
	}
}

// Option 配置函数类型
type Option func(*Options)

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// WithKeyPrefix 设置键前缀
func WithKeyPrefix(prefix string) Option {
	return func(o *Options) {
		o.KeyPrefix = prefix
	}
}

// WithEnableStats 设置是否启用统计
func WithEnableStats(enable bool) Option {
	return func(o *Options) {
		o.EnableStats = enable
	}
}

// WithEnableLogging 设置是否启用日志
func WithEnableLogging(enable bool) Option {
	return func(o *Options) {
		o.EnableLogging = enable
	}
}
