package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type UserInboxRepository interface {
	// GetUserInboxIDListByUserIDWithStartCursorAndLimit 游标查询消息返回FeedID切片
	GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startCursor string, startInboxID string, limit int64) (feedIDList []string, err error)
	// CreateUserInboxBatch 批量新增消息
	CreateUserInboxBatch(ctx context.Context, db *gorm.DB, userID []string) (successUserIDList []string, err error)
	// DeleteUserInboxBatch 删除过期消息（软删除）
	DeleteUserInboxBatch(ctx context.Context, db *gorm.DB, expireTime time.Time) (successDeleteIDList []string, err error)
	// DeleteUserInboxBatchInDisk 删除过期消息（硬删除）
	DeleteUserInboxBatchInDisk(ctx context.Context, db *gorm.DB, expireTime time.Time) (successDeleteIDList []string, err error)
}
