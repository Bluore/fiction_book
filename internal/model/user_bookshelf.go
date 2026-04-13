package model

import "time"

type UserBookshelf struct {
	ID        string    `json:"id" gorm:"primary_key;type:varchar(36)"`
	UserID    string    `json:"user_id" gorm:"index;type:varchar(36)"`
	BookID    string    `json:"book_id" gorm:"index;type:varchar(36)"`
	Order     int       `json:"order" gorm:"type:integer;not null;default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

func (UserBookshelf) TableName() string {
	return "user_bookshelf"
}
