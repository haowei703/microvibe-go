package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProfileRepository 用户资料数据访问层接口
type ProfileRepository interface {
	// Create 创建用户资料
	Create(ctx context.Context, profile *model.UserProfile) error
	// FindByUserID 根据用户ID查找资料
	FindByUserID(ctx context.Context, userID uint) (*model.UserProfile, error)
	// Update 更新用户资料
	Update(ctx context.Context, profile *model.UserProfile) error
	// Delete 删除用户资料
	Delete(ctx context.Context, userID uint) error
}

// profileRepositoryImpl 用户资料数据访问层实现
type profileRepositoryImpl struct {
	db *gorm.DB
}

// NewProfileRepository 创建用户资料数据访问层实例
func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepositoryImpl{
		db: db,
	}
}

// Create 创建用户资料
func (r *profileRepositoryImpl) Create(ctx context.Context, profile *model.UserProfile) error {
	logger.Debug("创建用户资料", zap.Uint("user_id", profile.UserID))

	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		logger.Error("创建用户资料失败", zap.Error(err), zap.Uint("user_id", profile.UserID))
		return err
	}

	logger.Info("用户资料创建成功", zap.Uint("profile_id", profile.ID), zap.Uint("user_id", profile.UserID))
	return nil
}

// FindByUserID 根据用户ID查找资料
func (r *profileRepositoryImpl) FindByUserID(ctx context.Context, userID uint) (*model.UserProfile, error) {
	logger.Debug("查找用户资料", zap.Uint("user_id", userID))

	var profile model.UserProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("用户资料不存在", zap.Uint("user_id", userID))
		} else {
			logger.Error("查找用户资料失败", zap.Error(err), zap.Uint("user_id", userID))
		}
		return nil, err
	}

	return &profile, nil
}

// Update 更新用户资料
func (r *profileRepositoryImpl) Update(ctx context.Context, profile *model.UserProfile) error {
	logger.Debug("更新用户资料", zap.Uint("user_id", profile.UserID))

	if err := r.db.WithContext(ctx).Save(profile).Error; err != nil {
		logger.Error("更新用户资料失败", zap.Error(err), zap.Uint("user_id", profile.UserID))
		return err
	}

	logger.Info("用户资料更新成功", zap.Uint("user_id", profile.UserID))
	return nil
}

// Delete 删除用户资料
func (r *profileRepositoryImpl) Delete(ctx context.Context, userID uint) error {
	logger.Debug("删除用户资料", zap.Uint("user_id", userID))

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.UserProfile{}).Error; err != nil {
		logger.Error("删除用户资料失败", zap.Error(err), zap.Uint("user_id", userID))
		return err
	}

	logger.Info("用户资料删除成功", zap.Uint("user_id", userID))
	return nil
}
