package repository

import (
	"context"
	"fiction-book/internal/model"

	"gorm.io/gorm"
)

type OutboxRepository interface {
	AddEvent(ctx context.Context, db *gorm.DB, outbox *model.Outbox) error
	GetEventByTableName(ctx context.Context, db *gorm.DB, table interface{}) (*model.Outbox, error)
	FinishEventByTableNameAndID(ctx context.Context, db *gorm.DB, table interface{}, id string) error
}

type outboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) OutboxRepository {
	return &outboxRepository{
		db: db,
	}
}

func (r outboxRepository) AddEvent(ctx context.Context, db *gorm.DB, outbox *model.Outbox) error {
	return db.WithContext(ctx).Create(outbox).Error
}

func (r outboxRepository) GetEventByTableName(ctx context.Context, db *gorm.DB, table interface{}) (*model.Outbox, error) {
	var outbox model.Outbox
	err := db.WithContext(ctx).Model(table).Where("status = ?", model.OutboxStatusCreated).Order("created_at asc").First(&outbox).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &outbox, nil
}

func (r outboxRepository) FinishEventByTableNameAndID(ctx context.Context, db *gorm.DB, table interface{}, id string) error {
	err := db.WithContext(ctx).Model(table).Where("id = ?", id).Update("status", model.OutboxStatusFinish).Error
	if err != nil {
		return err
	}
	return nil
}
