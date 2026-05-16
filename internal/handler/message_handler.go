package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/model"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	messageService service.MessageService
}

// NewMessageHandler 创建消息处理器实例
func NewMessageHandler(messageService service.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// SendMessage 发送消息
func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req service.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	req.SenderID = userID

	message, err := h.messageService.SendMessage(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, toMessageVO(message, userID))
}

// GetConversationMessages 获取会话消息
// GetConversationMessages 根据会话ID获取消息
func (h *MessageHandler) GetConversationMessages(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "会话ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	messages, total, err := h.messageService.GetConversationMessagesByID(
		c.Request.Context(), userID, uint(conversationID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	vos := make([]*model.MessageVO, len(messages))
	for i, msg := range messages {
		vos[i] = toMessageVO(msg, userID)
	}

	response.PageSuccess(c, vos, total, page, pageSize)
}

// MarkAsRead 标记消息为已读
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "消息ID格式错误")
		return
	}

	if err := h.messageService.MarkAsRead(c.Request.Context(), uint(messageID), userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "标记成功", nil)
}

// MarkConversationAsRead 标记会话所有消息为已读
func (h *MessageHandler) MarkConversationAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "会话ID格式错误")
		return
	}

	if err := h.messageService.MarkConversationAsReadByID(c.Request.Context(), userID, uint(conversationID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "标记成功", nil)
}

// CreateConversation 创建或获取会话
func (h *MessageHandler) CreateConversation(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req struct {
		TargetUserID uint `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	conversation, err := h.messageService.GetOrCreateConversation(c.Request.Context(), userID, req.TargetUserID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, toConversationVO(conversation, userID))
}

// GetConversationList 获取会话列表
func (h *MessageHandler) GetConversationList(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	conversations, total, err := h.messageService.GetConversationList(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	vos := make([]model.ConversationVO, 0, len(conversations))
	for _, conv := range conversations {
		vos = append(vos, toConversationVO(conv, userID))
	}

	response.PageSuccess(c, vos, total, page, pageSize)
}

// GetUnreadMessageCount 获取未读消息数
func (h *MessageHandler) GetUnreadMessageCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	count, err := h.messageService.GetUnreadMessageCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// DeleteMessage 删除消息
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "消息ID格式错误")
		return
	}

	if err := h.messageService.DeleteMessage(c.Request.Context(), uint(messageID), userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetNotificationList 获取通知列表
func (h *MessageHandler) GetNotificationList(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	notifications, total, err := h.messageService.GetNotificationList(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, notifications, total, page, pageSize)
}

// MarkNotificationAsRead 标记通知为已读
func (h *MessageHandler) MarkNotificationAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := strconv.ParseUint(notificationIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "通知ID格式错误")
		return
	}

	if err := h.messageService.MarkNotificationAsRead(c.Request.Context(), uint(notificationID), userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "标记成功", nil)
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func (h *MessageHandler) MarkAllNotificationsAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.messageService.MarkAllNotificationsAsRead(c.Request.Context(), userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "标记成功", nil)
}

// GetUnreadNotificationCount 获取未读通知数
func (h *MessageHandler) GetUnreadNotificationCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	count, err := h.messageService.GetUnreadNotificationCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// toConversationVO 将模型转换为VO
func toConversationVO(c *model.Conversation, currentUserID uint) model.ConversationVO {
	vo := model.ConversationVO{
		ID:          c.ID,
		LastMessage: toMessageVO(c.LastMessage, currentUserID),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	// 识别对方信息
	if c.User1ID == currentUserID {
		vo.UserID = c.User2ID
		if c.User2 != nil {
			vo.Nickname = c.User2.Nickname
			vo.Avatar = c.User2.Avatar
		}
		vo.UnreadCount = c.UnreadCount1
	} else {
		vo.UserID = c.User1ID
		if c.User1 != nil {
			vo.Nickname = c.User1.Nickname
			vo.Avatar = c.User1.Avatar
		}
		vo.UnreadCount = c.UnreadCount2
	}

	return vo
}

// toMessageVO 将消息模型转换为VO
func toMessageVO(m *model.Message, currentUserID uint) *model.MessageVO {
	if m == nil {
		return nil
	}
	return &model.MessageVO{
		ID:             m.ID,
		SenderID:       m.SenderID,
		ReceiverID:     m.ReceiverID,
		ConversationID: m.ConversationID,
		Type:           m.Type,
		Content:        m.Content,
		MediaURL:       m.MediaURL,
		VideoID:        m.VideoID,
		IsRead:         m.IsRead,
		ReadAt:         m.ReadAt,
		CreatedAt:      m.CreatedAt,
		IsMine:         m.SenderID == currentUserID,
		Sender:         m.Sender.ToAuthorVO(),
		Receiver:       m.Receiver.ToAuthorVO(),
		Video:          m.Video,
	}
}
