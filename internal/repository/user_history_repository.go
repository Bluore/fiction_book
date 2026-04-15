package repository

import "gorm.io/gorm"

type UserHistoryRepository interface {
}

type userHistoryRepository struct {
	db *gorm.DB
}

func NewUserHistoryRepository(db *gorm.DB) UserHistoryRepository {
	return &userHistoryRepository{
		db: db,
	}
}
