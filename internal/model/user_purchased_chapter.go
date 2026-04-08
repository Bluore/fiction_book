package model

import (
	"time"

	"gorm.io/gorm"
)

// UserPurchasedChapter 用户已购买章节记录
type UserPurchasedChapter struct {
	ID        string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	UserID    string         `json:"user_id" gorm:"index;type:varchar(36)"`
	ChapterID string         `json:"chapter_id" gorm:"index;type:varchar(36)"`
	BookID    string         `json:"book_id" gorm:"index;type:varchar(36)"` // 冗余存储便于查询
	ActiveAt  time.Time      `json:"active_at" gorm:"index"`
	CreatedAt time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"index"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserPurchasedChapter) TableName() string {
	return "user_purchased_chapter"
}
