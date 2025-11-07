package handler

import (
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LiveFansClubHandler 粉丝团Handler
type LiveFansClubHandler struct {
	fansClubService service.LiveFansClubService
}

// NewLiveFansClubHandler 创建粉丝团Handler
func NewLiveFansClubHandler(fansClubService service.LiveFansClubService) *LiveFansClubHandler {
	return &LiveFansClubHandler{
		fansClubService: fansClubService,
	}
}

// JoinFansClub 加入粉丝团
// @Summary 加入粉丝团
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/join [post]
func (h *LiveFansClubHandler) JoinFansClub(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	liveIDStr := c.Query("live_id")
	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	member, err := h.fansClubService.JoinFansClub(c.Request.Context(), userID.(uint), uint(liveID))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "加入粉丝团成功", member)
}

// QuitFansClub 退出粉丝团
// @Summary 退出粉丝团
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/quit [post]
func (h *LiveFansClubHandler) QuitFansClub(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	liveIDStr := c.Query("live_id")
	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	if err := h.fansClubService.QuitFansClub(c.Request.Context(), userID.(uint), uint(liveID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "退出粉丝团成功", nil)
}

// GetMemberInfo 获取粉丝团成员信息
// @Summary 获取粉丝团成员信息
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/member [get]
func (h *LiveFansClubHandler) GetMemberInfo(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	liveIDStr := c.Query("live_id")
	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	member, err := h.fansClubService.GetMemberInfo(c.Request.Context(), userID.(uint), uint(liveID))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, member)
}

// ListMembers 获取粉丝团成员列表
// @Summary 获取粉丝团成员列表
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/members [get]
func (h *LiveFansClubHandler) ListMembers(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	members, total, err := h.fansClubService.ListMembers(c.Request.Context(), uint(liveID), pageInt, pageSizeInt)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, members, total, pageInt, pageSizeInt)
}

// GetTopMembers 获取粉丝团排行榜
// @Summary 获取粉丝团排行榜
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Param limit query int false "数量限制"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/top [get]
func (h *LiveFansClubHandler) GetTopMembers(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	limit := c.DefaultQuery("limit", "10")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	limitInt, _ := strconv.Atoi(limit)

	members, err := h.fansClubService.GetTopMembers(c.Request.Context(), uint(liveID), limitInt)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, members)
}

// GetMemberCount 获取粉丝团人数
// @Summary 获取粉丝团人数
// @Tags 粉丝团
// @Param live_id query int true "直播间ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/fans-club/count [get]
func (h *LiveFansClubHandler) GetMemberCount(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	count, err := h.fansClubService.GetMemberCount(c.Request.Context(), uint(liveID))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}
