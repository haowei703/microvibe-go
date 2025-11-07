package service

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// SearchService 搜索服务层接口
type SearchService interface {
	// Search 综合搜索
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	// SearchVideos 搜索视频
	SearchVideos(ctx context.Context, keyword string, page, pageSize int) ([]*model.Video, int64, error)
	// SearchUsers 搜索用户
	SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error)
	// SearchHashtags 搜索话题
	SearchHashtags(ctx context.Context, keyword string, page, pageSize int) ([]*model.Hashtag, int64, error)
	// GetSearchHistory 获取搜索历史
	GetSearchHistory(ctx context.Context, userID uint, limit int) ([]*model.SearchHistory, error)
	// ClearSearchHistory 清空搜索历史
	ClearSearchHistory(ctx context.Context, userID uint) error
	// GetHotSearches 获取热搜榜
	GetHotSearches(ctx context.Context, limit int) ([]*model.HotSearch, error)
	// GetSearchSuggestions 获取搜索建议
	GetSearchSuggestions(ctx context.Context, keyword string, limit int) ([]string, error)
}

// searchServiceImpl 搜索服务层实现
type searchServiceImpl struct {
	searchRepo repository.SearchRepository
}

// NewSearchService 创建搜索服务实例
func NewSearchService(searchRepo repository.SearchRepository) SearchService {
	return &searchServiceImpl{
		searchRepo: searchRepo,
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword  string `json:"keyword" binding:"required,min=1,max=100"`
	Category string `json:"category"` // video, user, hashtag, all
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	UserID   *uint  `json:"user_id"` // 可选，用于记录搜索历史
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Videos   []*model.Video   `json:"videos,omitempty"`
	Users    []*model.User    `json:"users,omitempty"`
	Hashtags []*model.Hashtag `json:"hashtags,omitempty"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// Search 综合搜索
func (s *searchServiceImpl) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	logger.Info("搜索请求", zap.String("keyword", req.Keyword), zap.String("category", req.Category))

	// 记录搜索历史（异步，不影响主流程）
	if req.UserID != nil {
		go func() {
			history := &model.SearchHistory{
				UserID:   *req.UserID,
				Keyword:  req.Keyword,
				Category: req.Category,
			}
			if err := s.searchRepo.CreateSearchHistory(context.Background(), history); err != nil {
				logger.Error("记录搜索历史失败", zap.Error(err))
			}
		}()
	}

	// 更新热搜（异步）
	go func() {
		if err := s.searchRepo.IncrementSearchCount(context.Background(), req.Keyword); err != nil {
			logger.Error("更新热搜失败", zap.Error(err))
		}
	}()

	resp := &SearchResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 根据分类搜索
	switch req.Category {
	case "video":
		videos, total, err := s.searchRepo.SearchVideos(ctx, req.Keyword, req.Page, req.PageSize)
		if err != nil {
			logger.Error("搜索视频失败", zap.Error(err))
			return nil, err
		}
		resp.Videos = videos
		resp.Total = total

	case "user":
		users, total, err := s.searchRepo.SearchUsers(ctx, req.Keyword, req.Page, req.PageSize)
		if err != nil {
			logger.Error("搜索用户失败", zap.Error(err))
			return nil, err
		}
		resp.Users = users
		resp.Total = total

	case "hashtag":
		hashtags, total, err := s.searchRepo.SearchHashtags(ctx, req.Keyword, req.Page, req.PageSize)
		if err != nil {
			logger.Error("搜索话题失败", zap.Error(err))
			return nil, err
		}
		resp.Hashtags = hashtags
		resp.Total = total

	default: // "all" 或空
		// 综合搜索：返回各类型的前几条结果
		videos, _, err := s.searchRepo.SearchVideos(ctx, req.Keyword, 1, 10)
		if err != nil {
			logger.Error("搜索视频失败", zap.Error(err))
		} else {
			resp.Videos = videos
		}

		users, _, err := s.searchRepo.SearchUsers(ctx, req.Keyword, 1, 5)
		if err != nil {
			logger.Error("搜索用户失败", zap.Error(err))
		} else {
			resp.Users = users
		}

		hashtags, _, err := s.searchRepo.SearchHashtags(ctx, req.Keyword, 1, 5)
		if err != nil {
			logger.Error("搜索话题失败", zap.Error(err))
		} else {
			resp.Hashtags = hashtags
		}

		resp.Total = int64(len(videos) + len(users) + len(hashtags))
	}

	return resp, nil
}

// SearchVideos 搜索视频
func (s *searchServiceImpl) SearchVideos(ctx context.Context, keyword string, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Info("搜索视频", zap.String("keyword", keyword))
	return s.searchRepo.SearchVideos(ctx, keyword, page, pageSize)
}

// SearchUsers 搜索用户
func (s *searchServiceImpl) SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	logger.Info("搜索用户", zap.String("keyword", keyword))
	return s.searchRepo.SearchUsers(ctx, keyword, page, pageSize)
}

// SearchHashtags 搜索话题
func (s *searchServiceImpl) SearchHashtags(ctx context.Context, keyword string, page, pageSize int) ([]*model.Hashtag, int64, error) {
	logger.Info("搜索话题", zap.String("keyword", keyword))
	return s.searchRepo.SearchHashtags(ctx, keyword, page, pageSize)
}

// GetSearchHistory 获取搜索历史
func (s *searchServiceImpl) GetSearchHistory(ctx context.Context, userID uint, limit int) ([]*model.SearchHistory, error) {
	logger.Info("获取搜索历史", zap.Uint("user_id", userID))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.searchRepo.GetUserSearchHistory(ctx, userID, limit)
}

// ClearSearchHistory 清空搜索历史
func (s *searchServiceImpl) ClearSearchHistory(ctx context.Context, userID uint) error {
	logger.Info("清空搜索历史", zap.Uint("user_id", userID))
	return s.searchRepo.DeleteUserSearchHistory(ctx, userID)
}

// GetHotSearches 获取热搜榜
func (s *searchServiceImpl) GetHotSearches(ctx context.Context, limit int) ([]*model.HotSearch, error) {
	logger.Info("获取热搜榜", zap.Int("limit", limit))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.searchRepo.GetHotSearches(ctx, limit)
}

// GetSearchSuggestions 获取搜索建议（基于热搜和历史）
func (s *searchServiceImpl) GetSearchSuggestions(ctx context.Context, keyword string, limit int) ([]string, error) {
	logger.Info("获取搜索建议", zap.String("keyword", keyword))

	if limit <= 0 || limit > 10 {
		limit = 10
	}

	// 从热搜中匹配
	hotSearches, err := s.searchRepo.GetHotSearches(ctx, 50)
	if err != nil {
		return nil, err
	}

	suggestions := make([]string, 0, limit)
	for _, hs := range hotSearches {
		if len(hs.Keyword) >= len(keyword) && hs.Keyword[:len(keyword)] == keyword {
			suggestions = append(suggestions, hs.Keyword)
			if len(suggestions) >= limit {
				break
			}
		}
	}

	return suggestions, nil
}
