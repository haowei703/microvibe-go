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

// SendGiftRequest 送礼请求
type SendGiftRequest struct {
	LiveID      uint   `json:"live_id" binding:"required"`                // 直播间ID
	GiftID      uint   `json:"gift_id" binding:"required"`                // 礼物ID
	Quantity    int    `json:"quantity" binding:"required,min=1,max=999"` // 数量
	IsAnonymous bool   `json:"is_anonymous"`                              // 是否匿名
	Message     string `json:"message"`                                   // 附带消息
}

// LiveGiftService 直播礼物服务接口
type LiveGiftService interface {
	// ListGifts 获取礼物列表
	ListGifts(ctx context.Context, giftType int8, status int8) ([]*model.LiveGift, error)

	// GetGiftByID 根据ID获取礼物
	GetGiftByID(ctx context.Context, id uint) (*model.LiveGift, error)

	// SendGift 送礼
	SendGift(ctx context.Context, userID uint, req *SendGiftRequest) (*model.LiveGiftRecord, error)

	// ListGiftRecords 获取送礼记录
	ListGiftRecords(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveGiftRecord, int64, error)

	// GetTopGivers 获取送礼榜单
	GetTopGivers(ctx context.Context, liveID uint, limit int) ([]*model.LiveGiftRecord, error)

	// GetUserGiftStats 获取用户送礼统计
	GetUserGiftStats(ctx context.Context, liveID, userID uint) (int64, int, error)
}

type liveGiftServiceImpl struct {
	giftRepo       repository.LiveGiftRepository
	liveStreamRepo repository.LiveStreamRepository
	userRepo       repository.UserRepository
}

// NewLiveGiftService 创建礼物服务
func NewLiveGiftService(
	giftRepo repository.LiveGiftRepository,
	liveStreamRepo repository.LiveStreamRepository,
	userRepo repository.UserRepository,
) LiveGiftService {
	return &liveGiftServiceImpl{
		giftRepo:       giftRepo,
		liveStreamRepo: liveStreamRepo,
		userRepo:       userRepo,
	}
}

// ListGifts 获取礼物列表
func (s *liveGiftServiceImpl) ListGifts(ctx context.Context, giftType int8, status int8) ([]*model.LiveGift, error) {
	gifts, err := s.giftRepo.List(ctx, giftType, status)
	if err != nil {
		logger.Error("查询礼物列表失败", zap.Error(err))
		return nil, errors.New("查询礼物列表失败")
	}

	logger.Info("查询礼物列表成功", zap.Int("count", len(gifts)))
	return gifts, nil
}

// GetGiftByID 根据ID获取礼物
func (s *liveGiftServiceImpl) GetGiftByID(ctx context.Context, id uint) (*model.LiveGift, error) {
	gift, err := s.giftRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("礼物不存在")
		}
		logger.Error("查询礼物失败", zap.Error(err), zap.Uint("gift_id", id))
		return nil, errors.New("查询礼物失败")
	}

	return gift, nil
}

// SendGift 送礼
func (s *liveGiftServiceImpl) SendGift(ctx context.Context, userID uint, req *SendGiftRequest) (*model.LiveGiftRecord, error) {
	// 1. 查询直播间
	liveStream, err := s.liveStreamRepo.FindByID(ctx, req.LiveID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("live_id", req.LiveID))
		return nil, errors.New("查询直播间失败")
	}

	// 2. 检查直播状态
	if liveStream.Status != "live" {
		return nil, errors.New("直播未开始")
	}

	// 3. 检查是否允许送礼
	if !liveStream.AllowGift {
		return nil, errors.New("该直播间暂不支持送礼")
	}

	// 4. 查询礼物信息
	gift, err := s.giftRepo.FindByID(ctx, req.GiftID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("礼物不存在")
		}
		logger.Error("查询礼物失败", zap.Error(err), zap.Uint("gift_id", req.GiftID))
		return nil, errors.New("查询礼物失败")
	}

	// 5. 检查礼物状态
	if gift.Status != 1 {
		return nil, errors.New("该礼物已下架")
	}

	// 6. 计算总价值
	totalValue := gift.Price * int64(req.Quantity)

	// TODO: 这里应该检查用户余额并扣费，暂时省略

	// 7. 创建送礼记录
	record := &model.LiveGiftRecord{
		LiveID:      req.LiveID,
		UserID:      userID,
		TargetID:    liveStream.OwnerID, // 送给主播
		GiftID:      req.GiftID,
		GiftName:    gift.Name,
		Quantity:    req.Quantity,
		UnitPrice:   gift.Price,
		TotalValue:  totalValue,
		ComboCount:  req.Quantity,
		IsAnonymous: req.IsAnonymous,
		Message:     req.Message,
	}

	if err := s.giftRepo.CreateGiftRecord(ctx, record); err != nil {
		logger.Error("创建送礼记录失败", zap.Error(err))
		return nil, errors.New("送礼失败")
	}

	// 8. 更新直播间礼物统计
	if err := s.liveStreamRepo.UpdateGiftStats(ctx, req.LiveID, int64(req.Quantity), totalValue); err != nil {
		logger.Error("更新直播间礼物统计失败", zap.Error(err))
		// 不影响主流程，继续执行
	}

	logger.Info("送礼成功",
		zap.Uint("user_id", userID),
		zap.Uint("live_id", req.LiveID),
		zap.String("gift_name", gift.Name),
		zap.Int("quantity", req.Quantity),
		zap.Int64("total_value", totalValue))

	// 重新查询以获取关联信息
	record.Gift = gift
	return record, nil
}

// ListGiftRecords 获取送礼记录
func (s *liveGiftServiceImpl) ListGiftRecords(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveGiftRecord, int64, error) {
	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := s.giftRepo.ListGiftRecords(ctx, liveID, page, pageSize)
	if err != nil {
		logger.Error("查询送礼记录失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, 0, errors.New("查询送礼记录失败")
	}

	logger.Info("查询送礼记录成功",
		zap.Uint("live_id", liveID),
		zap.Int("page", page),
		zap.Int("size", pageSize),
		zap.Int64("total", total))

	return records, total, nil
}

// GetTopGivers 获取送礼榜单
func (s *liveGiftServiceImpl) GetTopGivers(ctx context.Context, liveID uint, limit int) ([]*model.LiveGiftRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	records, err := s.giftRepo.GetTopGivers(ctx, liveID, limit)
	if err != nil {
		logger.Error("查询送礼榜单失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, errors.New("查询送礼榜单失败")
	}

	logger.Info("查询送礼榜单成功", zap.Uint("live_id", liveID), zap.Int("count", len(records)))
	return records, nil
}

// GetUserGiftStats 获取用户送礼统计
func (s *liveGiftServiceImpl) GetUserGiftStats(ctx context.Context, liveID, userID uint) (int64, int, error) {
	totalValue, giftCount, err := s.giftRepo.GetUserGiftStats(ctx, liveID, userID)
	if err != nil {
		logger.Error("查询用户送礼统计失败", zap.Error(err), zap.Uint("live_id", liveID), zap.Uint("user_id", userID))
		return 0, 0, errors.New("查询用户送礼统计失败")
	}

	return totalValue, giftCount, nil
}
