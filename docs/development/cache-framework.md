# 缓存框架使用文档

## 概述

本项目集成了一个高性能、功能丰富的缓存框架。该框架基于泛型设计，提供类型安全的缓存操作，支持多种缓存策略和装饰器模式，使用体验类似 Spring Cache。

## 核心特性

### 1. 泛型支持
- 使用 Go 1.18+ 泛型特性
- 类型安全，避免类型断言
- 编译时类型检查
- 更好的 IDE 支持

### 2. 多种缓存实现
- **内存缓存** (Memory Cache)
  - LRU 淘汰策略
  - TTL 支持
  - 分片锁设计，减少锁竞争
  - 后台自动清理过期数据

- **Redis 缓存** (Redis Cache)
  - 基于 go-redis 客户端
  - 连接池管理
  - 自动序列化/反序列化
  - 支持批量操作

- **多级缓存** (Multi-Level Cache)
  - L1: 内存缓存（快速访问）
  - L2: Redis 缓存（持久化）
  - 自动回填机制
  - 写穿透支持

### 3. Spring Cache 风格的装饰器

**这是最推荐的使用方式！** 类似 Spring 的 `@Cacheable`、`@CacheEvict` 注解：

- **WithCache**: 自动缓存查询结果（类似 `@Cacheable`）
- **WithCacheEvict**: 自动清除单个缓存（类似 `@CacheEvict`）
- **WithMultiCacheEvict**: 自动清除多个缓存（类似 `@Caching`）

### 4. 高阶函数 KeyGenerator

支持自定义缓存键生成，使用高阶函数和装饰器模式：

- **6种内置生成器**: Default、JSON、MD5、SHA256、PrefixOnly、Custom
- **装饰器组合**: WithUpperCase、WithLowerCase、WithPrefix、WithSuffix
- **链式组合**: ChainKeyGenerators 实现复杂组合

### 5. 设计模式
- **策略模式**: 统一的 Cache 接口，多种实现
- **工厂模式**: Builder 构建器创建缓存实例
- **单例模式**: 全局缓存管理器
- **装饰器模式**: 日志、监控、方法级缓存
- **组合模式**: 多级缓存组合
- **高阶函数**: KeyGenerator 函数式设计

### 6. 高性能设计
- 分片锁减少并发竞争
- 批量操作支持
- 异步缓存清除（使用 WaitGroup 同步）
- 零内存分配（atomic 操作）
- FNV 哈希算法

## 快速开始

### 1. 初始化缓存

在应用启动时初始化缓存（`cmd/server/main.go`）：

```go
import (
    "microvibe-go/pkg/cache"
)

func main() {
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    // 初始化缓存
    redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
    if err := cache.InitCaches(cfg, redisAddr); err != nil {
        logger.Error("初始化缓存失败", zap.Error(err))
        // 缓存初始化失败不影响应用启动
    }

    // 应用退出时关闭缓存
    defer cache.CloseCaches()

    // ... 其他初始化代码
}
```

### 2. 在 Repository 层使用装饰器（⭐ 推荐方式）

**这是最简洁、最优雅的方式！**缓存逻辑完全集成在方法内部，无需手动管理缓存。

#### 示例：UserRepository

```go
package repository

import (
    "context"
    "fmt"
    "microvibe-go/internal/model"
    "microvibe-go/pkg/cache"
    "time"
    "gorm.io/gorm"
)

type UserRepository interface {
    FindByID(ctx context.Context, id uint) (*model.User, error)
    FindByUsername(ctx context.Context, username string) (*model.User, error)
    UpdateAge(ctx context.Context, id uint, age int) error
    Update(ctx context.Context, user *model.User) error
}

type userRepositoryImpl struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepositoryImpl{db: db}
}

// FindByID 查询用户（自动缓存）
// 使用 WithCache 装饰器，类似 Spring 的 @Cacheable
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",          // 缓存名称
            KeyPrefix: "user:id",       // 缓存键前缀
            TTL:       10 * time.Minute, // 过期时间
            // KeyGenerator: cache.MD5KeyGenerator(), // 可选：自定义键生成器
        },
        func() (*model.User, error) {
            // 实际的数据库查询逻辑
            var user model.User
            if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, id) // 传入 context 和参数，生成缓存键 "user:id:123"
}

// FindByUsername 查询用户（使用不同的缓存键）
func (r *userRepositoryImpl) FindByUsername(ctx context.Context, username string) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:username", // 不同的键前缀
            TTL:       10 * time.Minute,
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("username = ?", username).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, username) // 生成缓存键 "user:username:alice"
}

// UpdateAge 更新年龄（自动清除缓存）
// 使用 WithCacheEvict 装饰器，类似 Spring 的 @CacheEvict
func (r *userRepositoryImpl) UpdateAge(ctx context.Context, id uint, age int) error {
    return cache.WithCacheEvict(
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:id",
        },
        func() error {
            // 实际的数据库更新逻辑
            return r.db.WithContext(ctx).
                Model(&model.User{}).
                Where("id = ?", id).
                Update("age", age).
                Error
        },
    )(ctx, id) // 异步清除缓存键 "user:id:123"
}

// Update 更新用户（清除多个相关缓存）
// 使用 WithMultiCacheEvict 装饰器，类似 Spring 的 @Caching
func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
    // 需要清除的多个缓存键
    keys := []string{
        fmt.Sprintf("user:id:%d", user.ID),
        fmt.Sprintf("user:username:%s", user.Username),
        fmt.Sprintf("user:email:%s", user.Email),
    }

    return cache.WithMultiCacheEvict("user", keys, func() error {
        // 实际的数据库更新逻辑
        return r.db.WithContext(ctx).Save(user).Error
    })(ctx) // 异步清除所有相关缓存
}
```

#### 装饰器模式的优点

✅ **无需手动管理缓存**：不需要调用 `Get`/`Set`/`Delete`
✅ **代码简洁**：缓存逻辑集中在方法内部
✅ **自动降级**：缓存不可用时自动降级到数据库查询
✅ **异步清除**：缓存清除不影响主流程性能
✅ **类型安全**：基于泛型，编译时类型检查

### 3. 使用 KeyGenerator 自定义缓存键

#### 使用 MD5 处理长 token

```go
func (r *userRepositoryImpl) FindByToken(ctx context.Context, token string) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName:    "user",
            KeyPrefix:    "user:token",
            TTL:          5 * time.Minute,
            KeyGenerator: cache.MD5KeyGenerator(), // MD5 生成固定长度的键
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("token = ?", token).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, token)
}
```

#### 使用 JSON KeyGenerator 处理复杂查询

```go
type UserQuery struct {
    Email  string
    Active bool
    Role   string
}

func (r *userRepositoryImpl) FindByQuery(ctx context.Context, query UserQuery) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName:    "user",
            KeyPrefix:    "user:query",
            TTL:          5 * time.Minute,
            KeyGenerator: cache.JSONKeyGenerator(), // JSON 序列化复杂对象
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("email = ? AND active = ? AND role = ?",
                      query.Email, query.Active, query.Role).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, query)
}
```

#### 使用链式组合实现多租户隔离

```go
func (r *userRepositoryImpl) FindByIDWithTenant(
    ctx context.Context,
    tenantID string,
    id uint,
) (*model.User, error) {
    // 组合：小写 + 租户前缀
    tenantKeyGen := cache.ChainKeyGenerators(
        cache.DefaultKeyGenerator(),
        cache.WithLowerCase,
        func(g cache.KeyGenerator) cache.KeyGenerator {
            return cache.WithPrefix(fmt.Sprintf("tenant:%s:", tenantID), g)
        },
    )

    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName:    "user",
            KeyPrefix:    "USER:ID", // 会被转为小写
            TTL:          10 * time.Minute,
            KeyGenerator: tenantKeyGen, // 使用自定义生成器
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("tenant_id = ? AND id = ?", tenantID, id).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, id) // 生成: "tenant:abc:user:id:123"
}
```

### 4. 传统方式：直接使用缓存接口

如果需要更细粒度的控制，也可以直接使用缓存接口：

#### 从管理器获取缓存

```go
import (
    "context"
    "microvibe-go/pkg/cache"
    "time"
)

// 获取类型化的缓存实例
userCache, err := cache.GetTyped[*model.User]("user")
if err != nil {
    // 处理错误
}

// 设置缓存
ctx := context.Background()
user := &model.User{ID: 1, Username: "test"}
err = userCache.Set(ctx, "user:1", user, 10*time.Minute)

// 获取缓存
user, err = userCache.Get(ctx, "user:1")
if err == cache.ErrCacheMiss {
    // 缓存未命中
}

// 删除缓存
err = userCache.Delete(ctx, "user:1")
```

#### 使用 GetOrSet 模式

```go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
    userCache, _ := cache.GetTyped[*model.User]("user")
    cacheKey := fmt.Sprintf("user:id:%d", id)

    // GetOrSet: 如果缓存存在则返回，否则调用 loader 函数
    user, err := userCache.GetOrSet(ctx, cacheKey, func() (*model.User, error) {
        // 从数据库加载
        var user model.User
        if err := r.db.First(&user, id).Error; err != nil {
            return nil, err
        }
        return &user, nil
    }, 10*time.Minute)

    return user, err
}
```

#### 批量操作

```go
// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
result, err := userCache.GetMulti(ctx, keys)
for key, user := range result {
    fmt.Printf("Key: %s, User: %+v\n", key, user)
}

// 批量设置
items := map[string]*model.User{
    "user:1": {ID: 1, Username: "user1"},
    "user:2": {ID: 2, Username: "user2"},
}
err = userCache.SetMulti(ctx, items, 10*time.Minute)

// 批量删除
err = userCache.DeleteMulti(ctx, keys)
```

### 5. 使用 Builder 创建自定义缓存

```go
// 创建内存缓存
memCache := cache.NewBuilder[*model.Video]().
    WithType(cache.TypeMemory).
    WithMemoryOptions(&cache.MemoryOptions{
        MaxEntries:      10000,
        CleanupInterval: 1 * time.Minute,
        EvictionPolicy:  "lru",
        ShardCount:      32,
    }).
    WithOptions(&cache.Options{
        DefaultTTL:    5 * time.Minute,
        KeyPrefix:     "video",
        EnableStats:   true,
        EnableLogging: true,
    }).
    MustBuild()

// 注册到管理器
cache.GetManager().Register("video", memCache)

// 使用缓存
video := &model.Video{ID: 1, Title: "测试视频"}
memCache.Set(ctx, "1", video, 0) // 0 表示使用默认 TTL
```

### 6. 缓存统计

```go
stats := userCache.GetStats()
fmt.Printf("命中率: %.2f%%\n", stats.HitRate*100)
fmt.Printf("命中次数: %d\n", stats.Hits)
fmt.Printf("未命中次数: %d\n", stats.Misses)
fmt.Printf("缓存项数量: %d\n", stats.ItemCount)
```

## 装饰器详解

### WithCache - 自动缓存查询（@Cacheable）

**功能**：自动处理缓存逻辑，缓存命中则返回，未命中则执行 loader 函数并缓存结果。

**签名**：
```go
func WithCache[T any](
    config CacheConfig,
    loader func() (T, error),
) func(ctx context.Context, args ...interface{}) (T, error)
```

**使用场景**：
- 数据库查询
- 外部 API 调用
- 计算密集型操作

**示例**：
```go
func (r *VideoRepository) FindByID(ctx context.Context, id uint) (*model.Video, error) {
    return cache.WithCache[*model.Video](
        cache.CacheConfig{
            CacheName: "video",
            KeyPrefix: "video:id",
            TTL:       15 * time.Minute,
        },
        func() (*model.Video, error) {
            var video model.Video
            if err := r.db.First(&video, id).Error; err != nil {
                return nil, err
            }
            return &video, nil
        },
    )(ctx, id)
}
```

### WithCacheEvict - 自动清除缓存（@CacheEvict）

**功能**：执行函数后异步清除指定缓存。

**签名**：
```go
func WithCacheEvict(
    config CacheConfig,
    fn func() error,
) func(ctx context.Context, args ...interface{}) error
```

**使用场景**：
- 更新单个字段
- 删除数据
- 状态变更

**示例**：
```go
func (r *VideoRepository) Delete(ctx context.Context, id uint) error {
    return cache.WithCacheEvict(
        cache.CacheConfig{
            CacheName: "video",
            KeyPrefix: "video:id",
        },
        func() error {
            return r.db.Delete(&model.Video{}, id).Error
        },
    )(ctx, id)
}
```

### WithMultiCacheEvict - 清除多个缓存（@Caching）

**功能**：执行函数后异步清除多个相关缓存。

**签名**：
```go
func WithMultiCacheEvict(
    cacheName string,
    keys []string,
    fn func() error,
) func(ctx context.Context) error
```

**使用场景**：
- 更新影响多个索引的数据
- 批量更新
- 关联数据变更

**示例**：
```go
func (r *VideoRepository) Update(ctx context.Context, video *model.Video) error {
    keys := []string{
        fmt.Sprintf("video:id:%d", video.ID),
        fmt.Sprintf("video:list:user:%d", video.UserID),
        fmt.Sprintf("video:list:category:%d", video.CategoryID),
    }

    return cache.WithMultiCacheEvict("video", keys, func() error {
        return r.db.Save(video).Error
    })(ctx)
}
```

## KeyGenerator 详解

详细文档请参考 `docs/keygen.md`。

### 内置生成器

| 生成器 | 特点 | 性能 | 使用场景 |
|--------|------|------|----------|
| `DefaultKeyGenerator()` | 字符串拼接 | ~43 ns/op | 通用场景 |
| `JSONKeyGenerator()` | JSON序列化 | ~153 ns/op | 复杂对象 |
| `MD5KeyGenerator()` | MD5哈希 | ~197 ns/op | 长参数/特殊字符 |
| `SHA256KeyGenerator()` | SHA256哈希 | ~161 ns/op | 安全性要求高 |
| `PrefixOnlyKeyGenerator()` | 仅前缀 | - | 全局缓存 |
| `CustomKeyGenerator()` | 自定义 | - | 特殊需求 |

### 装饰器函数

```go
// 大小写转换
gen := cache.WithUpperCase(cache.DefaultKeyGenerator())
gen := cache.WithLowerCase(cache.DefaultKeyGenerator())

// 添加前缀/后缀
gen := cache.WithPrefix("app:v1:", cache.DefaultKeyGenerator())
gen := cache.WithSuffix(":prod", cache.DefaultKeyGenerator())

// 链式组合
gen := cache.ChainKeyGenerators(
    cache.DefaultKeyGenerator(),
    cache.WithLowerCase,
    func(g cache.KeyGenerator) cache.KeyGenerator {
        return cache.WithPrefix("app:", g)
    },
)
```

## 缓存策略

### 1. Cache-Aside（旁路缓存）⭐ 推荐

使用 `WithCache` 装饰器自动实现：

```go
cache.WithCache[T](config, loader)(ctx, args...)
```

**优点**：
- 缓存失效时自动加载
- 并发控制（同一 key 只有一个请求查询数据库）
- 代码简洁

### 2. Cache-Invalidation（缓存失效）⭐ 推荐

使用 `WithCacheEvict` 或 `WithMultiCacheEvict` 装饰器：

```go
// 单个清除
cache.WithCacheEvict(config, fn)(ctx, args...)

// 批量清除
cache.WithMultiCacheEvict(cacheName, keys, fn)(ctx)
```

**优点**：
- 异步清除，不影响主流程
- WaitGroup 同步，确保安全关闭
- 自动降级

### 3. Write-Through（写穿透）

更新数据时同时更新缓存：

```go
// 更新数据库
db.Update(data)

// 更新缓存
cache.Set(ctx, key, data, ttl)
```

**适用场景**：读多写少，数据一致性要求高

### 4. Write-Behind（写回）

先更新缓存，异步更新数据库：

```go
// 更新缓存
cache.Set(ctx, key, data, ttl)

// 异步更新数据库
go db.Update(data)
```

**适用场景**：高性能要求，可接受短暂数据丢失

## 缓存键设计规范

### 命名规则
- 使用冒号 `:` 分隔层级
- 使用前缀区分不同业务
- 键名要有意义，易于理解
- 使用小写字母（推荐使用 `WithLowerCase`）

### 示例
```go
// 用户缓存
"user:id:1"           // 按 ID 查询
"user:username:john"  // 按用户名查询
"user:email:a@b.com"  // 按邮箱查询

// 视频缓存
"video:id:123"        // 视频详情
"video:list:hot"      // 热门视频列表
"video:user:1:list"   // 用户的视频列表

// 分类缓存
"category:all"        // 所有分类
"category:id:5"       // 单个分类

// 多租户隔离
"tenant:abc:user:id:1"     // 租户 abc 的用户
"tenant:xyz:video:id:100"  // 租户 xyz 的视频
```

### 使用 KeyGenerator 生成规范键

```go
// 统一使用小写
config := cache.CacheConfig{
    CacheName:    "user",
    KeyPrefix:    "USER:ID", // 原始前缀
    KeyGenerator: cache.WithLowerCase(cache.DefaultKeyGenerator()), // 转小写
}

// 多租户前缀
tenantGen := cache.WithPrefix("tenant:abc:", cache.DefaultKeyGenerator())
config.KeyGenerator = tenantGen
```

## 缓存过期时间建议

```go
// 用户信息（不常变化）
10 * time.Minute

// 视频信息（较稳定）
15 * time.Minute

// 分类、标签（很少变化）
30 * time.Minute

// 热门列表（实时性要求高）
1 * time.Minute

// 统计数据（准确性要求不高）
5 * time.Minute

// 会话信息
30 * time.Minute

// 验证码
5 * time.Minute
```

## 性能优化建议

### 1. 优先使用装饰器模式

```go
// ✅ 推荐：使用装饰器
return cache.WithCache[*User](config, loader)(ctx, id)

// ❌ 不推荐：手动管理
user, err := cache.Get(ctx, key)
if err == cache.ErrCacheMiss {
    user = loadFromDB()
    cache.Set(ctx, key, user, ttl)
}
```

### 2. 使用分片锁

内存缓存已默认使用 32 个分片，如果并发量特别大，可以增加分片数：

```go
memOpts := &cache.MemoryOptions{
    ShardCount: 64, // 增加到 64 个分片
}
```

### 3. 批量操作

尽可能使用批量操作减少网络往返：

```go
// ❌ 不好：多次调用
for _, key := range keys {
    cache.Get(ctx, key)
}

// ✅ 好：批量获取
cache.GetMulti(ctx, keys)
```

### 4. 异步清除缓存

装饰器已自动异步清除，无需手动处理：

```go
// WithCacheEvict 和 WithMultiCacheEvict 已自动异步清除
cache.WithCacheEvict(config, fn)(ctx, args...)
```

### 5. 使用多级缓存

对于访问频繁的数据，使用多级缓存：

```go
multiCache := cache.NewBuilder[Data]().
    WithType(cache.TypeMultiLevel).
    WithMultiLevelOptions(cache.DefaultMultiLevelOptions()).
    MustBuild()
```

### 6. 选择合适的 KeyGenerator

```go
// 简单参数：使用默认生成器（最快）
KeyGenerator: nil // 自动使用 DefaultKeyGenerator

// 长参数：使用 MD5（固定长度）
KeyGenerator: cache.MD5KeyGenerator()

// 复杂对象：使用 JSON（支持嵌套）
KeyGenerator: cache.JSONKeyGenerator()

// 链式组合：环境隔离 + 小写
KeyGenerator: cache.ChainKeyGenerators(
    cache.DefaultKeyGenerator(),
    cache.WithLowerCase,
    func(g cache.KeyGenerator) cache.KeyGenerator {
        return cache.WithPrefix("prod:", g)
    },
)
```

## 注意事项

### 1. 缓存穿透

使用 `WithCache` 装饰器自动防止缓存穿透：

```go
// WithCache 内部有并发控制，同一个 key 只会有一个请求查询数据库
cache.WithCache[T](config, loader)(ctx, args...)
```

### 2. 缓存雪崩

设置随机的过期时间：

```go
// 基础 TTL + 随机时间
baseTTL := 10 * time.Minute
randomTTL := time.Duration(rand.Intn(60)) * time.Second
ttl := baseTTL + randomTTL
```

### 3. 缓存一致性

使用 `WithCacheEvict` 或 `WithMultiCacheEvict` 确保缓存一致性：

```go
// ✅ 推荐：使用装饰器自动清除
cache.WithCacheEvict(config, fn)(ctx, args...)

// ⚠️ 手动清除：确保在更新数据库后清除
db.Update(data)
cache.Delete(ctx, key)
```

### 4. 内存限制

设置合理的 MaxEntries 防止内存溢出：

```go
memOpts := &cache.MemoryOptions{
    MaxEntries: 10000, // 限制最大缓存项
}
```

### 5. 异步操作安全性

框架使用 `sync.WaitGroup` 确保异步操作安全：

- `WithCacheEvict` 和 `WithMultiCacheEvict` 异步清除缓存
- `Manager.Unregister()` 和 `CloseAll()` 等待所有异步操作完成
- 测试通过 race 检测，无竞态条件

## 故障处理

### 缓存不可用时的自动降级

使用装饰器模式会自动降级：

```go
// 如果缓存实例不存在，WithCache 会自动降级到直接查询
cache.WithCache[T](config, loader)(ctx, args...)
```

### 手动降级示例

```go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
    // 尝试从缓存获取
    if r.cache != nil {
        user, err := r.cache.Get(ctx, cacheKey)
        if err == nil {
            return user, nil
        }
    }

    // 缓存失败，直接查数据库
    var user model.User
    if err := r.db.First(&user, id).Error; err != nil {
        return nil, err
    }

    // 尝试回填缓存（忽略错误）
    if r.cache != nil {
        go r.cache.Set(context.Background(), cacheKey, &user, ttl)
    }

    return &user, nil
}
```

## 完整示例

### UserRepository 完整实现

```go
package repository

import (
    "context"
    "fmt"
    "microvibe-go/internal/model"
    "microvibe-go/pkg/cache"
    "time"
    "gorm.io/gorm"
)

type UserRepository interface {
    FindByID(ctx context.Context, id uint) (*model.User, error)
    FindByUsername(ctx context.Context, username string) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    UpdateAge(ctx context.Context, id uint, age int) error
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id uint) error
}

type userRepositoryImpl struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepositoryImpl{db: db}
}

// FindByID 按 ID 查询用户（自动缓存）
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:id",
            TTL:       10 * time.Minute,
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, id)
}

// FindByUsername 按用户名查询（自动缓存）
func (r *userRepositoryImpl) FindByUsername(ctx context.Context, username string) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:username",
            TTL:       10 * time.Minute,
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("username = ?", username).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, username)
}

// FindByEmail 按邮箱查询（自动缓存）
func (r *userRepositoryImpl) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:email",
            TTL:       10 * time.Minute,
        },
        func() (*model.User, error) {
            var user model.User
            if err := r.db.WithContext(ctx).
                Where("email = ?", email).
                First(&user).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, email)
}

// Create 创建用户
func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

// UpdateAge 更新年龄（清除单个缓存）
func (r *userRepositoryImpl) UpdateAge(ctx context.Context, id uint, age int) error {
    return cache.WithCacheEvict(
        cache.CacheConfig{
            CacheName: "user",
            KeyPrefix: "user:id",
        },
        func() error {
            return r.db.WithContext(ctx).
                Model(&model.User{}).
                Where("id = ?", id).
                Update("age", age).
                Error
        },
    )(ctx, id)
}

// Update 更新用户（清除多个缓存）
func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
    keys := []string{
        fmt.Sprintf("user:id:%d", user.ID),
        fmt.Sprintf("user:username:%s", user.Username),
        fmt.Sprintf("user:email:%s", user.Email),
    }

    return cache.WithMultiCacheEvict("user", keys, func() error {
        return r.db.WithContext(ctx).Save(user).Error
    })(ctx)
}

// Delete 删除用户（清除多个缓存）
func (r *userRepositoryImpl) Delete(ctx context.Context, id uint) error {
    // 先查询用户获取完整信息
    user, err := r.FindByID(ctx, id)
    if err != nil {
        return err
    }

    keys := []string{
        fmt.Sprintf("user:id:%d", user.ID),
        fmt.Sprintf("user:username:%s", user.Username),
        fmt.Sprintf("user:email:%s", user.Email),
    }

    return cache.WithMultiCacheEvict("user", keys, func() error {
        return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
    })(ctx)
}
```

## 常见问题

### Q: 如何选择缓存类型？
- **单机部署**: 使用内存缓存
- **分布式部署**: 使用 Redis 缓存或多级缓存
- **高并发场景**: 使用多级缓存（内存 + Redis）

### Q: 装饰器模式和传统模式哪个更好？
- **推荐装饰器模式**：代码简洁，自动降级，类型安全
- 传统模式适用于需要更细粒度控制的场景

### Q: 如何选择 KeyGenerator？
- **简单参数**: 使用 `DefaultKeyGenerator`（默认）
- **长参数/特殊字符**: 使用 `MD5KeyGenerator`
- **复杂对象**: 使用 `JSONKeyGenerator`
- **多租户/环境隔离**: 使用 `ChainKeyGenerators` 组合

### Q: 缓存 TTL 设置多长合适？
- 根据数据更新频率决定
- 更新频繁：1-5 分钟
- 较少更新：10-30 分钟
- 几乎不变：1 小时以上

### Q: 何时清除缓存？
- 数据更新时使用 `WithCacheEvict` 或 `WithMultiCacheEvict`
- 数据删除时使用 `WithMultiCacheEvict`
- 装饰器会自动异步清除，无需手动管理

### Q: 缓存命中率多少算正常？
- 一般场景：60-80% 算正常
- 热点数据：90% 以上
- 如果低于 50%，需要优化缓存策略

### Q: 异步清除缓存安全吗？
- **安全**：框架使用 `sync.WaitGroup` 跟踪异步操作
- `Manager.Unregister()` 和 `CloseAll()` 会等待所有异步操作完成
- 测试通过 race 检测，无竞态条件

## 参考文档

- **KeyGenerator 详细文档**: `docs/keygen.md`
- **优化文档**: `docs/cache_optimization.md`
- **使用示例**: `examples/cache/cache_example.go`
- **KeyGenerator 示例**: `examples/cache/keygen_example.go`
- **单元测试**: `test/cache_test.go`、`test/keygen_test.go`

## 总结

本缓存框架提供了：
- ✅ **类型安全**（泛型）
- ✅ **高性能**（分片锁、批量操作、纳秒级 KeyGenerator）
- ✅ **易用性**（装饰器模式，类似 Spring Cache）
- ✅ **可扩展**（高阶函数 KeyGenerator，装饰器组合）
- ✅ **多策略**（内存、Redis、多级）
- ✅ **自动管理**（过期清理、异步操作同步、统计）
- ✅ **灵活性**（6种 KeyGenerator + 自定义）
- ✅ **安全性**（WaitGroup 同步，race 检测通过）

**推荐使用装饰器模式（WithCache、WithCacheEvict、WithMultiCacheEvict）+ 高阶函数 KeyGenerator**，获得最佳的开发体验和性能！
