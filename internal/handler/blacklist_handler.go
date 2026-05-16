package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BlacklistHandler struct {
	blacklistService service.BlacklistService
}

func NewBlacklistHandler(blacklistService service.BlacklistService) *BlacklistHandler {
	return &BlacklistHandler{blacklistService: blacklistService}
}

// BlockUser POST /api/v1/user/blacklist
func (h *BlacklistHandler) BlockUser(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		BlockedUserID uint `json:"blocked_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	if err := h.blacklistService.BlockUser(c.Request.Context(), userID, req.BlockedUserID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "已将用户拉入黑名单", nil)
}

// UnblockUser DELETE /api/v1/user/blacklist/:id
func (h *BlacklistHandler) UnblockUser(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	blockedUserIDStr := c.Param("id")
	blockedUserID, _ := strconv.ParseUint(blockedUserIDStr, 10, 64)

	if err := h.blacklistService.UnblockUser(c.Request.Context(), userID, uint(blockedUserID)); err != nil {
		response.Error(c, http.StatusInternalServerError, "取消拉黑失败")
		return
	}

	response.SuccessWithMessage(c, "已将用户移出黑名单", nil)
}

// GetBlacklist GET /api/v1/user/blacklist
func (h *BlacklistHandler) GetBlacklist(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.blacklistService.GetBlacklist(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取黑名单失败")
		return
	}

	response.Success(c, gin.H{
		"items": list,
		"total": total,
	})
}
