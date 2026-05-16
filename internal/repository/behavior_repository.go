package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BehaviorRepository 用户行为数据访问层接口
type BehaviorRepository interface {
	// Create 创建行为记录
	Create(ctx context.Context, behavior *model.UserBehavior) error
}

type behaviorRepositoryImpl struct {
	db *gorm.DB
}

// NewBehaviorRepository 创建用户行为数据访问层实例
func NewBehaviorRepository(db *gorm.DB) BehaviorRepository {
	return &behaviorRepositoryImpl{
		db: db,
	}
}

// Create 创建行为记录
func (r *behaviorRepositoryImpl) Create(ctx context.Context, behavior *model.UserBehavior) error {
	logger.Debug("记录用户行为", zap.Uint("user_id", behavior.UserID), zap.Int8("action", behavior.Action))

	if err := r.db.WithContext(ctx).Create(behavior).Error; err != nil {
		logger.Error("记录用户行为失败", zap.Error(err))
		return err
	}

	return nil
}
