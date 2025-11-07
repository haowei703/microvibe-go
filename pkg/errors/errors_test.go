package errors_test

import (
	"testing"

	"microvibe-go/pkg/errors"

	"gorm.io/gorm"
)

// ========================================
// 核心错误测试
// ========================================

func TestNewAppError(t *testing.T) {
	err := errors.NewAppError(errors.CodeInvalidParam, "参数错误")

	if err.Code != errors.CodeInvalidParam {
		t.Errorf("期望错误码 %d, 得到 %d", errors.CodeInvalidParam, err.Code)
	}

	if err.Message != "参数错误" {
		t.Errorf("期望错误信息 '参数错误', 得到 '%s'", err.Message)
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *errors.AppError
		contains string
	}{
		{
			name:     "无原始错误",
			err:      errors.NewAppError(1001, "测试错误"),
			contains: "[1001] 测试错误",
		},
		{
			name:     "带原始错误",
			err:      errors.NewAppErrorWithCause(1001, "测试错误", errors.New("原因")),
			contains: "原因",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg == "" {
				t.Error("错误信息不应为空")
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AppError",
			err:      errors.NewAppError(errors.CodeInvalidParam, "参数错误"),
			expected: errors.CodeInvalidParam,
		},
		{
			name:     "普通错误",
			err:      errors.New("普通错误"),
			expected: errors.CodeUnknown,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: errors.CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := errors.GetCode(tt.err)
			if code != tt.expected {
				t.Errorf("期望错误码 %d, 得到 %d", tt.expected, code)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := errors.Wrap(originalErr, "包装信息")

	if wrappedErr == nil {
		t.Fatal("包装后的错误不应为 nil")
	}

	// 应该能unwrap到原始错误
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("应该能unwrap到原始错误")
	}
}

// ========================================
// 数据库错误测试
// ========================================

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GORM记录不存在",
			err:      gorm.ErrRecordNotFound,
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("其他错误"),
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestIsDuplicateKey(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "PostgreSQL重复键",
			err:      errors.New("ERROR: duplicate key value violates unique constraint"),
			expected: true,
		},
		{
			name:     "MySQL重复键",
			err:      errors.New("Error 1062: Duplicate entry"),
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("其他错误"),
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsDuplicateKey(tt.err)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "PostgreSQL外键约束",
			err:      errors.New("ERROR: violates foreign key constraint"),
			expected: true,
		},
		{
			name:     "MySQL外键约束",
			err:      errors.New("Cannot add or update a child row"),
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("其他错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsForeignKeyViolation(tt.err)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "连接被拒绝",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "连接超时",
			err:      errors.New("dial tcp: timeout"),
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("其他错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestConvertDBError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "记录不存在",
			err:          gorm.ErrRecordNotFound,
			expectedCode: errors.CodeRecordNotFound,
		},
		{
			name:         "重复键",
			err:          errors.New("duplicate key value"),
			expectedCode: errors.CodeDuplicateKey,
		},
		{
			name:         "连接错误",
			err:          errors.New("connection refused"),
			expectedCode: errors.CodeConnectionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.ConvertDBError(tt.err)
			code := errors.GetCode(result)
			if code != tt.expectedCode {
				t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, code)
			}
		})
	}
}

func TestConvertNotFoundError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		resourceType string
		expectedCode int
	}{
		{
			name:         "用户不存在",
			err:          gorm.ErrRecordNotFound,
			resourceType: "user",
			expectedCode: errors.CodeUserNotFound,
		},
		{
			name:         "视频不存在",
			err:          gorm.ErrRecordNotFound,
			resourceType: "video",
			expectedCode: errors.CodeVideoNotFound,
		},
		{
			name:         "评论不存在",
			err:          gorm.ErrRecordNotFound,
			resourceType: "comment",
			expectedCode: errors.CodeCommentNotFound,
		},
		{
			name:         "默认记录不存在",
			err:          gorm.ErrRecordNotFound,
			resourceType: "unknown",
			expectedCode: errors.CodeRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.ConvertNotFoundError(tt.err, tt.resourceType)
			code := errors.GetCode(result)
			if code != tt.expectedCode {
				t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, code)
			}
		})
	}
}

func TestMustNotExist(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectError  bool
		expectedCode int
	}{
		{
			name:         "记录存在（应该返回重复错误）",
			err:          nil,
			expectError:  true,
			expectedCode: errors.CodeDuplicateKey,
		},
		{
			name:        "记录不存在（符合预期）",
			err:         gorm.ErrRecordNotFound,
			expectError: false,
		},
		{
			name:         "查询出错",
			err:          errors.New("database error"),
			expectError:  true,
			expectedCode: errors.CodeDBError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.MustNotExist(tt.err)
			if tt.expectError {
				if result == nil {
					t.Error("期望返回错误，但得到 nil")
					return
				}
				code := errors.GetCode(result)
				if code != tt.expectedCode {
					t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, code)
				}
			} else {
				if result != nil {
					t.Errorf("期望 nil, 得到错误: %v", result)
				}
			}
		})
	}
}

func TestMustExist(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		resourceType string
		expectError  bool
		expectedCode int
	}{
		{
			name:        "记录存在（符合预期）",
			err:         nil,
			expectError: false,
		},
		{
			name:         "记录不存在（应该返回NotFound）",
			err:          gorm.ErrRecordNotFound,
			resourceType: "user",
			expectError:  true,
			expectedCode: errors.CodeUserNotFound,
		},
		{
			name:         "查询出错",
			err:          errors.New("database error"),
			resourceType: "user",
			expectError:  true,
			expectedCode: errors.CodeDBError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.MustExist(tt.err, tt.resourceType)
			if tt.expectError {
				if result == nil {
					t.Error("期望返回错误，但得到 nil")
					return
				}
				code := errors.GetCode(result)
				if code != tt.expectedCode {
					t.Errorf("期望错误码 %d, 得到 %d", tt.expectedCode, code)
				}
			} else {
				if result != nil {
					t.Errorf("期望 nil, 得到错误: %v", result)
				}
			}
		})
	}
}

// ========================================
// 业务错误测试
// ========================================

func TestNewUserNotFoundError(t *testing.T) {
	err := errors.NewUserNotFoundError("alice")

	if errors.GetCode(err) != errors.CodeUserNotFound {
		t.Errorf("期望错误码 %d, 得到 %d", errors.CodeUserNotFound, errors.GetCode(err))
	}

	msg := err.Error()
	if msg == "" {
		t.Error("错误信息不应为空")
	}
}

func TestIsUserNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "用户不存在错误",
			err:      errors.NewUserNotFoundError("alice"),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      errors.ErrInvalidParam,
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.IsUserNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name        string
		userID      uint
		expectError bool
	}{
		{
			name:        "已认证",
			userID:      1,
			expectError: false,
		},
		{
			name:        "未认证",
			userID:      0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.RequireAuth(tt.userID)
			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但得到 nil")
				}
			} else {
				if err != nil {
					t.Errorf("期望 nil, 得到错误: %v", err)
				}
			}
		})
	}
}

func TestValidateParam(t *testing.T) {
	tests := []struct {
		name        string
		valid       bool
		param       string
		expectError bool
	}{
		{
			name:        "参数有效",
			valid:       true,
			param:       "username",
			expectError: false,
		},
		{
			name:        "参数无效",
			valid:       false,
			param:       "username",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.ValidateParam(tt.valid, tt.param)
			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但得到 nil")
				}
			} else {
				if err != nil {
					t.Errorf("期望 nil, 得到错误: %v", err)
				}
			}
		})
	}
}
