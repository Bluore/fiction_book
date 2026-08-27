package model

import (
	"time"

	"gorm.io/gorm"
)

type UserBookshelf struct {
	ID     string `json:"id" gorm:"primary_key;type:varchar(36)"`
	UserID string `json:"user_id" gorm:"index;type:varchar(36);index:idx_userID_bookID"`
	BookID string `json:"book_id" gorm:"index;type:varchar(36);index:idx_userID_bookID"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
}

func (UserBookshelf) TableName() string {
	return "user_bookshelf"
}
