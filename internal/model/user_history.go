package model

import (
	"time"

	"gorm.io/gorm"
)

type UserHistory struct {
	ID        string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	UserID    string         `json:"user_id" gorm:"type:varchar(36);not null"`
	BookID    string         `json:"book_id" gorm:"type:varchar(36);not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserHistory) TableName() string {
	return "user_history"
}
