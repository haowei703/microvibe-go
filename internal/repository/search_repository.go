package repository

import (
	"context"
	"microvibe-go/internal/model"

	"gorm.io/gorm"
)

// SearchRepository 搜索仓储层接口
type SearchRepository interface {
	// CreateSearchHistory 创建搜索历史
	CreateSearchHistory(ctx context.Context, history *model.SearchHistory) error
	// GetUserSearchHistory 获取用户搜索历史
	GetUserSearchHistory(ctx context.Context, userID uint, limit int) ([]*model.SearchHistory, error)
	// DeleteUserSearchHistory 删除用户搜索历史
	DeleteUserSearchHistory(ctx context.Context, userID uint) error
	// GetHotSearches 获取热搜列表
	GetHotSearches(ctx context.Context, limit int) ([]*model.HotSearch, error)
	// IncrementSearchCount 增加搜索次数
	IncrementSearchCount(ctx context.Context, keyword string) error
	// SearchVideos 搜索视频
	SearchVideos(ctx context.Context, keyword string, page, pageSize int) ([]*model.Video, int64, error)
	// SearchUsers 搜索用户
	SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error)
	// SearchHashtags 搜索话题
	SearchHashtags(ctx context.Context, keyword string, page, pageSize int) ([]*model.Hashtag, int64, error)
}

// searchRepositoryImpl 搜索仓储层实现
type searchRepositoryImpl struct {
	db *gorm.DB
}

// NewSearchRepository 创建搜索仓储实例
func NewSearchRepository(db *gorm.DB) SearchRepository {
	return &searchRepositoryImpl{db: db}
}

// CreateSearchHistory 创建搜索历史
func (r *searchRepositoryImpl) CreateSearchHistory(ctx context.Context, history *model.SearchHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// GetUserSearchHistory 获取用户搜索历史
func (r *searchRepositoryImpl) GetUserSearchHistory(ctx context.Context, userID uint, limit int) ([]*model.SearchHistory, error) {
	var histories []*model.SearchHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories).Error
	return histories, err
}

// DeleteUserSearchHistory 删除用户搜索历史
func (r *searchRepositoryImpl) DeleteUserSearchHistory(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.SearchHistory{}).Error
}

// GetHotSearches 获取热搜列表
func (r *searchRepositoryImpl) GetHotSearches(ctx context.Context, limit int) ([]*model.HotSearch, error) {
	var hotSearches []*model.HotSearch
	err := r.db.WithContext(ctx).
		Order("is_sticky DESC, hot_score DESC, search_count DESC").
		Limit(limit).
		Find(&hotSearches).Error
	return hotSearches, err
}

// IncrementSearchCount 增加搜索次数
func (r *searchRepositoryImpl) IncrementSearchCount(ctx context.Context, keyword string) error {
	// 先尝试查找是否存在
	var hotSearch model.HotSearch
	err := r.db.WithContext(ctx).Where("keyword = ?", keyword).First(&hotSearch).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在则创建
		hotSearch = model.HotSearch{
			Keyword:     keyword,
			SearchCount: 1,
			HotScore:    1.0,
		}
		return r.db.WithContext(ctx).Create(&hotSearch).Error
	}

	if err != nil {
		return err
	}

	// 存在则更新计数和热度
	return r.db.WithContext(ctx).Model(&hotSearch).Updates(map[string]interface{}{
		"search_count": gorm.Expr("search_count + 1"),
		"hot_score":    gorm.Expr("hot_score + 0.1"),
	}).Error
}

// SearchVideos 搜索视频
func (r *searchRepositoryImpl) SearchVideos(ctx context.Context, keyword string, page, pageSize int) ([]*model.Video, int64, error) {
	var videos []*model.Video
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Video{}).
		Where("status = ?", 1). // 只搜索已发布的视频
		Where("title LIKE ? OR description LIKE ? OR tags LIKE ?",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Preload("Category").
		Order("hot_score DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&videos).Error

	return videos, total, err
}

// SearchUsers 搜索用户
func (r *searchRepositoryImpl) SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := r.db.WithContext(ctx).Model(&model.User{}).
		Where("status = ?", 1). // 只搜索正常状态用户
		Where("username LIKE ? OR nickname LIKE ? OR signature LIKE ?",
			"%"+keyword+"%",
			"%"+keyword+"%",
			"%"+keyword+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("follower_count DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// SearchHashtags 搜索话题
func (r *searchRepositoryImpl) SearchHashtags(ctx context.Context, keyword string, page, pageSize int) ([]*model.Hashtag, int64, error) {
	var hashtags []*model.Hashtag
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Hashtag{}).
		Where("name LIKE ?", "%"+keyword+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("hot_score DESC, view_count DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&hashtags).Error

	return hashtags, total, err
}
