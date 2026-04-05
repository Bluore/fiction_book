package model

import (
	"time"

	"gorm.io/gorm"
)

type BookChapter struct {
	ID        string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	BookID    string         `json:"book_id" gorm:"type:varchar(36)"`
	Title     string         `json:"title" gorm:"type:varchar(255)"`
	Content   string         `json:"content" gorm:"type:text"`
	Order     int            `json:"order" gorm:"type:integer;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (BookChapter) TableName() string {
	return "book_chapter"
}

type BookChapterInfo struct {
	ID     string `json:"id"`
	BookID string `json:"book_id"`
	Title  string `json:"title"`
	Order  int    `json:"order"`
}
