package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	"time"

	"gorm.io/gorm"
)

// LiveFansClubRepository 粉丝团数据访问接口
type LiveFansClubRepository interface {
	// Create 创建粉丝团成员
	Create(ctx context.Context, member *model.LiveFansClub) error

	// Update 更新粉丝团成员
	Update(ctx context.Context, member *model.LiveFansClub) error

	// FindByID 根据ID查询
	FindByID(ctx context.Context, id uint) (*model.LiveFansClub, error)

	// FindByLiveAndUser 根据直播间和用户查询
	FindByLiveAndUser(ctx context.Context, liveID, userID uint) (*model.LiveFansClub, error)

	// ListByLiveID 查询直播间的粉丝团列表
	ListByLiveID(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveFansClub, int64, error)

	// Delete 删除粉丝团成员
	Delete(ctx context.Context, id uint) error

	// AddExperience 增加经验值
	AddExperience(ctx context.Context, id uint, exp int64) error

	// UpdateLevel 更新等级
	UpdateLevel(ctx context.Context, id uint, level int) error

	// GetTopMembers 获取粉丝团排行榜
	GetTopMembers(ctx context.Context, liveID uint, limit int) ([]*model.LiveFansClub, error)

	// CountMembers 统计粉丝团人数
	CountMembers(ctx context.Context, liveID uint) (int64, error)
}

type liveFansClubRepositoryImpl struct {
	db *gorm.DB
}

// NewLiveFansClubRepository 创建粉丝团Repository
func NewLiveFansClubRepository(db *gorm.DB) LiveFansClubRepository {
	return &liveFansClubRepositoryImpl{db: db}
}

// Create 创建粉丝团成员
func (r *liveFansClubRepositoryImpl) Create(ctx context.Context, member *model.LiveFansClub) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// Update 更新粉丝团成员（自动清除缓存）
func (r *liveFansClubRepositoryImpl) Update(ctx context.Context, member *model.LiveFansClub) error {
	keys := []string{
		fmt.Sprintf("live:fans:id:%d", member.ID),
		fmt.Sprintf("live:fans:lu:%d:%d", member.LiveID, member.UserID),
	}

	return cache.WithMultiCacheEvict("livefans", keys, func() error {
		return r.db.WithContext(ctx).Save(member).Error
	})(ctx)
}

// FindByID 根据ID查询（使用Redis缓存）
func (r *liveFansClubRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.LiveFansClub, error) {
	return cache.WithCache[*model.LiveFansClub](
		cache.CacheConfig{
			CacheName: "livefans",
			KeyPrefix: "live:fans:id",
			TTL:       10 * time.Minute,
		},
		func() (*model.LiveFansClub, error) {
			var member model.LiveFansClub
			if err := r.db.WithContext(ctx).Preload("User").First(&member, id).Error; err != nil {
				return nil, err
			}
			return &member, nil
		},
	)(ctx, id)
}

// FindByLiveAndUser 根据直播间和用户查询（使用Redis缓存）
func (r *liveFansClubRepositoryImpl) FindByLiveAndUser(ctx context.Context, liveID, userID uint) (*model.LiveFansClub, error) {
	cacheKey := fmt.Sprintf("%d:%d", liveID, userID)

	return cache.WithCache[*model.LiveFansClub](
		cache.CacheConfig{
			CacheName: "livefans",
			KeyPrefix: "live:fans:lu",
			TTL:       10 * time.Minute,
		},
		func() (*model.LiveFansClub, error) {
			var member model.LiveFansClub
			err := r.db.WithContext(ctx).
				Preload("User").
				Where("live_id = ? AND user_id = ?", liveID, userID).
				First(&member).Error
			if err != nil {
				return nil, err
			}
			return &member, nil
		},
	)(ctx, cacheKey)
}

// ListByLiveID 查询直播间的粉丝团列表
func (r *liveFansClubRepositoryImpl) ListByLiveID(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveFansClub, int64, error) {
	var members []*model.LiveFansClub
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LiveFansClub{}).Where("live_id = ?", liveID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Order("level DESC, experience DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&members).Error

	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// Delete 删除粉丝团成员（自动清除缓存）
func (r *liveFansClubRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// 先查询获取成员信息
	var member model.LiveFansClub
	if err := r.db.WithContext(ctx).First(&member, id).Error; err != nil {
		return err
	}

	keys := []string{
		fmt.Sprintf("live:fans:id:%d", member.ID),
		fmt.Sprintf("live:fans:lu:%d:%d", member.LiveID, member.UserID),
	}

	return cache.WithMultiCacheEvict("livefans", keys, func() error {
		return r.db.WithContext(ctx).Delete(&model.LiveFansClub{}, id).Error
	})(ctx)
}

// AddExperience 增加经验值
func (r *liveFansClubRepositoryImpl) AddExperience(ctx context.Context, id uint, exp int64) error {
	key := fmt.Sprintf("live:fans:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livefans",
			KeyPrefix: "live:fans:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveFansClub{}).
				Where("id = ?", id).
				UpdateColumn("experience", gorm.Expr("experience + ?", exp)).Error
		},
	)(ctx, key)
}

// UpdateLevel 更新等级
func (r *liveFansClubRepositoryImpl) UpdateLevel(ctx context.Context, id uint, level int) error {
	key := fmt.Sprintf("live:fans:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livefans",
			KeyPrefix: "live:fans:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveFansClub{}).
				Where("id = ?", id).
				Update("level", level).Error
		},
	)(ctx, key)
}

// GetTopMembers 获取粉丝团排行榜（使用Redis缓存）
func (r *liveFansClubRepositoryImpl) GetTopMembers(ctx context.Context, liveID uint, limit int) ([]*model.LiveFansClub, error) {
	return cache.WithCache[[]*model.LiveFansClub](
		cache.CacheConfig{
			CacheName: "livefans",
			KeyPrefix: "live:fans:top",
			TTL:       5 * time.Minute,
		},
		func() ([]*model.LiveFansClub, error) {
			var members []*model.LiveFansClub
			err := r.db.WithContext(ctx).
				Where("live_id = ? AND is_activated = ?", liveID, true).
				Preload("User").
				Order("level DESC, experience DESC").
				Limit(limit).
				Find(&members).Error
			if err != nil {
				return nil, err
			}
			return members, nil
		},
	)(ctx, liveID)
}

// CountMembers 统计粉丝团人数
func (r *liveFansClubRepositoryImpl) CountMembers(ctx context.Context, liveID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.LiveFansClub{}).
		Where("live_id = ? AND is_activated = ?", liveID, true).
		Count(&count).Error
	return count, err
}
