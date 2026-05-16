package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// VideoHistoryHandler 视频播放历史处理器
type VideoHistoryHandler struct {
	historyService service.VideoHistoryService
}

// NewVideoHistoryHandler 创建视频播放历史处理器实例
func NewVideoHistoryHandler(historyService service.VideoHistoryService) *VideoHistoryHandler {
	return &VideoHistoryHandler{
		historyService: historyService,
	}
}

// PlaybackHistoryRequest 上报播放进度请求
type PlaybackHistoryRequest struct {
	Position int  `json:"position" binding:"required,min=0"` // 播放进度 (秒)
	Duration int  `json:"duration" binding:"required,min=1"` // 视频总时长 (秒)
	Finished bool `json:"finished"`                          // 是否播放完成
}

// ReportProgress 上报播放进度
func (h *VideoHistoryHandler) ReportProgress(c *gin.Context) {
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

	var req PlaybackHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	finished, err := h.historyService.ReportProgress(c.Request.Context(), userID, uint(videoID), req.Position, req.Duration, req.Finished)
	if err != nil {
		response.ServerError(c, "上报播放历史失败")
		return
	}

	response.Success(c, gin.H{
		"finished": finished,
	})
}

// GetHistory 获取播放历史
func (h *VideoHistoryHandler) GetHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var finished *bool
	finishedStr := c.Query("finished")
	if finishedStr != "" {
		b := finishedStr == "true" || finishedStr == "1"
		finished = &b
	}

	histories, total, err := h.historyService.GetHistory(c.Request.Context(), userID, page, pageSize, finished)
	if err != nil {
		response.ServerError(c, "获取播放历史失败")
		return
	}

	response.PageSuccess(c, histories, total, page, pageSize)
}

// DeleteHistory 删除播放历史
func (h *VideoHistoryHandler) DeleteHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	historyIDStr := c.Param("id")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "历史ID格式错误")
		return
	}

	if err := h.historyService.DeleteHistory(c.Request.Context(), userID, uint(historyID)); err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.Success(c, nil)
}

// ClearHistory 清空播放历史
func (h *VideoHistoryHandler) ClearHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.historyService.ClearHistory(c.Request.Context(), userID); err != nil {
		response.ServerError(c, "清空失败")
		return
	}

	response.Success(c, nil)
}
