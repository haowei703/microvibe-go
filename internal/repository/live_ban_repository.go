package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	"time"

	"gorm.io/gorm"
)

// LiveBanRepository 直播禁言数据访问接口
type LiveBanRepository interface {
	// Create 创建禁言记录
	Create(ctx context.Context, ban *model.LiveBan) error

	// Update 更新禁言记录
	Update(ctx context.Context, ban *model.LiveBan) error

	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint) (*model.LiveBan, error)

	// FindActiveBan 查询用户在直播间的有效禁言记录
	FindActiveBan(ctx context.Context, liveID, userID uint) (*model.LiveBan, error)

	// ListByLiveID 查询直播间的禁言列表
	ListByLiveID(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveBan, int64, error)

	// Delete 删除禁言记录
	Delete(ctx context.Context, id uint) error

	// UnbanUser 解除禁言
	UnbanUser(ctx context.Context, liveID, userID uint) error

	// CheckBanned 检查用户是否被禁言
	CheckBanned(ctx context.Context, liveID, userID uint) (bool, error)
}

type liveBanRepositoryImpl struct {
	db *gorm.DB
}

// NewLiveBanRepository 创建禁言Repository
func NewLiveBanRepository(db *gorm.DB) LiveBanRepository {
	return &liveBanRepositoryImpl{db: db}
}

// Create 创建禁言记录
func (r *liveBanRepositoryImpl) Create(ctx context.Context, ban *model.LiveBan) error {
	return r.db.WithContext(ctx).Create(ban).Error
}

// Update 更新禁言记录（自动清除缓存）
func (r *liveBanRepositoryImpl) Update(ctx context.Context, ban *model.LiveBan) error {
	keys := []string{
		fmt.Sprintf("live:ban:id:%d", ban.ID),
		fmt.Sprintf("live:ban:lu:%d:%d", ban.LiveID, ban.UserID),
	}

	return cache.WithMultiCacheEvict("liveban", keys, func() error {
		return r.db.WithContext(ctx).Save(ban).Error
	})(ctx)
}

// FindByID 根据ID查询
func (r *liveBanRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.LiveBan, error) {
	return cache.WithCache[*model.LiveBan](
		cache.CacheConfig{
			CacheName: "liveban",
			KeyPrefix: "live:ban:id",
			TTL:       5 * time.Minute,
		},
		func() (*model.LiveBan, error) {
			var ban model.LiveBan
			if err := r.db.WithContext(ctx).
				Preload("User").
				Preload("Operator").
				First(&ban, id).Error; err != nil {
				return nil, err
			}
			return &ban, nil
		},
	)(ctx, id)
}

// FindActiveBan 查询用户在直播间的有效禁言记录（使用Redis缓存）
func (r *liveBanRepositoryImpl) FindActiveBan(ctx context.Context, liveID, userID uint) (*model.LiveBan, error) {
	cacheKey := fmt.Sprintf("%d:%d", liveID, userID)

	return cache.WithCache[*model.LiveBan](
		cache.CacheConfig{
			CacheName: "liveban",
			KeyPrefix: "live:ban:lu",
			TTL:       2 * time.Minute, // 禁言信息缓存时间较短，保持实时性
		},
		func() (*model.LiveBan, error) {
			var ban model.LiveBan
			now := time.Now()
			err := r.db.WithContext(ctx).
				Where("live_id = ? AND user_id = ? AND status = ?", liveID, userID, 1).
				Where("expired_at IS NULL OR expired_at > ?", now).
				Order("created_at DESC").
				First(&ban).Error
			if err != nil {
				return nil, err
			}
			return &ban, nil
		},
	)(ctx, cacheKey)
}

// ListByLiveID 查询直播间的禁言列表
func (r *liveBanRepositoryImpl) ListByLiveID(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveBan, int64, error) {
	var bans []*model.LiveBan
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LiveBan{}).Where("live_id = ?", liveID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Preload("Operator").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&bans).Error

	if err != nil {
		return nil, 0, err
	}

	return bans, total, nil
}

// Delete 删除禁言记录（自动清除缓存）
func (r *liveBanRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// 先查询获取禁言信息
	var ban model.LiveBan
	if err := r.db.WithContext(ctx).First(&ban, id).Error; err != nil {
		return err
	}

	keys := []string{
		fmt.Sprintf("live:ban:id:%d", ban.ID),
		fmt.Sprintf("live:ban:lu:%d:%d", ban.LiveID, ban.UserID),
	}

	return cache.WithMultiCacheEvict("liveban", keys, func() error {
		return r.db.WithContext(ctx).Delete(&model.LiveBan{}, id).Error
	})(ctx)
}

// UnbanUser 解除禁言（自动清除缓存）
func (r *liveBanRepositoryImpl) UnbanUser(ctx context.Context, liveID, userID uint) error {
	key := fmt.Sprintf("live:ban:lu:%d:%d", liveID, userID)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveban",
			KeyPrefix: "live:ban:lu",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveBan{}).
				Where("live_id = ? AND user_id = ? AND status = ?", liveID, userID, 1).
				Update("status", 0).Error
		},
	)(ctx, key)
}

// CheckBanned 检查用户是否被禁言（使用Redis缓存）
func (r *liveBanRepositoryImpl) CheckBanned(ctx context.Context, liveID, userID uint) (bool, error) {
	ban, err := r.FindActiveBan(ctx, liveID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// 检查是否过期
	if ban.ExpiredAt != nil && ban.ExpiredAt.Before(time.Now()) {
		// 过期了，自动解除
		_ = r.UnbanUser(ctx, liveID, userID)
		return false, nil
	}

	return ban.Status == 1, nil
}
