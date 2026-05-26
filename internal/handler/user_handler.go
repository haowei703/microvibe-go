package handler

import (
	"context"
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
	userService    service.UserService
	visitorService service.UserVisitorService
	cfg            *config.Config
	tokenBlacklist *middleware.TokenBlacklist
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, visitorService service.UserVisitorService, cfg *config.Config, blacklist *middleware.TokenBlacklist) *UserHandler {
	return &UserHandler{
		userService:    userService,
		visitorService: visitorService,
		cfg:            cfg,
		tokenBlacklist: blacklist,
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
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, h.cfg.JWT.Secret, h.cfg.JWT.Expire)
	if err != nil {
		response.ServerError(c, "生成Token失败")
		return
	}

	response.Success(c, gin.H{
		"user":  user.ToVO(false),
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
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, h.cfg.JWT.Secret, h.cfg.JWT.Expire)
	if err != nil {
		response.ServerError(c, "生成Token失败")
		return
	}

	response.Success(c, gin.H{
		"user":  user.ToVO(false),
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

	currentUserID, _ := middleware.GetUserID(c)

	userVO, err := h.userService.GetUserWithFollowStatus(c.Request.Context(), uint(userID), currentUserID)
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	// 记录访客历史（异步）
	if currentUserID > 0 && currentUserID != uint(userID) {
		go func() {
			_ = h.visitorService.RecordVisit(context.Background(), uint(userID), currentUserID)
		}()
	}

	response.Success(c, userVO)
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

	response.Success(c, user.ToVO(false))
}

// UpdateUserInfo 更新用户信息
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if err := h.userService.UpdateUser(c.Request.Context(), userID, &req); err != nil {
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

// GetUserFollowings 获取用户关注列表
func (h *UserHandler) GetUserFollowings(c *gin.Context) {
	targetIDStr := c.Param("id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	currentUserID, _ := middleware.GetUserID(c)

	users, total, err := h.userService.GetUserFollowings(c.Request.Context(), uint(targetID), currentUserID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, users, total, page, pageSize)
}

// GetUserFollowers 获取用户粉丝列表
func (h *UserHandler) GetUserFollowers(c *gin.Context) {
	targetIDStr := c.Param("id")
	targetID, err := strconv.ParseUint(targetIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "用户ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	currentUserID, _ := middleware.GetUserID(c)

	users, total, err := h.userService.GetUserFollowers(c.Request.Context(), uint(targetID), currentUserID, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.PageSuccess(c, users, total, page, pageSize)
}

// UpdatePrivacySettings 更新隐私设置
func (h *UserHandler) UpdatePrivacySettings(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	if req.ShowFavorites == nil && req.ShowLikes == nil &&
		req.ShowFollowing == nil && req.ShowFollowers == nil {
		response.InvalidParam(c, "至少需要更新一个隐私设置")
		return
	}

	if err := h.userService.UpdateUser(c.Request.Context(), userID, &req); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "隐私设置更新成功", nil)
}

// Logout 用户登出，将当前 token 加入黑名单
func (h *UserHandler) Logout(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		response.SuccessWithMessage(c, "已登出", nil)
		return
	}

	jwtClaims, ok := claims.(*utils.Claims)
	if !ok {
		response.SuccessWithMessage(c, "已登出", nil)
		return
	}

	if h.tokenBlacklist != nil {
		ttl := middleware.TTL(jwtClaims.ExpiresAt.Time)
		if err := h.tokenBlacklist.Add(c.Request.Context(), jwtClaims.JTI, ttl); err != nil {
			response.Error(c, response.CodeError, "登出失败")
			return
		}
	}

	response.SuccessWithMessage(c, "登出成功", nil)
}
