package repository

import (
	"context"
	"microvibe-go/internal/model"
	"strings"

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
	// SearchBestMatchUser 搜索最佳匹配用户
	SearchBestMatchUser(ctx context.Context, keyword string) (*model.User, error)
	// GetUserRecentVideos 获取用户最近/置顶视频
	GetUserRecentVideos(ctx context.Context, userID uint, limit int) ([]*model.Video, error)
	// SearchHashtags 搜索话题
	SearchHashtags(ctx context.Context, keyword string, page, pageSize int) ([]*model.Hashtag, int64, error)
	// RecommendUsers 推荐用户 (当搜索关键字为空时)
	RecommendUsers(ctx context.Context, userID uint, page, pageSize int) ([]*model.User, int64, error)
	// GetSuggestUsers 获取搜索建议用户
	GetSuggestUsers(ctx context.Context, keyword string, limit int) ([]*model.User, error)
	// GetSuggestHashtags 获取搜索建议话题
	GetSuggestHashtags(ctx context.Context, keyword string, limit int) ([]*model.Hashtag, error)
	// DeleteHotSearch 删除热搜
	DeleteHotSearch(ctx context.Context, keyword string) error
	// UpdateHotSearchCount 更新热度计数
	UpdateHotSearchCount(ctx context.Context, keyword string, count int64) error
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
		Where("username LIKE ? OR nickname LIKE ? OR bio LIKE ?",
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

// RecommendUsers 推荐用户 (当搜索关键字为空时)
func (r *searchRepositoryImpl) RecommendUsers(ctx context.Context, userID uint, page, pageSize int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	// 基础查询
	baseQuery := r.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", 1).Where("id != ?", userID)

	if userID == 0 {
		// 未登录用户：返回热门用户
		if err := baseQuery.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		err := baseQuery.Order("follower_count DESC, created_at DESC").
			Offset((page - 1) * pageSize).
			Limit(pageSize).
			Find(&users).Error
		return users, total, err
	}

	// 登录用户：基于社交关系的推荐评分
	// 评分逻辑：
	// 1. 互相关注: 100分
	// 2. 单向关注/粉丝: 50分
	// 3. 有过聊天: 30分
	// 4. 共同关注 (朋友的朋友): 每人 10分

	// 使用子查询计算评分并排序
	scoreSubQuery := r.db.Table("users").
		Select("users.id, ("+
			"CASE "+
			"  WHEN EXISTS(SELECT 1 FROM follows f1 WHERE f1.user_id = ? AND f1.followed_id = users.id) AND EXISTS(SELECT 1 FROM follows f2 WHERE f2.user_id = users.id AND f2.followed_id = ?) THEN 100 "+
			"  WHEN EXISTS(SELECT 1 FROM follows f3 WHERE f3.user_id = ? AND f3.followed_id = users.id OR f3.user_id = users.id AND f3.followed_id = ?) THEN 50 "+
			"  ELSE 0 "+
			"END + "+
			"CASE WHEN EXISTS(SELECT 1 FROM conversations c WHERE (c.user1_id = ? AND c.user2_id = users.id) OR (c.user1_id = users.id AND c.user2_id = ?)) THEN 30 ELSE 0 END + "+
			"COALESCE((SELECT COUNT(*) * 10 FROM follows f4 JOIN follows f5 ON f4.followed_id = f5.user_id WHERE f4.user_id = ? AND f5.followed_id = users.id), 0)"+
			") as score", userID, userID, userID, userID, userID, userID, userID)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询并按分数排序
	err := r.db.WithContext(ctx).
		Select("users.*, s.score").
		Table("users").
		Joins("JOIN (?) AS s ON s.id = users.id", scoreSubQuery).
		Where("users.status = ?", 1).
		Where("users.id != ?", userID).
		Order("s.score DESC, users.follower_count DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// GetSuggestUsers 获取搜索建议用户
func (r *searchRepositoryImpl) GetSuggestUsers(ctx context.Context, keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	err := r.db.WithContext(ctx).
		Select("id, username, nickname").
		Where("(username LIKE ? OR nickname LIKE ?) AND status = 1", keyword+"%", keyword+"%").
		Order("follower_count DESC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// GetSuggestHashtags 获取搜索建议话题
func (r *searchRepositoryImpl) GetSuggestHashtags(ctx context.Context, keyword string, limit int) ([]*model.Hashtag, error) {
	var hashtags []*model.Hashtag
	err := r.db.WithContext(ctx).
		Select("id, name").
		Where("name LIKE ?", keyword+"%").
		Order("video_count DESC").
		Limit(limit).
		Find(&hashtags).Error
	return hashtags, err
}

// SearchBestMatchUser 搜索最佳匹配用户（精确匹配优先，关联度次之）
func (r *searchRepositoryImpl) SearchBestMatchUser(ctx context.Context, keyword string) (*model.User, error) {
	var user model.User
	keyword = strings.TrimSpace(keyword)

	// 1. 精确匹配用户名
	err := r.db.WithContext(ctx).
		Where("username = ? AND status = 1", keyword).
		First(&user).Error
	if err == nil {
		return &user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 2. 精确匹配昵称
	err = r.db.WithContext(ctx).
		Where("nickname = ? AND status = 1", keyword).
		Order("follower_count DESC").
		First(&user).Error
	if err == nil {
		return &user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 3. 模糊匹配，按关联度排序（粉丝数 + 视频数）
	err = r.db.WithContext(ctx).
		Where("(username LIKE ? OR nickname LIKE ?) AND status = 1", "%"+keyword+"%", "%"+keyword+"%").
		Order("follower_count DESC, video_count DESC").
		First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserRecentVideos 获取用户最近/置顶视频（置顶优先，然后按发布时间倒序）
func (r *searchRepositoryImpl) GetUserRecentVideos(ctx context.Context, userID uint, limit int) ([]*model.Video, error) {
	var videos []*model.Video
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = 1", userID).
		Order("is_top DESC, published_at DESC").
		Limit(limit).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	if videos == nil {
		videos = []*model.Video{}
	}
	return videos, nil
}

// DeleteHotSearch 删除热搜
func (r *searchRepositoryImpl) DeleteHotSearch(ctx context.Context, keyword string) error {
	return r.db.WithContext(ctx).Where("keyword = ?", keyword).Delete(&model.HotSearch{}).Error
}

// UpdateHotSearchCount 更新热度计数
func (r *searchRepositoryImpl) UpdateHotSearchCount(ctx context.Context, keyword string, count int64) error {
	return r.db.WithContext(ctx).Model(&model.HotSearch{}).Where("keyword = ?", keyword).Updates(map[string]interface{}{
		"search_count": count,
		"hot_score":    float64(count) * 0.1, // 简化的热度计算
	}).Error
}
