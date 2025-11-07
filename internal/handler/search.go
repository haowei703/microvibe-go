package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SearchHandler 搜索处理器
type SearchHandler struct {
	searchService service.SearchService
}

// NewSearchHandler 创建搜索处理器实例
func NewSearchHandler(searchService service.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

// Search 综合搜索
func (h *SearchHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "搜索关键词不能为空")
		return
	}

	category := c.DefaultQuery("category", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取用户ID（可选）
	userID, _ := middleware.GetUserID(c)
	var userIDPtr *uint
	if userID > 0 {
		userIDPtr = &userID
	}

	req := &service.SearchRequest{
		Keyword:  keyword,
		Category: category,
		Page:     page,
		PageSize: pageSize,
		UserID:   userIDPtr,
	}

	result, err := h.searchService.Search(c.Request.Context(), req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, result)
}

// SearchVideos 搜索视频
func (h *SearchHandler) SearchVideos(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	videos, total, err := h.searchService.SearchVideos(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, videos, total, page, pageSize)
}

// SearchUsers 搜索用户
func (h *SearchHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.searchService.SearchUsers(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, users, total, page, pageSize)
}

// SearchHashtags 搜索话题
func (h *SearchHandler) SearchHashtags(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	hashtags, total, err := h.searchService.SearchHashtags(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, hashtags, total, page, pageSize)
}

// GetSearchHistory 获取搜索历史
func (h *SearchHandler) GetSearchHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	histories, err := h.searchService.GetSearchHistory(c.Request.Context(), userID, limit)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, histories)
}

// ClearSearchHistory 清空搜索历史
func (h *SearchHandler) ClearSearchHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.searchService.ClearSearchHistory(c.Request.Context(), userID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "清空成功", nil)
}

// GetHotSearches 获取热搜榜
func (h *SearchHandler) GetHotSearches(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	hotSearches, err := h.searchService.GetHotSearches(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, hotSearches)
}

// GetSearchSuggestions 获取搜索建议
func (h *SearchHandler) GetSearchSuggestions(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.InvalidParam(c, "关键词不能为空")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 10 {
		limit = 10
	}

	suggestions, err := h.searchService.GetSearchSuggestions(c.Request.Context(), keyword, limit)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"suggestions": suggestions,
	})
}
