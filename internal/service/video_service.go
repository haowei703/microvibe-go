package service

import (
	"context"
	"errors"
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
	// GetUserVideos 获取用户的视频列表
	GetUserVideos(ctx context.Context, userID uint, page, pageSize int) ([]*model.Video, int64, error)
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
}

// videoServiceImpl 视频服务层实现
type videoServiceImpl struct {
	videoRepo    repository.VideoRepository
	likeRepo     repository.LikeRepository
	favoriteRepo repository.FavoriteRepository
}

// NewVideoService 创建视频服务实例
func NewVideoService(
	videoRepo repository.VideoRepository,
	likeRepo repository.LikeRepository,
	favoriteRepo repository.FavoriteRepository,
) VideoService {
	return &videoServiceImpl{
		videoRepo:    videoRepo,
		likeRepo:     likeRepo,
		favoriteRepo: favoriteRepo,
	}
}

// CreateVideoRequest 创建视频请求
type CreateVideoRequest struct {
	Title       string   `json:"title" binding:"required,max=200"`
	Description string   `json:"description" binding:"max=1000"`
	VideoURL    string   `json:"video_url" binding:"required"`
	CoverURL    string   `json:"cover_url" binding:"required"`
	Duration    int      `json:"duration" binding:"required,min=1"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	FileSize    int64    `json:"file_size"`
	CategoryID  uint     `json:"category_id"`
	Tags        []string `json:"tags"`
}

// CreateVideo 创建视频
func (s *videoServiceImpl) CreateVideo(ctx context.Context, req *CreateVideoRequest) (*model.Video, error) {
	logger.Info("创建视频", zap.String("title", req.Title))

	// 构建视频对象
	now := time.Now()
	video := &model.Video{
		Title:       req.Title,
		Description: req.Description,
		VideoURL:    req.VideoURL,
		CoverURL:    req.CoverURL,
		Duration:    req.Duration,
		Width:       req.Width,
		Height:      req.Height,
		FileSize:    req.FileSize,
		CategoryID:  req.CategoryID,
		Status:      1, // 1-已发布
		IsPublic:    true,
		PublishedAt: &now,
	}

	// 处理标签 - 将标签数组转换为逗号分隔的字符串
	if len(req.Tags) > 0 {
		video.Tags = strings.Join(req.Tags, ",")
	}

	if err := s.videoRepo.Create(ctx, video); err != nil {
		logger.Error("创建视频失败", zap.Error(err))
		return nil, errors.New("创建视频失败")
	}

	logger.Info("视频创建成功", zap.Uint("video_id", video.ID))
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

	// 异步增加播放量
	go func() {
		ctx := context.Background()
		s.videoRepo.IncrementPlayCount(ctx, videoID)
	}()

	return video, nil
}

// GetUserVideos 获取用户的视频列表
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
		logger.Error("查询用户视频总数失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	return videos, total, nil
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

	logger.Info("点赞成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
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

	// 创建收藏记录
	favorite := &model.Favorite{
		UserID:  userID,
		VideoID: videoID,
	}

	if err := s.favoriteRepo.Create(ctx, favorite); err != nil {
		logger.Error("创建收藏记录失败", zap.Error(err))
		return errors.New("收藏失败")
	}

	logger.Info("收藏成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
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

	logger.Info("取消收藏成功", zap.Uint("user_id", userID), zap.Uint("video_id", videoID))
	return nil
}
