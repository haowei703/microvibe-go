package handler

import (
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LiveGiftHandler 直播礼物Handler
type LiveGiftHandler struct {
	giftService service.LiveGiftService
}

// NewLiveGiftHandler 创建礼物Handler
func NewLiveGiftHandler(giftService service.LiveGiftService) *LiveGiftHandler {
	return &LiveGiftHandler{
		giftService: giftService,
	}
}

// ListGifts 获取礼物列表
// @Summary 获取礼物列表
// @Tags 直播礼物
// @Param gift_type query int false "礼物类型"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts [get]
func (h *LiveGiftHandler) ListGifts(c *gin.Context) {
	giftType := c.DefaultQuery("gift_type", "0")
	status := c.DefaultQuery("status", "1")

	giftTypeInt, _ := strconv.Atoi(giftType)
	statusInt, _ := strconv.Atoi(status)

	gifts, err := h.giftService.ListGifts(c.Request.Context(), int8(giftTypeInt), int8(statusInt))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gifts)
}

// GetGift 获取礼物详情
// @Summary 获取礼物详情
// @Tags 直播礼物
// @Param id path int true "礼物ID"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts/:id [get]
func (h *LiveGiftHandler) GetGift(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的礼物ID")
		return
	}

	gift, err := h.giftService.GetGiftByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gift)
}

// SendGift 送礼
// @Summary 送礼
// @Tags 直播礼物
// @Accept json
// @Param request body service.SendGiftRequest true "送礼请求"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts/send [post]
func (h *LiveGiftHandler) SendGift(c *gin.Context) {
	// 获取当前登录用户ID
	userID, exists := c.Get("uid")
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req service.SendGiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	record, err := h.giftService.SendGift(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "送礼成功", record)
}

// ListGiftRecords 获取送礼记录
// @Summary 获取送礼记录
// @Tags 直播礼物
// @Param live_id query int true "直播间ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts/records [get]
func (h *LiveGiftHandler) ListGiftRecords(c *gin.Context) {
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

	records, total, err := h.giftService.ListGiftRecords(c.Request.Context(), uint(liveID), pageInt, pageSizeInt)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, records, total, pageInt, pageSizeInt)
}

// GetTopGivers 获取送礼榜单
// @Summary 获取送礼榜单
// @Tags 直播礼物
// @Param live_id query int true "直播间ID"
// @Param limit query int false "数量限制"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts/top [get]
func (h *LiveGiftHandler) GetTopGivers(c *gin.Context) {
	liveIDStr := c.Query("live_id")
	limit := c.DefaultQuery("limit", "10")

	liveID, err := strconv.ParseUint(liveIDStr, 10, 32)
	if err != nil {
		response.InvalidParam(c, "无效的直播间ID")
		return
	}

	limitInt, _ := strconv.Atoi(limit)

	records, err := h.giftService.GetTopGivers(c.Request.Context(), uint(liveID), limitInt)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, records)
}

// GetUserGiftStats 获取用户送礼统计
// @Summary 获取用户送礼统计
// @Tags 直播礼物
// @Param live_id query int true "直播间ID"
// @Param user_id query int false "用户ID（默认当前用户）"
// @Success 200 {object} response.Response
// @Router /api/v1/live/gifts/stats [get]
func (h *LiveGiftHandler) GetUserGiftStats(c *gin.Context) {
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

	totalValue, giftCount, err := h.giftService.GetUserGiftStats(c.Request.Context(), uint(liveID), userID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"total_value": totalValue,
		"gift_count":  giftCount,
	})
}
