package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 评论模型
type Comment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	VideoID       uint   `gorm:"index;not null" json:"video_id"`    // 视频ID
	UserID        uint   `gorm:"index;not null" json:"user_id"`     // 评论用户ID
	Content       string `gorm:"type:text;not null" json:"content"` // 评论内容
	ParentID      *uint  `gorm:"index" json:"parent_id"`            // 父评论ID（用于回复）
	RootID        *uint  `gorm:"index" json:"root_id"`              // 根评论ID（一级评论）
	ReplyToUserID *uint  `gorm:"index" json:"reply_to_user_id"`     // 回复的用户ID

	// 统计
	LikeCount  int64 `gorm:"default:0" json:"like_count"`  // 点赞数
	ReplyCount int64 `gorm:"default:0" json:"reply_count"` // 回复数

	// 状态
	Status int8 `gorm:"default:1;index" json:"status"` // 状态：0-已删除，1-正常，2-审核中，3-违规

	// 关联
	User        *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Video       *Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
	ReplyToUser *User  `gorm:"foreignKey:ReplyToUserID" json:"reply_to_user,omitempty"`
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// Like 点赞模型
type Like struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID  uint `gorm:"index;not null" json:"user_id"`  // 用户ID
	VideoID uint `gorm:"index;not null" json:"video_id"` // 视频ID
	Type    int8 `gorm:"default:1" json:"type"`          // 类型：1-点赞，2-踩

	// 组合唯一索引
	// gorm:"uniqueIndex:idx_user_video"
}

// TableName 指定表名
func (Like) TableName() string {
	return "likes"
}

// CommentLike 评论点赞模型
type CommentLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID    uint `gorm:"index;not null" json:"user_id"`    // 用户ID
	CommentID uint `gorm:"index;not null" json:"comment_id"` // 评论ID

	// 关联
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Comment *Comment `gorm:"foreignKey:CommentID" json:"comment,omitempty"`
}

// TableName 指定表名
func (CommentLike) TableName() string {
	return "comment_likes"
}

// Favorite 收藏模型
type Favorite struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID   uint  `gorm:"index;not null" json:"user_id"`  // 用户ID
	VideoID  uint  `gorm:"index;not null" json:"video_id"` // 视频ID
	FolderID *uint `gorm:"index" json:"folder_id"`         // 收藏夹ID

	// 关联
	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Video *Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

// TableName 指定表名
func (Favorite) TableName() string {
	return "favorites"
}

// FavoriteFolder 收藏夹
type FavoriteFolder struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID      uint   `gorm:"index;not null" json:"user_id"` // 用户ID
	Name        string `gorm:"size:100;not null" json:"name"` // 收藏夹名称
	Description string `gorm:"size:255" json:"description"`   // 描述
	CoverURL    string `gorm:"size:255" json:"cover_url"`     // 封面
	IsPublic    bool   `gorm:"default:true" json:"is_public"` // 是否公开
	VideoCount  int64  `gorm:"default:0" json:"video_count"`  // 视频数量
}

// TableName 指定表名
func (FavoriteFolder) TableName() string {
	return "favorite_folders"
}

// Follow 关注关系模型
type Follow struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID     uint `gorm:"index;not null" json:"user_id"`     // 关注者ID
	FollowedID uint `gorm:"index;not null" json:"followed_id"` // 被关注者ID

	// 关联
	User     *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Followed *User `gorm:"foreignKey:FollowedID" json:"followed,omitempty"`
}

// TableName 指定表名
func (Follow) TableName() string {
	return "follows"
}

// Share 分享记录
type Share struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID   uint   `gorm:"index;not null" json:"user_id"`  // 分享用户ID
	VideoID  uint   `gorm:"index;not null" json:"video_id"` // 视频ID
	Platform string `gorm:"size:50" json:"platform"`        // 分享平台
}

// TableName 指定表名
func (Share) TableName() string {
	return "shares"
}
