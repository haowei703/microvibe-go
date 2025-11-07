package errors

import (
	"errors"
	"fmt"
)

// ========================================
// 核心错误类型和方法
// ========================================

// AppError 应用错误结构
type AppError struct {
	Code    int    // 错误码
	Message string // 错误信息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// ========================================
// 错误码定义
// ========================================

const (
	// 通用错误码 (1000-1999)
	CodeUnknown            = 1000 // 未知错误
	CodeInvalidParam       = 1001 // 参数错误
	CodeInternalError      = 1002 // 内部错误
	CodeNotImplemented     = 1003 // 未实现
	CodeUnauthorized       = 1004 // 未授权
	CodeForbidden          = 1005 // 禁止访问
	CodeTooManyRequests    = 1006 // 请求过多
	CodeServiceUnavailable = 1007 // 服务不可用

	// 数据库错误码 (2000-2999)
	CodeDBError             = 2000 // 数据库错误
	CodeRecordNotFound      = 2001 // 记录不存在
	CodeDuplicateKey        = 2002 // 重复键
	CodeConstraintViolation = 2003 // 约束违反
	CodeConnectionFailed    = 2004 // 连接失败

	// 业务错误码 (3000-3999)
	CodeUserNotFound      = 3001 // 用户不存在
	CodeUserAlreadyExists = 3002 // 用户已存在
	CodeInvalidPassword   = 3003 // 密码错误
	CodeVideoNotFound     = 3004 // 视频不存在
	CodeCommentNotFound   = 3005 // 评论不存在
	CodePermissionDenied  = 3006 // 权限不足

	// 缓存错误码 (4000-4999)
	CodeCacheError   = 4000 // 缓存错误
	CodeCacheMiss    = 4001 // 缓存未命中
	CodeCacheExpired = 4002 // 缓存过期
)

// ========================================
// 错误构造函数
// ========================================

// New 创建新错误
func New(message string) error {
	return errors.New(message)
}

// Errorf 格式化错误
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Wrap 包装错误
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf 格式化包装错误
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// NewAppError 创建应用错误
func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewAppErrorWithCause 创建带原因的应用错误
func NewAppErrorWithCause(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     cause,
	}
}

// ========================================
// 通用错误判断方法
// ========================================

// Is 判断错误是否匹配目标错误
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 尝试将错误转换为目标类型
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// IsAppError 判断是否为应用错误
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetCode 获取错误码
func GetCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeUnknown
}

// GetMessage 获取错误信息
func GetMessage(err error) string {
	if err == nil {
		return ""
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return err.Error()
}

// ========================================
// 预定义错误
// ========================================

var (
	// 通用错误
	ErrUnknown            = NewAppError(CodeUnknown, "未知错误")
	ErrInvalidParam       = NewAppError(CodeInvalidParam, "参数错误")
	ErrInternalError      = NewAppError(CodeInternalError, "内部错误")
	ErrNotImplemented     = NewAppError(CodeNotImplemented, "功能未实现")
	ErrUnauthorized       = NewAppError(CodeUnauthorized, "未授权")
	ErrForbidden          = NewAppError(CodeForbidden, "禁止访问")
	ErrTooManyRequests    = NewAppError(CodeTooManyRequests, "请求过多")
	ErrServiceUnavailable = NewAppError(CodeServiceUnavailable, "服务不可用")

	// 数据库错误
	ErrDBError             = NewAppError(CodeDBError, "数据库错误")
	ErrRecordNotFound      = NewAppError(CodeRecordNotFound, "记录不存在")
	ErrDuplicateKey        = NewAppError(CodeDuplicateKey, "记录已存在")
	ErrConstraintViolation = NewAppError(CodeConstraintViolation, "数据约束违反")
	ErrConnectionFailed    = NewAppError(CodeConnectionFailed, "数据库连接失败")

	// 业务错误
	ErrUserNotFound      = NewAppError(CodeUserNotFound, "用户不存在")
	ErrUserAlreadyExists = NewAppError(CodeUserAlreadyExists, "用户已存在")
	ErrInvalidPassword   = NewAppError(CodeInvalidPassword, "密码错误")
	ErrVideoNotFound     = NewAppError(CodeVideoNotFound, "视频不存在")
	ErrCommentNotFound   = NewAppError(CodeCommentNotFound, "评论不存在")
	ErrPermissionDenied  = NewAppError(CodePermissionDenied, "权限不足")

	// 缓存错误
	ErrCacheError   = NewAppError(CodeCacheError, "缓存错误")
	ErrCacheMiss    = NewAppError(CodeCacheMiss, "缓存未命中")
	ErrCacheExpired = NewAppError(CodeCacheExpired, "缓存已过期")
)
