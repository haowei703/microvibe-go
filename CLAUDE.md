# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

MicroVibe-Go 是一个基于 AI 推荐算法的多端短视频平台后端系统，对标抖音的核心功能。项目采用 Go 语言开发，集成了自研的推荐算法引擎，支持视频上传、推荐、社交互动、直播等完整功能。

## ⚠️ 开发前必读

**在开始任何功能开发之前，请先阅读 `PROGRESS.md` 文件！**

- `PROGRESS.md` 记录了项目所有功能模块的实现状态和完成度
- 开发新功能前，必须检查该功能是否已有代码实现，避免重复开发
- 每次完成新任务后，必须更新 `PROGRESS.md` 的相应章节
- 文件中包含详细的开发优先级建议和待实现功能清单

## 常用命令

```bash
# 构建和运行
make build          # 编译应用（输出到 ./main）
make run            # 直接运行应用
make clean          # 清理构建产物

# 数据库
make migrate        # 执行数据库迁移和种子数据初始化

# 测试
make test           # 运行所有测试

# Docker
make docker-build   # 构建 Docker 镜像
make docker-up      # 启动所有服务（PostgreSQL、Redis、应用）
make docker-down    # 停止所有服务
make docker-logs    # 查看服务日志

# 代码质量（重要！）
make fmt                # 格式化所有 Go 代码（gofmt + goimports）
make pre-commit-install # 安装 pre-commit hooks
make pre-commit-run     # 手动运行所有 pre-commit 检查
```

## 核心架构

### 三层架构设计（类似 Spring Web）

项目采用标准的 **Model-Repository-Service-Handler** 四层架构：

```
HTTP Request (Gin)
      ↓
Handler 层 (internal/handler/)
  - 处理 HTTP 请求和响应
  - 参数验证（使用 binding tags）
  - 调用 Service 层
  - 错误处理和响应格式化
      ↓
Service 层 (internal/service/)
  - 业务逻辑处理
  - 事务管理
  - 调用 Repository 层
  - Zap 日志记录
      ↓
Repository 层 (internal/repository/)
  - 数据库 CRUD 操作
  - 查询封装
  - 数据持久化
      ↓
Model 层 (internal/model/)
  - GORM 数据模型定义
      ↓
PostgreSQL 数据库
```

### 依赖注入模式

项目使用**构造函数注入**实现依赖注入（参考 `internal/router/router.go`）：

```go
// 1. 初始化 Repository 层
userRepo := repository.NewUserRepository(db)
followRepo := repository.NewFollowRepository(db)

// 2. 初始化 Service 层（注入 Repository）
userService := service.NewUserService(userRepo, followRepo)

// 3. 初始化 Handler 层（注入 Service）
userHandler := handler.NewUserHandler(userService, cfg)
```

### 接口设计模式

所有层都使用**接口 + 实现**的方式：

```go
// Repository 接口定义
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id uint) (*model.User, error)
    // ...
}

// 实现
type userRepositoryImpl struct {
    db *gorm.DB
}

// 构造函数返回接口类型
func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepositoryImpl{db: db}
}
```

**重要**: Handler 中注入 Service 时，使用接口类型而不是指针：
```go
// 正确 ✅
type UserHandler struct {
    userService service.UserService  // 接口类型
}

// 错误 ❌
type UserHandler struct {
    userService *service.UserService  // 指针类型会导致编译错误
}
```

## 日志系统

### Zap 日志框架

项目使用 Uber 的 **Zap** 高性能日志框架（`pkg/logger/`）：

```go
import (
    "microvibe-go/pkg/logger"
    "go.uber.org/zap"
)

// 使用示例
logger.Info("用户登录成功",
    zap.Uint("user_id", user.ID),
    zap.String("username", user.Username))

logger.Error("数据库操作失败", zap.Error(err))
logger.Debug("处理请求", zap.String("method", "GET"))
logger.Warn("用户名已存在", zap.String("username", username))
```

**日志级别**:
- `Debug`: 调试信息（仅开发环境）
- `Info`: 正常操作（推荐用于关键业务流程）
- `Warn`: 警告信息
- `Error`: 错误信息

**环境配置**:
- **开发环境**: Console 格式，Debug 级别，彩色输出
- **生产环境**: JSON 格式，Info 级别，输出到文件

## 推荐算法架构

项目的核心特性是自研的推荐算法引擎（`internal/algorithm/`），采用四层架构：

### 1. 召回层 (recommend/recall.go)
从海量视频中快速召回候选集，包含 5 种召回策略：
- 协同过滤召回（基于用户行为相似度）
- 内容召回（基于标签、分类匹配）
- 热门召回（基于热度分数）
- 关注召回（用户关注的作者视频）
- 新视频召回（冷启动策略）

### 2. 特征工程层 (feature/engineer.go)
提取用户和视频特征，使用 Redis 缓存：
- 用户特征：观看历史、互动行为、兴趣标签
- 视频特征：分类、标签、热度、质量分
- 交叉特征：用户-视频匹配度

### 3. 排序层 (rank/ranker.go)
精准排序推荐结果，多目标融合：
- CTR 预估（点击率）
- 完播率预估
- 互动率预估
- 加权融合排序

### 4. 过滤层 (filter/filter.go)
内容过滤和去重：
- 已观看视频过滤
- 相似视频去重
- 低质量内容过滤
- 黑名单过滤

## 开发规范

### 添加新功能的标准流程

1. **定义 Model**（`internal/model/`）
   ```go
   type NewFeature struct {
       ID        uint      `gorm:"primarykey" json:"id"`
       CreatedAt time.Time `json:"created_at"`
       // 字段定义...
   }
   ```

2. **创建 Repository**（`internal/repository/`）
   ```go
   type NewFeatureRepository interface {
       Create(ctx context.Context, item *model.NewFeature) error
       // 其他方法...
   }

   type newFeatureRepositoryImpl struct {
       db *gorm.DB
   }

   func NewNewFeatureRepository(db *gorm.DB) NewFeatureRepository {
       return &newFeatureRepositoryImpl{db: db}
   }
   ```

3. **创建 Service**（`internal/service/`）
   ```go
   type NewFeatureService interface {
       DoSomething(ctx context.Context, req *Request) (*Response, error)
   }

   type newFeatureServiceImpl struct {
       repo repository.NewFeatureRepository
   }

   func NewNewFeatureService(repo repository.NewFeatureRepository) NewFeatureService {
       return &newFeatureServiceImpl{repo: repo}
   }
   ```

4. **创建 Handler**（`internal/handler/`）
   ```go
   type NewFeatureHandler struct {
       service service.NewFeatureService
   }

   func NewNewFeatureHandler(service service.NewFeatureService) *NewFeatureHandler {
       return &NewFeatureHandler{service: service}
   }

   func (h *NewFeatureHandler) HandleRequest(c *gin.Context) {
       // 参数绑定
       var req Request
       if err := c.ShouldBindJSON(&req); err != nil {
           response.InvalidParam(c, "参数错误: "+err.Error())
           return
       }

       // 调用 Service
       result, err := h.service.DoSomething(c.Request.Context(), &req)
       if err != nil {
           response.Error(c, response.CodeError, err.Error())
           return
       }

       response.Success(c, result)
   }
   ```

5. **注册路由**（`internal/router/router.go`）
   ```go
   // 在 Setup 函数中初始化
   newRepo := repository.NewNewFeatureRepository(db)
   newService := service.NewNewFeatureService(newRepo)
   newHandler := handler.NewNewFeatureHandler(newService)

   // 注册路由
   v1.POST("/new-feature", authMiddleware, newHandler.HandleRequest)
   ```

6. **更新数据库迁移**（`internal/database/migrate.go`）
   ```go
   // 添加到 AutoMigrate 列表
   db.AutoMigrate(&model.NewFeature{})
   ```

### 代码规范

- **命名规范**:
  - 接口: 大写字母开头（如 `UserRepository`）
  - 实现: 小写字母开头 + `Impl` 后缀（如 `userRepositoryImpl`）
  - 构造函数: `New` + 类型名（如 `NewUserRepository`）

- **注释规范**: 所有公共接口必须有中文注释
  ```go
  // Create 创建用户
  // ctx: 上下文
  // user: 用户对象
  // 返回: error
  func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error
  ```

- **日志规范**: 在关键操作点记录日志
  ```go
  logger.Info("关键操作", zap.Uint("id", id), zap.String("action", "create"))
  logger.Error("操作失败", zap.Error(err), zap.Uint("id", id))
  ```

- **错误处理**: Service 层统一处理错误，Handler 层转换为 HTTP 响应
  ```go
  // Service 层
  if err != nil {
      logger.Error("操作失败", zap.Error(err))
      return nil, errors.New("友好的错误信息")
  }

  // Handler 层
  if err != nil {
      response.Error(c, response.CodeError, err.Error())
      return
  }
  ```

- **Context 传递**: 所有数据库操作必须传递 Context
  ```go
  func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
      return r.db.WithContext(ctx).Create(user).Error
  }
  ```

### 统一响应格式

使用 `pkg/response/` 中的统一响应函数：

```go
response.Success(c, data)                           // 成功响应
response.SuccessWithMessage(c, "操作成功", data)     // 带消息的成功响应
response.Error(c, code, message)                    // 错误响应
response.InvalidParam(c, "参数错误")                 // 参数错误
response.Unauthorized(c, "未登录")                   // 未授权
response.NotFound(c, "资源不存在")                   // 资源不存在
response.ServerError(c, "服务器错误")                // 服务器错误
response.PageSuccess(c, list, total, page, size)    // 分页响应
```

响应格式：
```json
{
    "code": 0,
    "message": "success",
    "data": { ... }
}
```

## 缓存框架

项目集成了一个高性能、功能丰富的缓存框架（`pkg/cache/`），类似于 Spring Cache。

### 核心特性

1. **泛型支持**: 使用 Go 1.18+ 泛型，类型安全，无需类型断言
2. **多种实现**: 内存缓存（LRU）、Redis 缓存、多级缓存（内存+Redis）
3. **高性能设计**: 分片锁、批量操作、异步清理、零内存分配
4. **设计模式**: 策略模式、工厂模式、单例模式、装饰器模式、组合模式

### 快速使用

#### 1. 初始化缓存（应用启动时）

```go
import "microvibe-go/pkg/cache"

// 在 main 函数中
redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
if err := cache.InitCaches(cfg, redisAddr); err != nil {
    logger.Error("初始化缓存失败", zap.Error(err))
}
defer cache.CloseCaches()
```

#### 2. 使用装饰器模式（推荐 ⭐）

**类似 Spring Cache 的 `@Cacheable` 注解，自动管理缓存，无需手动 Get/Set**

这是最推荐的方式！缓存逻辑完全透明，无需创建额外的 Cached 对象：

```go
// Repository 实现（内置缓存装饰器）
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
    // 使用 WithCache 装饰器自动管理缓存
    // - 缓存命中: 直接返回缓存结果
    // - 缓存未命中: 执行loader函数并自动设置缓存
    return cache.WithCache[*model.User](
        cache.CacheConfig{
            CacheName: "user",          // 缓存名称
            KeyPrefix: "user:id",       // 缓存键前缀
            TTL:       10 * time.Minute, // 过期时间
        },
        func() (*model.User, error) {
            // 实际的数据库查询逻辑
            var user model.User
            if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
                return nil, err
            }
            return &user, nil
        },
    )(ctx, id) // 传入context和参数，参数用于生成缓存键 "user:id:1"
}

// 更新数据时自动清除缓存（类似 Spring @CacheEvict）
func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
    // 自动清除多个相关缓存键
    keys := []string{
        fmt.Sprintf("user:id:%d", user.ID),
        fmt.Sprintf("user:username:%s", user.Username),
        fmt.Sprintf("user:email:%s", user.Email),
    }

    return cache.WithMultiCacheEvict("user", keys, func() error {
        return r.db.WithContext(ctx).Save(user).Error
    })(ctx)
}
```

**优点**：
- ✅ 无需手动调用 `Get`/`Set`/`Delete`
- ✅ 无需创建单独的 `UserRepositoryCached` 对象
- ✅ 缓存逻辑集中在方法内部，对外透明
- ✅ 自动处理缓存未命中和过期
- ✅ 支持多缓存键清除

#### 3. 使用 GetOrSet 模式（传统方式）

如果需要更灵活的控制，也可以直接使用 GetOrSet：

```go
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
    userCache, _ := cache.GetTyped[*model.User]("user")

    cacheKey := fmt.Sprintf("user:id:%d", id)
    return userCache.GetOrSet(ctx, cacheKey, func() (*model.User, error) {
        var user model.User
        if err := r.db.First(&user, id).Error; err != nil {
            return nil, err
        }
        return &user, nil
    }, 10*time.Minute)
}
```

#### 4. 创建自定义缓存

使用 Builder 模式创建：

```go
cache := cache.NewBuilder[*model.Video]().
    WithType(cache.TypeMemory).
    WithMemoryOptions(&cache.MemoryOptions{
        MaxEntries:      10000,
        CleanupInterval: 1 * time.Minute,
        EvictionPolicy:  "lru",
        ShardCount:      32,
    }).
    WithOptions(&cache.Options{
        DefaultTTL:  5 * time.Minute,
        KeyPrefix:   "video",
        EnableStats: true,
    }).
    MustBuild()
```

#### 5. 缓存类型选择

- **内存缓存** (`TypeMemory`): 单机部署，速度最快
- **Redis 缓存** (`TypeRedis`): 分布式部署，多服务共享
- **多级缓存** (`TypeMultiLevel`): 高并发场景，内存+Redis

### 缓存策略

1. **Cache-Aside（旁路缓存）**: 使用 `GetOrSet`，最常用
2. **Write-Through（写穿透）**: 更新数据时同时更新缓存
3. **Cache-Invalidation（缓存失效）**: 更新数据时删除缓存

### 装饰器 API

提供三种装饰器函数：

1. **WithCache** - 自动缓存查询（类似 `@Cacheable`）
2. **WithCacheEvict** - 自动清除单个缓存（类似 `@CacheEvict`）
3. **WithMultiCacheEvict** - 自动清除多个缓存（类似 `@Caching`）

```go
// 查询装饰器 - 自动缓存
cache.WithCache[T](config, loader)(ctx, args...)

// 单个清除装饰器
cache.WithCacheEvict(config, fn)(ctx, args...)

// 批量清除装饰器
cache.WithMultiCacheEvict(cacheName, keys, fn)(ctx)
```

### 缓存键命名规范

使用冒号 `:` 分隔层级，使用前缀区分业务：

```go
"user:id:1"           // 用户 ID 查询
"user:username:john"  // 用户名查询
"video:id:123"        // 视频详情
"video:list:hot"      // 热门视频列表
"category:all"        // 所有分类
```

### 缓存过期时间建议

```go
10 * time.Minute  // 用户信息（不常变化）
15 * time.Minute  // 视频信息（较稳定）
30 * time.Minute  // 分类、标签（很少变化）
1 * time.Minute   // 热门列表（实时性要求高）
5 * time.Minute   // 统计数据（准确性要求不高）
```

### 参考文档

- **详细使用文档**: `docs/cache.md`
- **缓存示例代码**: `examples/cache_example.go`
- **装饰器示例代码**: `examples/decorator_example.go` ⭐
- **Repository 集成示例**: `internal/repository/user_repository.go`

## 重要技术栈

- **Web 框架**: Gin
- **ORM**: GORM
- **日志**: Zap (Uber)
- **缓存框架**: 自研泛型缓存（内存/Redis/多级）
- **配置**: Viper
- **认证**: JWT-Go + bcrypt
- **数据库**: PostgreSQL 16
- **缓存存储**: Redis 7
- **容器**: Docker & Docker Compose

## 配置文件

配置文件位于 `configs/config.yaml`，可通过环境变量覆盖。

关键配置项：
- `server.mode`: "debug" 或 "release"（影响日志级别和格式）
- `jwt.secret`: JWT 密钥
- `jwt.expire`: Token 过期时间（小时）

## 注意事项

1. **进度跟踪**: 开发前必读 `PROGRESS.md`，完成任务后必须更新对应章节的完成度和功能列表
2. **格式化代码**: 提交前务必运行 `gofmt -w -s .`
3. **接口 vs 指针**: Handler/Service 中注入依赖时，使用接口类型而非指针
4. **日志记录**: 在 Repository、Service 的关键操作点添加日志
5. **Context 传递**: 所有数据库操作必须使用 `WithContext(ctx)`
6. **错误处理**: Service 层返回友好的错误信息，避免暴露底层细节
7. **数据库迁移**: 添加新 Model 后记得更新 `internal/database/migrate.go`
8. **中文注释**: 所有公共接口必须有中文注释说明
9. **缓存一致性**: 更新或删除数据时必须清除相关缓存，使用异步操作避免影响主流程
10. **缓存键设计**: 使用统一的命名规范（如 `user:id:1`），便于管理和清除
11. **缓存降级**: 缓存失败时应该降级到数据库查询，不影响正常业务流程

## Git 提交规范

本项目使用 **Pre-commit Hooks** 强制执行 Git 提交规范，确保代码质量和提交信息的一致性。

### 提交信息格式

所有 Git 提交必须遵循以下格式：

```
emoji type: subject

- change 1
- change 2
- change 3
```

**必填部分**：
- **emoji**: 表示提交类型的 emoji（必填）
- **type**: 提交类型（必填）
- **subject**: 简短描述（必填，不超过 72 字符）
- **body**: 详细修改列表（可选，使用 `-` 开头）

### Emoji 和 Type 对照表

| Emoji | Type | 用途 | 示例 |
|-------|------|------|------|
| ✨ | `feat` | 新功能 | `✨ feat: 添加用户认证功能` |
| 🐛 | `fix` | Bug 修复 | `🐛 fix: 修复登录验证错误` |
| 📝 | `docs` | 文档更新 | `📝 docs: 更新 API 文档` |
| 💄 | `style` | 代码格式 | `💄 style: 格式化代码` |
| ♻️ | `refactor` | 重构 | `♻️ refactor: 重构用户服务层` |
| ⚡ | `perf` | 性能优化 | `⚡ perf: 优化数据库查询` |
| ✅ | `test` | 测试 | `✅ test: 添加单元测试` |
| 📦 | `build` | 构建/依赖 | `📦 build: 升级依赖版本` |
| 👷 | `ci` | CI/CD | `👷 ci: 添加 GitHub Actions` |
| 🔧 | `chore` | 其他杂项 | `🔧 chore: 更新配置文件` |
| ⏪ | `revert` | 回滚 | `⏪ revert: 回滚功能` |

### 提交示例

**✅ 正确示例**：

```bash
git commit -m "✨ feat: 添加视频推荐算法"

git commit -m "🐛 fix: 修复评论点赞重复问题

- 添加 CommentLike 模型
- 创建唯一索引防止重复
- 实现幂等操作
- 添加事务回滚逻辑"

git commit -m "📝 docs: 更新缓存框架使用文档"
```

**❌ 错误示例**：

```bash
git commit -m "更新代码"              # ❌ 缺少 emoji 和 type
git commit -m "feat: 添加功能"        # ❌ 缺少 emoji
git commit -m "✨ 添加功能"           # ❌ 缺少 type
git commit -m "[✨] feat: 添加功能"   # ❌ emoji 不应该被 [] 包裹
```

### Pre-commit 检查项

每次提交前，Pre-commit 会自动执行以下检查：

**代码质量检查**：
- ✅ `go-fmt`: 自动格式化 Go 代码
- ✅ `go-build`: 确保代码可编译
- ✅ 移除文件末尾的空白字符
- ✅ 确保文件以换行符结尾
- ✅ 验证 YAML/JSON 文件格式
- ✅ 检查大文件（>1MB）
- ✅ 检测私钥泄露

**提交信息检查**：
- ✅ 必须以有效 emoji 开头
- ✅ 必须包含有效的 type（feat/fix/docs/等）
- ✅ type 后必须跟 `: ` 和描述
- ✅ 主题行长度建议 ≤ 72 字符

### 如何使用

1. **安装 Pre-commit Hooks**（首次使用）：
   ```bash
   make pre-commit-install
   ```

2. **正常提交流程**：
   ```bash
   git add .
   git commit -m "✨ feat: 添加新功能"
   ```

3. **手动运行检查**（可选）：
   ```bash
   make pre-commit-run
   ```

4. **跳过钩子**（不推荐，仅紧急情况）：
   ```bash
   git commit --no-verify -m "提交信息"
   ```

### 更多信息

详细的 Pre-commit 使用指南请参考：[Pre-commit 使用指南](docs/development/pre-commit-guide.md)

## OpenAPI 文档维护规范

本项目使用 **OpenAPI 3.0.3 规范** 管理 API 文档，所有 API 定义集中在 `openapi.json` 文件中。

### 核心原则

**⚠️ 重要：每次修改、新增或删除 API 时，必须同步更新 `openapi.json` 文件！**

### OpenAPI 文档结构

```json
{
  "openapi": "3.0.3",
  "info": { ... },           // API 基本信息
  "servers": [ ... ],        // 服务器配置
  "tags": [ ... ],          // API 标签分类
  "paths": { ... },         // API 路由定义（核心部分）
  "components": {           // 可复用的组件
    "securitySchemes": { ... },  // 认证方案
    "parameters": { ... },       // 公共参数
    "schemas": { ... },          // 数据模型
    "responses": { ... }         // 公共响应
  }
}
```

### API 修改工作流

#### 1. 新增 API

在 `openapi.json` 的 `paths` 部分添加新的端点定义：

```json
"/api/v1/your-new-endpoint": {
  "post": {
    "summary": "端点简短描述",
    "tags": ["对应的标签"],
    "security": [{"BearerAuth": []}],  // 如果需要认证
    "requestBody": {
      "required": true,
      "content": {
        "application/json": {
          "schema": {
            "$ref": "#/components/schemas/YourRequestSchema"
          }
        }
      }
    },
    "responses": {
      "200": {
        "description": "成功",
        "content": {
          "application/json": {
            "schema": {
              "allOf": [
                {"$ref": "#/components/schemas/Response"},
                {
                  "type": "object",
                  "properties": {
                    "data": {
                      "$ref": "#/components/schemas/YourResponseSchema"
                    }
                  }
                }
              ]
            }
          }
        }
      },
      "401": {"$ref": "#/components/responses/Unauthorized"}
    }
  }
}
```

#### 2. 修改已有 API

- 更新 `paths` 中对应端点的定义
- 如果修改了请求/响应结构，同步更新 `components/schemas`
- 如果修改了路径参数，更新 `parameters`
- 添加版本说明或废弃标记（如果适用）

#### 3. 删除 API

- 从 `paths` 中移除对应的端点定义
- 检查 `components/schemas` 中是否有仅该端点使用的 schema，如果有则一并删除
- 在文档变更日志中记录删除原因

#### 4. 定义新的数据模型

在 `components/schemas` 中添加：

```json
"YourModel": {
  "type": "object",
  "properties": {
    "id": {
      "type": "integer",
      "format": "uint",
      "description": "唯一标识"
    },
    "created_at": {
      "type": "string",
      "format": "date-time",
      "description": "创建时间"
    },
    "name": {
      "type": "string",
      "description": "名称"
    }
  },
  "required": ["name"]
}
```

### 标签（Tags）规范

所有 API 必须归属于一个标签，当前支持的标签：

- `健康检查` - 系统健康检查接口
- `认证` - 用户注册和登录
- `用户` - 用户信息管理
- `视频` - 视频上传、推荐、互动
- `评论` - 评论管理
- `直播` - 直播间管理和互动
- `OAuth` - 第三方登录认证
- `搜索` - 搜索功能
- `消息` - 私信聊天
- `通知` - 系统通知
- `话题` - 话题标签管理

**添加新功能模块时，需要在 `tags` 数组中先定义标签。**

### 认证标记规范

- **需要登录的接口**：必须添加 `"security": [{"BearerAuth": []}]`
- **可选登录的接口**：不添加 security 字段，在描述中说明
- **公开接口**：不添加 security 字段

### 路径参数和查询参数规范

#### 路径参数（Path Parameters）

```json
"parameters": [
  {
    "name": "id",
    "in": "path",
    "required": true,
    "schema": {
      "type": "integer"
    },
    "description": "资源ID"
  }
]
```

#### 查询参数（Query Parameters）

- 使用公共参数引用分页参数：`{"$ref": "#/components/parameters/Page"}`
- 自定义查询参数直接定义在接口的 `parameters` 数组中

### 响应格式规范

#### 成功响应（统一格式）

```json
"200": {
  "description": "成功",
  "content": {
    "application/json": {
      "schema": {
        "allOf": [
          {"$ref": "#/components/schemas/Response"},
          {
            "type": "object",
            "properties": {
              "data": {
                // 具体的响应数据类型
              }
            }
          }
        ]
      }
    }
  }
}
```

#### 错误响应（使用公共响应）

```json
"400": {"$ref": "#/components/responses/InvalidParam"},
"401": {"$ref": "#/components/responses/Unauthorized"},
"404": {"$ref": "#/components/responses/NotFound"}
```

### 检查工具和验证

#### 1. 在线验证

使用 Swagger Editor 验证 OpenAPI 文档：
```bash
# 访问 https://editor.swagger.io/
# 将 openapi.json 的内容粘贴到编辑器中
```

#### 2. 命令行验证（推荐）

```bash
# 使用 make 命令验证（会检查格式和规范）
make openapi-validate

# 注意：验证工具可能会显示一些警告（warnings），这些警告是关于最佳实践的建议，
# 不影响文档的正确性和使用。主要关注错误（errors）即可。
```

#### 3. 对比检查

定期运行对比检查脚本，确保 `router.go` 和 `openapi.json` 一致：

```bash
# 查看对比报告
cat docs/api-comparison.md
```

### 文档查看

#### 本地预览

使用 Swagger UI 预览 API 文档：

```bash
# 方式一：使用 Docker（推荐）
docker run -p 8081:8080 -e SWAGGER_JSON=/app/openapi.json -v $(pwd)/openapi.json:/app/openapi.json swaggerapi/swagger-ui

# 方式二：使用 VS Code 插件
# 安装 "OpenAPI (Swagger) Editor" 插件
# 打开 openapi.json 文件，右键选择 "Preview Swagger"
```

访问 `http://localhost:8081` 查看文档。

#### 在线查看

1. 将 `openapi.json` 上传到项目仓库
2. 使用 Swagger UI 在线查看器：`https://petstore.swagger.io/`
3. 输入你的 openapi.json URL

### 最佳实践

1. **描述要详细**：
   - `summary` 字段：简短描述（不超过 50 字符）
   - `description` 字段：详细说明（包括业务逻辑、注意事项等）

2. **使用引用减少重复**：
   - 公共的请求/响应 schema 定义在 `components/schemas` 中
   - 公共的参数定义在 `components/parameters` 中
   - 公共的响应定义在 `components/responses` 中

3. **保持一致性**：
   - 所有响应都使用统一的 `Response` 包装
   - 分页接口都使用 `PageData` schema
   - 错误响应都使用公共的错误响应定义

4. **类型要准确**：
   - 整数使用 `"type": "integer"`
   - 枚举使用 `"enum": [...]`
   - 日期时间使用 `"type": "string", "format": "date-time"`
   - ID 类型使用 `"type": "integer", "format": "uint"`

5. **及时更新**：
   - ✅ 修改 Handler 后立即更新 openapi.json
   - ✅ 添加新路由后立即添加文档定义
   - ✅ 删除接口后立即删除文档定义
   - ✅ 修改数据模型后立即更新 schema 定义

### 常见错误

❌ **错误示例 1：忘记更新 openapi.json**
```go
// 在 router.go 中添加了新路由
v1.POST("/new-endpoint", handler.NewEndpoint)
// 但忘记在 openapi.json 中添加对应的定义 ❌
```

❌ **错误示例 2：路径不一致**
```json
// openapi.json 中定义的路径
"/api/v1/users/:id"  // ❌ 错误：应该使用 {id}

// 正确的写法
"/api/v1/users/{id}"  // ✅ 正确
```

❌ **错误示例 3：缺少认证标记**
```json
// 需要登录的接口，但忘记添加 security 字段
{
  "post": {
    "summary": "创建视频",
    "tags": ["视频"]
    // ❌ 缺少 "security": [{"BearerAuth": []}]
  }
}
```

### 参考资源

- **OpenAPI 官方规范**: https://spec.openapis.org/oas/v3.0.3
- **Swagger 编辑器**: https://editor.swagger.io/
- **OpenAPI 最佳实践**: https://oai.github.io/Documentation/best-practices.html
- **项目 API 对比报告**: `docs/api-comparison.md`
- 每次任务必须先阅读CLAUDE.md，不要生成没用的总结文档
