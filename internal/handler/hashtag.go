package handler

import (
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// HashtagHandler 话题处理器
type HashtagHandler struct {
	hashtagService service.HashtagService
}

// NewHashtagHandler 创建话题处理器实例
func NewHashtagHandler(hashtagService service.HashtagService) *HashtagHandler {
	return &HashtagHandler{
		hashtagService: hashtagService,
	}
}

// CreateHashtag 创建话题
func (h *HashtagHandler) CreateHashtag(c *gin.Context) {
	var req service.CreateHashtagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	hashtag, err := h.hashtagService.CreateHashtag(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, hashtag)
}

// GetHashtagDetail 获取话题详情
func (h *HashtagHandler) GetHashtagDetail(c *gin.Context) {
	hashtagIDStr := c.Param("id")
	hashtagID, err := strconv.ParseUint(hashtagIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "话题ID格式错误")
		return
	}

	hashtag, err := h.hashtagService.GetHashtagDetail(c.Request.Context(), uint(hashtagID))
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, hashtag)
}

// GetHotHashtags 获取热门话题
func (h *HashtagHandler) GetHotHashtags(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	hashtags, err := h.hashtagService.GetHotHashtags(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, hashtags)
}

// GetHashtagVideos 获取话题下的视频
func (h *HashtagHandler) GetHashtagVideos(c *gin.Context) {
	hashtagIDStr := c.Param("id")
	hashtagID, err := strconv.ParseUint(hashtagIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "话题ID格式错误")
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

	videos, total, err := h.hashtagService.GetHashtagVideos(c.Request.Context(), uint(hashtagID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, videos, total, page, pageSize)
}
