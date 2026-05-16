package handler

import (
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// AuditVideo 审核视频
func (h *AdminHandler) AuditVideo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Status int8 `json:"status" binding:"required"` // 1:通过, 2:下架/拒绝
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.AuditVideo(c.Request.Context(), uint(id), req.Status); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// AuditComment 审核评论
func (h *AdminHandler) AuditComment(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Status int8 `json:"status" binding:"required"` // 1:通过, 2:下架
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.AuditComment(c.Request.Context(), uint(id), req.Status); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// UpdateUserStatus 更新用户状态 (禁言/封号)
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Status int8 `json:"status" binding:"required"` // 1:正常, 2:禁用/禁言
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.UpdateUserStatus(c.Request.Context(), uint(id), req.Status); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// SetVideoTop 设置视频置顶
func (h *AdminHandler) SetVideoTop(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		IsTop bool `json:"is_top"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.SetVideoTop(c.Request.Context(), uint(id), req.IsTop); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// UpdateVideoWeight 调整视频推荐权重
func (h *AdminHandler) UpdateVideoWeight(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		HotScore float64 `json:"hot_score" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.UpdateVideoRecommendWeight(c.Request.Context(), uint(id), req.HotScore); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// DeleteHotSearch 删除热搜词
func (h *AdminHandler) DeleteHotSearch(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "keyword is required")
		return
	}
	if err := h.adminService.DeleteHotSearch(c.Request.Context(), keyword); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// UpdateHotSearchWeight 调整热搜词热度
func (h *AdminHandler) UpdateHotSearchWeight(c *gin.Context) {
	var req struct {
		Keyword string `json:"keyword" binding:"required"`
		Count   int64  `json:"count" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}
	if err := h.adminService.UpdateHotSearchWeight(c.Request.Context(), req.Keyword, req.Count); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

// ListVideos 分页获取视频列表
func (h *AdminHandler) ListVideos(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	var statusPtr *int8
	if s := c.Query("status"); s != "" {
		st, _ := strconv.Atoi(s)
		st8 := int8(st)
		statusPtr = &st8
	}

	videos, total, err := h.adminService.ListVideos(c.Request.Context(), page, pageSize, statusPtr)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      videos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListUsers 分页获取用户列表
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	query := c.Query("query")

	users, total, err := h.adminService.ListUsers(c.Request.Context(), page, pageSize, query)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListHotSearches 获取热搜列表
func (h *AdminHandler) ListHotSearches(c *gin.Context) {
	list, err := h.adminService.ListHotSearches(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, list)
}

// ListReports 获取举报列表
func (h *AdminHandler) ListReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))

	list, total, err := h.adminService.ListReports(c.Request.Context(), page, pageSize, int8(status))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
