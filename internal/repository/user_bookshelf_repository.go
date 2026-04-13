package repository

import (
	"gorm.io/gorm"
)

type UserBookshelfRepository interface {
}

type userBookshelfRepository struct {
	db *gorm.DB
}

func NewUserBookshelfRepository(db *gorm.DB) UserBookshelfRepository {
	return &userBookshelfRepository{
		db: db,
	}
}
