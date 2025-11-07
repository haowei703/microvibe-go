# 项目架构说明

## 概述

MicroVibe-Go 采用标准的三层架构设计，类似于 Spring Web 的分层模式，实现了**展示层（Handler）**、**业务逻辑层（Service）**、**数据访问层（Repository）**的清晰分离。

## 架构图

```
┌──────────────────────────────────────┐
│         HTTP Request (Gin)           │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│      Handler 层 (展示层)              │
│  - 处理 HTTP 请求和响应               │
│  - 参数验证                          │
│  - 调用 Service 层                   │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│     Service 层 (业务逻辑层)           │
│  - 业务逻辑处理                      │
│  - 事务管理                          │
│  - 调用 Repository 层                │
│  - 日志记录                          │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│   Repository 层 (数据访问层)          │
│  - 数据库 CRUD 操作                  │
│  - 数据持久化                        │
│  - 查询封装                          │
└──────────────────┬───────────────────┘
                   ↓
┌──────────────────────────────────────┐
│        Database (PostgreSQL)         │
└──────────────────────────────────────┘
```

## 各层职责

### 1. Model 层（数据模型）

**位置**: `internal/model/`

**职责**:
- 定义数据库表结构
- ORM 映射（GORM）
- 数据验证标签

**示例**:
```go
// User 用户模型
type User struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    Username  string `gorm:"uniqueIndex;size:50;not null" json:"username"`
    Password  string `gorm:"size:255;not null" json:"-"`
    Email     string `gorm:"uniqueIndex;size:100" json:"email"`
    Nickname  string `gorm:"size:50" json:"nickname"`
    // ...
}
```

**特点**:
- 使用 GORM 标签定义字段约束
- JSON 序列化控制
- 软删除支持

---

### 2. Repository 层（数据访问层）

**位置**: `internal/repository/`

**职责**:
- 封装数据库操作
- 提供 CRUD 接口
- 查询逻辑封装
- 数据库事务

**设计模式**:
- 接口 + 实现的方式
- 依赖注入

**示例**:
```go
// UserRepository 用户数据访问层接口
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id uint) (*model.User, error)
    FindByUsername(ctx context.Context, username string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    // ...
}

// userRepositoryImpl 实现
type userRepositoryImpl struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
    logger.Debug("创建用户", zap.String("username", user.Username))

    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        logger.Error("创建用户失败", zap.Error(err))
        return err
    }

    logger.Info("用户创建成功", zap.Uint("user_id", user.ID))
    return nil
}
```

**特点**:
- 接口定义规范
- 日志记录完善
- Context 传递
- 错误处理统一

---

### 3. Service 层（业务逻辑层）

**位置**: `internal/service/`

**职责**:
- 业务逻辑处理
- 调用 Repository 进行数据操作
- 事务管理
- 业务规则验证
- 日志记录

**示例**:
```go
// UserService 用户服务层接口
type UserService interface {
    Register(ctx context.Context, req *RegisterRequest) (*model.User, error)
    Login(ctx context.Context, req *LoginRequest) (*model.User, error)
    GetUserByID(ctx context.Context, userID uint) (*model.User, error)
    // ...
}

// userServiceImpl 实现
type userServiceImpl struct {
    userRepo   repository.UserRepository
    followRepo repository.FollowRepository
}

func NewUserService(userRepo repository.UserRepository, followRepo repository.FollowRepository) UserService {
    return &userServiceImpl{
        userRepo:   userRepo,
        followRepo: followRepo,
    }
}

func (s *userServiceImpl) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
    logger.Info("用户注册请求", zap.String("username", req.Username))

    // 检查用户名是否已存在
    existUser, err := s.userRepo.FindByUsername(ctx, req.Username)
    if err == nil && existUser != nil {
        return nil, errors.New("用户名已存在")
    }

    // 加密密码
    hashedPassword, err := utils.HashPassword(req.Password)
    if err != nil {
        return nil, errors.New("密码加密失败")
    }

    // 创建用户
    user := &model.User{
        Username: req.Username,
        Password: hashedPassword,
        Email:    req.Email,
        // ...
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }

    logger.Info("用户注册成功", zap.Uint("user_id", user.ID))
    return user, nil
}
```

**特点**:
- 业务逻辑集中
- 依赖多个 Repository
- 统一的错误处理
- 完善的日志记录

---

### 4. Handler 层（展示层/控制器）

**位置**: `internal/handler/`

**职责**:
- 处理 HTTP 请求
- 参数验证和绑定
- 调用 Service 层
- 组装响应数据
- 错误处理和响应

**示例**:
```go
// UserHandler 用户处理器
type UserHandler struct {
    userService service.UserService
    cfg         *config.Config
}

func NewUserHandler(userService service.UserService, cfg *config.Config) *UserHandler {
    return &UserHandler{
        userService: userService,
        cfg:         cfg,
    }
}

func (h *UserHandler) Register(c *gin.Context) {
    var req service.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.InvalidParam(c, "参数错误: "+err.Error())
        return
    }

    user, err := h.userService.Register(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, response.CodeError, err.Error())
        return
    }

    // 生成 Token
    token, err := utils.GenerateToken(user.ID, user.Username, h.cfg.JWT.Secret, h.cfg.JWT.Expire)
    if err != nil {
        response.ServerError(c, "生成Token失败")
        return
    }

    response.Success(c, gin.H{
        "user":  user,
        "token": token,
    })
}
```

**特点**:
- 简洁明了
- 参数验证
- 统一响应格式
- 错误处理规范

---

## 日志系统

### Zap 日志框架

项目使用 Uber 的 **Zap** 日志框架，这是 Go 社区最流行、性能最高的日志库。

**位置**: `pkg/logger/`

**特性**:
- 高性能（零内存分配）
- 结构化日志
- 多种日志级别（Debug、Info、Warn、Error、Fatal）
- 支持 JSON 和 Console 两种输出格式
- 自动日志文件切割

**使用示例**:
```go
import (
    "microvibe-go/pkg/logger"
    "go.uber.org/zap"
)

// 基础日志
logger.Info("用户登录成功", zap.Uint("user_id", user.ID))
logger.Error("数据库连接失败", zap.Error(err))

// 带多个字段
logger.Debug("处理请求",
    zap.String("method", "GET"),
    zap.String("path", "/api/users"),
    zap.Int("status", 200))
```

**配置**:
- **开发环境**: Console 格式，Debug 级别，彩色输出
- **生产环境**: JSON 格式，Info 级别，输出到文件

---

## 依赖注入

项目采用**构造函数注入**的方式实现依赖注入：

```go
// 在 router.Setup 中初始化各层
func Setup(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
    // 1. 初始化 Repository 层
    userRepo := repository.NewUserRepository(db)
    followRepo := repository.NewFollowRepository(db)

    // 2. 初始化 Service 层（注入 Repository）
    userService := service.NewUserService(userRepo, followRepo)

    // 3. 初始化 Handler 层（注入 Service）
    userHandler := handler.NewUserHandler(userService, cfg)

    // 4. 注册路由
    v1.POST("/auth/register", userHandler.Register)
    // ...
}
```

**优点**:
- 清晰的依赖关系
- 易于测试（可以 Mock）
- 松耦合

---

## 错误处理

### 统一错误响应

**位置**: `pkg/response/`

```go
// 成功响应
response.Success(c, data)

// 错误响应
response.Error(c, code, message)
response.InvalidParam(c, "参数错误")
response.Unauthorized(c, "未登录")
response.NotFound(c, "资源不存在")
response.ServerError(c, "服务器错误")

// 分页响应
response.PageSuccess(c, list, total, page, pageSize)
```

### 响应格式

```json
{
    "code": 0,
    "message": "success",
    "data": { ... }
}
```

---

## 数据库迁移

### 自动迁移

**位置**: `internal/database/migrate.go`

```bash
# 执行迁移
make migrate

# 或
go run cmd/migrate/main.go
```

**功能**:
- 自动创建所有表
- 创建索引
- 填充初始数据（分类、礼物等）

---

## 配置管理

使用 **Viper** 进行配置管理：

**位置**: `configs/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: "8080"
  mode: "debug"

database:
  host: "localhost"
  port: "5432"
  user: "postgres"
  password: "postgres"
  dbname: "microvibe"

jwt:
  secret: "your-secret-key"
  expire: 24
```

**特性**:
- 支持环境变量覆盖
- 默认值配置
- 多环境配置

---

## 项目结构

```
microvibe-go/
├── cmd/
│   ├── server/              # 应用入口
│   └── migrate/             # 数据库迁移工具
├── internal/
│   ├── model/               # 数据模型（Model 层）
│   ├── repository/          # 数据访问层（Repository 层）
│   ├── service/             # 业务逻辑层（Service 层）
│   ├── handler/             # HTTP 处理器（Handler 层）
│   ├── middleware/          # 中间件
│   ├── router/              # 路由配置
│   ├── algorithm/           # 推荐算法引擎
│   ├── config/              # 配置管理
│   └── database/            # 数据库连接
├── pkg/
│   ├── logger/              # 日志工具
│   ├── response/            # 统一响应格式
│   └── utils/               # 工具函数
└── configs/                 # 配置文件
```

---

## 代码规范

### 命名规范

- **接口**: 大写字母开头，如 `UserRepository`
- **实现**: 小写字母开头 + Impl 后缀，如 `userRepositoryImpl`
- **构造函数**: `New` + 类型名，如 `NewUserRepository`

### 注释规范

- 所有公共接口必须有中文注释
- 说明函数功能、参数、返回值

```go
// Create 创建用户
// ctx: 上下文
// user: 用户对象
// 返回: error
func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error
```

### 日志规范

- 关键操作必须记录日志
- 使用结构化日志（zap.Field）
- Debug: 调试信息
- Info: 正常操作
- Warn: 警告信息
- Error: 错误信息

```go
logger.Info("用户登录成功",
    zap.Uint("user_id", user.ID),
    zap.String("username", user.Username))
```

---

## 测试建议

### 单元测试

每层都应该有独立的单元测试：

```go
// Repository 层测试
func TestUserRepository_Create(t *testing.T) {
    // 使用 test database 或 mock
}

// Service 层测试（Mock Repository）
func TestUserService_Register(t *testing.T) {
    // 使用 Mock Repository
}

// Handler 层测试（Mock Service）
func TestUserHandler_Register(t *testing.T) {
    // 使用 httptest
}
```

---

## 性能优化

### 1. Repository 层

- 使用预加载（Preload）减少查询次数
- 合理使用索引
- 批量操作

### 2. Service 层

- 避免 N+1 查询
- 使用事务保证一致性
- 缓存频繁查询的数据

### 3. Handler 层

- 参数验证前置
- 使用中间件减少重复代码
- 合理的响应结构

---

## 扩展性

### 添加新功能

1. 在 `model/` 中定义数据模型
2. 在 `repository/` 中实现数据访问接口
3. 在 `service/` 中实现业务逻辑
4. 在 `handler/` 中实现 HTTP 处理
5. 在 `router/` 中注册路由

### 示例：添加评论功能

```bash
# 1. 定义模型（已有 model/social.go）
# 2. 创建 Repository
internal/repository/comment_repository.go

# 3. 创建 Service
internal/service/comment_service.go

# 4. 创建 Handler
internal/handler/comment.go

# 5. 注册路由
internal/router/router.go
```

---

## 总结

MicroVibe-Go 采用清晰的三层架构：

1. **Model 层**: 数据模型定义
2. **Repository 层**: 数据访问，封装数据库操作
3. **Service 层**: 业务逻辑，调用 Repository
4. **Handler 层**: HTTP 处理，调用 Service

配合 **Zap 日志框架**、**依赖注入**、**统一错误处理**，形成了一个结构清晰、易于维护和扩展的项目架构。
