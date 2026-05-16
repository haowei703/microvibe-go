package service

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
)

type ReportService interface {
	CreateReport(ctx context.Context, report *model.Report) error
	GetReportsByReporter(ctx context.Context, reporterID uint, page, pageSize int) ([]*model.Report, int64, error)
}

type reportServiceImpl struct {
	reportRepo  repository.ReportRepository
	userRepo    repository.UserRepository
	videoRepo   repository.VideoRepository
	commentRepo repository.CommentRepository
}

func NewReportService(
	reportRepo repository.ReportRepository,
	userRepo repository.UserRepository,
	videoRepo repository.VideoRepository,
	commentRepo repository.CommentRepository,
) ReportService {
	return &reportServiceImpl{
		reportRepo:  reportRepo,
		userRepo:    userRepo,
		videoRepo:   videoRepo,
		commentRepo: commentRepo,
	}
}

func (s *reportServiceImpl) CreateReport(ctx context.Context, report *model.Report) error {
	// 验证目标是否存在
	switch report.TargetType {
	case 1: // 用户
		if _, err := s.userRepo.FindByID(ctx, report.TargetID); err != nil {
			return errors.New("举报对象(用户)不存在")
		}
	case 2: // 视频
		if _, err := s.videoRepo.FindByID(ctx, report.TargetID); err != nil {
			return errors.New("举报对象(视频)不存在")
		}
	case 3: // 评论
		if _, err := s.commentRepo.FindByID(ctx, report.TargetID); err != nil {
			return errors.New("举报对象(评论)不存在")
		}
	default:
		return errors.New("无效的举报类型")
	}

	report.Status = 0 // 待处理
	return s.reportRepo.Create(ctx, report)
}

func (s *reportServiceImpl) GetReportsByReporter(ctx context.Context, reporterID uint, page, pageSize int) ([]*model.Report, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.reportRepo.FindByReporterID(ctx, reporterID, pageSize, offset)
}
