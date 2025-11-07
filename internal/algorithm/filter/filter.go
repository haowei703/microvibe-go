package filter

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// VideoFilter 视频过滤器
type VideoFilter struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewVideoFilter 创建视频过滤器实例
func NewVideoFilter(db *gorm.DB, redis *redis.Client) *VideoFilter {
	return &VideoFilter{
		db:    db,
		redis: redis,
	}
}

// FilterRequest 过滤请求
type FilterRequest struct {
	UserID uint
	Videos []*model.Video
}

// Filter 过滤视频
func (f *VideoFilter) Filter(ctx context.Context, req *FilterRequest) ([]*model.Video, error) {
	var result []*model.Video

	// 获取用户已观看的视频ID（用于去重）
	watchedSet, err := f.getWatchedVideos(ctx, req.UserID)
	if err != nil {
		watchedSet = make(map[uint]bool)
	}

	// 用于检测相似视频的集合
	seenCategories := make(map[uint]int)

	for _, video := range req.Videos {
		// 1. 过滤已观看的视频
		if watchedSet[video.ID] {
			continue
		}

		// 2. 过滤低质量视频
		if !f.passQualityFilter(video) {
			continue
		}

		// 3. 过滤相似视频（同一分类不要太多）
		if seenCategories[video.CategoryID] >= 2 {
			continue
		}

		// 4. 过滤用户屏蔽的作者
		if f.isBlockedAuthor(ctx, req.UserID, video.UserID) {
			continue
		}

		// 通过所有过滤条件
		result = append(result, video)
		seenCategories[video.CategoryID]++

		// 记录推荐（用于后续去重）
		f.recordRecommendation(ctx, req.UserID, video.ID)
	}

	return result, nil
}

// getWatchedVideos 获取用户已观看的视频ID集合
func (f *VideoFilter) getWatchedVideos(ctx context.Context, userID uint) (map[uint]bool, error) {
	// 从 Redis 获取最近观看的视频ID
	cacheKey := fmt.Sprintf("user:watched:%d", userID)
	videoIDs, err := f.redis.SMembers(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	watchedSet := make(map[uint]bool)
	for _, idStr := range videoIDs {
		var id uint
		fmt.Sscanf(idStr, "%d", &id)
		watchedSet[id] = true
	}

	// 如果 Redis 中没有数据，从数据库加载
	if len(watchedSet) == 0 {
		var behaviors []model.UserBehavior
		sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
		if err := f.db.Where("user_id = ? AND created_at > ? AND action = ?", userID, sevenDaysAgo, 1).
			Select("DISTINCT video_id").
			Find(&behaviors).Error; err == nil {

			pipe := f.redis.Pipeline()
			for _, b := range behaviors {
				watchedSet[b.VideoID] = true
				pipe.SAdd(ctx, cacheKey, fmt.Sprintf("%d", b.VideoID))
			}
			pipe.Expire(ctx, cacheKey, 7*24*time.Hour)
			pipe.Exec(ctx)
		}
	}

	return watchedSet, nil
}

// passQualityFilter 质量过滤
func (f *VideoFilter) passQualityFilter(video *model.Video) bool {
	// 质量分数太低
	if video.QualityScore < 30 {
		return false
	}

	// 视频状态不是已发布
	if video.Status != 1 {
		return false
	}

	// 视频时长异常（太短或太长）
	if video.Duration < 3 || video.Duration > 3600 {
		return false
	}

	return true
}

// isBlockedAuthor 检查是否是被屏蔽的作者
func (f *VideoFilter) isBlockedAuthor(ctx context.Context, userID, authorID uint) bool {
	// 从 Redis 检查黑名单
	cacheKey := fmt.Sprintf("user:blocked:%d", userID)
	isBlocked, err := f.redis.SIsMember(ctx, cacheKey, fmt.Sprintf("%d", authorID)).Result()
	if err == nil && isBlocked {
		return true
	}

	// 这里可以从数据库查询用户黑名单表（需要添加相应的模型）
	// 简化处理，暂时返回 false
	return false
}

// recordRecommendation 记录推荐（用于去重和效果分析）
func (f *VideoFilter) recordRecommendation(ctx context.Context, userID, videoID uint) {
	// 记录到 Redis，用于后续去重
	cacheKey := fmt.Sprintf("user:recommended:%d", userID)
	f.redis.SAdd(ctx, cacheKey, fmt.Sprintf("%d", videoID))
	f.redis.Expire(ctx, cacheKey, 24*time.Hour)
}

// FilterDuplicates 去除重复视频
func (f *VideoFilter) FilterDuplicates(videos []*model.Video) []*model.Video {
	seen := make(map[uint]bool)
	var result []*model.Video

	for _, video := range videos {
		if !seen[video.ID] {
			seen[video.ID] = true
			result = append(result, video)
		}
	}

	return result
}

// FilterByUserPreference 根据用户偏好过滤
func (f *VideoFilter) FilterByUserPreference(ctx context.Context, userID uint, videos []*model.Video) ([]*model.Video, error) {
	// 可以根据用户的偏好设置进行过滤
	// 例如：不看某些分类、不看某些标签等
	// 这里简化处理，直接返回原列表
	return videos, nil
}
