package repository

import (
	"context"
	"fmt"
	"time"

	"microvibe-go/internal/model"
	pkgerrors "microvibe-go/pkg/errors"
	"microvibe-go/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	commentLikeCacheKey = "comment:like:%d:%d" // comment:like:{userID}:{commentID}
)

// CommentRepository 评论数据访问层接口
type CommentRepository interface {
	// Create 创建评论
	Create(ctx context.Context, comment *model.Comment) error
	// FindByID 根据ID查找评论
	FindByID(ctx context.Context, id uint) (*model.Comment, error)
	// FindByVideoID 查找视频的评论列表（分页）
	FindByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Comment, int64, error)
	// FindTopLevelByVideoID 查找视频的顶级评论列表（分页，只返回ParentID为NULL的评论）
	FindTopLevelByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Comment, int64, error)
	// FindByParentID 查找子评论列表
	FindByParentID(ctx context.Context, parentID uint, limit, offset int) ([]*model.Comment, error)
	// FindByRootID 根据根评论ID查找所有后代评论（支持嵌套回复链和分页）
	FindByRootID(ctx context.Context, rootID uint, limit, offset int) ([]*model.Comment, error)
	// Update 更新评论
	Update(ctx context.Context, comment *model.Comment) error
	// Delete 删除评论
	Delete(ctx context.Context, id uint) error
	// IncrementLikeCount 增加点赞数
	IncrementLikeCount(ctx context.Context, id uint) error
	// DecrementLikeCount 减少点赞数
	DecrementLikeCount(ctx context.Context, id uint) error
	// IncrementReplyCount 增加回复数
	IncrementReplyCount(ctx context.Context, id uint) error
	// DecrementReplyCount 减少回复数
	DecrementReplyCount(ctx context.Context, id uint) error
	// CountByVideoID 统计视频评论数
	CountByVideoID(ctx context.Context, videoID uint) (int64, error)
	// CountByRootID 统计根评论的所有后代回复数
	CountByRootID(ctx context.Context, rootID uint) (int64, error)

	// CreateCommentLike 创建评论点赞记录
	CreateCommentLike(ctx context.Context, userID, commentID uint) error
	// DeleteCommentLike 删除评论点赞记录
	DeleteCommentLike(ctx context.Context, userID, commentID uint) error
	// HasCommentLike 检查用户是否已点赞评论
	HasCommentLike(ctx context.Context, userID, commentID uint) (bool, error)

	// FindMentionsByCommentID 获取评论的提及列表
	FindMentionsByCommentID(ctx context.Context, commentID uint) ([]*model.CommentMention, error)
	// FindMentionsByCommentIDs 批量获取评论的提及列表
	FindMentionsByCommentIDs(ctx context.Context, commentIDs []uint) ([]*model.CommentMention, error)
	// CreateMention 创建提及记录
	CreateMention(ctx context.Context, mention *model.CommentMention) error
	// ResetAndPinComment 重置并置顶新评论
	ResetAndPinComment(ctx context.Context, videoID, commentID uint, isPinned bool) error
	// FindReceivedByUserID 获取创作者收到的评论
	FindReceivedByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Comment, int64, error)
	// FindSentByUserID 获取用户发出的评论
	FindSentByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Comment, int64, error)
}

// commentRepositoryImpl 评论数据访问层实现
type commentRepositoryImpl struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewCommentRepository 创建评论数据访问层实例
func NewCommentRepository(db *gorm.DB, redisClient *redis.Client) CommentRepository {
	return &commentRepositoryImpl{
		db:    db,
		redis: redisClient,
	}
}

// Create 创建评论
func (r *commentRepositoryImpl) Create(ctx context.Context, comment *model.Comment) error {
	logger.Debug("创建评论", zap.Uint("user_id", comment.UserID), zap.Uint("video_id", comment.VideoID))

	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		logger.Error("创建评论失败", zap.Error(err))
		return err
	}

	logger.Info("评论创建成功", zap.Uint("comment_id", comment.ID))
	return nil
}

// FindByID 根据ID查找评论
func (r *commentRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	logger.Debug("查找评论", zap.Uint("comment_id", id))

	var comment model.Comment
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReplyToUser").
		First(&comment, id).Error; err != nil {
		if pkgerrors.IsNotFound(err) {
			logger.Warn("评论不存在", zap.Uint("comment_id", id))
		} else {
			logger.Error("查找评论失败", zap.Error(err))
		}
		return nil, err
	}

	return &comment, nil
}

// FindByVideoID 查找视频的评论列表（分页）
func (r *commentRepositoryImpl) FindByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Comment, int64, error) {
	logger.Debug("查找视频评论列表", zap.Uint("video_id", videoID), zap.Int("limit", limit), zap.Int("offset", offset))

	var comments []*model.Comment
	var total int64

	// 查询总数
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("video_id = ? AND status = 1", videoID).
		Count(&total).Error; err != nil {
		logger.Error("统计视频评论数失败", zap.Error(err))
		return nil, 0, err
	}

	// 查询评论列表（返回所有评论，包括回复，以扁平化形式）
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReplyToUser").
		Where("video_id = ? AND status = 1", videoID).
		Order("is_pinned DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		logger.Error("查找视频评论列表失败", zap.Error(err))
		return nil, 0, err
	}

	return comments, total, nil
}

// FindTopLevelByVideoID 查找视频的顶级评论列表（分页，只返回ParentID为NULL的评论）
func (r *commentRepositoryImpl) FindTopLevelByVideoID(ctx context.Context, videoID uint, limit, offset int) ([]*model.Comment, int64, error) {
	logger.Debug("查找视频顶级评论列表", zap.Uint("video_id", videoID), zap.Int("limit", limit), zap.Int("offset", offset))

	var comments []*model.Comment
	var total int64

	// 查询总数（只统计顶级评论）
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("video_id = ? AND parent_id IS NULL AND status = 1", videoID).
		Count(&total).Error; err != nil {
		logger.Error("统计视频顶级评论数失败", zap.Error(err))
		return nil, 0, err
	}

	// 查询顶级评论列表（ParentID 为 NULL）
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReplyToUser").
		Where("video_id = ? AND parent_id IS NULL AND status = 1", videoID).
		Order("is_pinned DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		logger.Error("查找视频顶级评论列表失败", zap.Error(err))
		return nil, 0, err
	}

	return comments, total, nil
}

// FindByParentID 查找子评论列表
func (r *commentRepositoryImpl) FindByParentID(ctx context.Context, parentID uint, limit, offset int) ([]*model.Comment, error) {
	logger.Debug("查找子评论列表", zap.Uint("parent_id", parentID))

	var comments []*model.Comment
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReplyToUser").
		Where("parent_id = ? AND status = 1", parentID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		logger.Error("查找子评论列表失败", zap.Error(err))
		return nil, err
	}

	return comments, nil
}

// FindByRootID 根据根评论ID查找所有后代评论（支持嵌套回复 A→B→C，支持分页）
func (r *commentRepositoryImpl) FindByRootID(ctx context.Context, rootID uint, limit, offset int) ([]*model.Comment, error) {
	logger.Debug("查找根评论的所有后代", zap.Uint("root_id", rootID))

	var comments []*model.Comment
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReplyToUser").
		Where("root_id = ? AND status = 1", rootID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		logger.Error("查找后代评论失败", zap.Error(err))
		return nil, err
	}

	return comments, nil
}

// Update 更新评论
func (r *commentRepositoryImpl) Update(ctx context.Context, comment *model.Comment) error {
	logger.Debug("更新评论", zap.Uint("comment_id", comment.ID))

	if err := r.db.WithContext(ctx).Save(comment).Error; err != nil {
		logger.Error("更新评论失败", zap.Error(err))
		return err
	}

	logger.Info("评论更新成功", zap.Uint("comment_id", comment.ID))
	return nil
}

// Delete 删除评论
func (r *commentRepositoryImpl) Delete(ctx context.Context, id uint) error {
	logger.Debug("删除评论", zap.Uint("comment_id", id))

	if err := r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error; err != nil {
		logger.Error("删除评论失败", zap.Error(err))
		return err
	}

	logger.Info("评论删除成功", zap.Uint("comment_id", id))
	return nil
}

// IncrementLikeCount 增加点赞数
func (r *commentRepositoryImpl) IncrementLikeCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		logger.Error("增加评论点赞数失败", zap.Error(err))
		return err
	}

	return nil
}

// DecrementLikeCount 减少点赞数
func (r *commentRepositoryImpl) DecrementLikeCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ? AND like_count > 0", id).
		UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
		logger.Error("减少评论点赞数失败", zap.Error(err))
		return err
	}

	return nil
}

// IncrementReplyCount 增加回复数
func (r *commentRepositoryImpl) IncrementReplyCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", id).
		UpdateColumn("reply_count", gorm.Expr("reply_count + ?", 1)).Error; err != nil {
		logger.Error("增加评论回复数失败", zap.Error(err))
		return err
	}

	return nil
}

// DecrementReplyCount 减少回复数
func (r *commentRepositoryImpl) DecrementReplyCount(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ? AND reply_count > 0", id).
		UpdateColumn("reply_count", gorm.Expr("reply_count - ?", 1)).Error; err != nil {
		logger.Error("减少评论回复数失败", zap.Error(err))
		return err
	}

	return nil
}

// CountByVideoID 统计视频评论数
func (r *commentRepositoryImpl) CountByVideoID(ctx context.Context, videoID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("video_id = ? AND status = 1", videoID).
		Count(&count).Error; err != nil {
		logger.Error("统计视频评论数失败", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// CountByRootID 统计根评论的所有后代回复数
func (r *commentRepositoryImpl) CountByRootID(ctx context.Context, rootID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("root_id = ? AND status = 1", rootID).
		Count(&count).Error; err != nil {
		logger.Error("统计后代回复数失败", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// CreateCommentLike 创建评论点赞记录（幂等操作）
func (r *commentRepositoryImpl) CreateCommentLike(ctx context.Context, userID, commentID uint) error {
	logger.Debug("创建评论点赞记录", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))

	commentLike := &model.CommentLike{
		UserID:    userID,
		CommentID: commentID,
	}

	// 使用 FirstOrCreate 实现幂等性，如果已存在则不重复创建
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		FirstOrCreate(commentLike).Error; err != nil {
		logger.Error("创建评论点赞记录失败", zap.Error(err))
		return err
	}

	// 设置缓存
	if r.redis != nil {
		key := fmt.Sprintf(commentLikeCacheKey, userID, commentID)
		r.redis.Set(ctx, key, "1", time.Hour*24)
	}

	logger.Info("评论点赞记录创建成功", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
	return nil
}

// DeleteCommentLike 删除评论点赞记录
func (r *commentRepositoryImpl) DeleteCommentLike(ctx context.Context, userID, commentID uint) error {
	logger.Debug("删除评论点赞记录", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))

	result := r.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Delete(&model.CommentLike{})

	if result.Error != nil {
		logger.Error("删除评论点赞记录失败", zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warn("评论点赞记录不存在", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
	} else {
		// 删除缓存
		if r.redis != nil {
			key := fmt.Sprintf(commentLikeCacheKey, userID, commentID)
			r.redis.Del(ctx, key)
		}
		logger.Info("评论点赞记录删除成功", zap.Uint("user_id", userID), zap.Uint("comment_id", commentID))
	}

	return nil
}

// HasCommentLike 检查用户是否已点赞评论
func (r *commentRepositoryImpl) HasCommentLike(ctx context.Context, userID, commentID uint) (bool, error) {
	// 先查缓存
	if r.redis != nil {
		key := fmt.Sprintf(commentLikeCacheKey, userID, commentID)
		val, err := r.redis.Get(ctx, key).Result()
		if err == nil {
			return val == "1", nil
		}
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Count(&count).Error; err != nil {
		logger.Error("检查评论点赞状态失败", zap.Error(err))
		return false, err
	}

	hasLiked := count > 0

	// 回写缓存
	if r.redis != nil {
		key := fmt.Sprintf(commentLikeCacheKey, userID, commentID)
		val := "0"
		if hasLiked {
			val = "1"
		}
		r.redis.Set(ctx, key, val, time.Hour*24)
	}

	return hasLiked, nil
}

// FindMentionsByCommentID 获取评论的提及列表
func (r *commentRepositoryImpl) FindMentionsByCommentID(ctx context.Context, commentID uint) ([]*model.CommentMention, error) {
	var mentions []*model.CommentMention
	err := r.db.WithContext(ctx).Preload("User").Where("comment_id = ?", commentID).Find(&mentions).Error
	return mentions, err
}

// FindMentionsByCommentIDs 批量获取评论的提及列表
func (r *commentRepositoryImpl) FindMentionsByCommentIDs(ctx context.Context, commentIDs []uint) ([]*model.CommentMention, error) {
	if len(commentIDs) == 0 {
		return nil, nil
	}
	var mentions []*model.CommentMention
	err := r.db.WithContext(ctx).Preload("User").Where("comment_id IN ?", commentIDs).Find(&mentions).Error
	return mentions, err
}

// CreateMention 创建提及记录
func (r *commentRepositoryImpl) CreateMention(ctx context.Context, mention *model.CommentMention) error {
	return r.db.WithContext(ctx).Create(mention).Error
}

// ResetAndPinComment 重置并置顶新评论 (原子操作)
func (r *commentRepositoryImpl) ResetAndPinComment(ctx context.Context, videoID, commentID uint, isPinned bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPinned {
			// 1. 先取消该视频下所有已置顶的评论
			if err := tx.Model(&model.Comment{}).
				Where("video_id = ? AND is_pinned = ?", videoID, true).
				Update("is_pinned", false).Error; err != nil {
				return err
			}
		}

		// 2. 更新目标评论状态
		if err := tx.Model(&model.Comment{}).
			Where("id = ?", commentID).
			Update("is_pinned", isPinned).Error; err != nil {
			return err
		}

		return nil
	})
}

// FindReceivedByUserID 获取创作者收到的评论
func (r *commentRepositoryImpl) FindReceivedByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	db := r.db.WithContext(ctx).
		Table("comments").
		Joins("JOIN videos ON videos.id = comments.video_id").
		Where("videos.user_id = ? AND comments.status = 1", userID)

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := db.Preload("User").
		Preload("Video").
		Order("comments.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// FindSentByUserID 获取用户发出的评论
func (r *commentRepositoryImpl) FindSentByUserID(ctx context.Context, userID uint, limit, offset int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("user_id = ? AND status = 1", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Video").
		Where("user_id = ? AND status = 1", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}
