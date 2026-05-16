package recommend

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// Recaller 召回器
type Recaller struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewRecaller 创建召回器实例
func NewRecaller(db *gorm.DB, redis *redis.Client) *Recaller {
	return &Recaller{
		db:    db,
		redis: redis,
	}
}

// RecallRequest 召回请求
type RecallRequest struct {
	UserID uint   // 用户ID
	Scene  string // 场景
	Limit  int    // 召回数量
}

// Recall 多路召回策略
func (r *Recaller) Recall(ctx context.Context, req *RecallRequest) ([]*model.Video, error) {
	// 如果是关注流或朋友流场景，采用纯净召回策略
	if req.Scene == "follow" {
		return r.followRecall(ctx, req.UserID, req.Limit)
	}
	if req.Scene == "friends" {
		return r.friendsRecall(ctx, req.UserID, req.Limit)
	}

	// 其他场景并发执行 4 路召回
	type recallResult struct {
		videos []*model.Video
	}
	results := make([]recallResult, 4)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		videos, _ := r.collaborativeFilteringRecall(egCtx, req.UserID, req.Limit/4)
		results[0] = recallResult{videos: videos}
		return nil
	})
	eg.Go(func() error {
		videos, _ := r.contentBasedRecall(egCtx, req.UserID, req.Limit/4)
		results[1] = recallResult{videos: videos}
		return nil
	})
	eg.Go(func() error {
		videos, _ := r.hotRecall(egCtx, req.Limit/4)
		results[2] = recallResult{videos: videos}
		return nil
	})
	eg.Go(func() error {
		videos, _ := r.newVideoRecall(egCtx, req.Limit/4)
		results[3] = recallResult{videos: videos}
		return nil
	})

	_ = eg.Wait()

	// 合并去重
	videoMap := make(map[uint]*model.Video)
	for _, res := range results {
		for _, v := range res.videos {
			videoMap[v.ID] = v
		}
	}

	videos := make([]*model.Video, 0, len(videoMap))
	for _, v := range videoMap {
		videos = append(videos, v)
	}

	// 召回不足时兜底随机补充
	if len(videos) < req.Limit {
		randomVideos, err := r.randomRecall(ctx, req.Limit-len(videos))
		if err == nil {
			for _, v := range randomVideos {
				if _, exists := videoMap[v.ID]; !exists {
					videos = append(videos, v)
				}
			}
		}
	}

	return videos, nil
}

// collaborativeFilteringRecall 协同过滤召回
func (r *Recaller) collaborativeFilteringRecall(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	// 获取当前用户最近喜欢的视频
	var userLikes []model.Like
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&userLikes).Error; err != nil {
		return nil, err
	}

	// 新用户冷启动：无点赞记录时降级为热门召回
	if len(userLikes) == 0 {
		return r.hotRecall(ctx, limit)
	}

	// 提取视频ID
	var videoIDs []uint
	for _, like := range userLikes {
		videoIDs = append(videoIDs, like.VideoID)
	}

	// 找到也喜欢这些视频的其他用户
	var similarUsers []uint
	if err := r.db.WithContext(ctx).Model(&model.Like{}).
		Select("DISTINCT user_id").
		Where("video_id IN ? AND user_id != ?", videoIDs, userID).
		Limit(50).
		Pluck("user_id", &similarUsers).Error; err != nil {
		return nil, err
	}

	if len(similarUsers) == 0 {
		return r.hotRecall(ctx, limit)
	}

	// 获取这些相似用户喜欢的视频（排除当前用户已经看过的）
	var videos []*model.Video
	if err := r.db.WithContext(ctx).Preload("User").
		Joins("INNER JOIN likes ON videos.id = likes.video_id").
		Where("likes.user_id IN ?", similarUsers).
		Where("videos.id NOT IN ?", videoIDs).
		Where("videos.status = ?", 1).
		Group("videos.id").
		Order("COUNT(likes.id) DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}

// contentBasedRecall 基于内容的召回
func (r *Recaller) contentBasedRecall(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	// 1. 获取用户兴趣标签
	var interests []model.UserInterest
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("score DESC").
		Limit(5).
		Find(&interests).Error; err != nil {
		return nil, err
	}

	if len(interests) == 0 {
		return []*model.Video{}, nil
	}

	// 2. 根据用户兴趣的分类召回视频
	var categoryIDs []uint
	for _, interest := range interests {
		categoryIDs = append(categoryIDs, interest.CategoryID)
	}

	var videos []*model.Video
	if err := r.db.WithContext(ctx).Where("category_id IN ?", categoryIDs).
		Preload("User").
		Where("status = ?", 1).
		Order("hot_score DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}

// hotRecall 热门召回
func (r *Recaller) hotRecall(ctx context.Context, limit int) ([]*model.Video, error) {
	// 从 Redis 获取热门视频ID列表
	cacheKey := "hot:videos:7d"
	videoIDs, err := r.redis.ZRevRange(ctx, cacheKey, 0, int64(limit-1)).Result()
	if err == nil && len(videoIDs) > 0 {
		var ids []uint
		for _, idStr := range videoIDs {
			var id uint
			fmt.Sscanf(idStr, "%d", &id)
			ids = append(ids, id)
		}

		var videos []*model.Video
		if err := r.db.WithContext(ctx).Preload("User").Where("id IN ?", ids).Find(&videos).Error; err == nil {
			return videos, nil
		}
	}

	// 缓存未命中，从数据库查询（扩大到7天窗口，保证内容充足）
	var videos []*model.Video
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	if err := r.db.WithContext(ctx).Where("status = ? AND published_at > ?", 1, sevenDaysAgo).
		Preload("User").
		Order("hot_score DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	// 更新缓存
	if len(videos) > 0 {
		pipe := r.redis.Pipeline()
		for _, video := range videos {
			pipe.ZAdd(ctx, cacheKey, redis.Z{
				Score:  video.HotScore,
				Member: fmt.Sprintf("%d", video.ID),
			})
		}
		pipe.Expire(ctx, cacheKey, 1*time.Hour)
		pipe.Exec(ctx)
	}

	return videos, nil
}

// followRecall 关注召回：获取关注用户的所有视频，按发布时间降序，不做已阅去重
func (r *Recaller) followRecall(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	var follows []model.Follow
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Limit(500).
		Find(&follows).Error; err != nil {
		return nil, err
	}

	if len(follows) == 0 {
		return []*model.Video{}, nil
	}

	var followedIDs []uint
	for _, follow := range follows {
		followedIDs = append(followedIDs, follow.FollowedID)
	}

	var videos []*model.Video
	if err := r.db.WithContext(ctx).Where("user_id IN ?", followedIDs).
		Preload("User").
		Where("status = ?", 1).
		Order("published_at DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}

// newVideoRecall 新视频召回（冷启动）
func (r *Recaller) newVideoRecall(ctx context.Context, limit int) ([]*model.Video, error) {
	var videos []*model.Video
	// 扩大到72小时，保证新视频有足够曝光窗口
	threeDaysAgo := time.Now().Add(-72 * time.Hour)
	if err := r.db.WithContext(ctx).Where("status = ? AND published_at > ?", 1, threeDaysAgo).
		Preload("User").
		Order("published_at DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}

// randomRecall 随机召回（兜底策略）
func (r *Recaller) randomRecall(ctx context.Context, limit int) ([]*model.Video, error) {
	var videos []*model.Video
	// 优先挑选热门视频中的一部分进行随机，保证兜底质量
	if err := r.db.WithContext(ctx).Where("status = ?", 1).
		Preload("User").
		Order("hot_score DESC, RANDOM()").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}

// friendsRecall 朋友召回 (双向关注)
func (r *Recaller) friendsRecall(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	// 获取互相关注的人 (朋友)
	var friendIDs []uint
	if err := r.db.WithContext(ctx).Table("follows f1").
		Select("f1.followed_id").
		Joins("INNER JOIN follows f2 ON f1.user_id = f2.followed_id AND f1.followed_id = f2.user_id").
		Where("f1.user_id = ?", userID).
		Limit(200).
		Pluck("f1.followed_id", &friendIDs).Error; err != nil {
		return nil, err
	}

	if len(friendIDs) == 0 {
		return []*model.Video{}, nil
	}

	// 获取朋友发布的视频
	var videos []*model.Video
	if err := r.db.WithContext(ctx).Where("user_id IN ?", friendIDs).
		Preload("User").
		Where("status = ?", 1).
		Order("published_at DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}
