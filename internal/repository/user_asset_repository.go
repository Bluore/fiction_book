package repository

import "gorm.io/gorm"

type UserAssetRepository interface {
}

type userAssetRepository struct {
	db *gorm.DB
}

func NewUserAssetRepository(db *gorm.DB) UserAssetRepository {
	return &userAssetRepository{db: db}
}
