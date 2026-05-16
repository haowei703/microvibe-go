package repository

import (
	"context"
	"gorm.io/gorm"
	"microvibe-go/internal/model"
)

// ShareRepository 分享数据访问层接口
type ShareRepository interface {
	Create(ctx context.Context, share *model.Share) error
	CountByVideoID(ctx context.Context, videoID uint) (int64, error)
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Share, int64, error)
}

type shareRepositoryImpl struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepositoryImpl{db: db}
}

func (r *shareRepositoryImpl) Create(ctx context.Context, share *model.Share) error {
	return r.db.WithContext(ctx).Create(share).Error
}

func (r *shareRepositoryImpl) CountByVideoID(ctx context.Context, videoID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Share{}).Where("video_id = ?", videoID).Count(&count).Error
	return count, err
}

func (r *shareRepositoryImpl) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Share, int64, error) {
	var shares []*model.Share
	var total int64
	db := r.db.WithContext(ctx).Model(&model.Share{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&shares).Error
	return shares, total, err
}
