package repository

import "gorm.io/gorm"

type UserPurchasedBookRepository interface {
}

type userPurchasedBookRepository struct {
	db *gorm.DB
}

func NewUserPurchasedBookRepository(db *gorm.DB) UserPurchasedBookRepository {
	return &userPurchasedBookRepository{
		db: db,
	}
}
