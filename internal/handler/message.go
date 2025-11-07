package handler

import (
	"microvibe-go/internal/middleware"
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

	response.Success(c, message)
}

// GetConversationMessages 获取会话消息
func (h *MessageHandler) GetConversationMessages(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	messages, total, err := h.messageService.GetConversationMessages(
		c.Request.Context(), userID, uint(targetUserID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, messages, total, page, pageSize)
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

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	if err := h.messageService.MarkConversationAsRead(c.Request.Context(), userID, uint(targetUserID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "标记成功", nil)
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

	response.PageSuccess(c, conversations, total, page, pageSize)
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
