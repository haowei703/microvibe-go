package repository

import (
	"context"
	"microvibe-go/internal/model"
	pkgerrors "microvibe-go/pkg/errors"
	"time"

	"gorm.io/gorm"
)

// MessageRepository 消息仓储层接口
type MessageRepository interface {
	// CreateMessage 创建消息
	CreateMessage(ctx context.Context, message *model.Message) error
	// GetMessageByID 根据ID获取消息
	GetMessageByID(ctx context.Context, id uint) (*model.Message, error)
	// GetConversationMessages 获取会话消息列表
	GetConversationMessages(ctx context.Context, user1ID, user2ID uint, page, pageSize int) ([]*model.Message, int64, error)
	// MarkAsRead 标记消息为已读
	MarkAsRead(ctx context.Context, messageID, userID uint) error
	// MarkConversationAsRead 标记会话所有消息为已读
	MarkConversationAsRead(ctx context.Context, user1ID, user2ID uint) error
	// DeleteMessage 删除消息
	DeleteMessage(ctx context.Context, messageID, userID uint) error

	// GetConversationList 获取会话列表
	GetConversationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Conversation, int64, error)
	// GetOrCreateConversation 获取或创建会话
	GetOrCreateConversation(ctx context.Context, user1ID, user2ID uint) (*model.Conversation, error)
	// UpdateConversation 更新会话
	UpdateConversation(ctx context.Context, conversation *model.Conversation) error
	// GetUnreadMessageCount 获取未读消息总数
	GetUnreadMessageCount(ctx context.Context, userID uint) (int64, error)
}

// messageRepositoryImpl 消息仓储层实现
type messageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepository 创建消息仓储实例
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepositoryImpl{db: db}
}

// CreateMessage 创建消息
func (r *messageRepositoryImpl) CreateMessage(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建消息
		if err := tx.Create(message).Error; err != nil {
			return err
		}

		// 更新会话
		var conversation model.Conversation
		user1ID, user2ID := message.SenderID, message.ReceiverID
		if user1ID > user2ID {
			user1ID, user2ID = user2ID, user1ID
		}

		err := tx.Where("user1_id = ? AND user2_id = ?", user1ID, user2ID).
			First(&conversation).Error

		if pkgerrors.IsNotFound(err) {
			// 创建新会话
			conversation = model.Conversation{
				User1ID:       user1ID,
				User2ID:       user2ID,
				LastMessageID: &message.ID,
				LastContent:   message.Content,
			}
			if message.ReceiverID == user1ID {
				conversation.UnreadCount1 = 1
			} else {
				conversation.UnreadCount2 = 1
			}
			return tx.Create(&conversation).Error
		}

		if err != nil {
			return err
		}

		// 更新会话
		updates := map[string]interface{}{
			"last_message_id": message.ID,
			"last_content":    message.Content,
			"updated_at":      time.Now(),
		}

		// 增加接收者未读数
		if message.ReceiverID == user1ID {
			updates["unread_count1"] = gorm.Expr("unread_count1 + 1")
		} else {
			updates["unread_count2"] = gorm.Expr("unread_count2 + 1")
		}

		return tx.Model(&conversation).Updates(updates).Error
	})
}

// GetMessageByID 根据ID获取消息
func (r *messageRepositoryImpl) GetMessageByID(ctx context.Context, id uint) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Receiver").
		First(&message, id).Error
	return &message, err
}

// GetConversationMessages 获取会话消息列表
func (r *messageRepositoryImpl) GetConversationMessages(ctx context.Context, user1ID, user2ID uint, page, pageSize int) ([]*model.Message, int64, error) {
	var messages []*model.Message
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Message{}).
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			user1ID, user2ID, user2ID, user1ID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Sender").
		Preload("Receiver").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error

	return messages, total, err
}

// MarkAsRead 标记消息为已读
func (r *messageRepositoryImpl) MarkAsRead(ctx context.Context, messageID, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Message{}).
		Where("id = ? AND receiver_id = ? AND is_read = ?", messageID, userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// MarkConversationAsRead 标记会话所有消息为已读
func (r *messageRepositoryImpl) MarkConversationAsRead(ctx context.Context, user1ID, user2ID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 标记消息为已读
		now := time.Now()
		if err := tx.Model(&model.Message{}).
			Where("sender_id = ? AND receiver_id = ? AND is_read = ?", user2ID, user1ID, false).
			Updates(map[string]interface{}{
				"is_read": true,
				"read_at": now,
			}).Error; err != nil {
			return err
		}

		// 更新会话未读数
		conversationUser1ID, conversationUser2ID := user1ID, user2ID
		if conversationUser1ID > conversationUser2ID {
			conversationUser1ID, conversationUser2ID = conversationUser2ID, conversationUser1ID
		}

		var conversation model.Conversation
		if err := tx.Where("user1_id = ? AND user2_id = ?", conversationUser1ID, conversationUser2ID).
			First(&conversation).Error; err != nil {
			return err
		}

		// 清零当前用户的未读数
		if user1ID == conversationUser1ID {
			return tx.Model(&conversation).Update("unread_count1", 0).Error
		}
		return tx.Model(&conversation).Update("unread_count2", 0).Error
	})
}

// DeleteMessage 删除消息
func (r *messageRepositoryImpl) DeleteMessage(ctx context.Context, messageID, userID uint) error {
	// 只能删除自己发送的消息
	return r.db.WithContext(ctx).
		Where("id = ? AND sender_id = ?", messageID, userID).
		Delete(&model.Message{}).Error
}

// GetConversationList 获取会话列表
func (r *messageRepositoryImpl) GetConversationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Conversation, int64, error) {
	var conversations []*model.Conversation
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("user1_id = ? OR user2_id = ?", userID, userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User1").
		Preload("User2").
		Preload("LastMessage").
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&conversations).Error

	return conversations, total, err
}

// GetOrCreateConversation 获取或创建会话
func (r *messageRepositoryImpl) GetOrCreateConversation(ctx context.Context, user1ID, user2ID uint) (*model.Conversation, error) {
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var conversation model.Conversation
	err := r.db.WithContext(ctx).
		Where("user1_id = ? AND user2_id = ?", user1ID, user2ID).
		First(&conversation).Error

	if err == gorm.ErrRecordNotFound {
		conversation = model.Conversation{
			User1ID: user1ID,
			User2ID: user2ID,
		}
		if err := r.db.WithContext(ctx).Create(&conversation).Error; err != nil {
			return nil, err
		}
		return &conversation, nil
	}

	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

// UpdateConversation 更新会话
func (r *messageRepositoryImpl) UpdateConversation(ctx context.Context, conversation *model.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

// GetUnreadMessageCount 获取未读消息总数
func (r *messageRepositoryImpl) GetUnreadMessageCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Message{}).
		Where("receiver_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}
