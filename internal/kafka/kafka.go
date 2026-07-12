package kafka

import (
	"fiction-book/internal/repository"

	"gorm.io/gorm"
)

type Handler struct {
	OutboxRepository repository.OutboxRepository
	DB               *gorm.DB
}

func NewHandler(
	outboxRepository repository.OutboxRepository,
	db *gorm.DB,
) *Handler {
	return &Handler{
		OutboxRepository: outboxRepository,
		DB:               db,
	}
}
