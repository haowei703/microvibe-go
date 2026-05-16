package service

import (
	"context"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// UserVisitorService 用户访客服务层接口
type UserVisitorService interface {
	// RecordVisit 记录访问
	RecordVisit(ctx context.Context, ownerID, visitorID uint) error
	// GetVisitors 获取谁访问了我
	GetVisitors(ctx context.Context, userID uint, page, pageSize int) ([]*VisitorVO, int64, error)
	// GetVisited 获取我访问了谁
	GetVisited(ctx context.Context, userID uint, page, pageSize int) ([]*VisitorVO, int64, error)
}

// VisitorVO 访客视图对象
type VisitorVO struct {
	ID              uint   `json:"id"`
	Nickname        string `json:"nickname"`
	Avatar          string `json:"avatar"`
	BackgroundImage string `json:"background_image"`
	VisitTime       string `json:"visit_time"`
	IsFollowed      bool   `json:"is_followed"`
}

type userVisitorServiceImpl struct {
	visitorRepo repository.UserVisitorRepository
	followRepo  repository.FollowRepository
}

// NewUserVisitorService 创建用户访客服务实例
func NewUserVisitorService(
	visitorRepo repository.UserVisitorRepository,
	followRepo repository.FollowRepository,
) UserVisitorService {
	return &userVisitorServiceImpl{
		visitorRepo: visitorRepo,
		followRepo:  followRepo,
	}
}

// RecordVisit 记录访问
func (s *userVisitorServiceImpl) RecordVisit(ctx context.Context, ownerID, visitorID uint) error {
	if ownerID == visitorID {
		return nil // 自己访问自己不记录
	}

	logger.Debug("记录个人主页访客", zap.Uint("owner_id", ownerID), zap.Uint("visitor_id", visitorID))
	return s.visitorRepo.Upsert(ctx, ownerID, visitorID)
}

// GetVisitors 获取谁访问了我
func (s *userVisitorServiceImpl) GetVisitors(ctx context.Context, userID uint, page, pageSize int) ([]*VisitorVO, int64, error) {
	records, total, err := s.visitorRepo.FindByOwnerID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	vos := make([]*VisitorVO, 0)
	for _, r := range records {
		if r.Visitor == nil {
			continue
		}

		isFollowed, _ := s.followRepo.Exists(ctx, userID, r.VisitorID)

		vos = append(vos, &VisitorVO{
			ID:              r.VisitorID,
			Nickname:        r.Visitor.Nickname,
			Avatar:          s.fullURL(r.Visitor.Avatar),
			BackgroundImage: s.fullURL(r.Visitor.BackgroundImage),
			VisitTime:       r.UpdatedAt.Format("2006-01-02 15:04:05"),
			IsFollowed:      isFollowed,
		})
	}

	return vos, total, nil
}

// GetVisited 获取我访问了谁
func (s *userVisitorServiceImpl) GetVisited(ctx context.Context, userID uint, page, pageSize int) ([]*VisitorVO, int64, error) {
	records, total, err := s.visitorRepo.FindByVisitorID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	vos := make([]*VisitorVO, 0)
	for _, r := range records {
		if r.Owner == nil {
			continue
		}

		isFollowed, _ := s.followRepo.Exists(ctx, userID, r.OwnerID)

		vos = append(vos, &VisitorVO{
			ID:              r.OwnerID,
			Nickname:        r.Owner.Nickname,
			Avatar:          s.fullURL(r.Owner.Avatar),
			BackgroundImage: s.fullURL(r.Owner.BackgroundImage),
			VisitTime:       r.UpdatedAt.Format("2006-01-02 15:04:05"),
			IsFollowed:      isFollowed,
		})
	}

	return vos, total, nil
}

// fullURL 将相对路径转换为完整 URL
func (s *userVisitorServiceImpl) fullURL(path string) string {
	if path == "" {
		return path
	}
	// 这里目前只是简单返回，如果以后有 CDN 域名可以在这里拼接
	// 暂时保持与 video_service 逻辑一致，如果不包含 http 则返回原样（或者拼接 base url）
	// 注意：video_service 中的 fullURL 逻辑逻辑是检查 strings.HasPrefix(path, "http")
	return path
}
