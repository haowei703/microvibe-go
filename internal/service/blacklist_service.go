package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
)

type BlacklistService interface {
	BlockUser(ctx context.Context, userID, blockedUserID uint) error
	UnblockUser(ctx context.Context, userID, blockedUserID uint) error
	IsBlocked(ctx context.Context, userID, blockedUserID uint) (bool, error)
	GetBlacklist(ctx context.Context, userID uint, page, pageSize int) ([]*model.Blacklist, int64, error)
}

type blacklistServiceImpl struct {
	blacklistRepo repository.BlacklistRepository
	userRepo      repository.UserRepository
}

func NewBlacklistService(blacklistRepo repository.BlacklistRepository, userRepo repository.UserRepository) BlacklistService {
	return &blacklistServiceImpl{
		blacklistRepo: blacklistRepo,
		userRepo:      userRepo,
	}
}

func (s *blacklistServiceImpl) BlockUser(ctx context.Context, userID, blockedUserID uint) error {
	if userID == blockedUserID {
		return errors.New("不能拉黑自己")
	}

	// 检查被拉黑用户是否存在
	_, err := s.userRepo.FindByID(ctx, blockedUserID)
	if err != nil {
		return errors.New("用户不存在")
	}

	blacklist := &model.Blacklist{
		UserID:        userID,
		BlockedUserID: blockedUserID,
	}

	return s.blacklistRepo.Create(ctx, blacklist)
}

func (s *blacklistServiceImpl) UnblockUser(ctx context.Context, userID, blockedUserID uint) error {
	return s.blacklistRepo.Delete(ctx, userID, blockedUserID)
}

func (s *blacklistServiceImpl) IsBlocked(ctx context.Context, userID, blockedUserID uint) (bool, error) {
	return s.blacklistRepo.IsBlocked(ctx, userID, blockedUserID)
}

func (s *blacklistServiceImpl) GetBlacklist(ctx context.Context, userID uint, page, pageSize int) ([]*model.Blacklist, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.blacklistRepo.FindByUserID(ctx, userID, pageSize, offset)
}
