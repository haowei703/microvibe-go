package repository

import (
	"context"
	"microvibe-go/internal/model"

	"gorm.io/gorm"
)

// HashtagRepository 话题仓储层接口
type HashtagRepository interface {
	// CreateHashtag 创建话题
	CreateHashtag(ctx context.Context, hashtag *model.Hashtag) error
	// GetHashtagByID 根据ID获取话题
	GetHashtagByID(ctx context.Context, id uint) (*model.Hashtag, error)
	// GetHashtagByName 根据名称获取话题
	GetHashtagByName(ctx context.Context, name string) (*model.Hashtag, error)
	// GetHotHashtags 获取热门话题
	GetHotHashtags(ctx context.Context, limit int) ([]*model.Hashtag, error)
	// UpdateHashtag 更新话题
	UpdateHashtag(ctx context.Context, hashtag *model.Hashtag) error
	// IncrementViewCount 增加浏览量
	IncrementViewCount(ctx context.Context, id uint) error
	// GetHashtagVideos 获取话题下的视频
	GetHashtagVideos(ctx context.Context, hashtagID uint, page, pageSize int) ([]*model.Video, int64, error)
	// AddVideoToHashtag 将视频添加到话题
	AddVideoToHashtag(ctx context.Context, videoID, hashtagID uint) error
}

// hashtagRepositoryImpl 话题仓储层实现
type hashtagRepositoryImpl struct {
	db *gorm.DB
}

// NewHashtagRepository 创建话题仓储实例
func NewHashtagRepository(db *gorm.DB) HashtagRepository {
	return &hashtagRepositoryImpl{db: db}
}

// CreateHashtag 创建话题
func (r *hashtagRepositoryImpl) CreateHashtag(ctx context.Context, hashtag *model.Hashtag) error {
	return r.db.WithContext(ctx).Create(hashtag).Error
}

// GetHashtagByID 根据ID获取话题
func (r *hashtagRepositoryImpl) GetHashtagByID(ctx context.Context, id uint) (*model.Hashtag, error) {
	var hashtag model.Hashtag
	err := r.db.WithContext(ctx).First(&hashtag, id).Error
	return &hashtag, err
}

// GetHashtagByName 根据名称获取话题
func (r *hashtagRepositoryImpl) GetHashtagByName(ctx context.Context, name string) (*model.Hashtag, error) {
	var hashtag model.Hashtag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&hashtag).Error
	return &hashtag, err
}

// GetHotHashtags 获取热门话题
func (r *hashtagRepositoryImpl) GetHotHashtags(ctx context.Context, limit int) ([]*model.Hashtag, error) {
	var hashtags []*model.Hashtag
	err := r.db.WithContext(ctx).
		Where("is_hot = ?", true).
		Order("hot_score DESC, view_count DESC").
		Limit(limit).
		Find(&hashtags).Error
	return hashtags, err
}

// UpdateHashtag 更新话题
func (r *hashtagRepositoryImpl) UpdateHashtag(ctx context.Context, hashtag *model.Hashtag) error {
	return r.db.WithContext(ctx).Save(hashtag).Error
}

// IncrementViewCount 增加浏览量
func (r *hashtagRepositoryImpl) IncrementViewCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.Hashtag{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"view_count": gorm.Expr("view_count + 1"),
			"hot_score":  gorm.Expr("hot_score + 0.01"),
		}).Error
}

// GetHashtagVideos 获取话题下的视频
func (r *hashtagRepositoryImpl) GetHashtagVideos(ctx context.Context, hashtagID uint, page, pageSize int) ([]*model.Video, int64, error) {
	var videos []*model.Video
	var total int64

	// 通过关联表查询
	query := r.db.WithContext(ctx).
		Table("videos").
		Joins("INNER JOIN video_hashtags ON videos.id = video_hashtags.video_id").
		Where("video_hashtags.hashtag_id = ? AND videos.status = ?", hashtagID, 1)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Preload("Category").
		Order("videos.hot_score DESC, videos.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&videos).Error

	return videos, total, err
}

// AddVideoToHashtag 将视频添加到话题
func (r *hashtagRepositoryImpl) AddVideoToHashtag(ctx context.Context, videoID, hashtagID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否已存在
		var count int64
		if err := tx.Model(&model.VideoHashtag{}).
			Where("video_id = ? AND hashtag_id = ?", videoID, hashtagID).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return nil // 已存在，不重复添加
		}

		// 创建关联
		videoHashtag := &model.VideoHashtag{
			VideoID:   videoID,
			HashtagID: hashtagID,
		}
		if err := tx.Create(videoHashtag).Error; err != nil {
			return err
		}

		// 更新话题视频数量
		return tx.Model(&model.Hashtag{}).
			Where("id = ?", hashtagID).
			Update("video_count", gorm.Expr("video_count + 1")).Error
	})
}
