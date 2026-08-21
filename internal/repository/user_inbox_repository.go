package repository

import (
	"context"
	"fiction-book/internal/model"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserInboxRepository interface {
	// GetUserInboxIDListByUserIDWithStartCursorAndLimit 游标查询消息返回FeedID切片
	GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startCursor string, startInboxID string, limit int) (feedIDList []string, err error)
	// CreateUserInboxBatch 为一个Feed批量新增收件人消息
	CreateUserInboxBatch(ctx context.Context, db *gorm.DB, feedID string, userID []string) (successUserIDList []string, err error)
	// DeleteUserInboxBatch 删除过期消息（软删除）
	DeleteUserInboxBatch(ctx context.Context, db *gorm.DB, expireTime time.Time) (successDeleteIDList []string, err error)
	// DeleteUserInboxBatchInDisk 删除过期消息（硬删除）
	DeleteUserInboxBatchInDisk(ctx context.Context, db *gorm.DB, expireTime time.Time) (successDeleteIDList []string, err error)
}

type userInboxRepository struct {
	db *gorm.DB
}

func NewUserInboxRepository(db *gorm.DB) UserInboxRepository {
	return &userInboxRepository{db: db}
}

func (r *userInboxRepository) GetUserInboxIDListByUserIDWithStartCursorAndLimit(ctx context.Context, db *gorm.DB, userID string, startCursor string, startInboxID string, limit int) ([]string, error) {
	query := db.WithContext(ctx).
		Model(&model.UserInbox{}).
		Where("user_id = ?", userID)

	startTime, hasCursor, err := parseInboxCursor(startCursor)
	if err != nil {
		return nil, err
	}
	if hasCursor {
		// 列表按 created_at DESC, id DESC 排序，游标表示上一页最后一条
		query = query.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			startTime,
			startTime,
			startInboxID,
		)
	}

	// limit 小于等于 0 时不做限制，由调用方保证传入有效分页大小
	if limit > 0 {
		query = query.Limit(int(limit))
	}
	query = query.Order("created_at DESC").Order("id DESC")

	var feedIDList []string
	if err := query.Pluck("feed_id", &feedIDList).Error; err != nil {
		return nil, err
	}
	return feedIDList, nil
}

func (r *userInboxRepository) CreateUserInboxBatch(ctx context.Context, db *gorm.DB, feedID string, userID []string) ([]string, error) {
	if feedID == "" || len(userID) == 0 {
		return nil, nil
	}

	inboxList := make([]*model.UserInbox, 0, len(userID))
	for _, uid := range userID {
		if uid == "" {
			continue
		}
		inboxList = append(inboxList, &model.UserInbox{
			ID:     uuid.NewString(),
			UserID: uid,
			FeedID: feedID,
		})
	}
	if len(inboxList) == 0 {
		return nil, nil
	}

	successUserIDList := make([]string, 0, len(inboxList))
	for start := 0; start < len(inboxList); start += batchMaxSize {
		end := start + batchMaxSize
		if end > len(inboxList) {
			end = len(inboxList)
		}
		chunk := inboxList[start:end]
		if err := db.WithContext(ctx).Create(chunk).Error; err != nil {
			return successUserIDList, err
		}
		for _, inbox := range chunk {
			successUserIDList = append(successUserIDList, inbox.UserID)
		}
	}
	return successUserIDList, nil
}

func (r *userInboxRepository) DeleteUserInboxBatch(ctx context.Context, db *gorm.DB, expireTime time.Time) ([]string, error) {
	idList, err := r.pluckExpiredInboxIDList(ctx, db, expireTime, false)
	if err != nil || len(idList) == 0 {
		return nil, err
	}

	err = db.WithContext(ctx).
		Where("id IN ?", idList).
		Delete(&model.UserInbox{}).Error
	if err != nil {
		return nil, err
	}
	return idList, nil
}

func (r *userInboxRepository) DeleteUserInboxBatchInDisk(
	ctx context.Context,
	db *gorm.DB,
	expireTime time.Time,
) ([]string, error) {
	idList, err := r.pluckExpiredInboxIDList(ctx, db, expireTime, true)
	if err != nil || len(idList) == 0 {
		return nil, err
	}

	err = db.WithContext(ctx).
		Unscoped().
		Where("id IN ?", idList).
		Delete(&model.UserInbox{}).Error
	if err != nil {
		return nil, err
	}
	return idList, nil
}

// pluckExpiredInboxIDList 查询创建时间早于 expireTime 的收件箱记录 ID。
// inDisk 为 true 时包含已软删除的记录（硬删除清理场景），否则只查未删除记录。
func (r *userInboxRepository) pluckExpiredInboxIDList(ctx context.Context, db *gorm.DB, expireTime time.Time, inDisk bool) ([]string, error) {
	query := db.WithContext(ctx).Model(&model.UserInbox{})
	if inDisk {
		query = query.Unscoped()
	}

	var idList []string
	err := query.
		Where("created_at < ?", expireTime).
		Pluck("id", &idList).Error
	if err != nil {
		return nil, err
	}
	return idList, nil
}

// parseInboxCursor 解析收件箱游标中的时间部分，支持 RFC3339、MySQL datetime 和 unix 时间戳格式。
// 返回 (时间, 是否有游标, 错误)；空字符串表示从最新开始，hasCursor 为 false。
func parseInboxCursor(startCursor string) (time.Time, bool, error) {
	if startCursor == "" {
		return time.Time{}, false, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, startCursor); err == nil {
			return t, true, nil
		}
	}

	if n, err := strconv.ParseInt(startCursor, 10, 64); err == nil {
		if n > 1e12 {
			return time.UnixMilli(n), true, nil
		}
		return time.Unix(n, 0), true, nil
	}
	return time.Time{}, false, fmt.Errorf("invalid inbox cursor time format: %s", startCursor)
}
