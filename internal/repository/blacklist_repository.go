package repository

import (
	"context"
	"microvibe-go/internal/model"

	"gorm.io/gorm"
)

// BlacklistRepository 黑名单数据访问层接口
type BlacklistRepository interface {
	Create(ctx context.Context, blacklist *model.Blacklist) error
	Delete(ctx context.Context, userID, blockedUserID uint) error
	IsBlocked(ctx context.Context, userID, blockedUserID uint) (bool, error)
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Blacklist, int64, error)
}

type blacklistRepositoryImpl struct {
	db *gorm.DB
}

func NewBlacklistRepository(db *gorm.DB) BlacklistRepository {
	return &blacklistRepositoryImpl{db: db}
}

func (r *blacklistRepositoryImpl) Create(ctx context.Context, blacklist *model.Blacklist) error {
	return r.db.WithContext(ctx).FirstOrCreate(blacklist, model.Blacklist{
		UserID:        blacklist.UserID,
		BlockedUserID: blacklist.BlockedUserID,
	}).Error
}

func (r *blacklistRepositoryImpl) Delete(ctx context.Context, userID, blockedUserID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.Blacklist{}).Error
}

func (r *blacklistRepositoryImpl) IsBlocked(ctx context.Context, userID, blockedUserID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Blacklist{}).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Count(&count).Error
	return count > 0, err
}

func (r *blacklistRepositoryImpl) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Blacklist, int64, error) {
	var blacklists []*model.Blacklist
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Blacklist{}).Where("user_id = ?", userID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("BlockedUser").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&blacklists).Error

	return blacklists, total, err
}
