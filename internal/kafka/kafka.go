package kafka

import (
	"fiction-book/internal/repository"

	"gorm.io/gorm"
)

type Handler struct {
	OutboxRepository        repository.OutboxRepository
	BookESRepository        repository.BookESRepository
	BookChapterESRepository repository.BookChapterESRepository
	DB                      *gorm.DB
}

func NewHandler(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) *Handler {
	return &Handler{
		OutboxRepository:        outboxRepository,
		BookESRepository:        bookESRepository,
		BookChapterESRepository: bookChapterESRepository,
		DB:                      db,
	}
}
