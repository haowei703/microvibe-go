package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FavoriteRepository 收藏数据访问层接口
type FavoriteRepository interface {
	// Create 创建收藏记录
	Create(ctx context.Context, favorite *model.Favorite) error
	// Delete 删除收藏记录
	Delete(ctx context.Context, userID, videoID uint) error
	// Exists 检查是否已收藏
	Exists(ctx context.Context, userID, videoID uint) (bool, error)
	// FindByUserID 查找用户的收藏列表
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Favorite, error)
}

// favoriteRepositoryImpl 收藏数据访问层实现
type favoriteRepositoryImpl struct {
	db *gorm.DB
}

// NewFavoriteRepository 创建收藏数据访问层实例
func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepositoryImpl{
		db: db,
	}
}

// Create 创建收藏记录
func (r *favoriteRepositoryImpl) Create(ctx context.Context, favorite *model.Favorite) error {
	logger.Debug("创建收藏记录", zap.Uint("user_id", favorite.UserID), zap.Uint("video_id", favorite.VideoID))

	if err := r.db.WithContext(ctx).Create(favorite).Error; err != nil {
		logger.Error("创建收藏记录失败", zap.Error(err))
		return err
	}

	logger.Info("收藏记录创建成功", zap.Uint("favorite_id", favorite.ID))
	return nil
}

// Delete 删除收藏记录
func (r *favoriteRepositoryImpl) Delete(ctx context.Context, userID, videoID uint) error {
	logger.Debug("删除收藏记录", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Delete(&model.Favorite{}).Error; err != nil {
		logger.Error("删除收藏记录失败", zap.Error(err))
		return err
	}

	logger.Info("收藏记录删除成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// Exists 检查是否已收藏
func (r *favoriteRepositoryImpl) Exists(ctx context.Context, userID, videoID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error; err != nil {
		logger.Error("检查收藏状态失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// FindByUserID 查找用户的收藏列表
func (r *favoriteRepositoryImpl) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Favorite, error) {
	logger.Debug("查找用户收藏列表", zap.Uint("user_id", userID))

	var favorites []*model.Favorite
	if err := r.db.WithContext(ctx).
		Preload("Video").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&favorites).Error; err != nil {
		logger.Error("查找用户收藏列表失败", zap.Error(err))
		return nil, err
	}

	return favorites, nil
}
