package service

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
)

type AdminService interface {
	// 内容审核
	AuditVideo(ctx context.Context, videoID uint, status int8) error
	AuditComment(ctx context.Context, commentID uint, status int8) error

	// 用户与权限管理
	UpdateUserStatus(ctx context.Context, userID uint, status int8) error

	// 推荐干预与置顶
	SetVideoTop(ctx context.Context, videoID uint, isTop bool) error
	UpdateVideoRecommendWeight(ctx context.Context, videoID uint, hotScore float64) error

	// 搜索与热搜维护
	DeleteHotSearch(ctx context.Context, keyword string) error
	UpdateHotSearchWeight(ctx context.Context, keyword string, count int64) error

	// 列表获取
	ListVideos(ctx context.Context, page, pageSize int, status *int8) ([]*model.Video, int64, error)
	ListUsers(ctx context.Context, page, pageSize int, query string) ([]*model.User, int64, error)
	ListHotSearches(ctx context.Context) ([]*model.HotSearch, error)
	ListReports(ctx context.Context, page, pageSize int, status int8) ([]*model.Report, int64, error)
}

type adminServiceImpl struct {
	userRepo    repository.UserRepository
	videoRepo   repository.VideoRepository
	commentRepo repository.CommentRepository
	searchRepo  repository.SearchRepository
	reportRepo  repository.ReportRepository
}

func NewAdminService(
	userRepo repository.UserRepository,
	videoRepo repository.VideoRepository,
	commentRepo repository.CommentRepository,
	searchRepo repository.SearchRepository,
	reportRepo repository.ReportRepository,
) AdminService {
	return &adminServiceImpl{
		userRepo:    userRepo,
		videoRepo:   videoRepo,
		commentRepo: commentRepo,
		searchRepo:  searchRepo,
		reportRepo:  reportRepo,
	}
}

func (s *adminServiceImpl) AuditVideo(ctx context.Context, videoID uint, status int8) error {
	return s.videoRepo.UpdateFields(ctx, videoID, map[string]interface{}{"status": status})
}

func (s *adminServiceImpl) AuditComment(ctx context.Context, commentID uint, status int8) error {
	// 假设 Comment 模型有 Status 字段，如果没有则需要添加
	// 目前通过删除来实现“强制下架”
	if status == 2 { // 2: 违规下架
		return s.commentRepo.Delete(ctx, commentID)
	}
	return nil
}

func (s *adminServiceImpl) UpdateUserStatus(ctx context.Context, userID uint, status int8) error {
	return s.userRepo.Update(ctx, &model.User{ID: userID, Status: status})
}

func (s *adminServiceImpl) SetVideoTop(ctx context.Context, videoID uint, isTop bool) error {
	return s.videoRepo.UpdateFields(ctx, videoID, map[string]interface{}{"is_top": isTop})
}

func (s *adminServiceImpl) UpdateVideoRecommendWeight(ctx context.Context, videoID uint, hotScore float64) error {
	return s.videoRepo.UpdateFields(ctx, videoID, map[string]interface{}{"hot_score": hotScore})
}

func (s *adminServiceImpl) DeleteHotSearch(ctx context.Context, keyword string) error {
	return s.searchRepo.DeleteHotSearch(ctx, keyword)
}

func (s *adminServiceImpl) UpdateHotSearchWeight(ctx context.Context, keyword string, count int64) error {
	return s.searchRepo.UpdateHotSearchCount(ctx, keyword, count)
}

func (s *adminServiceImpl) ListVideos(ctx context.Context, page, pageSize int, status *int8) ([]*model.Video, int64, error) {
	return s.videoRepo.List(ctx, page, pageSize, status)
}

func (s *adminServiceImpl) ListUsers(ctx context.Context, page, pageSize int, query string) ([]*model.User, int64, error) {
	return s.userRepo.List(ctx, page, pageSize, query)
}

func (s *adminServiceImpl) ListHotSearches(ctx context.Context) ([]*model.HotSearch, error) {
	return s.searchRepo.GetHotSearches(ctx, 50) // 默认返回前 50 条
}

func (s *adminServiceImpl) ListReports(ctx context.Context, page, pageSize int, status int8) ([]*model.Report, int64, error) {
	return s.reportRepo.FindAll(ctx, status, pageSize, (page-1)*pageSize)
}
