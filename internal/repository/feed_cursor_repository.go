package repository

import (
	"context"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FeedCursorRepository interface {
	// GetOrCreate 获取或创建游标
	GetOrCreate(ctx context.Context, userID string) (*model.FeedCursor, error)

	// UpdateUnreadCount 更新未读数量
	UpdateUnreadCount(ctx context.Context, userID string, delta int) error

	// UpdateLastRead 更新最后阅读位置
	UpdateLastRead(ctx context.Context, userID string, feedID string) error

	// ResetUnreadCount 重置未读数量为 0
	ResetUnreadCount(ctx context.Context, userID string) error

	// BatchUpdateUnreadCount 批量增加未读数（写扩散时使用）
	BatchUpdateUnreadCount(ctx context.Context, userIDs []string, delta int) error
}

type feedCursorRepository struct {
	db *gorm.DB
}

func NewFeedCursorRepository(db *gorm.DB) FeedCursorRepository {
	return &feedCursorRepository{db: db}
}

func (r *feedCursorRepository) GetOrCreate(ctx context.Context, userID string) (*model.FeedCursor, error) {
	cursor := &model.FeedCursor{
		UserID:      userID,
		UnreadCount: 0,
		UpdatedAt:   time.Now(),
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).
		Create(cursor).Error

	if err != nil {
		return nil, err
	}

	// 读取实际记录
	var result model.FeedCursor
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&result).Error

	return &result, err
}

func (r *feedCursorRepository) UpdateUnreadCount(ctx context.Context, userID string, delta int) error {
	return r.db.WithContext(ctx).
		Model(&model.FeedCursor{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"unread_count": gorm.Expr("unread_count + ?", delta),
			"updated_at":   time.Now(),
		}).Error
}

func (r *feedCursorRepository) UpdateLastRead(ctx context.Context, userID string, feedID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.FeedCursor{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"last_read_at": now,
			"last_feed_id": feedID,
			"updated_at":   now,
		}).Error
}

func (r *feedCursorRepository) ResetUnreadCount(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&model.FeedCursor{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"updated_at":   time.Now(),
		}).Error
}

func (r *feedCursorRepository) BatchUpdateUnreadCount(ctx context.Context, userIDs []string, delta int) error {
	if len(userIDs) == 0 {
		return nil
	}

	// 批量 upsert
	batchSize := 1000
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}

		batch := userIDs[i:end]

		// 使用 ON DUPLICATE KEY UPDATE 语法（MySQL）
		err := r.db.WithContext(ctx).Exec(`
			INSERT INTO feed_cursor (user_id, unread_count, updated_at)
			VALUES ` + r.buildBatchValues(batch, delta) + `
			ON DUPLICATE KEY UPDATE
				unread_count = unread_count + VALUES(unread_count),
				updated_at = VALUES(updated_at)
		`).Error

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *feedCursorRepository) buildBatchValues(userIDs []string, delta int) string {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	values := ""
	for i, userID := range userIDs {
		if i > 0 {
			values += ","
		}
		values += "('" + userID + "', " + string(rune(delta)) + ", '" + now + "')"
	}
	return values
}
