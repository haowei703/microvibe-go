package model

import (
	"time"
)

// UserBehavior 用户行为模型（用于推荐算法）
type UserBehavior struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID   uint `gorm:"index;not null" json:"user_id"`  // 用户ID
	VideoID  uint `gorm:"index;not null" json:"video_id"` // 视频ID
	Action   int8 `gorm:"index;not null" json:"action"`   // 行为类型：1-浏览，2-点赞，3-评论，4-分享，5-收藏，6-完播
	Duration int  `gorm:"default:0" json:"duration"`      // 观看时长（秒）
	Progress int  `gorm:"default:0" json:"progress"`      // 观看进度（百分比）

	// 上下文信息
	Source   string `gorm:"size:50" json:"source"`   // 来源：推荐、搜索、关注等
	Platform string `gorm:"size:20" json:"platform"` // 平台：ios、android、web
	IP       string `gorm:"size:45" json:"ip"`       // IP地址
}

// TableName 指定表名
func (UserBehavior) TableName() string {
	return "user_behaviors"
}

// UserInterest 用户兴趣标签
type UserInterest struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID     uint    `gorm:"index;not null" json:"user_id"`     // 用户ID
	CategoryID uint    `gorm:"index;not null" json:"category_id"` // 分类ID
	TagID      *uint   `gorm:"index" json:"tag_id"`               // 标签ID
	Score      float64 `gorm:"not null" json:"score"`             // 兴趣分数（0-1）
	Weight     float64 `gorm:"default:1.0" json:"weight"`         // 权重

	// 统计
	ViewCount int64 `gorm:"default:0" json:"view_count"` // 浏览次数
	LikeCount int64 `gorm:"default:0" json:"like_count"` // 点赞次数
}

// TableName 指定表名
func (UserInterest) TableName() string {
	return "user_interests"
}

// VideoStats 视频统计数据（定时汇总）
type VideoStats struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	VideoID uint      `gorm:"uniqueIndex:idx_video_date;not null" json:"video_id"`       // 视频ID
	Date    time.Time `gorm:"uniqueIndex:idx_video_date;type:date;not null" json:"date"` // 统计日期

	// 播放相关
	PlayCount       int64   `gorm:"default:0" json:"play_count"`        // 播放量
	UniqueViewCount int64   `gorm:"default:0" json:"unique_view_count"` // 独立访客数
	AvgDuration     float64 `gorm:"default:0" json:"avg_duration"`      // 平均观看时长
	FinishRate      float64 `gorm:"default:0" json:"finish_rate"`       // 完播率

	// 互动相关
	LikeCount     int64 `gorm:"default:0" json:"like_count"`     // 点赞数
	CommentCount  int64 `gorm:"default:0" json:"comment_count"`  // 评论数
	ShareCount    int64 `gorm:"default:0" json:"share_count"`    // 分享数
	FavoriteCount int64 `gorm:"default:0" json:"favorite_count"` // 收藏数

	// 流量来源
	RecommendCount int64 `gorm:"default:0" json:"recommend_count"` // 推荐流量
	SearchCount    int64 `gorm:"default:0" json:"search_count"`    // 搜索流量
	FollowCount    int64 `gorm:"default:0" json:"follow_count"`    // 关注流量
	ShareTraffic   int64 `gorm:"default:0" json:"share_traffic"`   // 分享流量
}

// TableName 指定表名
func (VideoStats) TableName() string {
	return "video_stats"
}

// SearchHistory 搜索历史
type SearchHistory struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID      uint   `gorm:"index;not null" json:"user_id"`          // 用户ID
	Keyword     string `gorm:"size:200;index;not null" json:"keyword"` // 搜索关键词
	Category    string `gorm:"size:50" json:"category"`                // 搜索类型：video、user、hashtag
	ResultCount int    `gorm:"default:0" json:"result_count"`          // 结果数量
}

// TableName 指定表名
func (SearchHistory) TableName() string {
	return "search_histories"
}

// HotSearch 热搜
type HotSearch struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Keyword     string  `gorm:"size:200;uniqueIndex;not null" json:"keyword"` // 关键词
	SearchCount int64   `gorm:"default:0;index" json:"search_count"`          // 搜索次数
	HotScore    float64 `gorm:"default:0;index" json:"hot_score"`             // 热度分数
	Rank        int     `gorm:"default:0;index" json:"rank"`                  // 排名
	IsSticky    bool    `gorm:"default:false" json:"is_sticky"`               // 是否置顶
}

// TableName 指定表名
func (HotSearch) TableName() string {
	return "hot_searches"
}
