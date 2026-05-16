package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
)

// VideoStatsService 视频统计服务接口
type VideoStatsService interface {
	// RecordPlay 记录播放行为
	RecordPlay(ctx context.Context, videoID uint, duration, position int, finished bool) error
	// RecordLike 记录点赞行为
	RecordLike(ctx context.Context, videoID uint, delta int) error
	// RecordFavorite 记录收藏行为
	RecordFavorite(ctx context.Context, videoID uint, delta int) error
	// RecordComment 记录评论行为
	RecordComment(ctx context.Context, videoID uint, delta int) error
	// RecordShare 记录分享行为
	RecordShare(ctx context.Context, videoID uint) error
	// GetVideoStats 获取视频总览统计
	GetVideoStats(ctx context.Context, ownerID, videoID uint) (*VideoStatsVO, error)
	// GetVideoDailyStats 获取视频每日统计
	GetVideoDailyStats(ctx context.Context, ownerID, videoID uint, days int) ([]*model.VideoStats, error)
	// GetCreatorStats 获取创作者所有视频汇总统计
	GetCreatorStats(ctx context.Context, userID uint) (*CreatorStatsVO, error)
	// GetCreatorTrendingStats 获取创作者近期趋势统计
	GetCreatorTrendingStats(ctx context.Context, userID uint, days int) ([]*repository.TrendingStats, error)
}

// VideoStatsVO 视频统计视图对象
type VideoStatsVO struct {
	VideoID            uint    `json:"video_id"`
	TotalPlayCount     int64   `json:"total_play_count"`
	TotalLikeCount     int64   `json:"total_like_count"`
	TotalCommentCount  int64   `json:"total_comment_count"`
	TotalFavoriteCount int64   `json:"total_favorite_count"`
	TotalShareCount    int64   `json:"total_share_count"`
	AvgDuration        float64 `json:"avg_duration"`
	FinishRate         float64 `json:"finish_rate"`
}

// CreatorStatsVO 创作者统计视图对象
type CreatorStatsVO struct {
	TotalVideos        int64 `json:"total_videos"`
	TotalPlayCount     int64 `json:"total_play_count"`
	TotalLikeCount     int64 `json:"total_like_count"`
	TotalCommentCount  int64 `json:"total_comment_count"`
	TotalFavoriteCount int64 `json:"total_favorite_count"`
	TotalShareCount    int64 `json:"total_share_count"`
	FollowerCount      int64 `json:"follower_count"`
}

type videoStatsServiceImpl struct {
	statsRepo  repository.VideoStatsRepository
	videoRepo  repository.VideoRepository
	followRepo repository.FollowRepository
}

// NewVideoStatsService 创建视频统计服务实例
func NewVideoStatsService(
	statsRepo repository.VideoStatsRepository,
	videoRepo repository.VideoRepository,
	followRepo repository.FollowRepository,
) VideoStatsService {
	return &videoStatsServiceImpl{
		statsRepo:  statsRepo,
		videoRepo:  videoRepo,
		followRepo: followRepo,
	}
}

// RecordPlay 记录播放行为
func (s *videoStatsServiceImpl) RecordPlay(ctx context.Context, videoID uint, duration, position int, finished bool) error {
	today := time.Now().Truncate(24 * time.Hour)
	return s.statsRepo.IncrementPlay(ctx, videoID, today, position, finished)
}

// RecordLike 记录点赞行为
func (s *videoStatsServiceImpl) RecordLike(ctx context.Context, videoID uint, delta int) error {
	today := time.Now().Truncate(24 * time.Hour)
	return s.statsRepo.IncrementField(ctx, videoID, today, "like_count", delta)
}

// RecordFavorite 记录收藏行为
func (s *videoStatsServiceImpl) RecordFavorite(ctx context.Context, videoID uint, delta int) error {
	today := time.Now().Truncate(24 * time.Hour)
	return s.statsRepo.IncrementField(ctx, videoID, today, "favorite_count", delta)
}

// RecordComment 记录评论行为
func (s *videoStatsServiceImpl) RecordComment(ctx context.Context, videoID uint, delta int) error {
	today := time.Now().Truncate(24 * time.Hour)
	return s.statsRepo.IncrementField(ctx, videoID, today, "comment_count", delta)
}

// RecordShare 记录分享行为
func (s *videoStatsServiceImpl) RecordShare(ctx context.Context, videoID uint) error {
	today := time.Now().Truncate(24 * time.Hour)
	return s.statsRepo.IncrementField(ctx, videoID, today, "share_count", 1)
}

// GetVideoStats 获取视频总览统计
func (s *videoStatsServiceImpl) GetVideoStats(ctx context.Context, ownerID, videoID uint) (*VideoStatsVO, error) {
	// 查询视频信息（验证权限）
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		logger.Error("查询视频失败", zap.Error(err), zap.Uint("video_id", videoID))
		return nil, errors.New("视频不存在")
	}

	// 权限校验：只有视频作者可查看统计
	if video.UserID != ownerID {
		return nil, errors.New("无权查看该视频统计")
	}

	// 从 videos 表取累计总量
	vo := &VideoStatsVO{
		VideoID:            videoID,
		TotalPlayCount:     video.PlayCount,
		TotalLikeCount:     video.LikeCount,
		TotalCommentCount:  video.CommentCount,
		TotalFavoriteCount: video.FavoriteCount,
		TotalShareCount:    video.ShareCount,
	}

	// 从 video_stats 聚合 avg_duration 和 finish_rate（最近30天加权平均）
	stats, err := s.statsRepo.FindByVideoID(ctx, videoID, 30)
	if err != nil {
		logger.Error("查询视频统计数据失败", zap.Error(err))
		return vo, nil // 返回基础数据，不阻塞
	}

	if len(stats) > 0 {
		var totalPlayCount int64
		var weightedDuration float64
		var weightedFinishRate float64

		for _, stat := range stats {
			totalPlayCount += stat.PlayCount
			weightedDuration += stat.AvgDuration * float64(stat.PlayCount)
			weightedFinishRate += stat.FinishRate * float64(stat.PlayCount)
		}

		if totalPlayCount > 0 {
			vo.AvgDuration = weightedDuration / float64(totalPlayCount)
			vo.FinishRate = weightedFinishRate / float64(totalPlayCount)
		}
	}

	return vo, nil
}

// GetVideoDailyStats 获取视频每日统计
func (s *videoStatsServiceImpl) GetVideoDailyStats(ctx context.Context, ownerID, videoID uint, days int) ([]*model.VideoStats, error) {
	// 查询视频信息（验证权限）
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return nil, errors.New("视频不存在")
	}

	// 权限校验
	if video.UserID != ownerID {
		return nil, errors.New("无权查看该视频统计")
	}

	// 限制查询天数
	if days <= 0 || days > 90 {
		days = 7
	}

	return s.statsRepo.FindByVideoID(ctx, videoID, days)
}

// GetCreatorStats 获取创作者所有视频汇总统计
func (s *videoStatsServiceImpl) GetCreatorStats(ctx context.Context, userID uint) (*CreatorStatsVO, error) {
	summary, err := s.statsRepo.SumByUserID(ctx, userID)
	if err != nil {
		logger.Error("获取创作者统计失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, errors.New("获取统计数据失败")
	}

	vo := &CreatorStatsVO{
		TotalVideos:        summary.TotalVideos,
		TotalPlayCount:     summary.TotalPlayCount,
		TotalLikeCount:     summary.TotalLikeCount,
		TotalCommentCount:  summary.TotalCommentCount,
		TotalFavoriteCount: summary.TotalFavoriteCount,
		TotalShareCount:    summary.TotalShareCount,
	}

	// 从 follows 表查询粉丝数
	followerCount, err := s.followRepo.CountFollowers(ctx, userID)
	if err == nil {
		vo.FollowerCount = followerCount
	}

	return vo, nil
}

// GetCreatorTrendingStats 获取创作者近期趋势统计
func (s *videoStatsServiceImpl) GetCreatorTrendingStats(ctx context.Context, userID uint, days int) ([]*repository.TrendingStats, error) {
	if days <= 0 || days > 90 {
		days = 7
	}

	return s.statsRepo.SumTrendingByUserID(ctx, userID, days)
}
