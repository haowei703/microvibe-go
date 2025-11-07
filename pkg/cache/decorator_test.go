package cache_test

import (
	"context"
	"errors"
	"fmt"
	"microvibe-go/pkg/cache"
	"microvibe-go/pkg/logger"
	"testing"
	"time"
)

// User 测试用的用户结构（与cache_example.go中的User一致）
type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

// UserRepository 模拟的用户仓库
type UserRepository struct {
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

// FindByID 使用WithCache装饰器查询用户
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	return cache.WithCache[*User](
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
			TTL:       5 * time.Minute,
		},
		func() (*User, error) {
			time.Sleep(10 * time.Millisecond) // 模拟数据库查询延迟
			user, ok := r.users[id]
			if !ok {
				return nil, fmt.Errorf("用户不存在: id=%d", id)
			}
			return user, nil
		},
	)(ctx, id)
}

// FindByUsername 使用WithCache装饰器查询用户名
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	return cache.WithCache[*User](
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:username",
			TTL:       5 * time.Minute,
		},
		func() (*User, error) {
			time.Sleep(10 * time.Millisecond)
			for _, user := range r.users {
				if user.Username == username {
					return user, nil
				}
			}
			return nil, fmt.Errorf("用户不存在: username=%s", username)
		},
	)(ctx, username)
}

// UpdateAge 使用WithCacheEvict装饰器更新年龄
func (r *UserRepository) UpdateAge(ctx context.Context, id uint, age int) error {
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
		},
		func() error {
			user, ok := r.users[id]
			if !ok {
				return fmt.Errorf("用户不存在")
			}
			user.Age = age
			return nil
		},
	)(ctx, id)
}

// Update 使用WithMultiCacheEvict装饰器更新用户
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	keys := []string{
		fmt.Sprintf("user:id:%d", user.ID),
		fmt.Sprintf("user:username:%s", user.Username),
		fmt.Sprintf("user:email:%s", user.Email),
	}

	return cache.WithMultiCacheEvict("user", keys, func() error {
		r.users[user.ID] = user
		return nil
	})(ctx)
}

// ========================================
// Repository 测试辅助函数
// ========================================

// setupRepositoryTest 设置Repository测试环境
func setupRepositoryTest(t *testing.T) (*UserRepository, cache.Cache[*User]) {
	t.Helper()

	// 初始化logger（如果尚未初始化）
	logger.InitLogger("error") // 使用error级别，减少测试输出

	// 先清理可能存在的旧缓存
	if oldCache, err := cache.GetManager().Get("user"); err == nil {
		if closer, ok := oldCache.(interface{ Close() error }); ok {
			closer.Close()
		}
		cache.GetManager().Unregister("user")
	}

	// 创建缓存
	userCache, err := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(&cache.MemoryOptions{
			MaxEntries:      100,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  "lru",
			ShardCount:      4,
		}).
		WithOptions(&cache.Options{
			DefaultTTL:  5 * time.Minute,
			KeyPrefix:   "user",
			EnableStats: true,
		}).
		Build()

	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	// 注册缓存
	cache.GetManager().Register("user", userCache)

	// 创建Repository
	repo := NewUserRepository()

	return repo, userCache
}

// teardownRepositoryTest 清理Repository测试环境
func teardownRepositoryTest(t *testing.T, c cache.Cache[*User]) {
	t.Helper()
	cache.GetManager().Unregister("user")
	if err := c.Close(); err != nil {
		// 忽略"缓存已关闭"错误
		if !errors.Is(err, cache.ErrCacheClosed) {
			t.Errorf("关闭缓存失败: %v", err)
		}
	}
}

// ========================================
// UserRepository 测试
// ========================================

func TestNewUserRepository(t *testing.T) {
	repo := NewUserRepository()

	if repo == nil {
		t.Fatal("NewUserRepository 返回 nil")
	}

	if repo.users == nil {
		t.Error("users map 未初始化")
	}

	tests := []struct {
		name         string
		id           uint
		wantUsername string
	}{
		{"用户1", 1, "alice"},
		{"用户2", 2, "bob"},
		{"用户3", 3, "charlie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, ok := repo.users[tt.id]
			if !ok {
				t.Fatalf("用户 ID=%d 不存在", tt.id)
			}
			if user.Username != tt.wantUsername {
				t.Errorf("用户名: 期望 = %s, 实际 = %s", tt.wantUsername, user.Username)
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	tests := []struct {
		name         string
		id           uint
		wantUsername string
		wantErr      bool
	}{
		{
			name:         "查询存在的用户",
			id:           1,
			wantUsername: "alice",
			wantErr:      false,
		},
		{
			name:    "查询不存在的用户",
			id:      999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清空缓存和统计
			userCache.Clear(ctx)

			// 第一次查询
			user1, err := repo.FindByID(ctx, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindByID 失败: %v", err)
			}

			if user1.Username != tt.wantUsername {
				t.Errorf("用户名: 期望 = %s, 实际 = %s", tt.wantUsername, user1.Username)
			}

			// 第二次查询 - 应该从缓存获取
			user2, err := repo.FindByID(ctx, tt.id)
			if err != nil {
				t.Fatalf("第二次 FindByID 失败: %v", err)
			}

			if user2.ID != user1.ID {
				t.Error("两次查询结果不一致")
			}

			if user2.Username != user1.Username {
				t.Error("两次查询的用户名不一致")
			}
		})
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	tests := []struct {
		name      string
		username  string
		wantEmail string
		wantErr   bool
	}{
		{
			name:      "查询存在的用户名",
			username:  "alice",
			wantEmail: "alice@example.com",
			wantErr:   false,
		},
		{
			name:     "查询不存在的用户名",
			username: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.FindByUsername(ctx, tt.username)

			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindByUsername 失败: %v", err)
			}

			if user.Email != tt.wantEmail {
				t.Errorf("Email: 期望 = %s, 实际 = %s", tt.wantEmail, user.Email)
			}
		})
	}
}

func TestUserRepository_UpdateAge(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	tests := []struct {
		name       string
		id         uint
		newAge     int
		wantErr    bool
		errMessage string
	}{
		{
			name:    "更新存在的用户",
			id:      1,
			newAge:  30,
			wantErr: false,
		},
		{
			name:       "更新不存在的用户",
			id:         999,
			newAge:     30,
			wantErr:    true,
			errMessage: "用户不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先查询用户，建立缓存
			if !tt.wantErr {
				user1, _ := repo.FindByID(ctx, tt.id)
				originalAge := user1.Age

				// 更新年龄
				err := repo.UpdateAge(ctx, tt.id, tt.newAge)
				if err != nil {
					t.Fatalf("UpdateAge 失败: %v", err)
				}

				// 验证数据已更新
				if repo.users[tt.id].Age != tt.newAge {
					t.Errorf("Age: 期望 = %d, 实际 = %d", tt.newAge, repo.users[tt.id].Age)
				}

				// 等待异步缓存清除
				time.Sleep(100 * time.Millisecond)

				// 再次查询，验证返回新数据
				user2, err := repo.FindByID(ctx, tt.id)
				if err != nil {
					t.Fatalf("第二次 FindByID 失败: %v", err)
				}

				if user2.Age == originalAge {
					t.Error("缓存未被清除，仍然返回旧数据")
				}

				if user2.Age != tt.newAge {
					t.Errorf("Age: 期望 = %d, 实际 = %d", tt.newAge, user2.Age)
				}
			} else {
				err := repo.UpdateAge(ctx, tt.id, tt.newAge)
				if err == nil {
					t.Error("期望返回错误，但没有错误")
					return
				}

				if err.Error() != tt.errMessage {
					t.Errorf("错误消息: 期望 = %s, 实际 = %s", tt.errMessage, err.Error())
				}
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	// 先查询用户，建立多个缓存
	repo.FindByID(ctx, 2)
	repo.FindByUsername(ctx, "bob")

	// 更新用户
	updatedUser := &User{
		ID:       2,
		Username: "bob",
		Email:    "bob_updated@example.com",
		Age:      35,
	}

	t.Run("更新用户并清除多个缓存", func(t *testing.T) {
		err := repo.Update(ctx, updatedUser)
		if err != nil {
			t.Fatalf("Update 失败: %v", err)
		}

		// 验证数据已更新
		if repo.users[2].Email != "bob_updated@example.com" {
			t.Errorf("Email: 期望 = bob_updated@example.com, 实际 = %s", repo.users[2].Email)
		}

		// 等待异步缓存清除
		time.Sleep(100 * time.Millisecond)

		// 再次查询，验证返回新数据
		user, err := repo.FindByID(ctx, 2)
		if err != nil {
			t.Fatalf("FindByID 失败: %v", err)
		}

		if user.Email != "bob_updated@example.com" {
			t.Error("缓存未被清除，仍然返回旧数据")
		}

		if user.Age != 35 {
			t.Errorf("Age: 期望 = 35, 实际 = %d", user.Age)
		}
	})
}

// ========================================
// 缓存装饰器功能测试
// ========================================

func TestWithCache_Decorator(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	t.Run("装饰器应该缓存查询结果", func(t *testing.T) {
		// 第一次查询
		user1, err := repo.FindByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindByID 失败: %v", err)
		}

		// 验证第一个用户数据正确
		if user1.ID != 1 {
			t.Errorf("期望 ID = 1, 实际 = %d", user1.ID)
		}
		if user1.Username != "alice" {
			t.Errorf("期望 Username = alice, 实际 = %s", user1.Username)
		}

		// 第二次查询
		user2, err := repo.FindByID(ctx, 1)
		if err != nil {
			t.Fatalf("第二次 FindByID 失败: %v", err)
		}

		// 验证返回相同数据
		if user1.ID != user2.ID {
			t.Error("两次查询返回不同数据")
		}
		if user1.Username != user2.Username {
			t.Error("两次查询的用户名不一致")
		}
	})
}

func TestWithCacheEvict_Decorator(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	t.Run("装饰器应该清除缓存", func(t *testing.T) {
		// 先查询，建立缓存
		user1, _ := repo.FindByID(ctx, 1)
		originalAge := user1.Age

		// 更新年龄（应该清除缓存）
		newAge := originalAge + 5
		err := repo.UpdateAge(ctx, 1, newAge)
		if err != nil {
			t.Fatalf("UpdateAge 失败: %v", err)
		}

		// 等待异步清除
		time.Sleep(100 * time.Millisecond)

		// 再次查询（应该获取新数据，而不是缓存的旧数据）
		user2, err := repo.FindByID(ctx, 1)
		if err != nil {
			t.Fatalf("FindByID 失败: %v", err)
		}

		if user2.Age != newAge {
			t.Errorf("Age: 期望 = %d, 实际 = %d (缓存可能未被清除)", newAge, user2.Age)
		}
	})
}

func TestWithMultiCacheEvict_Decorator(t *testing.T) {
	repo, userCache := setupRepositoryTest(t)
	defer teardownRepositoryTest(t, userCache)

	ctx := context.Background()

	t.Run("装饰器应该清除多个缓存键", func(t *testing.T) {
		// 建立多个缓存
		user1, err := repo.FindByID(ctx, 2)
		if err != nil {
			t.Fatalf("FindByID 失败: %v", err)
		}
		user2, err := repo.FindByUsername(ctx, "bob")
		if err != nil {
			t.Fatalf("FindByUsername 失败: %v", err)
		}

		// 验证查询结果正确
		if user1.ID != 2 || user2.Username != "bob" {
			t.Error("查询结果不正确")
		}

		// 更新用户（应该清除多个缓存键）
		originalEmail := user1.Email
		updatedUser := &User{
			ID:       2,
			Username: "bob",
			Email:    "bob_new@example.com",
			Age:      40,
		}
		err = repo.Update(ctx, updatedUser)
		if err != nil {
			t.Fatalf("Update 失败: %v", err)
		}

		// 等待异步清除
		time.Sleep(100 * time.Millisecond)

		// 再次查询，验证获取到的是更新后的数据（说明缓存已被清除）
		user3, err := repo.FindByID(ctx, 2)
		if err != nil {
			t.Fatalf("更新后 FindByID 失败: %v", err)
		}

		// 如果缓存被正确清除，应该获取到新的 Email
		if user3.Email == originalEmail {
			t.Error("缓存未被清除，仍然返回旧数据")
		}
		if user3.Email != "bob_new@example.com" {
			t.Errorf("Email 应该已更新: 期望 = bob_new@example.com, 实际 = %s", user3.Email)
		}
	})
}

// ========================================
// 性能基准测试
// ========================================

func BenchmarkRepository_FindByID_WithCache(b *testing.B) {
	// 创建缓存
	userCache, _ := cache.NewBuilder[*User]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		Build()
	defer userCache.Close()

	cache.GetManager().Register("user", userCache)
	defer cache.GetManager().Unregister("user")

	repo := NewUserRepository()
	ctx := context.Background()

	// 预热缓存
	repo.FindByID(ctx, 1)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			repo.FindByID(ctx, 1)
		}
	})
}

func BenchmarkRepository_FindByID_NoCache(b *testing.B) {
	repo := NewUserRepository()

	// 直接查询函数（不使用缓存）
	directFind := func(id uint) (*User, error) {
		time.Sleep(10 * time.Millisecond)
		user, ok := repo.users[id]
		if !ok {
			return nil, fmt.Errorf("用户不存在")
		}
		return user, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		directFind(1)
	}
}
