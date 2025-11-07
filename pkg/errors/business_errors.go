package errors

// ========================================
// 业务错误构造函数
// ========================================

// NewInvalidParamError 创建参数错误
func NewInvalidParamError(param string) *AppError {
	return &AppError{
		Code:    CodeInvalidParam,
		Message: "参数错误: " + param,
	}
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(reason string) *AppError {
	msg := "未授权"
	if reason != "" {
		msg = msg + ": " + reason
	}
	return &AppError{
		Code:    CodeUnauthorized,
		Message: msg,
	}
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(reason string) *AppError {
	msg := "禁止访问"
	if reason != "" {
		msg = msg + ": " + reason
	}
	return &AppError{
		Code:    CodeForbidden,
		Message: msg,
	}
}

// NewUserNotFoundError 创建用户不存在错误
func NewUserNotFoundError(identifier string) *AppError {
	msg := "用户不存在"
	if identifier != "" {
		msg = msg + ": " + identifier
	}
	return &AppError{
		Code:    CodeUserNotFound,
		Message: msg,
	}
}

// NewVideoNotFoundError 创建视频不存在错误
func NewVideoNotFoundError(videoID string) *AppError {
	msg := "视频不存在"
	if videoID != "" {
		msg = msg + ": " + videoID
	}
	return &AppError{
		Code:    CodeVideoNotFound,
		Message: msg,
	}
}

// NewCommentNotFoundError 创建评论不存在错误
func NewCommentNotFoundError(commentID string) *AppError {
	msg := "评论不存在"
	if commentID != "" {
		msg = msg + ": " + commentID
	}
	return &AppError{
		Code:    CodeCommentNotFound,
		Message: msg,
	}
}

// NewPermissionDeniedError 创建权限不足错误
func NewPermissionDeniedError(action string) *AppError {
	msg := "权限不足"
	if action != "" {
		msg = msg + ": " + action
	}
	return &AppError{
		Code:    CodePermissionDenied,
		Message: msg,
	}
}

// ========================================
// 业务错误判断方法
// ========================================

// IsUserNotFound 判断是否为用户不存在错误
func IsUserNotFound(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeUserNotFound
}

// IsVideoNotFound 判断是否为视频不存在错误
func IsVideoNotFound(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeVideoNotFound
}

// IsCommentNotFound 判断是否为评论不存在错误
func IsCommentNotFound(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeCommentNotFound
}

// IsUnauthorized 判断是否为未授权错误
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeUnauthorized
}

// IsForbidden 判断是否为禁止访问错误
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeForbidden
}

// IsPermissionDenied 判断是否为权限不足错误
func IsPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodePermissionDenied
}

// IsInvalidParam 判断是否为参数错误
func IsInvalidParam(err error) bool {
	if err == nil {
		return false
	}
	return GetCode(err) == CodeInvalidParam
}

// ========================================
// 业务错误辅助函数
// ========================================

// RequireAuth 要求用户已认证
// 如果用户ID为空，则返回未授权错误
func RequireAuth(userID uint) error {
	if userID == 0 {
		return NewUnauthorizedError("用户未登录")
	}
	return nil
}

// RequirePermission 要求用户有特定权限
// 如果没有权限，则返回权限不足错误
func RequirePermission(hasPermission bool, action string) error {
	if !hasPermission {
		return NewPermissionDeniedError(action)
	}
	return nil
}

// ValidateParam 验证参数
// 如果验证失败，则返回参数错误
func ValidateParam(valid bool, param string) error {
	if !valid {
		return NewInvalidParamError(param)
	}
	return nil
}
