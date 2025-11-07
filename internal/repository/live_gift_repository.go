package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	"time"

	"gorm.io/gorm"
)

// LiveGiftRepository 直播礼物数据访问接口
type LiveGiftRepository interface {
	// Create 创建礼物
	Create(ctx context.Context, gift *model.LiveGift) error

	// Update 更新礼物
	Update(ctx context.Context, gift *model.LiveGift) error

	// FindByID 根据ID查询礼物
	FindByID(ctx context.Context, id uint) (*model.LiveGift, error)

	// List 查询礼物列表
	List(ctx context.Context, giftType int8, status int8) ([]*model.LiveGift, error)

	// Delete 删除礼物
	Delete(ctx context.Context, id uint) error

	// CreateGiftRecord 创建送礼记录
	CreateGiftRecord(ctx context.Context, record *model.LiveGiftRecord) error

	// ListGiftRecords 查询送礼记录列表
	ListGiftRecords(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveGiftRecord, int64, error)

	// GetUserGiftStats 获取用户在直播间的送礼统计
	GetUserGiftStats(ctx context.Context, liveID, userID uint) (int64, int, error)

	// GetTopGivers 获取直播间送礼榜单
	GetTopGivers(ctx context.Context, liveID uint, limit int) ([]*model.LiveGiftRecord, error)
}

type liveGiftRepositoryImpl struct {
	db *gorm.DB
}

// NewLiveGiftRepository 创建礼物Repository
func NewLiveGiftRepository(db *gorm.DB) LiveGiftRepository {
	return &liveGiftRepositoryImpl{db: db}
}

// Create 创建礼物
func (r *liveGiftRepositoryImpl) Create(ctx context.Context, gift *model.LiveGift) error {
	return r.db.WithContext(ctx).Create(gift).Error
}

// Update 更新礼物（自动清除缓存）
func (r *liveGiftRepositoryImpl) Update(ctx context.Context, gift *model.LiveGift) error {
	key := fmt.Sprintf("live:gift:id:%d", gift.ID)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livegift",
			KeyPrefix: "live:gift:id",
		},
		func() error {
			return r.db.WithContext(ctx).Save(gift).Error
		},
	)(ctx, key)
}

// FindByID 根据ID查询礼物（使用Redis缓存）
func (r *liveGiftRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.LiveGift, error) {
	return cache.WithCache[*model.LiveGift](
		cache.CacheConfig{
			CacheName: "livegift",
			KeyPrefix: "live:gift:id",
			TTL:       30 * time.Minute, // 礼物信息不常变化，缓存30分钟
		},
		func() (*model.LiveGift, error) {
			var gift model.LiveGift
			if err := r.db.WithContext(ctx).First(&gift, id).Error; err != nil {
				return nil, err
			}
			return &gift, nil
		},
	)(ctx, id)
}

// List 查询礼物列表（使用Redis缓存）
func (r *liveGiftRepositoryImpl) List(ctx context.Context, giftType int8, status int8) ([]*model.LiveGift, error) {
	// 根据不同的筛选条件生成不同的缓存键
	cacheKey := fmt.Sprintf("type:%d:status:%d", giftType, status)

	return cache.WithCache[[]*model.LiveGift](
		cache.CacheConfig{
			CacheName: "livegift",
			KeyPrefix: "live:gift:list",
			TTL:       15 * time.Minute,
		},
		func() ([]*model.LiveGift, error) {
			var gifts []*model.LiveGift
			query := r.db.WithContext(ctx).Model(&model.LiveGift{})

			// 筛选条件
			if giftType > 0 {
				query = query.Where("type = ?", giftType)
			}
			if status >= 0 {
				query = query.Where("status = ?", status)
			}

			err := query.Order("sort ASC, id ASC").Find(&gifts).Error
			if err != nil {
				return nil, err
			}
			return gifts, nil
		},
	)(ctx, cacheKey)
}

// Delete 删除礼物（自动清除缓存）
func (r *liveGiftRepositoryImpl) Delete(ctx context.Context, id uint) error {
	key := fmt.Sprintf("live:gift:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livegift",
			KeyPrefix: "live:gift:id",
		},
		func() error {
			return r.db.WithContext(ctx).Delete(&model.LiveGift{}, id).Error
		},
	)(ctx, key)
}

// CreateGiftRecord 创建送礼记录
func (r *liveGiftRepositoryImpl) CreateGiftRecord(ctx context.Context, record *model.LiveGiftRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// ListGiftRecords 查询送礼记录列表
func (r *liveGiftRepositoryImpl) ListGiftRecords(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveGiftRecord, int64, error) {
	var records []*model.LiveGiftRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LiveGiftRecord{}).Where("live_id = ?", liveID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Preload("Gift").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records).Error

	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetUserGiftStats 获取用户在直播间的送礼统计
func (r *liveGiftRepositoryImpl) GetUserGiftStats(ctx context.Context, liveID, userID uint) (int64, int, error) {
	var result struct {
		TotalValue int64
		GiftCount  int
	}

	err := r.db.WithContext(ctx).
		Model(&model.LiveGiftRecord{}).
		Select("COALESCE(SUM(total_value), 0) as total_value, COALESCE(COUNT(*), 0) as gift_count").
		Where("live_id = ? AND user_id = ?", liveID, userID).
		Scan(&result).Error

	if err != nil {
		return 0, 0, err
	}

	return result.TotalValue, result.GiftCount, nil
}

// GetTopGivers 获取直播间送礼榜单（使用Redis缓存）
func (r *liveGiftRepositoryImpl) GetTopGivers(ctx context.Context, liveID uint, limit int) ([]*model.LiveGiftRecord, error) {
	// 使用缓存，实时性要求较高，缓存时间设置为1分钟
	return cache.WithCache[[]*model.LiveGiftRecord](
		cache.CacheConfig{
			CacheName: "livegift",
			KeyPrefix: "live:gift:top",
			TTL:       1 * time.Minute,
		},
		func() ([]*model.LiveGiftRecord, error) {
			var records []*model.LiveGiftRecord

			// 使用子查询聚合每个用户的送礼总值
			err := r.db.WithContext(ctx).
				Table("live_gift_records").
				Select("user_id, live_id, SUM(total_value) as total_value, COUNT(*) as gift_count").
				Where("live_id = ?", liveID).
				Group("user_id, live_id").
				Order("total_value DESC").
				Limit(limit).
				Find(&records).Error

			if err != nil {
				return nil, err
			}

			// 预加载用户信息
			if len(records) > 0 {
				userIDs := make([]uint, len(records))
				for i, r := range records {
					userIDs[i] = r.UserID
				}

				var users []*model.User
				r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)

				// 构建用户映射
				userMap := make(map[uint]*model.User)
				for _, u := range users {
					userMap[u.ID] = u
				}

				// 填充用户信息
				for i := range records {
					if user, ok := userMap[records[i].UserID]; ok {
						records[i].User = user
					}
				}
			}

			return records, nil
		},
	)(ctx, liveID)
}
