package repository

import (
	"context"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
)

type UserFeedRepository interface {
	// GetFeedByFeedID 根据FeedID查寻单个Feed
	GetFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) (feedDetail *model.UserFeed, err error)
	// GetFeedListByUserIDWithStartCursorAndLimit 起始游标和数量限制查询
	GetFeedListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, itemLimit int64) (feedDetailList []*model.UserFeed, err error)
	// GetFeedListByUserIDWithStartCursorAndEndCursor 起始游标和结束游标查询
	GetFeedListByUserIDWithStartCursorAndEndCursor(ctx context.Context, db *gorm.DB, userID string, startTime time.Time, startFeedID string, endTime time.Time) (feedDetailList []*model.UserFeed, err error)
	// UpdateFeedByFeedID 根据FeedID更新单个Feed
	UpdateFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string, feedDetail *model.UserFeed) (err error)
	// DeleteFeedByFeedID 根据FeedID删除Feed（软删除）
	DeleteFeedByFeedID(ctx context.Context, db *gorm.DB, feedID string) (err error)
}
