package service

import (
	"context"
	"errors"
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
)

type ShareService interface {
	ShareVideo(ctx context.Context, userID, videoID uint, platform string) error
	GetVideoShareCount(ctx context.Context, videoID uint) (int64, error)
}

type shareServiceImpl struct {
	shareRepo       repository.ShareRepository
	videoRepo       repository.VideoRepository
	statsService    VideoStatsService
	recommendEngine *recommend.Engine
}

func NewShareService(shareRepo repository.ShareRepository, videoRepo repository.VideoRepository) ShareService {
	return &shareServiceImpl{
		shareRepo: shareRepo,
		videoRepo: videoRepo,
	}
}

// SetStatsService 设置统计服务（延迟注入避免循环依赖）
func (s *shareServiceImpl) SetStatsService(statsService VideoStatsService) {
	s.statsService = statsService
}

// SetRecommendEngine 设置推荐引擎
func (s *shareServiceImpl) SetRecommendEngine(engine *recommend.Engine) {
	s.recommendEngine = engine
}

func (s *shareServiceImpl) ShareVideo(ctx context.Context, userID, videoID uint, platform string) error {
	// 检查视频是否存在
	if _, err := s.videoRepo.FindByID(ctx, videoID); err != nil {
		return errors.New("视频不存在")
	}

	share := &model.Share{
		UserID:   userID,
		VideoID:  videoID,
		Platform: platform,
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return err
	}

	// 更新统计数据（异步）
	if s.statsService != nil {
		go func() { _ = s.statsService.RecordShare(context.Background(), videoID) }()
	}

	// 更新用户兴趣画像（异步）
	if s.recommendEngine != nil {
		go func() {
			_ = s.recommendEngine.UpdateUserProfile(context.Background(), userID, &model.UserBehavior{
				UserID:  userID,
				VideoID: videoID,
				Action:  4, // 4-分享
			})
		}()
	}

	return nil
}

func (s *shareServiceImpl) GetVideoShareCount(ctx context.Context, videoID uint) (int64, error) {
	return s.shareRepo.CountByVideoID(ctx, videoID)
}
