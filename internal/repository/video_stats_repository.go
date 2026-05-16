package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoStatsRepository 视频统计数据仓库接口
type VideoStatsRepository interface {
	// IncrementPlay 原子累加播放相关统计
	IncrementPlay(ctx context.Context, videoID uint, date time.Time, position int, finished bool) error
	// IncrementField 原子累加指定字段
	IncrementField(ctx context.Context, videoID uint, date time.Time, field string, delta int) error
	// FindByVideoID 查询视频最近N天的统计数据
	FindByVideoID(ctx context.Context, videoID uint, days int) ([]*model.VideoStats, error)
	// SumByUserID 聚合创作者所有视频的统计数据
	SumByUserID(ctx context.Context, userID uint) (*VideoStatsSummary, error)
	// SumTrendingByUserID 聚合创作者近期趋势统计
	SumTrendingByUserID(ctx context.Context, userID uint, days int) ([]*TrendingStats, error)
}

// TrendingStats 趋势统计项
type TrendingStats struct {
	Date        time.Time `json:"date"`
	NewPlay     int64     `json:"new_play"`
	NewLike     int64     `json:"new_like"`
	NewComment  int64     `json:"new_comment"`
	NewFavorite int64     `json:"new_favorite"`
}

// VideoStatsSummary 创作者视频统计汇总
type VideoStatsSummary struct {
	TotalVideos        int64 `json:"total_videos"`
	TotalPlayCount     int64 `json:"total_play_count"`
	TotalLikeCount     int64 `json:"total_like_count"`
	TotalCommentCount  int64 `json:"total_comment_count"`
	TotalFavoriteCount int64 `json:"total_favorite_count"`
	TotalShareCount    int64 `json:"total_share_count"`
}

type videoStatsRepositoryImpl struct {
	db *gorm.DB
}

// NewVideoStatsRepository 创建视频统计数据仓库实例
func NewVideoStatsRepository(db *gorm.DB) VideoStatsRepository {
	return &videoStatsRepositoryImpl{db: db}
}

// ensureRow 确保当天记录存在（INSERT IGNORE）
func (r *videoStatsRepositoryImpl) ensureRow(ctx context.Context, videoID uint, date time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO video_stats (video_id, date, play_count, unique_view_count, avg_duration, finish_rate,
			like_count, comment_count, share_count, favorite_count,
			recommend_count, search_count, follow_count, share_traffic, created_at, updated_at)
		VALUES (?, ?, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, NOW(), NOW())
		ON CONFLICT (video_id, date) DO NOTHING
	`, videoID, date).Error
}

// IncrementPlay 原子累加播放相关统计（加权平均 avg_duration 和 finish_rate）
func (r *videoStatsRepositoryImpl) IncrementPlay(ctx context.Context, videoID uint, date time.Time, position int, finished bool) error {
	if err := r.ensureRow(ctx, videoID, date); err != nil {
		logger.Error("确保统计行存在失败", zap.Error(err), zap.Uint("video_id", videoID))
		return err
	}

	finishedVal := 0
	if finished {
		finishedVal = 1
	}

	// 加权平均：new_avg = (old_avg * old_count + new_val) / new_count
	err := r.db.WithContext(ctx).Exec(`
		UPDATE video_stats SET
			play_count        = play_count + 1,
			unique_view_count = unique_view_count + 1,
			avg_duration      = (avg_duration * play_count + ?) / (play_count + 1),
			finish_rate       = (finish_rate * play_count + ?) / (play_count + 1),
			updated_at        = NOW()
		WHERE video_id = ? AND date = ?
	`, position, finishedVal, videoID, date).Error

	if err != nil {
		logger.Error("更新播放统计失败", zap.Error(err), zap.Uint("video_id", videoID))
	}
	return err
}

// IncrementField 原子累加指定字段（like_count, comment_count, share_count, favorite_count）
func (r *videoStatsRepositoryImpl) IncrementField(ctx context.Context, videoID uint, date time.Time, field string, delta int) error {
	// 白名单校验，防止 SQL 注入
	allowed := map[string]bool{
		"like_count": true, "comment_count": true,
		"share_count": true, "favorite_count": true,
	}
	if !allowed[field] {
		return nil
	}

	if err := r.ensureRow(ctx, videoID, date); err != nil {
		return err
	}

	err := r.db.WithContext(ctx).Exec(
		"UPDATE video_stats SET "+field+" = "+field+" + ?, updated_at = NOW() WHERE video_id = ? AND date = ?",
		delta, videoID, date,
	).Error

	if err != nil {
		logger.Error("更新统计字段失败", zap.Error(err), zap.Uint("video_id", videoID), zap.String("field", field))
	}
	return err
}

// FindByVideoID 查询视频最近N天的统计数据
func (r *videoStatsRepositoryImpl) FindByVideoID(ctx context.Context, videoID uint, days int) ([]*model.VideoStats, error) {
	var stats []*model.VideoStats
	startDate := time.Now().AddDate(0, 0, -days)

	err := r.db.WithContext(ctx).
		Where("video_id = ? AND date >= ?", videoID, startDate).
		Order("date ASC").
		Find(&stats).Error

	if err != nil {
		logger.Error("查询视频统计数据失败", zap.Error(err), zap.Uint("video_id", videoID))
		return nil, err
	}

	return stats, nil
}

// SumByUserID 聚合创作者所有视频的统计数据（从 videos 表取累计总量）
func (r *videoStatsRepositoryImpl) SumByUserID(ctx context.Context, userID uint) (*VideoStatsSummary, error) {
	var summary VideoStatsSummary

	err := r.db.WithContext(ctx).
		Model(&model.Video{}).
		Select(`
			COUNT(*) as total_videos,
			COALESCE(SUM(play_count), 0) as total_play_count,
			COALESCE(SUM(like_count), 0) as total_like_count,
			COALESCE(SUM(comment_count), 0) as total_comment_count,
			COALESCE(SUM(favorite_count), 0) as total_favorite_count,
			COALESCE(SUM(share_count), 0) as total_share_count
		`).
		Where("user_id = ?", userID).
		Scan(&summary).Error

	if err != nil {
		logger.Error("聚合创作者视频统计失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, err
	}

	return &summary, nil
}

// SumTrendingByUserID 聚合创作者近期趋势统计
func (r *videoStatsRepositoryImpl) SumTrendingByUserID(ctx context.Context, userID uint, days int) ([]*TrendingStats, error) {
	var stats []*TrendingStats
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	err := r.db.WithContext(ctx).
		Table("video_stats").
		Select(`
			video_stats.date,
			SUM(video_stats.play_count) as new_play,
			SUM(video_stats.like_count) as new_like,
			SUM(video_stats.comment_count) as new_comment,
			SUM(video_stats.favorite_count) as new_favorite
		`).
		Joins("JOIN videos ON videos.id = video_stats.video_id").
		Where("videos.user_id = ? AND video_stats.date >= ?", userID, startDate).
		Group("video_stats.date").
		Order("video_stats.date ASC").
		Scan(&stats).Error

	if err != nil {
		logger.Error("查询创作者趋势统计失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, err
	}

	return stats, nil
}
