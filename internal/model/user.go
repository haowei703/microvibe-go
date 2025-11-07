package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 基本信息
	Username  string     `gorm:"uniqueIndex;size:50;not null" json:"username"` // 用户名
	Password  string     `gorm:"size:255;not null" json:"-"`                   // 密码（加密后）
	Email     string     `gorm:"uniqueIndex;size:100" json:"email"`            // 邮箱
	Phone     string     `gorm:"uniqueIndex;size:20" json:"phone"`             // 手机号
	Nickname  string     `gorm:"size:50" json:"nickname"`                      // 昵称
	Avatar    string     `gorm:"size:255" json:"avatar"`                       // 头像URL
	Gender    int8       `gorm:"default:0" json:"gender"`                      // 性别：0-未知，1-男，2-女
	Birthday  *time.Time `json:"birthday"`                                     // 生日
	Signature string     `gorm:"size:255" json:"signature"`                    // 个性签名

	// 地理位置
	Province string `gorm:"size:50" json:"province"` // 省份
	City     string `gorm:"size:50" json:"city"`     // 城市

	// 统计信息
	FollowCount   int64 `gorm:"default:0" json:"follow_count"`   // 关注数
	FollowerCount int64 `gorm:"default:0" json:"follower_count"` // 粉丝数
	VideoCount    int64 `gorm:"default:0" json:"video_count"`    // 视频数
	LikeCount     int64 `gorm:"default:0" json:"like_count"`     // 获赞数
	FavoriteCount int64 `gorm:"default:0" json:"favorite_count"` // 收藏数

	// 状态
	Status      int8       `gorm:"default:1" json:"status"`          // 状态：0-禁用，1-正常
	IsVerified  bool       `gorm:"default:false" json:"is_verified"` // 是否认证
	LastLoginAt *time.Time `json:"last_login_at"`                    // 最后登录时间
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserProfile 用户扩展资料
type UserProfile struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID       uint   `gorm:"uniqueIndex;not null" json:"user_id"` // 用户ID
	School       string `gorm:"size:100" json:"school"`              // 学校
	Company      string `gorm:"size:100" json:"company"`             // 公司
	Profession   string `gorm:"size:50" json:"profession"`           // 职业
	Homepage     string `gorm:"size:255" json:"homepage"`            // 个人主页
	Introduction string `gorm:"type:text" json:"introduction"`       // 个人简介
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}
