package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FollowRepository 关注数据访问层接口
type FollowRepository interface {
	// Create 创建关注关系
	Create(ctx context.Context, follow *model.Follow) error
	// Delete 删除关注关系
	Delete(ctx context.Context, userID, followedID uint) error
	// Exists 检查关注关系是否存在
	Exists(ctx context.Context, userID, followedID uint) (bool, error)
	// FindFollowings 查找用户关注的人
	FindFollowings(ctx context.Context, userID uint, limit, offset int) ([]*model.Follow, error)
	// FindFollowers 查找用户的粉丝
	FindFollowers(ctx context.Context, userID uint, limit, offset int) ([]*model.Follow, error)
}

// followRepositoryImpl 关注数据访问层实现
type followRepositoryImpl struct {
	db *gorm.DB
}

// NewFollowRepository 创建关注数据访问层实例
func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepositoryImpl{
		db: db,
	}
}

// Create 创建关注关系
func (r *followRepositoryImpl) Create(ctx context.Context, follow *model.Follow) error {
	logger.Debug("创建关注关系",
		zap.Uint("user_id", follow.UserID),
		zap.Uint("followed_id", follow.FollowedID))

	if err := r.db.WithContext(ctx).Create(follow).Error; err != nil {
		logger.Error("创建关注关系失败",
			zap.Error(err),
			zap.Uint("user_id", follow.UserID),
			zap.Uint("followed_id", follow.FollowedID))
		return err
	}

	logger.Info("关注成功",
		zap.Uint("user_id", follow.UserID),
		zap.Uint("followed_id", follow.FollowedID))
	return nil
}

// Delete 删除关注关系
func (r *followRepositoryImpl) Delete(ctx context.Context, userID, followedID uint) error {
	logger.Debug("删除关注关系",
		zap.Uint("user_id", userID),
		zap.Uint("followed_id", followedID))

	result := r.db.WithContext(ctx).
		Where("user_id = ? AND followed_id = ?", userID, followedID).
		Delete(&model.Follow{})

	if result.Error != nil {
		logger.Error("删除关注关系失败",
			zap.Error(result.Error),
			zap.Uint("user_id", userID),
			zap.Uint("followed_id", followedID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warn("关注关系不存在",
			zap.Uint("user_id", userID),
			zap.Uint("followed_id", followedID))
		return gorm.ErrRecordNotFound
	}

	logger.Info("取消关注成功",
		zap.Uint("user_id", userID),
		zap.Uint("followed_id", followedID))
	return nil
}

// Exists 检查关注关系是否存在
func (r *followRepositoryImpl) Exists(ctx context.Context, userID, followedID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Follow{}).
		Where("user_id = ? AND followed_id = ?", userID, followedID).
		Count(&count).Error; err != nil {
		logger.Error("检查关注关系失败",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("followed_id", followedID))
		return false, err
	}

	return count > 0, nil
}

// FindFollowings 查找用户关注的人
func (r *followRepositoryImpl) FindFollowings(ctx context.Context, userID uint, limit, offset int) ([]*model.Follow, error) {
	logger.Debug("查找用户关注列表",
		zap.Uint("user_id", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var follows []*model.Follow
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&follows).Error; err != nil {
		logger.Error("查找关注列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, err
	}

	return follows, nil
}

// FindFollowers 查找用户的粉丝
func (r *followRepositoryImpl) FindFollowers(ctx context.Context, userID uint, limit, offset int) ([]*model.Follow, error) {
	logger.Debug("查找用户粉丝列表",
		zap.Uint("user_id", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	var follows []*model.Follow
	if err := r.db.WithContext(ctx).
		Where("followed_id = ?", userID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&follows).Error; err != nil {
		logger.Error("查找粉丝列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, err
	}

	return follows, nil
}
