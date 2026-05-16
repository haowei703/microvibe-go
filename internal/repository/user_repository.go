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
	// FindByIDs 根据ID列表批量查找用户
	FindByIDs(ctx context.Context, ids []uint) ([]*model.User, error)
	// FindByUsername 根据用户名查找用户，useCache=false 时跳过缓存（用于登录校验密码）
	FindByUsername(ctx context.Context, username string, useCache ...bool) (*model.User, error)
	// FindByEmail 根据邮箱查找用户，useCache=false 时跳过缓存
	FindByEmail(ctx context.Context, email string, useCache ...bool) (*model.User, error)
	// Update 更新用户指定字段
	Update(ctx context.Context, user *model.User) error
	// UpdateFields 用 map 更新指定字段，可写入零值
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	// IncrementFollowCount 增加关注数
	IncrementFollowCount(ctx context.Context, id uint, delta int) error
	// IncrementFollowerCount 增加粉丝数
	IncrementFollowerCount(ctx context.Context, id uint, delta int) error
	// List 分页获取所有用户
	List(ctx context.Context, page, pageSize int, query string) ([]*model.User, int64, error)
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

// List 分页获取所有用户
func (r *userRepositoryImpl) List(ctx context.Context, page, pageSize int, query string) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64
	db := r.db.WithContext(ctx).Model(&model.User{})
	if query != "" {
		q := "%" + query + "%"
		db = db.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", q, q, q)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Limit(pageSize).Offset((page - 1) * pageSize).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

// FindByIDs 根据ID列表批量查找用户
func (r *userRepositoryImpl) FindByIDs(ctx context.Context, ids []uint) ([]*model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []*model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		logger.Error("批量查找用户失败", zap.Error(err))
		return nil, err
	}
	return users, nil
}

// FindByID 根据ID查找用户（自动缓存）
func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.User, error) {
	logger.Debug("查找用户", zap.Uint("user_id", id))

	// 使用装饰器自动管理缓存
	return cache.WithCache(
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

// FindByUsername 根据用户名查找用户
func (r *userRepositoryImpl) FindByUsername(ctx context.Context, username string, useCache ...bool) (*model.User, error) {
	logger.Debug("根据用户名查找用户", zap.String("username", username))

	loader := func() (*model.User, error) {
		var user model.User
		if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
			if !pkgerrors.IsNotFound(err) {
				logger.Error("查找用户失败", zap.Error(err), zap.String("username", username))
			}
			return nil, err
		}
		return &user, nil
	}

	if len(useCache) > 0 && !useCache[0] {
		return loader()
	}

	return cache.WithCache(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:username",
			TTL:       10 * time.Minute,
		},
		loader,
	)(ctx, username)
}

// FindByEmail 根据邮箱查找用户
func (r *userRepositoryImpl) FindByEmail(ctx context.Context, email string, useCache ...bool) (*model.User, error) {
	logger.Debug("根据邮箱查找用户", zap.String("email", email))

	loader := func() (*model.User, error) {
		var user model.User
		if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
			if !pkgerrors.IsNotFound(err) {
				logger.Error("查找用户失败", zap.Error(err), zap.String("email", email))
			}
			return nil, err
		}
		return &user, nil
	}

	if len(useCache) > 0 && !useCache[0] {
		return loader()
	}

	return cache.WithCache(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:email",
			TTL:       10 * time.Minute,
		},
		loader,
	)(ctx, email)
}

// Update 更新用户指定字段
func (r *userRepositoryImpl) Update(ctx context.Context, user *model.User) error {
	logger.Debug("更新用户信息", zap.Uint("user_id", user.ID))

	// 使用装饰器自动清除相关缓存
	keys := []string{
		fmt.Sprintf("user:id:%d", user.ID),
		fmt.Sprintf("user:username:%s", user.Username),
		fmt.Sprintf("user:email:%s", user.Email),
	}

	return cache.WithMultiCacheEvict("user", keys, func() error {
		if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).Updates(user).Error; err != nil {
			logger.Error("更新用户失败", zap.Error(err), zap.Uint("user_id", user.ID))
			return err
		}
		logger.Info("用户更新成功", zap.Uint("user_id", user.ID))
		return nil
	})(ctx)
}

// UpdateFields 通过 map 更新字段，确保零值（false/0/""）也能写入
func (r *userRepositoryImpl) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	logger.Debug("按字段更新用户", zap.Uint("user_id", id), zap.Int("field_count", len(fields)))

	// 取一次用户用于清缓存（username/email 字段构造 key）
	var u model.User
	if err := r.db.WithContext(ctx).Select("id", "username", "email").First(&u, id).Error; err != nil {
		logger.Error("查找用户失败", zap.Error(err), zap.Uint("user_id", id))
		return err
	}

	keys := []string{
		fmt.Sprintf("user:id:%d", u.ID),
		fmt.Sprintf("user:username:%s", u.Username),
		fmt.Sprintf("user:email:%s", u.Email),
	}

	return cache.WithMultiCacheEvict("user", keys, func() error {
		if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(fields).Error; err != nil {
			logger.Error("按字段更新用户失败", zap.Error(err), zap.Uint("user_id", id))
			return err
		}
		logger.Info("按字段更新用户成功", zap.Uint("user_id", id))
		return nil
	})(ctx)
}

// IncrementFollowCount 增加关注数（并清除用户缓存）
func (r *userRepositoryImpl) IncrementFollowCount(ctx context.Context, id uint, delta int) error {
	logger.Debug("更新关注数", zap.Uint("user_id", id), zap.Int("delta", delta))

	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
		},
		func() error {
			if err := r.db.WithContext(ctx).Model(&model.User{}).
				Where("id = ?", id).
				UpdateColumn("follow_count", gorm.Expr("follow_count + ?", delta)).Error; err != nil {
				logger.Error("更新关注数失败", zap.Error(err), zap.Uint("user_id", id))
				return err
			}
			return nil
		},
	)(ctx, id)
}

// IncrementFollowerCount 增加粉丝数（并清除用户缓存）
func (r *userRepositoryImpl) IncrementFollowerCount(ctx context.Context, id uint, delta int) error {
	logger.Debug("更新粉丝数", zap.Uint("user_id", id), zap.Int("delta", delta))

	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "user",
			KeyPrefix: "user:id",
		},
		func() error {
			if err := r.db.WithContext(ctx).Model(&model.User{}).
				Where("id = ?", id).
				UpdateColumn("follower_count", gorm.Expr("follower_count + ?", delta)).Error; err != nil {
				logger.Error("更新粉丝数失败", zap.Error(err), zap.Uint("user_id", id))
				return err
			}
			return nil
		},
	)(ctx, id)
}
