package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ShareHandler struct {
	shareService service.ShareService
}

func NewShareHandler(shareService service.ShareService) *ShareHandler {
	return &ShareHandler{shareService: shareService}
}

// ShareVideo POST /api/v1/videos/:id/share
func (h *ShareHandler) ShareVideo(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	videoIDStr := c.Param("id")
	videoID, _ := strconv.ParseUint(videoIDStr, 10, 64)

	var req struct {
		Platform string `json:"platform"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许不传 platform，默认为 internal
	}

	if req.Platform == "" {
		req.Platform = "internal"
	}

	if err := h.shareService.ShareVideo(c.Request.Context(), userID, uint(videoID), req.Platform); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "分享成功", nil)
}

// GetShareCount GET /api/v1/videos/:id/share/count
func (h *ShareHandler) GetShareCount(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, _ := strconv.ParseUint(videoIDStr, 10, 64)

	count, err := h.shareService.GetVideoShareCount(c.Request.Context(), uint(videoID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取分享数失败")
		return
	}

	response.Success(c, gin.H{"count": count})
}
