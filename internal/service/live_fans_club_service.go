package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LiveFansClubService 粉丝团服务接口
type LiveFansClubService interface {
	// JoinFansClub 加入粉丝团
	JoinFansClub(ctx context.Context, userID, liveID uint) (*model.LiveFansClub, error)

	// QuitFansClub 退出粉丝团
	QuitFansClub(ctx context.Context, userID, liveID uint) error

	// GetMemberInfo 获取粉丝团成员信息
	GetMemberInfo(ctx context.Context, userID, liveID uint) (*model.LiveFansClub, error)

	// ListMembers 获取粉丝团成员列表
	ListMembers(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveFansClub, int64, error)

	// GetTopMembers 获取粉丝团排行榜
	GetTopMembers(ctx context.Context, liveID uint, limit int) ([]*model.LiveFansClub, error)

	// AddExperience 增加经验值
	AddExperience(ctx context.Context, userID, liveID uint, exp int64) error

	// UpgradeLevel 升级（检查经验值并升级）
	UpgradeLevel(ctx context.Context, userID, liveID uint) error

	// GetMemberCount 获取粉丝团人数
	GetMemberCount(ctx context.Context, liveID uint) (int64, error)
}

type liveFansClubServiceImpl struct {
	fansClubRepo   repository.LiveFansClubRepository
	liveStreamRepo repository.LiveStreamRepository
}

// NewLiveFansClubService 创建粉丝团服务
func NewLiveFansClubService(
	fansClubRepo repository.LiveFansClubRepository,
	liveStreamRepo repository.LiveStreamRepository,
) LiveFansClubService {
	return &liveFansClubServiceImpl{
		fansClubRepo:   fansClubRepo,
		liveStreamRepo: liveStreamRepo,
	}
}

// JoinFansClub 加入粉丝团
func (s *liveFansClubServiceImpl) JoinFansClub(ctx context.Context, userID, liveID uint) (*model.LiveFansClub, error) {
	// 1. 查询直播间
	liveStream, err := s.liveStreamRepo.FindByID(ctx, liveID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, errors.New("查询直播间失败")
	}

	// 2. 检查是否已加入
	existingMember, err := s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
	if err == nil && existingMember != nil {
		if existingMember.IsActivated {
			return nil, errors.New("已经是粉丝团成员")
		}
		// 如果之前退出过，重新激活
		existingMember.IsActivated = true
		if err := s.fansClubRepo.Update(ctx, existingMember); err != nil {
			logger.Error("重新激活粉丝团失败", zap.Error(err))
			return nil, errors.New("加入粉丝团失败")
		}
		logger.Info("重新加入粉丝团", zap.Uint("user_id", userID), zap.Uint("live_id", liveID))
		return existingMember, nil
	}

	// 3. 创建粉丝团成员
	member := &model.LiveFansClub{
		LiveID:      liveID,
		UserID:      userID,
		Level:       1,
		Experience:  0,
		BadgeName:   "初级粉丝",
		IsActivated: true,
	}

	if err := s.fansClubRepo.Create(ctx, member); err != nil {
		logger.Error("加入粉丝团失败", zap.Error(err))
		return nil, errors.New("加入粉丝团失败")
	}

	logger.Info("加入粉丝团成功",
		zap.Uint("user_id", userID),
		zap.Uint("live_id", liveID),
		zap.Uint("owner_id", liveStream.OwnerID))

	return member, nil
}

// QuitFansClub 退出粉丝团
func (s *liveFansClubServiceImpl) QuitFansClub(ctx context.Context, userID, liveID uint) error {
	// 1. 查询粉丝团成员
	member, err := s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("您还不是粉丝团成员")
		}
		logger.Error("查询粉丝团成员失败", zap.Error(err))
		return errors.New("查询粉丝团成员失败")
	}

	if !member.IsActivated {
		return errors.New("您已退出粉丝团")
	}

	// 2. 取消激活
	member.IsActivated = false
	if err := s.fansClubRepo.Update(ctx, member); err != nil {
		logger.Error("退出粉丝团失败", zap.Error(err))
		return errors.New("退出粉丝团失败")
	}

	logger.Info("退出粉丝团成功", zap.Uint("user_id", userID), zap.Uint("live_id", liveID))
	return nil
}

// GetMemberInfo 获取粉丝团成员信息
func (s *liveFansClubServiceImpl) GetMemberInfo(ctx context.Context, userID, liveID uint) (*model.LiveFansClub, error) {
	member, err := s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("您还不是粉丝团成员")
		}
		logger.Error("查询粉丝团成员信息失败", zap.Error(err))
		return nil, errors.New("查询粉丝团成员信息失败")
	}

	return member, nil
}

// ListMembers 获取粉丝团成员列表
func (s *liveFansClubServiceImpl) ListMembers(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveFansClub, int64, error) {
	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	members, total, err := s.fansClubRepo.ListByLiveID(ctx, liveID, page, pageSize)
	if err != nil {
		logger.Error("查询粉丝团成员列表失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, 0, errors.New("查询粉丝团成员列表失败")
	}

	logger.Info("查询粉丝团成员列表成功",
		zap.Uint("live_id", liveID),
		zap.Int("page", page),
		zap.Int("size", pageSize),
		zap.Int64("total", total))

	return members, total, nil
}

// GetTopMembers 获取粉丝团排行榜
func (s *liveFansClubServiceImpl) GetTopMembers(ctx context.Context, liveID uint, limit int) ([]*model.LiveFansClub, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	members, err := s.fansClubRepo.GetTopMembers(ctx, liveID, limit)
	if err != nil {
		logger.Error("查询粉丝团排行榜失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, errors.New("查询粉丝团排行榜失败")
	}

	logger.Info("查询粉丝团排行榜成功", zap.Uint("live_id", liveID), zap.Int("count", len(members)))
	return members, nil
}

// AddExperience 增加经验值
func (s *liveFansClubServiceImpl) AddExperience(ctx context.Context, userID, liveID uint, exp int64) error {
	// 1. 查询粉丝团成员
	member, err := s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果用户还不是粉丝团成员，自动加入
			_, err := s.JoinFansClub(ctx, userID, liveID)
			if err != nil {
				return err
			}
			member, err = s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
			if err != nil {
				return errors.New("查询粉丝团成员失败")
			}
		} else {
			logger.Error("查询粉丝团成员失败", zap.Error(err))
			return errors.New("查询粉丝团成员失败")
		}
	}

	if !member.IsActivated {
		return errors.New("粉丝团成员未激活")
	}

	// 2. 增加经验值
	if err := s.fansClubRepo.AddExperience(ctx, member.ID, exp); err != nil {
		logger.Error("增加经验值失败", zap.Error(err))
		return errors.New("增加经验值失败")
	}

	logger.Info("增加经验值成功",
		zap.Uint("user_id", userID),
		zap.Uint("live_id", liveID),
		zap.Int64("exp", exp))

	// 3. 检查是否可以升级
	_ = s.UpgradeLevel(ctx, userID, liveID)

	return nil
}

// UpgradeLevel 升级（检查经验值并升级）
func (s *liveFansClubServiceImpl) UpgradeLevel(ctx context.Context, userID, liveID uint) error {
	// 1. 查询粉丝团成员
	member, err := s.fansClubRepo.FindByLiveAndUser(ctx, liveID, userID)
	if err != nil {
		return errors.New("查询粉丝团成员失败")
	}

	// 2. 计算等级（简单的升级规则：每1000经验值升1级，最高10级）
	newLevel := int(member.Experience/1000) + 1
	if newLevel > 10 {
		newLevel = 10
	}

	// 3. 检查是否需要升级
	if newLevel <= member.Level {
		return nil // 不需要升级
	}

	// 4. 更新等级
	if err := s.fansClubRepo.UpdateLevel(ctx, member.ID, newLevel); err != nil {
		logger.Error("升级失败", zap.Error(err))
		return errors.New("升级失败")
	}

	// 5. 更新徽章名称
	badgeName := getBadgeName(newLevel)
	member.BadgeName = badgeName
	member.Level = newLevel
	if err := s.fansClubRepo.Update(ctx, member); err != nil {
		logger.Warn("更新徽章名称失败", zap.Error(err))
	}

	logger.Info("粉丝团升级成功",
		zap.Uint("user_id", userID),
		zap.Uint("live_id", liveID),
		zap.Int("old_level", member.Level),
		zap.Int("new_level", newLevel))

	return nil
}

// GetMemberCount 获取粉丝团人数
func (s *liveFansClubServiceImpl) GetMemberCount(ctx context.Context, liveID uint) (int64, error) {
	count, err := s.fansClubRepo.CountMembers(ctx, liveID)
	if err != nil {
		logger.Error("统计粉丝团人数失败", zap.Error(err), zap.Uint("live_id", liveID))
		return 0, errors.New("统计粉丝团人数失败")
	}

	return count, nil
}

// getBadgeName 根据等级获取徽章名称
func getBadgeName(level int) string {
	badgeNames := map[int]string{
		1:  "初级粉丝",
		2:  "中级粉丝",
		3:  "高级粉丝",
		4:  "资深粉丝",
		5:  "核心粉丝",
		6:  "铁杆粉丝",
		7:  "超级粉丝",
		8:  "至尊粉丝",
		9:  "荣耀粉丝",
		10: "传奇粉丝",
	}

	if name, ok := badgeNames[level]; ok {
		return name
	}
	return "初级粉丝"
}
