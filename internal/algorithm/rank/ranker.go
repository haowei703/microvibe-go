package rank

import (
	"context"
	"math"
	"microvibe-go/internal/algorithm/feature"
	"microvibe-go/internal/model"
	"sort"
	"time"

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
func (r *Ranker) calculateScore(_ context.Context, userID uint, video *model.Video, features map[string]interface{}) float64 {
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
func (r *Ranker) estimateCTR(_ uint, video *model.Video, features map[string]interface{}) float64 {
	score := 0.5 // 基础分

	// 视频质量分
	score += r.normalizeScore(video.QualityScore, 0, 100) * 0.2

	// 历史CTR
	if video.PlayCount > 0 {
		historicalCTR := float64(video.PlayCount) / float64(video.PlayCount+1000) // 平滑处理
		score += historicalCTR * 0.3
	}

	// 用户兴趣匹配度：直接断言为 *feature.UserFeature
	if uf, ok := features["user"].(*feature.UserFeature); ok && uf != nil {
		catID := uint(0)
		if video.CategoryID != nil {
			catID = *video.CategoryID
		}
		if interestScore, exists := uf.InterestTags[catID]; exists {
			score += interestScore * 0.5
		}
	}

	return math.Min(score, 1.0)
}

// estimateFinishRate 预估完播率
func (r *Ranker) estimateFinishRate(_ uint, video *model.Video, features map[string]interface{}) float64 {
	score := 0.5 // 基础分

	// 视频时长因素（太长的视频完播率通常较低）
	durationScore := 1.0
	if video.Duration > 300 { // 超过5分钟
		durationScore = 300.0 / float64(video.Duration)
	}
	score += durationScore * 0.3

	// 历史完播率数据
	var stats []model.VideoStats
	if err := r.db.Where("video_id = ?", video.ID).
		Order("date DESC").
		Limit(1).
		Find(&stats).Error; err == nil && len(stats) > 0 {
		score += stats[0].FinishRate * 0.4
	}

	// 用户平均完播率
	if uf, ok := features["user"].(*feature.UserFeature); ok && uf != nil {
		score += uf.AvgFinishRate * 0.3
	}

	return math.Min(score, 1.0)
}

// estimateEngagement 预估互动率
func (r *Ranker) estimateEngagement(_ uint, video *model.Video, _ map[string]interface{}) float64 {
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

	// 使用指数衰减函数，半衰期为24小时
	hoursSincePublish := time.Since(*video.PublishedAt).Hours()
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
		// 如果没有分类，不参与强力打散限制（避免未分类视频被大面积过滤）
		if video.CategoryID == nil {
			result = append(result, video)
			continue
		}

		catID := *video.CategoryID
		// 如果该分类已经太多，跳过（实现多样性）
		if categoryCount[catID] >= 3 {
			continue
		}
		categoryCount[catID]++
		result = append(result, video)
	}

	return result
}
