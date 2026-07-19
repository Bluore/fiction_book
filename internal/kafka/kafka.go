package kafka

import (
	"fiction-book/internal/repository"

	"gorm.io/gorm"
)

type Handler struct {
	OutboxRepository repository.OutboxRepository
	BookESRepository repository.BookESRepository
	DB               *gorm.DB
}

func NewHandler(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	db *gorm.DB,
) *Handler {
	return &Handler{
		OutboxRepository: outboxRepository,
		BookESRepository: bookESRepository,
		DB:               db,
	}
}
