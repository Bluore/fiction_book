package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConsumeRecordRepository interface {
	// IsConsumed 检查消息是否已经被消费过（幂等检查）
	IsConsumed(ctx context.Context, db *gorm.DB, messageID string) (bool, error)

	// TryMarkConsumed 在事务中尝试标记消息已消费，返回是否首次写入
	TryMarkConsumed(ctx context.Context, db *gorm.DB, messageID string) (isFirstConsume bool, err error)
}

type consumeRecordRepository struct {
	db *gorm.DB
}

func NewConsumeRecordRepository(db *gorm.DB) ConsumeRecordRepository {
	return &consumeRecordRepository{db: db}
}

func (r *consumeRecordRepository) IsConsumed(ctx context.Context, db *gorm.DB, messageID string) (bool, error) {
	var record model.ConsumeRecord
	err := db.WithContext(ctx).
		Where("id = ?", messageID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *consumeRecordRepository) TryMarkConsumed(
	ctx context.Context,
	db *gorm.DB,
	messageID string,
) (bool, error) {
	record := &model.ConsumeRecord{
		ID:        messageID,
		CreatedAt: time.Now(),
	}

	// FOCUS 冲突时继续
	res := db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
