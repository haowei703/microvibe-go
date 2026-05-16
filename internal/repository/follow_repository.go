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
	// CountFollowings 统计用户关注数
	CountFollowings(ctx context.Context, userID uint) (int64, error)
	// CountFollowers 统计用户粉丝数
	CountFollowers(ctx context.Context, userID uint) (int64, error)
	// FindFollowingsWithInfo 查找用户关注的高级信息 (带 Join)
	FindFollowingsWithInfo(ctx context.Context, targetUserID, currentUserID uint, limit, offset int) ([]*model.UserFollowVO, error)
	// FindFollowersWithInfo 查找用户粉丝的高级信息 (带 Join)
	FindFollowersWithInfo(ctx context.Context, targetUserID, currentUserID uint, limit, offset int) ([]*model.UserFollowVO, error)
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

// CountFollowings 统计用户关注数
func (r *followRepositoryImpl) CountFollowings(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountFollowers 统计用户粉丝数
func (r *followRepositoryImpl) CountFollowers(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("followed_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindFollowingsWithInfo 查找用户关注的高级信息 (带 Join)
func (r *followRepositoryImpl) FindFollowingsWithInfo(ctx context.Context, targetUserID, currentUserID uint, limit, offset int) ([]*model.UserFollowVO, error) {
	var results []*model.UserFollowVO

	query := r.db.WithContext(ctx).Table("follows").
		Select("users.id, users.username, users.nickname, users.avatar, user_profiles.introduction").
		Joins("INNER JOIN users ON users.id = follows.followed_id"). // 必须有对应的用户
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("follows.user_id = ?", targetUserID)

	if currentUserID > 0 {
		query = query.Select("users.id, users.username, users.nickname, users.avatar, user_profiles.introduction, (f2.user_id IS NOT NULL) AS is_followed").
			Joins("LEFT JOIN follows AS f2 ON f2.user_id = ? AND f2.followed_id = users.id", currentUserID)
	}

	err := query.Order("follows.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&results).Error

	return results, err
}

// FindFollowersWithInfo 查找用户粉丝的高级信息 (带 Join)
func (r *followRepositoryImpl) FindFollowersWithInfo(ctx context.Context, targetUserID, currentUserID uint, limit, offset int) ([]*model.UserFollowVO, error) {
	var results []*model.UserFollowVO

	query := r.db.WithContext(ctx).Table("follows").
		Select("users.id, users.username, users.nickname, users.avatar, user_profiles.introduction").
		Joins("INNER JOIN users ON users.id = follows.user_id"). // 粉丝是 user_id
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("follows.followed_id = ?", targetUserID)

	if currentUserID > 0 {
		query = query.Select("users.id, users.username, users.nickname, users.avatar, user_profiles.introduction, (f2.user_id IS NOT NULL) AS is_followed").
			Joins("LEFT JOIN follows AS f2 ON f2.user_id = ? AND f2.followed_id = users.id", currentUserID)
	}

	err := query.Order("follows.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&results).Error

	return results, err
}
