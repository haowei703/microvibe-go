package service

import (
	"context"
	"errors"
	"fmt"
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/config"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"
	"strings"
	"time"

	"go.uber.org/zap"
)

// VideoService 视频服务层接口
type VideoService interface {
	// CreateVideo 创建视频
	CreateVideo(ctx context.Context, req *CreateVideoRequest) (*model.Video, error)
	// GetVideoByID 获取视频详情
	GetVideoByID(ctx context.Context, videoID uint) (*model.Video, error)
	// GetUserVideos 获取用户的视频列表（公开作品）
	GetUserVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetMyVideos 获取用户自己的所有视频列表（包含私密/审核中作品）
	GetMyVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.MyVideoVO, int64, error)
	// GetUserFavoriteVideos 获取用户收藏的视频列表
	GetUserFavoriteVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetUserFavoriteVideosWithPrivacy 获取用户收藏的视频列表（带隐私检查）
	GetUserFavoriteVideosWithPrivacy(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetUserLikedVideos 获取用户点赞的视频列表
	GetUserLikedVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetUserLikedVideosWithPrivacy 获取用户点赞的视频列表（带隐私检查）
	GetUserLikedVideosWithPrivacy(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.Video, int64, error)
	// GetHotVideos 获取热门视频
	GetHotVideos(ctx context.Context, page, pageSize int) ([]*model.Video, error)
	// UpdateVideo 更新视频信息
	UpdateVideo(ctx context.Context, userID, videoID uint, updates map[string]interface{}) error
	// DeleteVideo 删除视频
	DeleteVideo(ctx context.Context, userID, videoID uint) error
	// LikeVideo 点赞视频
	LikeVideo(ctx context.Context, userID, videoID uint) error
	// UnlikeVideo 取消点赞
	UnlikeVideo(ctx context.Context, userID, videoID uint) error
	// FavoriteVideo 收藏视频
	FavoriteVideo(ctx context.Context, userID, videoID uint) error
	// UnfavoriteVideo 取消收藏
	UnfavoriteVideo(ctx context.Context, userID, videoID uint) error
	// UpdateVideoStatus 更新视频状态 (审核)
	UpdateVideoStatus(ctx context.Context, videoID uint, status int8) error
	// RecordPlay 记录一次播放（每次新播放会话调用，用于统计播放量）
	RecordPlay(ctx context.Context, videoID uint) error
	// EnrichVideoList 丰富视频信息 (点赞、收藏、关注状态)
	EnrichVideoList(ctx context.Context, userID uint, videos []*model.Video) ([]*model.VideoVO, error)
	// EnrichVideo 丰富单个视频信息
	EnrichVideo(ctx context.Context, userID uint, video *model.Video) (*model.VideoVO, error)
	// GetVideoLikers 获取点赞视频的用户列表
	GetVideoLikers(ctx context.Context, videoID uint, page, pageSize int) ([]*model.AuthorVO, int64, error)
	// GetVideoFavoriters 获取收藏视频的用户列表
	GetVideoFavoriters(ctx context.Context, videoID uint, page, pageSize int) ([]*model.AuthorVO, int64, error)
}

// videoServiceImpl 视频服务层实现
type videoServiceImpl struct {
	videoRepo       repository.VideoRepository
	likeRepo        repository.LikeRepository
	favoriteRepo    repository.FavoriteRepository
	followRepo      repository.FollowRepository
	userRepo        repository.UserRepository
	cfg             *config.Config
	hashtagService  HashtagService
	messageService  MessageService
	statsService    VideoStatsService
	recommendEngine *recommend.Engine
}

// NewVideoService 创建视频服务实例
func NewVideoService(
	videoRepo repository.VideoRepository,
	likeRepo repository.LikeRepository,
	favoriteRepo repository.FavoriteRepository,
	followRepo repository.FollowRepository,
	cfg *config.Config,
) VideoService {
	return &videoServiceImpl{
		videoRepo:    videoRepo,
		likeRepo:     likeRepo,
		favoriteRepo: favoriteRepo,
		followRepo:   followRepo,
		cfg:          cfg,
	}
}

// SetUserRepo 设置用户 Repository（延迟注入，用于隐私检查）
func (s *videoServiceImpl) SetUserRepo(userRepo repository.UserRepository) {
	s.userRepo = userRepo
}

// SetHashtagService 设置话题服务（用于延迟注入避免循环依赖）
func (s *videoServiceImpl) SetHashtagService(hashtagService HashtagService) {
	s.hashtagService = hashtagService
}

// SetMessageService 设置消息服务 (用于针对延迟注入)
func (s *videoServiceImpl) SetMessageService(messageService MessageService) {
	s.messageService = messageService
}

// SetRecommendEngine 设置推荐引擎
func (s *videoServiceImpl) SetRecommendEngine(engine *recommend.Engine) {
	s.recommendEngine = engine
}

// SetStatsService 设置统计服务（延迟注入避免循环依赖）
func (s *videoServiceImpl) SetStatsService(statsService VideoStatsService) {
	s.statsService = statsService
}

// CreateVideoRequest 创建视频请求
type CreateVideoRequest struct {
	UserID       uint     `json:"user_id"`
	Title        string   `json:"title" binding:"required,max=200"`
	Description  string   `json:"description" binding:"max=1000"`
	VideoURL     string   `json:"video_url" binding:"required"`
	CoverURL     string   `json:"cover_url" binding:"required"`
	Duration     int      `json:"duration" binding:"required,min=1"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	FileSize     int64    `json:"file_size"`
	CategoryID   *uint    `json:"category_id"`
	Tags         []string `json:"tags"`
	IsPublic     *bool    `json:"is_public"`
	AllowComment *bool    `json:"allow_comment"`
}

// CreateVideo 创建视频
func (s *videoServiceImpl) CreateVideo(ctx context.Context, req *CreateVideoRequest) (*model.Video, error) {
	logger.Info("创建视频", zap.String("title", req.Title))

	// 构建视频对象
	now := time.Now()
	video := &model.Video{
		UserID:       req.UserID,
		Title:        req.Title,
		Description:  req.Description,
		VideoURL:     req.VideoURL,
		CoverURL:     req.CoverURL,
		Duration:     req.Duration,
		Width:        req.Width,
		Height:       req.Height,
		FileSize:     req.FileSize,
		CategoryID:   req.CategoryID,
		Status:       0, // 0-审核中
		IsPublic:     true,
		AllowComment: true,
		PublishedAt:  &now,
	}

	if req.IsPublic != nil {
		video.IsPublic = *req.IsPublic
	}
	if req.AllowComment != nil {
		video.AllowComment = *req.AllowComment
	}

	// 处理标签 - 将标签数组转换为逗号分隔的字符串
	if len(req.Tags) > 0 {
		video.Tags = strings.Join(req.Tags, ",")
	}

	if err := s.videoRepo.Create(ctx, video); err != nil {
		logger.Error("创建视频失败", zap.Error(err))
		return nil, errors.New("发布失败")
	}

	// 异步任务：关联话题
	go func() {
		bgCtx := context.Background()

		// 1. 关联话题
		if len(req.Tags) > 0 && s.hashtagService != nil {
			if err := s.hashtagService.AddVideoToHashtag(bgCtx, video.ID, req.Tags); err != nil {
				logger.Error("关联话题失败", zap.Error(err), zap.Uint("video_id", video.ID))
			}
		}
	}()

	logger.Info("视频发布申请已提交", zap.Uint("video_id", video.ID))
	return video, nil
}

// GetVideoByID 获取视频详情
func (s *videoServiceImpl) GetVideoByID(ctx context.Context, videoID uint) (*model.Video, error) {
	logger.Debug("获取视频详情", zap.Uint("video_id", videoID))

	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, errors.New("视频不存在")
		}
		logger.Error("获取视频详情失败", zap.Error(err), zap.Uint("video_id", videoID))
		return nil, err
	}

	return video, nil
}

// RecordPlay 记录一次播放（每次新播放会话调用，用于统计播放量）
func (s *videoServiceImpl) RecordPlay(ctx context.Context, videoID uint) error {
	logger.Debug("记录播放次数", zap.Uint("video_id", videoID))
	if err := s.videoRepo.IncrementPlayCount(ctx, videoID); err != nil {
		logger.Error("记录播放次数失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("记录播放失败")
	}
	return nil
}

// GetUserVideos 获取用户的视频列表（公开作品）
func (s *videoServiceImpl) GetUserVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Debug("获取用户视频列表", zap.Uint("user_id", userID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	videos, err := s.videoRepo.FindByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		logger.Error("获取用户视频列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	// 查询视频总数
	total, err := s.videoRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// GetMyVideos 获取用户自己的所有视频列表（包含私密/审核中作品）
func (s *videoServiceImpl) GetMyVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.MyVideoVO, int64, error) {
	logger.Debug("获取我自己的视频列表", zap.Uint("user_id", userID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	videos, err := s.videoRepo.FindAllByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		logger.Error("获取我自己的视频列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	// 查询视频总数
	total, err := s.videoRepo.CountAllByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// 丰富信息（包含互动用户采样）
	vos, err := s.EnrichMyVideoList(ctx, userID, videos)
	if err != nil {
		return nil, 0, err
	}

	return vos, total, nil
}

// GetUserFavoriteVideos 获取用户收藏的视频列表
func (s *videoServiceImpl) GetUserFavoriteVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Debug("获取用户收藏视频列表", zap.Uint("user_id", userID), zap.Int("page", page), zap.Int("page_size", pageSize))

	limit := pageSize
	offset := (page - 1) * pageSize

	favorites, err := s.favoriteRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		logger.Error("获取用户收藏列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	total, err := s.favoriteRepo.CountByUserID(ctx, userID)
	if err != nil {
		logger.Error("获取用户收藏总数失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	videos := make([]*model.Video, 0, len(favorites))
	for _, f := range favorites {
		if f.Video != nil {
			videos = append(videos, f.Video)
		}
	}

	return videos, total, nil
}

// GetUserFavoriteVideosWithPrivacy 获取用户收藏的视频列表（带隐私检查）
func (s *videoServiceImpl) GetUserFavoriteVideosWithPrivacy(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Debug("获取用户收藏视频列表（带隐私检查）", zap.Uint("target_user_id", targetUserID), zap.Uint("current_user_id", currentUserID))

	// 如果是查看自己的收藏，直接返回
	if targetUserID == currentUserID {
		return s.GetUserFavoriteVideos(ctx, targetUserID, page, pageSize)
	}

	// 查询目标用户的隐私设置
	targetUser, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.Error("查询用户失败", zap.Error(err), zap.Uint("user_id", targetUserID))
		return nil, 0, pkgerrors.ErrUserNotFound
	}

	// 检查隐私设置
	if !targetUser.ShowFavorites {
		logger.Warn("用户收藏列表未公开", zap.Uint("target_user_id", targetUserID))
		return nil, 0, pkgerrors.NewAppError(pkgerrors.CodeForbidden, "该用户的收藏列表未公开")
	}

	return s.GetUserFavoriteVideos(ctx, targetUserID, page, pageSize)
}

// GetUserLikedVideos 获取用户点赞的视频列表
func (s *videoServiceImpl) GetUserLikedVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Debug("获取用户点赞视频列表", zap.Uint("user_id", userID), zap.Int("page", page), zap.Int("page_size", pageSize))

	limit := pageSize
	offset := (page - 1) * pageSize

	likes, err := s.likeRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		logger.Error("获取用户点赞列表失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	total, err := s.likeRepo.CountByUserID(ctx, userID)
	if err != nil {
		logger.Error("获取用户点赞总数失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	videos := make([]*model.Video, 0, len(likes))
	for _, l := range likes {
		if l.Video != nil {
			videos = append(videos, l.Video)
		}
	}

	return videos, total, nil
}

// GetUserLikedVideosWithPrivacy 获取用户点赞的视频列表（带隐私检查）
func (s *videoServiceImpl) GetUserLikedVideosWithPrivacy(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.Video, int64, error) {
	logger.Debug("获取用户点赞视频列表（带隐私检查）", zap.Uint("target_user_id", targetUserID), zap.Uint("current_user_id", currentUserID))

	// 如果是查看自己的点赞，直接返回
	if targetUserID == currentUserID {
		return s.GetUserLikedVideos(ctx, targetUserID, page, pageSize)
	}

	// 查询目标用户的隐私设置
	targetUser, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.Error("查询用户失败", zap.Error(err), zap.Uint("user_id", targetUserID))
		return nil, 0, pkgerrors.ErrUserNotFound
	}

	// 检查隐私设置
	if !targetUser.ShowLikes {
		logger.Warn("用户点赞列表未公开", zap.Uint("target_user_id", targetUserID))
		return nil, 0, pkgerrors.NewAppError(pkgerrors.CodeForbidden, "该用户的点赞列表未公开")
	}

	return s.GetUserLikedVideos(ctx, targetUserID, page, pageSize)
}

// GetHotVideos 获取热门视频
func (s *videoServiceImpl) GetHotVideos(ctx context.Context, page, pageSize int) ([]*model.Video, error) {
	logger.Debug("获取热门视频", zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 查询最近7天的热门视频
	since := time.Now().AddDate(0, 0, -7)
	videos, err := s.videoRepo.FindHotVideos(ctx, since, pageSize, offset)
	if err != nil {
		logger.Error("获取热门视频失败", zap.Error(err))
		return nil, err
	}

	return videos, nil
}

// UpdateVideo 更新视频信息
func (s *videoServiceImpl) UpdateVideo(ctx context.Context, userID, videoID uint, updates map[string]interface{}) error {
	logger.Info("更新视频信息", zap.Uint("video_id", videoID))

	// 先查询视频
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return errors.New("视频不存在")
	}

	// 检查是否是视频所有者
	if video.UserID != userID {
		logger.Warn("非视频所有者尝试更新视频", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
		return errors.New("无权限修改该视频")
	}

	// 不允许更新的字段
	delete(updates, "id")
	delete(updates, "user_id")
	delete(updates, "video_url")
	delete(updates, "created_at")

	// 应用更新
	if title, ok := updates["title"].(string); ok {
		video.Title = title
	}
	if description, ok := updates["description"].(string); ok {
		video.Description = description
	}
	if coverURL, ok := updates["cover_url"].(string); ok {
		video.CoverURL = coverURL
	}

	if err := s.videoRepo.Update(ctx, video); err != nil {
		logger.Error("更新视频失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("更新视频失败")
	}

	logger.Info("视频更新成功", zap.Uint("video_id", videoID))
	return nil
}

// DeleteVideo 删除视频
func (s *videoServiceImpl) DeleteVideo(ctx context.Context, userID, videoID uint) error {
	logger.Info("删除视频", zap.Uint("video_id", videoID))

	// 先查询视频
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return errors.New("视频不存在")
	}

	// 检查是否是视频所有者
	if video.UserID != userID {
		logger.Warn("非视频所有者尝试删除视频", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
		return errors.New("无权限删除该视频")
	}

	if err := s.videoRepo.Delete(ctx, videoID); err != nil {
		logger.Error("删除视频失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("删除视频失败")
	}

	logger.Info("视频删除成功", zap.Uint("video_id", videoID))
	return nil
}

// LikeVideo 点赞视频
func (s *videoServiceImpl) LikeVideo(ctx context.Context, userID, videoID uint) error {
	logger.Info("点赞视频", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	// 检查是否已点赞
	exists, err := s.likeRepo.Exists(ctx, userID, videoID)
	if err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return errors.New("操作失败")
	}

	if exists {
		return errors.New("已经点赞过该视频")
	}

	// 获取视频信息，用于获取视频作者ID
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return errors.New("视频不存在")
		}
		logger.Error("获取视频信息失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("操作失败")
	}

	// 创建点赞记录
	like := &model.Like{
		UserID:  userID,
		VideoID: videoID,
	}

	if err := s.likeRepo.Create(ctx, like); err != nil {
		logger.Error("创建点赞记录失败", zap.Error(err))
		return errors.New("点赞失败")
	}

	// 更新视频点赞数
	if err := s.videoRepo.IncrementLikeCount(ctx, videoID, 1); err != nil {
		logger.Error("更新视频点赞数失败", zap.Error(err))
	}

	// 更新统计数据（异步）
	if s.statsService != nil {
		go func() { _ = s.statsService.RecordLike(context.Background(), videoID, 1) }()
	}

	// 更新用户兴趣画像（异步）
	if s.recommendEngine != nil {
		go func() {
			_ = s.recommendEngine.UpdateUserProfile(context.Background(), userID, &model.UserBehavior{
				UserID:  userID,
				VideoID: videoID,
				Action:  2, // 2-点赞
			})
		}()
	}

	// 发送通知（异步）
	if s.messageService != nil {
		go func() {
			s.messageService.CreateNotification(context.Background(), &CreateNotificationRequest{
				UserID:        video.UserID,
				Type:          NotifyTypeLike,
				SenderID:      &userID,
				RelatedID:     &videoID,
				Title:         "新的点赞",
				Content:       "有人点赞了你的视频",
				VideoID:       &videoID,
				VideoCoverURL: video.CoverURL,
				VideoTitle:    video.Title,
			})
		}()
	}

	logger.Info("点赞视频成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// UnlikeVideo 取消点赞
func (s *videoServiceImpl) UnlikeVideo(ctx context.Context, userID, videoID uint) error {
	logger.Info("取消点赞", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	if err := s.likeRepo.Delete(ctx, userID, videoID); err != nil {
		if pkgerrors.IsNotFound(err) {
			return errors.New("未点赞该视频")
		}
		logger.Error("取消点赞失败", zap.Error(err))
		return errors.New("取消点赞失败")
	}

	// 更新视频点赞数
	if err := s.videoRepo.IncrementLikeCount(ctx, videoID, -1); err != nil {
		logger.Error("更新视频点赞数失败", zap.Error(err))
	}

	// 更新统计数据（异步）
	if s.statsService != nil {
		go func() { _ = s.statsService.RecordLike(context.Background(), videoID, -1) }()
	}

	logger.Info("取消点赞成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// FavoriteVideo 收藏视频
func (s *videoServiceImpl) FavoriteVideo(ctx context.Context, userID, videoID uint) error {
	logger.Info("收藏视频", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	// 检查是否已收藏
	exists, err := s.favoriteRepo.Exists(ctx, userID, videoID)
	if err != nil {
		logger.Error("检查收藏状态失败", zap.Error(err))
		return errors.New("操作失败")
	}

	if exists {
		return errors.New("已经收藏过该视频")
	}

	// 获取视频信息，用于获取视频作者ID
	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return errors.New("视频不存在")
		}
		logger.Error("获取视频信息失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("操作失败")
	}

	// 创建收藏记录
	favorite := &model.Favorite{
		UserID:  userID,
		VideoID: videoID,
	}

	if err := s.favoriteRepo.Create(ctx, favorite); err != nil {
		logger.Error("创建收藏记录失败", zap.Error(err))
		return errors.New("收藏失败")
	}

	// 更新视频收藏数
	if err := s.videoRepo.IncrementFavoriteCount(ctx, videoID, 1); err != nil {
		logger.Error("更新视频收藏数失败", zap.Error(err))
	}

	// 更新统计数据（异步）
	if s.statsService != nil {
		go func() { _ = s.statsService.RecordFavorite(context.Background(), videoID, 1) }()
	}

	// 更新用户兴趣画像（异步）
	if s.recommendEngine != nil {
		go func() {
			_ = s.recommendEngine.UpdateUserProfile(context.Background(), userID, &model.UserBehavior{
				UserID:  userID,
				VideoID: videoID,
				Action:  5, // 5-收藏
			})
		}()
	}

	// 发送通知（异步）
	if s.messageService != nil {
		go func() {
			s.messageService.CreateNotification(context.Background(), &CreateNotificationRequest{
				UserID:        video.UserID,
				Type:          NotifyTypeLike,
				SenderID:      &userID,
				RelatedID:     &videoID,
				Title:         "新的收藏",
				Content:       "有人收藏了你的视频",
				VideoID:       &videoID,
				VideoCoverURL: video.CoverURL,
				VideoTitle:    video.Title,
			})
		}()
	}

	logger.Info("收藏视频成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// UnfavoriteVideo 取消收藏
func (s *videoServiceImpl) UnfavoriteVideo(ctx context.Context, userID, videoID uint) error {
	logger.Info("取消收藏", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))

	if err := s.favoriteRepo.Delete(ctx, userID, videoID); err != nil {
		if pkgerrors.IsNotFound(err) {
			return errors.New("未收藏该视频")
		}
		logger.Error("取消收藏失败", zap.Error(err))
		return errors.New("取消收藏失败")
	}

	// 更新视频收藏数
	if err := s.videoRepo.IncrementFavoriteCount(ctx, videoID, -1); err != nil {
		logger.Error("更新视频收藏数失败", zap.Error(err))
	}

	// 更新统计数据（异步）
	if s.statsService != nil {
		go func() { _ = s.statsService.RecordFavorite(context.Background(), videoID, -1) }()
	}

	logger.Info("取消收藏成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}

// UpdateVideoStatus 更新视频状态
func (s *videoServiceImpl) UpdateVideoStatus(ctx context.Context, videoID uint, status int8) error {
	logger.Info("更新视频状态", zap.Uint("video_id", videoID), zap.Int8("status", status))

	video, err := s.videoRepo.FindByID(ctx, videoID)
	if err != nil {
		return errors.New("视频不存在")
	}

	video.Status = status
	if err := s.videoRepo.Update(ctx, video); err != nil {
		logger.Error("更新视频状态失败", zap.Error(err), zap.Uint("video_id", videoID))
		return errors.New("更新状态失败")
	}

	logger.Info("视频状态更新成功", zap.Uint("video_id", videoID), zap.Int8("status", status))
	return nil
}

// EnrichVideoList 批量丰富视频信息
func (s *videoServiceImpl) EnrichVideoList(ctx context.Context, userID uint, videos []*model.Video) ([]*model.VideoVO, error) {
	result := make([]*model.VideoVO, len(videos))
	for i, v := range videos {
		vo, err := s.EnrichVideo(ctx, userID, v)
		if err != nil {
			return nil, err
		}
		result[i] = vo
	}
	return result, nil
}

// EnrichVideo 丰富单个视频信息
func (s *videoServiceImpl) EnrichVideo(ctx context.Context, userID uint, video *model.Video) (*model.VideoVO, error) {
	// 深度复制视频对象以避免修改原始 model 中的数据
	vo := &model.VideoVO{
		Video: video,
	}

	// 1. 处理视频和封面完整 URL
	vo.VideoURL = s.fullURL(video.VideoURL)
	vo.CoverURL = s.fullURL(video.CoverURL)

	// 2. 处理作者信息 (精简字段)
	if video.User != nil {
		isFollowed := false
		if userID > 0 {
			isFollowed, _ = s.followRepo.Exists(ctx, userID, video.UserID)
		}
		vo.User = &model.AuthorVO{
			ID:              video.User.ID,
			Username:        video.User.Username,
			Nickname:        video.User.Nickname,
			Avatar:          s.fullURL(video.User.Avatar),
			BackgroundImage: s.fullURL(video.User.BackgroundImage),
			IsFollowed:      isFollowed,
		}
	} else {
		// 如果没有 Preload User，则只填充 ID
		vo.User = &model.AuthorVO{
			ID: video.UserID,
		}
	}

	// 3. 检查互动状态
	if userID > 0 {
		vo.IsLiked, _ = s.likeRepo.Exists(ctx, userID, video.ID)
		vo.IsFavorited, _ = s.favoriteRepo.Exists(ctx, userID, video.ID)
	}

	return vo, nil
}

// EnrichMyVideoList 批量丰富个人视频信息
func (s *videoServiceImpl) EnrichMyVideoList(ctx context.Context, userID uint, videos []*model.Video) ([]*model.MyVideoVO, error) {
	result := make([]*model.MyVideoVO, len(videos))
	for i, v := range videos {
		vo, err := s.EnrichMyVideo(ctx, userID, v)
		if err != nil {
			return nil, err
		}
		result[i] = vo
	}
	return result, nil
}

// EnrichMyVideo 丰富单个个人视频信息
func (s *videoServiceImpl) EnrichMyVideo(ctx context.Context, userID uint, video *model.Video) (*model.MyVideoVO, error) {
	vo := &model.MyVideoVO{
		Video: video,
	}

	vo.VideoURL = s.fullURL(video.VideoURL)
	vo.CoverURL = s.fullURL(video.CoverURL)

	// 1. 检查自己的互动状态
	if userID > 0 {
		vo.IsLiked, _ = s.likeRepo.Exists(ctx, userID, video.ID)
		vo.IsFavorited, _ = s.favoriteRepo.Exists(ctx, userID, video.ID)
	}

	return vo, nil
}

// GetVideoLikers 获取点赞视频的用户列表
func (s *videoServiceImpl) GetVideoLikers(ctx context.Context, videoID uint, page, pageSize int) ([]*model.AuthorVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	likes, err := s.likeRepo.FindByVideoID(ctx, videoID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	authors := make([]*model.AuthorVO, 0)
	for _, l := range likes {
		if l.User != nil {
			authors = append(authors, &model.AuthorVO{
				ID:       l.User.ID,
				Nickname: l.User.Nickname,
				Avatar:   s.fullURL(l.User.Avatar),
			})
		}
	}

	return authors, int64(len(authors)), nil
}

// GetVideoFavoriters 获取收藏视频的用户列表
func (s *videoServiceImpl) GetVideoFavoriters(ctx context.Context, videoID uint, page, pageSize int) ([]*model.AuthorVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	favorites, err := s.favoriteRepo.FindByVideoID(ctx, videoID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	authors := make([]*model.AuthorVO, 0)
	for _, f := range favorites {
		if f.User != nil {
			authors = append(authors, &model.AuthorVO{
				ID:       f.User.ID,
				Nickname: f.User.Nickname,
				Avatar:   s.fullURL(f.User.Avatar),
			})
		}
	}

	return authors, int64(len(authors)), nil
}

// fullURL 将相对路径转换为完整 URL
func (s *videoServiceImpl) fullURL(path string) string {
	if path == "" || strings.HasPrefix(path, "http") {
		return path
	}
	// 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	baseURL := s.cfg.Upload.BaseURL
	// 去掉 baseURL 末尾的 /
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s%s", baseURL, path)
}
