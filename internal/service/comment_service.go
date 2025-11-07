package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// CommentService 评论服务层接口
type CommentService interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, req *CreateCommentRequest) (*model.Comment, error)
	// GetCommentByID 获取评论详情
	GetCommentByID(ctx context.Context, commentID uint) (*model.Comment, error)
	// GetVideoComments 获取视频的评论列表
	GetVideoComments(ctx context.Context, videoID uint, page, pageSize int) ([]*model.Comment, int64, error)
	// GetReplies 获取评论的回复列表
	GetReplies(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, error)
	// DeleteComment 删除评论
	DeleteComment(ctx context.Context, userID, commentID uint) error
	// LikeComment 点赞评论
	LikeComment(ctx context.Context, userID, commentID uint) error
	// UnlikeComment 取消点赞评论
	UnlikeComment(ctx context.Context, userID, commentID uint) error
}

// commentServiceImpl 评论服务层实现
type commentServiceImpl struct {
	commentRepo repository.CommentRepository
	videoRepo   repository.VideoRepository
}

// NewCommentService 创建评论服务实例
func NewCommentService(
	commentRepo repository.CommentRepository,
	videoRepo repository.VideoRepository,
) CommentService {
	return &commentServiceImpl{
		commentRepo: commentRepo,
		videoRepo:   videoRepo,
	}
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	VideoID       uint   `json:"video_id" binding:"required"`
	Content       string `json:"content" binding:"required,min=1,max=1000"`
	ParentID      *uint  `json:"parent_id"`        // 父评论ID（回复评论时使用）
	ReplyToUserID *uint  `json:"reply_to_user_id"` // 回复的用户ID
}

// CreateComment 创建评论
func (s *commentServiceImpl) CreateComment(ctx context.Context, req *CreateCommentRequest) (*model.Comment, error) {
	logger.Info("创建评论", zap.Uint("video_id", req.VideoID))

	// 检查视频是否存在
	video, err := s.videoRepo.FindByID(ctx, req.VideoID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, errors.New("视频不存在")
		}
		logger.Error("查询视频失败", zap.Error(err))
		return nil, errors.New("创建评论失败")
	}

	// 检查视频是否允许评论
	if !video.AllowComment {
		return nil, errors.New("该视频不允许评论")
	}

	// 如果是回复评论，检查父评论是否存在
	if req.ParentID != nil {
		parent, err := s.commentRepo.FindByID(ctx, *req.ParentID)
		if err != nil {
			return nil, errors.New("父评论不存在")
		}
		// 确保父评论属于同一视频
		if parent.VideoID != req.VideoID {
			return nil, errors.New("父评论不属于该视频")
		}
	}

	// 构建评论对象
	comment := &model.Comment{
		VideoID:       req.VideoID,
		Content:       req.Content,
		ParentID:      req.ParentID,
		ReplyToUserID: req.ReplyToUserID,
		Status:        1, // 1-正常
	}

	// 创建评论
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		logger.Error("创建评论失败", zap.Error(err))
		return nil, errors.New("创建评论失败")
	}

	// 异步更新统计数据
	go func() {
		ctx := context.Background()

		// 更新视频评论数
		if err := s.videoRepo.IncrementCommentCount(ctx, req.VideoID, 1); err != nil {
			logger.Error("更新视频评论数失败", zap.Error(err))
		}

		// 如果是回复，更新父评论的回复数
		if req.ParentID != nil {
			if err := s.commentRepo.IncrementReplyCount(ctx, *req.ParentID); err != nil {
				logger.Error("更新父评论回复数失败", zap.Error(err))
			}
		}
	}()

	logger.Info("评论创建成功", zap.Uint("comment_id", comment.ID))
	return comment, nil
}

// GetCommentByID 获取评论详情
func (s *commentServiceImpl) GetCommentByID(ctx context.Context, commentID uint) (*model.Comment, error) {
	logger.Debug("获取评论详情", zap.Uint("comment_id", commentID))

	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, errors.New("评论不存在")
		}
		logger.Error("获取评论详情失败", zap.Error(err), zap.Uint("comment_id", commentID))
		return nil, err
	}

	return comment, nil
}

// GetVideoComments 获取视频的评论列表
func (s *commentServiceImpl) GetVideoComments(ctx context.Context, videoID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	logger.Debug("获取视频评论列表", zap.Uint("video_id", videoID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	comments, total, err := s.commentRepo.FindByVideoID(ctx, videoID, pageSize, offset)
	if err != nil {
		logger.Error("获取视频评论列表失败", zap.Error(err), zap.Uint("video_id", videoID))
		return nil, 0, err
	}

	return comments, total, nil
}

// GetReplies 获取评论的回复列表
func (s *commentServiceImpl) GetReplies(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, error) {
	logger.Debug("获取评论回复列表", zap.Uint("parent_id", parentID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	replies, err := s.commentRepo.FindByParentID(ctx, parentID, pageSize, offset)
	if err != nil {
		logger.Error("获取评论回复列表失败", zap.Error(err), zap.Uint("parent_id", parentID))
		return nil, err
	}

	return replies, nil
}

// DeleteComment 删除评论
func (s *commentServiceImpl) DeleteComment(ctx context.Context, userID, commentID uint) error {
	logger.Info("删除评论", zap.Uint("comment_id", commentID))

	// 先查询评论
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 检查是否是评论所有者
	if comment.UserID != userID {
		logger.Warn("非评论所有者尝试删除评论", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
		return errors.New("无权限删除该评论")
	}

	if err := s.commentRepo.Delete(ctx, commentID); err != nil {
		logger.Error("删除评论失败", zap.Error(err), zap.Uint("comment_id", commentID))
		return errors.New("删除评论失败")
	}

	// 异步更新统计数据
	go func() {
		ctx := context.Background()

		// 更新视频评论数
		if err := s.videoRepo.IncrementCommentCount(ctx, comment.VideoID, -1); err != nil {
			logger.Error("更新视频评论数失败", zap.Error(err))
		}

		// 如果是回复，更新父评论的回复数
		if comment.ParentID != nil {
			// 注意：由于删除操作，回复数减少
			// 这里需要实现 DecrementReplyCount 或使用负数
			logger.Debug("删除回复，父评论回复数需要减少", zap.Uint("parent_id", *comment.ParentID))
		}
	}()

	logger.Info("评论删除成功", zap.Uint("comment_id", commentID))
	return nil
}

// LikeComment 点赞评论
func (s *commentServiceImpl) LikeComment(ctx context.Context, userID, commentID uint) error {
	logger.Info("点赞评论", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))

	// 检查评论是否存在
	_, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 检查是否已点赞
	hasLiked, err := s.commentRepo.HasCommentLike(ctx, userID, commentID)
	if err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return errors.New("操作失败")
	}

	if hasLiked {
		return errors.New("已经点赞过该评论")
	}

	// 创建点赞记录（幂等操作）
	if err := s.commentRepo.CreateCommentLike(ctx, userID, commentID); err != nil {
		logger.Error("创建点赞记录失败", zap.Error(err))
		return errors.New("点赞失败")
	}

	// 增加点赞数
	if err := s.commentRepo.IncrementLikeCount(ctx, commentID); err != nil {
		logger.Error("更新点赞数失败", zap.Error(err))
		// 尝试回滚点赞记录
		_ = s.commentRepo.DeleteCommentLike(ctx, userID, commentID)
		return errors.New("点赞失败")
	}

	logger.Info("点赞评论成功", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
	return nil
}

// UnlikeComment 取消点赞评论
func (s *commentServiceImpl) UnlikeComment(ctx context.Context, userID, commentID uint) error {
	logger.Info("取消点赞评论", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))

	// 检查评论是否存在
	_, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 检查是否已点赞
	hasLiked, err := s.commentRepo.HasCommentLike(ctx, userID, commentID)
	if err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return errors.New("操作失败")
	}

	if !hasLiked {
		return errors.New("未点赞该评论")
	}

	// 删除点赞记录
	if err := s.commentRepo.DeleteCommentLike(ctx, userID, commentID); err != nil {
		logger.Error("删除点赞记录失败", zap.Error(err))
		return errors.New("取消点赞失败")
	}

	// 减少点赞数
	if err := s.commentRepo.DecrementLikeCount(ctx, commentID); err != nil {
		logger.Error("更新点赞数失败", zap.Error(err))
		// 尝试回滚点赞记录
		_ = s.commentRepo.CreateCommentLike(ctx, userID, commentID)
		return errors.New("取消点赞失败")
	}

	logger.Info("取消点赞评论成功", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
	return nil
}
