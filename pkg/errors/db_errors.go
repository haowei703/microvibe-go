package errors

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ========================================
// 数据库错误判断方法
// ========================================

// IsNotFound 判断是否为记录不存在错误
// 统一封装 GORM 的 ErrRecordNotFound 错误判断
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsDuplicateKey 判断是否为重复键错误
// 支持 PostgreSQL 和 MySQL 的重复键错误
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: unique_violation (23505)
	if strings.Contains(errMsg, "23505") ||
		strings.Contains(errMsg, "duplicate key value") ||
		strings.Contains(errMsg, "violates unique constraint") {
		return true
	}

	// MySQL: Duplicate entry
	if strings.Contains(errMsg, "Duplicate entry") ||
		strings.Contains(errMsg, "Error 1062") {
		return true
	}

	return false
}

// IsForeignKeyViolation 判断是否为外键约束错误
// 支持 PostgreSQL 和 MySQL 的外键约束错误
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: foreign_key_violation (23503)
	if strings.Contains(errMsg, "23503") ||
		strings.Contains(errMsg, "violates foreign key constraint") {
		return true
	}

	// MySQL: Cannot add or update a child row
	if strings.Contains(errMsg, "Cannot add or update a child row") ||
		strings.Contains(errMsg, "Error 1452") {
		return true
	}

	return false
}

// IsConstraintViolation 判断是否为约束违反错误
// 包括唯一约束、外键约束、检查约束等
func IsConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	return IsDuplicateKey(err) || IsForeignKeyViolation(err) || IsCheckViolation(err)
}

// IsCheckViolation 判断是否为检查约束错误
func IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: check_violation (23514)
	if strings.Contains(errMsg, "23514") ||
		strings.Contains(errMsg, "violates check constraint") {
		return true
	}

	// MySQL: Check constraint
	if strings.Contains(errMsg, "Check constraint") {
		return true
	}

	return false
}

// IsConnectionError 判断是否为数据库连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	connectionKeywords := []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"no connection",
		"dial tcp",
		"timeout",
		"too many connections",
		"server closed",
	}

	for _, keyword := range connectionKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// IsDeadlock 判断是否为死锁错误
func IsDeadlock(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: deadlock_detected (40P01)
	if strings.Contains(errMsg, "40P01") ||
		strings.Contains(errMsg, "deadlock detected") {
		return true
	}

	// MySQL: Deadlock found
	if strings.Contains(errMsg, "Deadlock found") ||
		strings.Contains(errMsg, "Error 1213") {
		return true
	}

	return false
}

// IsSyntaxError 判断是否为SQL语法错误
func IsSyntaxError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// PostgreSQL: syntax_error (42601)
	if strings.Contains(errMsg, "42601") ||
		strings.Contains(errMsg, "syntax error") {
		return true
	}

	// MySQL: You have an error in your SQL syntax
	if strings.Contains(errMsg, "error in your SQL syntax") ||
		strings.Contains(errMsg, "Error 1064") {
		return true
	}

	return false
}

// ========================================
// 数据库错误转换方法
// ========================================

// ConvertDBError 将数据库错误转换为应用错误
// 统一处理数据库错误，返回友好的错误信息
func ConvertDBError(err error) error {
	if err == nil {
		return nil
	}

	// 记录不存在
	if IsNotFound(err) {
		return ErrRecordNotFound
	}

	// 重复键
	if IsDuplicateKey(err) {
		return ErrDuplicateKey
	}

	// 外键约束
	if IsForeignKeyViolation(err) {
		return NewAppErrorWithCause(CodeConstraintViolation, "数据关联约束违反", err)
	}

	// 检查约束
	if IsCheckViolation(err) {
		return NewAppErrorWithCause(CodeConstraintViolation, "数据检查约束违反", err)
	}

	// 连接错误
	if IsConnectionError(err) {
		return ErrConnectionFailed
	}

	// 死锁
	if IsDeadlock(err) {
		return NewAppErrorWithCause(CodeDBError, "数据库死锁", err)
	}

	// SQL语法错误
	if IsSyntaxError(err) {
		return NewAppErrorWithCause(CodeDBError, "SQL语法错误", err)
	}

	// 其他数据库错误
	return NewAppErrorWithCause(CodeDBError, "数据库操作失败", err)
}

// ConvertNotFoundError 转换记录不存在错误为特定的业务错误
// 根据资源类型返回更友好的错误信息
func ConvertNotFoundError(err error, resourceType string) error {
	if err == nil {
		return nil
	}

	if !IsNotFound(err) {
		return err
	}

	switch resourceType {
	case "user":
		return ErrUserNotFound
	case "video":
		return ErrVideoNotFound
	case "comment":
		return ErrCommentNotFound
	default:
		return ErrRecordNotFound
	}
}

// ========================================
// 辅助函数
// ========================================

// HandleDBError 处理数据库错误的辅助函数
// 自动转换数据库错误，并记录日志
func HandleDBError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// 转换为应用错误
	appErr := ConvertDBError(err)

	// 这里可以添加日志记录
	// logger.Error("数据库操作失败", zap.String("operation", operation), zap.Error(err))

	return appErr
}

// MustNotExist 检查记录必须不存在
// 如果记录存在则返回重复错误，如果查询失败则返回查询错误
func MustNotExist(err error) error {
	if err == nil {
		// 记录存在
		return ErrDuplicateKey
	}

	if IsNotFound(err) {
		// 记录不存在，符合预期
		return nil
	}

	// 查询出错
	return ConvertDBError(err)
}

// MustExist 检查记录必须存在
// 如果记录不存在则返回NotFound错误，如果查询失败则返回查询错误
func MustExist(err error, resourceType string) error {
	if err == nil {
		// 记录存在，符合预期
		return nil
	}

	if IsNotFound(err) {
		// 记录不存在
		return ConvertNotFoundError(err, resourceType)
	}

	// 查询出错
	return ConvertDBError(err)
}
