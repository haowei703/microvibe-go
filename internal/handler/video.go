package handler

import (
	"fmt"
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/logger"
	"microvibe-go/pkg/media"
	"microvibe-go/pkg/response"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), userID, resp.Videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, resp.Total, page, pageSize)
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

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), userID, resp.Videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, resp.Total, page, pageSize)
}

// GetFriendsFeed 获取朋友的视频 (互相关注)
func (h *VideoHandler) GetFriendsFeed(c *gin.Context) {
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
		Scene:    "friends",
	})

	if err != nil {
		response.ServerError(c, "获取朋友视频失败: "+err.Error())
		return
	}

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), userID, resp.Videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, resp.Total, page, pageSize)
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

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), userID, resp.Videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, resp.Total, page, pageSize)
}

// CreateVideo 上传视频 (Legacy API via JSON)
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

	req.UserID = userID
	video, err := h.videoService.CreateVideo(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	enrichedVideo, _ := h.videoService.EnrichVideo(c.Request.Context(), userID, video)
	response.Success(c, enrichedVideo)
}

// UploadVideo 上传视频文件并由FFmpeg处理
func (h *VideoHandler) UploadVideo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	file, err := c.FormFile("video")
	if err != nil {
		response.InvalidParam(c, "无法获取上传的视频: "+err.Error())
		return
	}

	title := c.PostForm("title")
	if title == "" {
		title = "未命名视频"
	}
	description := c.PostForm("description")
	categoryIDStr := c.PostForm("category_id")
	tagsStr := c.PostForm("tags")
	isPublicStr := c.PostForm("is_public")
	allowCommentStr := c.PostForm("allow_comment")

	var categoryID *uint
	if categoryIDStr != "" {
		if id, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			uID := uint(id)
			categoryID = &uID
		}
	}

	isPublic := true
	if isPublicStr == "false" || isPublicStr == "0" {
		isPublic = false
	}

	allowComment := true
	if allowCommentStr == "false" || allowCommentStr == "0" {
		allowComment = false
	}

	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	// 获取可选的自定义封面
	coverFile, coverErr := c.FormFile("cover")

	// Create directories
	timestamp := time.Now().Unix()
	baseDir := "./uploads/videos"
	rawDir := filepath.Join(baseDir, "raw")
	hlsDir := filepath.Join(baseDir, "hls", fmt.Sprintf("%d_%d", userID, timestamp))
	coverDir := filepath.Join(baseDir, "covers")
	os.MkdirAll(rawDir, os.ModePerm)
	os.MkdirAll(hlsDir, os.ModePerm)
	os.MkdirAll(coverDir, os.ModePerm)

	filename := fmt.Sprintf("%d_%d_%s", userID, timestamp, file.Filename)
	savePath := filepath.Join(rawDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.ServerError(c, "保存视频失败: "+err.Error())
		return
	}

	coverName := fmt.Sprintf("%d_%d_cover.jpg", userID, timestamp)
	coverPath := filepath.Join(coverDir, coverName)

	duration, w, hght, err := media.GetVideoMetadata(savePath)
	if err != nil {
		response.ServerError(c, "获取视频元数据失败: "+err.Error())
		return
	}

	// 自定义封面覆盖
	if coverErr == nil && coverFile != nil {
		if err := c.SaveUploadedFile(coverFile, coverPath); err != nil {
			response.ServerError(c, "保存自定义封面失败: "+err.Error())
			return
		}
	} else {
		// 自动抽取首帧
		if err := media.ExtractCover(savePath, coverPath); err != nil {
			response.ServerError(c, "提取视频封面失败: "+err.Error())
			return
		}
	}

	// 构造相对路径 URL
	videoURL := fmt.Sprintf("/uploads/videos/hls/%d_%d/index.m3u8", userID, timestamp)
	coverURL := fmt.Sprintf("/uploads/videos/covers/%s", coverName)

	req := service.CreateVideoRequest{
		UserID:       userID,
		Title:        title,
		Description:  description,
		VideoURL:     videoURL,
		CoverURL:     coverURL,
		Duration:     duration,
		Width:        w,
		Height:       hght,
		FileSize:     file.Size,
		CategoryID:   categoryID,
		Tags:         tags,
		IsPublic:     &isPublic,
		AllowComment: &allowComment,
	}

	video, err := h.videoService.CreateVideo(c.Request.Context(), &req)
	if err != nil {
		// 数据库记录创建失败，清理已上传的文件和预创目录
		os.RemoveAll(hlsDir)
		os.Remove(coverPath)
		os.Remove(savePath)
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 只有在数据库记录创建成功后，才开始异步转码
	go func() {
		if err := media.TranscodeToHLS(savePath, hlsDir); err != nil {
			logger.Error("视频 HLS 转码失败", zap.Error(err), zap.String("path", savePath))
		} else {
			// 转码成功后删除原始视频文件以节省空间
			os.Remove(savePath)
			logger.Info("视频 HLS 转码成功，已清理原始文件", zap.String("path", savePath))
		}
	}()

	enrichedVideo, _ := h.videoService.EnrichVideo(c.Request.Context(), userID, video)
	response.Success(c, enrichedVideo)
}

// AuditVideo 审核视频
func (h *VideoHandler) AuditVideo(c *gin.Context) {
	// 这里可以加上管理员中间件校验，目前简化为一个直接调用的端口
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	var req struct {
		Status int8 `json:"status" binding:"required"` // 1: 通过, 2: 拒绝
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "状态参数错误: "+err.Error())
		return
	}

	err = h.videoService.UpdateVideoStatus(c.Request.Context(), uint(videoID), req.Status)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "审核状态更新成功", nil)
}

// GetVideoDetail 获取视频详情
func (h *VideoHandler) GetVideoDetail(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	// 获取用户ID（用于检查是否已点赞等互动状态）
	userID, _ := middleware.GetUserID(c)

	video, err := h.videoService.GetVideoByID(c.Request.Context(), uint(videoID))
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	// 丰富视频信息
	enrichedVideo, _ := h.videoService.EnrichVideo(c.Request.Context(), userID, video)
	response.Success(c, enrichedVideo)
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

// GetUserVideos 获取用户的视频列表（公开作品）
func (h *VideoHandler) GetUserVideos(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 获取当前登录用户ID (用于丰富视频信息)
	currentUserID, _ := middleware.GetUserID(c)

	videos, total, err := h.videoService.GetUserVideos(c.Request.Context(), uint(userID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), currentUserID, videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, total, page, pageSize)
}

// GetMyVideos 获取当前登录用户自己的视频列表
func (h *VideoHandler) GetMyVideos(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	videos, total, err := h.videoService.GetMyVideos(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, videos, total, page, pageSize)
}

// GetUserFavorites 获取用户收藏的视频列表
func (h *VideoHandler) GetUserFavorites(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
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

	// 获取当前登录用户ID (用于丰富视频信息，如点赞收藏状态)
	currentUserID, _ := middleware.GetUserID(c)

	// 使用带隐私检查的方法
	videos, total, err := h.videoService.GetUserFavoriteVideosWithPrivacy(c.Request.Context(), uint(userID), currentUserID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 丰富视频信息
	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), currentUserID, videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, total, page, pageSize)
}

// RecordPlay 记录一次播放（前端在每次新播放会话开始时调用）
func (h *VideoHandler) RecordPlay(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	if err := h.videoService.RecordPlay(c.Request.Context(), uint(videoID)); err != nil {
		response.ServerError(c, "记录播放失败")
		return
	}

	response.Success(c, nil)
}

// GetVideoLikers 获取点赞视频的用户列表
func (h *VideoHandler) GetVideoLikers(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	authors, total, err := h.videoService.GetVideoLikers(c.Request.Context(), uint(videoID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, authors, total, page, pageSize)
}

// GetVideoFavoriters 获取收藏视频的用户列表
func (h *VideoHandler) GetVideoFavoriters(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	authors, total, err := h.videoService.GetVideoFavoriters(c.Request.Context(), uint(videoID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, authors, total, page, pageSize)
}

// GetUserLikes 获取用户点赞的视频列表
func (h *VideoHandler) GetUserLikes(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
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

	currentUserID, _ := middleware.GetUserID(c)

	// 使用带隐私检查的方法
	videos, total, err := h.videoService.GetUserLikedVideosWithPrivacy(c.Request.Context(), uint(userID), currentUserID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	enrichedVideos, err := h.videoService.EnrichVideoList(c.Request.Context(), currentUserID, videos)
	if err != nil {
		response.ServerError(c, "处理视频信息失败: "+err.Error())
		return
	}

	response.PageSuccess(c, enrichedVideos, total, page, pageSize)
}
