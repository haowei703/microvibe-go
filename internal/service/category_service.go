package service

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
)

// CategoryService 分类服务层接口
type CategoryService interface {
	// GetCategories 获取所有视频分类
	GetCategories(ctx context.Context) ([]*model.Category, error)
}

type categoryServiceImpl struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryServiceImpl{categoryRepo: categoryRepo}
}

func (s *categoryServiceImpl) GetCategories(ctx context.Context) ([]*model.Category, error) {
	return s.categoryRepo.FindAll(ctx)
}
