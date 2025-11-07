package model

import (
	"time"

	"gorm.io/gorm"
)

// Video 视频模型
type Video struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 基本信息
	UserID      uint   `gorm:"index;not null" json:"user_id"`      // 作者ID
	Title       string `gorm:"size:200;not null" json:"title"`     // 标题
	Description string `gorm:"type:text" json:"description"`       // 描述
	CoverURL    string `gorm:"size:255;not null" json:"cover_url"` // 封面URL
	VideoURL    string `gorm:"size:255;not null" json:"video_url"` // 视频URL
	Duration    int    `gorm:"not null" json:"duration"`           // 视频时长（秒）
	Width       int    `gorm:"default:0" json:"width"`             // 视频宽度
	Height      int    `gorm:"default:0" json:"height"`            // 视频高度
	FileSize    int64  `gorm:"default:0" json:"file_size"`         // 文件大小（字节）

	// 分类和标签
	CategoryID uint   `gorm:"index" json:"category_id"` // 分类ID
	Tags       string `gorm:"size:500" json:"tags"`     // 标签（逗号分隔）

	// 统计信息
	PlayCount     int64 `gorm:"default:0;index" json:"play_count"` // 播放量
	LikeCount     int64 `gorm:"default:0;index" json:"like_count"` // 点赞数
	CommentCount  int64 `gorm:"default:0" json:"comment_count"`    // 评论数
	ShareCount    int64 `gorm:"default:0" json:"share_count"`      // 分享数
	FavoriteCount int64 `gorm:"default:0" json:"favorite_count"`   // 收藏数

	// 推荐算法相关
	HotScore     float64 `gorm:"default:0;index" json:"hot_score"` // 热度分数
	QualityScore float64 `gorm:"default:0" json:"quality_score"`   // 质量分数

	// 状态
	Status       int8       `gorm:"default:0;index" json:"status"`     // 状态：0-审核中，1-已发布，2-不通过，3-下架
	IsPublic     bool       `gorm:"default:true" json:"is_public"`     // 是否公开
	AllowComment bool       `gorm:"default:true" json:"allow_comment"` // 是否允许评论
	PublishedAt  *time.Time `gorm:"index" json:"published_at"`         // 发布时间

	// 关联
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

// TableName 指定表名
func (Video) TableName() string {
	return "videos"
}

// Category 视频分类
type Category struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name        string `gorm:"size:50;not null;uniqueIndex" json:"name"` // 分类名称
	Description string `gorm:"size:255" json:"description"`              // 分类描述
	Icon        string `gorm:"size:255" json:"icon"`                     // 分类图标
	Sort        int    `gorm:"default:0" json:"sort"`                    // 排序
	Status      int8   `gorm:"default:1" json:"status"`                  // 状态：0-禁用，1-启用
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}

// Hashtag 话题标签
type Hashtag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name       string  `gorm:"size:100;not null;uniqueIndex" json:"name"` // 话题名称
	ViewCount  int64   `gorm:"default:0;index" json:"view_count"`         // 浏览量
	VideoCount int64   `gorm:"default:0" json:"video_count"`              // 视频数
	HotScore   float64 `gorm:"default:0;index" json:"hot_score"`          // 热度分数
	IsHot      bool    `gorm:"default:false;index" json:"is_hot"`         // 是否热门
}

// TableName 指定表名
func (Hashtag) TableName() string {
	return "hashtags"
}

// VideoHashtag 视频话题关联表
type VideoHashtag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	VideoID   uint `gorm:"index;not null" json:"video_id"`   // 视频ID
	HashtagID uint `gorm:"index;not null" json:"hashtag_id"` // 话题ID
}

// TableName 指定表名
func (VideoHashtag) TableName() string {
	return "video_hashtags"
}
