package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LikeRepository 点赞数据访问层接口
type LikeRepository interface {
	// Create 创建点赞记录
	Create(ctx context.Context, like *model.Like) error
	// Delete 删除点赞记录
	Delete(ctx context.Context, userID, videoID uint) error
	// Exists 检查是否已点赞
	Exists(ctx context.Context, userID, videoID uint) (bool, error)
	// FindByUserID 查找用户的点赞列表
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Like, error)
	// FindByVideoID 查找视频的点赞列表
	FindByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Like, error)
}

// likeRepositoryImpl 点赞数据访问层实现
type likeRepositoryImpl struct {
	db *gorm.DB
}

// NewLikeRepository 创建点赞数据访问层实例
func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepositoryImpl{
		db: db,
	}
}

// Create 创建点赞记录
func (r *likeRepositoryImpl) Create(ctx context.Context, like *model.Like) error {
	logger.Debug("创建点赞记录", zap.Uint("user_id", like.UserID), zap.Uint("video_id", like.VideoID))

	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		logger.Error("创建点赞记录失败", zap.Error(err))
		return err
	}

	logger.Info("点赞记录创建成功", zap.Uint("like_id", like.ID))
	return nil
}

// Delete 删除点赞记录
func (r *likeRepositoryImpl) Delete(ctx context.Context, userID, videoID uint) error {
	logger.Debug("删除点赞记录", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Delete(&model.Like{}).Error; err != nil {
		logger.Error("删除点赞记录失败", zap.Error(err))
		return err
	}

	logger.Info("点赞记录删除成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// Exists 检查是否已点赞
func (r *likeRepositoryImpl) Exists(ctx context.Context, userID, videoID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Like{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error; err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// FindByUserID 查找用户的点赞列表
func (r *likeRepositoryImpl) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Like, error) {
	logger.Debug("查找用户点赞列表", zap.Uint("user_id", userID))

	var likes []*model.Like
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error; err != nil {
		logger.Error("查找用户点赞列表失败", zap.Error(err))
		return nil, err
	}

	return likes, nil
}

// FindByVideoID 查找视频的点赞列表
func (r *likeRepositoryImpl) FindByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Like, error) {
	logger.Debug("查找视频点赞列表", zap.Uint("video_id", videoID))

	var likes []*model.Like
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("video_id = ?", videoID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error; err != nil {
		logger.Error("查找视频点赞列表失败", zap.Error(err))
		return nil, err
	}

	return likes, nil
}
