package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserVisitorHandler 用户访客处理器
type UserVisitorHandler struct {
	visitorService service.UserVisitorService
}

// NewUserVisitorHandler 创建用户访客处理器实例
func NewUserVisitorHandler(visitorService service.UserVisitorService) *UserVisitorHandler {
	return &UserVisitorHandler{
		visitorService: visitorService,
	}
}

// GetVisitors 获取谁访问了我
func (h *UserVisitorHandler) GetVisitors(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	visitors, total, err := h.visitorService.GetVisitors(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取访客记录失败")
		return
	}

	response.PageSuccess(c, visitors, total, page, pageSize)
}

// GetVisited 获取我访问了谁
func (h *UserVisitorHandler) GetVisited(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	visited, total, err := h.visitorService.GetVisited(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ServerError(c, "获取访问历史失败")
		return
	}

	response.PageSuccess(c, visited, total, page, pageSize)
}
