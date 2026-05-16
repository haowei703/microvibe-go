package repository

import (
	"context"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserVisitorRepository 用户访客记录数据访问层接口
type UserVisitorRepository interface {
	// Upsert 记录或更新访客记录
	Upsert(ctx context.Context, ownerID, visitorID uint) error
	// FindByOwnerID 查询谁访问了我
	FindByOwnerID(ctx context.Context, ownerID uint, page, pageSize int) ([]*model.UserVisitor, int64, error)
	// FindByVisitorID 查询我访问了谁
	FindByVisitorID(ctx context.Context, visitorID uint, page, pageSize int) ([]*model.UserVisitor, int64, error)
	// PruneOldRecords 清理旧记录，保留最近的 limit 条 (针对 OwnerID 或 VisitorID)
	PruneOldRecords(ctx context.Context, userID uint, limit int, isOwner bool) error
}

type userVisitorRepositoryImpl struct {
	db *gorm.DB
}

// NewUserVisitorRepository 创建用户访客记录数据访问层实例
func NewUserVisitorRepository(db *gorm.DB) UserVisitorRepository {
	return &userVisitorRepositoryImpl{
		db: db,
	}
}

// Upsert 记录或更新访客记录
func (r *userVisitorRepositoryImpl) Upsert(ctx context.Context, ownerID, visitorID uint) error {
	logger.Debug("Upsert 访客记录", zap.Uint("owner_id", ownerID), zap.Uint("visitor_id", visitorID))

	visitor := &model.UserVisitor{
		OwnerID:   ownerID,
		VisitorID: visitorID,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_id"}, {Name: "visitor_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(visitor).Error

	if err != nil {
		logger.Error("Upsert 访客记录失败", zap.Error(err))
		return err
	}

	// 异步清理记录，保留最近 200 条 (被访问者的访客列表)
	go func() {
		_ = r.PruneOldRecords(context.Background(), ownerID, 200, true)
	}()
	// 同时异步清理主动访问记录，保留最近 200 条
	go func() {
		_ = r.PruneOldRecords(context.Background(), visitorID, 200, false)
	}()

	return nil
}

// FindByOwnerID 查询谁访问了我
func (r *userVisitorRepositoryImpl) FindByOwnerID(ctx context.Context, ownerID uint, page, pageSize int) ([]*model.UserVisitor, int64, error) {
	visitors := make([]*model.UserVisitor, 0)
	var total int64

	query := r.db.WithContext(ctx).Model(&model.UserVisitor{}).
		Where("owner_id = ?", ownerID)

	// 去重统计总数
	if err := query.Distinct("visitor_id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	// 使用子查询去重，确保每个 visitor_id 只返回最新的一条记录
	subQuery := r.db.Model(&model.UserVisitor{}).
		Select("MAX(id)").
		Where("owner_id = ?", ownerID).
		Group("visitor_id")

	err := r.db.WithContext(ctx).Model(&model.UserVisitor{}).
		Preload("Visitor").
		Where("id IN (?)", subQuery).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&visitors).Error

	return visitors, total, err
}

// FindByVisitorID 查询我访问了谁
func (r *userVisitorRepositoryImpl) FindByVisitorID(ctx context.Context, visitorID uint, page, pageSize int) ([]*model.UserVisitor, int64, error) {
	visited := make([]*model.UserVisitor, 0)
	var total int64

	query := r.db.WithContext(ctx).Model(&model.UserVisitor{}).
		Where("visitor_id = ?", visitorID)

	// 去重统计总数
	if err := query.Distinct("owner_id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	// 使用子查询去重，确保每个 owner_id 只返回最新的一条记录
	subQuery := r.db.Model(&model.UserVisitor{}).
		Select("MAX(id)").
		Where("visitor_id = ?", visitorID).
		Group("owner_id")

	err := r.db.WithContext(ctx).Model(&model.UserVisitor{}).
		Preload("Owner").
		Where("id IN (?)", subQuery).
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&visited).Error

	return visited, total, err
}

// PruneOldRecords 清理旧记录
func (r *userVisitorRepositoryImpl) PruneOldRecords(ctx context.Context, userID uint, limit int, isOwner bool) error {
	var count int64
	field := "visitor_id"
	if isOwner {
		field = "owner_id"
	}

	r.db.WithContext(ctx).Model(&model.UserVisitor{}).Where(field+" = ?", userID).Count(&count)

	if count <= int64(limit) {
		return nil
	}

	var lastRecord model.UserVisitor
	err := r.db.WithContext(ctx).
		Where(field+" = ?", userID).
		Order("updated_at DESC").
		Offset(limit - 1).
		Limit(1).
		First(&lastRecord).Error

	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where(field+" = ? AND updated_at < ?", userID, lastRecord.UpdatedAt).
		Delete(&model.UserVisitor{}).Error
}
