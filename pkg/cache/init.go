package cache

import (
	"fmt"
	"microvibe-go/internal/config"
	"microvibe-go/internal/model"
	"time"
)

var (
	memoryOpt = &MemoryOptions{}
	redisOpt  = &RedisOptions{}
)

// InitCaches 初始化缓存实例
// 在应用启动时调用此函数初始化常用的缓存
func InitCaches(cfg *config.Config) error {
	// 统一配置 Redis 连接参数
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	redisOpt.Addr = redisAddr
	redisOpt.Password = cfg.Redis.Password
	redisOpt.DB = 0
	redisOpt.PoolSize = 20
	redisOpt.MinIdleConns = 5
	redisOpt.DialTimeout = 5 * time.Second
	redisOpt.ReadTimeout = 3 * time.Second
	redisOpt.WriteTimeout = 3 * time.Second

	// 统一配置内存缓存参数
	memoryOpt.MaxEntries = 5000
	memoryOpt.CleanupInterval = 2 * time.Minute
	memoryOpt.EvictionPolicy = "lru"
	memoryOpt.ShardCount = 32

	// 1. 用户缓存（多级缓存 + 日志装饰器）
	_, err := NewBuilder[*model.User]().
		WithType(TypeMultiLevel).
		WithMultiLevelOptions(&MultiLevelOptions{
			L1:                 memoryOpt,
			L2:                 redisOpt,
			L1TTL:              1 * time.Minute,  // 内存缓存 1 分钟
			L2TTL:              10 * time.Minute, // Redis 缓存 10 分钟
			EnableWriteThrough: true,
		}).
		WithOptions(&Options{
			DefaultTTL:    5 * time.Minute,
			KeyPrefix:     "user",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("user").
		Build()

	if err != nil {
		return fmt.Errorf("初始化用户缓存失败: %w", err)
	}

	// 2. 视频缓存（多级缓存 + 日志装饰器）
	_, err = NewBuilder[*model.Video]().
		WithType(TypeMultiLevel).
		WithMultiLevelOptions(&MultiLevelOptions{
			L1:                 memoryOpt,
			L2:                 redisOpt,
			L1TTL:              2 * time.Minute,
			L2TTL:              15 * time.Minute,
			EnableWriteThrough: true,
		}).
		WithOptions(&Options{
			DefaultTTL:    10 * time.Minute,
			KeyPrefix:     "video",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("video").
		Build()

	if err != nil {
		return fmt.Errorf("初始化视频缓存失败: %w", err)
	}

	// 3. 分类缓存（内存缓存 + 日志装饰器，数据不常变化）
	_, err = NewBuilder[*model.Category]().
		WithType(TypeMemory).
		WithMemoryOptions(&MemoryOptions{
			MaxEntries:      1000,
			CleanupInterval: 5 * time.Minute,
			EvictionPolicy:  "lru",
			ShardCount:      8,
		}).
		WithOptions(&Options{
			DefaultTTL:    30 * time.Minute, // 较长的缓存时间
			KeyPrefix:     "category",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("category").
		Build()

	if err != nil {
		return fmt.Errorf("初始化分类缓存失败: %w", err)
	}

	// 4. 热门视频列表缓存（Redis缓存 + 日志装饰器，多个服务共享）
	// 注意：此缓存用于存储视频列表，保留 interface{} 以支持多种数据结构
	_, err = NewBuilder[interface{}]().
		WithType(TypeRedis).
		WithRedisOptions(redisOpt).
		WithOptions(&Options{
			DefaultTTL:    5 * time.Minute,
			KeyPrefix:     "hot",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("hot").
		Build()

	if err != nil {
		return fmt.Errorf("初始化热门缓存失败: %w", err)
	}

	// 5. 通用缓存（多级缓存 + 日志装饰器，用于临时数据）
	_, err = NewBuilder[interface{}]().
		WithType(TypeMultiLevel).
		WithMultiLevelOptions(DefaultMultiLevelOptions()).
		WithOptions(&Options{
			DefaultTTL:    5 * time.Minute,
			KeyPrefix:     "general",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("general").
		Build()

	if err != nil {
		return fmt.Errorf("初始化通用缓存失败: %w", err)
	}

	// 6. 直播缓存（多级缓存 + 日志装饰器，用于直播间信息）
	_, err = NewBuilder[*model.LiveStream]().
		WithType(TypeMultiLevel).
		WithMultiLevelOptions(&MultiLevelOptions{
			L1:                 memoryOpt,
			L2:                 redisOpt,
			L1TTL:              1 * time.Minute, // 内存缓存 1 分钟
			L2TTL:              5 * time.Minute, // Redis 缓存 5 分钟
			EnableWriteThrough: true,
		}).
		WithOptions(&Options{
			DefaultTTL:    5 * time.Minute,
			KeyPrefix:     "livestream",
			EnableStats:   true,
			EnableLogging: false,
		}).
		WithLogging(). // 添加日志装饰器
		WithName("livestream").
		Build()

	if err != nil {
		return fmt.Errorf("初始化直播缓存失败: %w", err)
	}

	return nil
}

// CloseCaches 关闭所有缓存
func CloseCaches() error {
	return GetManager().CloseAll()
}
