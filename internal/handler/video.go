package handler

import (
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// VideoHandler 视频处理器
type VideoHandler struct {
	recommendEngine *recommend.Engine
	videoService    service.VideoService
}

// NewVideoHandler 创建视频处理器实例
func NewVideoHandler(recommendEngine *recommend.Engine, videoService service.VideoService) *VideoHandler {
	return &VideoHandler{
		recommendEngine: recommendEngine,
		videoService:    videoService,
	}
}

// GetRecommendFeed 获取推荐视频流
func (h *VideoHandler) GetRecommendFeed(c *gin.Context) {
	// 获取用户ID（可选）
	userID, _ := middleware.GetUserID(c)

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取推荐视频
	resp, err := h.recommendEngine.Recommend(c.Request.Context(), &recommend.RecommendRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Scene:    "feed",
	})

	if err != nil {
		response.ServerError(c, "获取推荐失败: "+err.Error())
		return
	}

	response.PageSuccess(c, resp.Videos, resp.Total, page, pageSize)
}

// GetFollowFeed 获取关注的人的视频
func (h *VideoHandler) GetFollowFeed(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
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

	resp, err := h.recommendEngine.Recommend(c.Request.Context(), &recommend.RecommendRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Scene:    "follow",
	})

	if err != nil {
		response.ServerError(c, "获取关注视频失败: "+err.Error())
		return
	}

	response.PageSuccess(c, resp.Videos, resp.Total, page, pageSize)
}

// GetHotFeed 获取热门视频
func (h *VideoHandler) GetHotFeed(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.recommendEngine.Recommend(c.Request.Context(), &recommend.RecommendRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Scene:    "hot",
	})

	if err != nil {
		response.ServerError(c, "获取热门视频失败: "+err.Error())
		return
	}

	response.PageSuccess(c, resp.Videos, resp.Total, page, pageSize)
}

// CreateVideo 上传视频
func (h *VideoHandler) CreateVideo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req service.CreateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	video, err := h.videoService.CreateVideo(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 设置用户ID
	video.UserID = userID

	response.Success(c, video)
}

// GetVideoDetail 获取视频详情
func (h *VideoHandler) GetVideoDetail(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	video, err := h.videoService.GetVideoByID(c.Request.Context(), uint(videoID))
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, video)
}

// UpdateVideo 更新视频信息
func (h *VideoHandler) UpdateVideo(c *gin.Context) {
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

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if err := h.videoService.UpdateVideo(c.Request.Context(), userID, uint(videoID), updates); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteVideo 删除视频
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
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

	if err := h.videoService.DeleteVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// LikeVideo 点赞视频
func (h *VideoHandler) LikeVideo(c *gin.Context) {
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

	if err := h.videoService.LikeVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "点赞成功", nil)
}

// UnlikeVideo 取消点赞
func (h *VideoHandler) UnlikeVideo(c *gin.Context) {
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

	if err := h.videoService.UnlikeVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "取消点赞成功", nil)
}

// FavoriteVideo 收藏视频
func (h *VideoHandler) FavoriteVideo(c *gin.Context) {
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

	if err := h.videoService.FavoriteVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "收藏成功", nil)
}

// UnfavoriteVideo 取消收藏
func (h *VideoHandler) UnfavoriteVideo(c *gin.Context) {
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

	if err := h.videoService.UnfavoriteVideo(c.Request.Context(), userID, uint(videoID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "取消收藏成功", nil)
}

// GetUserVideos 获取用户的视频列表
func (h *VideoHandler) GetUserVideos(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	videos, total, err := h.videoService.GetUserVideos(c.Request.Context(), uint(userID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, videos, total, page, pageSize)
}
