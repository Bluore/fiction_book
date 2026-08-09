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
			"unread_count": gorm.Expr(
				"CASE WHEN unread_count + ? < 0 THEN 0 ELSE unread_count + ? END",
				delta,
				delta,
			),
			"updated_at": time.Now(),
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

		now := time.Now()
		cursors := make([]model.FeedCursor, 0, end-i)
		for _, userID := range userIDs[i:end] {
			cursors = append(cursors, model.FeedCursor{
				UserID:      userID,
				UnreadCount: delta,
				UpdatedAt:   now,
			})
		}

		// FOCUS MySQL计数器累加语句（CASE 表达式）
		err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"unread_count": gorm.Expr(
						"CASE WHEN unread_count + VALUES(unread_count) < 0 THEN 0 ELSE unread_count + VALUES(unread_count) END",
					),
					"updated_at": now,
				}),
			}).
			Create(&cursors).Error

		if err != nil {
			return err
		}
	}

	return nil
}
