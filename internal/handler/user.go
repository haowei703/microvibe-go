package handler

import (
	"microvibe-go/internal/config"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/response"
	"microvibe-go/pkg/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService service.UserService
	cfg         *config.Config
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		cfg:         cfg,
	}
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	user, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 生成 Token
	token, err := utils.GenerateToken(user.ID, user.Username, h.cfg.JWT.Secret, h.cfg.JWT.Expire)
	if err != nil {
		response.ServerError(c, "生成Token失败")
		return
	}

	response.Success(c, gin.H{
		"user":  user,
		"token": token,
	})
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	user, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	// 生成 Token
	token, err := utils.GenerateToken(user.ID, user.Username, h.cfg.JWT.Secret, h.cfg.JWT.Expire)
	if err != nil {
		response.ServerError(c, "生成Token失败")
		return
	}

	response.Success(c, gin.H{
		"user":  user,
		"token": token,
	})
}

// GetUserInfo 获取用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), uint(userID))
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, user)
}

// GetCurrentUser 获取当前登录用户信息
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, user)
}

// UpdateUserInfo 更新用户信息
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if err := h.userService.UpdateUser(c.Request.Context(), userID, updates); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Follow 关注用户
func (h *UserHandler) Follow(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	targetIDStr := c.Param("id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	if err := h.userService.FollowUser(c.Request.Context(), userID, uint(targetID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "关注成功", nil)
}

// Unfollow 取消关注
func (h *UserHandler) Unfollow(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	targetIDStr := c.Param("id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	if err := h.userService.UnfollowUser(c.Request.Context(), userID, uint(targetID)); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "取消关注成功", nil)
}
