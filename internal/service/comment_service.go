package service

import (
	"context"
	"errors"
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"
	"regexp"
	"strconv"

	"go.uber.org/zap"
)

var mentionRegex = regexp.MustCompile(`@\[(\d+):[^\]]+\]`)

// CommentService 评论服务层接口
type CommentService interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, userID uint, req *CreateCommentRequest) (*model.Comment, error)
	// GetCommentByID 获取评论详情
	GetCommentByID(ctx context.Context, commentID uint) (*model.Comment, error)
	// GetVideoComments 获取视频的评论列表
	GetVideoComments(ctx context.Context, videoID uint, page, pageSize int) ([]*model.Comment, int64, error)
	// GetReplies 获取评论的回复列表
	GetReplies(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error)
	// DeleteComment 删除评论
	DeleteComment(ctx context.Context, userID, commentID uint) error
	// LikeComment 点赞评论
	LikeComment(ctx context.Context, userID, commentID uint) error
	// UnlikeComment 取消点赞评论
	UnlikeComment(ctx context.Context, userID, commentID uint) error
	// TogglePinComment 切换置顶状态
	TogglePinComment(ctx context.Context, userID, commentID uint, isPinned bool) error
	// GetReceivedComments 获取创作者收到的评论
	GetReceivedComments(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error)
	// GetSentComments 获取用户发出的评论
	GetSentComments(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error)
}

// commentServiceImpl 评论服务层实现
type commentServiceImpl struct {
	commentRepo     repository.CommentRepository
	videoRepo       repository.VideoRepository
	messageService  MessageService
	statsService    VideoStatsService
	recommendEngine *recommend.Engine
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

// SetMessageService 设置消息服务
func (s *commentServiceImpl) SetMessageService(messageService MessageService) {
	s.messageService = messageService
}

// SetStatsService 设置统计服务（延迟注入避免循环依赖）
func (s *commentServiceImpl) SetStatsService(statsService VideoStatsService) {
	s.statsService = statsService
}

// SetRecommendEngine 设置推荐引擎
func (s *commentServiceImpl) SetRecommendEngine(engine *recommend.Engine) {
	s.recommendEngine = engine
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	VideoID       uint   `json:"video_id" binding:"required"`
	Content       string `json:"content" binding:"required,min=1,max=1000"`
	ParentID      *uint  `json:"parent_id"`        // 父评论ID（回复评论时使用）
	ReplyToUserID *uint  `json:"reply_to_user_id"` // 回复的用户ID
}

// CreateComment 创建评论
func (s *commentServiceImpl) CreateComment(ctx context.Context, userID uint, req *CreateCommentRequest) (*model.Comment, error) {
	logger.Info("创建评论", zap.Uint("user_id", userID), zap.Uint("video_id", req.VideoID))

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

	// 如果是回复评论，检查父评论是否存在，并确定根评论
	var rootID *uint
	if req.ParentID != nil {
		parent, err := s.commentRepo.FindByID(ctx, *req.ParentID)
		if err != nil {
			return nil, errors.New("父评论不存在")
		}
		if parent.VideoID != req.VideoID {
			return nil, errors.New("父评论不属于该视频")
		}
		// 确定根评论：如果父评论本身就是根（无parent_id），则根=父；否则根=父的根
		if parent.RootID != nil {
			rootID = parent.RootID
		} else {
			rootID = req.ParentID
		}
	}

	// 构建评论对象
	comment := &model.Comment{
		UserID:        userID,
		VideoID:       req.VideoID,
		Content:       req.Content,
		ParentID:      req.ParentID,
		RootID:        rootID,
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

		// 更新每日统计
		if s.statsService != nil {
			_ = s.statsService.RecordComment(ctx, req.VideoID, 1)
		}

		if rootID != nil {
			if err := s.commentRepo.IncrementReplyCount(ctx, *rootID); err != nil {
				logger.Error("更新根评论回复数失败", zap.Error(err))
			}
		}

		// 更新用户兴趣画像
		if s.recommendEngine != nil {
			_ = s.recommendEngine.UpdateUserProfile(ctx, userID, &model.UserBehavior{
				UserID:  userID,
				VideoID: req.VideoID,
				Action:  3, // 3-评论
			})
		}
	}()

	// 发送通知（异步）
	if s.messageService != nil {
		go func() {
			bgCtx := context.Background()

			// 处理 @提及
			matches := mentionRegex.FindAllStringSubmatch(comment.Content, -1)
			mentionedUserIDs := make(map[uint]bool)
			for _, match := range matches {
				if id, err := strconv.ParseUint(match[1], 10, 64); err == nil {
					targetUID := uint(id)
					if targetUID != userID { // 不要通知自己
						mentionedUserIDs[targetUID] = true
					}
				}
			}

			for mUID := range mentionedUserIDs {
				mention := &model.CommentMention{
					CommentID: comment.ID,
					UserID:    mUID,
				}
				_ = s.commentRepo.CreateMention(bgCtx, mention)

				// 发送 @ 提及通知
				s.messageService.CreateNotification(bgCtx, &CreateNotificationRequest{
					UserID:         mUID,
					Type:           NotifyTypeMention,
					SenderID:       &comment.UserID,
					RelatedID:      &comment.ID,
					Title:          "有人在评论中提到了你",
					Content:        comment.Content,
					VideoID:        &comment.VideoID,
					CommentID:      &comment.ID,
					CommentContent: comment.Content,
				})
			}

			// 原有的回复/评论通知逻辑
			if req.ParentID != nil {
				parentComment, err := s.commentRepo.FindByID(bgCtx, *req.ParentID)
				if err == nil {
					// 如果被提到的人里已经包含了父评论作者，则不再重复发送回复通知
					if !mentionedUserIDs[parentComment.UserID] {
						s.messageService.CreateNotification(bgCtx, &CreateNotificationRequest{
							UserID:         parentComment.UserID,
							Type:           NotifyTypeComment,
							SenderID:       &comment.UserID,
							RelatedID:      &comment.ID,
							Title:          "新的回复",
							Content:        "有人回复了你的评论",
							VideoID:        &comment.VideoID,
							CommentID:      &comment.ID,
							CommentContent: comment.Content,
						})
					}
				}
			} else {
				// 如果被提到的人里已经包含了视频作者，则不再重复发送评论通知
				if !mentionedUserIDs[video.UserID] {
					s.messageService.CreateNotification(bgCtx, &CreateNotificationRequest{
						UserID:         video.UserID,
						Type:           NotifyTypeComment,
						SenderID:       &comment.UserID,
						RelatedID:      &comment.ID,
						Title:          "新的评论",
						Content:        "有人评论了你的视频",
						VideoID:        &comment.VideoID,
						CommentID:      &comment.ID,
						CommentContent: comment.Content,
					})
				}
			}
		}()
	}

	logger.Info("评论创建成功", zap.Uint("comment_id", comment.ID))
	// 填充提及信息
	s.enrichComments(ctx, []*model.Comment{comment})

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

// GetVideoComments 获取视频的评论列表（顶级评论分页，每条顶级评论带3条子评论）
func (s *commentServiceImpl) GetVideoComments(ctx context.Context, videoID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	logger.Debug("获取视频评论列表", zap.Uint("video_id", videoID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 只获取顶级评论（ParentID 为 NULL）
	comments, total, err := s.commentRepo.FindTopLevelByVideoID(ctx, videoID, pageSize, offset)
	if err != nil {
		logger.Error("获取视频评论列表失败", zap.Error(err), zap.Uint("video_id", videoID))
		return nil, 0, err
	}

	// 填充提及信息
	s.enrichComments(ctx, comments)

	return comments, total, nil
}

// GetReplies 获取评论的回复列表
func (s *commentServiceImpl) GetReplies(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	logger.Debug("获取评论回复列表", zap.Uint("parent_id", parentID), zap.Int("page", page))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	replies, err := s.commentRepo.FindByRootID(ctx, parentID, pageSize, offset)
	if err != nil {
		logger.Error("获取评论回复列表失败", zap.Error(err), zap.Uint("parent_id", parentID))
		return nil, 0, err
	}

	total, err := s.commentRepo.CountByRootID(ctx, parentID)
	if err != nil {
		logger.Error("统计回复总数失败", zap.Error(err))
		total = int64(len(replies))
	}

	// 填充提及信息
	s.enrichComments(ctx, replies)

	return replies, total, nil
}

// DeleteComment 删除评论
func (s *commentServiceImpl) DeleteComment(ctx context.Context, userID, commentID uint) error {
	logger.Info("删除评论", zap.Uint("comment_id", commentID))

	// 先查询评论
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 检查是否是评论所有者或视频所有者 (创作者管理权限)
	if comment.UserID != userID {
		// 检查是否是视频作者
		video, err := s.videoRepo.FindByID(ctx, comment.VideoID)
		if err != nil || video.UserID != userID {
			logger.Warn("无权限删除评论", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
			return errors.New("无权限删除该评论")
		}
		logger.Info("视频作者删除评论", zap.Uint("creator_id", userID), zap.Uint("comment_id", commentID))
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

		// 更新每日统计
		if s.statsService != nil {
			_ = s.statsService.RecordComment(ctx, comment.VideoID, -1)
		}

		if comment.ParentID != nil {
			if err := s.commentRepo.DecrementReplyCount(ctx, *comment.ParentID); err != nil {
				logger.Error("更新父评论回复数失败", zap.Error(err))
			}
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

	// 获取评论信息，用于获取评论作者ID
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 创建点赞记录
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

	// 发送通知（异步）
	if s.messageService != nil {
		go func() {
			var coverURL, title string
			if video, err := s.videoRepo.FindByID(context.Background(), comment.VideoID); err == nil {
				coverURL = video.CoverURL
				title = video.Title
			}
			s.messageService.CreateNotification(context.Background(), &CreateNotificationRequest{
				UserID:         comment.UserID,
				Type:           NotifyTypeLike,
				SenderID:       &userID,
				RelatedID:      &commentID,
				Title:          "评论被点赞",
				Content:        "有人点赞了你的评论",
				VideoID:        &comment.VideoID,
				VideoCoverURL:  coverURL,
				VideoTitle:     title,
				CommentID:      &comment.ID,
				CommentContent: comment.Content,
			})
		}()
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

// enrichComments 填充评论的提及信息 (批量处理)
func (s *commentServiceImpl) enrichComments(ctx context.Context, comments []*model.Comment) {
	if len(comments) == 0 {
		return
	}

	commentIDs := make([]uint, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}

	mentions, err := s.commentRepo.FindMentionsByCommentIDs(ctx, commentIDs)
	if err != nil {
		logger.Error("批量获取提及信息失败", zap.Error(err))
		return
	}

	// 将提及信息按评论ID分组
	mentionsMap := make(map[uint][]*model.User)
	for _, m := range mentions {
		if m.User != nil {
			mentionsMap[m.CommentID] = append(mentionsMap[m.CommentID], m.User)
		}
	}

	// 填充到评论对象中
	for _, c := range comments {
		if c.Mentions == nil {
			c.Mentions = make([]*model.User, 0)
		}
		if users, ok := mentionsMap[c.ID]; ok {
			c.Mentions = users
		}
	}
}

// TogglePinComment 切换置顶状态
func (s *commentServiceImpl) TogglePinComment(ctx context.Context, userID, commentID uint, isPinned bool) error {
	logger.Info("切换评论置顶状态", zap.Uint("comment_id", commentID), zap.Bool("is_pinned", isPinned))

	// 1. 获取评论信息
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("评论不存在")
	}

	// 2. 只有视频作者可以置顶评论
	video, err := s.videoRepo.FindByID(ctx, comment.VideoID)
	if err != nil {
		return errors.New("视频不存在")
	}

	if video.UserID != userID {
		return errors.New("只有视频作者可以置顶评论")
	}

	// 3. 更新置顶状态 (ResetAndPinComment 会处理同一视频下的排他性置顶)
	if err := s.commentRepo.ResetAndPinComment(ctx, comment.VideoID, commentID, isPinned); err != nil {
		logger.Error("更新置顶状态失败", zap.Error(err))
		return errors.New("操作失败")
	}

	return nil
}

// GetReceivedComments 获取创作者收到的评论
func (s *commentServiceImpl) GetReceivedComments(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	comments, total, err := s.commentRepo.FindReceivedByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		logger.Error("获取创作者收到评论失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	// 填充提及信息
	s.enrichComments(ctx, comments)

	return comments, total, nil
}

// GetSentComments 获取用户发出的评论
func (s *commentServiceImpl) GetSentComments(ctx context.Context, userID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	comments, total, err := s.commentRepo.FindSentByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		logger.Error("获取用户发出评论失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	s.enrichComments(ctx, comments)

	return comments, total, nil
}
