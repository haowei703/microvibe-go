package repository

import (
	"context"
	"fmt"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/cache"
	"time"

	"gorm.io/gorm"
)

// LiveProductRepository 直播商品数据访问接口
type LiveProductRepository interface {
	// Create 创建商品
	Create(ctx context.Context, product *model.LiveProduct) error

	// Update 更新商品
	Update(ctx context.Context, product *model.LiveProduct) error

	// FindByID 根据ID查询商品
	FindByID(ctx context.Context, id uint) (*model.LiveProduct, error)

	// ListByLiveID 查询直播间的商品列表
	ListByLiveID(ctx context.Context, liveID uint, status int8) ([]*model.LiveProduct, error)

	// Delete 删除商品
	Delete(ctx context.Context, id uint) error

	// UpdateStock 更新库存
	UpdateStock(ctx context.Context, id uint, quantity int) error

	// IncrementSoldCount 增加销售数量
	IncrementSoldCount(ctx context.Context, id uint, count int) error

	// UpdateExplainedAt 更新讲解时间
	UpdateExplainedAt(ctx context.Context, id uint, explainedAt time.Time) error

	// GetHotProducts 获取热卖商品
	GetHotProducts(ctx context.Context, liveID uint, limit int) ([]*model.LiveProduct, error)
}

type liveProductRepositoryImpl struct {
	db *gorm.DB
}

// NewLiveProductRepository 创建商品Repository
func NewLiveProductRepository(db *gorm.DB) LiveProductRepository {
	return &liveProductRepositoryImpl{db: db}
}

// Create 创建商品
func (r *liveProductRepositoryImpl) Create(ctx context.Context, product *model.LiveProduct) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// Update 更新商品（自动清除缓存）
func (r *liveProductRepositoryImpl) Update(ctx context.Context, product *model.LiveProduct) error {
	key := fmt.Sprintf("live:product:id:%d", product.ID)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
		},
		func() error {
			return r.db.WithContext(ctx).Save(product).Error
		},
	)(ctx, key)
}

// FindByID 根据ID查询商品（使用Redis缓存）
func (r *liveProductRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.LiveProduct, error) {
	return cache.WithCache[*model.LiveProduct](
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
			TTL:       5 * time.Minute,
		},
		func() (*model.LiveProduct, error) {
			var product model.LiveProduct
			if err := r.db.WithContext(ctx).First(&product, id).Error; err != nil {
				return nil, err
			}
			return &product, nil
		},
	)(ctx, id)
}

// ListByLiveID 查询直播间的商品列表（使用Redis缓存）
func (r *liveProductRepositoryImpl) ListByLiveID(ctx context.Context, liveID uint, status int8) ([]*model.LiveProduct, error) {
	cacheKey := fmt.Sprintf("live:%d:status:%d", liveID, status)

	return cache.WithCache[[]*model.LiveProduct](
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:list",
			TTL:       2 * time.Minute, // 商品列表缓存2分钟，保持较高的实时性
		},
		func() ([]*model.LiveProduct, error) {
			var products []*model.LiveProduct
			query := r.db.WithContext(ctx).Where("live_id = ?", liveID)

			if status >= 0 {
				query = query.Where("status = ?", status)
			}

			err := query.Order("sort ASC, created_at DESC").Find(&products).Error
			if err != nil {
				return nil, err
			}
			return products, nil
		},
	)(ctx, cacheKey)
}

// Delete 删除商品（自动清除缓存）
func (r *liveProductRepositoryImpl) Delete(ctx context.Context, id uint) error {
	key := fmt.Sprintf("live:product:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
		},
		func() error {
			return r.db.WithContext(ctx).Delete(&model.LiveProduct{}, id).Error
		},
	)(ctx, key)
}

// UpdateStock 更新库存
func (r *liveProductRepositoryImpl) UpdateStock(ctx context.Context, id uint, quantity int) error {
	key := fmt.Sprintf("live:product:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveProduct{}).
				Where("id = ?", id).
				UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).Error
		},
	)(ctx, key)
}

// IncrementSoldCount 增加销售数量
func (r *liveProductRepositoryImpl) IncrementSoldCount(ctx context.Context, id uint, count int) error {
	key := fmt.Sprintf("live:product:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
		},
		func() error {
			// 同时减少库存和增加销售数量
			return r.db.WithContext(ctx).
				Model(&model.LiveProduct{}).
				Where("id = ? AND stock >= ?", id, count).
				Updates(map[string]interface{}{
					"stock":      gorm.Expr("stock - ?", count),
					"sold_count": gorm.Expr("sold_count + ?", count),
				}).Error
		},
	)(ctx, key)
}

// UpdateExplainedAt 更新讲解时间
func (r *liveProductRepositoryImpl) UpdateExplainedAt(ctx context.Context, id uint, explainedAt time.Time) error {
	key := fmt.Sprintf("live:product:id:%d", id)
	return cache.WithCacheEvict(
		cache.CacheConfig{
			CacheName: "liveproduct",
			KeyPrefix: "live:product:id",
		},
		func() error {
			return r.db.WithContext(ctx).
				Model(&model.LiveProduct{}).
				Where("id = ?", id).
				Update("explained_at", explainedAt).Error
		},
	)(ctx, key)
}

// GetHotProducts 获取热卖商品
func (r *liveProductRepositoryImpl) GetHotProducts(ctx context.Context, liveID uint, limit int) ([]*model.LiveProduct, error) {
	var products []*model.LiveProduct

	err := r.db.WithContext(ctx).
		Where("live_id = ? AND status = ? AND is_hot = ?", liveID, 1, true).
		Order("sold_count DESC, sort ASC").
		Limit(limit).
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	return products, nil
}
