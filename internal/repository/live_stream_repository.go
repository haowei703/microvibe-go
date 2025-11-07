package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	"time"

	"gorm.io/gorm"
)

// LiveStreamRepository 直播间数据访问接口
type LiveStreamRepository interface {
	// Create 创建直播间
	Create(ctx context.Context, liveStream *model.LiveStream) error

	// Update 更新直播间信息
	Update(ctx context.Context, liveStream *model.LiveStream) error

	// FindByID 根据ID查询直播间
	FindByID(ctx context.Context, id uint) (*model.LiveStream, error)

	// FindByRoomID 根据房间ID查询直播间
	FindByRoomID(ctx context.Context, roomID string) (*model.LiveStream, error)

	// FindByStreamKey 根据推流密钥查询直播间
	FindByStreamKey(ctx context.Context, streamKey string) (*model.LiveStream, error)

	// FindByOwnerID 查询用户的直播间
	FindByOwnerID(ctx context.Context, ownerID uint) (*model.LiveStream, error)

	// List 分页查询直播间列表
	List(ctx context.Context, status string, page, pageSize int) ([]*model.LiveStream, int64, error)

	// UpdateViewCount 更新观看人数
	UpdateViewCount(ctx context.Context, id uint, count int64) error

	// UpdateOnlineCount 更新在线人数
	UpdateOnlineCount(ctx context.Context, id uint, count int) error

	// IncrementLikeCount 增加点赞数
	IncrementLikeCount(ctx context.Context, id uint, count int64) error

	// Delete 删除直播间
	Delete(ctx context.Context, id uint) error

	// UpdateGiftStats 更新礼物统计
	UpdateGiftStats(ctx context.Context, id uint, giftCount, giftValue int64) error

	// UpdateProductStats 更新商品统计
	UpdateProductStats(ctx context.Context, id uint, productSales int64) error

	// IncrementCommentCount 增加评论数
	IncrementCommentCount(ctx context.Context, id uint, count int64) error

	// IncrementShareCount 增加分享数
	IncrementShareCount(ctx context.Context, id uint, count int64) error

	// UpdatePeakCount 更新峰值在线人数
	UpdatePeakCount(ctx context.Context, id uint, count int) error

	// ListByCategory 根据分类查询直播间列表
	ListByCategory(ctx context.Context, categoryID uint, status string, page, pageSize int) ([]*model.LiveStream, int64, error)

	// ListHotLiveStreams 查询热门直播间（按在线人数排序）
	ListHotLiveStreams(ctx context.Context, limit int) ([]*model.LiveStream, error)

	// UpdateStatus 更新直播间状态
	UpdateStatus(ctx context.Context, id uint, status string) error

	// UpdateStartTime 更新开始时间
	UpdateStartTime(ctx context.Context, id uint, startTime time.Time) error

	// UpdateEndTime 更新结束时间
	UpdateEndTime(ctx context.Context, id uint, endTime time.Time) error

	// UpdateDuration 更新直播时长
	UpdateDuration(ctx context.Context, id uint, duration int64) error

	// IncrementOnlineCount 增加在线人数
	IncrementOnlineCount(ctx context.Context, id uint) error

	// DecrementOnlineCount 减少在线人数
	DecrementOnlineCount(ctx context.Context, id uint) error

	// IncrementViewCount 增加观看次数
	IncrementViewCount(ctx context.Context, id uint) error

	// IncrementGiftCount 增加礼物数量
	IncrementGiftCount(ctx context.Context, id uint, count int) error

	// IncrementGiftValue 增加礼物价值
	IncrementGiftValue(ctx context.Context, id uint, value int64) error
}

type liveStreamRepositoryImpl struct {
	db *gorm.DB
}

// NewLiveStreamRepository 创建直播间Repository
func NewLiveStreamRepository(db *gorm.DB) LiveStreamRepository {
	return &liveStreamRepositoryImpl{db: db}
}

// Create 创建直播间
func (r *liveStreamRepositoryImpl) Create(ctx context.Context, liveStream *model.LiveStream) error {
	return r.db.WithContext(ctx).Create(liveStream).Error
}

// Update 更新直播间信息（自动清除缓存）
func (r *liveStreamRepositoryImpl) Update(ctx context.Context, liveStream *model.LiveStream) error {
	// 清除多个相关缓存键
	keys := []string{
		fmt.Sprintf("livestream:id:%d", liveStream.ID),
		fmt.Sprintf("livestream:room:%s", liveStream.RoomID),
		fmt.Sprintf("livestream:key:%s", liveStream.StreamKey),
	}

	return cache.WithMultiCacheEvict("livestream", keys, func() error {
		return r.db.WithContext(ctx).Save(liveStream).Error
	})(ctx)
}

// FindByID 根据ID查询直播间（使用Redis缓存）
func (r *liveStreamRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.LiveStream, error) {
	// 使用 WithCache 装饰器自动管理缓存
	return cache.WithCache[*model.LiveStream](
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
			TTL:       5 * time.Minute, // 直播间信息缓存5分钟
		},
		func() (*model.LiveStream, error) {
			var liveStream model.LiveStream
			err := r.db.WithContext(ctx).
				Preload("Owner").
				First(&liveStream, id).Error
			if err != nil {
				return nil, err
			}
			return &liveStream, nil
		},
	)(ctx, id)
}

// FindByRoomID 根据房间ID查询直播间（使用Redis缓存）
func (r *liveStreamRepositoryImpl) FindByRoomID(ctx context.Context, roomID string) (*model.LiveStream, error) {
	// 使用 WithCache 装饰器自动管理缓存
	return cache.WithCache[*model.LiveStream](
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:room",
			TTL:       5 * time.Minute,
		},
		func() (*model.LiveStream, error) {
			var liveStream model.LiveStream
			err := r.db.WithContext(ctx).
				Preload("Owner").
				Where("room_id = ?", roomID).
				First(&liveStream).Error
			if err != nil {
				return nil, err
			}
			return &liveStream, nil
		},
	)(ctx, roomID)
}

// FindByStreamKey 根据推流密钥查询直播间（使用Redis缓存）
func (r *liveStreamRepositoryImpl) FindByStreamKey(ctx context.Context, streamKey string) (*model.LiveStream, error) {
	// 使用 WithCache 装饰器自动管理缓存
	return cache.WithCache[*model.LiveStream](
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:key",
			TTL:       5 * time.Minute,
		},
		func() (*model.LiveStream, error) {
			var liveStream model.LiveStream
			err := r.db.WithContext(ctx).
				Preload("Owner").
				Where("stream_key = ?", streamKey).
				First(&liveStream).Error
			if err != nil {
				return nil, err
			}
			return &liveStream, nil
		},
	)(ctx, streamKey)
}

// FindByOwnerID 查询用户的直播间
func (r *liveStreamRepositoryImpl) FindByOwnerID(ctx context.Context, ownerID uint) (*model.LiveStream, error) {
	var liveStream model.LiveStream
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Where("owner_id = ? AND status IN ('waiting', 'live')", ownerID).
		Order("created_at DESC").
		First(&liveStream).Error
	if err != nil {
		return nil, err
	}
	return &liveStream, nil
}

// List 分页查询直播间列表
func (r *liveStreamRepositoryImpl) List(ctx context.Context, status string, page, pageSize int) ([]*model.LiveStream, int64, error) {
	var liveStreams []*model.LiveStream
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LiveStream{})

	// 如果指定了状态，添加过滤条件
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Owner").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&liveStreams).Error

	if err != nil {
		return nil, 0, err
	}

	return liveStreams, total, nil
}

// UpdateViewCount 更新观看人数
func (r *liveStreamRepositoryImpl) UpdateViewCount(ctx context.Context, id uint, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		Update("view_count", count).Error
}

// UpdateOnlineCount 更新在线人数
func (r *liveStreamRepositoryImpl) UpdateOnlineCount(ctx context.Context, id uint, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		Update("online_count", count).Error
}

// IncrementLikeCount 增加点赞数
func (r *liveStreamRepositoryImpl) IncrementLikeCount(ctx context.Context, id uint, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", count)).Error
}

// Delete 删除直播间（自动清除缓存）
func (r *liveStreamRepositoryImpl) Delete(ctx context.Context, id uint) error {
	// 先查询获取直播间信息，用于清除缓存
	var liveStream model.LiveStream
	if err := r.db.WithContext(ctx).First(&liveStream, id).Error; err != nil {
		return err
	}

	// 清除多个相关缓存键
	keys := []string{
		fmt.Sprintf("livestream:id:%d", liveStream.ID),
		fmt.Sprintf("livestream:room:%s", liveStream.RoomID),
		fmt.Sprintf("livestream:key:%s", liveStream.StreamKey),
	}

	return cache.WithMultiCacheEvict("livestream", keys, func() error {
		return r.db.WithContext(ctx).Delete(&model.LiveStream{}, id).Error
	})(ctx)
}

// UpdateGiftStats 更新礼物统计
func (r *liveStreamRepositoryImpl) UpdateGiftStats(ctx context.Context, id uint, giftCount, giftValue int64) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				Updates(map[string]interface{}{
					"gift_count": gorm.Expr("gift_count + ?", giftCount),
					"gift_value": gorm.Expr("gift_value + ?", giftValue),
				}).Error
		},
	)(ctx, key)
}

// UpdateProductStats 更新商品统计
func (r *liveStreamRepositoryImpl) UpdateProductStats(ctx context.Context, id uint, productSales int64) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				UpdateColumn("product_sales", gorm.Expr("product_sales + ?", productSales)).Error
		},
	)(ctx, key)
}

// IncrementCommentCount 增加评论数
func (r *liveStreamRepositoryImpl) IncrementCommentCount(ctx context.Context, id uint, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", count)).Error
}

// IncrementShareCount 增加分享数
func (r *liveStreamRepositoryImpl) IncrementShareCount(ctx context.Context, id uint, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("share_count", gorm.Expr("share_count + ?", count)).Error
}

// UpdatePeakCount 更新峰值在线人数
func (r *liveStreamRepositoryImpl) UpdatePeakCount(ctx context.Context, id uint, count int) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ? AND peak_count < ?", id, count).
				Update("peak_count", count).Error
		},
	)(ctx, key)
}

// ListByCategory 根据分类查询直播间列表
func (r *liveStreamRepositoryImpl) ListByCategory(ctx context.Context, categoryID uint, status string, page, pageSize int) ([]*model.LiveStream, int64, error) {
	var liveStreams []*model.LiveStream
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LiveStream{}).Where("category_id = ?", categoryID)

	// 如果指定了状态，添加过滤条件
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Owner").
		Order("online_count DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&liveStreams).Error

	if err != nil {
		return nil, 0, err
	}

	return liveStreams, total, nil
}

// ListHotLiveStreams 查询热门直播间（按在线人数排序，使用Redis缓存）
func (r *liveStreamRepositoryImpl) ListHotLiveStreams(ctx context.Context, limit int) ([]*model.LiveStream, error) {
	// 使用 WithCache 装饰器缓存热门直播列表
	return cache.WithCache[[]*model.LiveStream](
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:hot",
			TTL:       1 * time.Minute, // 热门列表缓存1分钟，保持较高的实时性
		},
		func() ([]*model.LiveStream, error) {
			var liveStreams []*model.LiveStream
			err := r.db.WithContext(ctx).
				Where("status = ?", "live").
				Preload("Owner").
				Order("online_count DESC, view_count DESC").
				Limit(limit).
				Find(&liveStreams).Error
			if err != nil {
				return nil, err
			}
			return liveStreams, nil
		},
	)(ctx, limit)
}

// UpdateStatus 更新直播间状态（自动清除缓存）
func (r *liveStreamRepositoryImpl) UpdateStatus(ctx context.Context, id uint, status string) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				Update("status", status).Error
		},
	)(ctx, key)
}

// UpdateStartTime 更新开始时间（自动清除缓存）
func (r *liveStreamRepositoryImpl) UpdateStartTime(ctx context.Context, id uint, startTime time.Time) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				Update("started_at", startTime).Error
		},
	)(ctx, key)
}

// UpdateEndTime 更新结束时间（自动清除缓存）
func (r *liveStreamRepositoryImpl) UpdateEndTime(ctx context.Context, id uint, endTime time.Time) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				Update("ended_at", endTime).Error
		},
	)(ctx, key)
}

// UpdateDuration 更新直播时长（自动清除缓存）
func (r *liveStreamRepositoryImpl) UpdateDuration(ctx context.Context, id uint, duration int64) error {
	key := fmt.Sprintf("livestream:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "livestream",
			KeyPrefix: "livestream:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveStream{}).
				Where("id = ?", id).
				Update("duration", duration).Error
		},
	)(ctx, key)
}

// IncrementOnlineCount 增加在线人数
func (r *liveStreamRepositoryImpl) IncrementOnlineCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("online_count", gorm.Expr("online_count + 1")).Error
}

// DecrementOnlineCount 减少在线人数
func (r *liveStreamRepositoryImpl) DecrementOnlineCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ? AND online_count > 0", id).
		UpdateColumn("online_count", gorm.Expr("online_count - 1")).Error
}

// IncrementViewCount 增加观看次数
func (r *liveStreamRepositoryImpl) IncrementViewCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementGiftCount 增加礼物数量
func (r *liveStreamRepositoryImpl) IncrementGiftCount(ctx context.Context, id uint, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("gift_count", gorm.Expr("gift_count + ?", count)).Error
}

// IncrementGiftValue 增加礼物价值
func (r *liveStreamRepositoryImpl) IncrementGiftValue(ctx context.Context, id uint, value int64) error {
	return r.db.WithContext(ctx).
		Model(&model.LiveStream{}).
		Where("id = ?", id).
		UpdateColumn("gift_value", gorm.Expr("gift_value + ?", value)).Error
}
