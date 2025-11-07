package main

import (
	"context"
	"fmt"
	"microvibe-go/pkg/cache"
	"microvibe-go/pkg/logger"
	"time"
)

// User 示例用户结构
type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

// UserRepository 示例Repository
type UserRepository struct {
	// 模拟数据库
	users map[uint]*User
}

// NewUserRepository 创建Repository实例
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: map[uint]*User{
			1: {ID: 1, Username: "alice", Email: "alice@example.com", Age: 25},
			2: {ID: 2, Username: "bob", Email: "bob@example.com", Age: 30},
			3: {ID: 3, Username: "charlie", Email: "charlie@example.com", Age: 35},
		},
	}
}

// FindByID 使用 WithCache 装饰器（类似 Spring @Cacheable）
// 查询方法自动使用缓存，无需手动管理
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	fmt.Printf("查询用户: id=%d\n", id)

	// 使用 WithCache 装饰器自动管理缓存
	// 缓存命中: 直接返回缓存结果
	// 缓存未命中: 执行loader函数并自动设置缓存
	return cache.WithCache(
		cache.CacheConfig{
			CacheName: "user",          // 缓存名称
			KeyPrefix: "user:id",       // 缓存键前缀
			TTL:       5 * time.Minute, // 缓存过期时间
		},
		func() (*User, error) {
			// 模拟数据库查询
			fmt.Println("  -> 从数据库查询（缓存未命中）")
			time.Sleep(100 * time.Millisecond) // 模拟查询延迟

			user, ok := r.users[id]
			if !ok {
				return nil, fmt.Errorf("用户不存在: id=%d", id)
			}
			return user, nil
		},
	)(ctx, id) // 传入context和参数，参数用于生成缓存键
}

// FindByUsername 使用 WithCache 装饰器查询用户名
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	fmt.Printf("查询用户: username=%s\n", username)

	return cache.WithCache(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:username",
			TTL:       5 * time.Minute,
		},
		func() (*User, error) {
			fmt.Println("  -> 从数据库查询（缓存未命中）")
			time.Sleep(100 * time.Millisecond)

			for _, user := range r.users {
				if user.Username == username {
					return user, nil
				}
			}
			return nil, fmt.Errorf("用户不存在: username=%s", username)
		},
	)(ctx, username)
}

// UpdateAge 使用 WithCacheEvict 装饰器（类似 Spring @CacheEvict）
// 更新操作自动清除缓存
func (r *UserRepository) UpdateAge(ctx context.Context, id uint, age int) error {
	fmt.Printf("更新用户年龄: id=%d, age=%d\n", id, age)

	// 使用 WithCacheEvict 装饰器自动清除缓存
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
		},
		func() error {
			// 模拟数据库更新
			fmt.Println("  -> 更新数据库")
			user, ok := r.users[id]
			if !ok {
				return fmt.Errorf("用户不存在")
			}
			user.Age = age
			return nil
		},
	)(ctx, id) // 执行完成后自动清除 "user:id:1" 缓存
}

// Update 使用 WithMultiCacheEvict 装饰器清除多个缓存键
// 当一个对象有多个缓存键时（如ID、用户名、邮箱）
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	fmt.Printf("更新用户: id=%d\n", user.ID)

	// 清除多个相关缓存键
	keys := []string{
		fmt.Sprintf("user:id:%d", user.ID),
		fmt.Sprintf("user:username:%s", user.Username),
		fmt.Sprintf("user:email:%s", user.Email),
	}

	return cache.WithMultiCacheEvict("user", keys, func() error {
		fmt.Println("  -> 更新数据库")
		r.users[user.ID] = user
		return nil
	})(ctx) // 执行完成后自动清除所有相关缓存
}

func main() {
	// 初始化logger
	logger.InitLogger("info")

	fmt.Println("=== 缓存框架使用示例 ===")

	// 示例 1: 创建内存缓存
	example1MemoryCache()

	// 示例 2: 使用 GetOrSet 模式
	example2GetOrSet()

	// 示例 3: 批量操作
	example3BatchOperations()

	// 示例 4: 使用装饰器
	example4Decorator()

	// 示例 5: 缓存统计
	example5Statistics()

	// 示例 6: 使用缓存管理器
	example6Manager()

	// 示例 7: 使用装饰器模式（类似 Spring Cache）
	example7RepositoryDecorator()
}

// 示例 1: 创建内存缓存
func example1MemoryCache() {
	fmt.Println("--- 示例 1: 创建内存缓存 ---")

	// 使用 Builder 创建缓存
	userCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(&cache.MemoryOptions{
			MaxEntries:      1000,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  "lru",
			ShardCount:      32,
		}).
		WithOptions(&cache.Options{
			DefaultTTL:    5 * time.Minute,
			KeyPrefix:     "user",
			EnableStats:   true,
			EnableLogging: false,
		}).
		MustBuild()

	ctx := context.Background()

	// 设置缓存
	user := &User{ID: 1, Username: "张三", Email: "zhangsan@example.com"}
	err := userCache.Set(ctx, "1", user, 0)
	if err != nil {
		fmt.Printf("设置缓存失败: %v\n", err)
		return
	}
	fmt.Println("✓ 设置缓存成功")

	// 获取缓存
	cachedUser, err := userCache.Get(ctx, "1")
	if err != nil {
		fmt.Printf("获取缓存失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 获取缓存成功: %+v\n", cachedUser)

	// 检查缓存是否存在
	exists, _ := userCache.Exists(ctx, "1")
	fmt.Printf("✓ 缓存是否存在: %v\n", exists)

	// 删除缓存
	userCache.Delete(ctx, "1")
	exists, _ = userCache.Exists(ctx, "1")
	fmt.Printf("✓ 删除后缓存是否存在: %v\n\n", exists)

	userCache.Close()
}

// 示例 2: 使用 GetOrSet 模式（推荐）
func example2GetOrSet() {
	fmt.Println("--- 示例 2: 使用 GetOrSet 模式 ---")

	userCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		WithOptions(&cache.Options{
			DefaultTTL:  5 * time.Minute,
			EnableStats: true,
		}).
		MustBuild()

	ctx := context.Background()

	// 模拟数据库查询函数
	loadUserFromDB := func(id uint) func() (*User, error) {
		return func() (*User, error) {
			fmt.Printf("  → 从数据库加载用户 %d\n", id)
			time.Sleep(100 * time.Millisecond) // 模拟数据库查询延迟
			return &User{
				ID:       id,
				Username: fmt.Sprintf("user%d", id),
				Email:    fmt.Sprintf("user%d@example.com", id),
			}, nil
		}
	}

	// 第一次调用 - 缓存未命中，从数据库加载
	fmt.Println("第一次调用 GetOrSet:")
	user1, err := userCache.GetOrSet(ctx, "100", loadUserFromDB(100), 5*time.Minute)
	if err != nil {
		fmt.Printf("GetOrSet 失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 获取用户: %+v\n", user1)

	// 第二次调用 - 缓存命中，不会查询数据库
	fmt.Println("第二次调用 GetOrSet:")
	user2, err := userCache.GetOrSet(ctx, "100", loadUserFromDB(100), 5*time.Minute)
	if err != nil {
		fmt.Printf("GetOrSet 失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 获取用户（来自缓存）: %+v\n\n", user2)

	userCache.Close()
}

// 示例 3: 批量操作
func example3BatchOperations() {
	fmt.Println("--- 示例 3: 批量操作 ---")

	userCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		MustBuild()

	ctx := context.Background()

	// 批量设置
	users := map[string]*User{
		"1": {ID: 1, Username: "user1", Email: "user1@example.com"},
		"2": {ID: 2, Username: "user2", Email: "user2@example.com"},
		"3": {ID: 3, Username: "user3", Email: "user3@example.com"},
	}

	err := userCache.SetMulti(ctx, users, 5*time.Minute)
	if err != nil {
		fmt.Printf("批量设置失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 批量设置 %d 个用户\n", len(users))

	// 批量获取
	keys := []string{"1", "2", "3", "4"} // 注意：key "4" 不存在
	result, err := userCache.GetMulti(ctx, keys)
	if err != nil {
		fmt.Printf("批量获取失败: %v\n", err)
		return
	}
	fmt.Printf("✓ 批量获取结果（请求 %d 个，找到 %d 个）:\n", len(keys), len(result))
	for key, user := range result {
		fmt.Printf("  - %s: %+v\n", key, user)
	}

	// 批量删除
	err = userCache.DeleteMulti(ctx, []string{"1", "2"})
	if err != nil {
		fmt.Printf("批量删除失败: %v\n", err)
		return
	}
	fmt.Println("✓ 批量删除成功")

	userCache.Close()
}

// 示例 4: 使用装饰器
func example4Decorator() {
	fmt.Println("--- 示例 4: 使用装饰器 ---")

	// 创建基础缓存
	baseCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		MustBuild()

	// 添加日志装饰器
	loggedCache := cache.NewLoggingDecorator[*User](baseCache)

	ctx := context.Background()

	user := &User{ID: 1, Username: "测试用户", Email: "test@example.com"}

	fmt.Println("操作将记录详细日志:")

	// 设置缓存（会记录日志）
	loggedCache.Set(ctx, "1", user, 5*time.Minute)

	// 获取缓存（会记录日志）
	loggedCache.Get(ctx, "1")

	// 删除缓存（会记录日志）
	loggedCache.Delete(ctx, "1")

	fmt.Println()

	loggedCache.Close()
}

// 示例 5: 缓存统计
func example5Statistics() {
	fmt.Println("--- 示例 5: 缓存统计 ---")

	userCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		WithOptions(&cache.Options{
			EnableStats: true,
		}).
		MustBuild()

	ctx := context.Background()

	// 执行一些操作
	for i := 1; i <= 10; i++ {
		user := &User{ID: uint(i), Username: fmt.Sprintf("user%d", i), Email: fmt.Sprintf("user%d@example.com", i)}
		userCache.Set(ctx, fmt.Sprintf("%d", i), user, 5*time.Minute)
	}

	// 模拟命中和未命中
	for i := 1; i <= 10; i++ {
		userCache.Get(ctx, fmt.Sprintf("%d", i)) // 命中
	}

	for i := 11; i <= 15; i++ {
		userCache.Get(ctx, fmt.Sprintf("%d", i)) // 未命中
	}

	// 获取统计信息
	stats := userCache.GetStats()
	fmt.Printf("✓ 缓存统计:\n")
	fmt.Printf("  - 命中次数: %d\n", stats.Hits)
	fmt.Printf("  - 未命中次数: %d\n", stats.Misses)
	fmt.Printf("  - 设置次数: %d\n", stats.Sets)
	fmt.Printf("  - 命中率: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("  - 缓存项数量: %d\n\n", stats.ItemCount)

	userCache.Close()
}

// 示例 6: 使用缓存管理器
func example6Manager() {
	fmt.Println("--- 示例 6: 使用缓存管理器 ---")

	manager := cache.GetManager()

	// 创建并注册多个缓存实例
	userCache := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		WithName("users"). // 指定名称，自动注册
		MustBuild()

	// 从管理器获取缓存
	retrievedCache, err := cache.GetTyped[*User]("users")
	if err != nil {
		fmt.Printf("获取缓存失败: %v\n", err)
		return
	}

	ctx := context.Background()
	user := &User{ID: 1, Username: "manager_test", Email: "test@example.com"}
	retrievedCache.Set(ctx, "1", user, 5*time.Minute)

	cachedUser, _ := retrievedCache.Get(ctx, "1")
	fmt.Printf("✓ 通过管理器获取缓存: %+v\n", cachedUser)

	// 列出所有缓存
	names := manager.List()
	fmt.Printf("✓ 已注册的缓存: %v\n", names)

	// 关闭指定缓存
	manager.Unregister("users")
	fmt.Println("✓ 已注销缓存")

	userCache.Close()
}

// 示例 7: 使用装饰器模式（类似 Spring Cache）
func example7RepositoryDecorator() {
	fmt.Println("--- 示例 7: 使用装饰器模式（类似 Spring Cache）---")

	// 初始化logger
	logger.InitLogger("debug")

	fmt.Println("========================================")
	fmt.Println("缓存装饰器示例 - 类似 Spring Cache")
	fmt.Println("========================================")

	// 初始化缓存
	userCache, _ := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(&cache.MemoryOptions{
			MaxEntries:      1000,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  "lru",
			ShardCount:      16,
		}).
		WithOptions(&cache.Options{
			DefaultTTL:  5 * time.Minute,
			KeyPrefix:   "user",
			EnableStats: true,
		}).
		WithName("user").
		Build()

	cache.GetManager().Register("user", userCache)

	ctx := context.Background()
	repo := NewUserRepository()

	// ========================================
	// 示例 1: @Cacheable 效果
	// ========================================
	fmt.Println("\n【示例 1】使用 WithCache 装饰器")
	fmt.Println("-----------------------------------")

	// 第一次查询 - 缓存未命中，从数据库查询
	fmt.Println("\n第一次查询:")
	user1, _ := repo.FindByID(ctx, 1)
	fmt.Printf("结果: %+v\n", user1)

	// 第二次查询 - 缓存命中，直接返回
	fmt.Println("\n第二次查询（相同ID）:")
	user2, _ := repo.FindByID(ctx, 1)
	fmt.Printf("结果: %+v\n", user2)

	// 查询其他用户
	fmt.Println("\n查询其他用户:")
	user3, _ := repo.FindByUsername(ctx, "bob")
	fmt.Printf("结果: %+v\n", user3)

	// ========================================
	// 示例 2: @CacheEvict 效果
	// ========================================
	fmt.Println("\n【示例 2】使用 WithCacheEvict 装饰器")
	fmt.Println("-----------------------------------")

	// 更新用户年龄 - 自动清除缓存
	fmt.Println("\n更新用户年龄:")
	repo.UpdateAge(ctx, 1, 26)

	// 再次查询 - 缓存已被清除，需要重新从数据库查询
	fmt.Println("\n更新后查询:")
	user4, _ := repo.FindByID(ctx, 1)
	fmt.Printf("结果: %+v (age已更新)\n", user4)

	// ========================================
	// 示例 3: @CacheEvict(allEntries) 效果
	// ========================================
	fmt.Println("\n【示例 3】使用 WithMultiCacheEvict 装饰器")
	fmt.Println("-----------------------------------")

	// 先查询，建立缓存
	fmt.Println("\n建立多个缓存:")
	repo.FindByID(ctx, 2)
	repo.FindByUsername(ctx, "bob")

	// 更新用户 - 自动清除多个相关缓存
	fmt.Println("\n更新用户（清除多个缓存键）:")
	updatedUser := &User{
		ID:       2,
		Username: "bob",
		Email:    "bob@newexample.com",
		Age:      31,
	}
	repo.Update(ctx, updatedUser)

	// 再次查询 - 所有相关缓存已被清除
	fmt.Println("\n更新后查询:")
	user5, _ := repo.FindByID(ctx, 2)
	fmt.Printf("结果: %+v\n", user5)

	// ========================================
	// 显示缓存统计
	// ========================================
	fmt.Println("\n【缓存统计】")
	fmt.Println("-----------------------------------")
	stats := userCache.GetStats()
	fmt.Printf("命中次数: %d\n", stats.Hits)
	fmt.Printf("未命中次数: %d\n", stats.Misses)
	fmt.Printf("命中率: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("总请求: %d\n", stats.Hits+stats.Misses)

	fmt.Println("\n========================================")
	fmt.Println("✅ 装饰器模式让缓存管理更简单！")
	fmt.Println("   - 无需手动调用 Get/Set")
	fmt.Println("   - 无需创建额外的 Cached 对象")
	fmt.Println("   - 自动处理缓存未命中")
	fmt.Println("   - 自动清除过期缓存")
	fmt.Println("========================================")

	userCache.Close()
}
