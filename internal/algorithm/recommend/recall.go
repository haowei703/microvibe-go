package recommend

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
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
	var videos []*model.Video
	videoMap := make(map[uint]*model.Video) // 用于去重

	// 1. 协同过滤召回（基于相似用户喜欢的视频）
	cfVideos, err := r.collaborativeFilteringRecall(ctx, req.UserID, req.Limit/4)
	if err == nil {
		for _, v := range cfVideos {
			videoMap[v.ID] = v
		}
	}

	// 2. 内容召回（基于用户兴趣标签）
	contentVideos, err := r.contentBasedRecall(ctx, req.UserID, req.Limit/4)
	if err == nil {
		for _, v := range contentVideos {
			videoMap[v.ID] = v
		}
	}

	// 3. 热门召回
	hotVideos, err := r.hotRecall(ctx, req.Limit/4)
	if err == nil {
		for _, v := range hotVideos {
			videoMap[v.ID] = v
		}
	}

	// 4. 关注召回（关注的人发布的视频）
	if req.Scene == "follow" {
		followVideos, err := r.followRecall(ctx, req.UserID, req.Limit/2)
		if err == nil {
			for _, v := range followVideos {
				videoMap[v.ID] = v
			}
		}
	}

	// 5. 新视频召回（保证新视频有曝光机会）
	newVideos, err := r.newVideoRecall(ctx, req.Limit/4)
	if err == nil {
		for _, v := range newVideos {
			videoMap[v.ID] = v
		}
	}

	// 转换为列表
	for _, v := range videoMap {
		videos = append(videos, v)
	}

	// 如果召回数量不足，补充随机视频
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
	// 简化实现：找到和当前用户行为相似的用户，推荐他们喜欢的视频
	// 1. 获取当前用户最近喜欢的视频
	var userLikes []model.Like
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&userLikes).Error; err != nil {
		return nil, err
	}

	if len(userLikes) == 0 {
		return []*model.Video{}, nil
	}

	// 提取视频ID
	var videoIDs []uint
	for _, like := range userLikes {
		videoIDs = append(videoIDs, like.VideoID)
	}

	// 2. 找到也喜欢这些视频的其他用户
	var similarUsers []uint
	if err := r.db.Model(&model.Like{}).
		Select("DISTINCT user_id").
		Where("video_id IN ? AND user_id != ?", videoIDs, userID).
		Limit(50).
		Pluck("user_id", &similarUsers).Error; err != nil {
		return nil, err
	}

	if len(similarUsers) == 0 {
		return []*model.Video{}, nil
	}

	// 3. 获取这些相似用户喜欢的视频（排除当前用户已经看过的）
	var videos []*model.Video
	if err := r.db.Table("videos").
		Joins("INNER JOIN likes ON videos.id = likes.video_id").
		Where("likes.user_id IN ?", similarUsers).
		Where("videos.id NOT IN ?", videoIDs).
		Where("videos.status = ?", 1). // 已发布
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
	if err := r.db.Where("user_id = ?", userID).
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
	if err := r.db.Where("category_id IN ?", categoryIDs).
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
	cacheKey := "hot:videos:24h"
	videoIDs, err := r.redis.ZRevRange(ctx, cacheKey, 0, int64(limit-1)).Result()
	if err == nil && len(videoIDs) > 0 {
		// 从缓存获取成功
		var ids []uint
		for _, idStr := range videoIDs {
			var id uint
			fmt.Sscanf(idStr, "%d", &id)
			ids = append(ids, id)
		}

		var videos []*model.Video
		if err := r.db.Where("id IN ?", ids).Find(&videos).Error; err == nil {
			return videos, nil
		}
	}

	// 缓存未命中，从数据库查询
	var videos []*model.Video
	yesterday := time.Now().Add(-24 * time.Hour)
	if err := r.db.Where("status = ? AND published_at > ?", 1, yesterday).
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

// followRecall 关注召回
func (r *Recaller) followRecall(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	// 获取用户关注的人
	var follows []model.Follow
	if err := r.db.Where("user_id = ?", userID).
		Limit(100).
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

	// 获取关注的人发布的视频
	var videos []*model.Video
	if err := r.db.Where("user_id IN ?", followedIDs).
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
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := r.db.Where("status = ? AND published_at > ?", 1, oneHourAgo).
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
	if err := r.db.Where("status = ?", 1).
		Order("RANDOM()").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, err
	}

	return videos, nil
}
