package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type Book struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	OwnerID     string         `json:"user_id" gorm:"type:varchar(255)"`
	Name        string         `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	Description string         `json:"description" gorm:"type:text;not null"`
	CoverImage  string         `json:"cover_image" gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Book) TableName() string {
	return "book"
}
