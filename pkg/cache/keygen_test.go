package cache_test

import (
	"microvibe-go/pkg/cache"
	"strings"
	"testing"
)

// ========================================
// 基础 KeyGenerator 测试
// ========================================

func TestDefaultKeyGenerator(t *testing.T) {
	gen := cache.DefaultKeyGenerator()

	tests := []struct {
		name     string
		prefix   string
		args     []interface{}
		expected string
	}{
		{
			name:     "无参数",
			prefix:   "user:id",
			args:     []interface{}{},
			expected: "user:id",
		},
		{
			name:     "单个参数",
			prefix:   "user:id",
			args:     []interface{}{123},
			expected: "user:id:123",
		},
		{
			name:     "多个参数",
			prefix:   "user:search",
			args:     []interface{}{"alice", 25, true},
			expected: "user:search:alice:25:true",
		},
		{
			name:     "字符串参数",
			prefix:   "session",
			args:     []interface{}{"abc-def-123"},
			expected: "session:abc-def-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)
			if result != tt.expected {
				t.Errorf("期望 %s, 得到 %s", tt.expected, result)
			}
		})
	}
}

func TestJSONKeyGenerator(t *testing.T) {
	gen := cache.JSONKeyGenerator()

	tests := []struct {
		name     string
		prefix   string
		args     []interface{}
		contains string // 检查是否包含特定字符串
	}{
		{
			name:     "无参数",
			prefix:   "user:query",
			args:     []interface{}{},
			contains: "user:query",
		},
		{
			name:     "简单参数",
			prefix:   "user:id",
			args:     []interface{}{123},
			contains: "123",
		},
		{
			name:   "对象参数",
			prefix: "user:search",
			args: []interface{}{
				map[string]interface{}{
					"name": "alice",
					"age":  25,
				},
			},
			contains: "alice",
		},
		{
			name:     "多个参数",
			prefix:   "user:filter",
			args:     []interface{}{"alice", 25},
			contains: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)
			if !strings.Contains(result, tt.prefix) {
				t.Errorf("结果应该包含前缀 %s", tt.prefix)
			}
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("结果应该包含 %s, 得到 %s", tt.contains, result)
			}
		})
	}
}

func TestMD5KeyGenerator(t *testing.T) {
	gen := cache.MD5KeyGenerator()

	tests := []struct {
		name   string
		prefix string
		args   []interface{}
	}{
		{
			name:   "无参数",
			prefix: "user:hash",
			args:   []interface{}{},
		},
		{
			name:   "单个参数",
			prefix: "user:id",
			args:   []interface{}{123},
		},
		{
			name:   "长字符串参数",
			prefix: "user:token",
			args:   []interface{}{"very-long-token-string-12345678901234567890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)

			// MD5 生成的 key 应该包含前缀
			if !strings.HasPrefix(result, tt.prefix) {
				t.Errorf("结果应该以 %s 开头, 得到 %s", tt.prefix, result)
			}

			// MD5 hash 长度是 32 个字符
			parts := strings.Split(result, ":")
			if len(tt.args) > 0 && len(parts) > 0 {
				hash := parts[len(parts)-1]
				if len(hash) != 32 {
					t.Errorf("MD5 hash 长度应该是 32, 得到 %d", len(hash))
				}
			}
		})
	}
}

func TestSHA256KeyGenerator(t *testing.T) {
	gen := cache.SHA256KeyGenerator()

	tests := []struct {
		name   string
		prefix string
		args   []interface{}
	}{
		{
			name:   "无参数",
			prefix: "user:secure",
			args:   []interface{}{},
		},
		{
			name:   "单个参数",
			prefix: "user:token",
			args:   []interface{}{"secret-data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)

			// SHA256 生成的 key 应该包含前缀
			if !strings.HasPrefix(result, tt.prefix) {
				t.Errorf("结果应该以 %s 开头, 得到 %s", tt.prefix, result)
			}

			// SHA256 hash 长度是 64 个字符
			parts := strings.Split(result, ":")
			if len(tt.args) > 0 && len(parts) > 0 {
				hash := parts[len(parts)-1]
				if len(hash) != 64 {
					t.Errorf("SHA256 hash 长度应该是 64, 得到 %d", len(hash))
				}
			}
		})
	}
}

func TestPrefixOnlyKeyGenerator(t *testing.T) {
	gen := cache.PrefixOnlyKeyGenerator()

	tests := []struct {
		name     string
		prefix   string
		args     []interface{}
		expected string
	}{
		{
			name:     "无参数",
			prefix:   "global:config",
			args:     []interface{}{},
			expected: "global:config",
		},
		{
			name:     "有参数但被忽略",
			prefix:   "global:config",
			args:     []interface{}{123, "abc", true},
			expected: "global:config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)
			if result != tt.expected {
				t.Errorf("期望 %s, 得到 %s", tt.expected, result)
			}
		})
	}
}

func TestCustomKeyGenerator(t *testing.T) {
	// 自定义生成器：只使用第一个参数
	gen := cache.CustomKeyGenerator(func(prefix string, args ...interface{}) string {
		if len(args) > 0 {
			return prefix + ":" + args[0].(string)
		}
		return prefix
	})

	tests := []struct {
		name     string
		prefix   string
		args     []interface{}
		expected string
	}{
		{
			name:     "无参数",
			prefix:   "custom",
			args:     []interface{}{},
			expected: "custom",
		},
		{
			name:     "单个参数",
			prefix:   "custom",
			args:     []interface{}{"first"},
			expected: "custom:first",
		},
		{
			name:     "多个参数但只用第一个",
			prefix:   "custom",
			args:     []interface{}{"first", "second", "third"},
			expected: "custom:first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen(tt.prefix, tt.args...)
			if result != tt.expected {
				t.Errorf("期望 %s, 得到 %s", tt.expected, result)
			}
		})
	}
}

// ========================================
// 装饰器 KeyGenerator 测试
// ========================================

func TestWithUpperCase(t *testing.T) {
	gen := cache.WithUpperCase(cache.DefaultKeyGenerator())

	result := gen("user:id", 123)
	expected := "USER:ID:123"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

func TestWithLowerCase(t *testing.T) {
	gen := cache.WithLowerCase(cache.DefaultKeyGenerator())

	result := gen("USER:ID", 123)
	expected := "user:id:123"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

func TestWithPrefix(t *testing.T) {
	gen := cache.WithPrefix("app:v1:", cache.DefaultKeyGenerator())

	result := gen("user:id", 123)
	expected := "app:v1:user:id:123"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

func TestWithSuffix(t *testing.T) {
	gen := cache.WithSuffix(":v1", cache.DefaultKeyGenerator())

	result := gen("user:id", 123)
	expected := "user:id:123:v1"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

func TestChainKeyGenerators(t *testing.T) {
	// 链式组合：小写 + 添加前缀
	gen := cache.ChainKeyGenerators(
		cache.DefaultKeyGenerator(),
		cache.WithLowerCase,
		func(g cache.KeyGenerator) cache.KeyGenerator {
			return cache.WithPrefix("app:", g)
		},
	)

	result := gen("USER:ID", 123)
	expected := "app:user:id:123"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

func TestChainKeyGeneratorsComplex(t *testing.T) {
	// 复杂链式组合：小写 + 前缀 + 后缀
	gen := cache.ChainKeyGenerators(
		cache.DefaultKeyGenerator(),
		cache.WithLowerCase,
		func(g cache.KeyGenerator) cache.KeyGenerator {
			return cache.WithPrefix("myapp:", g)
		},
		func(g cache.KeyGenerator) cache.KeyGenerator {
			return cache.WithSuffix(":prod", g)
		},
	)

	result := gen("USER:ID", 123)
	expected := "myapp:user:id:123:prod"

	if result != expected {
		t.Errorf("期望 %s, 得到 %s", expected, result)
	}
}

// ========================================
// 一致性和稳定性测试
// ========================================

func TestKeyGeneratorConsistency(t *testing.T) {
	// 测试相同输入生成相同的 key
	gen := cache.DefaultKeyGenerator()

	key1 := gen("user:id", 123)
	key2 := gen("user:id", 123)

	if key1 != key2 {
		t.Errorf("相同输入应该生成相同的 key: %s != %s", key1, key2)
	}
}

func TestMD5KeyGeneratorConsistency(t *testing.T) {
	gen := cache.MD5KeyGenerator()

	key1 := gen("user:token", "secret-data")
	key2 := gen("user:token", "secret-data")

	if key1 != key2 {
		t.Errorf("相同输入应该生成相同的 MD5 hash: %s != %s", key1, key2)
	}
}

func TestSHA256KeyGeneratorConsistency(t *testing.T) {
	gen := cache.SHA256KeyGenerator()

	key1 := gen("user:token", "secret-data")
	key2 := gen("user:token", "secret-data")

	if key1 != key2 {
		t.Errorf("相同输入应该生成相同的 SHA256 hash: %s != %s", key1, key2)
	}
}

func TestJSONKeyGeneratorConsistency(t *testing.T) {
	gen := cache.JSONKeyGenerator()

	// 注意：JSON 序列化可能因为 map 的顺序不同而不同
	// 所以这里使用数组确保顺序一致
	args := []interface{}{"alice", 25}

	key1 := gen("user:search", args...)
	key2 := gen("user:search", args...)

	if key1 != key2 {
		t.Errorf("相同输入应该生成相同的 JSON key: %s != %s", key1, key2)
	}
}

// ========================================
// 性能基准测试
// ========================================

func BenchmarkDefaultKeyGenerator(b *testing.B) {
	gen := cache.DefaultKeyGenerator()
	for i := 0; i < b.N; i++ {
		gen("user:id", 123)
	}
}

func BenchmarkJSONKeyGenerator(b *testing.B) {
	gen := cache.JSONKeyGenerator()
	for i := 0; i < b.N; i++ {
		gen("user:query", "alice", 25)
	}
}

func BenchmarkMD5KeyGenerator(b *testing.B) {
	gen := cache.MD5KeyGenerator()
	for i := 0; i < b.N; i++ {
		gen("user:token", "secret-data")
	}
}

func BenchmarkSHA256KeyGenerator(b *testing.B) {
	gen := cache.SHA256KeyGenerator()
	for i := 0; i < b.N; i++ {
		gen("user:token", "secret-data")
	}
}

func BenchmarkWithUpperCase(b *testing.B) {
	gen := cache.WithUpperCase(cache.DefaultKeyGenerator())
	for i := 0; i < b.N; i++ {
		gen("user:id", 123)
	}
}

func BenchmarkChainKeyGenerators(b *testing.B) {
	gen := cache.ChainKeyGenerators(
		cache.DefaultKeyGenerator(),
		cache.WithLowerCase,
		func(g cache.KeyGenerator) cache.KeyGenerator {
			return cache.WithPrefix("app:", g)
		},
	)
	for i := 0; i < b.N; i++ {
		gen("USER:ID", 123)
	}
}
