package repository

import (
	"context"
	"microvibe-go/internal/model"
	"time"

	"gorm.io/gorm"
)

// NotificationRepository 通知仓储层接口
type NotificationRepository interface {
	// CreateNotification 创建通知
	CreateNotification(ctx context.Context, notification *model.Notification) error
	// GetNotificationByID 根据ID获取通知
	GetNotificationByID(ctx context.Context, id uint) (*model.Notification, error)
	// GetNotificationList 获取通知列表
	GetNotificationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Notification, int64, error)
	// MarkAsRead 标记通知为已读
	MarkAsRead(ctx context.Context, id, userID uint) error
	// MarkAllAsRead 标记所有通知为已读
	MarkAllAsRead(ctx context.Context, userID uint) error
	// GetUnreadCount 获取未读通知数
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
	// DeleteNotification 删除通知
	DeleteNotification(ctx context.Context, id, userID uint) error
}

// notificationRepositoryImpl 通知仓储层实现
type notificationRepositoryImpl struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepositoryImpl{db: db}
}

// CreateNotification 创建通知
func (r *notificationRepositoryImpl) CreateNotification(ctx context.Context, notification *model.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

// GetNotificationByID 根据ID获取通知
func (r *notificationRepositoryImpl) GetNotificationByID(ctx context.Context, id uint) (*model.Notification, error) {
	var notification model.Notification
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Sender").
		First(&notification, id).Error
	return &notification, err
}

// GetNotificationList 获取通知列表
func (r *notificationRepositoryImpl) GetNotificationList(ctx context.Context, userID uint, page, pageSize int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Sender").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notifications).Error

	return notifications, total, err
}

// MarkAsRead 标记通知为已读
func (r *notificationRepositoryImpl) MarkAsRead(ctx context.Context, id, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ? AND is_read = ?", id, userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead 标记所有通知为已读
func (r *notificationRepositoryImpl) MarkAllAsRead(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// GetUnreadCount 获取未读通知数
func (r *notificationRepositoryImpl) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// DeleteNotification 删除通知
func (r *notificationRepositoryImpl) DeleteNotification(ctx context.Context, id, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Notification{}).Error
}
