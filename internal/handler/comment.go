package handler

import (
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CommentHandler 评论处理器
type CommentHandler struct {
	commentService service.CommentService
}

// NewCommentHandler 创建评论处理器实例
func NewCommentHandler(commentService service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req service.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 设置用户ID
	comment.UserID = userID

	response.Success(c, comment)
}

// GetVideoComments 获取视频的评论列表
func (h *CommentHandler) GetVideoComments(c *gin.Context) {
	videoIDStr := c.Param("video_id")
	videoID, err := strconv.ParseUint(videoIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "视频ID格式错误")
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

	comments, total, err := h.commentService.GetVideoComments(c.Request.Context(), uint(videoID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, comments, total, page, pageSize)
}

// GetReplies 获取评论的回复列表
func (h *CommentHandler) GetReplies(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "评论ID格式错误")
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

	replies, err := h.commentService.GetReplies(c.Request.Context(), uint(commentID), page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"replies": replies,
		"page":    page,
		"size":    pageSize,
	})
}

// GetCommentDetail 获取评论详情
func (h *CommentHandler) GetCommentDetail(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "评论ID格式错误")
		return
	}

	comment, err := h.commentService.GetCommentByID(c.Request.Context(), uint(commentID))
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, comment)
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "评论ID格式错误")
		return
	}

	if err := h.commentService.DeleteComment(c.Request.Context(), userID, uint(commentID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// LikeComment 点赞评论
func (h *CommentHandler) LikeComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "评论ID格式错误")
		return
	}

	if err := h.commentService.LikeComment(c.Request.Context(), userID, uint(commentID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "点赞成功", nil)
}

// UnlikeComment 取消点赞评论
func (h *CommentHandler) UnlikeComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录")
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "评论ID格式错误")
		return
	}

	if err := h.commentService.UnlikeComment(c.Request.Context(), userID, uint(commentID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "取消点赞成功", nil)
}
