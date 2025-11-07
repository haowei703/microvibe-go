package rank

import (
	"context"
	"math"
	"microvibe-go/internal/model"
	"sort"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Ranker 排序器
type Ranker struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewRanker 创建排序器实例
func NewRanker(db *gorm.DB, redis *redis.Client) *Ranker {
	return &Ranker{
		db:    db,
		redis: redis,
	}
}

// RankRequest 排序请求
type RankRequest struct {
	UserID   uint
	Videos   []*model.Video
	Features map[string]interface{}
}

// VideoScore 视频分数
type VideoScore struct {
	Video *model.Video
	Score float64
}

// Rank 对视频进行排序
func (r *Ranker) Rank(ctx context.Context, req *RankRequest) ([]*model.Video, error) {
	var scores []VideoScore

	// 计算每个视频的综合分数
	for _, video := range req.Videos {
		score := r.calculateScore(ctx, req.UserID, video, req.Features)
		scores = append(scores, VideoScore{
			Video: video,
			Score: score,
		})
	}

	// 按分数降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 提取排序后的视频列表
	var rankedVideos []*model.Video
	for _, s := range scores {
		rankedVideos = append(rankedVideos, s.Video)
	}

	return rankedVideos, nil
}

// calculateScore 计算视频的综合分数
// 采用多目标加权融合的方式
func (r *Ranker) calculateScore(ctx context.Context, userID uint, video *model.Video, features map[string]interface{}) float64 {
	// 1. 点击率预估（CTR）权重：30%
	ctrScore := r.estimateCTR(userID, video, features) * 0.3

	// 2. 完播率预估权重：25%
	finishScore := r.estimateFinishRate(userID, video, features) * 0.25

	// 3. 互动率预估（点赞、评论、分享）权重：25%
	engagementScore := r.estimateEngagement(userID, video, features) * 0.25

	// 4. 热度分数权重：10%
	hotScore := r.normalizeScore(video.HotScore, 0, 1000) * 0.1

	// 5. 新鲜度分数权重：10%
	freshnessScore := r.calculateFreshnessScore(video) * 0.1

	// 综合分数
	totalScore := ctrScore + finishScore + engagementScore + hotScore + freshnessScore

	return totalScore
}

// estimateCTR 预估点击率
func (r *Ranker) estimateCTR(userID uint, video *model.Video, features map[string]interface{}) float64 {
	// 简化的CTR预估模型
	// 实际项目中可以使用机器学习模型（LR、GBDT、DeepFM等）

	score := 0.5 // 基础分

	// 视频质量分
	score += r.normalizeScore(video.QualityScore, 0, 100) * 0.2

	// 历史CTR
	if video.PlayCount > 0 {
		historicalCTR := float64(video.PlayCount) / float64(video.PlayCount+1000) // 平滑处理
		score += historicalCTR * 0.3
	}

	// 用户兴趣匹配度
	if userFeature, ok := features["user"]; ok {
		if uf, ok := userFeature.(map[string]interface{}); ok {
			if interests, ok := uf["interest_tags"].(map[uint]float64); ok {
				if interestScore, exists := interests[video.CategoryID]; exists {
					score += interestScore * 0.5
				}
			}
		}
	}

	return math.Min(score, 1.0)
}

// estimateFinishRate 预估完播率
func (r *Ranker) estimateFinishRate(userID uint, video *model.Video, features map[string]interface{}) float64 {
	score := 0.5 // 基础分

	// 视频时长因素（太长的视频完播率通常较低）
	durationScore := 1.0
	if video.Duration > 300 { // 超过5分钟
		durationScore = 300.0 / float64(video.Duration)
	}
	score += durationScore * 0.3

	// 历史完播率数据
	var stats model.VideoStats
	if err := r.db.Where("video_id = ?", video.ID).
		Order("date DESC").
		First(&stats).Error; err == nil {
		score += stats.FinishRate * 0.4
	}

	// 用户平均完播率
	if userFeature, ok := features["user"]; ok {
		if uf, ok := userFeature.(map[string]interface{}); ok {
			if avgFinishRate, ok := uf["avg_finish_rate"].(float64); ok {
				score += avgFinishRate * 0.3
			}
		}
	}

	return math.Min(score, 1.0)
}

// estimateEngagement 预估互动率
func (r *Ranker) estimateEngagement(userID uint, video *model.Video, features map[string]interface{}) float64 {
	score := 0.0

	if video.PlayCount > 0 {
		// 点赞率
		likeRate := float64(video.LikeCount) / float64(video.PlayCount)
		score += likeRate * 0.4

		// 评论率
		commentRate := float64(video.CommentCount) / float64(video.PlayCount)
		score += commentRate * 0.3

		// 分享率
		shareRate := float64(video.ShareCount) / float64(video.PlayCount)
		score += shareRate * 0.3
	}

	return math.Min(score, 1.0)
}

// calculateFreshnessScore 计算新鲜度分数
func (r *Ranker) calculateFreshnessScore(video *model.Video) float64 {
	if video.PublishedAt == nil {
		return 0.0
	}

	// 使用指数衰减函数
	// 半衰期为24小时
	hoursSincePublish := float64(video.UpdatedAt.Unix()-video.PublishedAt.Unix()) / 3600.0
	halfLife := 24.0
	freshnessScore := math.Exp(-0.693 * hoursSincePublish / halfLife)

	return freshnessScore
}

// normalizeScore 归一化分数到 [0, 1]
func (r *Ranker) normalizeScore(value, min, max float64) float64 {
	if max == min {
		return 0.0
	}
	normalized := (value - min) / (max - min)
	if normalized < 0 {
		return 0.0
	}
	if normalized > 1 {
		return 1.0
	}
	return normalized
}

// ApplyDiversity 应用多样性策略（避免信息茧房）
func (r *Ranker) ApplyDiversity(videos []*model.Video, diversityRatio float64) []*model.Video {
	if diversityRatio <= 0 || len(videos) == 0 {
		return videos
	}

	// 计算需要多样化的数量
	diversityCount := int(float64(len(videos)) * diversityRatio)
	if diversityCount == 0 {
		return videos
	}

	// 分类统计
	categoryCount := make(map[uint]int)
	result := make([]*model.Video, 0, len(videos))

	for _, video := range videos {
		// 如果该分类已经太多，跳过（实现多样性）
		if categoryCount[video.CategoryID] >= 3 {
			continue
		}
		categoryCount[video.CategoryID]++
		result = append(result, video)
	}

	return result
}
