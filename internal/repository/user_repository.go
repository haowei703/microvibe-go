package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserRepository 用户数据访问层接口
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *model.User) error
	// FindByID 根据ID查找用户
	FindByID(ctx context.Context, id uint) (*model.User, error)
	// FindByUsername 根据用户名查找用户
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	// FindByEmail 根据邮箱查找用户
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	// Update 更新用户信息
	Update(ctx context.Context, user *model.User) error
	// UpdateFields 更新指定字段
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	// IncrementFollowCount 增加关注数
	IncrementFollowCount(ctx context.Context, id uint, delta int) error
	// IncrementFollowerCount 增加粉丝数
	IncrementFollowerCount(ctx context.Context, id uint, delta int) error
}

// userRepositoryImpl 用户数据访问层实现
type userRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问层实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{
		db: db,
	}
}

// Create 创建用户
func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
	logger.Debug("创建用户", zap.String("username", user.Username))

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		logger.Error("创建用户失败", zap.Error(err), zap.String("username", user.Username))
		return err
	}

	logger.Info("用户创建成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	return nil
}

// FindByID 根据ID查找用户（自动缓存）
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
	logger.Debug("查找用户", zap.Uint("user_id", id))

	// 使用装饰器自动管理缓存
	return cache.WithCache[*model.User](
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
			TTL:       10 * time.Minute,
		},
		func() (*model.User, error) {
			// 实际的数据库查询逻辑
			var user model.User
			if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
				if pkgerrors.IsNotFound(err) {
					logger.Warn("用户不存在", zap.Uint("user_id", id))
				} else {
					logger.Error("查找用户失败", zap.Error(err), zap.Uint("user_id", id))
				}
				return nil, err
			}
			return &user, nil
		},
	)(ctx, id)
}

// FindByUsername 根据用户名查找用户（自动缓存）
func (r *userRepositoryImpl) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	logger.Debug("根据用户名查找用户", zap.String("username", username))

	// 使用装饰器自动管理缓存
	return cache.WithCache[*model.User](
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:username",
			TTL:       10 * time.Minute,
		},
		func() (*model.User, error) {
			// 实际的数据库查询逻辑
			var user model.User
			if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
				if !pkgerrors.IsNotFound(err) {
					logger.Error("查找用户失败", zap.Error(err), zap.String("username", username))
				}
				return nil, err
			}
			return &user, nil
		},
	)(ctx, username)
}

// FindByEmail 根据邮箱查找用户（自动缓存）
func (r *userRepositoryImpl) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	logger.Debug("根据邮箱查找用户", zap.String("email", email))

	// 使用装饰器自动管理缓存
	return cache.WithCache[*model.User](
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:email",
			TTL:       10 * time.Minute,
		},
		func() (*model.User, error) {
			// 实际的数据库查询逻辑
			var user model.User
			if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
				if !pkgerrors.IsNotFound(err) {
					logger.Error("查找用户失败", zap.Error(err), zap.String("email", email))
				}
				return nil, err
			}
			return &user, nil
		},
	)(ctx, email)
}

// Update 更新用户信息（自动清除缓存）
func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
	logger.Debug("更新用户信息", zap.Uint("user_id", user.ID))

	// 使用装饰器自动清除相关缓存
	keys := []string{
		fmt.Sprintf("user:id:%d", user.ID),
		fmt.Sprintf("user:username:%s", user.Username),
		fmt.Sprintf("user:email:%s", user.Email),
	}

	return cache.WithMultiCacheEvict("user", keys, func() error {
		// 实际的数据库更新逻辑
		if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
			logger.Error("更新用户失败", zap.Error(err), zap.Uint("user_id", user.ID))
			return err
		}
		logger.Info("用户更新成功", zap.Uint("user_id", user.ID))
		return nil
	})(ctx)
}

// UpdateFields 更新指定字段（自动清除缓存）
func (r *userRepositoryImpl) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	logger.Debug("更新用户字段", zap.Uint("user_id", id), zap.Any("fields", fields))

	// 使用装饰器自动清除ID相关的缓存
	// 注意：由于不知道username和email，这里只清除ID缓存
	// 如果更新了username或email，建议使用Update方法而非UpdateFields
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
		},
		func() error {
			// 实际的数据库更新逻辑
			if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(fields).Error; err != nil {
				logger.Error("更新用户字段失败", zap.Error(err), zap.Uint("user_id", id))
				return err
			}
			logger.Info("用户字段更新成功", zap.Uint("user_id", id))
			return nil
		},
	)(ctx, id)
}

// IncrementFollowCount 增加关注数
func (r *userRepositoryImpl) IncrementFollowCount(ctx context.Context, id uint, delta int) error {
	logger.Debug("更新关注数", zap.Uint("user_id", id), zap.Int("delta", delta))

	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("follow_count", gorm.Expr("follow_count + ?", delta)).Error; err != nil {
		logger.Error("更新关注数失败", zap.Error(err), zap.Uint("user_id", id))
		return err
	}

	return nil
}

// IncrementFollowerCount 增加粉丝数
func (r *userRepositoryImpl) IncrementFollowerCount(ctx context.Context, id uint, delta int) error {
	logger.Debug("更新粉丝数", zap.Uint("user_id", id), zap.Int("delta", delta))

	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("follower_count", gorm.Expr("follower_count + ?", delta)).Error; err != nil {
		logger.Error("更新粉丝数失败", zap.Error(err), zap.Uint("user_id", id))
		return err
	}

	return nil
}
