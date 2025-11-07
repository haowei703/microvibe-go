package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AddProductRequest 添加商品请求
type AddProductRequest struct {
	LiveID         uint    `json:"live_id" binding:"required"`
	ProductID      uint    `json:"product_id" binding:"required"`
	Name           string  `json:"name" binding:"required"`
	Cover          string  `json:"cover"`
	Price          float64 `json:"price" binding:"required,min=0"`
	SalePrice      float64 `json:"sale_price" binding:"required,min=0"`
	Stock          int     `json:"stock" binding:"required,min=0"`
	Sort           int     `json:"sort"`
	Discount       int     `json:"discount"`
	Description    string  `json:"description"`
	CommissionRate float64 `json:"commission_rate"`
}

// UpdateProductRequest 更新商品请求
type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Cover       string  `json:"cover"`
	Price       float64 `json:"price"`
	SalePrice   float64 `json:"sale_price"`
	Stock       int     `json:"stock"`
	Sort        int     `json:"sort"`
	Status      int8    `json:"status"`
	IsHot       bool    `json:"is_hot"`
	Discount    int     `json:"discount"`
	Description string  `json:"description"`
}

// LiveProductService 直播商品服务接口
type LiveProductService interface {
	// AddProduct 添加商品到直播间
	AddProduct(ctx context.Context, userID uint, req *AddProductRequest) (*model.LiveProduct, error)

	// UpdateProduct 更新商品
	UpdateProduct(ctx context.Context, userID uint, productID uint, req *UpdateProductRequest) error

	// DeleteProduct 删除商品
	DeleteProduct(ctx context.Context, userID uint, productID uint) error

	// GetProduct 获取商品详情
	GetProduct(ctx context.Context, productID uint) (*model.LiveProduct, error)

	// ListProducts 获取直播间商品列表
	ListProducts(ctx context.Context, liveID uint, status int8) ([]*model.LiveProduct, error)

	// GetHotProducts 获取热卖商品
	GetHotProducts(ctx context.Context, liveID uint, limit int) ([]*model.LiveProduct, error)

	// ExplainProduct 讲解商品
	ExplainProduct(ctx context.Context, userID uint, productID uint) error

	// UpdateStock 更新库存
	UpdateStock(ctx context.Context, userID uint, productID uint, quantity int) error

	// SellProduct 销售商品（减少库存，增加销量）
	SellProduct(ctx context.Context, productID uint, quantity int) error
}

type liveProductServiceImpl struct {
	productRepo    repository.LiveProductRepository
	liveStreamRepo repository.LiveStreamRepository
}

// NewLiveProductService 创建商品服务
func NewLiveProductService(
	productRepo repository.LiveProductRepository,
	liveStreamRepo repository.LiveStreamRepository,
) LiveProductService {
	return &liveProductServiceImpl{
		productRepo:    productRepo,
		liveStreamRepo: liveStreamRepo,
	}
}

// AddProduct 添加商品到直播间
func (s *liveProductServiceImpl) AddProduct(ctx context.Context, userID uint, req *AddProductRequest) (*model.LiveProduct, error) {
	// 1. 查询直播间
	liveStream, err := s.liveStreamRepo.FindByID(ctx, req.LiveID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("live_id", req.LiveID))
		return nil, errors.New("查询直播间失败")
	}

	// 2. 验证权限（只有主播可以添加商品）
	if liveStream.OwnerID != userID {
		return nil, errors.New("无权限操作")
	}

	// 3. 创建商品
	product := &model.LiveProduct{
		LiveID:         req.LiveID,
		ProductID:      req.ProductID,
		Name:           req.Name,
		Cover:          req.Cover,
		Price:          req.Price,
		SalePrice:      req.SalePrice,
		Stock:          req.Stock,
		Sort:           req.Sort,
		Status:         1, // 默认上架
		Discount:       req.Discount,
		Description:    req.Description,
		CommissionRate: req.CommissionRate,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		logger.Error("添加商品失败", zap.Error(err))
		return nil, errors.New("添加商品失败")
	}

	logger.Info("添加商品成功",
		zap.Uint("product_id", product.ID),
		zap.Uint("live_id", req.LiveID),
		zap.String("name", req.Name))

	return product, nil
}

// UpdateProduct 更新商品
func (s *liveProductServiceImpl) UpdateProduct(ctx context.Context, userID uint, productID uint, req *UpdateProductRequest) error {
	// 1. 查询商品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("商品不存在")
		}
		logger.Error("查询商品失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("查询商品失败")
	}

	// 2. 查询直播间，验证权限
	liveStream, err := s.liveStreamRepo.FindByID(ctx, product.LiveID)
	if err != nil {
		return errors.New("查询直播间失败")
	}

	if liveStream.OwnerID != userID {
		return errors.New("无权限操作")
	}

	// 3. 更新字段
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Cover != "" {
		product.Cover = req.Cover
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.SalePrice > 0 {
		product.SalePrice = req.SalePrice
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}
	product.Sort = req.Sort
	product.Status = req.Status
	product.IsHot = req.IsHot
	product.Discount = req.Discount
	if req.Description != "" {
		product.Description = req.Description
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		logger.Error("更新商品失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("更新商品失败")
	}

	logger.Info("更新商品成功", zap.Uint("product_id", productID))
	return nil
}

// DeleteProduct 删除商品
func (s *liveProductServiceImpl) DeleteProduct(ctx context.Context, userID uint, productID uint) error {
	// 1. 查询商品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("商品不存在")
		}
		return errors.New("查询商品失败")
	}

	// 2. 查询直播间，验证权限
	liveStream, err := s.liveStreamRepo.FindByID(ctx, product.LiveID)
	if err != nil {
		return errors.New("查询直播间失败")
	}

	if liveStream.OwnerID != userID {
		return errors.New("无权限操作")
	}

	// 3. 删除商品
	if err := s.productRepo.Delete(ctx, productID); err != nil {
		logger.Error("删除商品失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("删除商品失败")
	}

	logger.Info("删除商品成功", zap.Uint("product_id", productID))
	return nil
}

// GetProduct 获取商品详情
func (s *liveProductServiceImpl) GetProduct(ctx context.Context, productID uint) (*model.LiveProduct, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("商品不存在")
		}
		logger.Error("查询商品失败", zap.Error(err), zap.Uint("product_id", productID))
		return nil, errors.New("查询商品失败")
	}

	return product, nil
}

// ListProducts 获取直播间商品列表
func (s *liveProductServiceImpl) ListProducts(ctx context.Context, liveID uint, status int8) ([]*model.LiveProduct, error) {
	products, err := s.productRepo.ListByLiveID(ctx, liveID, status)
	if err != nil {
		logger.Error("查询商品列表失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, errors.New("查询商品列表失败")
	}

	logger.Info("查询商品列表成功", zap.Uint("live_id", liveID), zap.Int("count", len(products)))
	return products, nil
}

// GetHotProducts 获取热卖商品
func (s *liveProductServiceImpl) GetHotProducts(ctx context.Context, liveID uint, limit int) ([]*model.LiveProduct, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	products, err := s.productRepo.GetHotProducts(ctx, liveID, limit)
	if err != nil {
		logger.Error("查询热卖商品失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, errors.New("查询热卖商品失败")
	}

	return products, nil
}

// ExplainProduct 讲解商品
func (s *liveProductServiceImpl) ExplainProduct(ctx context.Context, userID uint, productID uint) error {
	// 1. 查询商品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("商品不存在")
		}
		return errors.New("查询商品失败")
	}

	// 2. 查询直播间，验证权限
	liveStream, err := s.liveStreamRepo.FindByID(ctx, product.LiveID)
	if err != nil {
		return errors.New("查询直播间失败")
	}

	if liveStream.OwnerID != userID {
		return errors.New("无权限操作")
	}

	// 3. 更新讲解时间
	now := time.Now()
	if err := s.productRepo.UpdateExplainedAt(ctx, productID, now); err != nil {
		logger.Error("更新讲解时间失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("更新讲解时间失败")
	}

	logger.Info("讲解商品成功", zap.Uint("product_id", productID), zap.String("name", product.Name))
	return nil
}

// UpdateStock 更新库存
func (s *liveProductServiceImpl) UpdateStock(ctx context.Context, userID uint, productID uint, quantity int) error {
	// 1. 查询商品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("商品不存在")
		}
		return errors.New("查询商品失败")
	}

	// 2. 查询直播间，验证权限
	liveStream, err := s.liveStreamRepo.FindByID(ctx, product.LiveID)
	if err != nil {
		return errors.New("查询直播间失败")
	}

	if liveStream.OwnerID != userID {
		return errors.New("无权限操作")
	}

	// 3. 更新库存
	if err := s.productRepo.UpdateStock(ctx, productID, quantity); err != nil {
		logger.Error("更新库存失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("更新库存失败")
	}

	logger.Info("更新库存成功", zap.Uint("product_id", productID), zap.Int("quantity", quantity))
	return nil
}

// SellProduct 销售商品（减少库存，增加销量）
func (s *liveProductServiceImpl) SellProduct(ctx context.Context, productID uint, quantity int) error {
	// 1. 查询商品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("商品不存在")
		}
		return errors.New("查询商品失败")
	}

	// 2. 检查库存
	if product.Stock < quantity {
		return errors.New("库存不足")
	}

	// 3. 更新销售数量
	if err := s.productRepo.IncrementSoldCount(ctx, productID, quantity); err != nil {
		logger.Error("销售商品失败", zap.Error(err), zap.Uint("product_id", productID))
		return errors.New("销售商品失败")
	}

	// 4. 更新直播间商品销售额
	totalSales := product.SalePrice * float64(quantity)
	if err := s.liveStreamRepo.UpdateProductStats(ctx, product.LiveID, int64(totalSales)); err != nil {
		logger.Error("更新直播间销售统计失败", zap.Error(err))
		// 不影响主流程
	}

	logger.Info("销售商品成功",
		zap.Uint("product_id", productID),
		zap.String("name", product.Name),
		zap.Int("quantity", quantity))

	return nil
}
