package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HashtagService 话题服务层接口
type HashtagService interface {
	// CreateHashtag 创建话题
	CreateHashtag(ctx context.Context, req *CreateHashtagRequest) (*model.Hashtag, error)
	// GetHashtagDetail 获取话题详情
	GetHashtagDetail(ctx context.Context, id uint) (*model.Hashtag, error)
	// GetHotHashtags 获取热门话题
	GetHotHashtags(ctx context.Context, limit int) ([]*model.Hashtag, error)
	// GetHashtagVideos 获取话题下的视频
	GetHashtagVideos(ctx context.Context, hashtagID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetOrCreateHashtagByName 根据名称获取或创建话题
	GetOrCreateHashtagByName(ctx context.Context, name string) (*model.Hashtag, error)
	// AddVideoToHashtag 将视频添加到话题
	AddVideoToHashtag(ctx context.Context, videoID uint, hashtagNames []string) error
}

// hashtagServiceImpl 话题服务层实现
type hashtagServiceImpl struct {
	hashtagRepo repository.HashtagRepository
}

// NewHashtagService 创建话题服务实例
func NewHashtagService(hashtagRepo repository.HashtagRepository) HashtagService {
	return &hashtagServiceImpl{
		hashtagRepo: hashtagRepo,
	}
}

// CreateHashtagRequest 创建话题请求
type CreateHashtagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
}

// CreateHashtag 创建话题
func (s *hashtagServiceImpl) CreateHashtag(ctx context.Context, req *CreateHashtagRequest) (*model.Hashtag, error) {
	logger.Info("创建话题", zap.String("name", req.Name))

	// 检查话题是否已存在
	existing, err := s.hashtagRepo.GetHashtagByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, errors.New("话题已存在")
	}

	hashtag := &model.Hashtag{
		Name:     req.Name,
		HotScore: 1.0,
	}

	if err := s.hashtagRepo.CreateHashtag(ctx, hashtag); err != nil {
		logger.Error("创建话题失败", zap.Error(err))
		return nil, errors.New("创建话题失败")
	}

	return hashtag, nil
}

// GetHashtagDetail 获取话题详情
func (s *hashtagServiceImpl) GetHashtagDetail(ctx context.Context, id uint) (*model.Hashtag, error) {
	logger.Info("获取话题详情", zap.Uint("id", id))

	hashtag, err := s.hashtagRepo.GetHashtagByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("话题不存在")
		}
		logger.Error("获取话题详情失败", zap.Error(err))
		return nil, errors.New("获取话题详情失败")
	}

	// 增加浏览量（异步）
	go func() {
		if err := s.hashtagRepo.IncrementViewCount(context.Background(), id); err != nil {
			logger.Error("增加话题浏览量失败", zap.Error(err))
		}
	}()

	return hashtag, nil
}

// GetHotHashtags 获取热门话题
func (s *hashtagServiceImpl) GetHotHashtags(ctx context.Context, limit int) ([]*model.Hashtag, error) {
	logger.Info("获取热门话题", zap.Int("limit", limit))

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	return s.hashtagRepo.GetHotHashtags(ctx, limit)
}

// GetHashtagVideos 获取话题下的视频
func (s *hashtagServiceImpl) GetHashtagVideos(ctx context.Context, hashtagID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Info("获取话题视频", zap.Uint("hashtag_id", hashtagID))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.hashtagRepo.GetHashtagVideos(ctx, hashtagID, page, pageSize)
}

// GetOrCreateHashtagByName 根据名称获取或创建话题
func (s *hashtagServiceImpl) GetOrCreateHashtagByName(ctx context.Context, name string) (*model.Hashtag, error) {
	hashtag, err := s.hashtagRepo.GetHashtagByName(ctx, name)
	if err == nil {
		return hashtag, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 创建新话题
	hashtag = &model.Hashtag{
		Name:     name,
		HotScore: 1.0,
	}

	if err := s.hashtagRepo.CreateHashtag(ctx, hashtag); err != nil {
		return nil, err
	}

	return hashtag, nil
}

// AddVideoToHashtag 将视频添加到话题
func (s *hashtagServiceImpl) AddVideoToHashtag(ctx context.Context, videoID uint, hashtagNames []string) error {
	logger.Info("添加视频到话题", zap.Uint("video_id", videoID), zap.Strings("hashtags", hashtagNames))

	for _, name := range hashtagNames {
		// 获取或创建话题
		hashtag, err := s.GetOrCreateHashtagByName(ctx, name)
		if err != nil {
			logger.Error("获取或创建话题失败", zap.Error(err), zap.String("name", name))
			continue
		}

		// 添加关联
		if err := s.hashtagRepo.AddVideoToHashtag(ctx, videoID, hashtag.ID); err != nil {
			logger.Error("添加视频到话题失败", zap.Error(err))
			continue
		}
	}

	return nil
}
