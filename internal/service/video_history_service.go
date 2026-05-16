package service

import (
	"context"
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// VideoHistoryService 视频播放历史服务层接口
type VideoHistoryService interface {
	// ReportProgress 上报播放进度，返回是否已处理为”已看完”
	ReportProgress(ctx context.Context, userID, videoID uint, position, duration int, finished bool) (bool, error)
	// GetHistory 获取用户播放历史
	GetHistory(ctx context.Context, userID uint, page, pageSize int, finished *bool) ([]*model.VideoHistoryVO, int64, error)
	// DeleteHistory 删除单条历史
	DeleteHistory(ctx context.Context, userID, historyID uint) error
	// ClearHistory 清空历史
	ClearHistory(ctx context.Context, userID uint) error
}

type videoHistoryServiceImpl struct {
	historyRepo     repository.VideoHistoryRepository
	behaviorRepo    repository.BehaviorRepository
	likeRepo        repository.LikeRepository
	favoriteRepo    repository.FavoriteRepository
	followRepo      repository.FollowRepository
	statsService    VideoStatsService
	recommendEngine *recommend.Engine
}

// NewVideoHistoryService 创建视频播放历史服务实例
func NewVideoHistoryService(
	historyRepo repository.VideoHistoryRepository,
	behaviorRepo repository.BehaviorRepository,
	likeRepo repository.LikeRepository,
	favoriteRepo repository.FavoriteRepository,
	followRepo repository.FollowRepository,
) VideoHistoryService {
	return &videoHistoryServiceImpl{
		historyRepo:  historyRepo,
		behaviorRepo: behaviorRepo,
		likeRepo:     likeRepo,
		favoriteRepo: favoriteRepo,
		followRepo:   followRepo,
	}
}

// SetStatsService 设置统计服务（延迟注入避免循环依赖）
func (s *videoHistoryServiceImpl) SetStatsService(statsService VideoStatsService) {
	s.statsService = statsService
}

// SetRecommendEngine 设置推荐引擎
func (s *videoHistoryServiceImpl) SetRecommendEngine(engine *recommend.Engine) {
	s.recommendEngine = engine
}

// ReportProgress 上报播放进度
func (s *videoHistoryServiceImpl) ReportProgress(ctx context.Context, userID, videoID uint, position, duration int, finished bool) (bool, error) {
	logger.Debug("上报播放进度", zap.Uint("user_id", userID), zap.Uint("video_id", videoID), zap.Int("position", position))

	// 1. 如果 position 不足但已经达到总时长的 90%，也自动认为完成
	if !finished && duration > 0 && float64(position)/float64(duration) >= 0.9 {
		finished = true
	}

	// 2. 更新或创建历史记录
	history := &model.VideoHistory{
		UserID:   userID,
		VideoID:  videoID,
		Position: position,
		Duration: duration,
		Finished: finished,
	}

	if err := s.historyRepo.Upsert(ctx, history); err != nil {
		return false, err
	}

	// 3. 记录行为日志（用于推荐算法）
	// Action 类型：1-浏览，2-点赞，3-评论，4-分享，5-收藏，6-完播
	action := int8(1) // 默认浏览
	if finished {
		action = 6 // 完播
	}

	behavior := &model.UserBehavior{
		UserID:   userID,
		VideoID:  videoID,
		Action:   action,
		Duration: position,
		Progress: 0, // 可以根据 position/duration 计算百分比
	}
	if duration > 0 {
		behavior.Progress = int((float64(position) / float64(duration)) * 100)
	}

	// 异步记录行为日志，不阻塞进度返回
	go func() {
		bgCtx := context.Background()
		_ = s.behaviorRepo.Create(bgCtx, behavior)

		// 更新用户画像 (兴趣分数)
		if s.recommendEngine != nil {
			_ = s.recommendEngine.UpdateUserProfile(bgCtx, userID, behavior)
			s.recommendEngine.RecordWatched(bgCtx, userID, videoID)
		}

		// 更新视频统计数据
		if s.statsService != nil {
			_ = s.statsService.RecordPlay(bgCtx, videoID, duration, position, finished)
		}
	}()

	return finished, nil
}

// GetHistory 获取用户播放历史
func (s *videoHistoryServiceImpl) GetHistory(ctx context.Context, userID uint, page, pageSize int, finished *bool) ([]*model.VideoHistoryVO, int64, error) {
	histories, total, err := s.historyRepo.FindByUserID(ctx, userID, page, pageSize, finished)
	if err != nil {
		return nil, 0, err
	}

	vos := make([]*model.VideoHistoryVO, 0, len(histories))
	for _, h := range histories {
		vos = append(vos, s.toVideoHistoryVO(ctx, h, userID))
	}
	return vos, total, nil
}

// toVideoHistoryVO 将 VideoHistory 转换为 VideoHistoryVO（附带互动状态）
func (s *videoHistoryServiceImpl) toVideoHistoryVO(ctx context.Context, h *model.VideoHistory, currentUserID uint) *model.VideoHistoryVO {
	if h == nil {
		return nil
	}

	var videoVO *model.VideoVO
	if h.Video != nil {
		authorVO := &model.AuthorVO{
			ID: h.Video.UserID,
		}
		if h.Video.User != nil {
			authorVO = h.Video.User.ToAuthorVO()
		}

		isLiked := false
		isFavorited := false
		isFollowed := false
		if currentUserID > 0 {
			isLiked, _ = s.likeRepo.Exists(ctx, currentUserID, h.Video.ID)
			isFavorited, _ = s.favoriteRepo.Exists(ctx, currentUserID, h.Video.ID)
			isFollowed, _ = s.followRepo.Exists(ctx, currentUserID, h.Video.UserID)
			authorVO.IsFollowed = isFollowed
		}

		videoVO = &model.VideoVO{
			Video:       h.Video,
			User:        authorVO,
			IsLiked:     isLiked,
			IsFavorited: isFavorited,
			IsFollowed:  isFollowed,
		}
	}

	return &model.VideoHistoryVO{
		ID:        h.ID,
		CreatedAt: h.CreatedAt,
		UpdatedAt: h.UpdatedAt,
		UserID:    h.UserID,
		VideoID:   h.VideoID,
		Position:  h.Position,
		Duration:  h.Duration,
		Finished:  h.Finished,
		Video:     videoVO,
	}
}

// DeleteHistory 删除单条历史
func (s *videoHistoryServiceImpl) DeleteHistory(ctx context.Context, userID, historyID uint) error {
	return s.historyRepo.Delete(ctx, userID, historyID)
}

// ClearHistory 清空历史
func (s *videoHistoryServiceImpl) ClearHistory(ctx context.Context, userID uint) error {
	return s.historyRepo.ClearAll(ctx, userID)
}
