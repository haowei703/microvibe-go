package cache

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// KeyGenerator 缓存键生成器函数类型
// 接收前缀和参数，返回完整的缓存键
type KeyGenerator func(prefix string, args ...interface{}) string

// ========================================
// 内置的 Key 生成器工厂函数（高阶函数）
// ========================================

// DefaultKeyGenerator 默认的 key 生成器
// 使用简单的字符串拼接：prefix:arg1:arg2:arg3
//
// 示例:
//
//	generator := cache.DefaultKeyGenerator()
//	key := generator("user:id", 123)  // "user:id:123"
//	key := generator("user:name", "alice")  // "user:name:alice"
func DefaultKeyGenerator() KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		if len(args) == 0 {
			return prefix
		}

		// 简单拼接：prefix:arg1:arg2:...
		key := prefix
		for _, arg := range args {
			key += fmt.Sprintf(":%v", arg)
		}
		return key
	}
}

// JSONKeyGenerator 基于 JSON 序列化的 key 生成器
// 将参数序列化为 JSON 后生成 key，适用于复杂对象
//
// 示例:
//
//	generator := cache.JSONKeyGenerator()
//	key := generator("user:query", map[string]interface{}{
//	    "name": "alice",
//	    "age": 25,
//	})
//	// "user:query:{"age":25,"name":"alice"}"
func JSONKeyGenerator() KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		if len(args) == 0 {
			return prefix
		}

		// 如果只有一个参数，直接序列化
		if len(args) == 1 {
			data, err := json.Marshal(args[0])
			if err != nil {
				// 序列化失败，回退到默认方式
				return fmt.Sprintf("%s:%v", prefix, args[0])
			}
			return fmt.Sprintf("%s:%s", prefix, string(data))
		}

		// 多个参数，序列化为数组
		data, err := json.Marshal(args)
		if err != nil {
			// 序列化失败，回退到默认方式
			return DefaultKeyGenerator()(prefix, args...)
		}
		return fmt.Sprintf("%s:%s", prefix, string(data))
	}
}

// MD5KeyGenerator 基于 MD5 哈希的 key 生成器
// 适用于参数较长或包含特殊字符的场景，生成固定长度的 key
//
// 示例:
//
//	generator := cache.MD5KeyGenerator()
//	key := generator("user:query", "very-long-username-123456789")
//	// "user:query:5d41402abc4b2a76b9719d911017c592"
func MD5KeyGenerator() KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		if len(args) == 0 {
			return prefix
		}

		// 将所有参数转为字符串并拼接
		var parts []string
		for _, arg := range args {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		combined := strings.Join(parts, ":")

		// 计算 MD5 哈希
		hash := md5.Sum([]byte(combined))
		return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hash[:]))
	}
}

// SHA256KeyGenerator 基于 SHA256 哈希的 key 生成器
// 比 MD5 更安全，适用于对安全性有要求的场景
//
// 示例:
//
//	generator := cache.SHA256KeyGenerator()
//	key := generator("user:token", "secret-data")
//	// "user:token:2c26b46b68ffc68ff99b453c1d30413413422d706..."
func SHA256KeyGenerator() KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		if len(args) == 0 {
			return prefix
		}

		// 将所有参数转为字符串并拼接
		var parts []string
		for _, arg := range args {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		combined := strings.Join(parts, ":")

		// 计算 SHA256 哈希
		hash := sha256.Sum256([]byte(combined))
		return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hash[:]))
	}
}

// PrefixOnlyKeyGenerator 仅使用前缀的 key 生成器
// 忽略所有参数，适用于全局缓存或单例缓存的场景
//
// 示例:
//
//	generator := cache.PrefixOnlyKeyGenerator()
//	key := generator("global:config", 123, "abc")  // "global:config"
func PrefixOnlyKeyGenerator() KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		return prefix
	}
}

// CustomKeyGenerator 自定义 key 生成器
// 允许用户提供完全自定义的生成逻辑
//
// 示例:
//
//	generator := cache.CustomKeyGenerator(func(prefix string, args ...interface{}) string {
//	    // 自定义逻辑：只使用第一个参数
//	    if len(args) > 0 {
//	        return fmt.Sprintf("%s:%v", prefix, args[0])
//	    }
//	    return prefix
//	})
func CustomKeyGenerator(fn KeyGenerator) KeyGenerator {
	return fn
}

// ========================================
// 组合生成器（高阶函数组合）
// ========================================

// WithUpperCase 将生成的 key 转为大写（装饰器模式）
//
// 示例:
//
//	generator := cache.WithUpperCase(cache.DefaultKeyGenerator())
//	key := generator("user:id", 123)  // "USER:ID:123"
func WithUpperCase(gen KeyGenerator) KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		key := gen(prefix, args...)
		return strings.ToUpper(key)
	}
}

// WithLowerCase 将生成的 key 转为小写（装饰器模式）
//
// 示例:
//
//	generator := cache.WithLowerCase(cache.DefaultKeyGenerator())
//	key := generator("USER:ID", 123)  // "user:id:123"
func WithLowerCase(gen KeyGenerator) KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		key := gen(prefix, args...)
		return strings.ToLower(key)
	}
}

// WithPrefix 在生成的 key 前添加额外前缀（装饰器模式）
//
// 示例:
//
//	generator := cache.WithPrefix("app:v1:", cache.DefaultKeyGenerator())
//	key := generator("user:id", 123)  // "app:v1:user:id:123"
func WithPrefix(extraPrefix string, gen KeyGenerator) KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		key := gen(prefix, args...)
		return extraPrefix + key
	}
}

// WithSuffix 在生成的 key 后添加后缀（装饰器模式）
//
// 示例:
//
//	generator := cache.WithSuffix(":v1", cache.DefaultKeyGenerator())
//	key := generator("user:id", 123)  // "user:id:123:v1"
func WithSuffix(suffix string, gen KeyGenerator) KeyGenerator {
	return func(prefix string, args ...interface{}) string {
		key := gen(prefix, args...)
		return key + suffix
	}
}

// ChainKeyGenerators 链式组合多个生成器（函数组合）
//
// 示例:
//
//	generator := cache.ChainKeyGenerators(
//	    cache.DefaultKeyGenerator(),
//	    cache.WithLowerCase,
//	    cache.WithPrefix("app:"),
//	)
//	key := generator("USER:ID", 123)  // "app:user:id:123"
func ChainKeyGenerators(base KeyGenerator, decorators ...func(KeyGenerator) KeyGenerator) KeyGenerator {
	result := base
	for _, decorator := range decorators {
		result = decorator(result)
	}
	return result
}

// ========================================
// 便捷函数
// ========================================

// getKeyGenerator 获取 key 生成器，如果为 nil 则返回默认生成器
func getKeyGenerator(gen KeyGenerator) KeyGenerator {
	if gen == nil {
		return DefaultKeyGenerator()
	}
	return gen
}
