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
}

// NewMessageService 创建消息服务实例
func NewMessageService(
	messageRepo repository.MessageRepository,
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
) MessageService {
	return &messageServiceImpl{
		messageRepo:      messageRepo,
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
	}
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	SenderID   uint   `json:"sender_id"`
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Type       int8   `json:"type" binding:"required,oneof=1 2 3 4"`
	Content    string `json:"content" binding:"required,max=5000"`
	MediaURL   string `json:"media_url"`
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	Type      int8   `json:"type" binding:"required,oneof=1 2 3 4"`
	SenderID  *uint  `json:"sender_id"`
	RelatedID *uint  `json:"related_id"`
	Title     string `json:"title" binding:"required,max=200"`
	Content   string `json:"content" binding:"max=1000"`
	Link      string `json:"link"`
}

// SendMessage 发送消息
func (s *messageServiceImpl) SendMessage(ctx context.Context, req *SendMessageRequest) (*model.Message, error) {
	logger.Info("发送消息", zap.Uint("sender_id", req.SenderID), zap.Uint("receiver_id", req.ReceiverID))

	// 检查接收者是否存在
	receiver, err := s.userRepo.FindByID(ctx, req.ReceiverID)
	if err != nil {
		logger.Error("接收者不存在", zap.Error(err))
		return nil, errors.New("接收者不存在")
	}

	// 不能给自己发消息
	if req.SenderID == req.ReceiverID {
		return nil, errors.New("不能给自己发送消息")
	}

	// 检查用户状态
	if receiver.Status != 1 {
		return nil, errors.New("接收者账号异常，无法发送消息")
	}

	// 创建消息
	message := &model.Message{
		SenderID:   req.SenderID,
		ReceiverID: req.ReceiverID,
		Type:       req.Type,
		Content:    req.Content,
		MediaURL:   req.MediaURL,
	}

	if err := s.messageRepo.CreateMessage(ctx, message); err != nil {
		logger.Error("创建消息失败", zap.Error(err))
		return nil, errors.New("发送消息失败")
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

	return s.messageRepo.GetConversationMessages(ctx, userID, targetUserID, page, pageSize)
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
	logger.Info("创建通知", zap.Uint("user_id", req.UserID), zap.Int8("type", req.Type))

	notification := &model.Notification{
		UserID:    req.UserID,
		Type:      req.Type,
		SenderID:  req.SenderID,
		RelatedID: req.RelatedID,
		Title:     req.Title,
		Content:   req.Content,
		Link:      req.Link,
	}

	if err := s.notificationRepo.CreateNotification(ctx, notification); err != nil {
		logger.Error("创建通知失败", zap.Error(err))
		return errors.New("创建通知失败")
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
