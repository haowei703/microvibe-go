package repository

import (
	"context"
	"gorm.io/gorm"
	"microvibe-go/internal/model"
)

// ReportRepository 举报数据访问层接口
type ReportRepository interface {
	Create(ctx context.Context, report *model.Report) error
	FindByID(ctx context.Context, id uint) (*model.Report, error)
	FindByReporterID(ctx context.Context, reporterID uint, limit, offset int) ([]*model.Report, int64, error)
	FindAll(ctx context.Context, status int8, limit, offset int) ([]*model.Report, int64, error)
	UpdateStatus(ctx context.Context, id uint, status int8) error
}

type reportRepositoryImpl struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepositoryImpl{db: db}
}

func (r *reportRepositoryImpl) Create(ctx context.Context, report *model.Report) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *reportRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.Report, error) {
	var report model.Report
	err := r.db.WithContext(ctx).First(&report, id).Error
	return &report, err
}

func (r *reportRepositoryImpl) FindByReporterID(ctx context.Context, reporterID uint, limit, offset int) ([]*model.Report, int64, error) {
	var reports []*model.Report
	var total int64
	db := r.db.WithContext(ctx).Model(&model.Report{}).Where("reporter_id = ?", reporterID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&reports).Error
	return reports, total, err
}

func (r *reportRepositoryImpl) FindAll(ctx context.Context, status int8, limit, offset int) ([]*model.Report, int64, error) {
	var reports []*model.Report
	var total int64
	db := r.db.WithContext(ctx).Model(&model.Report{})
	if status >= 0 {
		db = db.Where("status = ?", status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&reports).Error
	return reports, total, err
}

func (r *reportRepositoryImpl) UpdateStatus(ctx context.Context, id uint, status int8) error {
	return r.db.WithContext(ctx).Model(&model.Report{}).Where("id = ?", id).Update("status", status).Error
}
