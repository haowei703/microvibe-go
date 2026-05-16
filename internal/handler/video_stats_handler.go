package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// VideoStatsHandler 视频统计处理器
type VideoStatsHandler struct {
	statsService service.VideoStatsService
}

// NewVideoStatsHandler 创建视频统计处理器实例
func NewVideoStatsHandler(statsService service.VideoStatsService) *VideoStatsHandler {
	return &VideoStatsHandler{statsService: statsService}
}

// GetVideoStats 获取视频总览统计
func (h *VideoStatsHandler) GetVideoStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	stats, err := h.statsService.GetVideoStats(c.Request.Context(), userID, uint(videoID))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetVideoDailyStats 获取视频每日趋势统计
func (h *VideoStatsHandler) GetVideoDailyStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	stats, err := h.statsService.GetVideoDailyStats(c.Request.Context(), userID, uint(videoID), days)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetCreatorStats 获取创作者所有视频汇总统计
func (h *VideoStatsHandler) GetCreatorStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	stats, err := h.statsService.GetCreatorStats(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetCreatorTrendingStats 获取创作者近期趋势统计
func (h *VideoStatsHandler) GetCreatorTrendingStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	stats, err := h.statsService.GetCreatorTrendingStats(c.Request.Context(), userID, days)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, stats)
}
