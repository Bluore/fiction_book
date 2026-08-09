package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
)

type ConsumeRecordRepository interface {
	// IsConsumed 检查消息是否已经被消费过（幂等检查）
	IsConsumed(ctx context.Context, messageID string) (bool, error)

	// MarkConsumed 标记消息已消费
	MarkConsumed(ctx context.Context, messageID string) error
}

type consumeRecordRepository struct {
	db *gorm.DB
}

func NewConsumeRecordRepository(db *gorm.DB) ConsumeRecordRepository {
	return &consumeRecordRepository{db: db}
}

func (r *consumeRecordRepository) IsConsumed(ctx context.Context, messageID string) (bool, error) {
	var record model.ConsumeRecord
	err := r.db.WithContext(ctx).
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

func (r *consumeRecordRepository) MarkConsumed(ctx context.Context, messageID string) error {
	record := &model.ConsumeRecord{
		ID:        messageID,
		CreatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(record).Error
}
