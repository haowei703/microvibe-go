package recommend

import (
	"context"
	"encoding/json"
	"fmt"
	"microvibe-go/internal/algorithm/feature"
	"microvibe-go/internal/algorithm/filter"
	"microvibe-go/internal/algorithm/rank"
	"microvibe-go/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Engine 推荐引擎
type Engine struct {
	db          *gorm.DB
	redis       *redis.Client
	recaller    *Recaller
	featureEng  *feature.Engineer
	ranker      *rank.Ranker
	videoFilter *filter.VideoFilter
}

// NewEngine 创建推荐引擎实例
func NewEngine(db *gorm.DB, redis *redis.Client) *Engine {
	return &Engine{
		db:          db,
		redis:       redis,
		recaller:    NewRecaller(db, redis),
		featureEng:  feature.NewEngineer(db, redis),
		ranker:      rank.NewRanker(db, redis),
		videoFilter: filter.NewVideoFilter(db, redis),
	}
}

// RecommendRequest 推荐请求
type RecommendRequest struct {
	UserID   uint   // 用户ID
	Page     int    // 页码
	PageSize int    // 每页数量
	Scene    string // 场景：feed-推荐流、follow-关注、hot-热门
}

// RecommendResponse 推荐响应
type RecommendResponse struct {
	Videos []*model.Video // 推荐视频列表
	Total  int64          // 总数
}

// cacheKey 生成推荐结果缓存键
func cacheKey(scene string, userID uint, page, pageSize int) string {
	if userID > 0 {
		return fmt.Sprintf("recommend:%s:%d:%d:%d", scene, userID, page, pageSize)
	}
	return fmt.Sprintf("recommend:%s:anon:%d:%d", scene, page, pageSize)
}

// cacheTTL 推荐结果缓存过期时间
func cacheTTL(scene string) time.Duration {
	switch scene {
	case "hot":
		return 30 * time.Second
	case "feed":
		return 20 * time.Second
	default:
		return 10 * time.Second
	}
}

// Recommend 获取推荐视频（带 Redis 缓存）
// 核心推荐流程：缓存 -> 召回 -> 特征工程 -> 排序 -> 过滤
func (e *Engine) Recommend(ctx context.Context, req *RecommendRequest) (*RecommendResponse, error) {
	// 0. 尝试从缓存获取（匿名和热门场景优先走缓存）
	if e.redis != nil && (req.UserID == 0 || req.Scene == "hot") {
		key := cacheKey(req.Scene, req.UserID, req.Page, req.PageSize)
		cached, err := e.redis.Get(ctx, key).Result()
		if err == nil && cached != "" {
			var resp RecommendResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return &resp, nil
			}
		}
	}

	// 1. 召回阶段：从海量视频中快速召回候选集
	candidates, err := e.recaller.Recall(ctx, &RecallRequest{
		UserID: req.UserID,
		Scene:  req.Scene,
		Limit:  req.PageSize * 10, // 召回数量是最终需要的10倍
	})
	if err != nil {
		return nil, err
	}

	// 2. 特征工程：提取用户和视频特征
	features, err := e.featureEng.Extract(ctx, req.UserID, candidates)
	if err != nil {
		return nil, err
	}

	// 3. 排序阶段：对候选视频进行精准排序
	rankedVideos, err := e.ranker.Rank(ctx, &rank.RankRequest{
		UserID:   req.UserID,
		Videos:   candidates,
		Features: features,
	})
	if err != nil {
		return nil, err
	}

	// 4. 过滤阶段：场景感知过滤（关注/朋友流跳过去重和多样性裁剪）
	filteredVideos, err := e.videoFilter.Filter(ctx, &filter.FilterRequest{
		UserID: req.UserID,
		Videos: rankedVideos,
		Scene:  req.Scene,
	})
	if err != nil {
		return nil, err
	}

	// 5. 分页处理
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize

	if start >= len(filteredVideos) {
		return &RecommendResponse{
			Videos: []*model.Video{},
			Total:  int64(len(filteredVideos)),
		}, nil
	}
	if end > len(filteredVideos) {
		end = len(filteredVideos)
	}

	resp := &RecommendResponse{
		Videos: filteredVideos[start:end],
		Total:  int64(len(filteredVideos)),
	}

	// 异步缓存结果（仅缓存匿名和热门场景）
	if e.redis != nil && (req.UserID == 0 || req.Scene == "hot") {
		key := cacheKey(req.Scene, req.UserID, req.Page, req.PageSize)
		if data, err := json.Marshal(resp); err == nil {
			e.redis.Set(ctx, key, string(data), cacheTTL(req.Scene))
		}
	}

	return resp, nil
}

// UpdateUserProfile 更新用户画像
func (e *Engine) UpdateUserProfile(ctx context.Context, userID uint, behavior *model.UserBehavior) error {
	return e.featureEng.UpdateUserProfile(ctx, userID, behavior)
}

// UpdateVideoFeature 更新视频特征
func (e *Engine) UpdateVideoFeature(ctx context.Context, videoID uint) error {
	return e.featureEng.UpdateVideoFeature(ctx, videoID)
}

// RecordWatched 记录已观看视频（供播放历史服务调用，实时更新去重集合）
func (e *Engine) RecordWatched(ctx context.Context, userID, videoID uint) {
	e.videoFilter.RecordWatched(ctx, userID, videoID)
}
