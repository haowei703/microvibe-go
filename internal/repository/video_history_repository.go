package repository

import (
	"context"
	"errors"
	"microvibe-go/internal/model"
	"microvibe-go/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoHistoryRepository 视频播放历史数据访问层接口
type VideoHistoryRepository interface {
	// Upsert 更新或创建播放历史
	Upsert(ctx context.Context, history *model.VideoHistory) error
	// FindByUserID 分页查询用户播放历史
	FindByUserID(ctx context.Context, userID uint, page, pageSize int, finished *bool) ([]*model.VideoHistory, int64, error)
	// Delete 删除指定播放历史
	Delete(ctx context.Context, userID, historyID uint) error
	// ClearAll 清空用户所有播放历史
	ClearAll(ctx context.Context, userID uint) error
	// PruneOldRecords 清理旧记录，保留最近的 limit 条
	PruneOldRecords(ctx context.Context, userID uint, limit int) error
}

type videoHistoryRepositoryImpl struct {
	db *gorm.DB
}

// NewVideoHistoryRepository 创建视频播放历史数据访问层实例
func NewVideoHistoryRepository(db *gorm.DB) VideoHistoryRepository {
	return &videoHistoryRepositoryImpl{
		db: db,
	}
}

// Upsert 更新或创建播放历史，支持恢复软删除记录
func (r *videoHistoryRepositoryImpl) Upsert(ctx context.Context, history *model.VideoHistory) error {
	logger.Debug("Upsert 播放历史", zap.Uint("user_id", history.UserID), zap.Uint("video_id", history.VideoID))

	// 先查找是否存在记录（含软删除）
	var existing model.VideoHistory
	err := r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND video_id = ?", history.UserID, history.VideoID).
		Order("deleted_at IS NULL DESC, updated_at DESC").
		First(&existing).Error

	if err == nil {
		// 记录存在（可能已软删除），恢复并更新
		now := time.Now()
		updateErr := r.db.WithContext(ctx).Unscoped().Model(&existing).Updates(map[string]interface{}{
			"position":   history.Position,
			"duration":   history.Duration,
			"finished":   history.Finished,
			"deleted_at": nil,
			"updated_at": now,
		}).Error
		if updateErr != nil {
			logger.Error("更新播放历史失败", zap.Error(updateErr), zap.Uint("user_id", history.UserID))
			return updateErr
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// 不存在，创建新记录
		if createErr := r.db.WithContext(ctx).Create(history).Error; createErr != nil {
			logger.Error("创建播放历史失败", zap.Error(createErr), zap.Uint("user_id", history.UserID))
			return createErr
		}
	} else {
		logger.Error("查询播放历史失败", zap.Error(err), zap.Uint("user_id", history.UserID))
		return err
	}

	// 异步清理旧记录，保留最近 200 条
	go func() {
		_ = r.PruneOldRecords(context.Background(), history.UserID, 200)
	}()

	return nil
}

// FindByUserID 分页查询用户播放历史
func (r *videoHistoryRepositoryImpl) FindByUserID(ctx context.Context, userID uint, page, pageSize int, finished *bool) ([]*model.VideoHistory, int64, error) {
	logger.Debug("查询播放历史", zap.Uint("user_id", userID), zap.Int("page", page))

	history := make([]*model.VideoHistory, 0)
	var total int64

	query := r.db.WithContext(ctx).Model(&model.VideoHistory{}).
		Preload("Video").
		Preload("Video.User").
		Where("user_id = ?", userID)

	if finished != nil {
		query = query.Where("finished = ?", *finished)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.Error("统计播放历史总数失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	// 分页查询，按更新时间倒序
	offset := (page - 1) * pageSize
	err := query.Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&history).Error

	if err != nil {
		logger.Error("查询播放历史失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, 0, err
	}

	return history, total, nil
}

// Delete 软删除指定播放历史（保留数据用于推荐算法）
func (r *videoHistoryRepositoryImpl) Delete(ctx context.Context, userID, historyID uint) error {
	logger.Debug("软删除播放历史", zap.Uint("history_id", historyID))

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", historyID, userID).
		Delete(&model.VideoHistory{})

	if result.Error != nil {
		logger.Error("软删除播放历史失败", zap.Error(result.Error), zap.Uint("history_id", historyID))
		return result.Error
	}

	return nil
}

// ClearAll 软删除用户所有播放历史（保留数据用于推荐算法）
func (r *videoHistoryRepositoryImpl) ClearAll(ctx context.Context, userID uint) error {
	logger.Info("软删除用户所有播放历史", zap.Uint("user_id", userID))

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.VideoHistory{}).Error

	if err != nil {
		logger.Error("软删除播放历史失败", zap.Error(err), zap.Uint("user_id", userID))
		return err
	}

	return nil
}

// PruneOldRecords 清理旧记录，保留最近的 limit 条
func (r *videoHistoryRepositoryImpl) PruneOldRecords(ctx context.Context, userID uint, limit int) error {
	var count int64
	r.db.WithContext(ctx).Model(&model.VideoHistory{}).Where("user_id = ?", userID).Count(&count)

	if count <= int64(limit) {
		return nil
	}

	// 找到第 limit 条记录的 updated_at
	var lastRecord model.VideoHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Offset(limit - 1).
		Limit(1).
		First(&lastRecord).Error

	if err != nil {
		return err
	}

	// 删除早于该时间的记录
	return r.db.WithContext(ctx).
		Where("user_id = ? AND updated_at < ?", userID, lastRecord.UpdatedAt).
		Delete(&model.VideoHistory{}).Error
}
