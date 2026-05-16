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
	Username string  `gorm:"uniqueIndex;size:50;not null" json:"username"` // 用户名
	Password string  `gorm:"size:255;not null" json:"-"`                   // 密码（加密后，禁止序列化）
	Email    string  `gorm:"uniqueIndex;size:100" json:"email"`            // 邮箱
	Phone    *string `gorm:"uniqueIndex;size:20" json:"phone"`             // 手机号

	Nickname        string     `gorm:"size:50" json:"nickname"`          // 昵称
	Avatar          string     `gorm:"size:255" json:"avatar"`           // 头像URL
	BackgroundImage string     `gorm:"size:255" json:"background_image"` // 主页背景图URL
	Gender          int8       `gorm:"default:0" json:"gender"`          // 性别：0-未知，1-男，2-女
	Birthday        *time.Time `json:"birthday"`                         // 生日
	Bio             string     `gorm:"size:255" json:"bio"`              // 个人简介

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
	Role        int8       `gorm:"default:0" json:"role"`            // 角色：0-普通用户，1-管理员
	IsVerified  bool       `gorm:"default:false" json:"is_verified"` // 是否认证
	LastLoginAt *time.Time `json:"last_login_at"`                    // 最后登录时间

	// 隐私设置
	ShowFavorites bool `gorm:"default:false" json:"show_favorites"` // 是否公开收藏列表
	ShowLikes     bool `gorm:"default:false" json:"show_likes"`     // 是否公开点赞列表
	ShowFollowing bool `gorm:"default:true" json:"show_following"`  // 是否公开关注列表
	ShowFollowers bool `gorm:"default:true" json:"show_followers"`  // 是否公开粉丝列表
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// ToAuthorVO 转换为 AuthorVO
func (u *User) ToAuthorVO() *AuthorVO {
	if u == nil {
		return nil
	}
	return &AuthorVO{
		ID:              u.ID,
		Username:        u.Username,
		Nickname:        u.Nickname,
		Avatar:          u.Avatar,
		BackgroundImage: u.BackgroundImage,
		IsFollowed:      false, // 默认不关注，如需真实状态由业务层设置
	}
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

// UserVO 用户视图对象（包含计算字段）
type UserVO struct {
	*User
	Password   string `json:"-"`           // 遮蔽 User 的 Password，确保不被返回
	IsFollowed bool   `json:"is_followed"` // 当前用户是否已关注该用户
}

// ToVO 转换为 UserVO
func (u *User) ToVO(isFollowed bool) *UserVO {
	if u == nil {
		return nil
	}
	return &UserVO{
		User:       u,
		IsFollowed: isFollowed,
	}
}

// AuthorVO 精简的用户信息（用于消息列表等场景）
type AuthorVO struct {
	ID              uint   `json:"id"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	Avatar          string `json:"avatar"`
	BackgroundImage string `json:"background_image"`
	IsFollowed      bool   `json:"is_followed"`
}

// UserFollowVO 用户关注/粉丝列表视图对象（精简字段）
type UserFollowVO struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar"`
	Introduction string `json:"introduction"` // 用户简介（来自 user_profiles 表）
	IsFollowed   bool   `json:"is_followed"`  // 当前用户是否已关注该用户
}
