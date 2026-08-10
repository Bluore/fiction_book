package repository

import (
	"context"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const batchMaxSize = 1000

type FeedRepository interface {
	// CreateBatchIgnoreConflict 在指定事务中批量创建 feed，主键冲突时忽略（写扩散）
	CreateBatchIgnoreConflict(ctx context.Context, db *gorm.DB, feeds []*model.FeedInbox) (successNum int64, err error)

	// GetUserFeeds 获取用户的 feed 流
	GetUserFeeds(ctx context.Context, db *gorm.DB, filter model.FeedFilterWithTimePage) (feeds []*model.FeedInbox, err error)

	// CountUserUnread 统计用户未读数量
	CountUserUnread(ctx context.Context, db *gorm.DB, userID string) (unreadNum int64, err error)

	// MarkAsRead 标记为已读
	MarkAsRead(ctx context.Context, db *gorm.DB, userID string, feedIDs []string) (int64, error)

	// MarkAllAsRead 标记所有为已读
	MarkAllAsRead(ctx context.Context, db *gorm.DB, userID string) error

	// DeleteOldFeeds 删除过期 feed（清理任务）
	DeleteOldFeeds(ctx context.Context, db *gorm.DB, before time.Time) error

	// DeleteByBookID 删除指定书籍的所有 feed
	DeleteByBookID(ctx context.Context, db *gorm.DB, bookID string) error
}

type feedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) FeedRepository {
	return &feedRepository{db: db}
}

func (r *feedRepository) CreateBatch(ctx context.Context, feeds []*model.FeedInbox) error {
	if len(feeds) == 0 {
		return nil
	}

	// 批量插入，每批 1000 条
	batchSize := 1000
	for i := 0; i < len(feeds); i += batchSize {
		end := i + batchSize
		if end > len(feeds) {
			end = len(feeds)
		}

		if err := r.db.WithContext(ctx).Create(feeds[i:end]).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *feedRepository) CreateBatchIgnoreConflict(
	ctx context.Context,
	db *gorm.DB,
	feeds []*model.FeedInbox,
) (int64, error) {
	if len(feeds) == 0 {
		return 0, nil
	}

	var affected int64

	for i := 0; i < len(feeds); i += batchMaxSize {
		end := i + batchMaxSize
		if end > len(feeds) {
			end = len(feeds)
		}

		res := db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(feeds[i:end], batchMaxSize)
		if res.Error != nil {
			return 0, res.Error
		}
		affected += res.RowsAffected
	}

	return affected, nil
}

func (r *feedRepository) GetUserFeeds(
	ctx context.Context,
	db *gorm.DB,
	filter model.FeedFilterWithTimePage,
) ([]*model.FeedInbox, error) {
	var feeds []*model.FeedInbox

	query := db.WithContext(ctx).Model(&model.FeedInbox{})

	// 用户ID过滤
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}

	// 事件类型过滤
	if len(filter.EventTypes) > 0 {
		query = query.Where("event_type IN ?", filter.EventTypes)
	}

	// 书籍ID过滤
	if filter.BookID != "" {
		query = query.Where("book_id = ?", filter.BookID)
	}

	// 已读状态过滤
	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}

	// 游标分页：与 ORDER BY created_at DESC, id ASC 保持一致
	if !filter.LastTime.IsZero() && filter.LastID != nil {
		query = query.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			filter.LastTime,
			filter.LastTime,
			*filter.LastID,
		)
	}

	query = query.Order("created_at DESC").Order("id DESC")

	// 限制数量
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(20) // 默认 20 条
	}

	if err := query.Find(&feeds).Error; err != nil {
		return nil, err
	}

	return feeds, nil
}

func (r *feedRepository) CountUserUnread(ctx context.Context, db *gorm.DB, userID string) (int64, error) {
	var count int64

	err := db.WithContext(ctx).
		Model(&model.FeedInbox{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error

	return count, err
}

func (r *feedRepository) MarkAsRead(ctx context.Context, db *gorm.DB, userID string, feedIDs []string) (int64, error) {
	if len(feedIDs) == 0 {
		return 0, nil
	}

	res := db.WithContext(ctx).
		Model(&model.FeedInbox{}).
		Where("user_id = ? AND id IN ? AND is_read = ?", userID, feedIDs, false).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (r *feedRepository) MarkAllAsRead(ctx context.Context, db *gorm.DB, userID string) error {
	return db.WithContext(ctx).
		Model(&model.FeedInbox{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		}).Error
}

func (r *feedRepository) DeleteOldFeeds(ctx context.Context, db *gorm.DB, before time.Time) error {
	return db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&model.FeedInbox{}).
		Error
}

func (r *feedRepository) DeleteByBookID(ctx context.Context, db *gorm.DB, bookID string) error {
	return db.WithContext(ctx).
		Where("book_id = ?", bookID).
		Delete(&model.FeedInbox{}).
		Error
}
