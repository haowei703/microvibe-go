package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"microvibe-go/internal/model"
	pkgerrors "microvibe-go/pkg/errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Engineer 特征工程
type Engineer struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewEngineer 创建特征工程实例
func NewEngineer(db *gorm.DB, redis *redis.Client) *Engineer {
	return &Engineer{
		db:    db,
		redis: redis,
	}
}

// UserFeature 用户特征
type UserFeature struct {
	UserID uint `json:"user_id"`

	// 基础特征
	Age      int    `json:"age"`      // 年龄
	Gender   int8   `json:"gender"`   // 性别
	Province string `json:"province"` // 省份
	City     string `json:"city"`     // 城市

	// 行为特征
	ActiveDays    int     `json:"active_days"`     // 活跃天数
	AvgWatchTime  float64 `json:"avg_watch_time"`  // 平均观看时长
	AvgFinishRate float64 `json:"avg_finish_rate"` // 平均完播率
	LikeRate      float64 `json:"like_rate"`       // 点赞率
	CommentRate   float64 `json:"comment_rate"`    // 评论率
	ShareRate     float64 `json:"share_rate"`      // 分享率

	// 兴趣标签（分类ID和分数）
	InterestTags map[uint]float64 `json:"interest_tags"`

	// 时间偏好（哪个时段活跃）
	ActiveHours []int `json:"active_hours"`

	UpdatedAt time.Time `json:"updated_at"`
}

// VideoFeature 视频特征
type VideoFeature struct {
	VideoID uint `json:"video_id"`

	// 内容特征
	CategoryID uint     `json:"category_id"` // 分类
	Duration   int      `json:"duration"`    // 时长
	Tags       []string `json:"tags"`        // 标签

	// 质量特征
	QualityScore float64 `json:"quality_score"`  // 质量分
	FinishRate   float64 `json:"finish_rate"`    // 完播率
	AvgWatchTime float64 `json:"avg_watch_time"` // 平均观看时长

	// 互动特征
	CTR         float64 `json:"ctr"`          // 点击率
	LikeRate    float64 `json:"like_rate"`    // 点赞率
	CommentRate float64 `json:"comment_rate"` // 评论率
	ShareRate   float64 `json:"share_rate"`   // 分享率

	// 热度特征
	HotScore   float64 `json:"hot_score"`   // 热度分数
	TrendScore float64 `json:"trend_score"` // 趋势分数（上升/下降）

	// 新鲜度
	PublishedAt    time.Time `json:"published_at"`
	FreshnessScore float64   `json:"freshness_score"` // 新鲜度分数

	UpdatedAt time.Time `json:"updated_at"`
}

// Extract 提取特征
func (e *Engineer) Extract(ctx context.Context, userID uint, videos []*model.Video) (map[string]interface{}, error) {
	features := make(map[string]interface{})

	// 提取用户特征
	userFeature, err := e.GetUserFeature(ctx, userID)
	if err == nil {
		features["user"] = userFeature
	}

	// 提取视频特征
	videoFeatures := make(map[uint]*VideoFeature)
	for _, video := range videos {
		videoFeature, err := e.GetVideoFeature(ctx, video.ID)
		if err == nil {
			videoFeatures[video.ID] = videoFeature
		}
	}
	features["videos"] = videoFeatures

	return features, nil
}

// GetUserFeature 获取用户特征
func (e *Engineer) GetUserFeature(ctx context.Context, userID uint) (*UserFeature, error) {
	// 先从 Redis 缓存获取
	cacheKey := fmt.Sprintf("user:feature:%d", userID)
	cached, err := e.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var feature UserFeature
		if err := json.Unmarshal([]byte(cached), &feature); err == nil {
			return &feature, nil
		}
	}

	// 缓存未命中，计算特征
	feature, err := e.calculateUserFeature(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	data, _ := json.Marshal(feature)
	e.redis.Set(ctx, cacheKey, data, 1*time.Hour)

	return feature, nil
}

// calculateUserFeature 计算用户特征
func (e *Engineer) calculateUserFeature(ctx context.Context, userID uint) (*UserFeature, error) {
	feature := &UserFeature{
		UserID:       userID,
		InterestTags: make(map[uint]float64),
		ActiveHours:  make([]int, 0),
		UpdatedAt:    time.Now(),
	}

	// 获取用户基本信息
	var user model.User
	if err := e.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	feature.Gender = user.Gender
	feature.Province = user.Province
	feature.City = user.City

	// 计算年龄
	if user.Birthday != nil {
		feature.Age = time.Now().Year() - user.Birthday.Year()
	}

	// 获取用户行为统计
	var behaviors []model.UserBehavior
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	if err := e.db.Where("user_id = ? AND created_at > ?", userID, thirtyDaysAgo).
		Find(&behaviors).Error; err == nil {

		if len(behaviors) > 0 {
			// 计算活跃天数
			dayMap := make(map[string]bool)
			totalWatchTime := 0
			totalProgress := 0
			likeCount := 0
			commentCount := 0
			shareCount := 0
			hourMap := make(map[int]int)

			for _, b := range behaviors {
				// 活跃天数
				day := b.CreatedAt.Format("2006-01-02")
				dayMap[day] = true

				// 观看时长和完播率
				if b.Action == 1 { // 浏览
					totalWatchTime += b.Duration
					totalProgress += b.Progress
				}

				// 互动率
				if b.Action == 2 {
					likeCount++
				} else if b.Action == 3 {
					commentCount++
				} else if b.Action == 4 {
					shareCount++
				}

				// 活跃时段
				hour := b.CreatedAt.Hour()
				hourMap[hour]++
			}

			feature.ActiveDays = len(dayMap)
			if len(behaviors) > 0 {
				feature.AvgWatchTime = float64(totalWatchTime) / float64(len(behaviors))
				feature.AvgFinishRate = float64(totalProgress) / float64(len(behaviors))
				feature.LikeRate = float64(likeCount) / float64(len(behaviors))
				feature.CommentRate = float64(commentCount) / float64(len(behaviors))
				feature.ShareRate = float64(shareCount) / float64(len(behaviors))
			}

			// 找出最活跃的3个时段
			type hourCount struct {
				hour  int
				count int
			}
			var hours []hourCount
			for h, c := range hourMap {
				hours = append(hours, hourCount{h, c})
			}
			// 简单排序（这里可以优化）
			for i := 0; i < len(hours) && i < 3; i++ {
				feature.ActiveHours = append(feature.ActiveHours, hours[i].hour)
			}
		}
	}

	// 获取用户兴趣标签
	var interests []model.UserInterest
	if err := e.db.Where("user_id = ?", userID).
		Order("score DESC").
		Limit(10).
		Find(&interests).Error; err == nil {
		for _, interest := range interests {
			feature.InterestTags[interest.CategoryID] = interest.Score
		}
	}

	return feature, nil
}

// GetVideoFeature 获取视频特征
func (e *Engineer) GetVideoFeature(ctx context.Context, videoID uint) (*VideoFeature, error) {
	// 先从 Redis 缓存获取
	cacheKey := fmt.Sprintf("video:feature:%d", videoID)
	cached, err := e.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var feature VideoFeature
		if err := json.Unmarshal([]byte(cached), &feature); err == nil {
			return &feature, nil
		}
	}

	// 缓存未命中，计算特征
	feature, err := e.calculateVideoFeature(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	data, _ := json.Marshal(feature)
	e.redis.Set(ctx, cacheKey, data, 30*time.Minute)

	return feature, nil
}

// calculateVideoFeature 计算视频特征
func (e *Engineer) calculateVideoFeature(ctx context.Context, videoID uint) (*VideoFeature, error) {
	var video model.Video
	if err := e.db.First(&video, videoID).Error; err != nil {
		return nil, err
	}

	feature := &VideoFeature{
		VideoID:      videoID,
		CategoryID:   video.CategoryID,
		Duration:     video.Duration,
		QualityScore: video.QualityScore,
		HotScore:     video.HotScore,
		PublishedAt:  *video.PublishedAt,
		UpdatedAt:    time.Now(),
	}

	// 计算新鲜度分数（越新分数越高）
	hoursSincePublish := time.Since(*video.PublishedAt).Hours()
	feature.FreshnessScore = 1.0 / (1.0 + hoursSincePublish/24.0) // 24小时衰减

	// 计算互动率
	if video.PlayCount > 0 {
		feature.LikeRate = float64(video.LikeCount) / float64(video.PlayCount)
		feature.CommentRate = float64(video.CommentCount) / float64(video.PlayCount)
		feature.ShareRate = float64(video.ShareCount) / float64(video.PlayCount)
	}

	// 从统计表获取更详细的数据
	var stats model.VideoStats
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	if err := e.db.Where("video_id = ? AND date = ?", videoID, yesterday).
		First(&stats).Error; err == nil {
		feature.FinishRate = stats.FinishRate
		feature.AvgWatchTime = stats.AvgDuration
		if stats.UniqueViewCount > 0 {
			feature.CTR = float64(stats.PlayCount) / float64(stats.UniqueViewCount)
		}
	}

	return feature, nil
}

// UpdateUserProfile 更新用户画像
func (e *Engineer) UpdateUserProfile(ctx context.Context, userID uint, behavior *model.UserBehavior) error {
	// 保存行为记录
	if err := e.db.Create(behavior).Error; err != nil {
		return err
	}

	// 异步更新用户兴趣标签
	go e.updateUserInterest(context.Background(), userID, behavior)

	// 清除用户特征缓存
	cacheKey := fmt.Sprintf("user:feature:%d", userID)
	e.redis.Del(ctx, cacheKey)

	return nil
}

// updateUserInterest 更新用户兴趣
func (e *Engineer) updateUserInterest(ctx context.Context, userID uint, behavior *model.UserBehavior) {
	// 获取视频信息
	var video model.Video
	if err := e.db.First(&video, behavior.VideoID).Error; err != nil {
		return
	}

	// 计算兴趣分数增量
	var scoreIncrement float64
	switch behavior.Action {
	case 1: // 浏览
		scoreIncrement = 0.1 * (float64(behavior.Progress) / 100.0)
	case 2: // 点赞
		scoreIncrement = 0.3
	case 3: // 评论
		scoreIncrement = 0.4
	case 4: // 分享
		scoreIncrement = 0.5
	case 5: // 收藏
		scoreIncrement = 0.6
	case 6: // 完播
		scoreIncrement = 0.8
	}

	// 更新用户兴趣表
	var interest model.UserInterest
	err := e.db.Where("user_id = ? AND category_id = ?", userID, video.CategoryID).
		First(&interest).Error

	if pkgerrors.IsNotFound(err) {
		// 创建新的兴趣记录
		interest = model.UserInterest{
			UserID:     userID,
			CategoryID: video.CategoryID,
			Score:      scoreIncrement,
			Weight:     1.0,
			ViewCount:  1,
		}
		if behavior.Action == 2 {
			interest.LikeCount = 1
		}
		e.db.Create(&interest)
	} else if err == nil {
		// 更新现有兴趣记录
		interest.Score = interest.Score*0.9 + scoreIncrement // 指数衰减
		if interest.Score > 1.0 {
			interest.Score = 1.0
		}
		interest.ViewCount++
		if behavior.Action == 2 {
			interest.LikeCount++
		}
		e.db.Save(&interest)
	}
}

// UpdateVideoFeature 更新视频特征
func (e *Engineer) UpdateVideoFeature(ctx context.Context, videoID uint) error {
	// 清除视频特征缓存
	cacheKey := fmt.Sprintf("video:feature:%d", videoID)
	return e.redis.Del(ctx, cacheKey).Err()
}
