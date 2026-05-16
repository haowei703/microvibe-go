package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// MessageService 消息服务层接口
type MessageService interface {
	// SendMessage 发送消息
	SendMessage(ctx context.Context, req *SendMessageRequest) (*model.Message, error)
	// GetConversationMessages 获取会话消息
	GetConversationMessages(ctx context.Context, userID, targetUserID uint, page, pageSize int) ([]*model.Message, int64, error)
	// MarkAsRead 标记消息为已读
	MarkAsRead(ctx context.Context, messageID, userID uint) error
	// MarkConversationAsRead 标记会话所有消息为已读
	MarkConversationAsRead(ctx context.Context, userID, targetUserID uint) error
	// GetConversationMessagesByID 根据会话ID获取消息
	GetConversationMessagesByID(ctx context.Context, userID, conversationID uint, page, pageSize int) ([]*model.Message, int64, error)
	// MarkConversationAsReadByID 根据会话ID标记已读
	MarkConversationAsReadByID(ctx context.Context, userID, conversationID uint) error
	// GetOrCreateConversation 获取或创建会话
	GetOrCreateConversation(ctx context.Context, userID, targetUserID uint) (*model.Conversation, error)
	// GetConversationList 获取会话列表
	GetConversationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Conversation, int64, error)
	// GetUnreadMessageCount 获取未读消息数
	GetUnreadMessageCount(ctx context.Context, userID uint) (int64, error)
	// DeleteMessage 删除消息
	DeleteMessage(ctx context.Context, messageID, userID uint) error

	// CreateNotification 创建通知
	CreateNotification(ctx context.Context, req *CreateNotificationRequest) error
	// GetNotificationList 获取通知列表
	GetNotificationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Notification, int64, error)
	// MarkNotificationAsRead 标记通知为已读
	MarkNotificationAsRead(ctx context.Context, id, userID uint) error
	// MarkAllNotificationsAsRead 标记所有通知为已读
	MarkAllNotificationsAsRead(ctx context.Context, userID uint) error
	// GetUnreadNotificationCount 获取未读通知数
	GetUnreadNotificationCount(ctx context.Context, userID uint) (int64, error)
}

// messageServiceImpl 消息服务层实现
type messageServiceImpl struct {
	messageRepo      repository.MessageRepository
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	videoRepo        repository.VideoRepository
	signalingService MessageSignalingService
}

// NewMessageService 创建消息服务实例
func NewMessageService(
	messageRepo repository.MessageRepository,
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	videoRepo repository.VideoRepository,
) MessageService {
	return &messageServiceImpl{
		messageRepo:      messageRepo,
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		videoRepo:        videoRepo,
	}
}

// SetSignalingService 设置信令服务
func (s *messageServiceImpl) SetSignalingService(signalingService MessageSignalingService) {
	s.signalingService = signalingService
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	SenderID       uint   `json:"sender_id"`
	ConversationID uint   `json:"conversation_id" binding:"required"`
	Type           int8   `json:"type" binding:"required,oneof=1 2 3 4 5"`
	Content        string `json:"content" binding:"max=5000"`
	MediaURL       string `json:"media_url"`
	VideoID        *uint  `json:"video_id"`
}

// NotifyType 通知类型枚举
type NotifyType int8

const (
	NotifyTypeLike    NotifyType = iota + 1 // 点赞
	NotifyTypeComment                       // 评论/回复
	NotifyTypeFollow                        // 关注
	NotifyTypeMention                       // @提及（评论提及和简介提及）
	NotifyTypeSystem                        // 系统通知
)

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	UserID         uint       `json:"user_id" binding:"required"`
	Type           NotifyType `json:"type" binding:"required"`
	SenderID       *uint      `json:"sender_id"`
	RelatedID      *uint      `json:"related_id"`
	Title          string     `json:"title" binding:"required,max=200"`
	Content        string     `json:"content" binding:"max=1000"`
	Link           string     `json:"link"`
	VideoID        *uint      `json:"video_id"`
	VideoCoverURL  string     `json:"video_cover_url"`
	VideoTitle     string     `json:"video_title"`
	CommentID      *uint      `json:"comment_id"`
	CommentContent string     `json:"comment_content"`
}

// SendMessage 发送消息
func (s *messageServiceImpl) SendMessage(ctx context.Context, req *SendMessageRequest) (*model.Message, error) {
	logger.Info("发送消息", zap.Uint("sender_id", req.SenderID), zap.Uint("conversation_id", req.ConversationID))

	// 获取会话并验证权限
	conversation, err := s.messageRepo.GetConversationByID(ctx, req.ConversationID)
	if err != nil {
		logger.Error("会话不存在", zap.Error(err), zap.Uint("conversation_id", req.ConversationID))
		return nil, errors.New("会话不存在")
	}

	// 确定接收者 (目前为私聊逻辑)
	var receiverID uint
	if conversation.User1ID == req.SenderID {
		receiverID = conversation.User2ID
	} else if conversation.User2ID == req.SenderID {
		receiverID = conversation.User1ID
	} else {
		return nil, errors.New("无权在该会话发送消息")
	}

	// 检查接收者状态
	receiver, err := s.userRepo.FindByID(ctx, receiverID)
	if err != nil {
		return nil, errors.New("接收者不存在")
	}
	if receiver.Status != 1 {
		return nil, errors.New("接收者账号异常，无法发送消息")
	}

	// 处理媒体消息的内容占位符
	content := req.Content
	if content == "" {
		switch req.Type {
		case 2:
			content = "[图片]"
		case 3:
			content = "[视频]"
		case 4:
			content = "[语音]"
		case 5:
			content = "[分享视频]"
		}
	}

	// 视频分享必须带 video_id
	if req.Type == 5 && req.VideoID == nil {
		return nil, errors.New("分享视频缺少 video_id")
	}

	// 如果是文本消息且内容为空，则报错
	if req.Type == 1 && content == "" {
		return nil, errors.New("消息内容不能为空")
	}

	// 创建消息
	message := &model.Message{
		SenderID:       req.SenderID,
		ReceiverID:     receiverID,
		ConversationID: &req.ConversationID, // 确保 Message 模型也支持 ConversationID
		Type:           req.Type,
		Content:        content,
		MediaURL:       req.MediaURL,
		VideoID:        req.VideoID,
	}

	if err := s.messageRepo.CreateMessage(ctx, message); err != nil {
		logger.Error("创建消息失败", zap.Error(err))
		return nil, errors.New("发送消息失败")
	}

	// 视频分享：增加视频分享数
	if req.Type == 5 && req.VideoID != nil {
		if err := s.videoRepo.IncrementShareCount(ctx, *req.VideoID, 1); err != nil {
			logger.Error("增加分享数失败", zap.Error(err), zap.Uint("video_id", *req.VideoID))
		}
	}

	// 加载关联的发送者和接收者信息用于 VO 转换
	sender, _ := s.userRepo.FindByID(ctx, message.SenderID)
	message.Sender = sender
	message.Receiver = receiver

	// 视频分享：拉取关联视频用于推送展示
	if req.Type == 5 && req.VideoID != nil {
		if v, err := s.videoRepo.FindByID(ctx, *req.VideoID); err == nil {
			message.Video = v
		}
	}

	// 实时推送（通过 WebSocket）- 推送给接收者时，isMine 应该为 false
	if s.signalingService != nil {
		go func() {
			// 构造 MessageVO，从接收者视角看 isMine = false
			messageVO := &model.MessageVO{
				ID:             message.ID,
				SenderID:       message.SenderID,
				ReceiverID:     message.ReceiverID,
				ConversationID: message.ConversationID,
				Type:           message.Type,
				Content:        message.Content,
				MediaURL:       message.MediaURL,
				VideoID:        message.VideoID,
				IsRead:         message.IsRead,
				ReadAt:         message.ReadAt,
				CreatedAt:      message.CreatedAt,
				IsMine:         false, // 接收者视角
				Sender:         sender.ToAuthorVO(),
				Receiver:       receiver.ToAuthorVO(),
			}
			if err := s.signalingService.PushToUser(message.ReceiverID, "message", messageVO); err != nil {
				logger.Error("实时推送消息失败", zap.Error(err), zap.Uint("receiver_id", message.ReceiverID))
			}
		}()
	}

	logger.Info("消息发送成功", zap.Uint("message_id", message.ID))
	return message, nil
}

// GetConversationMessages 获取会话消息
func (s *messageServiceImpl) GetConversationMessages(ctx context.Context, userID, targetUserID uint, page, pageSize int) ([]*model.Message, int64, error) {
	logger.Info("获取会话消息", zap.Uint("user_id", userID), zap.Uint("target_user_id", targetUserID))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 先获取会话ID
	conversation, err := s.messageRepo.GetOrCreateConversation(ctx, userID, targetUserID)
	if err != nil {
		return nil, 0, err
	}

	return s.messageRepo.GetConversationMessagesByID(ctx, conversation.ID, page, pageSize)
}

// MarkAsRead 标记消息为已读
func (s *messageServiceImpl) MarkAsRead(ctx context.Context, messageID, userID uint) error {
	logger.Info("标记消息已读", zap.Uint("message_id", messageID), zap.Uint("user_id", userID))
	return s.messageRepo.MarkAsRead(ctx, messageID, userID)
}

// MarkConversationAsRead 标记会话所有消息为已读
func (s *messageServiceImpl) MarkConversationAsRead(ctx context.Context, userID, targetUserID uint) error {
	logger.Info("标记会话已读", zap.Uint("user_id", userID), zap.Uint("target_user_id", targetUserID))
	return s.messageRepo.MarkConversationAsRead(ctx, userID, targetUserID)
}

// GetConversationMessagesByID 根据会话ID获取消息
func (s *messageServiceImpl) GetConversationMessagesByID(ctx context.Context, userID, conversationID uint, page, pageSize int) ([]*model.Message, int64, error) {
	logger.Info("根据会话ID获取消息", zap.Uint("user_id", userID), zap.Uint("conversation_id", conversationID))

	// 获取会话并验证权限
	conversation, err := s.messageRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, 0, err
	}

	if conversation.User1ID != userID && conversation.User2ID != userID {
		return nil, 0, errors.New("无权访问该会话")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.messageRepo.GetConversationMessagesByID(ctx, conversationID, page, pageSize)
}

// MarkConversationAsReadByID 根据会话ID标记已读
func (s *messageServiceImpl) MarkConversationAsReadByID(ctx context.Context, userID, conversationID uint) error {
	logger.Info("根据会话ID标记已读", zap.Uint("user_id", userID), zap.Uint("conversation_id", conversationID))

	// 获取会话并验证权限
	conversation, err := s.messageRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return err
	}

	if conversation.User1ID != userID && conversation.User2ID != userID {
		return errors.New("无权访问该会话")
	}

	targetUserID := conversation.User1ID
	if targetUserID == userID {
		targetUserID = conversation.User2ID
	}

	return s.messageRepo.MarkConversationAsRead(ctx, userID, targetUserID)
}

// GetOrCreateConversation 获取或创建会话
func (s *messageServiceImpl) GetOrCreateConversation(ctx context.Context, userID, targetUserID uint) (*model.Conversation, error) {
	logger.Info("获取或创建会话", zap.Uint("user_id", userID), zap.Uint("target_user_id", targetUserID))

	if userID == targetUserID {
		return nil, errors.New("不能和自己创建会话")
	}

	// 检查目标用户是否存在
	_, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("目标用户不存在")
	}

	return s.messageRepo.GetOrCreateConversation(ctx, userID, targetUserID)
}

// GetConversationList 获取会话列表
func (s *messageServiceImpl) GetConversationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Conversation, int64, error) {
	logger.Info("获取会话列表", zap.Uint("user_id", userID))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.messageRepo.GetConversationList(ctx, userID, page, pageSize)
}

// GetUnreadMessageCount 获取未读消息数
func (s *messageServiceImpl) GetUnreadMessageCount(ctx context.Context, userID uint) (int64, error) {
	logger.Info("获取未读消息数", zap.Uint("user_id", userID))
	return s.messageRepo.GetUnreadMessageCount(ctx, userID)
}

// DeleteMessage 删除消息
func (s *messageServiceImpl) DeleteMessage(ctx context.Context, messageID, userID uint) error {
	logger.Info("删除消息", zap.Uint("message_id", messageID), zap.Uint("user_id", userID))
	return s.messageRepo.DeleteMessage(ctx, messageID, userID)
}

// CreateNotification 创建通知
func (s *messageServiceImpl) CreateNotification(ctx context.Context, req *CreateNotificationRequest) error {
	logger.Info("创建通知", zap.Uint("user_id", req.UserID), zap.Int8("type", int8(req.Type)))

	notification := &model.Notification{
		UserID:         req.UserID,
		Type:           int8(req.Type),
		SenderID:       req.SenderID,
		RelatedID:      req.RelatedID,
		Title:          req.Title,
		Content:        req.Content,
		Link:           req.Link,
		VideoID:        req.VideoID,
		VideoCoverURL:  req.VideoCoverURL,
		VideoTitle:     req.VideoTitle,
		CommentID:      req.CommentID,
		CommentContent: req.CommentContent,
	}

	if err := s.notificationRepo.CreateNotification(ctx, notification); err != nil {
		logger.Error("创建通知失败", zap.Error(err))
		return errors.New("创建通知失败")
	}

	// 实时推送通知
	if s.signalingService != nil {
		go func() {
			if err := s.signalingService.PushToUser(notification.UserID, "notification", notification); err != nil {
				logger.Error("实时推送通知失败", zap.Error(err), zap.Uint("user_id", notification.UserID))
			}
		}()
	}

	return nil
}

// GetNotificationList 获取通知列表
func (s *messageServiceImpl) GetNotificationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Notification, int64, error) {
	logger.Info("获取通知列表", zap.Uint("user_id", userID))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.notificationRepo.GetNotificationList(ctx, userID, page, pageSize)
}

// MarkNotificationAsRead 标记通知为已读
func (s *messageServiceImpl) MarkNotificationAsRead(ctx context.Context, id, userID uint) error {
	logger.Info("标记通知已读", zap.Uint("id", id), zap.Uint("user_id", userID))
	return s.notificationRepo.MarkAsRead(ctx, id, userID)
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func (s *messageServiceImpl) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	logger.Info("标记所有通知已读", zap.Uint("user_id", userID))
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

// GetUnreadNotificationCount 获取未读通知数
func (s *messageServiceImpl) GetUnreadNotificationCount(ctx context.Context, userID uint) (int64, error) {
	logger.Info("获取未读通知数", zap.Uint("user_id", userID))
	return s.notificationRepo.GetUnreadCount(ctx, userID)
}
