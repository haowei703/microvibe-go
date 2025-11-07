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

// testUser 测试用的用户结构
type testUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

// setupCache 创建测试用的内存缓存
func setupCache(t *testing.T) cache.Cache[*testUser] {
	t.Helper()

	// 初始化logger（如果尚未初始化）
	logger.InitLogger("error") // 使用error级别，减少测试输出

	userCache, err := cache.NewBuilder[*testUser]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(&cache.MemoryOptions{
			MaxEntries:      100,
			CleanupInterval: 1 * time.Minute,
			EvictionPolicy:  "lru",
			ShardCount:      4,
		}).
		WithOptions(&cache.Options{
			DefaultTTL:  5 * time.Minute,
			KeyPrefix:   "test:user",
			EnableStats: true,
		}).
		Build()

	if err != nil {
		t.Fatalf("创建测试缓存失败: %v", err)
	}

	return userCache
}

// teardownCache 清理测试缓存
func teardownCache(t *testing.T, c cache.Cache[*testUser]) {
	t.Helper()
	if err := c.Close(); err != nil {
		// 忽略"缓存已关闭"错误
		if !errors.Is(err, cache.ErrCacheClosed) {
			t.Errorf("关闭缓存失败: %v", err)
		}
	}
}

// ========================================
// 缓存基本操作测试
// ========================================

func TestCache_Set(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		user    *testUser
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "正常设置缓存",
			key:     "user:1",
			user:    &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25},
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "设置缓存TTL为0使用默认值",
			key:     "user:2",
			user:    &testUser{ID: 2, Username: "bob", Email: "bob@example.com", Age: 30},
			ttl:     0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Set(ctx, tt.key, tt.user, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证缓存已设置
				exists, _ := c.Exists(ctx, tt.key)
				if !exists {
					t.Error("缓存应该已设置，但不存在")
				}
			}
		})
	}
}

func TestCache_Get(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}
	c.Set(ctx, "user:1", user, 5*time.Minute)

	tests := []struct {
		name    string
		key     string
		want    *testUser
		wantErr bool
	}{
		{
			name:    "获取存在的缓存",
			key:     "user:1",
			want:    user,
			wantErr: false,
		},
		{
			name:    "获取不存在的缓存",
			key:     "user:999",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Get(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.ID != tt.want.ID {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCache_Delete(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}
	c.Set(ctx, "user:1", user, 5*time.Minute)

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "删除存在的缓存",
			key:     "user:1",
			wantErr: false,
		},
		{
			name:    "删除不存在的缓存",
			key:     "user:999",
			wantErr: true, // 删除不存在的key会返回错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Delete(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 只在成功删除时验证缓存已删除
			if !tt.wantErr {
				exists, _ := c.Exists(ctx, tt.key)
				if exists {
					t.Error("缓存应该已删除，但仍然存在")
				}
			}
		})
	}
}

func TestCache_Exists(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}
	c.Set(ctx, "user:1", user, 5*time.Minute)

	tests := []struct {
		name       string
		key        string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "检查存在的缓存",
			key:        "user:1",
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "检查不存在的缓存",
			key:        "user:999",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := c.Exists(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.wantExists {
				t.Errorf("Exists() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	loadCount := 0
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}

	loader := func() (*testUser, error) {
		loadCount++
		return user, nil
	}

	t.Run("首次调用执行loader", func(t *testing.T) {
		loadCount = 0
		got, err := c.GetOrSet(ctx, "user:1", loader, 5*time.Minute)
		if err != nil {
			t.Fatalf("GetOrSet() error = %v", err)
		}
		if loadCount != 1 {
			t.Errorf("loader应该被调用1次, 实际%d次", loadCount)
		}
		if got.ID != user.ID {
			t.Errorf("GetOrSet() = %v, want %v", got, user)
		}
	})

	t.Run("第二次调用使用缓存", func(t *testing.T) {
		loadCount = 0
		got, err := c.GetOrSet(ctx, "user:1", loader, 5*time.Minute)
		if err != nil {
			t.Fatalf("GetOrSet() error = %v", err)
		}
		if loadCount != 0 {
			t.Errorf("loader不应该被调用, 实际调用%d次", loadCount)
		}
		if got.ID != user.ID {
			t.Errorf("GetOrSet() = %v, want %v", got, user)
		}
	})

	t.Run("loader返回错误", func(t *testing.T) {
		expectedErr := errors.New("loader error")
		errorLoader := func() (*testUser, error) {
			return nil, expectedErr
		}

		_, err := c.GetOrSet(ctx, "user:error", errorLoader, 5*time.Minute)
		if err == nil {
			t.Error("期望返回错误，但没有错误")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("期望错误 = %v, 实际 = %v", expectedErr, err)
		}
	})
}

func TestCache_Clear(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()

	// 设置多个缓存项
	for i := 1; i <= 5; i++ {
		user := &testUser{
			ID:       uint(i),
			Username: fmt.Sprintf("user%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Age:      20 + i,
		}
		c.Set(ctx, fmt.Sprintf("user:%d", i), user, 5*time.Minute)
	}

	// 验证缓存已设置
	stats := c.GetStats()
	if stats.ItemCount == 0 {
		t.Error("缓存项应该存在")
	}

	// 清空缓存
	err := c.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// 验证缓存已清空
	stats = c.GetStats()
	if stats.ItemCount != 0 {
		t.Errorf("清空后缓存项应该为0, 实际 = %d", stats.ItemCount)
	}
}

// ========================================
// 批量操作测试
// ========================================

func TestCache_SetMulti_GetMulti(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()

	users := map[string]*testUser{
		"user:1": {ID: 1, Username: "alice", Email: "alice@example.com", Age: 25},
		"user:2": {ID: 2, Username: "bob", Email: "bob@example.com", Age: 30},
		"user:3": {ID: 3, Username: "charlie", Email: "charlie@example.com", Age: 35},
	}

	t.Run("批量设置缓存", func(t *testing.T) {
		err := c.SetMulti(ctx, users, 5*time.Minute)
		if err != nil {
			t.Fatalf("SetMulti() error = %v", err)
		}

		// 验证所有缓存已设置
		for key := range users {
			exists, _ := c.Exists(ctx, key)
			if !exists {
				t.Errorf("键 %s 应该存在", key)
			}
		}
	})

	t.Run("批量获取缓存", func(t *testing.T) {
		keys := []string{"user:1", "user:2", "user:3", "user:999"}
		result, err := c.GetMulti(ctx, keys)
		if err != nil {
			t.Fatalf("GetMulti() error = %v", err)
		}

		// 验证结果数量（user:999不存在）
		expectedCount := 3
		if len(result) != expectedCount {
			t.Errorf("期望获取%d个用户, 实际%d个", expectedCount, len(result))
		}

		// 验证每个结果的内容
		for key, got := range result {
			want, ok := users[key]
			if !ok {
				t.Errorf("返回了意外的键: %s", key)
				continue
			}
			if got.ID != want.ID {
				t.Errorf("键 %s: 期望ID=%d, 实际ID=%d", key, want.ID, got.ID)
			}
		}
	})
}

func TestCache_DeleteMulti(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()

	// 设置缓存
	users := map[string]*testUser{
		"user:1": {ID: 1, Username: "alice", Email: "alice@example.com", Age: 25},
		"user:2": {ID: 2, Username: "bob", Email: "bob@example.com", Age: 30},
		"user:3": {ID: 3, Username: "charlie", Email: "charlie@example.com", Age: 35},
	}
	c.SetMulti(ctx, users, 5*time.Minute)

	// 批量删除
	keysToDelete := []string{"user:1", "user:2"}
	err := c.DeleteMulti(ctx, keysToDelete)
	if err != nil {
		t.Fatalf("DeleteMulti() error = %v", err)
	}

	// 验证已删除的缓存
	for _, key := range keysToDelete {
		exists, _ := c.Exists(ctx, key)
		if exists {
			t.Errorf("键 %s 应该已删除", key)
		}
	}

	// 验证未删除的缓存仍然存在
	exists, _ := c.Exists(ctx, "user:3")
	if !exists {
		t.Error("user:3 不应该被删除")
	}
}

// ========================================
// 统计信息测试
// ========================================

func TestCache_GetStats(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}

	// 清空统计
	c.Clear(ctx)

	// 执行一些操作
	c.Set(ctx, "user:1", user, 5*time.Minute)
	c.Get(ctx, "user:1")   // 命中
	c.Get(ctx, "user:999") // 未命中
	c.Delete(ctx, "user:1")

	stats := c.GetStats()

	tests := []struct {
		name  string
		field string
		got   int64
		want  int64
	}{
		{"设置次数", "Sets", stats.Sets, 1},
		{"命中次数", "Hits", stats.Hits, 1},
		{"未命中次数", "Misses", stats.Misses, 1},
		{"删除次数", "Deletes", stats.Deletes, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: 期望 = %d, 实际 = %d", tt.field, tt.want, tt.got)
			}
		})
	}

	// 验证命中率
	expectedHitRate := 0.5
	if stats.HitRate != expectedHitRate {
		t.Errorf("命中率: 期望 = %.2f, 实际 = %.2f", expectedHitRate, stats.HitRate)
	}
}

// ========================================
// 装饰器测试
// ========================================

func TestLoggingDecorator(t *testing.T) {
	baseCache := setupCache(t)
	defer teardownCache(t, baseCache)

	// 包装日志装饰器
	loggedCache := cache.NewLoggingDecorator[*testUser](baseCache)

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}

	t.Run("装饰器应该透传Set操作", func(t *testing.T) {
		err := loggedCache.Set(ctx, "user:1", user, 5*time.Minute)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	})

	t.Run("装饰器应该透传Get操作", func(t *testing.T) {
		got, err := loggedCache.Get(ctx, "user:1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("Get() = %v, want %v", got, user)
		}
	})

	t.Run("装饰器应该透传Delete操作", func(t *testing.T) {
		err := loggedCache.Delete(ctx, "user:1")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		exists, _ := loggedCache.Exists(ctx, "user:1")
		if exists {
			t.Error("缓存应该已删除")
		}
	})
}

// ========================================
// 并发测试
// ========================================

func TestCache_Concurrent(t *testing.T) {
	c := setupCache(t)
	defer teardownCache(t, c)

	ctx := context.Background()
	concurrency := 50
	done := make(chan bool, concurrency)

	// 并发写入
	t.Run("并发写入", func(t *testing.T) {
		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()
				user := &testUser{
					ID:       uint(id),
					Username: fmt.Sprintf("user%d", id),
					Email:    fmt.Sprintf("user%d@example.com", id),
					Age:      20 + id,
				}
				c.Set(ctx, fmt.Sprintf("user:%d", id), user, 5*time.Minute)
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < concurrency; i++ {
			<-done
		}

		stats := c.GetStats()
		if stats.Sets != int64(concurrency) {
			t.Errorf("Sets: 期望 = %d, 实际 = %d", concurrency, stats.Sets)
		}
	})

	// 并发读取
	t.Run("并发读取", func(t *testing.T) {
		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()
				c.Get(ctx, fmt.Sprintf("user:%d", id))
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < concurrency; i++ {
			<-done
		}

		stats := c.GetStats()
		if stats.Hits != int64(concurrency) {
			t.Errorf("Hits: 期望 = %d, 实际 = %d", concurrency, stats.Hits)
		}
	})
}

// ========================================
// 性能基准测试
// ========================================

func BenchmarkCache_Set(b *testing.B) {
	c, _ := cache.NewBuilder[*testUser]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		Build()
	defer c.Close()

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(ctx, fmt.Sprintf("user:%d", i), user, 5*time.Minute)
			i++
		}
	})
}

func BenchmarkCache_Get(b *testing.B) {
	c, _ := cache.NewBuilder[*testUser]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		Build()
	defer c.Close()

	ctx := context.Background()
	user := &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}

	// 预先设置数据
	for i := 0; i < 10000; i++ {
		c.Set(ctx, fmt.Sprintf("user:%d", i), user, 5*time.Minute)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(ctx, fmt.Sprintf("user:%d", i%10000))
			i++
		}
	})
}

func BenchmarkCache_GetOrSet(b *testing.B) {
	c, _ := cache.NewBuilder[*testUser]().
		WithType(cache.TypeMemory).
		WithMemoryOptions(cache.DefaultMemoryOptions()).
		Build()
	defer c.Close()

	ctx := context.Background()
	loader := func() (*testUser, error) {
		return &testUser{ID: 1, Username: "alice", Email: "alice@example.com", Age: 25}, nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.GetOrSet(ctx, "user:1", loader, 5*time.Minute)
		}
	})
}
