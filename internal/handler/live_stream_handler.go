package handler

import (
	"strconv"

	"microvibe-go/internal/config"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// LiveStreamHandler 直播处理器
type LiveStreamHandler struct {
	liveService service.LiveStreamService
	cfg         *config.Config
}

// NewLiveStreamHandler 创建直播处理器
func NewLiveStreamHandler(liveService service.LiveStreamService, cfg *config.Config) *LiveStreamHandler {
	return &LiveStreamHandler{
		liveService: liveService,
		cfg:         cfg,
	}
}

// CreateLiveStream 创建直播间
// @Summary 创建直播间
// @Tags 直播
// @Accept json
// @Produce json
// @Param request body service.CreateLiveStreamRequest true "创建直播请求"
// @Success 200 {object} response.Response{data=model.LiveStream}
// @Router /api/v1/live/create [post]
func (h *LiveStreamHandler) CreateLiveStream(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 绑定请求参数
	var req service.CreateLiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	// 调用服务创建直播间
	liveStream, err := h.liveService.CreateLiveStream(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, liveStream)
}

// StartLiveStream 开始直播
// @Summary 开始直播
// @Tags 直播
// @Accept json
// @Produce json
// @Param request body service.StartLiveStreamRequest true "开始直播请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/start [post]
func (h *LiveStreamHandler) StartLiveStream(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 绑定请求参数
	var req service.StartLiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	// 调用服务开始直播
	if err := h.liveService.StartLiveStream(c.Request.Context(), userID.(uint), req.StreamKey); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "直播已开始", nil)
}

// EndLiveStream 结束直播
// @Summary 结束直播
// @Tags 直播
// @Accept json
// @Produce json
// @Param request body service.EndLiveStreamRequest true "结束直播请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/end [post]
func (h *LiveStreamHandler) EndLiveStream(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 绑定请求参数
	var req service.EndLiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	// 调用服务结束直播
	if err := h.liveService.EndLiveStream(c.Request.Context(), userID.(uint), req.StreamKey); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "直播已结束", nil)
}

// GetLiveStream 获取直播间信息
// @Summary 获取直播间信息
// @Tags 直播
// @Produce json
// @Param id path int true "直播间ID"
// @Success 200 {object} response.Response{data=model.LiveStream}
// @Router /api/v1/live/{id} [get]
func (h *LiveStreamHandler) GetLiveStream(c *gin.Context) {
	// 获取直播间ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	// 调用服务查询直播间
	liveStream, err := h.liveService.GetLiveStreamByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, liveStream)
}

// GetLiveStreamByRoomID 根据房间ID获取直播间信息
// @Summary 根据房间ID获取直播间信息
// @Tags 直播
// @Produce json
// @Param room_id path string true "房间ID"
// @Success 200 {object} response.Response{data=model.LiveStream}
// @Router /api/v1/live/room/{room_id} [get]
func (h *LiveStreamHandler) GetLiveStreamByRoomID(c *gin.Context) {
	// 获取房间ID
	roomID := c.Param("room_id")
	if roomID == "" {
		response.InvalidParam(c, "房间ID不能为空")
		return
	}

	// 调用服务查询直播间
	liveStream, err := h.liveService.GetLiveStreamByRoomID(c.Request.Context(), roomID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, liveStream)
}

// GetMyLiveStream 获取我的直播间
// @Summary 获取我的直播间
// @Tags 直播
// @Produce json
// @Success 200 {object} response.Response{data=model.LiveStream}
// @Router /api/v1/live/my [get]
func (h *LiveStreamHandler) GetMyLiveStream(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 调用服务查询直播间
	liveStream, err := h.liveService.GetMyLiveStream(c.Request.Context(), userID.(uint))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, liveStream)
}

// ListLiveStreams 获取直播列表
// @Summary 获取直播列表
// @Tags 直播
// @Produce json
// @Param status query string false "状态过滤（waiting/live/ended）"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=[]model.LiveStream}
// @Router /api/v1/live/list [get]
func (h *LiveStreamHandler) ListLiveStreams(c *gin.Context) {
	// 获取查询参数
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务查询列表
	liveStreams, total, err := h.liveService.ListLiveStreams(c.Request.Context(), status, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, liveStreams, total, page, pageSize)
}

// JoinLiveStream 加入直播间
// @Summary 加入直播间
// @Tags 直播
// @Produce json
// @Param room_id path string true "房间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/join/{room_id} [post]
func (h *LiveStreamHandler) JoinLiveStream(c *gin.Context) {
	// 获取房间ID
	roomID := c.Param("room_id")
	if roomID == "" {
		response.InvalidParam(c, "房间ID不能为空")
		return
	}

	// 获取当前用户ID（可选，用于统计）
	var userID uint = 0
	if uid, exists := c.Get("uid"); exists {
		userID = uid.(uint)
	}

	// 调用服务加入直播间
	if err := h.liveService.JoinLiveStream(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "加入直播间成功", nil)
}

// LeaveLiveStream 离开直播间
// @Summary 离开直播间
// @Tags 直播
// @Produce json
// @Param room_id path string true "房间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/leave/{room_id} [post]
func (h *LiveStreamHandler) LeaveLiveStream(c *gin.Context) {
	// 获取房间ID
	roomID := c.Param("room_id")
	if roomID == "" {
		response.InvalidParam(c, "房间ID不能为空")
		return
	}

	// 获取当前用户ID（可选，用于统计）
	var userID uint = 0
	if uid, exists := c.Get("uid"); exists {
		userID = uid.(uint)
	}

	// 调用服务离开直播间
	if err := h.liveService.LeaveLiveStream(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "离开直播间成功", nil)
}

// IncrementLike 点赞直播
// @Summary 点赞直播
// @Tags 直播
// @Produce json
// @Param id path int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/{id}/like [post]
func (h *LiveStreamHandler) IncrementLike(c *gin.Context) {
	// 获取直播间ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	// 调用服务增加点赞
	if err := h.liveService.IncrementLike(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "点赞成功", nil)
}

// DeleteLiveStream 删除直播间
// @Summary 删除直播间
// @Tags 直播
// @Produce json
// @Param id path int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/{id} [delete]
func (h *LiveStreamHandler) DeleteLiveStream(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取直播间ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	// 调用服务删除直播间
	if err := h.liveService.DeleteLiveStream(c.Request.Context(), userID.(uint), uint(id)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// BanUser 禁言用户
// @Summary 禁言用户
// @Tags 直播
// @Accept json
// @Produce json
// @Param request body service.BanUserRequest true "禁言请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/ban [post]
func (h *LiveStreamHandler) BanUser(c *gin.Context) {
	// 获取当前用户ID（操作者）
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req service.BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if err := h.liveService.BanUser(c.Request.Context(), userID.(uint), &req); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "禁言成功", nil)
}

// UnbanUser 解除禁言
// @Summary 解除禁言
// @Tags 直播
// @Produce json
// @Param live_id query int true "直播间ID"
// @Param user_id query int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/unban [post]
func (h *LiveStreamHandler) UnbanUser(c *gin.Context) {
	// 获取当前用户ID（操作者）
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	liveIDStr := c.Query("live_id")
	targetUserIDStr := c.Query("user_id")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的用户ID")
		return
	}

	if err := h.liveService.UnbanUser(c.Request.Context(), userID.(uint), uint(liveID), uint(targetUserID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "解除禁言成功", nil)
}

// CheckBanned 检查用户是否被禁言
// @Summary 检查用户是否被禁言
// @Tags 直播
// @Produce json
// @Param live_id query int true "直播间ID"
// @Param user_id query int false "用户ID（默认当前用户）"
// @Success 200 {object} response.Response
// @Router /api/v1/live/check-banned [get]
func (h *LiveStreamHandler) CheckBanned(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	userIDStr := c.Query("user_id")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	var userID uint
	if userIDStr != "" {
		uid, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			response.InvalidParam(c, "无效的用户ID")
			return
		}
		userID = uint(uid)
	} else {
		// 获取当前登录用户ID
		uid, exists := c.Get("uid")
		if !exists {
			response.Unauthorized(c, "未登录")
			return
		}
		userID = uid.(uint)
	}

	isBanned, err := h.liveService.CheckBanned(c.Request.Context(), uint(liveID), userID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"is_banned": isBanned,
	})
}

// ListBans 获取禁言列表
// @Summary 获取禁言列表
// @Tags 直播
// @Produce json
// @Param live_id query int true "直播间ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response
// @Router /api/v1/live/bans [get]
func (h *LiveStreamHandler) ListBans(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	bans, total, err := h.liveService.ListBans(c.Request.Context(), uint(liveID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, bans, total, page, pageSize)
}

// GetHotLiveStreams 获取热门直播间
// @Summary 获取热门直播间
// @Tags 直播
// @Produce json
// @Param limit query int false "数量限制" default(20)
// @Success 200 {object} response.Response{data=[]model.LiveStream}
// @Router /api/v1/live/hot [get]
func (h *LiveStreamHandler) GetHotLiveStreams(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	liveStreams, err := h.liveService.GetHotLiveStreams(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, liveStreams)
}

// ListByCategory 根据分类获取直播间列表
// @Summary 根据分类获取直播间列表
// @Tags 直播
// @Produce json
// @Param category_id query int true "分类ID"
// @Param status query string false "状态过滤（waiting/live/ended）"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=[]model.LiveStream}
// @Router /api/v1/live/category [get]
func (h *LiveStreamHandler) ListByCategory(c *gin.Context) {
	categoryIDStr := c.Query("category_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的分类ID")
		return
	}

	liveStreams, total, err := h.liveService.ListByCategory(c.Request.Context(), uint(categoryID), status, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, liveStreams, total, page, pageSize)
}
