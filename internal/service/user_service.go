package service

import (
	"context"

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
	// GetUserByEmail 根据Email获取用户信息
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error
	// FollowUser 关注用户
	FollowUser(ctx context.Context, userID, targetID uint) error
	// UnfollowUser 取消关注
	UnfollowUser(ctx context.Context, userID, targetID uint) error
}

// userServiceImpl 用户服务层实现
type userServiceImpl struct {
	userRepo    repository.UserRepository
	followRepo  repository.FollowRepository
	profileRepo repository.ProfileRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, followRepo repository.FollowRepository, profileRepo repository.ProfileRepository) UserService {
	return &userServiceImpl{
		userRepo:    userRepo,
		followRepo:  followRepo,
		profileRepo: profileRepo,
	}
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
		Phone:    req.Phone,
		Nickname: req.Nickname,
		Status:   1, // 正常状态
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
		// 注意: 这里不返回错误，因为用户已经创建成功，只是资料创建失败
		// 可以在后续通过更新接口补充资料
	}

	logger.Info("用户注册成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	return user, nil
}

// Login 用户登录
func (s *userServiceImpl) Login(ctx context.Context, req *LoginRequest) (*model.User, error) {
	logger.Info("用户登录请求", zap.String("username", req.Username))

	// 查找用户（支持用户名或邮箱登录）
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		// 尝试使用邮箱查找
		user, err = s.userRepo.FindByEmail(ctx, req.Username)
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

	return user, nil
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
func (s *userServiceImpl) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error {
	logger.Info("更新用户信息", zap.Uint("user_id", userID), zap.Any("updates", updates))

	// 不允许更新的字段
	delete(updates, "id")
	delete(updates, "username")
	delete(updates, "password")
	delete(updates, "created_at")

	if err := s.userRepo.UpdateFields(ctx, userID, updates); err != nil {
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

	logger.Info("关注成功", zap.Uint("user_id", userID), zap.Uint("target_id", targetID))
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
