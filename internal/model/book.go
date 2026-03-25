package model

import (
	"time"

	"gorm.io/gorm"
)

// Book 模型
type Book struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	OwnerID         string         `json:"user_id" gorm:"type:varchar(255)"`
	Name            string         `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	Description     string         `json:"description" gorm:"type:text;not null"`
	VIPLevel        string         `json:"vip_level" gorm:"type:varchar(255)"`
	Price           float64        `json:"price" gorm:"type:decimal(10,2)"`
	CoverImage      string         `json:"cover_image" gorm:"type:varchar(255);not null"`
	BrowseAmount    uint64         `json:"browse_amount" gorm:"type:integer;not null;default:0"`
	BookshelfAmount uint64         `json:"bookshelf_amount" gorm:"type:integer;not null;default:0"`
	PushedAt        time.Time      `json:"pushed_at" gorm:"type:timestamp;not null"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Book) TableName() string {
	return "book"
}
