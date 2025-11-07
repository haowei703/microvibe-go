# Errors 包使用文档

## 概述

`pkg/errors` 包提供了统一的错误处理机制，封装了常见的错误类型判断和错误转换功能。该包的主要目标是：

1. **统一错误处理入口** - 不再直接使用 `errors.Is(err, gorm.ErrRecordNotFound)`，而是使用 `pkgerrors.IsNotFound(err)`
2. **错误码管理** - 所有错误都有对应的错误码，便于前端识别和国际化
3. **错误封装** - 将第三方库（如GORM）的错误转换为应用错误
4. **友好的错误信息** - 提供更友好的业务错误信息

## 包结构

```
pkg/errors/
├── errors.go           // 核心错误定义、错误码、通用错误方法
├── db_errors.go        // 数据库错误判断和转换
├── business_errors.go  // 业务错误构造和判断
└── errors_test.go      // 测试文件
```

## 快速开始

### 1. 导入

```go
import (
    pkgerrors "microvibe-go/pkg/errors"
)
```

### 2. 数据库错误判断

**旧的写法**（不推荐）：
```go
import (
    "errors"
    "gorm.io/gorm"
)

user, err := userRepo.FindByID(ctx, userID)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.New("用户不存在")
    }
    return nil, err
}
```

**新的写法**（推荐）：
```go
import (
    pkgerrors "microvibe-go/pkg/errors"
)

user, err := userRepo.FindByID(ctx, userID)
if err != nil {
    if pkgerrors.IsNotFound(err) {
        return nil, pkgerrors.ErrUserNotFound
    }
    return nil, pkgerrors.ConvertDBError(err)
}
```

### 3. 业务错误

```go
// 使用预定义错误
if user.Status != 1 {
    return pkgerrors.NewAppError(pkgerrors.CodeForbidden, "账号已被禁用")
}

// 使用业务错误构造函数
if userID == targetID {
    return pkgerrors.NewInvalidParamError("不能关注自己")
}

// 使用预定义的全局错误
if !utils.CheckPassword(password, user.Password) {
    return pkgerrors.ErrInvalidPassword
}
```

## 错误码定义

### 通用错误码 (1000-1999)

| 错误码 | 常量名 | 说明 |
|--------|--------|------|
| 1000 | `CodeUnknown` | 未知错误 |
| 1001 | `CodeInvalidParam` | 参数错误 |
| 1002 | `CodeInternalError` | 内部错误 |
| 1003 | `CodeNotImplemented` | 未实现 |
| 1004 | `CodeUnauthorized` | 未授权 |
| 1005 | `CodeForbidden` | 禁止访问 |
| 1006 | `CodeTooManyRequests` | 请求过多 |
| 1007 | `CodeServiceUnavailable` | 服务不可用 |

### 数据库错误码 (2000-2999)

| 错误码 | 常量名 | 说明 |
|--------|--------|------|
| 2000 | `CodeDBError` | 数据库错误 |
| 2001 | `CodeRecordNotFound` | 记录不存在 |
| 2002 | `CodeDuplicateKey` | 重复键 |
| 2003 | `CodeConstraintViolation` | 约束违反 |
| 2004 | `CodeConnectionFailed` | 连接失败 |

### 业务错误码 (3000-3999)

| 错误码 | 常量名 | 说明 |
|--------|--------|------|
| 3001 | `CodeUserNotFound` | 用户不存在 |
| 3002 | `CodeUserAlreadyExists` | 用户已存在 |
| 3003 | `CodeInvalidPassword` | 密码错误 |
| 3004 | `CodeVideoNotFound` | 视频不存在 |
| 3005 | `CodeCommentNotFound` | 评论不存在 |
| 3006 | `CodePermissionDenied` | 权限不足 |

### 缓存错误码 (4000-4999)

| 错误码 | 常量名 | 说明 |
|--------|--------|------|
| 4000 | `CodeCacheError` | 缓存错误 |
| 4001 | `CodeCacheMiss` | 缓存未命中 |
| 4002 | `CodeCacheExpired` | 缓存过期 |

## 核心API

### 错误构造函数

```go
// 创建简单错误
err := pkgerrors.New("错误信息")

// 格式化错误
err := pkgerrors.Errorf("用户%s不存在", username)

// 包装错误（保留原始错误链）
err := pkgerrors.Wrap(originalErr, "操作失败")

// 创建应用错误（带错误码）
err := pkgerrors.NewAppError(pkgerrors.CodeInvalidParam, "参数错误")

// 创建带原因的应用错误
err := pkgerrors.NewAppErrorWithCause(pkgerrors.CodeDBError, "数据库操作失败", originalErr)
```

### 数据库错误判断

```go
// 判断是否为记录不存在
if pkgerrors.IsNotFound(err) {
    // 处理记录不存在
}

// 判断是否为重复键错误
if pkgerrors.IsDuplicateKey(err) {
    // 处理重复键错误
}

// 判断是否为外键约束错误
if pkgerrors.IsForeignKeyViolation(err) {
    // 处理外键约束错误
}

// 判断是否为连接错误
if pkgerrors.IsConnectionError(err) {
    // 处理连接错误
}

// 判断是否为死锁错误
if pkgerrors.IsDeadlock(err) {
    // 处理死锁
}
```

### 数据库错误转换

```go
// 自动转换数据库错误为应用错误
appErr := pkgerrors.ConvertDBError(err)

// 转换记录不存在错误为特定业务错误
appErr := pkgerrors.ConvertNotFoundError(err, "user")  // 返回 ErrUserNotFound
appErr := pkgerrors.ConvertNotFoundError(err, "video") // 返回 ErrVideoNotFound

// 辅助函数：处理数据库错误
err := pkgerrors.HandleDBError(err, "查询用户")
```

### 业务错误构造

```go
// 参数错误
err := pkgerrors.NewInvalidParamError("username")

// 未授权错误
err := pkgerrors.NewUnauthorizedError("用户未登录")

// 禁止访问错误
err := pkgerrors.NewForbiddenError("权限不足")

// 用户不存在
err := pkgerrors.NewUserNotFoundError("alice")

// 视频不存在
err := pkgerrors.NewVideoNotFoundError("123")

// 权限不足
err := pkgerrors.NewPermissionDeniedError("删除视频")
```

### 业务错误判断

```go
// 判断是否为用户不存在
if pkgerrors.IsUserNotFound(err) {
    // 处理用户不存在
}

// 判断是否为未授权
if pkgerrors.IsUnauthorized(err) {
    // 处理未授权
}

// 判断是否为权限不足
if pkgerrors.IsPermissionDenied(err) {
    // 处理权限不足
}
```

### 辅助函数

```go
// 要求用户已认证
if err := pkgerrors.RequireAuth(userID); err != nil {
    return err
}

// 要求用户有特定权限
if err := pkgerrors.RequirePermission(hasPermission, "删除评论"); err != nil {
    return err
}

// 验证参数
if err := pkgerrors.ValidateParam(len(username) >= 3, "username"); err != nil {
    return err
}

// 检查记录必须不存在（用于创建操作）
err := pkgerrors.MustNotExist(err)

// 检查记录必须存在（用于更新/删除操作）
err := pkgerrors.MustExist(err, "user")
```

## 使用场景

### 场景1：Repository层

```go
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
    var user model.User
    if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
        // 统一转换数据库错误
        return nil, pkgerrors.ConvertDBError(err)
    }
    return &user, nil
}

func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        // 转换数据库错误（会自动识别重复键等）
        return pkgerrors.ConvertDBError(err)
    }
    return nil
}
```

### 场景2：Service层 - 查询

```go
func (s *userServiceImpl) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        if pkgerrors.IsNotFound(err) {
            // 转换为业务错误
            return nil, pkgerrors.ErrUserNotFound
        }
        return nil, err
    }
    return user, nil
}
```

### 场景3：Service层 - 创建

```go
func (s *userServiceImpl) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
    // 检查用户名是否已存在
    existUser, err := s.userRepo.FindByUsername(ctx, req.Username)
    if err == nil && existUser != nil {
        // 返回预定义错误
        return nil, pkgerrors.ErrUserAlreadyExists
    }

    // 创建用户
    user := &model.User{...}
    if err := s.userRepo.Create(ctx, user); err != nil {
        // 如果是重复键错误，会自动转换
        return nil, err
    }

    return user, nil
}
```

### 场景4：Service层 - 业务逻辑验证

```go
func (s *userServiceImpl) FollowUser(ctx context.Context, userID, targetID uint) error {
    // 参数验证
    if userID == targetID {
        return pkgerrors.NewInvalidParamError("不能关注自己")
    }

    // 权限验证
    if err := pkgerrors.RequireAuth(userID); err != nil {
        return err
    }

    // 检查是否已关注
    exists, err := s.followRepo.Exists(ctx, userID, targetID)
    if err != nil {
        return pkgerrors.ConvertDBError(err)
    }
    if exists {
        return pkgerrors.NewAppError(pkgerrors.CodeDuplicateKey, "已经关注过该用户")
    }

    // 创建关注关系
    return s.followRepo.Create(ctx, &model.Follow{...})
}
```

### 场景5：Handler层 - 错误响应

```go
func (h *UserHandler) GetUser(c *gin.Context) {
    userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

    user, err := h.userService.GetUserByID(c.Request.Context(), uint(userID))
    if err != nil {
        // 根据错误码返回不同的HTTP状态码
        code := pkgerrors.GetCode(err)
        message := pkgerrors.GetMessage(err)

        switch code {
        case pkgerrors.CodeUserNotFound:
            response.NotFound(c, message)
        case pkgerrors.CodeUnauthorized:
            response.Unauthorized(c, message)
        case pkgerrors.CodeForbidden:
            response.Forbidden(c, message)
        default:
            response.ServerError(c, message)
        }
        return
    }

    response.Success(c, user)
}
```

## 预定义错误

```go
// 通用错误
pkgerrors.ErrUnknown
pkgerrors.ErrInvalidParam
pkgerrors.ErrInternalError
pkgerrors.ErrNotImplemented
pkgerrors.ErrUnauthorized
pkgerrors.ErrForbidden
pkgerrors.ErrTooManyRequests
pkgerrors.ErrServiceUnavailable

// 数据库错误
pkgerrors.ErrDBError
pkgerrors.ErrRecordNotFound
pkgerrors.ErrDuplicateKey
pkgerrors.ErrConstraintViolation
pkgerrors.ErrConnectionFailed

// 业务错误
pkgerrors.ErrUserNotFound
pkgerrors.ErrUserAlreadyExists
pkgerrors.ErrInvalidPassword
pkgerrors.ErrVideoNotFound
pkgerrors.ErrCommentNotFound
pkgerrors.ErrPermissionDenied

// 缓存错误
pkgerrors.ErrCacheError
pkgerrors.ErrCacheMiss
pkgerrors.ErrCacheExpired
```

## 最佳实践

### 1. Repository层

- 所有数据库错误都使用 `ConvertDBError()` 转换
- 不要在Repository层判断业务逻辑，只负责数据访问

### 2. Service层

- 使用 `IsNotFound()` 等方法判断错误类型
- 将数据库错误转换为业务错误（如 `ErrUserNotFound`）
- 使用预定义错误或构造函数创建业务错误

### 3. Handler层

- 根据错误码返回合适的HTTP状态码
- 使用 `GetCode()` 和 `GetMessage()` 获取错误信息
- 不要直接暴露数据库错误给前端

### 4. 错误包装

- 使用 `Wrap()` 或 `Wrapf()` 包装错误，保留错误链
- 包装时添加上下文信息，便于调试

### 5. 错误日志

- 数据库错误使用 `logger.Error()` 记录
- 业务错误（如参数错误）使用 `logger.Warn()` 记录
- 在Service层记录详细日志，Handler层简化处理

## 迁移指南

### 从旧代码迁移

1. **修改导入**:
```go
// 旧的
import (
    "errors"
    "gorm.io/gorm"
)

// 新的
import (
    pkgerrors "microvibe-go/pkg/errors"
)
```

2. **替换错误判断**:
```go
// 旧的
if errors.Is(err, gorm.ErrRecordNotFound) { ... }

// 新的
if pkgerrors.IsNotFound(err) { ... }
```

3. **替换错误创建**:
```go
// 旧的
return errors.New("用户不存在")

// 新的
return pkgerrors.ErrUserNotFound
```

4. **添加错误转换**:
```go
// 旧的
return err

// 新的
return pkgerrors.ConvertDBError(err)
```

## 注意事项

1. **不要混用标准库errors和pkg/errors** - 统一使用 `pkgerrors`
2. **错误码要保持稳定** - 前端可能依赖错误码进行判断
3. **错误信息要友好** - 不要暴露数据库细节
4. **合理使用错误包装** - 保留错误链便于调试
5. **及时添加新的错误类型** - 当有新的业务错误时，在 `business_errors.go` 中添加

## 扩展

### 添加新的错误码

在 `errors.go` 中添加：

```go
const (
    // 添加新的错误码
    CodeOrderNotFound = 3007 // 订单不存在
)

var (
    // 添加预定义错误
    ErrOrderNotFound = NewAppError(CodeOrderNotFound, "订单不存在")
)
```

### 添加新的业务错误

在 `business_errors.go` 中添加：

```go
// NewOrderNotFoundError 创建订单不存在错误
func NewOrderNotFoundError(orderID string) *AppError {
    msg := "订单不存在"
    if orderID != "" {
        msg = msg + ": " + orderID
    }
    return &AppError{
        Code:    CodeOrderNotFound,
        Message: msg,
    }
}

// IsOrderNotFound 判断是否为订单不存在错误
func IsOrderNotFound(err error) bool {
    if err == nil {
        return false
    }
    return GetCode(err) == CodeOrderNotFound
}
```

## 参考

- [Go 错误处理最佳实践](https://blog.golang.org/error-handling-and-go)
- [pkg/errors](https://github.com/pkg/errors)
- GORM 错误处理文档
