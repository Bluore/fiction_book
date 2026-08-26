package repository

import (
	"context"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserInboxRepository interface {
	// GetUserInboxIDListByUserIDWithStartCursorAndLimit 游标查询消息返回FeedID切片
	GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startInboxID string, limit int) (feedIDList []string, err error)
	// GetUserInboxIDListByUserIDWithStartCursorAndEndCursor 游标查询消息返回FeedID切片
	GetUserInboxIDListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startInboxID string, endTime time.Time) (feedIDList []string, err error)
	// CreateUserInboxBatch 为一个Feed批量新增收件人消息
	CreateUserInboxBatch(ctx context.Context, db *gorm.DB, userInboxList []*model.UserInbox) error
	// DeleteUserInboxBatch 删除过期消息（软删除）
	DeleteUserInboxBatch(ctx context.Context, db *gorm.DB, expireTime time.Time) error
	// DeleteUserInboxBatchInDisk 删除过期消息（硬删除）
	DeleteUserInboxBatchInDisk(ctx context.Context, db *gorm.DB, expireTime time.Time) error
}

type userInboxRepository struct {
	db *gorm.DB
}

func NewUserInboxRepository(db *gorm.DB) UserInboxRepository {
	return &userInboxRepository{db: db}
}

func (r *userInboxRepository) GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startInboxID string, limit int) ([]string, error) {
	query := db.WithContext(ctx).
		Model(&model.UserInbox{}).
		Where("user_id = ?", userID)

	if !startTime.IsZero() {
		// 列表按 created_at DESC, id DESC 排序，游标表示上一页最后一条
		query = query.Where(
			"(created_at < ? OR (created_at = ? AND feed_id < ?))",
			startTime,
			startTime,
			startInboxID,
		)
	}

	// limit 小于等于 0 时不做限制，由调用方保证传入有效分页大小
	if limit <= 0 {
		return make([]string, 0), fmt.Errorf("limit must be greater than 0")
	}
	query = query.Limit(limit)
	query = query.Order("created_at DESC").Order("id DESC")

	var feedIDList []string
	if err := query.Pluck("feed_id", &feedIDList).Error; err != nil {
		return make([]string, 0), err
	}
	return feedIDList, nil
}

// CreateUserInboxBatch 为一个 Feed 批量新增收件人消息。
// 数量超过 batchMaxSize 直接返回错误，不内部切片，要么全部成功，要么失败返回错误。
func (r *userInboxRepository) CreateUserInboxBatch(ctx context.Context, db *gorm.DB, userInboxList []*model.UserInbox) error {
	if len(userInboxList) == 0 {
		return nil
	}
	if len(userInboxList) > batchMaxSize {
		return fmt.Errorf("user inbox batch size %d exceeds max %d", len(userInboxList), batchMaxSize)
	}

	// 补齐缺失的 ID，避免主键冲突
	for _, userInbox := range userInboxList {
		if userInbox != nil && userInbox.ID == "" {
			userInbox.ID = uuid.NewString()
		}
	}

	return db.WithContext(ctx).Create(userInboxList).Error
}

// DeleteUserInboxBatch 删除过期消息（软删除）。
// 单条 DELETE 直接删除，不返回删除列表，要么全部成功，要么失败返回错误。
func (r *userInboxRepository) DeleteUserInboxBatch(ctx context.Context, db *gorm.DB, expireTime time.Time) error {
	return db.WithContext(ctx).
		Where("created_at < ?", expireTime).
		Delete(&model.UserInbox{}).Error
}

// DeleteUserInboxBatchInDisk 删除过期消息（硬删除）。
// 单条 DELETE 直接删除，不返回删除列表，要么全部成功，要么失败返回错误。
func (r *userInboxRepository) DeleteUserInboxBatchInDisk(ctx context.Context, db *gorm.DB, expireTime time.Time) error {
	return db.WithContext(ctx).
		Unscoped().
		Where("created_at < ?", expireTime).
		Delete(&model.UserInbox{}).Error
}

func (r *userInboxRepository) GetUserInboxIDListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startInboxID string, endTime time.Time) ([]string, error) {
	query := db.WithContext(ctx).
		Model(&model.UserInbox{}).
		Where("user_id = ?", userID)

	if endTime.IsZero() {
		return make([]string, 0), fmt.Errorf("endTime is zero")
	}

	if !startTime.IsZero() {
		// 列表按 created_at DESC, id DESC 排序，游标表示上一页最后一条
		query = query.Where(
			"(created_at > ? AND (created_at, feed_id) < (?, ?) )",
			endTime,
			startTime,
			startInboxID,
		)
	}

	query = query.Order("created_at DESC").Order("feed_id DESC")

	var feedIDList []string
	if err := query.Pluck("feed_id", &feedIDList).Error; err != nil {
		return make([]string, 0), err
	}
	return feedIDList, nil
}
