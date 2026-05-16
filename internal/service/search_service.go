package service

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"
	"strings"

	"go.uber.org/zap"
)

// SearchService 搜索服务层接口
type SearchService interface {
	// Search 综合搜索
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	// SearchVideos 搜索视频
	SearchVideos(ctx context.Context, keyword string, userID uint, page, pageSize int) ([]*model.VideoVO, int64, error)
	// SearchUsers 搜索用户
	SearchUsers(ctx context.Context, keyword string, userID uint, page, pageSize int) ([]*model.UserVO, int64, error)
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
	searchRepo   repository.SearchRepository
	followRepo   repository.FollowRepository
	likeRepo     repository.LikeRepository
	favoriteRepo repository.FavoriteRepository
}

// NewSearchService 创建搜索服务实例
func NewSearchService(
	searchRepo repository.SearchRepository,
	followRepo repository.FollowRepository,
	likeRepo repository.LikeRepository,
	favoriteRepo repository.FavoriteRepository,
) SearchService {
	return &searchServiceImpl{
		searchRepo:   searchRepo,
		followRepo:   followRepo,
		likeRepo:     likeRepo,
		favoriteRepo: favoriteRepo,
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
	Videos        []*model.VideoVO `json:"videos,omitempty"`
	Hashtags      []*model.Hashtag `json:"hashtags,omitempty"`
	BestMatchUser *UserWithVideos  `json:"best_match_user,omitempty"`
	Total         int64            `json:"total"`
	Page          int              `json:"page"`
	PageSize      int              `json:"page_size"`
}

// UserWithVideos 最佳匹配用户及其视频
type UserWithVideos struct {
	User   *model.UserVO    `json:"user"`
	Videos []*model.VideoVO `json:"videos"`
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
		var currentUserID uint
		if req.UserID != nil {
			currentUserID = *req.UserID
		}
		resp.Videos = s.toVideoVOList(ctx, videos, currentUserID)
		resp.Total = total

	case "user":
		// category=user 时，将最佳匹配用户附带视频返回
		bestUser, err := s.searchRepo.SearchBestMatchUser(ctx, req.Keyword)
		if err != nil {
			logger.Error("搜索最佳匹配用户失败", zap.Error(err))
			return nil, err
		}
		if bestUser != nil {
			userVideos, err := s.searchRepo.GetUserRecentVideos(ctx, bestUser.ID, 6)
			if err != nil {
				logger.Error("获取用户视频失败", zap.Error(err))
			} else {
				// 获取当前用户ID
				var currentUserID uint
				if req.UserID != nil {
					currentUserID = *req.UserID
				}

				resp.BestMatchUser = &UserWithVideos{
					User:   s.toUserVO(ctx, bestUser, currentUserID),
					Videos: s.toVideoVOList(ctx, userVideos, currentUserID),
				}
			}
			resp.Total = 1
		}

	case "hashtag":
		hashtags, total, err := s.searchRepo.SearchHashtags(ctx, req.Keyword, req.Page, req.PageSize)
		if err != nil {
			logger.Error("搜索话题失败", zap.Error(err))
			return nil, err
		}
		resp.Hashtags = hashtags
		resp.Total = total

	default: // "all" 或空
		videos, _, err := s.searchRepo.SearchVideos(ctx, req.Keyword, 1, 10)
		if err != nil {
			logger.Error("搜索视频失败", zap.Error(err))
		} else {
			var currentUserID uint
			if req.UserID != nil {
				currentUserID = *req.UserID
			}
			resp.Videos = s.toVideoVOList(ctx, videos, currentUserID)
		}

		hashtags, _, err := s.searchRepo.SearchHashtags(ctx, req.Keyword, 1, 5)
		if err != nil {
			logger.Error("搜索话题失败", zap.Error(err))
		} else {
			resp.Hashtags = hashtags
		}

		// 最佳匹配用户附带其视频
		bestUser, err := s.searchRepo.SearchBestMatchUser(ctx, req.Keyword)
		if err != nil {
			logger.Error("搜索最佳匹配用户失败", zap.Error(err))
		} else if bestUser != nil {
			userVideos, err := s.searchRepo.GetUserRecentVideos(ctx, bestUser.ID, 6)
			if err != nil {
				logger.Error("获取用户视频失败", zap.Error(err))
			} else {
				// 获取当前用户ID
				var currentUserID uint
				if req.UserID != nil {
					currentUserID = *req.UserID
				}

				resp.BestMatchUser = &UserWithVideos{
					User:   s.toUserVO(ctx, bestUser, currentUserID),
					Videos: s.toVideoVOList(ctx, userVideos, currentUserID),
				}
			}
		}

		resp.Total = int64(len(resp.Videos) + len(resp.Hashtags))
	}

	return resp, nil
}

// SearchVideos 搜索视频
func (s *searchServiceImpl) SearchVideos(ctx context.Context, keyword string, userID uint, page, pageSize int) ([]*model.VideoVO, int64, error) {
	logger.Info("搜索视频", zap.String("keyword", keyword))
	videos, total, err := s.searchRepo.SearchVideos(ctx, keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return s.toVideoVOList(ctx, videos, userID), total, nil
}

// toVideoVO 将 Video 转换为 VideoVO
func (s *searchServiceImpl) toVideoVO(ctx context.Context, video *model.Video, currentUserID uint) *model.VideoVO {
	if video == nil {
		return nil
	}

	authorVO := &model.AuthorVO{
		ID: video.UserID,
	}

	isFollowed := false
	isLiked := false
	isFavorited := false
	if video.User != nil {
		authorVO = video.User.ToAuthorVO()
	}
	if currentUserID > 0 {
		isFollowed, _ = s.followRepo.Exists(ctx, currentUserID, video.UserID)
		isLiked, _ = s.likeRepo.Exists(ctx, currentUserID, video.ID)
		isFavorited, _ = s.favoriteRepo.Exists(ctx, currentUserID, video.ID)
		authorVO.IsFollowed = isFollowed
	}

	return &model.VideoVO{
		Video:       video,
		User:        authorVO,
		IsLiked:     isLiked,
		IsFavorited: isFavorited,
		IsFollowed:  isFollowed,
	}
}

// toVideoVOList 将 Video 列表转换为 VideoVO 列表
func (s *searchServiceImpl) toVideoVOList(ctx context.Context, videos []*model.Video, currentUserID uint) []*model.VideoVO {
	vos := make([]*model.VideoVO, 0, len(videos))
	for _, video := range videos {
		vos = append(vos, s.toVideoVO(ctx, video, currentUserID))
	}
	return vos
}

// SearchUsers 搜索用户 (当关键字为空时返回推荐)
func (s *searchServiceImpl) SearchUsers(ctx context.Context, keyword string, userID uint, page, pageSize int) ([]*model.UserVO, int64, error) {
	var users []*model.User
	var total int64
	var err error

	if keyword == "" {
		logger.Info("关键词为空，返回推荐用户", zap.Uint("user_id", userID))
		users, total, err = s.searchRepo.RecommendUsers(ctx, userID, page, pageSize)
	} else {
		logger.Info("搜索用户", zap.String("keyword", keyword))
		users, total, err = s.searchRepo.SearchUsers(ctx, keyword, page, pageSize)
	}

	if err != nil {
		return nil, 0, err
	}

	return s.toUserVOList(ctx, users, userID), total, nil
}

// toUserVO 将 User 转换为 UserVO
func (s *searchServiceImpl) toUserVO(ctx context.Context, user *model.User, currentUserID uint) *model.UserVO {
	if user == nil {
		return nil
	}
	vo := &model.UserVO{
		User: user,
	}
	if currentUserID > 0 {
		isFollowed, _ := s.followRepo.Exists(ctx, currentUserID, user.ID)
		vo.IsFollowed = isFollowed
	}
	return vo
}

// toUserVOList 将 User 列表转换为 UserVO 列表
func (s *searchServiceImpl) toUserVOList(ctx context.Context, users []*model.User, currentUserID uint) []*model.UserVO {
	vos := make([]*model.UserVO, 0, len(users))
	for _, user := range users {
		vos = append(vos, s.toUserVO(ctx, user, currentUserID))
	}
	return vos
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

// GetSearchSuggestions 获取搜索建议（基于用户、话题和热搜）
func (s *searchServiceImpl) GetSearchSuggestions(ctx context.Context, keyword string, limit int) ([]string, error) {
	logger.Info("获取搜索建议", zap.String("keyword", keyword))

	if limit <= 0 || limit > 10 {
		limit = 10
	}

	suggestions := make([]string, 0, limit)
	seen := make(map[string]bool)

	// 1. 用户建议 (用户名、昵称)
	users, _ := s.searchRepo.GetSuggestUsers(ctx, keyword, limit)
	for _, u := range users {
		item := u.Nickname
		if item == "" {
			item = u.Username
		}
		if !seen[item] {
			suggestions = append(suggestions, item)
			seen[item] = true
			if len(suggestions) >= limit {
				return suggestions, nil
			}
		}
	}

	// 2. 话题建议 (#开头)
	hashtags, _ := s.searchRepo.GetSuggestHashtags(ctx, keyword, limit)
	for _, h := range hashtags {
		item := "#" + h.Name
		if !seen[item] {
			suggestions = append(suggestions, item)
			seen[item] = true
			if len(suggestions) >= limit {
				return suggestions, nil
			}
		}
	}

	// 3. 原有的热搜建议
	hotSearches, _ := s.searchRepo.GetHotSearches(ctx, 50)
	for _, hs := range hotSearches {
		if strings.HasPrefix(strings.ToLower(hs.Keyword), strings.ToLower(keyword)) {
			if !seen[hs.Keyword] {
				suggestions = append(suggestions, hs.Keyword)
				seen[hs.Keyword] = true
				if len(suggestions) >= limit {
					return suggestions, nil
				}
			}
		}
	}

	return suggestions, nil
}
