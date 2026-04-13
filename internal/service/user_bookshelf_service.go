package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type UserBookshelfService interface {
}

type userBookshelfService struct {
	userRepo          repository.UserRepository
	bookRepo          repository.BookRepository
	userBookshelfRepo repository.UserBookshelfRepository
	db                *gorm.DB
	cfg               *config.Config
}

func NewUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	db *gorm.DB,
	cfg *config.Config,
) UserBookshelfService {
	return &userBookshelfService{
		userRepo:          userRepo,
		bookRepo:          bookRepo,
		userBookshelfRepo: userBookshelfRepo,
		db:                db,
		cfg:               cfg,
	}
}
