package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
)

type UserFeedRepository interface {
	// CreateUserFeed 创建 Feed
	CreateUserFeed(ctx context.Context, db *gorm.DB, userFeed *model.UserFeed) error
	// GetFeedByFeedID 根据FeedID查寻单个Feed
	GetFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) (feedDetail *model.UserFeed, err error)
	// GetFeedListByUserIDWithStartCursorAndLimit 起始游标和数量限制查询
	GetFeedListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, itemLimit int) (feedDetailList []*model.UserFeed, err error)
	// GetFeedListWithStartCursorAndLimit 全局 Feed 起始游标和数量限制查询（不区分用户，供游客 Feed 使用）
	// 按 created_at DESC, feed_id DESC 排序，返回比游标 (startTime, startFeedID) 更旧的 Feed
	GetFeedListWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, startTime time.Time, startFeedID string, itemLimit int) (feedDetailList []*model.UserFeed, err error)
	// GetFeedListByUserIDWithStartCursorAndEndCursor 起始游标和结束游标查询
	// 返回起始游标 (startTime, startFeedID)（不含）到结束游标 endTime（含）之间的 Feed 列表，按 created_at DESC 排序
	GetFeedListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, endTime time.Time) (feedDetailList []*model.UserFeed, err error)
	// GetFeedIDListByUserIDWithStartCursorAndEndCursor 起始游标和结束游标查询
	// 返回起始游标 (startTime, startFeedID)（不含）到结束游标 endTime（含）之间的 FeedID 列表，按 created_at DESC 排序
	GetFeedIDListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, endTime time.Time) (feedIDList []string, err error)
	// UpdateFeedByFeedID 根据FeedID更新单个Feed
	UpdateFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string, feedDetail *model.UserFeed) (err error)
	// DeleteFeedByFeedID 根据FeedID删除Feed（软删除）
	DeleteFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) (err error)
}

type userFeedRepository struct {
	db *gorm.DB
}

func NewUserFeedRepository(db *gorm.DB) UserFeedRepository {
	return &userFeedRepository{db: db}
}

func (r *userFeedRepository) GetFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) (*model.UserFeed, error) {
	var feedDetail model.UserFeed
	err := db.WithContext(ctx).
		Where("feed_id = ?", feedID).
		First(&feedDetail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feedDetail, nil
}

func (r *userFeedRepository) GetFeedListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, itemLimit int) ([]*model.UserFeed, error) {
	query := db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ?", userID)

	query = applyUserFeedCursor(query, startTime, startFeedID)

	// itemLimit 小于等于 0 时不做限制，由调用方保证传入有效分页大小
	if itemLimit > 0 {
		query = query.Limit(int(itemLimit))
	}
	query = query.Order("created_at DESC").Order("feed_id DESC")

	var feedDetailList []*model.UserFeed
	if err := query.Find(&feedDetailList).Error; err != nil {
		return nil, err
	}
	return feedDetailList, nil
}

func (r *userFeedRepository) GetFeedListWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, startTime time.Time, startFeedID string, itemLimit int) ([]*model.UserFeed, error) {
	query := db.WithContext(ctx).
		Model(&model.UserFeed{})

	query = applyUserFeedCursor(query, startTime, startFeedID)

	// itemLimit 小于等于 0 时不做限制，由调用方保证传入有效分页大小
	if itemLimit > 0 {
		query = query.Limit(int(itemLimit))
	}
	query = query.Order("created_at DESC").Order("feed_id DESC")

	var feedDetailList []*model.UserFeed
	if err := query.Find(&feedDetailList).Error; err != nil {
		return nil, err
	}
	return feedDetailList, nil
}

func (r *userFeedRepository) GetFeedListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, endTime time.Time) ([]*model.UserFeed, error) {
	query := db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ?", userID)

	// 起始游标是窗口的最新边界（不含），结束游标 endTime 是最旧边界（含）
	query = applyUserFeedCursor(query, startTime, startFeedID)
	if !endTime.IsZero() {
		query = query.Where("created_at >= ?", endTime)
	}
	query = query.Order("created_at DESC").Order("feed_id DESC")

	var feedDetailList []*model.UserFeed
	if err := query.Find(&feedDetailList).Error; err != nil {
		return nil, err
	}
	return feedDetailList, nil
}

func (r *userFeedRepository) UpdateFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string, feedDetail *model.UserFeed) error {
	if feedDetail == nil {
		return errors.New("user feed detail is nil")
	}
	return db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("feed_id = ?", feedID).
		Updates(feedDetail).Error
}

func (r *userFeedRepository) DeleteFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) error {
	return db.WithContext(ctx).
		Where("feed_id = ?", feedID).
		Delete(&model.UserFeed{}).Error
}

// applyUserFeedCursor 应用 Feed 游标条件。列表按 created_at DESC, feed_id DESC 排序，
// 游标 (startTime, startFeedID) 表示上一页最后一条，本页只返回严格更旧的记录（不含游标本身）。
// startTime 为零值时表示从最新开始，不附加游标条件。
func applyUserFeedCursor(query *gorm.DB, startTime time.Time, startFeedID string) *gorm.DB {
	if startTime.IsZero() {
		return query
	}
	return query.Where(
		"(created_at < ? OR (created_at = ? AND feed_id < ?))",
		startTime,
		startTime,
		startFeedID,
	)
}

func (r *userFeedRepository) GetFeedIDListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, endTime time.Time) ([]string, error) {
	query := db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ?", userID)

	// 起始游标是窗口的最新边界（不含），结束游标 endTime 是最旧边界（含）
	query = applyUserFeedCursor(query, startTime, startFeedID)
	if !endTime.IsZero() {
		query = query.Where("created_at >= ?", endTime)
	}
	query = query.Order("created_at DESC").Order("feed_id DESC")

	feedIDList := make([]string, 0)
	if err := query.Pluck("feed_id", &feedIDList).Error; err != nil {
		return nil, err
	}
	return feedIDList, nil
}

func (r *userFeedRepository) CreateUserFeed(ctx context.Context, db *gorm.DB, userFeed *model.UserFeed) error {
	return db.WithContext(ctx).Create(userFeed).Error
}
