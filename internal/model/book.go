package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameBook = "book"

// Book 模型
type Book struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	OwnerID         string         `json:"user_id" gorm:"type:varchar(255)"`
	Name            string         `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	Description     string         `json:"description" gorm:"type:text;not null"`
	VIPLevel        string         `json:"vip_level" gorm:"type:varchar(255)"`
	Price           int64          `json:"price" gorm:"type:decimal(10,0)"`
	CoverImage      string         `json:"cover_image" gorm:"type:varchar(255);not null"`
	BrowseAmount    uint64         `json:"browse_amount" gorm:"type:integer;not null;default:0"`
	BookshelfAmount uint64         `json:"bookshelf_amount" gorm:"type:integer;not null;default:0"`
	PushedAt        time.Time      `json:"pushed_at" gorm:"type:timestamp;not null"`
	ThermalValue    uint64         `json:"thermal_value" gorm:"type:integer;not null;default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Book) TableName() string {
	return TableNameBook
}

func (m *Book) CalculateThermal() {
	m.ThermalValue = m.BookshelfAmount*10 + m.BrowseAmount
}
