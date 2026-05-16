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
	Scene  string // 场景标识，"follow"/"friends" 时跳过已阅去重
}

// Filter 过滤视频
// 关注/朋友流：有限集合，仅做质量和拉黑检查，不做去重和分类多样性裁剪
// 推荐流：完整过滤链路（已观看去重、质量、分类多样性、已推荐去重）
func (f *VideoFilter) Filter(ctx context.Context, req *FilterRequest) ([]*model.Video, error) {
	isSocialFeed := req.Scene == "follow" || req.Scene == "friends"

	// 推荐流：加载已观看和已推荐集合用于去重
	// 关注/朋友流：不加载，避免有限集合被过度过滤
	var watchedSet map[uint]bool
	var recommendedSet map[uint]bool
	if !isSocialFeed {
		var err error
		watchedSet, err = f.getWatchedVideos(ctx, req.UserID)
		if err != nil {
			watchedSet = make(map[uint]bool)
		}
		recommendedSet, err = f.getRecommendedVideos(ctx, req.UserID)
		if err != nil {
			recommendedSet = make(map[uint]bool)
		}
	}

	const (
		maxPerCategory  = 5  // 单分类最大数量
		minResultSize   = 10 // 触发兜底补充的阈值
		maxBackfillSize = 30 // 兜底补充上限
	)

	var result []*model.Video
	var backfill []*model.Video // 不满足分类多样性或已推荐过的视频
	seenCategories := make(map[uint]int)

	for _, video := range req.Videos {
		// 过滤自己的视频
		if req.UserID > 0 && video.UserID == req.UserID {
			continue
		}

		// 推荐流：已观看去重（关注/朋友流不检查）
		if !isSocialFeed && watchedSet[video.ID] {
			continue
		}

		// 质量控制（所有场景统一检查）
		if !f.passQualityFilter(video) {
			continue
		}

		// 拉黑检查
		if f.isBlockedAuthor(ctx, req.UserID, video.UserID) {
			continue
		}

		if isSocialFeed {
			// 关注/朋友流：不限制分类多样性，全量保留
			result = append(result, video)
		} else {
			catID := uint(0)
			if video.CategoryID != nil {
				catID = *video.CategoryID
			}

			if recommendedSet[video.ID] {
				// 已推荐过但用户未观看，放入兜底池
				backfill = append(backfill, video)
			} else if catID == 0 || seenCategories[catID] < maxPerCategory {
				// 未分类视频不受多样性限制；已分类视频检查上限
				result = append(result, video)
				if catID > 0 {
					seenCategories[catID]++
				}
				// 记录推荐，用于后续请求去重
				f.recordRecommendation(ctx, req.UserID, video.ID)
			} else {
				// 分类已达上限，放入兜底池
				backfill = append(backfill, video)
			}
		}
	}

	// 兜底策略：推荐流结果不足时，从兜底池补充
	if !isSocialFeed && len(result) < minResultSize && len(backfill) > 0 {
		for _, video := range backfill {
			result = append(result, video)
			if len(result) >= maxBackfillSize {
				break
			}
		}
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
	// 质量分数过滤
	if video.QualityScore < 0 {
		return false
	}

	// 视频状态不是已发布
	if video.Status != 1 {
		return false
	}

	// 视频时长异常（Duration=0 允许通过，可能是刚上传未处理完）
	// 只过滤明显异常的：太短(<1秒)或太长(>1小时)
	if video.Duration > 0 && (video.Duration < 1 || video.Duration > 3600) {
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

// getRecommendedVideos 获取最近已推荐给用户的视频ID集合
func (f *VideoFilter) getRecommendedVideos(ctx context.Context, userID uint) (map[uint]bool, error) {
	if userID == 0 {
		return make(map[uint]bool), nil
	}

	cacheKey := fmt.Sprintf("user:recommended:%d", userID)
	videoIDs, err := f.redis.SMembers(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	recommendedSet := make(map[uint]bool)
	for _, idStr := range videoIDs {
		var id uint
		fmt.Sscanf(idStr, "%d", &id)
		recommendedSet[id] = true
	}

	return recommendedSet, nil
}

// recordRecommendation 记录推荐（用于去重和效果分析）
func (f *VideoFilter) recordRecommendation(ctx context.Context, userID, videoID uint) {
	if userID == 0 {
		return
	}

	// 记录到 Redis，用于后续去重
	cacheKey := fmt.Sprintf("user:recommended:%d", userID)
	f.redis.SAdd(ctx, cacheKey, fmt.Sprintf("%d", videoID))
	f.redis.Expire(ctx, cacheKey, 24*time.Hour)
}

// RecordWatched 记录已观看视频（播放进度上报时调用，实时更新去重集合）
func (f *VideoFilter) RecordWatched(ctx context.Context, userID, videoID uint) {
	if userID == 0 {
		return
	}

	cacheKey := fmt.Sprintf("user:watched:%d", userID)
	f.redis.SAdd(ctx, cacheKey, fmt.Sprintf("%d", videoID))
	f.redis.Expire(ctx, cacheKey, 7*24*time.Hour)
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
