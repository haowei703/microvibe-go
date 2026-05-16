package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"
	"microvibe-go/pkg/utils"

	"go.uber.org/zap"
)

// UserService 用户服务层接口
type UserService interface {
	// Register 用户注册
	Register(ctx context.Context, req *RegisterRequest) (*model.User, error)
	// Login 用户登录
	Login(ctx context.Context, req *LoginRequest) (*model.User, error)
	// GetUserByID 根据ID获取用户信息
	GetUserByID(ctx context.Context, userID uint) (*model.User, error)
	// GetUserWithFollowStatus 根据ID获取用户信息（包含是否已关注）
	GetUserWithFollowStatus(ctx context.Context, targetUserID, currentUserID uint) (*model.UserVO, error)
	// GetUserByEmail 根据Email获取用户信息
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, userID uint, req *UpdateUserRequest) error
	// FollowUser 关注用户
	FollowUser(ctx context.Context, userID, targetID uint) error
	// UnfollowUser 取消关注
	UnfollowUser(ctx context.Context, userID, targetID uint) error
	// GetUserFollowings 获取用户关注列表（带隐私检查）
	GetUserFollowings(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.UserFollowVO, int64, error)
	// GetUserFollowers 获取用户粉丝列表（带隐私检查）
	GetUserFollowers(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.UserFollowVO, int64, error)
}

// userServiceImpl 用户服务层实现
type userServiceImpl struct {
	userRepo       repository.UserRepository
	followRepo     repository.FollowRepository
	profileRepo    repository.ProfileRepository
	messageService MessageService
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, followRepo repository.FollowRepository, profileRepo repository.ProfileRepository) UserService {
	return &userServiceImpl{
		userRepo:    userRepo,
		followRepo:  followRepo,
		profileRepo: profileRepo,
	}
}

// SetMessageService 设置消息服务
func (s *userServiceImpl) SetMessageService(messageService MessageService) {
	s.messageService = messageService
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Nickname string `json:"nickname"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateUserRequest 更新用户信息请求
type UpdateUserRequest struct {
	Nickname        *string    `json:"nickname"`
	Bio             *string    `json:"bio"`
	Avatar          *string    `json:"avatar"`
	BackgroundImage *string    `json:"background_image"`
	Gender          *int8      `json:"gender"`
	Birthday        *time.Time `json:"birthday"`
	Province        *string    `json:"province"`
	City            *string    `json:"city"`
	ShowFavorites   *bool      `json:"show_favorites"`
	ShowLikes       *bool      `json:"show_likes"`
	ShowFollowing   *bool      `json:"show_following"`
	ShowFollowers   *bool      `json:"show_followers"`
}

// Register 用户注册
func (s *userServiceImpl) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	logger.Info("用户注册请求", zap.String("username", req.Username), zap.String("email", req.Email))

	// 检查用户名是否已存在
	existUser, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err == nil && existUser != nil {
		logger.Warn("用户名已存在", zap.String("username", req.Username))
		return nil, pkgerrors.ErrUserAlreadyExists
	}

	// 检查邮箱是否已存在
	existUser, err = s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existUser != nil {
		logger.Warn("邮箱已被注册", zap.String("email", req.Email))
		return nil, pkgerrors.NewAppError(pkgerrors.CodeDuplicateKey, "邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		logger.Error("密码加密失败", zap.Error(err))
		return nil, pkgerrors.Wrap(err, "密码加密失败")
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Nickname: req.Nickname,
		Status:   1, // 正常状态
	}

	if req.Phone != "" {
		user.Phone = &req.Phone
	}

	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		logger.Error("创建用户失败", zap.Error(err))
		return nil, err
	}

	// 创建用户扩展资料
	profile := &model.UserProfile{
		UserID: user.ID,
	}
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		logger.Error("创建用户资料失败", zap.Error(err), zap.Uint("user_id", user.ID))
	}

	logger.Info("用户注册成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	return user, nil
}

// Login 用户登录
func (s *userServiceImpl) Login(ctx context.Context, req *LoginRequest) (*model.User, error) {
	logger.Info("用户登录请求", zap.String("username", req.Username))

	// 查找用户（支持用户名或邮箱登录）
	user, err := s.userRepo.FindByUsername(ctx, req.Username, false)
	if err != nil {
		// 尝试使用邮箱查找
		user, err = s.userRepo.FindByEmail(ctx, req.Username, false)
		if err != nil {
			if pkgerrors.IsNotFound(err) {
				logger.Warn("用户不存在", zap.String("username", req.Username))
				return nil, pkgerrors.ErrUserNotFound
			}
			logger.Error("查找用户失败", zap.Error(err))
			return nil, pkgerrors.ConvertDBError(err)
		}
	}

	// 验证密码
	if !utils.CheckPassword(req.Password, user.Password) {
		logger.Warn("密码错误", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
		return nil, pkgerrors.ErrInvalidPassword
	}

	// 检查用户状态
	if user.Status != 1 {
		logger.Warn("账号已被禁用", zap.Uint("user_id", user.ID))
		return nil, pkgerrors.NewAppError(pkgerrors.CodeForbidden, "账号已被禁用")
	}

	// 解析简介中的 @ 提及，将 @[userId] 转换为 @[userId:nickname]
	if user.Bio != "" {
		user.Bio = s.parseBioMentions(ctx, user.Bio)
	}

	logger.Info("用户登录成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	return user, nil
}

// GetUserByID 根据ID获取用户信息
func (s *userServiceImpl) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	logger.Debug("获取用户信息", zap.Uint("user_id", userID))

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, pkgerrors.ErrUserNotFound
		}
		logger.Error("获取用户信息失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, pkgerrors.ConvertDBError(err)
	}

	// 解析简介中的 @ 提及，将 @[userId] 转换为 @[userId:nickname]
	if user.Bio != "" {
		user.Bio = s.parseBioMentions(ctx, user.Bio)
	}

	return user, nil
}

// GetUserWithFollowStatus 根据ID获取用户信息（包含当前用户是否已关注）
func (s *userServiceImpl) GetUserWithFollowStatus(ctx context.Context, targetUserID, currentUserID uint) (*model.UserVO, error) {
	logger.Debug("获取用户信息（含关注状态）", zap.Uint("target_user_id", targetUserID), zap.Uint("current_user_id", currentUserID))

	user, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, pkgerrors.ErrUserNotFound
		}
		logger.Error("获取用户信息失败", zap.Error(err), zap.Uint("user_id", targetUserID))
		return nil, pkgerrors.ConvertDBError(err)
	}

	// 解析简介中的 @ 提及
	if user.Bio != "" {
		user.Bio = s.parseBioMentions(ctx, user.Bio)
	}

	vo := &model.UserVO{User: user}

	// 查询当前用户是否已关注目标用户
	if currentUserID > 0 && currentUserID != targetUserID {
		followed, err := s.followRepo.Exists(ctx, currentUserID, targetUserID)
		if err != nil {
			logger.Error("查询关注状态失败", zap.Error(err))
		} else {
			vo.IsFollowed = followed
		}
	}

	return vo, nil
}

// GetUserByEmail 根据Email获取用户信息
func (s *userServiceImpl) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	logger.Debug("通过邮箱获取用户信息", zap.String("email", email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if pkgerrors.IsNotFound(err) {
			return nil, pkgerrors.ErrUserNotFound
		}
		logger.Error("获取用户信息失败", zap.Error(err), zap.String("email", email))
		return nil, pkgerrors.ConvertDBError(err)
	}

	return user, nil
}

// UpdateUser 更新用户信息
func (s *userServiceImpl) UpdateUser(ctx context.Context, userID uint, req *UpdateUserRequest) error {
	logger.Info("更新用户信息", zap.Uint("user_id", userID))

	fields := make(map[string]interface{})
	if req.Nickname != nil {
		fields["nickname"] = *req.Nickname
	}
	if req.Bio != nil {
		fields["bio"] = *req.Bio
	}
	if req.Avatar != nil {
		fields["avatar"] = *req.Avatar
	}
	if req.BackgroundImage != nil {
		fields["background_image"] = *req.BackgroundImage
	}
	if req.Gender != nil {
		fields["gender"] = *req.Gender
	}
	if req.Birthday != nil {
		fields["birthday"] = req.Birthday
	}
	if req.Province != nil {
		fields["province"] = *req.Province
	}
	if req.City != nil {
		fields["city"] = *req.City
	}
	if req.ShowFavorites != nil {
		fields["show_favorites"] = *req.ShowFavorites
	}
	if req.ShowLikes != nil {
		fields["show_likes"] = *req.ShowLikes
	}
	if req.ShowFollowing != nil {
		fields["show_following"] = *req.ShowFollowing
	}
	if req.ShowFollowers != nil {
		fields["show_followers"] = *req.ShowFollowers
	}

	// 如果更新了简介，处理 @ 提及
	if req.Bio != nil && *req.Bio != "" {
		mentionedUserIDs := extractBioMentions(*req.Bio)
		if len(mentionedUserIDs) > 0 && s.messageService != nil {
			go func() {
				bgCtx := context.Background()
				for _, targetUserID := range mentionedUserIDs {
					if targetUserID != userID {
						s.messageService.CreateNotification(bgCtx, &CreateNotificationRequest{
							UserID:   targetUserID,
							Type:     NotifyTypeMention,
							SenderID: &userID,
							Title:    "有人在简介中提到了你",
							Content:  *req.Bio,
						})
					}
				}
			}()
		}
	}

	if err := s.userRepo.UpdateFields(ctx, userID, fields); err != nil {
		logger.Error("更新用户信息失败", zap.Error(err), zap.Uint("user_id", userID))
		return err
	}

	logger.Info("用户信息更新成功", zap.Uint("user_id", userID))
	return nil
}

// FollowUser 关注用户
func (s *userServiceImpl) FollowUser(ctx context.Context, userID, targetID uint) error {
	logger.Info("关注用户", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))

	if userID == targetID {
		logger.Warn("不能关注自己", zap.Uint("user_id", userID))
		return pkgerrors.NewInvalidParamError("不能关注自己")
	}

	// 检查是否已关注
	exists, err := s.followRepo.Exists(ctx, userID, targetID)
	if err != nil {
		logger.Error("检查关注关系失败", zap.Error(err))
		return pkgerrors.ConvertDBError(err)
	}
	if exists {
		logger.Warn("已经关注过该用户", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))
		return pkgerrors.NewAppError(pkgerrors.CodeDuplicateKey, "已经关注过该用户")
	}

	// 创建关注关系
	follow := &model.Follow{
		UserID:     userID,
		FollowedID: targetID,
	}

	if err := s.followRepo.Create(ctx, follow); err != nil {
		logger.Error("创建关注关系失败", zap.Error(err))
		return err
	}

	// 更新关注数和粉丝数
	if err := s.userRepo.IncrementFollowCount(ctx, userID, 1); err != nil {
		logger.Error("更新关注数失败", zap.Error(err))
	}
	if err := s.userRepo.IncrementFollowerCount(ctx, targetID, 1); err != nil {
		logger.Error("更新粉丝数失败", zap.Error(err))
	}

	// 发送通知（异步）
	if s.messageService != nil {
		go func() {
			s.messageService.CreateNotification(context.Background(), &CreateNotificationRequest{
				UserID:    targetID,
				Type:      NotifyTypeFollow,
				SenderID:  &userID,
				RelatedID: &userID,
				Title:     "新的关注",
				Content:   "有人关注了你",
			})
		}()
	}

	logger.Info("关注用户成功", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))
	return nil
}

// UnfollowUser 取消关注
func (s *userServiceImpl) UnfollowUser(ctx context.Context, userID, targetID uint) error {
	logger.Info("取消关注", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))

	if err := s.followRepo.Delete(ctx, userID, targetID); err != nil {
		if pkgerrors.IsNotFound(err) {
			logger.Warn("未关注该用户", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))
			return pkgerrors.NewAppError(pkgerrors.CodeRecordNotFound, "未关注该用户")
		}
		logger.Error("删除关注关系失败", zap.Error(err))
		return pkgerrors.ConvertDBError(err)
	}

	// 更新关注数和粉丝数
	if err := s.userRepo.IncrementFollowCount(ctx, userID, -1); err != nil {
		logger.Error("更新关注数失败", zap.Error(err))
	}
	if err := s.userRepo.IncrementFollowerCount(ctx, targetID, -1); err != nil {
		logger.Error("更新粉丝数失败", zap.Error(err))
	}

	logger.Info("取消关注成功", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))
	return nil
}

// GetUserFollowings 获取用户关注列表（带隐私检查）
func (s *userServiceImpl) GetUserFollowings(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.UserFollowVO, int64, error) {
	logger.Debug("获取用户关注列表", zap.Uint("target_user_id", targetUserID), zap.Uint("current_user_id", currentUserID))

	// 查询目标用户
	targetUser, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, 0, pkgerrors.ErrUserNotFound
	}

	// 非本人查看时检查隐私设置
	if targetUserID != currentUserID && !targetUser.ShowFollowing {
		return nil, 0, pkgerrors.NewAppError(pkgerrors.CodeForbidden, "该用户的关注列表未公开")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	vos, err := s.followRepo.FindFollowingsWithInfo(ctx, targetUserID, currentUserID, pageSize, offset)
	if err != nil {
		logger.Error("获取关注列表失败", zap.Error(err))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}
	if vos == nil {
		vos = []*model.UserFollowVO{}
	}

	total, err := s.followRepo.CountFollowings(ctx, targetUserID)
	if err != nil {
		logger.Error("统计关注数失败", zap.Error(err))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	return vos, total, nil
}

// GetUserFollowers 获取用户粉丝列表（带隐私检查）
func (s *userServiceImpl) GetUserFollowers(ctx context.Context, targetUserID, currentUserID uint, page, pageSize int) ([]*model.UserFollowVO, int64, error) {
	logger.Debug("获取用户粉丝列表", zap.Uint("target_user_id", targetUserID), zap.Uint("current_user_id", currentUserID))

	// 查询目标用户
	targetUser, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, 0, pkgerrors.ErrUserNotFound
	}

	// 非本人查看时检查隐私设置
	if targetUserID != currentUserID && !targetUser.ShowFollowers {
		return nil, 0, pkgerrors.NewAppError(pkgerrors.CodeForbidden, "该用户的粉丝列表未公开")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	vos, err := s.followRepo.FindFollowersWithInfo(ctx, targetUserID, currentUserID, pageSize, offset)
	if err != nil {
		logger.Error("获取粉丝列表失败", zap.Error(err))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}
	if vos == nil {
		vos = []*model.UserFollowVO{}
	}

	total, err := s.followRepo.CountFollowers(ctx, targetUserID)
	if err != nil {
		logger.Error("统计粉丝数失败", zap.Error(err))
		return nil, 0, pkgerrors.ConvertDBError(err)
	}

	return vos, total, nil
}

// extractBioMentions 从简介中提取被 @ 的用户ID
// 格式：@[userId]
func extractBioMentions(bio string) []uint {
	var userIDs []uint
	// 匹配 @[数字] 格式
	regex := regexp.MustCompile(`@\[(\d+)\]`)
	matches := regex.FindAllStringSubmatch(bio, -1)

	for _, match := range matches {
		if len(match) > 1 {
			if id, err := strconv.ParseUint(match[1], 10, 64); err == nil {
				userIDs = append(userIDs, uint(id))
			}
		}
	}

	return userIDs
}

// parseBioMentions 将简介中的 @[userId] 解析为 @[userId:nickname]
// 用于返回给前端时实时显示最新的用户昵称
func (s *userServiceImpl) parseBioMentions(ctx context.Context, bio string) string {
	if bio == "" {
		return bio
	}

	// 提取所有被 @ 的用户ID
	userIDs := extractBioMentions(bio)
	if len(userIDs) == 0 {
		return bio
	}

	// 批量查询用户信息
	users, err := s.userRepo.FindByIDs(ctx, userIDs)
	if err != nil {
		logger.Error("查询用户信息失败", zap.Error(err))
		return bio
	}

	// 构建 userID -> nickname 映射
	userMap := make(map[uint]string)
	for _, user := range users {
		userMap[user.ID] = user.Nickname
	}

	// 替换 @[userId] 为 @[userId:nickname]
	regex := regexp.MustCompile(`@\[(\d+)\]`)
	result := regex.ReplaceAllStringFunc(bio, func(match string) string {
		// 提取 userId
		submatches := regex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			if id, err := strconv.ParseUint(submatches[1], 10, 64); err == nil {
				userID := uint(id)
				if nickname, ok := userMap[userID]; ok {
					return fmt.Sprintf("@[%d:%s]", userID, nickname)
				}
			}
		}
		return match
	})

	return result
}
