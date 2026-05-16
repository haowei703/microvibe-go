package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CategoryRepository 分类数据访问层接口
type CategoryRepository interface {
	// FindAll 获取所有分类
	FindAll(ctx context.Context) ([]*model.Category, error)
	// FindByID 根据ID获取分类
	FindByID(ctx context.Context, id uint) (*model.Category, error)
}

type categoryRepositoryImpl struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepositoryImpl{db: db}
}

func (r *categoryRepositoryImpl) FindAll(ctx context.Context) ([]*model.Category, error) {
	var categories []*model.Category
	if err := r.db.WithContext(ctx).Order("sort ASC").Find(&categories).Error; err != nil {
		logger.Error("获取分类列表失败", zap.Error(err))
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.Category, error) {
	var category model.Category
	if err := r.db.WithContext(ctx).First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}
