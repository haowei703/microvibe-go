package repository

import (
	"context"
	"microvibe-go/internal/model"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoRepository 视频数据访问层接口
type VideoRepository interface {
	// Create 创建视频
	Create(ctx context.Context, video *model.Video) error
	// FindByID 根据ID查找视频
	FindByID(ctx context.Context, id uint) (*model.Video, error)
	// FindByIDs 根据ID列表批量查找视频
	FindByIDs(ctx context.Context, ids []uint) ([]*model.Video, error)
	// FindByUserID 查找用户的视频列表
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Video, error)
	// FindByCategoryID 根据分类查找视频
	FindByCategoryID(ctx context.Context, categoryID uint, limit, offset int) ([]*model.Video, error)
	// FindHotVideos 查找热门视频
	FindHotVideos(ctx context.Context, since time.Time, limit, offset int) ([]*model.Video, error)
	// Update 更新视频
	Update(ctx context.Context, video *model.Video) error
	// Delete 删除视频
	Delete(ctx context.Context, id uint) error
	// IncrementPlayCount 增加播放量
	IncrementPlayCount(ctx context.Context, id uint) error
	// IncrementLikeCount 增加点赞数
	IncrementLikeCount(ctx context.Context, id uint, delta int) error
	// IncrementCommentCount 增加评论数
	IncrementCommentCount(ctx context.Context, id uint, delta int) error
	// CountByUserID 统计用户的视频总数
	CountByUserID(ctx context.Context, userID uint) (int64, error)
}

// videoRepositoryImpl 视频数据访问层实现
type videoRepositoryImpl struct {
	db *gorm.DB
}

// NewVideoRepository 创建视频数据访问层实例
func NewVideoRepository(db *gorm.DB) VideoRepository {
	return &videoRepositoryImpl{
		db: db,
	}
}

// Create 创建视频
func (r *videoRepositoryImpl) Create(ctx context.Context, video *model.Video) error {
	logger.Debug("创建视频", zap.Uint("user_id", video.UserID), zap.String("title", video.Title))

	if err := r.db.WithContext(ctx).Create(video).Error; err != nil {
		logger.Error("创建视频失败", zap.Error(err), zap.String("title", video.Title))
		return err
	}

	logger.Info("视频创建成功", zap.Uint("video_id", video.ID), zap.String("title", video.Title))
	return nil
}

// FindByID 根据ID查找视频
func (r *videoRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.Video, error) {
	logger.Debug("查找视频", zap.Uint("video_id", id))

	var video model.Video
	if err := r.db.WithContext(ctx).Preload("User").First(&video, id).Error; err != nil {
		if pkgerrors.IsNotFound(err) {
			logger.Warn("视频不存在", zap.Uint("video_id", id))
		} else {
			logger.Error("查找视频失败", zap.Error(err), zap.Uint("video_id", id))
		}
		return nil, err
	}

	return &video, nil
}

// FindByIDs 根据ID列表批量查找视频
func (r *videoRepositoryImpl) FindByIDs(ctx context.Context, ids []uint) ([]*model.Video, error) {
	logger.Debug("批量查找视频", zap.Int("count", len(ids)))

	var videos []*model.Video
	if err := r.db.WithContext(ctx).Preload("User").Where("id IN ?", ids).Find(&videos).Error; err != nil {
		logger.Error("批量查找视频失败", zap.Error(err))
		return nil, err
	}

	return videos, nil
}

// FindByUserID 查找用户的视频列表
func (r *videoRepositoryImpl) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Video, error) {
	logger.Debug("查找用户视频列表",
		zap.Uint("user_id", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var videos []*model.Video
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, 1).
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&videos).Error; err != nil {
		logger.Error("查找用户视频列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, err
	}

	return videos, nil
}

// FindByCategoryID 根据分类查找视频
func (r *videoRepositoryImpl) FindByCategoryID(ctx context.Context, categoryID uint, limit, offset int) ([]*model.Video, error) {
	logger.Debug("根据分类查找视频",
		zap.Uint("category_id", categoryID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var videos []*model.Video
	if err := r.db.WithContext(ctx).
		Where("category_id = ? AND status = ?", categoryID, 1).
		Order("hot_score DESC").
		Limit(limit).
		Offset(offset).
		Find(&videos).Error; err != nil {
		logger.Error("根据分类查找视频失败", zap.Error(err), zap.Uint("category_id", categoryID))
		return nil, err
	}

	return videos, nil
}

// FindHotVideos 查找热门视频
func (r *videoRepositoryImpl) FindHotVideos(ctx context.Context, since time.Time, limit, offset int) ([]*model.Video, error) {
	logger.Debug("查找热门视频",
		zap.Time("since", since),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var videos []*model.Video
	if err := r.db.WithContext(ctx).
		Where("status = ? AND published_at > ?", 1, since).
		Order("hot_score DESC").
		Limit(limit).
		Offset(offset).
		Find(&videos).Error; err != nil {
		logger.Error("查找热门视频失败", zap.Error(err))
		return nil, err
	}

	return videos, nil
}

// Update 更新视频
func (r *videoRepositoryImpl) Update(ctx context.Context, video *model.Video) error {
	logger.Debug("更新视频", zap.Uint("video_id", video.ID))

	if err := r.db.WithContext(ctx).Save(video).Error; err != nil {
		logger.Error("更新视频失败", zap.Error(err), zap.Uint("video_id", video.ID))
		return err
	}

	logger.Info("视频更新成功", zap.Uint("video_id", video.ID))
	return nil
}

// Delete 删除视频
func (r *videoRepositoryImpl) Delete(ctx context.Context, id uint) error {
	logger.Debug("删除视频", zap.Uint("video_id", id))

	if err := r.db.WithContext(ctx).Delete(&model.Video{}, id).Error; err != nil {
		logger.Error("删除视频失败", zap.Error(err), zap.Uint("video_id", id))
		return err
	}

	logger.Info("视频删除成功", zap.Uint("video_id", id))
	return nil
}

// IncrementPlayCount 增加播放量
func (r *videoRepositoryImpl) IncrementPlayCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&model.Video{}).
		Where("id = ?", id).
		UpdateColumn("play_count", gorm.Expr("play_count + ?", 1)).Error; err != nil {
		logger.Error("增加播放量失败", zap.Error(err), zap.Uint("video_id", id))
		return err
	}

	return nil
}

// IncrementLikeCount 增加点赞数
func (r *videoRepositoryImpl) IncrementLikeCount(ctx context.Context, id uint, delta int) error {
	if err := r.db.WithContext(ctx).Model(&model.Video{}).
		Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error; err != nil {
		logger.Error("更新点赞数失败", zap.Error(err), zap.Uint("video_id", id))
		return err
	}

	return nil
}

// IncrementCommentCount 增加评论数
func (r *videoRepositoryImpl) IncrementCommentCount(ctx context.Context, id uint, delta int) error {
	if err := r.db.WithContext(ctx).Model(&model.Video{}).
		Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error; err != nil {
		logger.Error("更新评论数失败", zap.Error(err), zap.Uint("video_id", id))
		return err
	}

	return nil
}

// CountByUserID 统计用户的视频总数
func (r *videoRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	logger.Debug("统计用户视频总数", zap.Uint("user_id", userID))

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Video{}).
		Where("user_id = ? AND status = ?", userID, 1).
		Count(&count).Error; err != nil {
		logger.Error("统计用户视频总数失败", zap.Error(err), zap.Uint("user_id", userID))
		return 0, err
	}

	return count, nil
}
